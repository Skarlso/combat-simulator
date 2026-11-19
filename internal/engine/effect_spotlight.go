package engine

import (
	"fmt"
	"strings"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// resolveAllySpotlight queues spotlights for the actor's allies; the combat engine
// drains the queue once the current action has resolved.
func (fr *FeatureResolver) resolveAllySpotlight(effect models.FeatureEffect, actor *models.Combatant, state *models.CombatState) error {
	candidates := state.EligibleSpotlightAllies(actor, effect)
	if len(candidates) == 0 {
		return fmt.Errorf("no allies to spotlight")
	}

	count := len(candidates)
	if !effect.SpotlightAll {
		n, err := fr.rollAmount(effect)
		if err != nil {
			return err
		}
		count = min(n, len(candidates))
	}
	if count <= 0 {
		return fmt.Errorf("spotlight count must be positive, got %d", count)
	}

	// Fisher-Yates prefix so the picked allies are random rather than list order
	for i := 0; i < count; i++ {
		if remaining := len(candidates) - i; remaining > 1 {
			j := i + fr.dice.RollDice(1, remaining, 0) - 1
			candidates[i], candidates[j] = candidates[j], candidates[i]
		}
	}

	names := make([]string, 0, count)
	for _, ally := range candidates[:count] {
		state.QueueSpotlight(ally, effect.HalfDamage)
		names = append(names, ally.Name)
	}

	msg := fmt.Sprintf("Spotlights %s", strings.Join(names, ", "))
	if effect.HalfDamage {
		msg += " (half damage)"
	}
	state.RecordAction(models.ActionParams{
		Actor:      actor,
		ActionType: models.ActionUseAbility,
		Result:     msg,
	})
	return nil
}
