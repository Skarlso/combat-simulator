package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/skarlso/daggerheart-combat-simulator/internal/engine"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func main() {
	fmt.Println("=== Armor Threshold Reduction Test ===")
	fmt.Println("Testing that armor reduces damage THRESHOLD, not damage amount")
	fmt.Println("Rule: Mark ONE armor slot to reduce severity (Severe→Major→Minor→None)")
	fmt.Println()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	dice := engine.NewStandardDiceRoller(rng)

	// Create a player with custom thresholds for predictable testing
	// Thresholds: <5=Minor, 5-9=Major, 10+=Severe
	player := &models.Combatant{
		ID:               "player",
		Name:             "Test Guardian",
		Side:             models.SidePlayer,
		HPMax:            10,
		HP:               10,
		ArmorSlots:       3,
		CustomThresholds: []int{5, 10}, // Minor<5, 5≤Major<10, 10≤Severe
		IsAdversary:      false,
	}

	fmt.Println("Player Setup:")
	fmt.Printf("  HP: %d/%d\n", player.HP, player.HPMax)
	fmt.Printf("  Armor Slots: %d\n", player.ArmorSlots)
	fmt.Printf("  Thresholds: Minor<5, 5≤Major<10, 10≤Severe\n")
	fmt.Println()

	// Test 1: Severe damage (10) with armor → Major damage (2 HP instead of 3 HP)
	fmt.Println("Test 1: Severe damage (10) WITH armor")
	fmt.Println("  Expected: Mark 1 armor, reduce Severe→Major, take 2 HP damage")
	initialHP := player.HP
	initialArmor := player.ArmorSlots
	result := player.TakeDamage(10, models.DamagePhysical, dice)
	fmt.Printf("  Result: Threshold=%s, HP damage=%d, Armor used=%d→%d\n",
		result.Threshold, result.HPDamage, initialArmor, player.ArmorSlots)
	if result.Threshold == models.ThresholdMajor && result.HPDamage == 2 && player.ArmorSlots == 2 {
		fmt.Printf("  ✓ PASS: %d HP → %d HP (lost %d HP)\n", initialHP, player.HP, initialHP-player.HP)
	} else {
		fmt.Printf("  ✗ FAIL: Expected Major/2HP/2armor, got %s/%dHP/%darmor\n",
			result.Threshold, result.HPDamage, player.ArmorSlots)
	}
	fmt.Println()

	// Test 2: Major damage (7) with armor → Minor damage (1 HP instead of 2 HP)
	fmt.Println("Test 2: Major damage (7) WITH armor")
	fmt.Println("  Expected: Mark 1 armor, reduce Major→Minor, take 1 HP damage")
	initialHP = player.HP
	initialArmor = player.ArmorSlots
	result = player.TakeDamage(7, models.DamagePhysical, dice)
	fmt.Printf("  Result: Threshold=%s, HP damage=%d, Armor used=%d→%d\n",
		result.Threshold, result.HPDamage, initialArmor, player.ArmorSlots)
	if result.Threshold == models.ThresholdMinor && result.HPDamage == 1 && player.ArmorSlots == 1 {
		fmt.Printf("  ✓ PASS: %d HP → %d HP (lost %d HP)\n", initialHP, player.HP, initialHP-player.HP)
	} else {
		fmt.Printf("  ✗ FAIL: Expected Minor/1HP/1armor, got %s/%dHP/%darmor\n",
			result.Threshold, result.HPDamage, player.ArmorSlots)
	}
	fmt.Println()

	// Test 3: Minor damage (3) with armor → No damage (0 HP)
	fmt.Println("Test 3: Minor damage (3) WITH armor")
	fmt.Println("  Expected: Mark 1 armor, reduce Minor→None, take 0 HP damage")
	initialHP = player.HP
	initialArmor = player.ArmorSlots
	result = player.TakeDamage(3, models.DamagePhysical, dice)
	fmt.Printf("  Result: Threshold=%s, HP damage=%d, Armor used=%d→%d\n",
		result.Threshold, result.HPDamage, initialArmor, player.ArmorSlots)
	if result.Threshold == models.ThresholdMinor && result.HPDamage == 0 && player.ArmorSlots == 0 {
		fmt.Printf("  ✓ PASS: %d HP → %d HP (lost %d HP)\n", initialHP, player.HP, initialHP-player.HP)
	} else {
		fmt.Printf("  ✗ FAIL: Expected Minor/0HP/0armor, got %s/%dHP/%darmor\n",
			result.Threshold, result.HPDamage, player.ArmorSlots)
	}
	fmt.Println()

	// Test 4: Major damage (7) WITHOUT armor → Major damage (2 HP)
	fmt.Println("Test 4: Major damage (7) WITHOUT armor")
	fmt.Println("  Expected: No armor to mark, take full Major damage (2 HP)")
	initialHP = player.HP
	result = player.TakeDamage(7, models.DamagePhysical, dice)
	fmt.Printf("  Result: Threshold=%s, HP damage=%d\n", result.Threshold, result.HPDamage)
	if result.Threshold == models.ThresholdMajor && result.HPDamage == 2 {
		fmt.Printf("  ✓ PASS: %d HP → %d HP (lost %d HP)\n", initialHP, player.HP, initialHP-player.HP)
	} else {
		fmt.Printf("  ✗ FAIL: Expected Major/2HP, got %s/%dHP\n",
			result.Threshold, result.HPDamage)
	}
	fmt.Println()

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Printf("Final HP: %d/%d\n", player.HP, player.HPMax)
	fmt.Printf("Armor remaining: %d\n", player.ArmorSlots)
	fmt.Println()
	fmt.Println("Expected damage breakdown:")
	fmt.Println("  Test 1: Severe(10) + armor → Major = 2 HP lost")
	fmt.Println("  Test 2: Major(7) + armor → Minor = 1 HP lost")
	fmt.Println("  Test 3: Minor(3) + armor → None = 0 HP lost")
	fmt.Println("  Test 4: Major(7) no armor → Major = 2 HP lost")
	fmt.Println("  Total expected: 2+1+0+2 = 5 HP lost")
	fmt.Printf("  Actual HP lost: %d\n", player.HPMax-player.HP)
	fmt.Println()

	if player.HP == 5 && player.ArmorSlots == 0 {
		fmt.Println("✓ All tests PASSED - Armor threshold reduction working correctly!")
	} else {
		fmt.Printf("✗ Tests FAILED - Expected 5 HP and 0 armor, got %d HP and %d armor\n",
			player.HP, player.ArmorSlots)
	}
}
