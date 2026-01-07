package main

import (
	"fmt"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/simulation"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)

func main() {
	fmt.Println("Daggerheart Combat Simulator - Example")
	fmt.Println("=======================================")
	fmt.Println()

	// Create player characters
	rogue := models.NewCombatant("p1", "Grimjaw the Rogue", models.SidePlayer, 1, 12, 2, 6, 6)
	rogue.DamageDice = "2d6+1"

	fighter := models.NewCombatant("p2", "Thora the Fighter", models.SidePlayer, 1, 11, 2, 8, 6)
	fighter.ArmorSlots = 2
	fighter.DamageDice = "2d8+2"

	// Create enemies (level 0 for adversaries - doesn't affect their thresholds)
	goblin1 := models.NewCombatant("e1", "Goblin Archer", models.SideEnemy, 0, 10, 1, 4, 4)
	goblin1.DamageDice = "1d6"

	goblin2 := models.NewCombatant("e2", "Goblin Warrior", models.SideEnemy, 0, 10, 1, 4, 4)
	goblin2.DamageDice = "1d6+1"

	goblinBoss := models.NewCombatant("e3", "Goblin Boss", models.SideEnemy, 0, 11, 2, 7, 5)
	goblinBoss.DamageDice = "2d6+2"

	// Configure simulation
	config := simulation.DefaultSimulationConfig()
	config.RunCount = 1000
	config.PlayerTemplates = []*models.Combatant{rogue, fighter}
	config.EnemyTemplates = []*models.Combatant{goblin1, goblin2, goblinBoss}
	config.PlayerStrategy = strategy.DefaultDesperateConfig()
	config.EnemyStrategy = strategy.DefaultGreedyConfig()
	config.SpotlightThreshold = 3

	fmt.Println("Combat Setup:")
	fmt.Printf("  Players: %s (HP: %d), %s (HP: %d)\n", rogue.Name, rogue.HP, fighter.Name, fighter.HP)
	fmt.Printf("  Enemies: %s (HP: %d), %s (HP: %d), %s (HP: %d)\n",
		goblin1.Name, goblin1.HP,
		goblin2.Name, goblin2.HP,
		goblinBoss.Name, goblinBoss.HP)
	fmt.Printf("  Player Strategy: %s\n", config.PlayerStrategy.ResourceUsage)
	fmt.Printf("  Enemy Strategy: %s\n", config.EnemyStrategy.ResourceUsage)
	fmt.Printf("  Simulations: %d\n\n", config.RunCount)

	// Run simulation
	fmt.Println("Running simulations...")
	runner := simulation.NewSimulationRunner(config)
	result, err := runner.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Display results
	fmt.Println("\nResults:")
	fmt.Println("========")
	fmt.Println(result.Summary())
	fmt.Printf("\nTurn Statistics:\n")
	fmt.Printf("  Min Turns: %d\n", result.MinTurns)
	fmt.Printf("  Max Turns: %d\n", result.MaxTurns)
	fmt.Printf("  Avg Turns: %.1f\n", result.AverageTurns)

	// Show sample outcomes
	fmt.Println("\nSample Outcomes (first 5):")
	for i := 0; i < 5 && i < len(result.CombatOutcomes); i++ {
		outcome := result.CombatOutcomes[i]
		winner := "Enemy"
		if outcome.PlayerVictory {
			winner = "Player"
		}
		fmt.Printf("  #%d: %s victory in %d turns\n", i+1, winner, outcome.Turns)
	}
}
