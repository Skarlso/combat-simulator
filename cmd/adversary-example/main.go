package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/skarlso/daggerheart-combat-simulator/internal/engine"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/parser"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)


func main() {
	fmt.Println("Daggerheart Combat Simulator - Adversary Feature Test")
	fmt.Println("=======================================================")

	// Load adversaries with feature parsing
	advParser, err := parser.NewAdversaryParser("data/feature_mappings.json")
	if err != nil {
		fmt.Printf("Failed to create adversary parser: %v\n", err)
		return
	}

	adversaries, err := advParser.LoadAdversariesFromFile("data/daggerheart_adversaries-tier-1.json")
	if err != nil {
		fmt.Printf("Failed to load adversaries: %v\n", err)
		return
	}

	fmt.Printf("Loaded %d adversaries from tier 1\n\n", len(adversaries))

	// Find adversaries with mapped features
	var testAdversaries []*models.Combatant
	for _, adv := range adversaries {
		if len(adv.AdversaryFeatures) > 0 {
			testAdversaries = append(testAdversaries, adv)
			fmt.Printf("✓ %s: %d mapped features\n", adv.Name, len(adv.AdversaryFeatures))
			for _, f := range adv.AdversaryFeatures {
				fmt.Printf("  - %s (Fear:%d, %d effects)\n", f.Name, f.FearCost, len(f.Effects))
			}
		}
	}

	if len(testAdversaries) == 0 {
		fmt.Println("\nNo adversaries with mapped features found.")
		return
	}

	// Pick first adversary with features for combat
	enemy := testAdversaries[0]
	fmt.Printf("\n\nRunning combat simulation with: %s\n", enemy.Name)
	fmt.Printf("HP: %d, Difficulty: %d, Features: %d\n\n", enemy.HP, enemy.Difficulty, len(enemy.AdversaryFeatures))

	// Create simple player party
	player1 := models.NewCombatant("p1", "Warrior", models.SidePlayer, 1, 14, 2, 8, 5)
	player1.DamageDice = "2d6+2"
	player1.ArmorSlots = 3

	player2 := models.NewCombatant("p2", "Rogue", models.SidePlayer, 1, 12, 3, 6, 5)
	player2.DamageDice = "2d6+1"
	player2.ArmorSlots = 2

	players := []*models.Combatant{player1, player2}

	enemyClone := models.CloneCombatant(enemy, enemy.ID+"_clone")
	enemies := []*models.Combatant{enemyClone}

	// Run simulation
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	dice := engine.NewStandardDiceRoller(rng)
	spotlight := engine.NewHybridSpotlightSelector(rng, 0.5)
	combatEngine := engine.NewCombatEngine(dice, spotlight)

	decisionMaker := strategy.NewUnifiedDecisionMaker(rng, strategy.DefaultDesperateConfig(), strategy.DefaultGreedyConfig())

	state := models.NewCombatState(players, enemies, 3, 2, 5)

	// Run single simulation
	fmt.Println("Running single combat...")
	enemyConfig := strategy.DefaultGreedyConfig()
	outcome := combatEngine.RunCombat(state, decisionMaker, 100, enemyConfig.ResourceUsage)

	// Display results
	fmt.Printf("\n=== Combat Result ===\n")
	if outcome.PlayerVictory {
		fmt.Println("Winner: Players")
	} else if outcome.EnemyVictory {
		fmt.Println("Winner: Enemies")
	} else {
		fmt.Println("Winner: Draw (max turns reached)")
	}

	fmt.Printf("Turns: %d\n", outcome.Turns)
	fmt.Printf("Final Hope: %d, Fear: %d\n\n", outcome.FinalHopePool, outcome.FinalFearPool)

	// Show feature usage in combat log
	fmt.Println("=== Feature Usage ===")
	featureCount := 0
	for _, action := range outcome.ActionHistory {
		if action.IsFeature {
			featureCount++
			fmt.Printf("Turn %d: %s used %s\n", action.Turn, action.Actor, action.FeatureName)
		}
	}

	if featureCount == 0 {
		fmt.Println("No features were activated during combat")
	} else {
		fmt.Printf("\nTotal feature activations: %d\n", featureCount)
	}
}
