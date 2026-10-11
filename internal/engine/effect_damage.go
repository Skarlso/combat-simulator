package engine

import (
	"fmt"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// rollEffectDamage rolls an effect's damage: its own formula, or the actor's standard attack
func (fr *FeatureResolver) rollEffectDamage(effect models.FeatureEffect, actor *models.Combatant) (int, error) {
	formula := effect.DiceFormula
	if effect.StandardAttack && actor != nil {
		formula = actor.DamageDice
	}

	damage := effect.Magnitude
	if formula != "" {
		notation, err := models.ParseDiceNotation(formula)
		if err != nil {
			return 0, fmt.Errorf("invalid dice formula %s: %w", formula, err)
		}
		damage += notation.Roll(fr.dice)
	}
	return damage, nil
}

func (fr *FeatureResolver) resolveDamage(effect models.FeatureEffect, actor, target *models.Combatant, state *models.CombatState, activationResult *DCCheckResult) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	damage, err := fr.rollEffectDamage(effect, actor)
	if err != nil {
		return err
	}

	damage = fr.damageAfterReaction(effect, actor, target, damage, activationResult)

	damageType := models.ParseDamageType(effect.DamageType)
	damageRes := target.TakeDamageWithArmorCost(damage, damageType, fr.dice, effect.ArmorCost)

	if activationResult != nil {
		activationResult.DamageRoll = damage
		activationResult.ProcessedDamage = damageRes.ProcessedDamage
		activationResult.DamageDealt = damageRes.HPDamage
		activationResult.Threshold = damageRes.Threshold
		activationResult.DamageResult = &damageRes
	}

	if damageRes.HPDamage > 0 {
		fr.ProcessTriggeredFeatures(target, actor, state, models.TriggerOnDamageTaken, false)
	}

	return nil
}

func (fr *FeatureResolver) resolveDirectDamage(effect models.FeatureEffect, actor, target *models.Combatant, state *models.CombatState, activationResult *DCCheckResult) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	damage, err := fr.rollEffectDamage(effect, actor)
	if err != nil {
		return err
	}

	damage = fr.damageAfterReaction(effect, actor, target, damage, activationResult)

	damageType := models.ParseDamageType(effect.DamageType)
	damageRes := target.TakeDirectDamage(damage, damageType, fr.dice)

	if activationResult != nil {
		activationResult.DamageRoll = damage
		activationResult.ProcessedDamage = damageRes.ProcessedDamage
		activationResult.DamageDealt = damageRes.HPDamage
		activationResult.Threshold = damageRes.Threshold
		activationResult.DamageResult = &damageRes
	}

	if damageRes.HPDamage > 0 {
		fr.ProcessTriggeredFeatures(target, actor, state, models.TriggerOnDamageTaken, false)
	}

	return nil
}

func (fr *FeatureResolver) resolveAreaDamage(effect models.FeatureEffect, actor, target *models.Combatant, state *models.CombatState, activationResult *DCCheckResult) error {
	damage, err := fr.rollEffectDamage(effect, actor)
	if err != nil {
		return err
	}

	damageType := models.ParseDamageType(effect.DamageType)
	enemies := fr.areaTargets(actor, target, state, activationResult)
	if len(enemies) == 0 {
		return nil
	}

	totalHPDamage := 0
	totalProcessedDamage := 0

	for _, target := range enemies {
		if !target.IsAlive() {
			continue
		}

		taken := fr.damageAfterReaction(effect, actor, target, damage, activationResult)
		damageRes := target.TakeDamageWithArmorCost(taken, damageType, fr.dice, effect.ArmorCost)
		totalProcessedDamage += damageRes.ProcessedDamage
		totalHPDamage += damageRes.HPDamage

		if damageRes.HPDamage > 0 {
			fr.ProcessTriggeredFeatures(target, actor, state, models.TriggerOnDamageTaken, false)
		}
	}

	if activationResult != nil {
		activationResult.DamageRoll = damage
		activationResult.ProcessedDamage = totalProcessedDamage
		activationResult.DamageDealt = totalHPDamage
		activationResult.Threshold = models.ThresholdMajor
	}

	return nil
}

