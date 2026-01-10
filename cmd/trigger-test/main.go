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
	fmt.Println("=== Domain Ability Trigger Test ===")
	fmt.Println("Testing reactive player ability with on_damage_taken trigger")
	fmt.Println()

	// Create player with reactive ability
	players := []*models.Combatant{
		{
			ID:          "player",
			Name:        "Guardian",
			Side:        models.SidePlayer,
			HP:          40,
			HPMax:       40,
			Evasion:     8, // Lower evasion so enemy can hit
			Proficiency: 5,
			ArmorSlots:  0, // No armor so damage goes through
			Stress:      0,
			StressMax:   5,
			DamageDice:  "1d6",
			DomainAbilities: []models.DomainAbility{
				{
					Name:        "Retribution",
					Description: "When you take damage, mark stress on your attacker",
					Passive:     true,
					Trigger: models.FeatureTriggerCondition{
						Type:   "on_damage_taken",
						Filter: "",
					},
					HopeCost:    0, // Reactive abilities should be free
					StressCost:  0,
					RefreshType: "unlimited",
					MaxUses:     0,
					UsesLeft:    0,
					DCType:      "none",
					FixedDC:     0,
					Effects: []models.FeatureEffect{
						{
							Category:   models.FeatureEffectStressInflict,
							Magnitude:  1,
							TargetType: models.FeatureTargetSingle,
							Duration:   "instant",
						},
					},
				},
			},
		},
	}

	// Create stronger enemy to ensure hits
	enemies := []*models.Combatant{
		{
			ID:         "enemy",
			Name:       "Tough Goblin",
			Side:       models.SideEnemy,
			HP:         40,
			HPMax:      40,
			Difficulty: 8, // Lower difficulty means easier to hit Guardian
			ATK:        8,  // Higher ATK for better chance to hit
			Stress:     0,
			StressMax:  5,
			DamageDice: "2d6+2", // Better damage
		},
	}

	state := models.NewCombatState(players, enemies, 5, 5, 5)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	dice := engine.NewStandardDiceRoller(rng)
	spotlight := engine.NewHybridSpotlightSelector(rng, 0.5)
	combatEngine := engine.NewCombatEngine(dice, spotlight)

	playerStrategy := strategy.NewAIDecisionMaker(rng, strategy.DefaultDesperateConfig())
	enemyStrategy := strategy.NewAIDecisionMaker(rng, strategy.DefaultConservativeConfig())

	decisionMaker := &UnifiedDecisionMaker{
		playerStrategy: playerStrategy,
		enemyStrategy:  enemyStrategy,
	}

	fmt.Println("Initial State:")
	fmt.Printf("  %s: %d/%d HP, %d/%d Stress\n", players[0].Name, players[0].HP, players[0].HPMax, players[0].Stress, players[0].StressMax)
	fmt.Printf("  %s: %d/%d HP, %d/%d Stress\n", enemies[0].Name, enemies[0].HP, enemies[0].HPMax, enemies[0].Stress, enemies[0].StressMax)
	fmt.Println()

	// Run combat
	fmt.Println("=== Running Combat ===")
	outcome := combatEngine.RunCombat(state, decisionMaker, 20, strategy.ResourceConservative)

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

	// Check if Retribution triggered (enemy should have gained stress)
	fmt.Println()
	fmt.Println("=== Trigger Analysis ===")
	fmt.Printf("Guardian final HP: %d/%d (took %d damage)\n", players[0].HP, players[0].HPMax, players[0].HPMax-players[0].HP)
	fmt.Printf("Enemy final stress: %d/%d\n", enemies[0].Stress, enemies[0].StressMax)

	// Count hits on Guardian
	guardianHitCount := 0
	for _, action := range outcome.ActionHistory {
		if action.Target == "Guardian" && action.HPDamage > 0 {
			guardianHitCount++
			fmt.Printf("  Turn %d: %s hit Guardian for %d HP damage\n", action.Turn, action.Actor, action.HPDamage)
		}
	}

	fmt.Printf("\nGuardian was hit %d times\n", guardianHitCount)
	fmt.Printf("Enemy gained %d stress (expected: %d)\n", enemies[0].Stress, guardianHitCount)

	// Retribution should inflict 1 stress per hit
	if guardianHitCount > 0 && enemies[0].Stress == guardianHitCount {
		fmt.Println("✓ Domain ability triggers are working!")
		fmt.Println("  Retribution inflicted 1 stress for each hit on Guardian")
	} else if guardianHitCount > 0 && enemies[0].Stress > 0 {
		fmt.Println("⚠ Partial success - triggers working but stress count doesn't match")
		fmt.Printf("  Expected %d stress, got %d\n", guardianHitCount, enemies[0].Stress)
	} else {
		fmt.Println("✗ Domain ability triggers did not activate")
		if guardianHitCount == 0 {
			fmt.Println("  (Note: Guardian was never hit, so no triggers expected)")
		}
	}
}
