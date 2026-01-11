package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/skarlso/daggerheart-combat-simulator/internal/engine"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func main() {
	fmt.Println("=== Resistance Test ===")

	// Create a player
	player := models.NewCombatant("player-1", "Player", models.SidePlayer, 1, 11, 1, 5, 6)

	// Add a passive physical resistance ability
	resistAbility := models.DomainAbility{
		Name:        "Iron Skin",
		Description: "Physical resistance",
		Passive:     true,
		Effects: []models.FeatureEffect{
			{
				Category:   models.FeatureEffectDamageResistance,
				Magnitude:  50,
				DamageType: "physical",
			},
		},
	}
	player.AddDomainAbility(resistAbility)

	// Apply passive resistances
	fmt.Println("Before ApplyPassiveResistances:")
	fmt.Printf("  Resistances count: %d\n", len(player.Resistances))

	player.ApplyPassiveResistances()

	fmt.Println("\nAfter ApplyPassiveResistances:")
	fmt.Printf("  Resistances count: %d\n", len(player.Resistances))
	for i, resist := range player.Resistances {
		fmt.Printf("  [%d] Type: %s, IsImmune: %v\n", i, resist.Type, resist.IsImmune)
	}

	// Test damage application
	fmt.Println("\n=== Damage Test ===")

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	dice := engine.NewStandardDiceRoller(rng)

	// Test 1: Physical damage (should be resisted)
	fmt.Println("\nTest 1: 10 physical damage (should be reduced to 5)")
	result1 := player.TakeDamage(10, models.DamagePhysical, dice)
	fmt.Printf("  Raw: 10, Processed: %d, HP Damage: %d\n", result1.ProcessedDamage, result1.HPDamage)
	fmt.Printf("  Player HP: %d\n", player.HP)

	// Reset HP
	player.HP = player.HPMax

	// Test 2: Magical damage (should NOT be resisted)
	fmt.Println("\nTest 2: 10 magical damage (should stay 10)")
	result2 := player.TakeDamage(10, models.DamageMagic, dice)
	fmt.Printf("  Raw: 10, Processed: %d, HP Damage: %d\n", result2.ProcessedDamage, result2.HPDamage)
	fmt.Printf("  Player HP: %d\n", player.HP)

	fmt.Println("\n=== Test Complete ===")
}
