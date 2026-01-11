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

// NewAdversaryRegistry creates a registry with loaded adversaries
func NewAdversaryRegistry(mappingFilePath string, adversaryFiles []string) (*AdversaryRegistry, error) {
	parser, err := NewAdversaryParser(mappingFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create adversary parser: %w", err)
	}

	registry := &AdversaryRegistry{
		templates: make(map[string]*models.Combatant),
		parser:    parser,
	}

	// Load all adversaries from files
	for _, filePath := range adversaryFiles {
		combatants, err := parser.LoadAdversariesFromFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load adversaries from %s: %w", filePath, err)
		}

		// Index by name
		for _, c := range combatants {
			registry.templates[c.Name] = c
		}
	}

	return registry, nil
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

// GetTemplate returns a template adversary by name
func (ar *AdversaryRegistry) GetTemplate(name string) (*models.Combatant, error) {
	template, exists := ar.templates[name]
	if !exists {
		return nil, fmt.Errorf("adversary template not found: %s", name)
	}
	return template, nil
}

// CloneTemplate creates a fresh instance of an adversary for summoning
func (ar *AdversaryRegistry) CloneTemplate(name string, uniqueID string) (*models.Combatant, error) {
	template, err := ar.GetTemplate(name)
	if err != nil {
		return nil, err
	}

	// Deep clone the template
	clone := &models.Combatant{
		ID:                uniqueID,
		Name:              template.Name,
		Side:              models.SideEnemy,
		Evasion:           template.Evasion,
		Proficiency:       template.Proficiency,
		IsAdversary:       true,
		Level:             template.Level,
		Tier:              template.Tier,
		AdversaryType:     template.AdversaryType,
		Difficulty:        template.Difficulty,
		ATK:               template.ATK,
		Motives:           template.Motives,
		Tactics:           template.Tactics,
		CustomThresholds:  append([]int{}, template.CustomThresholds...),
		HP:                template.HPMax,
		HPMax:             template.HPMax,
		Stress:            0,
		StressMax:         template.StressMax,
		ArmorSlots:        template.ArmorSlots,
		DamageDice:        template.DamageDice,
		SpotlightTokens:   0,
		AdversaryFeatures: append([]models.AdversaryFeature{}, template.AdversaryFeatures...),
		DomainAbilities:   make([]models.DomainAbility, 0),
		Resistances:       append([]models.Resistance{}, template.Resistances...),
		Conditions:        make([]*models.ActiveCondition, 0),
		Experience:        append([]models.ExperienceSkill{}, template.Experience...),
		IsDefeated:        false,
		IsFullyStressed:   false,
	}

	return clone, nil
}

// ListTemplates returns all available template names
func (ar *AdversaryRegistry) ListTemplates() []string {
	names := make([]string, 0, len(ar.templates))
	for name := range ar.templates {
		names = append(names, name)
	}
	return names
}
