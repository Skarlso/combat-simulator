package engine

import (
	"fmt"
	"slices"

	"github.com/skarlso/combat-simulator/internal/models"
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

func (fr *FeatureResolver) resolveStressClear(effect models.FeatureEffect, actor *models.Combatant, state *models.CombatState, result *DCCheckResult) error {
	amount, err := fr.rollAmount(effect)
	if err != nil {
		return err
	}
	for _, c := range fr.recipients(effect, actor, state, models.StressHeld) {
		before := models.StateOf(c)
		held := c.Stress
		c.ClearStress(amount)
		result.restored(c, before, held-c.Stress, "Stress")
	}
	return nil
}

func (fr *FeatureResolver) resolveHPHeal(effect models.FeatureEffect, actor *models.Combatant, state *models.CombatState, result *DCCheckResult) error {
	amount, err := fr.rollAmount(effect)
	if err != nil {
		return err
	}
	for _, c := range fr.recipients(effect, actor, state, models.MissingHP) {
		before := models.StateOf(c)
		hp := c.HP
		c.Heal(amount)
		result.restored(c, before, c.HP-hp, "HP")
	}
	return nil
}

// restored notes whom a restoring effect reached and what it cleared
func (r *DCCheckResult) restored(c *models.Combatant, before *models.CombatantState, cleared int, what string) {
	if r == nil || cleared <= 0 {
		return
	}
	if !slices.Contains(r.Restored, c) {
		if len(r.Restored) == 0 {
			r.restoredBefore = before
		}
		r.Restored = append(r.Restored, c)
	}
	r.Notes = append(r.Notes, fmt.Sprintf("%s clears %d %s", c.Name, cleared, what))
}

// restoresOnly reports an ability that does nothing but heal, clear Stress or clear Armor Slots
func restoresOnly(effects []models.FeatureEffect) bool {
	for _, e := range effects {
		switch e.Category {
		case models.FeatureEffectHPHeal, models.FeatureEffectStressClear, models.FeatureEffectArmorClear:
		default:
			return false
		}
	}
	return len(effects) > 0
}

// lacksRecipient reports an ally-targeted heal or stress clear with nobody who would gain from it,
// or an armor clear its recipient cannot use in full
func lacksRecipient(effects []models.FeatureEffect, actor *models.Combatant, state *models.CombatState) bool {
	if state == nil {
		return false
	}
	// An ability that does nothing but restore waits until its whole amount is needed
	onlyRestores := restoresOnly(effects)

	for _, e := range effects {
		var need func(*models.Combatant) int
		switch e.Category {
		case models.FeatureEffectHPHeal:
			need = models.MissingHP
		case models.FeatureEffectStressClear:
			need = models.StressHeld
		case models.FeatureEffectArmorClear:
			need = fullArmorClear(e)
		default:
			continue
		}
		switch {
		case e.TargetType == models.FeatureTargetAlly:
			if len(state.Recipients(actor, e.TargetType, need)) == 0 {
				return true
			}
		case e.TargetType == models.FeatureTargetAllies:
			if onlyRestores && !slices.ContainsFunc(state.GetAlliesOf(actor), func(c *models.Combatant) bool { return need(c) > 0 }) {
				return true
			}
		case e.Category == models.FeatureEffectArmorClear:
			if need(actor) == 0 {
				return true
			}
		case onlyRestores:
			if need(actor) < max(e.Magnitude, 1) {
				return true
			}
		}
	}
	return false
}

// fullArmorClear scores a combatant for an armor clear: their marked slots, or 0 while
// fewer are marked than the effect clears, so its cost is never spent on part of it
func fullArmorClear(e models.FeatureEffect) func(*models.Combatant) int {
	return func(c *models.Combatant) int {
		marked := models.MarkedArmor(c)
		if marked < min(e.Magnitude, c.ArmorMax) {
			return 0
		}
		return marked
	}
}

func (fr *FeatureResolver) resolveArmorDamage(effect models.FeatureEffect, target *models.Combatant, state *models.CombatState, result *DCCheckResult) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	hpMarked := target.LoseArmorSlots(effect.Magnitude)
	if hpMarked > 0 && result != nil {
		result.Notes = append(result.Notes, fmt.Sprintf("%s marks %d HP for having no Armor Slot to mark", target.Name, hpMarked))
	}
	if hpMarked > 0 && effect.FallbackFear > 0 {
		state.AddFearFor(effect.FallbackFear, "no Armor Slot to mark")
	}
	return nil
}

// resolveGrantArmor adds Armor Slots beyond the Armor Score to the actor's side
func (fr *FeatureResolver) resolveGrantArmor(effect models.FeatureEffect, actor *models.Combatant, state *models.CombatState) error {
	for _, c := range fr.recipients(effect, actor, state, func(c *models.Combatant) int { return 1 }) {
		c.ArmorSlots += effect.Magnitude
	}
	return nil
}

func (fr *FeatureResolver) resolveArmorClear(effect models.FeatureEffect, actor *models.Combatant, state *models.CombatState, result *DCCheckResult) error {
	for _, c := range fr.recipients(effect, actor, state, models.MarkedArmor) {
		before := models.StateOf(c)
		result.restored(c, before, c.ClearArmorSlots(effect.Magnitude), "Armor Slots")
	}
	return nil
}

// resolveMarkHP forces the target to mark Hit Points, whatever their thresholds and armor
func (fr *FeatureResolver) resolveMarkHP(effect models.FeatureEffect, target *models.Combatant, result *DCCheckResult) error {
	if target == nil || !target.IsAlive() {
		return nil
	}
	before := target.HP
	target.MarkHP(effect.Magnitude)
	if result != nil {
		result.DamageDealt += before - target.HP
	}
	return nil
}

func (fr *FeatureResolver) resolveHopeDrain(effect models.FeatureEffect, target *models.Combatant) error {
	if target == nil {
		return nil
	}
	target.LoseHope(effect.Magnitude)
	return nil
}
