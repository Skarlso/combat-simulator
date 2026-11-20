package engine

import (
	"fmt"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// rollEffectDamage rolls an effect's damage: its own formula, or the actor's standard attack.
// A critical activation adds the dice maximum, as for basic attacks.
func (fr *FeatureResolver) rollEffectDamage(effect models.FeatureEffect, actor *models.Combatant, activationResult *DCCheckResult) (int, error) {
	formula := effect.DiceFormula
	switch {
	case effect.StandardAttack && actor != nil:
		formula = actor.AttackDamageFormula()
	case effect.ProficiencyDice && actor != nil:
		formula = actor.ScaleToProficiency(formula)
	}

	damage := effect.Magnitude
	if formula != "" {
		notation, err := models.ParseDiceNotation(formula)
		if err != nil {
			return 0, fmt.Errorf("invalid dice formula %s: %w", formula, err)
		}
		damage += notation.Roll(fr.dice)
		if activationResult != nil && activationResult.IsCritical {
			damage += notation.MaxDice()
		}
	}
	if actor != nil && actor.HalfDamage {
		damage /= 2
	}
	return damage, nil
}

// noteDamage copies what one target took into the activation result for the log
func noteDamage(activationResult *DCCheckResult, damage int, damageRes models.DamageResult) {
	if activationResult == nil {
		return
	}
	activationResult.DamageRoll = damage
	activationResult.ProcessedDamage = damageRes.ProcessedDamage
	activationResult.DamageDealt = damageRes.HPDamage
	activationResult.Threshold = damageRes.Threshold
	activationResult.DamageResult = &damageRes
}

// resolveDamage damages one target; direct damage cannot be reduced by armor
func (fr *FeatureResolver) resolveDamage(effect models.FeatureEffect, actor, target *models.Combatant, state *models.CombatState, activationResult *DCCheckResult, direct bool) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	damage, err := fr.rollEffectDamage(effect, actor, activationResult)
	if err != nil {
		return err
	}
	damage = fr.damageAfterReaction(effect, actor, target, damage, activationResult)

	recipient := target
	if guardian := fr.shieldFor(target, damage, state); guardian != nil {
		recipient = guardian
		if activationResult != nil {
			activationResult.RedirectedTo = guardian
		}
	}

	damageType := models.ParseDamageType(effect.DamageType)
	var damageRes models.DamageResult
	if direct {
		damageRes = recipient.TakeDirectDamage(damage, damageType, fr.dice)
	} else {
		damageRes = recipient.TakeDamageWithArmorCost(damage, damageType, fr.dice, effect.ArmorCost)
	}
	noteDamage(activationResult, damage, damageRes)

	if damageRes.HPDamage > 0 {
		fr.ProcessTriggeredFeatures(recipient, actor, state, models.TriggerOnDamageTaken, false)
	}
	return nil
}

func (fr *FeatureResolver) resolveAreaDamage(effect models.FeatureEffect, actor, target *models.Combatant, state *models.CombatState, activationResult *DCCheckResult) error {
	damage, err := fr.rollEffectDamage(effect, actor, activationResult)
	if err != nil {
		return err
	}

	damageType := models.ParseDamageType(effect.DamageType)
	totalHP, totalProcessed := 0, 0
	for _, t := range fr.areaTargets(actor, target, state, activationResult) {
		taken := fr.damageAfterReaction(effect, actor, t, damage, activationResult)
		damageRes := t.TakeDamageWithArmorCost(taken, damageType, fr.dice, effect.ArmorCost)
		totalProcessed += damageRes.ProcessedDamage
		totalHP += damageRes.HPDamage
		if damageRes.HPDamage > 0 {
			fr.ProcessTriggeredFeatures(t, actor, state, models.TriggerOnDamageTaken, false)
		}
	}

	if activationResult != nil {
		activationResult.DamageRoll = damage
		activationResult.ProcessedDamage = totalProcessed
		activationResult.DamageDealt = totalHP
		activationResult.Threshold = models.ThresholdMajor
	}
	return nil
}
