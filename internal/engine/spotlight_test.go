package engine

import (
	"math/rand"
	"testing"

	"github.com/skarlso/combat-simulator/internal/models"
)

func TestChooseNextEmpty(t *testing.T) {
	selector := NewHybridSpotlightSelector(rand.New(rand.NewSource(42)), 0.5)
	result := selector.ChooseNext(nil, 5)
	if result != nil {
		t.Error("ChooseNext with nil should return nil")
	}

	result = selector.ChooseNext([]*models.Combatant{}, 5)
	if result != nil {
		t.Error("ChooseNext with empty slice should return nil")
	}
}

func TestChooseNextSingle(t *testing.T) {
	selector := NewHybridSpotlightSelector(rand.New(rand.NewSource(42)), 0.5)
	c := models.NewCombatant("p1", "Hero", models.SidePlayer, 3, 12, 2, 15, 8)

	result := selector.ChooseNext([]*models.Combatant{c}, 5)
	if result != c {
		t.Error("single combatant should always be chosen")
	}
}

func TestChooseNextForcedByThreshold(t *testing.T) {
	selector := NewHybridSpotlightSelector(rand.New(rand.NewSource(42)), 0.5)

	low := models.NewCombatant("p1", "Low", models.SidePlayer, 3, 12, 2, 15, 8)
	low.SpotlightTokens = 1

	high := models.NewCombatant("p2", "High", models.SidePlayer, 3, 12, 2, 15, 8)
	high.SpotlightTokens = 5

	for range 50 {
		result := selector.ChooseNext([]*models.Combatant{low, high}, 5)
		if result != high {
			t.Error("combatant at threshold should always be forced")
		}
	}
}

func TestChooseNextWeightedDistribution(t *testing.T) {
	selector := NewHybridSpotlightSelector(rand.New(rand.NewSource(42)), 1.0)

	zero := models.NewCombatant("p1", "Zero", models.SidePlayer, 3, 12, 2, 15, 8)
	zero.SpotlightTokens = 0

	heavy := models.NewCombatant("p2", "Heavy", models.SidePlayer, 3, 12, 2, 15, 8)
	heavy.SpotlightTokens = 10

	counts := map[string]int{}
	for range 1000 {
		result := selector.ChooseNext([]*models.Combatant{zero, heavy}, 20)
		counts[result.Name]++
	}

	if counts["Heavy"] < counts["Zero"] {
		t.Errorf("Heavy (%d) should be picked more than Zero (%d)", counts["Heavy"], counts["Zero"])
	}
}
