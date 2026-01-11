package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/simulation"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)

func main() {
	fmt.Println("Daggerheart Strategy Comparison")
	fmt.Println("=================================")

	// Define test scenarios
	players := createStandardParty()
	enemies := createStandardEnemies()

	// Define strategy variants to test
	strategies := []StrategyTest{
		{
			Name:   "Greedy (Focus Fire)",
			Config: strategy.DefaultGreedyConfig(),
		},
		{
			Name:   "Conservative (Threat Priority)",
			Config: strategy.DefaultConservativeConfig(),
		},
		{
			Name:   "Desperate (Adaptive)",
			Config: strategy.DefaultDesperateConfig(),
		},
		{
			Name:   "Random Chaos",
			Config: strategy.DefaultRandomConfig(),
		},
		{
			Name: "Focus Fire + Always Use Features",
			Config: strategy.StrategyConfig{
				ResourceUsage:       strategy.ResourceGreedy,
				ConditionClearing:   strategy.ClearNever,
				TargetSelection:     strategy.TargetLowestHP,
				AbilityUsage:        strategy.AbilityAlways,
				LowHPThreshold:      30,
				HighThreatThreshold: 10,
			},
		},
		{
			Name: "Tank Priority + Conservative",
			Config: strategy.StrategyConfig{
				ResourceUsage:       strategy.ResourceConservative,
				ConditionClearing:   strategy.ClearImmediately,
				TargetSelection:     strategy.TargetHighestHP,
				AbilityUsage:        strategy.AbilitySparingly,
				LowHPThreshold:      50,
				HighThreatThreshold: 8,
			},
		},
	}

	fmt.Printf("Testing %d strategies against each other...\n", len(strategies))
	fmt.Printf("Combat: %d players vs %d enemies\n\n", len(players), len(enemies))

	results := []StrategyResult{}

	for i, strat := range strategies {
		fmt.Printf("[%d/%d] Testing: %s\n", i+1, len(strategies), strat.Name)
		result := runStrategyTest(strat, players, enemies)
		results = append(results, result)
	}

	// Display results
	fmt.Println("\n=== STRATEGY COMPARISON RESULTS ===")
	fmt.Printf("%-40s | %8s | %8s | %8s | %8s | %8s\n",
		"Strategy", "Win Rate", "Avg Turns", "Min", "Max", "Avg Hope")
	fmt.Println("-------------------------------------------------------------------------------------------------------------------")

	for _, result := range results {
		fmt.Printf("%-40s | %7.1f%% | %8.1f | %8d | %8d | %8.1f\n",
			result.Name,
			result.WinRate*100,
			result.AvgTurns,
			result.MinTurns,
			result.MaxTurns,
			result.AvgFinalHope,
		)
	}

	// Find best strategy
	fmt.Println("\n=== BEST PERFORMERS ===")

	bestWinRate := results[0]
	bestSpeed := results[0]

	for _, r := range results {
		if r.WinRate > bestWinRate.WinRate {
			bestWinRate = r
		}
		if r.AvgTurns < bestSpeed.AvgTurns && r.WinRate > 0.95 {
			bestSpeed = r
		}
	}

	fmt.Printf("Highest Win Rate: %s (%.1f%%)\n", bestWinRate.Name, bestWinRate.WinRate*100)
	fmt.Printf("Fastest Victory:  %s (%.1f turns, %.1f%% win rate)\n",
		bestSpeed.Name, bestSpeed.AvgTurns, bestSpeed.WinRate*100)
}

type StrategyTest struct {
	Name   string
	Config strategy.StrategyConfig
}

type StrategyResult struct {
	Name          string
	WinRate       float64
	AvgTurns      float64
	MinTurns      int
	MaxTurns      int
	AvgFinalHope  float64
	AvgFinalFear  float64
}

func runStrategyTest(test StrategyTest, players, enemies []*models.Combatant) StrategyResult {
	config := simulation.DefaultSimulationConfig()
	config.RunCount = 500 // Run 500 simulations per strategy
	config.PlayerTemplates = players
	config.EnemyTemplates = enemies
	config.PlayerStrategy = test.Config
	config.EnemyStrategy = strategy.DefaultGreedyConfig() // Standard enemy strategy

	runner := simulation.NewSimulationRunner(config)
	simResult, err := runner.Run()
	if err != nil {
		panic(err)
	}

	// Calculate average final Hope
	totalHope := 0.0
	totalFear := 0.0
	for _, outcome := range simResult.CombatOutcomes {
		totalHope += float64(outcome.FinalHopePool)
		totalFear += float64(outcome.FinalFearPool)
	}

	return StrategyResult{
		Name:          test.Name,
		WinRate:       simResult.PlayerWinRate,
		AvgTurns:      simResult.AverageTurns,
		MinTurns:      simResult.MinTurns,
		MaxTurns:      simResult.MaxTurns,
		AvgFinalHope:  totalHope / float64(simResult.TotalRuns),
		AvgFinalFear:  totalFear / float64(simResult.TotalRuns),
	}
}

func createStandardParty() []*models.Combatant {
	player1 := models.NewCombatant("p1", "Warrior", models.SidePlayer, 1, 14, 2, 8, 5)
	player1.DamageDice = "2d6+2"
	player1.ArmorSlots = 3

	player2 := models.NewCombatant("p2", "Rogue", models.SidePlayer, 1, 12, 3, 6, 5)
	player2.DamageDice = "2d6+1"
	player2.ArmorSlots = 2

	return []*models.Combatant{player1, player2}
}

func createStandardEnemies() []*models.Combatant {
	enemy1 := &models.Combatant{
		ID:               "e1",
		Name:             "Goblin Archer",
		Side:             models.SideEnemy,
		Level:            1,
		HP:               4,
		HPMax:            4,
		Stress:           0,
		StressMax:        3,
		IsAdversary:      true,
		Tier:             "Tier 1",
		AdversaryType:    "Standard",
		Difficulty:       12,
		ATK:              4,
		CustomThresholds: []int{3, 6},
		DamageDice:       "1d6+1",
		Conditions:       make([]*models.ActiveCondition, 0),
	}

	enemy2 := &models.Combatant{
		ID:               "e2",
		Name:             "Goblin Warrior",
		Side:             models.SideEnemy,
		Level:            1,
		HP:               4,
		HPMax:            4,
		Stress:           0,
		StressMax:        3,
		IsAdversary:      true,
		Tier:             "Tier 1",
		AdversaryType:    "Standard",
		Difficulty:       13,
		ATK:              5,
		CustomThresholds: []int{3, 6},
		DamageDice:       "1d6+2",
		Conditions:       make([]*models.ActiveCondition, 0),
	}

	enemy3 := &models.Combatant{
		ID:               "e3",
		Name:             "Goblin Boss",
		Side:             models.SideEnemy,
		Level:            1,
		HP:               7,
		HPMax:            7,
		Stress:           0,
		StressMax:        4,
		IsAdversary:      true,
		Tier:             "Tier 1",
		AdversaryType:    "Elite",
		Difficulty:       14,
		ATK:              6,
		CustomThresholds: []int{4, 7},
		DamageDice:       "2d6+1",
		ArmorSlots:       2,
		Conditions:       make([]*models.ActiveCondition, 0),
	}

	return []*models.Combatant{enemy1, enemy2, enemy3}
}

// Seed RNG
func init() {
	rand.Seed(time.Now().UnixNano())
}
