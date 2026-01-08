package engine

import (
	"fmt"
	"math/rand"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// FeatureResolver handles resolution of adversary features and domain abilities
type FeatureResolver struct {
	dice DiceRoller
}

// NewFeatureResolver creates a feature resolver
func NewFeatureResolver(rng *rand.Rand) *FeatureResolver {
	return &FeatureResolver{
		dice: NewStandardDiceRoller(rng),
	}
}

// ResolveEffect applies a single effect to target(s)
func (fr *FeatureResolver) ResolveEffect(
	effect models.FeatureEffect,
	actor *models.Combatant,
	target *models.Combatant,
	state *models.CombatState,
) error {
	switch effect.Category {
	// Damage categories
	case models.FeatureEffectDamage:
		return fr.resolveDamage(effect, actor, target, state)
	case models.FeatureEffectDamageAmplifier:
		return fr.resolveDamageAmplifier(effect, actor, target, state)
	case models.FeatureEffectDirectDamage:
		return fr.resolveDirectDamage(effect, actor, target, state)

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
		return fmt.Errorf("summoning not yet implemented")

	// Defensive abilities
	case models.FeatureEffectDamageReduction:
		// Passive - applied when taking damage (not implemented in combat flow yet)
		return nil

	case models.FeatureEffectGrantArmor:
		return fr.resolveGrantArmor(effect, target)

	// Token mechanics
	case models.FeatureEffectTokenPlace:
		// Tokens are tracked on DomainAbility, not implemented in combat flow yet
		return nil
	case models.FeatureEffectTokenSpend:
		// Tokens are spent via cost checks, not implemented in combat flow yet
		return nil

	default:
		return fmt.Errorf("unknown effect category: %s", effect.Category)
	}
}

// resolveDamage applies standard damage with dice roll
func (fr *FeatureResolver) resolveDamage(effect models.FeatureEffect, actor, target *models.Combatant, state *models.CombatState) error {
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

	hpDamage := target.TakeDamage(damage, damageType, fr.dice)

	// TODO: Add proper logging via RecordAction
	_ = hpDamage

	return nil
}

// resolveDamageAmplifier adds bonus damage (typically used as modifier to attacks)
func (fr *FeatureResolver) resolveDamageAmplifier(effect models.FeatureEffect, actor, target *models.Combatant, state *models.CombatState) error {
	// This is a passive modifier that should be applied during attack resolution
	// For now, just log it
	return nil
}

// resolveDirectDamage bypasses armor entirely
func (fr *FeatureResolver) resolveDirectDamage(effect models.FeatureEffect, actor, target *models.Combatant, state *models.CombatState) error {
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

	// TODO: Add proper logging via RecordAction

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
// Returns true if the roll succeeds
func (fr *FeatureResolver) rollDCCheck(actor, target *models.Combatant, dcType string, fixedDC int, state *models.CombatState) bool {
	var dc int

	// Determine the DC
	switch dcType {
	case "target":
		if target == nil {
			return false // No target to roll against
		}
		dc = target.GetDefense()
	case "fixed":
		dc = fixedDC
	default:
		return true // Auto-succeed if no DC specified
	}

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

		// Handle Hope/Fear generation
		if dualityRoll.Result == models.ResultHope || dualityRoll.Result == models.ResultCriticalHope {
			state.AddHope(1)
		}
		if dualityRoll.Result == models.ResultFear || dualityRoll.Result == models.ResultCriticalFear {
			state.AddFear(1)
		}

		// Critical success always succeeds
		if dualityRoll.IsCritical {
			return true
		}

		// Check if roll beats DC
		return dualityRoll.WithSpirit >= dc

	} else {
		// Enemy uses d20 + ATK
		roll := fr.dice.RollDice(1, 20)
		attackBonus := actor.GetAttackBonus()
		totalRoll := roll + attackBonus

		return totalRoll >= dc
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

	return true
}

// ActivateAdversaryFeature triggers an adversary feature
func (fr *FeatureResolver) ActivateAdversaryFeature(
	feature models.AdversaryFeature,
	actor *models.Combatant,
	target *models.Combatant,
	state *models.CombatState,
) error {
	if !fr.CanActivateAdversaryFeature(feature, actor, state) {
		return fmt.Errorf("cannot activate feature %s", feature.Name)
	}

	// Pay costs BEFORE rolling (Daggerheart rule: pay costs to attempt)
	if feature.FearCost > 0 {
		state.SpendFear(feature.FearCost)
	}

	// Check DC if required
	if feature.DCType != "" && feature.DCType != "none" {
		success := fr.rollDCCheck(actor, target, feature.DCType, feature.FixedDC, state)
		if !success {
			// Feature failed - costs already spent
			return fmt.Errorf("feature %s failed DC check", feature.Name)
		}
	}

	// Apply effects
	for _, effect := range feature.Effects {
		if err := fr.ResolveEffect(effect, actor, target, state); err != nil {
			return err
		}
	}

	return nil
}

// ActivateDomainAbility triggers a player ability
func (fr *FeatureResolver) ActivateDomainAbility(
	ability *models.DomainAbility,
	actor *models.Combatant,
	target *models.Combatant,
	state *models.CombatState,
) error {
	if !fr.CanActivateDomainAbility(*ability, actor, state) {
		return fmt.Errorf("cannot activate ability %s", ability.Name)
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
		success := fr.rollDCCheck(actor, target, ability.DCType, ability.FixedDC, state)
		if !success {
			// Ability failed - costs already spent
			return fmt.Errorf("ability %s failed DC check", ability.Name)
		}
	}

	// Apply effects
	for _, effect := range ability.Effects {
		if err := fr.ResolveEffect(effect, actor, target, state); err != nil {
			return err
		}
	}

	return nil
}
