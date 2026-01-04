package engine

import (
	"fmt"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// EffectApplier applies ability effects to targets
type EffectApplier struct {
	resolver *CombatResolver
}

// NewEffectApplier creates an effect applier
func NewEffectApplier(resolver *CombatResolver) *EffectApplier {
	return &EffectApplier{
		resolver: resolver,
	}
}

// ApplyEffect applies a single effect to target(s)
func (ea *EffectApplier) ApplyEffect(effect models.Effect, actor *models.Combatant, targets []*models.Combatant, state *models.CombatState) error {
	switch e := effect.(type) {
	case models.DamageEffect:
		return ea.applyDamageEffect(e, actor, targets, state)
	case models.HealEffect:
		return ea.applyHealEffect(e, targets)
	case models.AddConditionEffect:
		return ea.applyAddConditionEffect(e, actor, targets)
	case models.RemoveConditionEffect:
		return ea.applyRemoveConditionEffect(e, targets)
	case models.ModifyResourceEffect:
		return ea.applyModifyResourceEffect(e, actor, state)
	default:
		return fmt.Errorf("unknown effect type: %T", effect)
	}
}

// applyDamageEffect deals damage to targets
func (ea *EffectApplier) applyDamageEffect(effect models.DamageEffect, actor *models.Combatant, targets []*models.Combatant, state *models.CombatState) error {
	for _, target := range targets {
		if target == nil || !target.IsAlive() {
			continue
		}

		// Use combat resolver to handle the attack
		ea.resolver.ResolveAttack(actor, target, effect, state)
	}
	return nil
}

// applyHealEffect restores HP to targets
func (ea *EffectApplier) applyHealEffect(effect models.HealEffect, targets []*models.Combatant) error {
	for _, target := range targets {
		if target == nil {
			continue
		}

		ea.resolver.ApplyHealing(target, effect)
	}
	return nil
}

// applyAddConditionEffect applies status effects
func (ea *EffectApplier) applyAddConditionEffect(effect models.AddConditionEffect, actor *models.Combatant, targets []*models.Combatant) error {
	// Determine who gets the condition
	var recipients []*models.Combatant

	switch effect.Target {
	case models.TargetSelf:
		recipients = []*models.Combatant{actor}
	case models.TargetOther:
		recipients = targets
	default:
		recipients = targets
	}

	for _, recipient := range recipients {
		if recipient == nil || !recipient.IsAlive() {
			continue
		}

		// Clone the condition to avoid sharing state
		conditionCopy := *effect.Condition
		recipient.AddCondition(&conditionCopy)
	}

	return nil
}

// applyRemoveConditionEffect clears status effects
func (ea *EffectApplier) applyRemoveConditionEffect(effect models.RemoveConditionEffect, targets []*models.Combatant) error {
	for _, target := range targets {
		if target == nil {
			continue
		}

		target.RemoveCondition(effect.ConditionType)
	}
	return nil
}

// applyModifyResourceEffect changes Hope/Fear/Stress/HP pools
func (ea *EffectApplier) applyModifyResourceEffect(effect models.ModifyResourceEffect, actor *models.Combatant, state *models.CombatState) error {
	if effect.TargetSelf {
		// Modify actor's personal resources
		switch effect.ResourceType {
		case models.ResourceHP:
			if effect.Amount > 0 {
				actor.Heal(effect.Amount)
			} else {
				actor.TakeDamage(-effect.Amount, models.DamagePhysical)
			}
		case models.ResourceStress:
			if effect.Amount > 0 {
				actor.AddStress(effect.Amount)
			} else {
				actor.ClearStress(-effect.Amount)
			}
		case models.ResourceArmor:
			if effect.Amount > 0 {
				actor.ArmorSlots += effect.Amount
			}
		}
	} else {
		// Modify global pools
		switch effect.ResourceType {
		case models.ResourceHope:
			if effect.Amount > 0 {
				state.AddHope(effect.Amount)
			} else {
				state.SpendHope(-effect.Amount)
			}
		case models.ResourceFear:
			if effect.Amount > 0 {
				state.AddFear(effect.Amount)
			} else {
				state.SpendFear(-effect.Amount)
			}
		}
	}

	return nil
}

// ApplyAllEffects applies all effects from an ability
func (ea *EffectApplier) ApplyAllEffects(ability *models.Ability, actor *models.Combatant, targets []*models.Combatant, state *models.CombatState) error {
	for _, effect := range ability.Effects {
		if err := ea.ApplyEffect(effect, actor, targets, state); err != nil {
			return err
		}
	}
	return nil
}
