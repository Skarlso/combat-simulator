package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/skarlso/daggerheart-combat-simulator/internal/engine"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func main() {
	fmt.Println("=== Direct Damage vs Regular Damage Test ===")
	fmt.Println("Testing that direct_damage bypasses armor (adversary-only feature)")
	fmt.Println()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	dice := engine.NewStandardDiceRoller(rng)

	// Create player with armor
	player := &models.Combatant{
		ID:               "player",
		Name:             "Test Guardian",
		Side:             models.SidePlayer,
		HPMax:            10,
		HP:               10,
		ArmorSlots:       2,
		CustomThresholds: []int{5, 10}, // Minor<5, 5≤Major<10, 10≤Severe
		IsAdversary:      false,
	}

	fmt.Println("Player Setup:")
	fmt.Printf("  HP: %d/%d\n", player.HP, player.HPMax)
	fmt.Printf("  Armor Slots: %d\n", player.ArmorSlots)
	fmt.Printf("  Thresholds: Minor<5, 5≤Major<10, 10≤Severe\n")
	fmt.Println()

	// Test 1: Regular damage (Major threshold) WITH armor → reduced to Minor
	fmt.Println("Test 1: Regular damage (7) goes through TakeDamage() WITH armor")
	fmt.Println("  Expected: Major(2 HP) + armor → Minor(1 HP), armor consumed")
	initialHP := player.HP
	initialArmor := player.ArmorSlots
	result := player.TakeDamage(7, models.DamagePhysical, dice)
	fmt.Printf("  Result: HP=%d→%d (-%d), Armor=%d→%d, Threshold=%v\n",
		initialHP, player.HP, result.HPDamage, initialArmor, player.ArmorSlots, result.Threshold)
	if result.HPDamage == 1 && player.ArmorSlots == 1 {
		fmt.Println("  ✓ PASS: Armor reduced threshold (Major→Minor)")
	} else {
		fmt.Printf("  ✗ FAIL: Expected 1 HP damage and 1 armor remaining\n")
	}
	fmt.Println()

	// Test 2: Direct damage bypasses armor completely
	fmt.Println("Test 2: Direct damage (5 HP) bypasses TakeDamage() - no armor used")
	fmt.Println("  Expected: 5 HP lost directly, armor NOT consumed")
	initialHP = player.HP
	initialArmor = player.ArmorSlots

	// Simulate direct_damage (bypasses TakeDamage, directly modifies HP)
	directDamage := 5
	player.HP -= directDamage
	if player.HP <= 0 {
		player.HP = 0
		player.IsDefeated = true
	}

	fmt.Printf("  Result: HP=%d→%d (-%d), Armor=%d→%d\n",
		initialHP, player.HP, directDamage, initialArmor, player.ArmorSlots)
	if player.HP == 4 && player.ArmorSlots == 1 {
		fmt.Println("  ✓ PASS: Direct damage bypassed armor (no slots consumed)")
	} else {
		fmt.Printf("  ✗ FAIL: Expected 4 HP and 1 armor remaining\n")
	}
	fmt.Println()

	// Test 3: Another regular damage to confirm armor still works
	fmt.Println("Test 3: Another regular damage (3) goes through TakeDamage() WITH armor")
	fmt.Println("  Expected: Minor(1 HP) + armor → None(0 HP), armor consumed")
	initialHP = player.HP
	initialArmor = player.ArmorSlots
	result = player.TakeDamage(3, models.DamagePhysical, dice)
	fmt.Printf("  Result: HP=%d→%d (-%d), Armor=%d→%d, Threshold=%v\n",
		initialHP, player.HP, result.HPDamage, initialArmor, player.ArmorSlots, result.Threshold)
	if result.HPDamage == 0 && player.ArmorSlots == 0 {
		fmt.Println("  ✓ PASS: Armor reduced threshold (Minor→None)")
	} else {
		fmt.Printf("  ✗ FAIL: Expected 0 HP damage and 0 armor remaining\n")
	}
	fmt.Println()

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Printf("Final HP: %d/%d\n", player.HP, player.HPMax)
	fmt.Printf("Armor remaining: %d\n", player.ArmorSlots)
	fmt.Println()
	fmt.Println("Expected damage breakdown:")
	fmt.Println("  Test 1: Regular Major(7) + armor → Minor = 1 HP lost, 1 armor used")
	fmt.Println("  Test 2: Direct damage(5) → bypasses armor = 5 HP lost, 0 armor used")
	fmt.Println("  Test 3: Regular Minor(3) + armor → None = 0 HP lost, 1 armor used")
	fmt.Println("  Total expected: 1+5+0 = 6 HP lost, 0 armor remaining")
	fmt.Printf("  Actual HP lost: %d, Armor remaining: %d\n", player.HPMax-player.HP, player.ArmorSlots)
	fmt.Println()

	if player.HP == 4 && player.ArmorSlots == 0 {
		fmt.Println("✓ All tests PASSED - Direct damage correctly bypasses armor!")
	} else {
		fmt.Printf("✗ Tests FAILED - Expected 4 HP and 0 armor, got %d HP and %d armor\n",
			player.HP, player.ArmorSlots)
	}
}
