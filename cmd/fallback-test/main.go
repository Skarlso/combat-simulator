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
	fmt.Println("Testing Fallback to Basic Attack")
	fmt.Println("=================================")

	// Create player with expensive ability
	player := &models.Combatant{
		ID:          "p1",
		Name:        "Fighter",
		Side:        models.SidePlayer,
		HP:          20,
		HPMax:       20,
		Stress:      0,
		StressMax:   10,
		Evasion:     12,
		Proficiency: 3,
		ArmorSlots:  2,
		DamageDice:  "1d6",
		DomainAbilities: []models.DomainAbility{
			{
				Name:        "Expensive Strike",
				Description: "Costs 5 Hope",
				HopeCost:    5, // Expensive!
				StressCost:  0,
				DCType:      "none",
				Effects: []models.FeatureEffect{
					{
						Category:    models.FeatureEffectDirectDamage,
						DiceFormula: "3d6",
						Magnitude:   5,
					},
				},
			},
		},
	}

	enemy := &models.Combatant{
		ID:          "e1",
		Name:        "Goblin",
		Side:        models.SideEnemy,
		HP:          15,
		HPMax:       15,
		Evasion:     12,
		Proficiency: 1,
		IsAdversary: true,
		Tier:        "Tier 1",
		DamageDice:  "1d6",
	}

	players := []*models.Combatant{player}
	enemies := []*models.Combatant{enemy}

	// Start with LOW Hope pool so ability can't be afforded
	state := models.NewCombatState(players, enemies, 2, 3, 5)

	fmt.Printf("Initial Hope Pool: %d (Ability needs: 5)\n", state.HopePool)
	fmt.Println()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	dice := engine.NewStandardDiceRoller(rng)
	spotlight := engine.NewHybridSpotlightSelector(rng, 0.5)
	combatEngine := engine.NewCombatEngine(dice, spotlight)

	// Greedy strategy - will try to use abilities
	playerStrategy := strategy.NewAIDecisionMaker(rng, strategy.DefaultGreedyConfig())
	enemyStrategy := strategy.NewAIDecisionMaker(rng, strategy.DefaultGreedyConfig())

	decisionMaker := &UnifiedDecisionMaker{
		playerStrategy: playerStrategy,
		enemyStrategy:  enemyStrategy,
	}

	// Run a few turns
	outcome := combatEngine.RunCombat(state, decisionMaker, 10, strategy.ResourceGreedy)

	fmt.Println("\n=== Combat Result ===")
	fmt.Printf("Turns: %d\n", outcome.Turns)
	fmt.Printf("Final Hope: %d, Fear: %d\n", outcome.FinalHopePool, outcome.FinalFearPool)

	fmt.Println("\n=== Action Log ===")
	failedAbilityCount := 0

	for i, action := range outcome.ActionHistory {
		if action.Actor == "Fighter" {
			if action.ActionType == models.ActionUseAbility {
				if action.Result != "Success" {
					failedAbilityCount++
					fmt.Printf("Turn %d: %s tried %s -> %s (FAILED)\n",
						i+1, action.Actor, action.FeatureName, action.Result)
				} else {
					fmt.Printf("Turn %d: %s used %s -> SUCCESS\n",
						i+1, action.Actor, action.FeatureName)
				}
			} else if action.ActionType == models.ActionAttack {
				fmt.Printf("Turn %d: %s -> %s (Basic Attack: %s) Hope: %d\n",
					i+1, action.Actor, action.Target, action.Result, action.HopePoolAfter)
			}
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Failed abilities: %d\n", failedAbilityCount)
	fmt.Printf("Total turns: %d\n", outcome.Turns)

	if failedAbilityCount > 0 {
		fmt.Println("\n✓ Fallback logic working - abilities failed but combat continued")
	} else {
		fmt.Println("\n⚠ No ability failures detected - test may need adjustment")
	}
}
