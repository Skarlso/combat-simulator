package simulation

import (
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)

func newTestConfig() SimulationConfig {
	config := DefaultSimulationConfig()
	config.RandomSeed = 42
	config.RunCount = 10
	config.PlayerTemplates = []*models.Combatant{
		models.NewCombatant("p1", "Hero", models.SidePlayer, 3, 12, 2, 15, 8),
	}
	config.PlayerTemplates[0].DamageDice = "1d6+2"

	config.EnemyTemplates = []*models.Combatant{
		models.NewAdversary("e1", "Goblin", "Tier 1", "Standard", 10, 3, 8, 4, []int{4, 7}),
	}
	config.EnemyTemplates[0].DamageDice = "1d6"
	return config
}

func TestDefaultSimulationConfig(t *testing.T) {
	config := DefaultSimulationConfig()
	if config.RunCount != 100 {
		t.Errorf("RunCount = %d, want 100", config.RunCount)
	}
	if config.MaxTurnsPerCombat != 100 {
		t.Errorf("MaxTurnsPerCombat = %d, want 100", config.MaxTurnsPerCombat)
	}
}

func TestRunSimulation(t *testing.T) {
	config := newTestConfig()
	runner := NewSimulationRunner(config)

	result, err := runner.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.TotalRuns != 10 {
		t.Errorf("TotalRuns = %d, want 10", result.TotalRuns)
	}
	if result.PlayerWins+result.EnemyWins == 0 {
		t.Error("should have at least one winner")
	}
	if result.AverageTurns <= 0 {
		t.Errorf("AverageTurns = %f, want > 0", result.AverageTurns)
	}
}

func TestRunSimulationNoPlayers(t *testing.T) {
	config := newTestConfig()
	config.PlayerTemplates = nil
	runner := NewSimulationRunner(config)

	_, err := runner.Run()
	if err == nil {
		t.Error("expected error for no players")
	}
}

func TestRunSimulationNoEnemies(t *testing.T) {
	config := newTestConfig()
	config.EnemyTemplates = nil
	runner := NewSimulationRunner(config)

	_, err := runner.Run()
	if err == nil {
		t.Error("expected error for no enemies")
	}
}

func TestRunSimulationDeterministic(t *testing.T) {
	config := newTestConfig()

	runner1 := NewSimulationRunner(config)
	result1, err := runner1.Run()
	if err != nil {
		t.Fatalf("runner1 error: %v", err)
	}

	runner2 := NewSimulationRunner(config)
	result2, err := runner2.Run()
	if err != nil {
		t.Fatalf("runner2 error: %v", err)
	}

	if result1.PlayerWins != result2.PlayerWins {
		t.Errorf("same seed should produce same results: wins %d vs %d", result1.PlayerWins, result2.PlayerWins)
	}
}

func TestRunSimulationMultipleParticipants(t *testing.T) {
	config := newTestConfig()
	config.PlayerTemplates = append(config.PlayerTemplates,
		models.NewCombatant("p2", "Mage", models.SidePlayer, 3, 10, 3, 12, 8),
	)
	config.PlayerTemplates[1].DamageDice = "1d8"

	config.EnemyTemplates = append(config.EnemyTemplates,
		models.NewAdversary("e2", "Orc", "Tier 1", "Bruiser", 12, 5, 15, 6, []int{5, 10}),
	)
	config.EnemyTemplates[1].DamageDice = "1d8+2"

	runner := NewSimulationRunner(config)
	result, err := runner.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalRuns != 10 {
		t.Errorf("TotalRuns = %d, want 10", result.TotalRuns)
	}
}

func TestRunSimulationWithStrategies(t *testing.T) {
	configs := []struct {
		name           string
		playerStrategy strategy.StrategyConfig
	}{
		{"greedy", strategy.DefaultGreedyConfig()},
		{"conservative", strategy.DefaultConservativeConfig()},
		{"desperate", strategy.DefaultDesperateConfig()},
		{"random", strategy.DefaultRandomConfig()},
	}

	for _, tt := range configs {
		t.Run(tt.name, func(t *testing.T) {
			config := newTestConfig()
			config.PlayerStrategy = tt.playerStrategy
			runner := NewSimulationRunner(config)

			result, err := runner.Run()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.PlayerWins+result.EnemyWins == 0 {
				t.Error("should have at least one winner")
			}
		})
	}
}

func TestDefaultFearIsOnePerPC(t *testing.T) {
	config := newTestConfig()
	config.RunCount = 1
	config.MaxTurnsPerCombat = 0
	config.PlayerTemplates = append(config.PlayerTemplates,
		models.NewCombatant("p2", "Mage", models.SidePlayer, 3, 10, 2, 10, 8))

	result, err := NewSimulationRunner(config).Run()
	if err != nil {
		t.Fatal(err)
	}

	outcome := result.CombatOutcomes[0]
	if outcome.FinalFearPool != 2 {
		t.Errorf("FinalFearPool = %d, want 2 (1 per PC)", outcome.FinalFearPool)
	}
	if outcome.FinalHopePool != 4 {
		t.Errorf("FinalHopePool = %d, want 4 (2 per PC)", outcome.FinalHopePool)
	}
}
