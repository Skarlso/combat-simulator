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
	DamageDealt     int                    // HP damage actually dealt to target
	DamageRoll      int                    // Raw damage before resistance/threshold
	ProcessedDamage int                    // Damage after resistance/immunity/vulnerability
	Threshold       models.DamageThreshold // Threshold category of damage
	DamageResult    *models.DamageResult   // Full damage result with armor/HP tracking
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
	case models.FeatureEffectAreaDamage:
		return fr.resolveAreaDamage(effect, actor, state, result)

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
	if feature.FearCost > 0 && state.FearPool < feature.FearCost {
		return false
	}

	if feature.StressCost > 0 && (actor.Stress+feature.StressCost) > actor.StressMax {
		return false
	}

	if feature.SpotlightLimit > 0 && actor.SpotlightTokens >= feature.SpotlightLimit {
		return false
	}

	if feature.MaxUses > 0 && feature.UsesLeft <= 0 {
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
		if !fr.CanActivateAdversaryFeature(*feature, actor, state) {
			continue
		}

		// Activate the feature
		_, _ = fr.ActivateAdversaryFeature(*feature, actor, target, state)
	}

	// Check domain abilities
	for i := range actor.DomainAbilities {
		ability := &actor.DomainAbilities[i]

		// Skip if not passive or trigger doesn't match
		if !ability.Passive {
			continue
		}

		if !fr.shouldTrigger(ability.Trigger, triggerType, isCritical, actor, target) {
			continue
		}

		// Try to activate (respects costs and usage limits)
		if !fr.CanActivateDomainAbility(*ability, actor, state) {
			continue
		}

		// Activate the ability
		_, _ = fr.ActivateDomainAbility(ability, actor, target, state)
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
		// TODO: Implement adjacency tracking in combat state
		return false
	default:
		// Unknown filter, assume it passes
		return true
	}
}
