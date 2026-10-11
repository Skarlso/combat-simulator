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
	IsCritical     bool
	VsDefense      bool // True if this was an attack roll against a target's Evasion/Difficulty
	// Who this activation's area effects can reach, picked once so every effect agrees
	areaCandidates []*models.Combatant
	// Reaction rolls made during this activation, one per target, in the order rolled
	reactions map[*models.Combatant]reactionResult
	Reactions []ReactionRoll
	HopeGenerated  int
	FearGenerated  int
	// Damage tracking (populated during effect resolution)
	DamageDealt     int                    // HP damage actually dealt to target
	DamageRoll      int                    // Raw damage before resistance/threshold
	ProcessedDamage int                    // Damage after resistance/immunity/vulnerability
	Threshold       models.DamageThreshold // Threshold category of damage
	DamageResult    *models.DamageResult   // Full damage result with armor/HP tracking
}

// hits reports whether the roll succeeds against one target of a multi-target attack.
// SRD: one attack roll, compared to each target separately.
func (r *DCCheckResult) hits(target *models.Combatant) bool {
	if r == nil || !r.VsDefense {
		return true
	}
	return r.IsCritical || r.TotalRoll >= target.GetDefense()
}

func isAreaEffect(effect models.FeatureEffect) bool {
	return effect.Category == models.FeatureEffectAreaDamage || effect.TargetType == models.FeatureTargetAOE
}

// pickAreaCandidates chooses who an area effect can reach. There are no ranges, so
// the area is assumed to catch the chosen target plus random others, up to areaTargetLimit.
func (fr *FeatureResolver) pickAreaCandidates(actor, target *models.Combatant, state *models.CombatState) []*models.Combatant {
	pool := make([]*models.Combatant, 0)
	for _, t := range state.GetEnemiesOf(actor) {
		if t.IsAlive() {
			pool = append(pool, t)
		}
	}
	if fr.areaTargetLimit <= 0 || len(pool) <= fr.areaTargetLimit {
		return pool
	}

	picked := make([]*models.Combatant, 0, fr.areaTargetLimit)
	for i, t := range pool {
		if t == target {
			picked = append(picked, t)
			pool = append(pool[:i], pool[i+1:]...)
			break
		}
	}
	for len(picked) < fr.areaTargetLimit {
		n := len(pool)
		i := ((fr.dice.RollDice(1, n, 0)-1)%n + n) % n
		picked = append(picked, pool[i])
		pool = append(pool[:i], pool[i+1:]...)
	}
	return picked
}

// areaTargets returns the area candidates that the activation's attack roll hits
func (fr *FeatureResolver) areaTargets(actor, target *models.Combatant, state *models.CombatState, result *DCCheckResult) []*models.Combatant {
	var candidates []*models.Combatant
	if result != nil {
		if result.areaCandidates == nil {
			result.areaCandidates = fr.pickAreaCandidates(actor, target, state)
		}
		candidates = result.areaCandidates
	} else {
		candidates = fr.pickAreaCandidates(actor, target, state)
	}

	targets := make([]*models.Combatant, 0, len(candidates))
	for _, t := range candidates {
		if t.IsAlive() && result.hits(t) {
			targets = append(targets, t)
		}
	}
	return targets
}

// resolveEffects applies a feature's effects after its roll. When the roll missed the
// chosen target, only area effects go ahead, and only if they hit someone else.
func (fr *FeatureResolver) resolveEffects(
	name string,
	effects []models.FeatureEffect,
	actor, target *models.Combatant,
	state *models.CombatState,
	result *DCCheckResult,
) error {
	primaryHit := result.Success
	if !primaryHit {
		hasArea := false
		for _, effect := range effects {
			hasArea = hasArea || isAreaEffect(effect)
		}
		if !hasArea || !result.VsDefense || len(fr.areaTargets(actor, target, state, result)) == 0 {
			return fmt.Errorf("%s failed DC check", name)
		}
		// The roll still succeeded against someone, so only a Fear result passes the spotlight
		result.SpotlightShift = result.FearGenerated > 0
	}

	for _, effect := range effects {
		if !primaryHit && !isAreaEffect(effect) {
			continue
		}
		if err := fr.ResolveEffect(effect, actor, target, state, result); err != nil {
			return err
		}
	}

	if primaryHit && result.VsDefense && target != nil {
		fr.ProcessTriggeredFeatures(target, actor, state, models.TriggerOnHitTaken, false)
	}
	return nil
}

// FeatureResolver handles resolution of adversary features and domain abilities
type FeatureResolver struct {
	dice     DiceRoller
	registry AdversaryRegistry // Optional: for summoning support
	// Most targets one area effect reaches (0 = everyone)
	areaTargetLimit int
}

// SetAreaTargetLimit caps how many targets a single area effect reaches
func (fr *FeatureResolver) SetAreaTargetLimit(limit int) {
	fr.areaTargetLimit = limit
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
	// An effect flagged as area lands on everyone the area catches
	if effect.TargetType == models.FeatureTargetAOE && effect.Category != models.FeatureEffectAreaDamage {
		single := effect
		single.TargetType = models.FeatureTargetSingle
		for _, t := range fr.areaTargets(actor, target, state, result) {
			if err := fr.ResolveEffect(single, actor, t, state, result); err != nil {
				return err
			}
		}
		return nil
	}

	switch effect.Category {
	case models.FeatureEffectDamage, models.FeatureEffectDirectDamage, models.FeatureEffectAreaDamage:
		// Damage effects apply the reaction roll themselves, since success may only halve
	default:
		if fr.avoidsEffect(effect, actor, target, result) {
			return nil
		}
	}

	switch effect.Category {
	// Damage categories
	case models.FeatureEffectDamage:
		return fr.resolveDamage(effect, actor, target, state, result)
	case models.FeatureEffectDirectDamage:
		return fr.resolveDirectDamage(effect, actor, target, state, result)
	case models.FeatureEffectAreaDamage:
		return fr.resolveAreaDamage(effect, actor, target, state, result)

	// Status application
	case models.FeatureEffectApplyVulnerable:
		return fr.applyCondition(actor, target, models.ConditionVulnerable, effect.Duration)
	case models.FeatureEffectApplyPoison:
		return fr.applyDamagingCondition(actor, target, models.ConditionPoisoned, effect)
	case models.FeatureEffectApplyIgnite:
		return fr.applyDamagingCondition(actor, target, models.ConditionBurning, effect)
	case models.FeatureEffectApplyStunned:
		return fr.applyCondition(actor, target, models.ConditionStunned, effect.Duration)
	case models.FeatureEffectApplyRestrained:
		return fr.applyCondition(actor, target, models.ConditionRestrained, effect.Duration)
	case models.FeatureEffectApplyTaunt:
		return fr.applyCondition(actor, target, models.ConditionTaunted, effect.Duration)
	case models.FeatureEffectApplyHidden:
		return fr.applyCondition(actor, actor, models.ConditionHidden, effect.Duration)

	// Resource manipulation
	case models.FeatureEffectStressInflict:
		return fr.resolveStressInflict(effect, target)
	case models.FeatureEffectStressClear:
		return fr.resolveStressClear(effect, actor)
	case models.FeatureEffectHPHeal:
		return fr.resolveHPHeal(effect, actor)
	case models.FeatureEffectArmorDamage:
		return fr.resolveArmorDamage(effect, target, state)
	case models.FeatureEffectHopeDrain:
		return fr.resolveHopeDrain(effect, target)
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

	case models.FeatureEffectDamageReduction:
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
	result.VsDefense = dcType == "target"

	// Roll based on actor's side
	if actor.Side == models.SidePlayer {
		// Player uses duality dice
		hopeModifier := actor.GetAttackBonus()
		fearModifier := 0

		hasAdv := actor.HasCondition(models.ConditionAdvantage) ||
			(target != nil && target.HasCondition(models.ConditionVulnerable))
		hasDisadv := actor.HasCondition(models.ConditionDisadvantage) ||
			(target != nil && target.HasCondition(models.ConditionHidden))

		if hasAdv && !hasDisadv {
			hopeModifier += fr.dice.RollDice(1, 6, 0)
		} else if hasDisadv && !hasAdv {
			hopeModifier -= fr.dice.RollDice(1, 6, 0)
		}

		dualityRoll := fr.dice.RollDuality(hopeModifier, fearModifier)

		// Store roll details
		result.HopeDie = dualityRoll.HopeDie
		result.FearDie = dualityRoll.FearDie
		result.TotalRoll = dualityRoll.WithSpirit

		// Handle Hope/Fear generation
		if dualityRoll.Result == models.ResultHope || dualityRoll.Result == models.ResultCriticalHope {
			actor.GainHope(1)
			result.HopeGenerated = 1
		}
		if dualityRoll.Result == models.ResultFear {
			state.AddFear(1)
			result.FearGenerated = 1
			result.SpotlightShift = true // Fear result shifts spotlight
		}

		if dualityRoll.IsCritical {
			result.Success = true
			result.IsCritical = true
			actor.ClearStress(1)
			return result
		}

		result.Success = dualityRoll.WithSpirit >= dc
		if !result.Success {
			result.SpotlightShift = true
		}
		return result

	} else {
		hasAdvEnemy := target != nil && target.HasCondition(models.ConditionVulnerable)
		hasDisadvEnemy := target != nil && target.HasCondition(models.ConditionHidden)
		roll := rollAdversaryD20(fr.dice, actor.GetAttackBonus(), hasAdvEnemy, hasDisadvEnemy)

		result.D20Roll = roll.D20Roll
		result.TotalRoll = roll.Total
		result.IsCritical = roll.IsCritical
		result.Success = roll.IsCritical || roll.Total >= dc
		return result
	}
}

// CanActivateAdversaryFeature checks if an adversary can use a feature
func (fr *FeatureResolver) CanActivateAdversaryFeature(feature *models.AdversaryFeature, actor *models.Combatant, state *models.CombatState) bool {
	if feature.FearCost > 0 && state.FearPool < feature.FearCost {
		return false
	}

	if feature.StressCost > 0 && (actor.Stress+feature.StressCost) > actor.StressMax {
		return false
	}

	if feature.MaxUses > 0 && feature.UsesLeft <= 0 {
		return false
	}

	for _, effect := range feature.Effects {
		if effect.Category == models.FeatureEffectTokenSpend {
			if feature.TokensHeld < effect.Magnitude {
				return false
			}
		}
	}

	return true
}

// CanActivateDomainAbility checks if a player can use an ability
func (fr *FeatureResolver) CanActivateDomainAbility(ability models.DomainAbility, actor *models.Combatant, state *models.CombatState) bool {
	// Check Hope cost
	if ability.HopeCost > 0 && actor.Hope < ability.HopeCost {
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
	feature *models.AdversaryFeature,
	actor *models.Combatant,
	target *models.Combatant,
	state *models.CombatState,
) (error, DCCheckResult) {
	dcResult := DCCheckResult{Success: true} // Default for features without DC checks

	if !fr.CanActivateAdversaryFeature(feature, actor, state) {
		return fmt.Errorf("cannot activate feature %s", feature.Name), dcResult
	}

	if feature.FearCost > 0 {
		state.SpendFear(feature.FearCost)
	}
	if feature.StressCost > 0 {
		actor.AddStress(feature.StressCost)
	}
	if feature.MaxUses > 0 {
		feature.UsesLeft--
	}

	// Check DC if required
	if feature.DCType != "" && feature.DCType != "none" {
		dcResult = fr.rollDCCheck(actor, target, feature.DCType, feature.FixedDC, state)
	}

	for _, effect := range feature.Effects {
		if effect.Category == models.FeatureEffectTokenSpend {
			if feature.TokensHeld < effect.Magnitude {
				return fmt.Errorf("not enough tokens: have %d, need %d", feature.TokensHeld, effect.Magnitude), dcResult
			}
			feature.TokensHeld -= effect.Magnitude
		}
	}

	if err := fr.resolveEffects("feature "+feature.Name, feature.Effects, actor, target, state, &dcResult); err != nil {
		return err, dcResult
	}

	for _, effect := range feature.Effects {
		if effect.Category == models.FeatureEffectTokenPlace {
			feature.TokensHeld += effect.Magnitude
			if feature.TokensMax > 0 && feature.TokensHeld > feature.TokensMax {
				feature.TokensHeld = feature.TokensMax
			}
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
		actor.SpendHope(ability.HopeCost)
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
	if err := fr.resolveEffects("ability "+ability.Name, ability.Effects, actor, target, state, &dcResult); err != nil {
		return err, dcResult
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
	triggerType models.FeatureTriggerType,
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
		if !fr.CanActivateAdversaryFeature(feature, actor, state) {
			continue
		}

		err, dcResult := fr.ActivateAdversaryFeature(feature, actor, target, state)
		recordReactions(state, feature.Name, dcResult)
		if err != nil {
			state.RecordAction(models.ActionParams{
				Actor:      actor,
				ActionType: models.ActionUseAbility,
				FeatureName: feature.Name,
				Result:     fmt.Sprintf("Triggered: %v", err),
				HopeDie:    dcResult.HopeDie,
				FearDie:    dcResult.FearDie,
			})
		}
	}

	// Check domain abilities
	for i := range actor.DomainAbilities {
		ability := &actor.DomainAbilities[i]

		if !ability.Passive {
			continue
		}

		if !fr.shouldTrigger(ability.Trigger, triggerType, isCritical, actor, target) {
			continue
		}

		if !fr.CanActivateDomainAbility(*ability, actor, state) {
			continue
		}

		err, dcResult := fr.ActivateDomainAbility(ability, actor, target, state)
		recordReactions(state, ability.Name, dcResult)
		if err != nil {
			state.RecordAction(models.ActionParams{
				Actor:      actor,
				ActionType: models.ActionUseAbility,
				FeatureName: ability.Name,
				Result:     fmt.Sprintf("Triggered: %v", err),
				HopeDie:    dcResult.HopeDie,
				FearDie:    dcResult.FearDie,
			})
		}
	}
}

// shouldTrigger checks if a trigger condition is met
func (fr *FeatureResolver) shouldTrigger(
	trigger models.FeatureTriggerCondition,
	triggerType models.FeatureTriggerType,
	isCritical bool,
	actor *models.Combatant,
	target *models.Combatant,
) bool {
	if trigger.Type != triggerType {
		return false
	}
	if triggerType == models.TriggerOnCritical && !isCritical {
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
		return false
	default:
		// Unknown filter, assume it passes
		return true
	}
}
