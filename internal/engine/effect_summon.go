package engine

import (
	"fmt"

	"github.com/skarlso/combat-simulator/internal/models"
)

func (fr *FeatureResolver) resolveSummon(effect models.FeatureEffect, actor *models.Combatant, state *models.CombatState) error {
	quantity, err := fr.rollAmount(effect)
	if err != nil {
		return err
	}
	if quantity <= 0 {
		return fmt.Errorf("summon quantity must be positive, got %d", quantity)
	}

	var summoned []*models.Combatant
	switch {
	case len(effect.SummonPool) > 0:
		summoned, err = fr.summonFromPool(effect.SummonPool, quantity, actor, state)
	case effect.SummonType != "" && fr.registry != nil:
		summoned, err = fr.summonFromRegistry(effect.SummonType, quantity, actor, state)
	default:
		return fmt.Errorf("summon effect requires either SummonPool or SummonType with registry")
	}
	if err != nil {
		return err
	}

	fr.recordSummon(actor, summoned, state)
	fr.queueSummonedSpotlights(effect, summoned, state)
	return nil
}

// queueSummonedSpotlights hands spotlights to "appear and immediately take the spotlight" arrivals
func (fr *FeatureResolver) queueSummonedSpotlights(effect models.FeatureEffect, summoned []*models.Combatant, state *models.CombatState) {
	count := effect.SpotlightCount
	if effect.SpotlightAll {
		count = len(summoned)
	}
	for i := 0; i < count && i < len(summoned); i++ {
		state.QueueSpotlight(summoned[i], effect.HalfDamage)
	}
}

func (fr *FeatureResolver) summonFromPool(pool []*models.Combatant, quantity int, actor *models.Combatant, state *models.CombatState) ([]*models.Combatant, error) {
	if len(pool) == 0 {
		return nil, fmt.Errorf("summon pool is empty")
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

	return summoned, nil
}

func (fr *FeatureResolver) summonFromRegistry(summonType string, quantity int, actor *models.Combatant, state *models.CombatState) ([]*models.Combatant, error) {
	var summoned []*models.Combatant
	for range quantity {
		state.SummonCounter++
		state.SummonedCounts[summonType]++
		uniqueID := fmt.Sprintf("%s_summoned_%d", summonType, state.SummonCounter)

		clone, err := fr.registry.CloneTemplate(summonType, uniqueID)
		if err != nil {
			return nil, fmt.Errorf("failed to summon %s: %w", summonType, err)
		}

		clone.Name = fmt.Sprintf("%s #%d", summonType, state.SummonedCounts[summonType])
		state.SummonCombatant(clone)
		summoned = append(summoned, clone)
	}

	return summoned, nil
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
