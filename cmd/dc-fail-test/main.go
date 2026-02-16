package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/skarlso/daggerheart-combat-simulator/internal/engine"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)


func main() {
	fmt.Println("Testing DC Check Failure - Turn Consumption")
	fmt.Println("============================================")

	// Create player with ability that requires high DC
	player := &models.Combatant{
		ID:          "p1",
		Name:        "Wizard",
		Side:        models.SidePlayer,
		HP:          20,
		HPMax:       20,
		Stress:      0,
		StressMax:   10,
		Evasion:     12,
		Proficiency: 1, // Low proficiency to likely fail
		ArmorSlots:  2,
		DamageDice:  "1d6",
		DomainAbilities: []models.DomainAbility{
			{
				Name:        "Difficult Spell",
				Description: "Requires DC vs very high defense",
				HopeCost:    0, // Free to cast - AI will always try
				StressCost:  0,
				DCType:      "fixed",
				FixedDC:     20, // Very high DC - will fail on non-crits
				Effects: []models.FeatureEffect{
					{
						Category:    models.FeatureEffectDirectDamage,
						DiceFormula: "5d6",
						Magnitude:   10, // High damage to make AI want to use it
					},
				},
			},
		},
	}

	// High defense enemy
	enemy := &models.Combatant{
		ID:          "e1",
		Name:        "Tank",
		Side:        models.SideEnemy,
		HP:          30,
		HPMax:       30,
		Evasion:     20, // High defense
		Proficiency: 1,
		IsAdversary: true,
		Tier:        "Tier 1",
		DamageDice:  "1d6",
	}

	players := []*models.Combatant{player}
	enemies := []*models.Combatant{enemy}

	// Plenty of Hope to use ability multiple times
	state := models.NewCombatState(players, enemies, 10, 3, 5)

	fmt.Printf("Initial Hope Pool: %d\n", state.HopePool)
	fmt.Printf("Ability DC: 18 (Player Proficiency: %d)\n", player.Proficiency)
	fmt.Println()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	dice := engine.NewStandardDiceRoller(rng)
	spotlight := engine.NewHybridSpotlightSelector(rng, 0.5)
	combatEngine := engine.NewCombatEngine(dice, spotlight)

	// Greedy strategy - will always try to use abilities
	decisionMaker := strategy.NewUnifiedDecisionMaker(rng, strategy.DefaultGreedyConfig(), strategy.DefaultGreedyConfig())

	// Run limited turns
	outcome := combatEngine.RunCombat(state, decisionMaker, 15, strategy.ResourceGreedy)

	fmt.Println("\n=== Combat Result ===")
	fmt.Printf("Turns: %d\n", outcome.Turns)
	fmt.Printf("Final Hope: %d, Fear: %d\n", outcome.FinalHopePool, outcome.FinalFearPool)

	fmt.Println("\n=== Action Log (Wizard only) ===")
	dcFailures := 0
	doubleActions := 0
	prevTurn := -1

	for i, action := range outcome.ActionHistory {
		if action.Actor == "Wizard" {
			// Check for double actions in same turn (BUG indicator)
			if action.Turn == prevTurn {
				doubleActions++
				fmt.Printf("⚠ Turn %d: DOUBLE ACTION DETECTED!\n", action.Turn)
			}
			prevTurn = action.Turn

			if action.ActionType == models.ActionUseAbility {
				if action.Result != "Success" {
					dcFailures++
					fmt.Printf("Turn %d: Wizard used %s -> FAILED DC CHECK (Hope: %d)\n",
						i+1, action.FeatureName, action.HopePoolAfter)
				} else {
					fmt.Printf("Turn %d: Wizard used %s -> SUCCESS (Hope: %d)\n",
						i+1, action.FeatureName, action.HopePoolAfter)
				}
			} else if action.ActionType == models.ActionAttack {
				fmt.Printf("Turn %d: Wizard -> Basic Attack (Hope: %d)\n",
					i+1, action.HopePoolAfter)
			}
		}
	}

	fmt.Printf("\n=== Test Results ===\n")
	fmt.Printf("DC check failures: %d\n", dcFailures)
	fmt.Printf("Double actions (BUG): %d\n", doubleActions)
	fmt.Printf("Hope spent: %d (started with 10, ended with %d)\n", 10-outcome.FinalHopePool, outcome.FinalHopePool)

	if doubleActions > 0 {
		fmt.Println("\n✗ FAILED - Ability failure should consume turn, no fallback attack")
	} else if dcFailures > 0 {
		fmt.Println("\n✓ PASSED - DC failures consumed turn without fallback")
	} else {
		fmt.Println("\n⚠ INCONCLUSIVE - No DC failures occurred (unlucky rolls)")
	}
}
