package engine

import (
	"fmt"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func (fr *FeatureResolver) resolveSummon(effect models.FeatureEffect, actor *models.Combatant, state *models.CombatState) error {
	quantity := effect.Magnitude
	if effect.DiceFormula != "" {
		notation, err := models.ParseDiceNotation(effect.DiceFormula)
		if err != nil {
			return fmt.Errorf("failed to parse summon dice formula: %w", err)
		}
		quantity += notation.Roll(fr.dice)
	}

	if quantity <= 0 {
		return fmt.Errorf("summon quantity must be positive, got %d", quantity)
	}

	if len(effect.SummonPool) > 0 {
		return fr.summonFromPool(effect.SummonPool, quantity, actor, state)
	}

	if effect.SummonType != "" && fr.registry != nil {
		return fr.summonFromRegistry(effect.SummonType, quantity, actor, state)
	}

	return fmt.Errorf("summon effect requires either SummonPool or SummonType with registry")
}

func (fr *FeatureResolver) summonFromPool(pool []*models.Combatant, quantity int, actor *models.Combatant, state *models.CombatState) error {
	if len(pool) == 0 {
		return fmt.Errorf("summon pool is empty")
	}

	var summoned []*models.Combatant
	for range quantity {
		template := pool[fr.dice.RollDice(1, len(pool), 0)-1]
		state.SummonCounter++
		state.SummonedCounts[template.Name]++
		numberedName := fmt.Sprintf("%s #%d", template.Name, state.SummonedCounts[template.Name])
		uniqueID := fmt.Sprintf("%s_summoned_%d", template.Name, state.SummonCounter)

		clone := models.CloneCombatant(template, uniqueID)
		clone.Name = numberedName
		state.SummonCombatant(clone)
		summoned = append(summoned, clone)
	}

	fr.recordSummon(actor, summoned, state)
	return nil
}

func (fr *FeatureResolver) summonFromRegistry(summonType string, quantity int, actor *models.Combatant, state *models.CombatState) error {
	var summoned []*models.Combatant
	for range quantity {
		state.SummonCounter++
		state.SummonedCounts[summonType]++
		uniqueID := fmt.Sprintf("%s_summoned_%d", summonType, state.SummonCounter)

		clone, err := fr.registry.CloneTemplate(summonType, uniqueID)
		if err != nil {
			return fmt.Errorf("failed to summon %s: %w", summonType, err)
		}

		clone.Name = fmt.Sprintf("%s #%d", summonType, state.SummonedCounts[summonType])
		state.SummonCombatant(clone)
		summoned = append(summoned, clone)
	}

	fr.recordSummon(actor, summoned, state)
	return nil
}

func (fr *FeatureResolver) recordSummon(actor *models.Combatant, summoned []*models.Combatant, state *models.CombatState) {
	msg := fmt.Sprintf("Summons %d allies", len(summoned))
	if len(summoned) == 1 {
		msg = fmt.Sprintf("Summons %s", summoned[0].Name)
	}
	state.RecordAction(models.ActionParams{
		Actor:      actor,
		ActionType: models.ActionUseAbility,
		Result:     msg,
	})
}
