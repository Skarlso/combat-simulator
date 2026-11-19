package parser

import (
	"fmt"
	"sort"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// AdversaryRegistry stores adversary templates for summoning
type AdversaryRegistry struct {
	templates map[string]*models.Combatant
	parser    *AdversaryParser
}

// NewAdversaryRegistryFromBytes creates a registry from in-memory JSON data
func NewAdversaryRegistryFromBytes(mappingData []byte, adversaryDataSlices [][]byte) (*AdversaryRegistry, error) {
	parser, err := NewAdversaryParserFromBytes(mappingData)
	if err != nil {
		return nil, fmt.Errorf("failed to create adversary parser: %w", err)
	}

	registry := &AdversaryRegistry{
		templates: make(map[string]*models.Combatant),
		parser:    parser,
	}

	// Load all adversaries from byte slices
	for i, data := range adversaryDataSlices {
		combatants, err := parser.LoadAdversariesFromBytes(data)
		if err != nil {
			return nil, fmt.Errorf("failed to load adversaries from data slice %d: %w", i, err)
		}

		for _, c := range combatants {
			registry.templates[c.Name] = c
		}
	}

	return registry, nil
}

// CloneTemplate creates a fresh instance of an adversary for summoning
func (ar *AdversaryRegistry) CloneTemplate(name string, uniqueID string) (*models.Combatant, error) {
	template, exists := ar.templates[name]
	if !exists {
		return nil, fmt.Errorf("adversary template not found: %s", name)
	}
	return models.CloneCombatant(template, uniqueID), nil
}

// Templates returns every loaded adversary ordered by tier, then name.
func (ar *AdversaryRegistry) Templates() []*models.Combatant {
	out := make([]*models.Combatant, 0, len(ar.templates))
	for _, c := range ar.templates {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Tier != out[j].Tier {
			return out[i].Tier < out[j].Tier
		}
		return out[i].Name < out[j].Name
	})
	return out
}
