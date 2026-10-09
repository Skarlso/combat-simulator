package engine

import (
	"fmt"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

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

	damageType := models.ParseDamageType(effect.DamageType)
	damageRes := target.TakeDamage(damage, damageType, fr.dice)

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

	damage := effect.Magnitude
	if effect.DiceFormula != "" {
		notation, err := models.ParseDiceNotation(effect.DiceFormula)
		if err != nil {
			return fmt.Errorf("invalid dice formula %s: %w", effect.DiceFormula, err)
		}
		damage += notation.Roll(fr.dice)
	}

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

func (fr *FeatureResolver) resolveAreaDamage(effect models.FeatureEffect, actor *models.Combatant, state *models.CombatState, activationResult *DCCheckResult) error {
	damage := effect.Magnitude
	if effect.DiceFormula != "" {
		notation, err := models.ParseDiceNotation(effect.DiceFormula)
		if err != nil {
			return fmt.Errorf("invalid dice formula %s: %w", effect.DiceFormula, err)
		}
		damage += notation.Roll(fr.dice)
	}

	damageType := models.ParseDamageType(effect.DamageType)
	enemies := state.GetEnemiesOf(actor)
	if len(enemies) == 0 {
		return nil
	}

	totalHPDamage := 0
	totalProcessedDamage := 0

	for _, target := range enemies {
		if !target.IsAlive() {
			continue
		}

		damageRes := target.TakeDamage(damage, damageType, fr.dice)
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

