package parser

import (
	"fmt"

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

		// Index by name
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
