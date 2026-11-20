package engine

import (
	"fmt"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func (fr *FeatureResolver) applyCondition(source, target *models.Combatant, condType models.ConditionType, duration string) error {
	return fr.addCondition(source, target, condType, duration, 0, models.DamagePhysical)
}

// applyDamagingCondition applies a condition that deals effect.Magnitude damage each turn
func (fr *FeatureResolver) applyDamagingCondition(source, target *models.Combatant, condType models.ConditionType, effect models.FeatureEffect) error {
	return fr.addCondition(source, target, condType, effect.Duration, effect.Magnitude, models.ParseDamageType(effect.DamageType))
}

func (fr *FeatureResolver) addCondition(source, target *models.Combatant, condType models.ConditionType, duration string, damagePerTick int, damageType models.DamageType) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	// A PC clears the condition by rolling against whoever applied it
	clearDifficulty := 0
	if source != nil && source != target {
		clearDifficulty = source.GetDefense()
	}

	durationType := models.DurationEndOfTurn
	switch duration {
	case "until_cleared", "permanent", "scene":
		durationType = models.DurationUntilCleared
	case "instant":
		return nil
	}

	target.AddCondition(&models.ActiveCondition{
		Type:            condType,
		Duration:        durationType,
		DamagePerTick:   damagePerTick,
		DamageType:      damageType,
		ClearDifficulty: clearDifficulty,
	})
	return nil
}

// rollAmount is an effect's Magnitude plus its DiceFormula roll, if any
func (fr *FeatureResolver) rollAmount(effect models.FeatureEffect) (int, error) {
	notation, err := models.ParseDiceNotation(effect.DiceFormula)
	if err != nil {
		return 0, fmt.Errorf("invalid dice formula %s: %w", effect.DiceFormula, err)
	}
	return effect.Magnitude + notation.Roll(fr.dice), nil
}

func (fr *FeatureResolver) resolveStressInflict(effect models.FeatureEffect, target *models.Combatant) error {
	if target == nil || !target.IsAlive() {
		return nil
	}
	amount, err := fr.rollAmount(effect)
	if err != nil {
		return err
	}
	target.AddStress(amount)
	return nil
}

// recipients resolves a self, ally or allies effect to the combatants it reaches
func (fr *FeatureResolver) recipients(effect models.FeatureEffect, actor *models.Combatant, state *models.CombatState, need func(*models.Combatant) int) []*models.Combatant {
	if actor == nil {
		return nil
	}
	if state == nil {
		return []*models.Combatant{actor}
	}
	return state.Recipients(actor, effect.TargetType, need)
}

func (fr *FeatureResolver) resolveStressClear(effect models.FeatureEffect, actor *models.Combatant, state *models.CombatState) error {
	amount, err := fr.rollAmount(effect)
	if err != nil {
		return err
	}
	for _, c := range fr.recipients(effect, actor, state, models.StressHeld) {
		c.ClearStress(amount)
	}
	return nil
}

func (fr *FeatureResolver) resolveHPHeal(effect models.FeatureEffect, actor *models.Combatant, state *models.CombatState) error {
	amount, err := fr.rollAmount(effect)
	if err != nil {
		return err
	}
	for _, c := range fr.recipients(effect, actor, state, models.MissingHP) {
		c.Heal(amount)
	}
	return nil
}

// lacksRecipient reports an ally-targeted heal or stress clear with nobody who would gain from it
func lacksRecipient(effects []models.FeatureEffect, actor *models.Combatant, state *models.CombatState) bool {
	if state == nil {
		return false
	}
	for _, e := range effects {
		if e.TargetType != models.FeatureTargetAlly {
			continue
		}
		switch e.Category {
		case models.FeatureEffectHPHeal:
			if len(state.Recipients(actor, e.TargetType, models.MissingHP)) == 0 {
				return true
			}
		case models.FeatureEffectStressClear:
			if len(state.Recipients(actor, e.TargetType, models.StressHeld)) == 0 {
				return true
			}
		}
	}
	return false
}

func (fr *FeatureResolver) resolveArmorDamage(effect models.FeatureEffect, target *models.Combatant, state *models.CombatState) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	if hpMarked := target.LoseArmorSlots(effect.Magnitude); hpMarked > 0 && effect.FallbackFear > 0 {
		state.AddFear(effect.FallbackFear)
	}
	return nil
}

func (fr *FeatureResolver) resolveGrantArmor(effect models.FeatureEffect, target *models.Combatant) error {
	if target == nil || !target.IsAlive() {
		return nil
	}
	target.ArmorSlots += effect.Magnitude
	return nil
}

func (fr *FeatureResolver) resolveHopeDrain(effect models.FeatureEffect, target *models.Combatant) error {
	if target == nil {
		return nil
	}
	target.LoseHope(effect.Magnitude)
	return nil
}
