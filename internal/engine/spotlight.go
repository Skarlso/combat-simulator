package engine

import (
	"math/rand"

	"github.com/skarlso/combat-simulator/internal/models"
)

// SpotlightSelector chooses the next combatant to act
type SpotlightSelector interface {
	ChooseNext(combatants []*models.Combatant, threshold int) *models.Combatant
}

// HybridSpotlightSelector implements weighted random selection with forced threshold
type HybridSpotlightSelector struct {
	rng          *rand.Rand
	weightFactor float64 // How much tokens bias selection
}

// NewHybridSpotlightSelector creates a selector with given weight factor
func NewHybridSpotlightSelector(rng *rand.Rand, weightFactor float64) *HybridSpotlightSelector {
	return &HybridSpotlightSelector{
		rng:          rng,
		weightFactor: weightFactor,
	}
}

// ChooseNext selects next actor based on spotlight tokens
func (hss *HybridSpotlightSelector) ChooseNext(combatants []*models.Combatant, threshold int) *models.Combatant {
	if len(combatants) == 0 {
		return nil
	}

	// Single combatant always goes
	if len(combatants) == 1 {
		return combatants[0]
	}

	// Check for forced selection (at or above threshold)
	forced := make([]*models.Combatant, 0)
	for _, c := range combatants {
		if c.SpotlightTokens >= threshold {
			forced = append(forced, c)
		}
	}

	// If anyone is forced, randomly pick from them
	if len(forced) > 0 {
		return forced[hss.rng.Intn(len(forced))]
	}

	// Otherwise, weighted random selection
	return hss.weightedRandomSelection(combatants)
}

// weightedRandomSelection picks based on token weights
func (hss *HybridSpotlightSelector) weightedRandomSelection(combatants []*models.Combatant) *models.Combatant {
	// Calculate weights (base probability + token bonus)
	weights := make([]float64, len(combatants))
	totalWeight := 0.0

	for i, c := range combatants {
		// Base weight of 1.0, plus token bonus
		weights[i] = 1.0 + (float64(c.SpotlightTokens) * hss.weightFactor)
		totalWeight += weights[i]
	}

	// Random selection weighted by tokens
	roll := hss.rng.Float64() * totalWeight
	cumulative := 0.0

	for i, weight := range weights {
		cumulative += weight
		if roll <= cumulative {
			return combatants[i]
		}
	}

	// Fallback (shouldn't happen)
	return combatants[len(combatants)-1]
}
