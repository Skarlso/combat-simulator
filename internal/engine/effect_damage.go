package engine

import (
	"fmt"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// rollEffectDamage rolls an effect's damage: its own formula, or the actor's standard attack.
// A critical activation adds the dice maximum, as for basic attacks. The parts name every piece.
func (fr *FeatureResolver) rollEffectDamage(effect models.FeatureEffect, actor *models.Combatant, activationResult *DCCheckResult) (int, []models.LogPart, error) {
	if activationResult != nil {
		activationResult.DamageFormula = ""
	}
	formula := effect.DiceFormula
	switch {
	case effect.StandardAttack && actor != nil:
		formula = actor.AttackDamageFormula()
	case effect.ProficiencyDice && actor != nil:
		formula = actor.ScaleToProficiency(formula)
	case actor != nil:
		formula = actor.ScaleDice(formula, effect.ScaleBy)
	}

	var parts []models.LogPart
	if formula != "" {
		notation, err := models.ParseDiceNotation(formula)
		if err != nil {
			return 0, nil, fmt.Errorf("invalid dice formula %s: %w", formula, err)
		}
		// The unnamed first part is the roll of the formula itself
		parts = append(parts, models.LogPart{Value: notation.Roll(fr.dice)})
		if activationResult != nil {
			activationResult.DamageFormula = notation.String()
			if activationResult.IsCritical {
				parts = append(parts, models.LogPart{Label: "critical", Value: notation.MaxDice()})
			}
		}
	}
	if effect.Magnitude != 0 {
		parts = append(parts, models.LogPart{Label: "flat", Value: effect.Magnitude})
	}
	var halvings []string
	if actor != nil {
		parts = append(parts, actor.DamageBonusParts(models.ParseDamageType(effect.DamageType), fr.dice)...)
		if actor.HalfDamage {
			halvings = append(halvings, "halved")
		}
	}
	if effect.HalfDamage {
		halvings = append(halvings, "halved")
	}
	damage, parts := finishDamage(models.SumParts(parts), parts, 1, nil, halvings...)
	return damage, parts, nil
}

// noteDamage copies what one target took into the activation result for the log
func noteDamage(activationResult *DCCheckResult, damage int, parts []models.LogPart, damageRes models.DamageResult) {
	if activationResult == nil {
		return
	}
	activationResult.DamageParts = parts
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

	damage, parts, err := fr.rollEffectDamage(effect, actor, activationResult)
	if err != nil {
		return err
	}
	// A weapon attack made through an ability gets the same riders a basic attack does
	fromAttack := activationResult != nil && activationResult.VsDefense
	if effect.StandardAttack && fromAttack {
		riders, factor, multipliers := fr.damageRiders(actor, target, state)
		parts = append(parts, riders...)
		damage, parts = finishDamage(damage+models.SumParts(riders), parts, factor, multipliers)
	}
	if after := fr.damageAfterReaction(effect, actor, target, damage, activationResult); after != damage {
		parts = append(parts, models.LogPart{Label: "reaction roll", Value: after - damage})
		damage = after
	}

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
	noteDamage(activationResult, damage, parts, damageRes)
	logReactions(state, recipient, damageRes)

	if damageRes.HPDamage > 0 {
		fr.afterDamageDealt(actor, state)
		if fromAttack {
			fr.ProcessTriggeredFeatures(actor, recipient, state, models.TriggerOnDamageDealt, false)
		}
		fr.ProcessTriggeredFeatures(recipient, actor, state, models.TriggerOnDamageTaken, false)
	}
	return nil
}

func (fr *FeatureResolver) resolveAreaDamage(effect models.FeatureEffect, actor, target *models.Combatant, state *models.CombatState, activationResult *DCCheckResult) error {
	damage, parts, err := fr.rollEffectDamage(effect, actor, activationResult)
	if err != nil {
		return err
	}

	damageType := models.ParseDamageType(effect.DamageType)
	totalHP, processed, hit := 0, 0, 0
	worst := models.ThresholdNone
	var primary *models.DamageResult
	// Reactions wait until every target has taken the damage, so they follow the action in the log
	var reacted []models.DamageResult
	var damaged []*models.Combatant
	for _, t := range fr.areaTargets(actor, target, state, activationResult) {
		// "Use the same roll against the other targets": it has to beat each one's defense too
		if effect.OthersOnly && (t == target || actor.LastRollTotal < t.GetDefense()) {
			continue
		}
		taken := fr.damageAfterReaction(effect, actor, t, damage, activationResult)
		damageRes := t.TakeDamageWithArmorCost(taken, damageType, fr.dice, effect.ArmorCost)
		reacted = append(reacted, damageRes)
		damaged = append(damaged, t)
		hit++
		totalHP += damageRes.HPDamage
		worst = max(worst, damageRes.Threshold)
		// The log row names the chosen target, so it shows that target's numbers when it was hit
		if t == target {
			primary = &damageRes
			processed = damageRes.ProcessedDamage
		} else if primary == nil {
			processed = max(processed, damageRes.ProcessedDamage)
		}
	}

	for i, t := range damaged {
		logReactions(state, t, reacted[i])
	}
	if totalHP > 0 {
		fr.afterDamageDealt(actor, state)
	}
	for i, t := range damaged {
		if reacted[i].HPDamage > 0 {
			fr.ProcessTriggeredFeatures(t, actor, state, models.TriggerOnDamageTaken, false)
		}
	}

	if activationResult != nil {
		activationResult.DamageRoll = damage
		activationResult.DamageParts = parts
		activationResult.ProcessedDamage = processed
		activationResult.DamageDealt = totalHP
		activationResult.Threshold = worst
		activationResult.TargetsHit = hit
		activationResult.DamageResult = primary
	}
	return nil
}
