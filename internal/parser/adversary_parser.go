package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/skarlso/combat-simulator/internal/models"
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

// LoadAdversariesFromBytes loads all adversaries from in-memory JSON data
func (ap *AdversaryParser) LoadAdversariesFromBytes(data []byte) ([]*models.Combatant, error) {
	var advFile AdversaryFile
	if err := json.Unmarshal(data, &advFile); err != nil {
		return nil, fmt.Errorf("failed to parse adversary file: %w", err)
	}

	combatants := make([]*models.Combatant, 0, len(advFile.Adversaries))
	for _, adv := range advFile.Adversaries {
		combatants = append(combatants, ap.ParseAdversary(adv))
	}
	return combatants, nil
}

// ParseAdversary converts an AdversaryJSON into a Combatant
func (ap *AdversaryParser) ParseAdversary(adv AdversaryJSON) *models.Combatant {
	id := fmt.Sprintf("%s_%d", strings.ToLower(strings.ReplaceAll(adv.Name, " ", "_")), adv.Tier)
	combatant := models.NewAdversary(id, adv.Name, fmt.Sprintf("Tier %d", adv.Tier), adv.Type,
		adv.Difficulty, adv.Attack.Bonus, adv.HP, adv.Stress, []int{adv.Thresholds.Minor, adv.Thresholds.Major})
	// PC-side mirrors keep threat scoring and thresholds working for adversaries too
	combatant.Level = adv.Tier
	combatant.Evasion = adv.Difficulty
	combatant.Proficiency = adv.Attack.Bonus
	combatant.DamageDice = adv.Attack.Damage
	combatant.Motives = strings.Join(adv.MotivesTactics, ", ")

	for _, jsonFeature := range adv.Features {
		if strings.EqualFold(jsonFeature.Name, "Relentless") {
			combatant.RelentlessLimit = jsonFeature.Uses
			continue
		}

		feature := ap.featureParser.ParseFeatureFor(adv.Name, jsonFeature.Name, jsonFeature.Description)
		if feature == nil {
			continue
		}
		// Reactions and Passives fire on triggers; the stat block decides, not the mapping
		if strings.EqualFold(jsonFeature.Type, "Passive") || strings.EqualFold(jsonFeature.Type, "Reaction") {
			feature.Passive = true
		}
		if jsonFeature.Uses > 0 {
			feature.MaxUses = jsonFeature.Uses
			feature.UsesLeft = jsonFeature.Uses
		}
		combatant.AdversaryFeatures = append(combatant.AdversaryFeatures, *feature)
	}

	return combatant
}
