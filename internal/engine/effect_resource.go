package engine

import "github.com/skarlso/daggerheart-combat-simulator/internal/models"

func (fr *FeatureResolver) applyCondition(target *models.Combatant, condType models.ConditionType, duration string) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	var durationType models.DurationType
	switch duration {
	case "until_cleared", "permanent":
		durationType = models.DurationUntilCleared
	case "1_turn":
		durationType = models.DurationEndOfTurn
	case "instant":
		return nil
	case "scene":
		durationType = models.DurationUntilCleared
	default:
		durationType = models.DurationEndOfTurn
	}

	target.AddCondition(&models.ActiveCondition{
		Type:     condType,
		Duration: durationType,
	})
	return nil
}

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

func (fr *FeatureResolver) resolveStressClear(effect models.FeatureEffect, actor *models.Combatant) error {
	if actor == nil {
		return nil
	}
	actor.ClearStress(effect.Magnitude)
	return nil
}

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

func (fr *FeatureResolver) resolveArmorDamage(effect models.FeatureEffect, target *models.Combatant) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	damage := min(effect.Magnitude, target.ArmorSlots)
	target.ArmorSlots -= damage
	return nil
}

func (fr *FeatureResolver) resolveGrantArmor(effect models.FeatureEffect, target *models.Combatant) error {
	if target == nil || !target.IsAlive() {
		return nil
	}
	target.ArmorSlots += effect.Magnitude
	return nil
}

func (fr *FeatureResolver) resolveHopeDrain(effect models.FeatureEffect, state *models.CombatState) error {
	state.SpendHope(effect.Magnitude)
	return nil
}

func (fr *FeatureResolver) resolveFearGenerate(effect models.FeatureEffect, state *models.CombatState) error {
	state.AddFear(effect.Magnitude)
	return nil
}
