package simulation

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/skarlso/daggerheart-combat-simulator/internal/engine"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/parser"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)

// SimulationConfig defines parameters for Monte Carlo simulation
type SimulationConfig struct {
	// Number of simulations to run
	RunCount int

	// Combat participants (templates to clone)
	PlayerTemplates []*models.Combatant
	EnemyTemplates  []*models.Combatant

	// Resource pools
	InitialHopePool int
	InitialFearPool int

	// Spotlight configuration
	SpotlightThreshold    int
	SpotlightWeightFactor float64

	// Strategy configurations
	PlayerStrategy strategy.StrategyConfig
	EnemyStrategy  strategy.StrategyConfig

	// Combat limits
	MaxTurnsPerCombat int

	// Random seed (0 = use time)
	RandomSeed int64

	// Adversary registry for summoning support (optional)
	AdversaryRegistry *parser.AdversaryRegistry
}

// DefaultSimulationConfig returns sensible defaults
func DefaultSimulationConfig() SimulationConfig {
	return SimulationConfig{
		RunCount:              100,
		InitialHopePool:       2,
		InitialFearPool:       5,
		SpotlightThreshold:    3,
		SpotlightWeightFactor: 0.2,
		PlayerStrategy:        strategy.DefaultDesperateConfig(),
		EnemyStrategy:         strategy.DefaultGreedyConfig(),
		MaxTurnsPerCombat:     100,
		RandomSeed:            0,
	}
}

// SimulationRunner executes Monte Carlo simulations
type SimulationRunner struct {
	config SimulationConfig
	rng    *rand.Rand
}

// NewSimulationRunner creates a runner with given config
func NewSimulationRunner(config SimulationConfig) *SimulationRunner {
	seed := config.RandomSeed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	return &SimulationRunner{
		config: config,
		rng:    rand.New(rand.NewSource(seed)),
	}
}

// Run executes all simulations and aggregates results
func (sr *SimulationRunner) Run() (*SimulationResult, error) {
	if len(sr.config.PlayerTemplates) == 0 {
		return nil, fmt.Errorf("no player templates provided")
	}
	if len(sr.config.EnemyTemplates) == 0 {
		return nil, fmt.Errorf("no enemy templates provided")
	}

	result := &SimulationResult{
		TotalRuns:      sr.config.RunCount,
		PlayerWins:     0,
		EnemyWins:      0,
		CombatOutcomes: make([]*engine.CombatOutcome, 0, sr.config.RunCount),
		AverageTurns:   0,
		AverageRounds:  0,
		MinTurns:       0,
		MaxTurns:       0,
	}

	totalTurns := 0
	totalRounds := 0

	for i := 0; i < sr.config.RunCount; i++ {
		outcome := sr.runSingleCombat()
		result.CombatOutcomes = append(result.CombatOutcomes, outcome)

		if outcome.PlayerVictory {
			result.PlayerWins++
		} else if outcome.EnemyVictory {
			result.EnemyWins++
		}

		totalTurns += outcome.Turns
		totalRounds += outcome.Rounds

		// Track min/max turns
		if i == 0 {
			result.MinTurns = outcome.Turns
			result.MaxTurns = outcome.Turns
		} else {
			if outcome.Turns < result.MinTurns {
				result.MinTurns = outcome.Turns
			}
			if outcome.Turns > result.MaxTurns {
				result.MaxTurns = outcome.Turns
			}
		}
	}

	// Calculate averages
	if result.TotalRuns > 0 {
		result.AverageTurns = float64(totalTurns) / float64(result.TotalRuns)
		result.AverageRounds = float64(totalRounds) / float64(result.TotalRuns)
		result.PlayerWinRate = float64(result.PlayerWins) / float64(result.TotalRuns)
		result.EnemyWinRate = float64(result.EnemyWins) / float64(result.TotalRuns)
	}

	return result, nil
}

// runSingleCombat executes one combat simulation
func (sr *SimulationRunner) runSingleCombat() *engine.CombatOutcome {
	// Clone combatants for this run
	players := sr.cloneCombatants(sr.config.PlayerTemplates)
	enemies := sr.cloneCombatants(sr.config.EnemyTemplates)

	// Create combat state
	state := models.NewCombatState(
		players,
		enemies,
		sr.config.InitialHopePool,
		sr.config.InitialFearPool,
		sr.config.SpotlightThreshold,
	)

	// Create engine components
	dice := engine.NewStandardDiceRoller(sr.rng)
	spotlight := engine.NewHybridSpotlightSelector(sr.rng, sr.config.SpotlightWeightFactor)
	combatEngine := engine.NewCombatEngine(dice, spotlight)

	// Set adversary registry if provided (enables summoning)
	if sr.config.AdversaryRegistry != nil {
		combatEngine.SetAdversaryRegistry(sr.config.AdversaryRegistry)
	}

	decisionMaker := strategy.NewUnifiedDecisionMaker(sr.rng, sr.config.PlayerStrategy, sr.config.EnemyStrategy)

	// Run combat (pass enemy Fear usage strategy for aggressive spotlight grabs)
	outcome := combatEngine.RunCombat(state, decisionMaker, sr.config.MaxTurnsPerCombat, sr.config.EnemyStrategy.ResourceUsage)

	return outcome
}

func (sr *SimulationRunner) cloneCombatants(templates []*models.Combatant) []*models.Combatant {
	clones := make([]*models.Combatant, len(templates))
	for i, template := range templates {
		clones[i] = models.CloneCombatant(template, template.ID)
	}
	return clones
}

// SimulationResult aggregates outcomes from multiple runs
type SimulationResult struct {
	TotalRuns      int
	PlayerWins     int
	EnemyWins      int
	PlayerWinRate  float64
	EnemyWinRate   float64
	AverageTurns   float64
	AverageRounds  float64
	MinTurns       int
	MaxTurns       int
	CombatOutcomes []*engine.CombatOutcome
}

// Summary returns a human-readable summary
func (sr *SimulationResult) Summary() string {
	return fmt.Sprintf(
		"Simulations: %d | Player Wins: %d (%.1f%%) | Enemy Wins: %d (%.1f%%) | Avg Turns: %.1f | Avg Rounds: %.1f",
		sr.TotalRuns,
		sr.PlayerWins,
		sr.PlayerWinRate*100,
		sr.EnemyWins,
		sr.EnemyWinRate*100,
		sr.AverageTurns,
		sr.AverageRounds,
	)
}
