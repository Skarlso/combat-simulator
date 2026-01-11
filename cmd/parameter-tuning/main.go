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
	fmt.Println("Daggerheart AI Parameter Tuning")
	fmt.Println("=================================")

	// Test different cost penalty multipliers
	fmt.Println("Testing Hope/Fear/Stress cost penalty multipliers...")

	// Test scenarios with different difficulty levels
	scenarios := []Scenario{
		{
			Name:    "Easy: 2v3 Goblins",
			Players: createStandardParty(),
			Enemies: createStandardEnemies(),
		},
		{
			Name:    "Hard: 2v5 Mixed Enemies",
			Players: createStandardParty(),
			Enemies: createHardEnemies(),
		},
		{
			Name:    "Boss Fight: 2v1 Elite",
			Players: createStandardParty(),
			Enemies: createBossFight(),
		},
	}

	// Test each scenario
	for _, scenario := range scenarios {
		fmt.Printf("\n=== %s ===\n\n", scenario.Name)
		tuneScenario(scenario)
	}
}

type Scenario struct {
	Name    string
	Players []*models.Combatant
	Enemies []*models.Combatant
}

func tuneScenario(scenario Scenario) {
	// Test baseline (current default)
	fmt.Println("Testing target selection strategies...")

	targetStrategies := []strategy.TargetSelectionStrategy{
		strategy.TargetLowestHP,
		strategy.TargetHighestHP,
		strategy.TargetHighestThreat,
		strategy.TargetRandom,
	}

	results := []TuningResult{}

	for _, targetStrat := range targetStrategies {
		config := strategy.StrategyConfig{
			ResourceUsage:       strategy.ResourceGreedy,
			ConditionClearing:   strategy.ClearNever,
			TargetSelection:     targetStrat,
			AbilityUsage:        strategy.AbilityAlways,
			LowHPThreshold:      30,
			HighThreatThreshold: 10,
		}

		result := runTuningTest(scenario, config)
		results = append(results, result)
	}

	// Display results
	fmt.Printf("\n%-25s | %8s | %8s | %8s\n", "Target Strategy", "Win Rate", "Avg Turns", "Efficiency")
	fmt.Println("--------------------------------------------------------------------")

	for _, result := range results {
		efficiency := calculateEfficiency(result)
		fmt.Printf("%-25s | %7.1f%% | %8.1f | %8.1f\n",
			result.StrategyName,
			result.WinRate*100,
			result.AvgTurns,
			efficiency,
		)
	}

	// Find optimal
	bestWinRate := results[0]
	bestEfficiency := results[0]

	for _, r := range results {
		if r.WinRate > bestWinRate.WinRate {
			bestWinRate = r
		}
		eff := calculateEfficiency(r)
		if eff > calculateEfficiency(bestEfficiency) {
			bestEfficiency = r
		}
	}

	fmt.Printf("\nRecommendation:\n")
	fmt.Printf("  Best Win Rate: %s (%.1f%%)\n", bestWinRate.StrategyName, bestWinRate.WinRate*100)
	fmt.Printf("  Most Efficient: %s (%.1f efficiency score)\n",
		bestEfficiency.StrategyName, calculateEfficiency(bestEfficiency))
}

type TuningResult struct {
	StrategyName string
	WinRate      float64
	AvgTurns     float64
	MinTurns     int
	MaxTurns     int
	AvgHope      float64
	AvgFear      float64
}

func runTuningTest(scenario Scenario, playerStrategy strategy.StrategyConfig) TuningResult {
	config := simulation.DefaultSimulationConfig()
	config.RunCount = 200 // Faster tuning runs
	config.PlayerTemplates = scenario.Players
	config.EnemyTemplates = scenario.Enemies
	config.PlayerStrategy = playerStrategy
	config.EnemyStrategy = strategy.DefaultGreedyConfig()

	runner := simulation.NewSimulationRunner(config)
	simResult, err := runner.Run()
	if err != nil {
		panic(err)
	}

	totalHope := 0.0
	totalFear := 0.0
	for _, outcome := range simResult.CombatOutcomes {
		totalHope += float64(outcome.FinalHopePool)
		totalFear += float64(outcome.FinalFearPool)
	}

	stratName := fmt.Sprintf("%s", playerStrategy.TargetSelection)

	return TuningResult{
		StrategyName: stratName,
		WinRate:      simResult.PlayerWinRate,
		AvgTurns:     simResult.AverageTurns,
		MinTurns:     simResult.MinTurns,
		MaxTurns:     simResult.MaxTurns,
		AvgHope:      totalHope / float64(simResult.TotalRuns),
		AvgFear:      totalFear / float64(simResult.TotalRuns),
	}
}

// calculateEfficiency scores strategy based on win rate and speed
func calculateEfficiency(result TuningResult) float64 {
	// Efficiency = winRate * 100 + (1 / avgTurns) * 50
	// Faster wins with high win rate = higher efficiency
	if result.AvgTurns == 0 {
		return 0
	}
	speedScore := (1.0 / result.AvgTurns) * 50.0
	return (result.WinRate * 100.0) + speedScore
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

func createHardEnemies() []*models.Combatant {
	enemies := createStandardEnemies()

	// Add two more tough enemies
	enemy4 := &models.Combatant{
		ID:               "e4",
		Name:             "Goblin Shaman",
		Side:             models.SideEnemy,
		Level:            1,
		HP:               5,
		HPMax:            5,
		Stress:           0,
		StressMax:        4,
		IsAdversary:      true,
		Tier:             "Tier 1",
		AdversaryType:    "Standard",
		Difficulty:       13,
		ATK:              5,
		CustomThresholds: []int{3, 6},
		DamageDice:       "1d8+2",
		Conditions:       make([]*models.ActiveCondition, 0),
	}

	enemy5 := &models.Combatant{
		ID:               "e5",
		Name:             "Goblin Brute",
		Side:             models.SideEnemy,
		Level:            1,
		HP:               6,
		HPMax:            6,
		Stress:           0,
		StressMax:        4,
		IsAdversary:      true,
		Tier:             "Tier 1",
		AdversaryType:    "Standard",
		Difficulty:       14,
		ATK:              6,
		CustomThresholds: []int{4, 7},
		DamageDice:       "2d6+2",
		ArmorSlots:       1,
		Conditions:       make([]*models.ActiveCondition, 0),
	}

	return append(enemies, enemy4, enemy5)
}

func createBossFight() []*models.Combatant {
	boss := &models.Combatant{
		ID:               "boss",
		Name:             "Goblin King",
		Side:             models.SideEnemy,
		Level:            2,
		HP:               20,
		HPMax:            20,
		Stress:           0,
		StressMax:        6,
		IsAdversary:      true,
		Tier:             "Tier 2",
		AdversaryType:    "Elite",
		Difficulty:       16,
		ATK:              8,
		CustomThresholds: []int{5, 10},
		DamageDice:       "2d8+4",
		ArmorSlots:       4,
		Conditions:       make([]*models.ActiveCondition, 0),
	}

	return []*models.Combatant{boss}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
