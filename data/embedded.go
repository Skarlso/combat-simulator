package data

import (
	"embed"
	"fmt"

	"github.com/skarlso/daggerheart-combat-simulator/internal/parser"
)

//go:embed *.json
var EmbeddedDataFS embed.FS

var adversaryFiles = []string{
	"daggerheart_adversaries-tier-1.json",
	"daggerheart_adversaries_tier2.json",
	"daggerheart_adversaries_tier3.json",
	"daggerheart_adversaries_tier4.json",
}

// GetFeatureMappings reads the embedded feature mappings JSON
func GetFeatureMappings() ([]byte, error) {
	return EmbeddedDataFS.ReadFile("feature_mappings.json")
}

// NewEmbeddedAdversaryRegistry creates an adversary registry from all embedded data
func NewEmbeddedAdversaryRegistry() (*parser.AdversaryRegistry, error) {
	mappingData, err := GetFeatureMappings()
	if err != nil {
		return nil, fmt.Errorf("failed to read feature mappings: %w", err)
	}

	adversaryData := make([][]byte, 0, len(adversaryFiles))
	for _, name := range adversaryFiles {
		raw, err := EmbeddedDataFS.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", name, err)
		}
		adversaryData = append(adversaryData, raw)
	}

	return parser.NewAdversaryRegistryFromBytes(mappingData, adversaryData)
}
