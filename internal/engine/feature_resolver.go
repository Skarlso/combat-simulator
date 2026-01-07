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

	hpDamage := target.TakeDamage(damage, damageType)

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

	// Pay costs
	if feature.FearCost > 0 {
		state.SpendFear(feature.FearCost)
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

	// Pay costs
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

	// Apply effects
	for _, effect := range ability.Effects {
		if err := fr.ResolveEffect(effect, actor, target, state); err != nil {
			return err
		}
	}

	return nil
}
