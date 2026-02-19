package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// AdversaryJSONFeature represents a feature in the JSON
type AdversaryJSONFeature struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Uses        int    `json:"uses,omitempty"`
	Description string `json:"description"`
}

// AdversaryJSON represents an adversary in the JSON file
type AdversaryJSON struct {
	Name           string   `json:"name"`
	Tier           int      `json:"tier"`
	LevelRange     string   `json:"level_range"`
	Type           string   `json:"type"`
	Description    string   `json:"description"`
	MotivesTactics []string `json:"motives_tactics"`
	Difficulty     int      `json:"difficulty"`
	Thresholds     struct {
		Minor int `json:"minor"`
		Major int `json:"major"`
	} `json:"thresholds"`
	HP     int `json:"hp"`
	Stress int `json:"stress"`
	Attack struct {
		Bonus      int    `json:"bonus"`
		Name       string `json:"name"`
		Range      string `json:"range"`
		Damage     string `json:"damage"`
		DamageType string `json:"damage_type"`
	} `json:"attack"`
	Features []AdversaryJSONFeature `json:"features"`
}

// AdversaryFile represents the JSON file structure
type AdversaryFile struct {
	Adversaries []AdversaryJSON `json:"adversaries"`
}

// AdversaryParser parses adversary JSON files
type AdversaryParser struct {
	featureParser *FeatureParser
}

// NewAdversaryParser creates a parser with feature mapping
func NewAdversaryParser(mappingFilePath string) (*AdversaryParser, error) {
	fp, err := NewFeatureParser(mappingFilePath)
	if err != nil {
		return nil, err
	}

	return &AdversaryParser{
		featureParser: fp,
	}, nil
}

// NewAdversaryParserFromBytes creates a parser from in-memory feature mappings
func NewAdversaryParserFromBytes(mappingData []byte) (*AdversaryParser, error) {
	fp, err := NewFeatureParserFromBytes(mappingData)
	if err != nil {
		return nil, err
	}

	return &AdversaryParser{
		featureParser: fp,
	}, nil
}

// LoadAdversariesFromFile loads all adversaries from a JSON file
func (ap *AdversaryParser) LoadAdversariesFromFile(filePath string) ([]*models.Combatant, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read adversary file: %w", err)
	}

	return ap.LoadAdversariesFromBytes(data)
}

// LoadAdversariesFromBytes loads all adversaries from in-memory JSON data
func (ap *AdversaryParser) LoadAdversariesFromBytes(data []byte) ([]*models.Combatant, error) {
	var advFile AdversaryFile
	if err := json.Unmarshal(data, &advFile); err != nil {
		return nil, fmt.Errorf("failed to parse adversary file: %w", err)
	}

	combatants := make([]*models.Combatant, 0, len(advFile.Adversaries))
	for _, adv := range advFile.Adversaries {
		combatant := ap.ParseAdversary(adv)
		combatants = append(combatants, combatant)
	}

	return combatants, nil
}

// ParseAdversary converts an AdversaryJSON into a Combatant
func (ap *AdversaryParser) ParseAdversary(adv AdversaryJSON) *models.Combatant {
	// Generate unique ID
	id := fmt.Sprintf("%s_%d", strings.ToLower(strings.ReplaceAll(adv.Name, " ", "_")), adv.Tier)

	// Create combatant
	combatant := &models.Combatant{
		ID:                id,
		Name:              adv.Name,
		Side:              models.SideEnemy,
		IsAdversary:       true,
		Level:             adv.Tier,
		HP:                adv.HP,
		HPMax:             adv.HP,
		Stress:            0,
		StressMax:         adv.Stress,
		Evasion:           adv.Difficulty,
		Proficiency:       adv.Attack.Bonus,
		ArmorSlots:        0,
		DamageDice:        adv.Attack.Damage,
		Tier:              fmt.Sprintf("Tier %d", adv.Tier),
		AdversaryType:     adv.Type,
		Difficulty:        adv.Difficulty,
		ATK:               adv.Attack.Bonus,
		Motives:           strings.Join(adv.MotivesTactics, ", "),
		Tactics:           "",
		CustomThresholds:  []int{adv.Thresholds.Minor, adv.Thresholds.Major},
		Conditions:        make([]*models.ActiveCondition, 0),
		Resistances:       make([]models.Resistance, 0),
		Experience:        make([]models.ExperienceSkill, 0),
		AdversaryFeatures: make([]models.AdversaryFeature, 0),
		DomainAbilities:   make([]models.DomainAbility, 0),
	}

	for _, jsonFeature := range adv.Features {
		if strings.EqualFold(jsonFeature.Name, "Relentless") {
			combatant.RelentlessLimit = jsonFeature.Uses
			continue
		}

		feature := ap.featureParser.ParseFeature(jsonFeature.Name, jsonFeature.Description)
		if feature != nil {
			if jsonFeature.Uses > 0 {
				feature.MaxUses = jsonFeature.Uses
				feature.UsesLeft = jsonFeature.Uses
			}
			combatant.AdversaryFeatures = append(combatant.AdversaryFeatures, *feature)
		}
	}

	return combatant
}
