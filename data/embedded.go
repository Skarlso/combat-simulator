package data

import (
	"embed"
	"fmt"

	"github.com/skarlso/daggerheart-combat-simulator/internal/parser"
)

//go:embed *.json
var EmbeddedDataFS embed.FS

// GetFeatureMappings reads the embedded feature mappings JSON
func GetFeatureMappings() ([]byte, error) {
	return EmbeddedDataFS.ReadFile("feature_mappings.json")
}

// GetAdversariesTier1 reads the embedded tier 1 adversaries JSON
func GetAdversariesTier1() ([]byte, error) {
	return EmbeddedDataFS.ReadFile("daggerheart_adversaries-tier-1.json")
}

// GetAdversariesTier2 reads the embedded tier 2 adversaries JSON
func GetAdversariesTier2() ([]byte, error) {
	return EmbeddedDataFS.ReadFile("daggerheart_adversaries_tier2.json")
}

// GetAdversariesTier3 reads the embedded tier 3 adversaries JSON
func GetAdversariesTier3() ([]byte, error) {
	return EmbeddedDataFS.ReadFile("daggerheart_adversaries_tier3.json")
}

// GetAdversariesTier4 reads the embedded tier 4 adversaries JSON
func GetAdversariesTier4() ([]byte, error) {
	return EmbeddedDataFS.ReadFile("daggerheart_adversaries_tier4.json")
}

// NewEmbeddedAdversaryRegistry creates an adversary registry from all embedded data
func NewEmbeddedAdversaryRegistry() (*parser.AdversaryRegistry, error) {
	mappingData, err := GetFeatureMappings()
	if err != nil {
		return nil, fmt.Errorf("failed to read feature mappings: %w", err)
	}

	tier1, err := GetAdversariesTier1()
	if err != nil {
		return nil, fmt.Errorf("failed to read tier 1 adversaries: %w", err)
	}

	tier2, err := GetAdversariesTier2()
	if err != nil {
		return nil, fmt.Errorf("failed to read tier 2 adversaries: %w", err)
	}

	tier3, err := GetAdversariesTier3()
	if err != nil {
		return nil, fmt.Errorf("failed to read tier 3 adversaries: %w", err)
	}

	tier4, err := GetAdversariesTier4()
	if err != nil {
		return nil, fmt.Errorf("failed to read tier 4 adversaries: %w", err)
	}

	adversaryData := [][]byte{tier1, tier2, tier3, tier4}

	return parser.NewAdversaryRegistryFromBytes(mappingData, adversaryData)
}
