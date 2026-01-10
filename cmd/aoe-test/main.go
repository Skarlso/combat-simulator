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
	fmt.Println("=== AOE Damage Test ===")
	fmt.Println("Testing area-of-effect damage that hits multiple targets")
	fmt.Println()

	// Create player with AOE ability
	players := []*models.Combatant{
		{
			ID:          "wizard",
			Name:        "Wizard",
			Side:        models.SidePlayer,
			HP:          30,
			HPMax:       30,
			Evasion:     10,
			Proficiency: 5,
			ArmorSlots:  0,
			Stress:      0,
			StressMax:   5,
			DamageDice:  "1d4",
			DomainAbilities: []models.DomainAbility{
				{
					Name:        "Fireball",
					Description: "AOE magic damage to all enemies",
					Passive:     false,
					HopeCost:    2,
					StressCost:  0,
					RefreshType: "scene",
					MaxUses:     3,
					UsesLeft:    3,
					DCType:      "none",
					FixedDC:     0,
					Effects: []models.FeatureEffect{
						{
							Category:    models.FeatureEffectAreaDamage,
							Magnitude:   2,
							DiceFormula: "3d6",
							DamageType:  "magic",
							TargetType:  models.FeatureTargetAOE,
							Duration:    "instant",
						},
					},
				},
			},
		},
	}

	// Create multiple weak enemies
	enemies := []*models.Combatant{
		{
			ID:         "goblin1",
			Name:       "Goblin 1",
			Side:       models.SideEnemy,
			HP:         15,
			HPMax:      15,
			Difficulty: 10,
			ATK:        2,
			Stress:     0,
			StressMax:  3,
			DamageDice: "1d4",
		},
		{
			ID:         "goblin2",
			Name:       "Goblin 2",
			Side:       models.SideEnemy,
			HP:         15,
			HPMax:      15,
			Difficulty: 10,
			ATK:        2,
			Stress:     0,
			StressMax:  3,
			DamageDice: "1d4",
		},
		{
			ID:         "goblin3",
			Name:       "Goblin 3",
			Side:       models.SideEnemy,
			HP:         15,
			HPMax:      15,
			Difficulty: 10,
			ATK:        2,
			Stress:     0,
			StressMax:  3,
			DamageDice: "1d4",
		},
	}

	state := models.NewCombatState(players, enemies, 10, 5, 5)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	dice := engine.NewStandardDiceRoller(rng)
	spotlight := engine.NewHybridSpotlightSelector(rng, 0.5)
	combatEngine := engine.NewCombatEngine(dice, spotlight)

	// Greedy strategy prefers abilities over basic attacks
	playerStrategy := strategy.NewAIDecisionMaker(rng, strategy.DefaultGreedyConfig())
	enemyStrategy := strategy.NewAIDecisionMaker(rng, strategy.DefaultConservativeConfig())

	decisionMaker := &UnifiedDecisionMaker{
		playerStrategy: playerStrategy,
		enemyStrategy:  enemyStrategy,
	}

	fmt.Println("Initial State:")
	fmt.Printf("  %s: %d HP\n", players[0].Name, players[0].HP)
	for _, enemy := range enemies {
		fmt.Printf("  %s: %d HP\n", enemy.Name, enemy.HP)
	}
	fmt.Println()

	// Run combat (short rounds to see Fireball effect)
	fmt.Println("=== Running Combat ===")
	outcome := combatEngine.RunCombat(state, decisionMaker, 10, strategy.ResourceGreedy)

	// Display results
	fmt.Println()
	fmt.Println("=== Combat Result ===")
	if outcome.PlayerVictory {
		fmt.Println("Winner: Players")
	} else if outcome.EnemyVictory {
		fmt.Println("Winner: Enemies")
	} else {
		fmt.Println("Winner: Draw")
	}

	fmt.Printf("Turns: %d\n", outcome.Turns)

	// Analyze Fireball usage
	fmt.Println()
	fmt.Println("=== AOE Analysis ===")

	fireballUses := 0
	for _, action := range outcome.ActionHistory {
		if action.IsFeature && action.FeatureName == "Fireball" {
			fireballUses++
			fmt.Printf("Turn %d: %s used Fireball (rolled %d damage)\n",
				action.Turn, action.Actor, action.DamageRoll)
		}
	}

	fmt.Printf("\nFireball used %d times\n", fireballUses)

	// Check final HP of all enemies
	fmt.Println("\nFinal Enemy HP:")
	allDamaged := true
	for _, enemy := range enemies {
		damageTaken := enemy.HPMax - enemy.HP
		fmt.Printf("  %s: %d/%d HP (took %d damage)\n", enemy.Name, enemy.HP, enemy.HPMax, damageTaken)
		if enemy.HPMax > 0 && damageTaken == 0 {
			allDamaged = false
		}
	}

	// Verdict
	fmt.Println()
	if fireballUses > 0 {
		if allDamaged {
			fmt.Println("✓ AOE damage is working!")
			fmt.Println("  Fireball hit multiple targets correctly")
		} else {
			fmt.Println("⚠ AOE used but not all targets damaged")
			fmt.Println("  Some enemies may have been killed before Fireball")
		}
	} else {
		fmt.Println("⚠ Fireball was not used in combat")
		fmt.Println("  AI may have preferred basic attacks or lacked Hope")
	}
}
