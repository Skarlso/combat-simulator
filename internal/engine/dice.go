package engine

import (
	"math/rand"

	"github.com/skarlso/combat-simulator/internal/models"
)

// DiceRoller handles all dice rolling operations
type DiceRoller interface {
	RollDuality(hopeModifier, fearModifier int) DualityRollResult
	RollD20Attack(modifier int) D20RollResult
	RollDice(diceCount, diceSize, flatBonus int) int
	RollDiceDetailed(diceCount, diceSize, flatBonus int) DamageRollResult
}

// DamageRollResult contains individual dice values
type DamageRollResult struct {
	DiceValues []int // Individual dice results
	FlatBonus  int   // Flat bonus added
	Total      int   // Sum of all values
}

// DualityRollResult contains outcome of a duality roll (players only)
type DualityRollResult struct {
	HopeDie    int
	FearDie    int
	HopeTotal  int
	FearTotal  int
	Result     models.DiceResult
	WithSpirit int // The actual value to use (highest die + modifiers)
	IsCritical bool
}

// D20RollResult contains outcome of a d20 roll (enemies)
type D20RollResult struct {
	D20Roll    int  // The d20 roll
	Modifier   int  // Attack modifier
	Total      int  // d20 + modifier
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
	hopeDie := dr.RollDice(1, 12, 0)
	fearDie := dr.RollDice(1, 12, 0)

	hopeTotal := hopeDie + hopeModifier
	fearTotal := fearDie + fearModifier

	result := DualityRollResult{
		HopeDie:    hopeDie,
		FearDie:    fearDie,
		HopeTotal:  hopeTotal,
		FearTotal:  fearTotal,
		WithSpirit: hopeTotal + fearTotal,
	}

	// Hope vs Fear is decided on the raw dice, not the modified totals
	switch {
	case hopeDie == fearDie:
		result.IsCritical = true
		result.Result = models.ResultCriticalHope
	case hopeDie > fearDie:
		result.Result = models.ResultHope
	default:
		result.Result = models.ResultFear
	}
	return result
}

// RollDice rolls dice and adds flat bonus
func (dr *StandardDiceRoller) RollDice(diceCount, diceSize, flatBonus int) int {
	total := flatBonus
	for range diceCount {
		total += dr.rng.Intn(diceSize) + 1
	}
	return total
}

// RollDiceDetailed rolls dice and returns individual values
func (dr *StandardDiceRoller) RollDiceDetailed(diceCount, diceSize, flatBonus int) DamageRollResult {
	result := DamageRollResult{DiceValues: make([]int, max(diceCount, 0)), FlatBonus: flatBonus, Total: flatBonus}
	for i := range result.DiceValues {
		result.DiceValues[i] = dr.rng.Intn(diceSize) + 1
		result.Total += result.DiceValues[i]
	}
	return result
}

// RollD20Attack rolls d20 + modifier for enemy attacks
func (dr *StandardDiceRoller) RollD20Attack(modifier int) D20RollResult {
	d20 := dr.RollDice(1, 20, 0)
	return D20RollResult{D20Roll: d20, Modifier: modifier, Total: d20 + modifier, IsCritical: d20 == 20}
}

// rollNotationDetailed rolls every term of a notation, keeping the individual dice for the log
func rollNotationDetailed(dice DiceRoller, notation *models.DiceNotation) DamageRollResult {
	var out DamageRollResult
	for _, term := range notation.Terms {
		switch {
		case term.Count == 0:
			out.FlatBonus += term.Size
		case term.Count > 0:
			r := dice.RollDiceDetailed(term.Count, term.Size, 0)
			out.DiceValues = append(out.DiceValues, r.DiceValues...)
			out.Total += r.Total
		}
	}
	out.Total += out.FlatBonus
	return out
}

// rollAdversaryD20 rolls the GM's d20. With advantage or disadvantage a second
// d20 is rolled and the higher or lower one is kept.
func rollAdversaryD20(dice DiceRoller, modifier int, hasAdv, hasDisadv bool) D20RollResult {
	roll := dice.RollD20Attack(modifier)
	if hasAdv == hasDisadv {
		return roll
	}
	other := dice.RollD20Attack(modifier)
	if (hasAdv && other.D20Roll > roll.D20Roll) || (hasDisadv && other.D20Roll < roll.D20Roll) {
		return other
	}
	return roll
}

// critRoller forces every duality roll to be a critical success
type critRoller struct {
	DiceRoller
}

func (c critRoller) RollDuality(hopeModifier, fearModifier int) DualityRollResult {
	roll := c.DiceRoller.RollDuality(hopeModifier, fearModifier)
	roll.FearDie = roll.HopeDie
	roll.FearTotal = roll.FearDie + fearModifier
	roll.WithSpirit = roll.HopeTotal + roll.FearTotal
	roll.Result = models.ResultCriticalHope
	roll.IsCritical = true
	return roll
}
