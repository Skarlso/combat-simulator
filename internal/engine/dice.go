package engine

import (
	"math/rand"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// DiceRoller handles all dice rolling operations
type DiceRoller interface {
	RollDuality(hopeModifier, fearModifier int) DualityRollResult
	RollD20Attack(modifier int) D20RollResult
	RollDamage(diceCount, diceSize, flatBonus int) int
	RollDamageDetailed(diceCount, diceSize, flatBonus int) DamageRollResult
	RollD6() int
	RollD12() int
	RollD20() int
	RollDice(diceCount, diceSize int) int
}

// DamageRollResult contains individual dice values
type DamageRollResult struct {
	DiceValues []int // Individual dice results
	FlatBonus  int   // Flat bonus added
	Total      int   // Sum of all values
}

// DualityRollResult contains outcome of a duality roll (players only)
type DualityRollResult struct {
	HopeDie      int
	FearDie      int
	HopeTotal    int
	FearTotal    int
	Result       models.DiceResult
	WithSpirit   int // The actual value to use (highest die + modifiers)
	IsCritical   bool
}

// D20RollResult contains outcome of a d20 roll (enemies)
type D20RollResult struct {
	D20Roll   int // The d20 roll
	Modifier  int // Attack modifier
	Total     int // d20 + modifier
	IsCritical bool // Natural 20
}

// StandardDiceRoller is the default implementation
type StandardDiceRoller struct {
	rng *rand.Rand
}

// NewStandardDiceRoller creates a dice roller with given random source
func NewStandardDiceRoller(rng *rand.Rand) *StandardDiceRoller {
	return &StandardDiceRoller{
		rng: rng,
	}
}

// RollDuality performs the core d12 Hope vs d12 Fear roll
func (dr *StandardDiceRoller) RollDuality(hopeModifier, fearModifier int) DualityRollResult {
	hopeDie := dr.RollD12()
	fearDie := dr.RollD12()

	hopeTotal := hopeDie + hopeModifier
	fearTotal := fearDie + fearModifier

	result := DualityRollResult{
		HopeDie:   hopeDie,
		FearDie:   fearDie,
		HopeTotal: hopeTotal,
		FearTotal: fearTotal,
	}

	// Check for critical success (matching dice)
	if hopeDie == fearDie {
		result.IsCritical = true
		// Critical success counts as "with Hope"
		result.Result = models.ResultCriticalHope
		result.WithSpirit = hopeTotal + fearTotal
		return result
	}

	// Sum both dice for total
	result.WithSpirit = hopeTotal + fearTotal

	// Determine winner (Hope vs Fear)
	if hopeTotal >= fearTotal {
		result.Result = models.ResultHope
	} else {
		result.Result = models.ResultFear
	}

	return result
}

// RollDamage rolls damage dice and adds flat bonus
func (dr *StandardDiceRoller) RollDamage(diceCount, diceSize, flatBonus int) int {
	total := flatBonus
	for i := 0; i < diceCount; i++ {
		total += dr.rng.Intn(diceSize) + 1
	}
	return total
}

// RollDamageDetailed rolls damage dice and returns individual values
func (dr *StandardDiceRoller) RollDamageDetailed(diceCount, diceSize, flatBonus int) DamageRollResult {
	diceValues := make([]int, diceCount)
	total := flatBonus

	for i := 0; i < diceCount; i++ {
		roll := dr.rng.Intn(diceSize) + 1
		diceValues[i] = roll
		total += roll
	}

	return DamageRollResult{
		DiceValues: diceValues,
		FlatBonus:  flatBonus,
		Total:      total,
	}
}

// RollD20Attack rolls d20 + modifier for enemy attacks
func (dr *StandardDiceRoller) RollD20Attack(modifier int) D20RollResult {
	d20 := dr.RollD20()
	total := d20 + modifier

	return D20RollResult{
		D20Roll:    d20,
		Modifier:   modifier,
		Total:      total,
		IsCritical: d20 == 20,
	}
}

// RollD6 rolls a single d6
func (dr *StandardDiceRoller) RollD6() int {
	return dr.rng.Intn(6) + 1
}

// RollD12 rolls a single d12
func (dr *StandardDiceRoller) RollD12() int {
	return dr.rng.Intn(12) + 1
}

// RollD20 rolls a single d20
func (dr *StandardDiceRoller) RollD20() int {
	return dr.rng.Intn(20) + 1
}

// RollDice rolls multiple dice of given size
func (dr *StandardDiceRoller) RollDice(diceCount, diceSize int) int {
	return dr.RollDamage(diceCount, diceSize, 0)
}
