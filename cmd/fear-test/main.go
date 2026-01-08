package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/skarlso/daggerheart-combat-simulator/internal/engine"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)

type UnifiedDecisionMaker struct {
	playerStrategy *strategy.AIDecisionMaker
	enemyStrategy  *strategy.AIDecisionMaker
}

func (udm *UnifiedDecisionMaker) ChooseAction(state *models.CombatState, actor *models.Combatant) *strategy.ActionDecision {
	if actor.Side == models.SidePlayer {
		return udm.playerStrategy.ChooseAction(state, actor)
	}
	return udm.enemyStrategy.ChooseAction(state, actor)
}

func main() {
	fmt.Println("Testing Aggressive Fear Usage Mechanic")
	fmt.Println("=======================================")

	// Create strong players and weak enemy to force multiple player turns
	players := []*models.Combatant{
		{
			ID:          "p1",
			Name:        "Strong Fighter",
			Side:        models.SidePlayer,
			HP:          50,
			HPMax:       50,
			Stress:      0,
			StressMax:   10,
			Evasion:     12,
			Proficiency: 5,
			ArmorSlots:  2,
			DamageDice:  "1d6",
		},
		{
			ID:          "p2",
			Name:        "Tank",
			Side:        models.SidePlayer,
			HP:          50,
			HPMax:       50,
			Stress:      0,
			StressMax:   10,
			Evasion:     14,
			Proficiency: 5,
			ArmorSlots:  3,
			DamageDice:  "1d6",
		},
	}

	// Weak enemy with high defense (to cause player misses and spotlight shifts)
	enemies := []*models.Combatant{
		{
			ID:          "e1",
			Name:        "Durable Enemy",
			Side:        models.SideEnemy,
			HP:          30,
			HPMax:       30,
			Evasion:     18, // High evasion to cause misses
			Proficiency: 1,  // Low proficiency
			IsAdversary: true,
			Tier:        "Tier 1",
			DamageDice:  "1d4",
		},
	}

	// Initialize with decent Fear pool
	state := models.NewCombatState(players, enemies, 5, 6, 5)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	dice := engine.NewStandardDiceRoller(rng)
	spotlight := engine.NewHybridSpotlightSelector(rng, 0.5)
	combatEngine := engine.NewCombatEngine(dice, spotlight)

	playerStrategy := strategy.NewAIDecisionMaker(rng, strategy.DefaultDesperateConfig())
	enemyStrategy := strategy.NewAIDecisionMaker(rng, strategy.DefaultGreedyConfig()) // Aggressive

	decisionMaker := &UnifiedDecisionMaker{
		playerStrategy: playerStrategy,
		enemyStrategy:  enemyStrategy,
	}

	// Run combat with aggressive Fear usage
	fmt.Println("\n=== Running Combat ===")
	fmt.Println("Enemy using GREEDY (Aggressive) Fear strategy")
	fmt.Println("Initial Fear Pool:", state.FearPool)
	fmt.Println()

	outcome := combatEngine.RunCombat(state, decisionMaker, 50, strategy.ResourceGreedy)

	// Display results
	fmt.Printf("\n=== Combat Result ===\n")
	if outcome.PlayerVictory {
		fmt.Println("Winner: Players")
	} else if outcome.EnemyVictory {
		fmt.Println("Winner: Enemies")
	} else {
		fmt.Println("Winner: Draw")
	}

	fmt.Printf("Turns: %d\n", outcome.Turns)
	fmt.Printf("Final Hope: %d, Fear: %d\n", outcome.FinalHopePool, outcome.FinalFearPool)

	// Check action history for spotlight grabs
	fmt.Println("\n=== Spotlight Grabs via Fear ===")
	fearGrabs := 0
	for i, action := range outcome.ActionHistory {
		if action.Actor == "GM" {
			fearGrabs++
			fmt.Printf("Turn %d: %s %s\n", i+1, action.Actor, action.Result)
		}
	}

	if fearGrabs == 0 {
		fmt.Println("No Fear-based spotlight grabs occurred (players may not have had 3+ consecutive turns)")
	} else {
		fmt.Printf("\nTotal Fear grabs: %d\n", fearGrabs)
	}

	// Show action log
	fmt.Println("\n=== Action Log (first 30 turns) ===")
	for i, action := range outcome.ActionHistory {
		if i >= 30 {
			break
		}
		if action.Actor == "" {
			fmt.Printf("Turn %d: %s (Fear: %d)\n", i+1, action.Result, action.FearPoolAfter)
		} else {
			fmt.Printf("Turn %d: %s -> %s (%s) | Fear: %d\n",
				i+1, action.Actor, action.Target, action.Result, action.FearPoolAfter)
		}
	}
}
