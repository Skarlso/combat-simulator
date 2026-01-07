package strategy

import (
	"math/rand"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// AIDecisionMaker implements strategic decision-making for combatants
type AIDecisionMaker struct {
	rng *rand.Rand

	// Strategy configurations
	ResourceUsage     ResourceUsageStrategy
	ConditionClearing ConditionClearStrategy
	TargetSelection   TargetSelectionStrategy
	AbilityUsage      AbilityUsageStrategy

	// Thresholds
	LowHPThreshold      int // Percentage (e.g., 30 = below 30% HP)
	HighThreatThreshold int // Damage threshold to consider enemy "high threat"
}

// NewAIDecisionMaker creates a decision maker with given strategies
func NewAIDecisionMaker(rng *rand.Rand, config StrategyConfig) *AIDecisionMaker {
	return &AIDecisionMaker{
		rng:                 rng,
		ResourceUsage:       config.ResourceUsage,
		ConditionClearing:   config.ConditionClearing,
		TargetSelection:     config.TargetSelection,
		AbilityUsage:        config.AbilityUsage,
		LowHPThreshold:      config.LowHPThreshold,
		HighThreatThreshold: config.HighThreatThreshold,
	}
}

// StrategyConfig bundles strategy settings
type StrategyConfig struct {
	ResourceUsage       ResourceUsageStrategy
	ConditionClearing   ConditionClearStrategy
	TargetSelection     TargetSelectionStrategy
	AbilityUsage        AbilityUsageStrategy
	LowHPThreshold      int
	HighThreatThreshold int
}

// DefaultGreedyConfig returns aggressive strategy
func DefaultGreedyConfig() StrategyConfig {
	return StrategyConfig{
		ResourceUsage:       ResourceGreedy,
		ConditionClearing:   ClearNever,
		TargetSelection:     TargetLowestHP,
		AbilityUsage:        AbilityAlways,
		LowHPThreshold:      30,
		HighThreatThreshold: 10,
	}
}

// DefaultConservativeConfig returns defensive strategy
func DefaultConservativeConfig() StrategyConfig {
	return StrategyConfig{
		ResourceUsage:       ResourceConservative,
		ConditionClearing:   ClearImmediately,
		TargetSelection:     TargetHighestThreat,
		AbilityUsage:        AbilitySparingly,
		LowHPThreshold:      50,
		HighThreatThreshold: 8,
	}
}

// DefaultDesperateConfig returns context-aware strategy
func DefaultDesperateConfig() StrategyConfig {
	return StrategyConfig{
		ResourceUsage:       ResourceDesperate,
		ConditionClearing:   ClearProbabilistic,
		TargetSelection:     TargetLowestHP,
		AbilityUsage:        AbilityContextual,
		LowHPThreshold:      40,
		HighThreatThreshold: 12,
	}
}

// DefaultRandomConfig returns unpredictable strategy
func DefaultRandomConfig() StrategyConfig {
	return StrategyConfig{
		ResourceUsage:       ResourceRandom,
		ConditionClearing:   ClearProbabilistic,
		TargetSelection:     TargetRandom,
		AbilityUsage:        AbilityContextual,
		LowHPThreshold:      35,
		HighThreatThreshold: 10,
	}
}

// ChooseAction selects a target for the actor to attack
// Returns nil if no valid target exists
func (ai *AIDecisionMaker) ChooseAction(state *models.CombatState, actor *models.Combatant) *models.Combatant {
	// Select target based on strategy
	target := ai.selectTarget(state, actor)
	return target
}

// selectTarget chooses a target from enemies
func (ai *AIDecisionMaker) selectTarget(state *models.CombatState, actor *models.Combatant) *models.Combatant {
	enemies := state.GetEnemiesOf(actor)
	return ai.selectTargetFrom(enemies)
}

// selectTargetFrom chooses from a list of candidates
func (ai *AIDecisionMaker) selectTargetFrom(candidates []*models.Combatant) *models.Combatant {
	if len(candidates) == 0 {
		return nil
	}

	switch ai.TargetSelection {
	case TargetLowestHP:
		lowest := candidates[0]
		for _, c := range candidates {
			if c.HP < lowest.HP {
				lowest = c
			}
		}
		return lowest

	case TargetHighestHP:
		highest := candidates[0]
		for _, c := range candidates {
			if c.HP > highest.HP {
				highest = c
			}
		}
		return highest

	case TargetRandom:
		return candidates[ai.rng.Intn(len(candidates))]

	case TargetHighestThreat:
		// Threat = proficiency + number of abilities
		highestThreat := candidates[0]
		maxThreat := ai.calculateThreat(highestThreat)

		for _, c := range candidates {
			threat := ai.calculateThreat(c)
			if threat > maxThreat {
				highestThreat = c
				maxThreat = threat
			}
		}
		return highestThreat

	default:
		return candidates[0]
	}
}

// calculateThreat estimates how dangerous a combatant is
func (ai *AIDecisionMaker) calculateThreat(combatant *models.Combatant) int {
	threat := combatant.Proficiency * 2

	// Higher HP = more threat potential
	hpPercent := (combatant.HP * 100) / combatant.HPMax
	threat += hpPercent / 10

	// Add threat based on features/abilities count
	threat += len(combatant.AdversaryFeatures)
	threat += len(combatant.DomainAbilities)

	return threat
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
