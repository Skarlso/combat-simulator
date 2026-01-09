package engine

import (
	"fmt"
	"math/rand"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// DCCheckResult contains the details of a DC check roll
type DCCheckResult struct {
	Success        bool
	HopeDie        int // Player's hope die (0 for enemies)
	FearDie        int // Player's fear die (0 for enemies)
	D20Roll        int // Enemy's d20 roll (0 for players)
	TotalRoll      int
	DC             int
	SpotlightShift bool // True if roll causes spotlight to shift (Fear result for players)
	HopeGenerated  int
	FearGenerated  int
	// Damage tracking (populated during effect resolution)
	DamageDealt     int                   // HP damage actually dealt to target
	DamageRoll      int                   // Raw damage before resistance/threshold
	ProcessedDamage int                   // Damage after resistance/immunity/vulnerability
	Threshold       models.DamageThreshold // Threshold category of damage
}

// FeatureResolver handles resolution of adversary features and domain abilities
type FeatureResolver struct {
	dice     DiceRoller
	registry AdversaryRegistry // Optional: for summoning support
}

// NewFeatureResolver creates a feature resolver
func NewFeatureResolver(rng *rand.Rand) *FeatureResolver {
	return &FeatureResolver{
		dice:     NewStandardDiceRoller(rng),
		registry: nil, // Set via SetAdversaryRegistry if summoning is needed
	}
}

// SetAdversaryRegistry sets the registry for summoning support
func (fr *FeatureResolver) SetAdversaryRegistry(registry AdversaryRegistry) {
	fr.registry = registry
}

// ResolveEffect applies a single effect to target(s)
func (fr *FeatureResolver) ResolveEffect(
	effect models.FeatureEffect,
	actor *models.Combatant,
	target *models.Combatant,
	state *models.CombatState,
	result *DCCheckResult, // For tracking damage dealt
) error {
	switch effect.Category {
	// Damage categories
	case models.FeatureEffectDamage:
		return fr.resolveDamage(effect, actor, target, state, result)
	case models.FeatureEffectDirectDamage:
		return fr.resolveDirectDamage(effect, actor, target, state, result)

	// Status application
	case models.FeatureEffectApplyVulnerable:
		return fr.applyCondition(target, models.ConditionVulnerable, effect.Duration)
	case models.FeatureEffectApplyPoison:
		return fr.applyCondition(target, models.ConditionPoisoned, effect.Duration)
	case models.FeatureEffectApplyIgnite:
		return fr.applyCondition(target, models.ConditionBurning, effect.Duration)
	case models.FeatureEffectApplyTaunt:
		return fr.applyCondition(target, models.ConditionTaunted, effect.Duration)
	case models.FeatureEffectApplyHidden:
		return fr.applyCondition(actor, models.ConditionHidden, effect.Duration)

	// Resource manipulation
	case models.FeatureEffectStressInflict:
		return fr.resolveStressInflict(effect, target)
	case models.FeatureEffectStressClear:
		return fr.resolveStressClear(effect, actor)
	case models.FeatureEffectHPHeal:
		return fr.resolveHPHeal(effect, actor)
	case models.FeatureEffectArmorDamage:
		return fr.resolveArmorDamage(effect, target)
	case models.FeatureEffectHopeDrain:
		return fr.resolveHopeDrain(effect, state)
	case models.FeatureEffectFearGenerate:
		return fr.resolveFearGenerate(effect, state)

	// Resistance (passive, applied during damage calculation)
	case models.FeatureEffectDamageResistance, models.FeatureEffectDamageVulnerability:
		// These are handled passively during damage resolution
		return nil

	// Action economy (passive modifiers)
	case models.FeatureEffectSpotlightModifier, models.FeatureEffectAllySpotlight:
		// These are handled by combat engine during spotlight phase
		return nil

	// Summoning (requires special handling)
	case models.FeatureEffectSummonAllies:
		return fr.resolveSummon(effect, actor, state)

	// Defensive abilities
	case models.FeatureEffectDamageReduction:
		// Passive - applied when taking damage (not implemented in combat flow yet)
		return nil

	case models.FeatureEffectGrantArmor:
		return fr.resolveGrantArmor(effect, target)

	// Token mechanics (handled separately in ActivateDomainAbility)
	case models.FeatureEffectTokenPlace:
		return nil
	case models.FeatureEffectTokenSpend:
		return nil

	default:
		return fmt.Errorf("unknown effect category: %s", effect.Category)
	}
}

// resolveDamage applies standard damage with dice roll
func (fr *FeatureResolver) resolveDamage(effect models.FeatureEffect, actor, target *models.Combatant, state *models.CombatState, activationResult *DCCheckResult) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	damage := effect.Magnitude
	if effect.DiceFormula != "" {
		notation, err := models.ParseDiceNotation(effect.DiceFormula)
		if err != nil {
			return fmt.Errorf("invalid dice formula %s: %w", effect.DiceFormula, err)
		}
		damage += notation.Roll(fr.dice)
	}

	// Convert string to DamageType
	damageType := models.DamagePhysical
	if effect.DamageType == "magic" {
		damageType = models.DamageMagic
	}

	// Apply damage (TakeDamage calculates threshold from processed damage)
	damageRes := target.TakeDamage(damage, damageType, fr.dice)

	// Track damage in activation result for logging
	if activationResult != nil {
		activationResult.DamageRoll = damage
		activationResult.ProcessedDamage = damageRes.ProcessedDamage
		activationResult.DamageDealt = damageRes.HPDamage
		activationResult.Threshold = damageRes.Threshold
	}

	// Trigger on_damage_taken features for the target if damage was dealt
	if damageRes.HPDamage > 0 {
		fr.ProcessTriggeredFeatures(target, actor, state, "on_damage_taken", false)
	}

	return nil
}

// resolveDirectDamage bypasses armor entirely
func (fr *FeatureResolver) resolveDirectDamage(effect models.FeatureEffect, actor, target *models.Combatant, state *models.CombatState, activationResult *DCCheckResult) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	damage := effect.Magnitude
	if effect.DiceFormula != "" {
		notation, err := models.ParseDiceNotation(effect.DiceFormula)
		if err != nil {
			return fmt.Errorf("invalid dice formula %s: %w", effect.DiceFormula, err)
		}
		damage += notation.Roll(fr.dice)
	}

	// Direct damage bypasses thresholds - directly marks HP
	target.HP -= damage
	if target.HP <= 0 {
		target.HP = 0
		target.IsDefeated = true
	}

	// Track damage in activation result for logging
	if activationResult != nil {
		activationResult.DamageRoll = damage
		activationResult.DamageDealt = damage // Direct damage = HP damage
		activationResult.Threshold = models.ThresholdSevere // Arbitrary for direct damage
	}

	// Trigger on_damage_taken features for the target if damage was dealt
	if damage > 0 {
		fr.ProcessTriggeredFeatures(target, actor, state, "on_damage_taken", false)
	}

	return nil
}

// applyCondition adds a status effect to target
func (fr *FeatureResolver) applyCondition(target *models.Combatant, condType models.ConditionType, duration string) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	// Parse duration string to DurationType
	var durationType models.DurationType
	switch duration {
	case "until_cleared", "permanent":
		durationType = models.DurationUntilCleared
	case "1_turn":
		durationType = models.DurationEndOfTurn
	case "instant":
		return nil // No lasting effect
	case "scene":
		durationType = models.DurationUntilCleared
	default:
		durationType = models.DurationEndOfTurn
	}

	condition := &models.ActiveCondition{
		Type:     condType,
		Duration: durationType,
	}

	target.AddCondition(condition)
	return nil
}

// resolveStressInflict marks stress on target
func (fr *FeatureResolver) resolveStressInflict(effect models.FeatureEffect, target *models.Combatant) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	amount := effect.Magnitude
	if effect.DiceFormula != "" {
		notation, err := models.ParseDiceNotation(effect.DiceFormula)
		if err != nil {
			return err
		}
		amount += notation.Roll(fr.dice)
	}

	target.AddStress(amount)
	return nil
}

// resolveStressClear removes stress from actor
func (fr *FeatureResolver) resolveStressClear(effect models.FeatureEffect, actor *models.Combatant) error {
	if actor == nil {
		return nil
	}

	amount := effect.Magnitude
	actor.ClearStress(amount)
	return nil
}

// resolveHPHeal restores HP to actor
func (fr *FeatureResolver) resolveHPHeal(effect models.FeatureEffect, actor *models.Combatant) error {
	if actor == nil {
		return nil
	}

	amount := effect.Magnitude
	if effect.DiceFormula != "" {
		notation, err := models.ParseDiceNotation(effect.DiceFormula)
		if err != nil {
			return err
		}
		amount += notation.Roll(fr.dice)
	}

	actor.Heal(amount)
	return nil
}

// resolveArmorDamage marks target's armor
func (fr *FeatureResolver) resolveArmorDamage(effect models.FeatureEffect, target *models.Combatant) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	if target.ArmorSlots > 0 {
		target.ArmorSlots--
	}
	return nil
}

// resolveGrantArmor grants armor slots to target
func (fr *FeatureResolver) resolveGrantArmor(effect models.FeatureEffect, target *models.Combatant) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	// Grant armor slots (magnitude = number of slots)
	target.ArmorSlots += effect.Magnitude
	return nil
}

// resolveHopeDrain removes Hope from the pool
func (fr *FeatureResolver) resolveHopeDrain(effect models.FeatureEffect, state *models.CombatState) error {
	amount := effect.Magnitude
	state.SpendHope(amount)
	return nil
}

// resolveFearGenerate adds Fear to the pool
func (fr *FeatureResolver) resolveFearGenerate(effect models.FeatureEffect, state *models.CombatState) error {
	amount := effect.Magnitude
	state.AddFear(amount)
	return nil
}

// rollDCCheck rolls to beat a Difficulty Check
// Returns DCCheckResult with roll details and success/failure
func (fr *FeatureResolver) rollDCCheck(actor, target *models.Combatant, dcType string, fixedDC int, state *models.CombatState) DCCheckResult {
	result := DCCheckResult{
		Success:        true,
		HopeDie:        0,
		FearDie:        0,
		TotalRoll:      0,
		DC:             0,
		SpotlightShift: false,
		HopeGenerated:  0,
		FearGenerated:  0,
	}

	var dc int

	// Determine the DC
	switch dcType {
	case "target":
		if target == nil {
			// No target - auto-succeed for self-targeted abilities
			return result
		}
		dc = target.GetDefense()
	case "fixed":
		dc = fixedDC
	case "none", "":
		return result // Auto-succeed
	default:
		return result // Auto-succeed if no DC specified
	}

	result.DC = dc

	// Roll based on actor's side
	if actor.Side == models.SidePlayer {
		// Player uses duality dice
		hopeModifier := actor.GetAttackBonus()
		fearModifier := 0

		// Check for advantage/disadvantage
		if actor.HasCondition(models.ConditionAdvantage) {
			hopeModifier += fr.dice.RollD6()
		}
		if actor.HasCondition(models.ConditionDisadvantage) {
			hopeModifier -= fr.dice.RollD6()
		}
		if actor.IsFullyStressed {
			hopeModifier -= 2
		}

		dualityRoll := fr.dice.RollDuality(hopeModifier, fearModifier)

		// Store roll details
		result.HopeDie = dualityRoll.HopeDie
		result.FearDie = dualityRoll.FearDie
		result.TotalRoll = dualityRoll.WithSpirit

		// Handle Hope/Fear generation
		if dualityRoll.Result == models.ResultHope || dualityRoll.Result == models.ResultCriticalHope {
			state.AddHope(1)
			result.HopeGenerated = 1
		}
		if dualityRoll.Result == models.ResultFear || dualityRoll.Result == models.ResultCriticalFear {
			state.AddFear(1)
			result.FearGenerated = 1
			result.SpotlightShift = true // Fear result shifts spotlight
		}

		// Critical success always succeeds
		if dualityRoll.IsCritical {
			result.Success = true
			return result
		}

		// Check if roll beats DC
		result.Success = dualityRoll.WithSpirit >= dc
		return result

	} else {
		// Enemy uses d20 + ATK
		roll := fr.dice.RollDice(1, 20)
		attackBonus := actor.GetAttackBonus()
		totalRoll := roll + attackBonus

		result.D20Roll = roll
		result.TotalRoll = totalRoll
		result.Success = totalRoll >= dc
		return result
	}
}

// CanActivateAdversaryFeature checks if an adversary can use a feature
func (fr *FeatureResolver) CanActivateAdversaryFeature(feature models.AdversaryFeature, actor *models.Combatant, state *models.CombatState) bool {
	// Check Fear cost
	if feature.FearCost > 0 && state.FearPool < feature.FearCost {
		return false
	}

	// Check spotlight limit
	if feature.SpotlightLimit > 0 && actor.SpotlightTokens >= feature.SpotlightLimit {
		return false
	}

	return true
}

// CanActivateDomainAbility checks if a player can use an ability
func (fr *FeatureResolver) CanActivateDomainAbility(ability models.DomainAbility, actor *models.Combatant, state *models.CombatState) bool {
	// Check Hope cost
	if ability.HopeCost > 0 && state.HopePool < ability.HopeCost {
		return false
	}

	// Check Stress cost
	if ability.StressCost > 0 && (actor.Stress+ability.StressCost) > actor.StressMax {
		return false
	}

	// Check uses left
	if ability.MaxUses > 0 && ability.UsesLeft <= 0 {
		return false
	}

	// Check token availability for token_spend effects
	for _, effect := range ability.Effects {
		if effect.Category == models.FeatureEffectTokenSpend {
			if ability.TokensHeld < effect.Magnitude {
				return false
			}
		}
	}

	return true
}

// ActivateAdversaryFeature triggers an adversary feature
// Returns error and DC check result (if DC check was performed)
func (fr *FeatureResolver) ActivateAdversaryFeature(
	feature models.AdversaryFeature,
	actor *models.Combatant,
	target *models.Combatant,
	state *models.CombatState,
) (error, DCCheckResult) {
	dcResult := DCCheckResult{Success: true} // Default for features without DC checks

	if !fr.CanActivateAdversaryFeature(feature, actor, state) {
		return fmt.Errorf("cannot activate feature %s", feature.Name), dcResult
	}

	// Pay costs BEFORE rolling (Daggerheart rule: pay costs to attempt)
	if feature.FearCost > 0 {
		state.SpendFear(feature.FearCost)
	}

	// Check DC if required
	if feature.DCType != "" && feature.DCType != "none" {
		dcResult = fr.rollDCCheck(actor, target, feature.DCType, feature.FixedDC, state)
		if !dcResult.Success {
			// Feature failed - costs already spent
			return fmt.Errorf("feature %s failed DC check", feature.Name), dcResult
		}
	}

	// Apply effects
	for _, effect := range feature.Effects {
		if err := fr.ResolveEffect(effect, actor, target, state, &dcResult); err != nil {
			return err, dcResult
		}
	}

	return nil, dcResult
}

// ActivateDomainAbility triggers a player ability
// Returns error and DC check result (if DC check was performed)
func (fr *FeatureResolver) ActivateDomainAbility(
	ability *models.DomainAbility,
	actor *models.Combatant,
	target *models.Combatant,
	state *models.CombatState,
) (error, DCCheckResult) {
	dcResult := DCCheckResult{Success: true} // Default for abilities without DC checks

	if !fr.CanActivateDomainAbility(*ability, actor, state) {
		return fmt.Errorf("cannot activate ability %s", ability.Name), dcResult
	}

	// Pay costs BEFORE rolling (Daggerheart rule: pay costs to attempt)
	if ability.HopeCost > 0 {
		state.SpendHope(ability.HopeCost)
	}
	if ability.StressCost > 0 {
		actor.AddStress(ability.StressCost)
	}

	// Decrement uses
	if ability.MaxUses > 0 {
		ability.UsesLeft--
	}

	// Check DC if required
	if ability.DCType != "" && ability.DCType != "none" {
		dcResult = fr.rollDCCheck(actor, target, ability.DCType, ability.FixedDC, state)
		if !dcResult.Success {
			// Ability failed - costs already spent
			return fmt.Errorf("ability %s failed DC check", ability.Name), dcResult
		}
	}

	// Handle token spending BEFORE applying effects
	for _, effect := range ability.Effects {
		if effect.Category == models.FeatureEffectTokenSpend {
			if ability.TokensHeld < effect.Magnitude {
				return fmt.Errorf("not enough tokens: have %d, need %d", ability.TokensHeld, effect.Magnitude), dcResult
			}
			ability.TokensHeld -= effect.Magnitude
		}
	}

	// Apply effects
	for _, effect := range ability.Effects {
		if err := fr.ResolveEffect(effect, actor, target, state, &dcResult); err != nil {
			return err, dcResult
		}
	}

	// Handle token placement AFTER applying effects
	for _, effect := range ability.Effects {
		if effect.Category == models.FeatureEffectTokenPlace {
			ability.TokensHeld += effect.Magnitude
			// Cap at TokensMax if set
			if ability.TokensMax > 0 && ability.TokensHeld > ability.TokensMax {
				ability.TokensHeld = ability.TokensMax
			}
		}
	}

	return nil, dcResult
}

// ProcessTriggeredFeatures checks and activates features based on trigger conditions
func (fr *FeatureResolver) ProcessTriggeredFeatures(
	actor *models.Combatant,
	target *models.Combatant,
	state *models.CombatState,
	triggerType string,
	isCritical bool,
) {
	// Check adversary features
	for i := range actor.AdversaryFeatures {
		feature := &actor.AdversaryFeatures[i]

		// Skip if not passive or trigger doesn't match
		if !feature.Passive {
			continue
		}

		if !fr.shouldTrigger(feature.Trigger, triggerType, isCritical, actor, target) {
			continue
		}

		// Try to activate (respects costs and spotlight limits)
		if !fr.CanActivateAdversaryFeature(*feature, actor, state) {
			continue
		}

		// Activate the feature
		_, _ = fr.ActivateAdversaryFeature(*feature, actor, target, state)
	}

	// Note: Domain abilities don't have trigger support in the current model
	// They would need a Trigger field added to DomainAbility struct to support this
}

// shouldTrigger checks if a trigger condition is met
func (fr *FeatureResolver) shouldTrigger(
	trigger models.FeatureTriggerCondition,
	triggerType string,
	isCritical bool,
	actor *models.Combatant,
	target *models.Combatant,
) bool {
	// Check trigger type match
	switch trigger.Type {
	case "on_attack":
		if triggerType != "on_attack" {
			return false
		}
	case "on_hit":
		if triggerType != "on_hit" {
			return false
		}
	case "on_damage_taken":
		if triggerType != "on_damage_taken" {
			return false
		}
	case "on_critical":
		if triggerType != "on_critical" || !isCritical {
			return false
		}
	case "start_turn":
		if triggerType != "start_turn" {
			return false
		}
	case "end_turn":
		if triggerType != "end_turn" {
			return false
		}
	default:
		return false
	}

	// Check filter conditions
	if trigger.Filter != "" {
		return fr.evaluateFilter(trigger.Filter, actor, target)
	}

	return true
}

// evaluateFilter checks additional trigger constraints
func (fr *FeatureResolver) evaluateFilter(filter string, actor *models.Combatant, target *models.Combatant) bool {
	switch filter {
	case "if_target_below_50_hp":
		if target == nil {
			return false
		}
		return (target.HP * 100 / target.HPMax) < 50
	case "if_hidden":
		return actor.HasCondition(models.ConditionHidden)
	case "if_ally_adjacent":
		// TODO: Implement adjacency tracking in combat state
		return false
	default:
		// Unknown filter, assume it passes
		return true
	}
}

// resolveSummon handles summoning allies
func (fr *FeatureResolver) resolveSummon(effect models.FeatureEffect, actor *models.Combatant, state *models.CombatState) error {
	// Check if registry is available
	if fr.registry == nil {
		return fmt.Errorf("summoning not available: adversary registry not set")
	}

	// Check if SummonType is specified
	if effect.SummonType == "" {
		return fmt.Errorf("summon effect missing summon_type")
	}

	// Determine quantity to summon
	quantity := effect.Magnitude
	if effect.DiceFormula != "" {
		notation, err := models.ParseDiceNotation(effect.DiceFormula)
		if err != nil {
			return fmt.Errorf("failed to parse summon dice formula: %w", err)
		}
		quantity += notation.Roll(fr.dice)
	}

	if quantity <= 0 {
		return fmt.Errorf("summon quantity must be positive, got %d", quantity)
	}

	// Summon the specified number of adversaries
	for i := 0; i < quantity; i++ {
		// Generate unique ID for summoned combatant
		state.SummonCounter++
		uniqueID := fmt.Sprintf("%s_summoned_%d", effect.SummonType, state.SummonCounter)

		// Clone template from registry
		summoned, err := fr.registry.CloneTemplate(effect.SummonType, uniqueID)
		if err != nil {
			return fmt.Errorf("failed to summon %s: %w", effect.SummonType, err)
		}

		// Add to combat state
		state.SummonCombatant(summoned)

		// Record summon event
		state.RecordAction(actor, models.ActionUseAbility, nil, "",
			fmt.Sprintf("Summons %s", summoned.Name),
			0, 0, nil, 0, 0, models.ThresholdMinor, 0, 0, 0, 0, false, 0, 0)
	}

	return nil
}
