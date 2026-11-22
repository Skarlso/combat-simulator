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

var domainFiles = []string{
	"daggerheart_domain_arcana.json",
	"daggerheart_domain_codex.json",
	"daggerheart_domain_dread.json",
	"daggerheart_domains_blade_bone.json",
	"daggerheart_domains_remaining.json",
}

var playerFeatureFiles = []string{"daggerheart_player_features.json"}

var abilityMappingFiles = []string{"domain_mappings.json", "player_feature_mappings.json"}

// LoadAbilityLibrary builds the player ability library from the embedded SRD text and mappings
func LoadAbilityLibrary() ([]parser.LibraryAbility, error) {
	mappingData, err := readAll(abilityMappingFiles)
	if err != nil {
		return nil, err
	}
	domainData, err := readAll(domainFiles)
	if err != nil {
		return nil, err
	}
	featureData, err := readAll(playerFeatureFiles)
	if err != nil {
		return nil, err
	}
	return parser.LoadAbilityLibrary(mappingData, domainData, featureData)
}

func readAll(names []string) ([][]byte, error) {
	out := make([][]byte, 0, len(names))
	for _, name := range names {
		raw, err := EmbeddedDataFS.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", name, err)
		}
		out = append(out, raw)
	}
	return out, nil
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

	adversaryData, err := readAll(adversaryFiles)
	if err != nil {
		return nil, err
	}
	return parser.NewAdversaryRegistryFromBytes(mappingData, adversaryData)
}
