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
	"daggerheart_domains_blade_bone.json",
	"daggerheart_domains_remaining.json",
}

// LoadDomainCards builds the domain card library from the embedded mappings and SRD card data
func LoadDomainCards() ([]parser.DomainCard, error) {
	mappingData, err := EmbeddedDataFS.ReadFile("domain_mappings.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read domain mappings: %w", err)
	}
	domainData, err := readAll(domainFiles)
	if err != nil {
		return nil, err
	}
	return parser.LoadDomainCards(mappingData, domainData)
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
