package engine

import (
	"math/rand"
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func newSeededRoller(seed int64) *StandardDiceRoller {
	return NewStandardDiceRoller(rand.New(rand.NewSource(seed)))
}

func TestRollDiceRanges(t *testing.T) {
	roller := newSeededRoller(42)
	tests := []struct {
		name string
		size int
	}{
		{"d6", 6},
		{"d12", 12},
		{"d20", 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 100; i++ {
				v := roller.RollDice(1, tt.size, 0)
				if v < 1 || v > tt.size {
					t.Errorf("RollDice(1, %d) = %d, want 1-%d", tt.size, v, tt.size)
				}
			}
		})
	}
}

func TestRollDiceWithBonus(t *testing.T) {
	roller := newSeededRoller(42)
	result := roller.RollDice(2, 6, 3)
	if result < 5 || result > 15 {
		t.Errorf("RollDice(2,6,3) = %d, want 5-15", result)
	}
}

func TestRollDiceDetailed(t *testing.T) {
	roller := newSeededRoller(42)
	result := roller.RollDiceDetailed(3, 8, 2)

	if len(result.DiceValues) != 3 {
		t.Errorf("DiceValues len = %d, want 3", len(result.DiceValues))
	}
	if result.FlatBonus != 2 {
		t.Errorf("FlatBonus = %d, want 2", result.FlatBonus)
	}

	sum := result.FlatBonus
	for _, v := range result.DiceValues {
		if v < 1 || v > 8 {
			t.Errorf("die value = %d, want 1-8", v)
		}
		sum += v
	}
	if result.Total != sum {
		t.Errorf("Total = %d, want %d", result.Total, sum)
	}
}

func TestRollDice(t *testing.T) {
	roller := newSeededRoller(42)
	result := roller.RollDice(4, 6, 0)
	if result < 4 || result > 24 {
		t.Errorf("RollDice(4,6) = %d, want 4-24", result)
	}
}

func TestRollDuality(t *testing.T) {
	roller := newSeededRoller(42)

	for i := 0; i < 100; i++ {
		result := roller.RollDuality(2, 0)

		if result.HopeDie < 1 || result.HopeDie > 12 {
			t.Errorf("HopeDie = %d, want 1-12", result.HopeDie)
		}
		if result.FearDie < 1 || result.FearDie > 12 {
			t.Errorf("FearDie = %d, want 1-12", result.FearDie)
		}
		if result.HopeTotal != result.HopeDie+2 {
			t.Errorf("HopeTotal = %d, want %d", result.HopeTotal, result.HopeDie+2)
		}

		if result.HopeDie == result.FearDie {
			if !result.IsCritical {
				t.Error("matching dice should be critical")
			}
			if result.Result != models.ResultCriticalHope {
				t.Errorf("critical result = %v, want CriticalHope", result.Result)
			}
		} else {
			if result.IsCritical {
				t.Error("non-matching dice should not be critical")
			}
			if result.HopeDie >= result.FearDie && result.Result != models.ResultHope {
				t.Error("hope >= fear should be ResultHope")
			}
			if result.HopeDie < result.FearDie && result.Result != models.ResultFear {
				t.Error("hope < fear should be ResultFear")
			}
		}
	}
}

func TestRollD20Attack(t *testing.T) {
	roller := newSeededRoller(42)

	for i := 0; i < 100; i++ {
		result := roller.RollD20Attack(5)
		if result.D20Roll < 1 || result.D20Roll > 20 {
			t.Errorf("D20Roll = %d, want 1-20", result.D20Roll)
		}
		if result.Modifier != 5 {
			t.Errorf("Modifier = %d, want 5", result.Modifier)
		}
		if result.Total != result.D20Roll+5 {
			t.Errorf("Total = %d, want %d", result.Total, result.D20Roll+5)
		}
		if result.IsCritical != (result.D20Roll == 20) {
			t.Errorf("IsCritical = %v, want %v", result.IsCritical, result.D20Roll == 20)
		}
	}
}

func TestGetRNG(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	roller := NewStandardDiceRoller(rng)
	if roller.GetRNG() != rng {
		t.Error("GetRNG should return the same rng instance")
	}
}
