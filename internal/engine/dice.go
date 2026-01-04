package engine

import (
	"math/rand"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// DiceRoller handles all dice rolling operations
type DiceRoller interface {
	RollDuality(hopeModifier, fearModifier int) DualityRollResult
	RollDamage(diceCount, diceSize, flatBonus int) int
	RollD12() int
	RollDice(diceCount, diceSize int) int
}

// DualityRollResult contains outcome of a duality roll
type DualityRollResult struct {
	HopeDie      int
	FearDie      int
	HopeTotal    int
	FearTotal    int
	Result       models.DiceResult
	WithSpirit   int // The actual value to use (highest die + modifiers)
	IsCritical   bool
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

	// Determine winner with Spirit (use highest die)
	if hopeTotal >= fearTotal {
		result.Result = models.ResultHope
		result.WithSpirit = hopeTotal
		if hopeDie == 12 {
			result.Result = models.ResultCriticalHope
			result.IsCritical = true
		}
	} else {
		result.Result = models.ResultFear
		result.WithSpirit = fearTotal
		if fearDie == 12 {
			result.Result = models.ResultCriticalFear
			result.IsCritical = true
		}
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

// RollD12 rolls a single d12
func (dr *StandardDiceRoller) RollD12() int {
	return dr.rng.Intn(12) + 1
}

// RollDice rolls multiple dice of given size
func (dr *StandardDiceRoller) RollDice(diceCount, diceSize int) int {
	return dr.RollDamage(diceCount, diceSize, 0)
}
