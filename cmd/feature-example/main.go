package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/skarlso/daggerheart-combat-simulator/internal/engine"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func main() {
	fmt.Println("=== Daggerheart Feature System Example ===\n")

	// Initialize RNG
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Create combat state
	state := &models.CombatState{
		HopePool: 3,
		FearPool: 2,
	}

	// Create player
	player := models.NewCombatant("p1", "Warrior", models.SidePlayer, 3, 12, 5, 15, 7)

	// Add a domain ability: "Rune Strike" - costs 2 Hope, deals 2d6+3 damage
	runeStrike := models.DomainAbility{
		Name:        "Rune Strike",
		Description: "Channel runic power into a devastating strike",
		HopeCost:    2,
		StressCost:  0,
		RefreshType: "rest",
		MaxUses:     3,
		UsesLeft:    3,
		Effects: []models.FeatureEffect{
			{
				Category:    models.FeatureEffectDamage,
				DiceFormula: "2d6+3",
				DamageType:  "magic",
				TargetType:  models.FeatureTargetSingle,
			},
		},
	}
	player.AddDomainAbility(runeStrike)

	// Create adversary
	goblin := models.NewAdversary("e1", "Goblin Shaman", "Tier 1", "Standard", 11, 2, 8, 5, []int{4, 8})

	// Add adversary feature: "Venomous Stinger" - passive, applies poison on hit
	venomousStinger := models.AdversaryFeature{
		Name:        "Venomous Stinger",
		Description: "Attacks apply poison to the target",
		Passive:     true,
		Trigger: models.FeatureTriggerCondition{
			Type: "on_attack",
		},
		FearCost:       0,
		SpotlightLimit: 0,
		Effects: []models.FeatureEffect{
			{
				Category:   models.FeatureEffectApplyPoison,
				Duration:   "until_cleared",
				TargetType: models.FeatureTargetSingle,
			},
			{
				Category:   models.FeatureEffectStressInflict,
				Magnitude:  2,
				TargetType: models.FeatureTargetSingle,
			},
		},
	}
	goblin.AddAdversaryFeature(venomousStinger)

	// Add adversary feature: "Dark Ritual" - costs 2 Fear, deals direct damage
	darkRitual := models.AdversaryFeature{
		Name:        "Dark Ritual",
		Description: "Sacrifices life force to deal direct damage",
		Passive:     false,
		Trigger: models.FeatureTriggerCondition{
			Type: "on_attack",
		},
		FearCost:       2,
		SpotlightLimit: 1,
		Effects: []models.FeatureEffect{
			{
				Category:    models.FeatureEffectDirectDamage,
				DiceFormula: "1d8",
				TargetType:  models.FeatureTargetSingle,
			},
		},
	}
	goblin.AddAdversaryFeature(darkRitual)

	// Create feature resolver
	resolver := engine.NewFeatureResolver(rng)

	fmt.Println("=== Initial State ===")
	fmt.Printf("Player: %s (HP: %d/%d, Stress: %d/%d)\n", player.Name, player.HP, player.HPMax, player.Stress, player.StressMax)
	fmt.Printf("Enemy: %s (HP: %d/%d, Stress: %d/%d)\n", goblin.Name, goblin.HP, goblin.HPMax, goblin.Stress, goblin.StressMax)
	fmt.Printf("Hope Pool: %d, Fear Pool: %d\n\n", state.HopePool, state.FearPool)

	// Test player domain ability
	fmt.Println("=== Player uses Rune Strike ===")
	ability := &player.DomainAbilities[0]
	if resolver.CanActivateDomainAbility(*ability, player, state) {
		fmt.Printf("✓ Can activate %s (costs %d Hope)\n", ability.Name, ability.HopeCost)
		err, _ := resolver.ActivateDomainAbility(ability, player, goblin, state)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("✓ %s activated successfully!\n", ability.Name)
			fmt.Printf("  Enemy HP: %d/%d\n", goblin.HP, goblin.HPMax)
			fmt.Printf("  Hope Pool: %d (spent %d)\n", state.HopePool, ability.HopeCost)
			fmt.Printf("  Uses left: %d/%d\n", ability.UsesLeft, ability.MaxUses)
		}
	} else {
		fmt.Printf("✗ Cannot activate %s (insufficient resources)\n", ability.Name)
	}
	fmt.Println()

	// Test adversary feature
	fmt.Println("=== Enemy uses Venomous Stinger ===")
	feature := goblin.AdversaryFeatures[0]
	if resolver.CanActivateAdversaryFeature(feature, goblin, state) {
		fmt.Printf("✓ Can activate %s\n", feature.Name)
		err, _ := resolver.ActivateAdversaryFeature(feature, goblin, player, state)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("✓ %s activated successfully!\n", feature.Name)
			fmt.Printf("  Player Stress: %d/%d\n", player.Stress, player.StressMax)
			fmt.Printf("  Player has conditions: %d\n", len(player.Conditions))
			if len(player.Conditions) > 0 {
				for _, cond := range player.Conditions {
					fmt.Printf("    - %s (duration: %s)\n", cond.Type, cond.Duration)
				}
			}
		}
	} else {
		fmt.Printf("✗ Cannot activate %s\n", feature.Name)
	}
	fmt.Println()

	// Test adversary feature with Fear cost
	fmt.Println("=== Enemy uses Dark Ritual ===")
	feature2 := goblin.AdversaryFeatures[1]
	if resolver.CanActivateAdversaryFeature(feature2, goblin, state) {
		fmt.Printf("✓ Can activate %s (costs %d Fear)\n", feature2.Name, feature2.FearCost)
		err, _ := resolver.ActivateAdversaryFeature(feature2, goblin, player, state)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("✓ %s activated successfully!\n", feature2.Name)
			fmt.Printf("  Player HP: %d/%d\n", player.HP, player.HPMax)
			fmt.Printf("  Fear Pool: %d (spent %d)\n", state.FearPool, feature2.FearCost)
		}
	} else {
		fmt.Printf("✗ Cannot activate %s (insufficient Fear)\n", feature2.Name)
	}
	fmt.Println()

	fmt.Println("=== Final State ===")
	fmt.Printf("Player: %s (HP: %d/%d, Stress: %d/%d)\n", player.Name, player.HP, player.HPMax, player.Stress, player.StressMax)
	fmt.Printf("Enemy: %s (HP: %d/%d, Stress: %d/%d)\n", goblin.Name, goblin.HP, goblin.HPMax, goblin.Stress, goblin.StressMax)
	fmt.Printf("Hope Pool: %d, Fear Pool: %d\n", state.HopePool, state.FearPool)
}
