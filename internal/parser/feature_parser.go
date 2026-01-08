package parser

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// FeatureMapping represents a mapping from feature name to quantized effects
type FeatureMapping struct {
	Category    string                   `json:"category"`
	FearCost    int                      `json:"fear_cost"`
	StressCost  int                      `json:"stress_cost"`
	Passive     bool                     `json:"passive"`
	Effects     []FeatureMappingEffect   `json:"effects"`
}

// FeatureMappingEffect represents a single effect in the mapping
type FeatureMappingEffect struct {
	Category     string `json:"category"`
	Magnitude    int    `json:"magnitude"`
	DiceFormula  string `json:"dice_formula"`
	DamageType   string `json:"damage_type"`
	Condition    string `json:"condition"`
	Duration     string `json:"duration"`
	TargetType   string `json:"target_type"`
	SummonType   string `json:"summon_type"`
	Description  string `json:"description"`
}

// FeatureMappingFile represents the JSON structure of feature_mappings.json
type FeatureMappingFile struct {
	Mappings map[string]FeatureMapping `json:"mappings"`
	Metadata struct {
		Version        string  `json:"version"`
		MappedFeatures int     `json:"mapped_features"`
		TotalFeatures  int     `json:"total_features"`
		Coverage       string  `json:"coverage"`
		LastUpdated    string  `json:"last_updated"`
	} `json:"metadata"`
}

// FeatureParser converts JSON feature descriptions into quantized AdversaryFeatures
type FeatureParser struct {
	mappings map[string]FeatureMapping
}

// NewFeatureParser creates a parser with loaded mappings
func NewFeatureParser(mappingFilePath string) (*FeatureParser, error) {
	data, err := os.ReadFile(mappingFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file: %w", err)
	}

	var mappingFile FeatureMappingFile
	if err := json.Unmarshal(data, &mappingFile); err != nil {
		return nil, fmt.Errorf("failed to parse mapping file: %w", err)
	}

	return &FeatureParser{
		mappings: mappingFile.Mappings,
	}, nil
}

// ParseFeature converts a feature name into an AdversaryFeature
// Returns nil if feature is not mapped
func (fp *FeatureParser) ParseFeature(name string, description string) *models.AdversaryFeature {
	mapping, exists := fp.mappings[name]
	if !exists {
		return nil // Feature not mapped yet
	}

	feature := &models.AdversaryFeature{
		Name:        name,
		Description: description,
		Passive:     mapping.Passive,
		FearCost:    mapping.FearCost,
		Effects:     make([]models.FeatureEffect, 0, len(mapping.Effects)),
	}

	// Convert mapping effects to FeatureEffects
	for _, mappingEffect := range mapping.Effects {
		effect := models.FeatureEffect{
			Category:    stringToEffectCategory(mappingEffect.Category),
			Magnitude:   mappingEffect.Magnitude,
			DiceFormula: mappingEffect.DiceFormula,
			Duration:    mappingEffect.Duration,
			TargetType:  stringToTargetType(mappingEffect.TargetType),
		}

		// Set damage type if specified
		if mappingEffect.DamageType != "" {
			effect.DamageType = mappingEffect.DamageType
		}

		feature.Effects = append(feature.Effects, effect)
	}

	return feature
}

// stringToEffectCategory converts string to FeatureEffectCategory
func stringToEffectCategory(s string) models.FeatureEffectCategory {
	switch s {
	case "FeatureEffectDamage":
		return models.FeatureEffectDamage
	case "FeatureEffectDirectDamage":
		return models.FeatureEffectDirectDamage
	case "FeatureEffectApplyVulnerable":
		return models.FeatureEffectApplyVulnerable
	case "FeatureEffectApplyPoison":
		return models.FeatureEffectApplyPoison
	case "FeatureEffectApplyIgnite":
		return models.FeatureEffectApplyIgnite
	case "FeatureEffectApplyTaunt":
		return models.FeatureEffectApplyTaunt
	case "FeatureEffectApplyHidden":
		return models.FeatureEffectApplyHidden
	case "FeatureEffectStressInflict":
		return models.FeatureEffectStressInflict
	case "FeatureEffectStressClear":
		return models.FeatureEffectStressClear
	case "FeatureEffectHPHeal":
		return models.FeatureEffectHPHeal
	case "FeatureEffectArmorDamage":
		return models.FeatureEffectArmorDamage
	case "FeatureEffectHopeDrain":
		return models.FeatureEffectHopeDrain
	case "FeatureEffectFearGenerate":
		return models.FeatureEffectFearGenerate
	case "FeatureEffectDamageResistance":
		return models.FeatureEffectDamageResistance
	case "FeatureEffectDamageVulnerability":
		return models.FeatureEffectDamageVulnerability
	case "FeatureEffectSpotlightModifier":
		return models.FeatureEffectSpotlightModifier
	case "FeatureEffectAllySpotlight":
		return models.FeatureEffectAllySpotlight
	case "FeatureEffectSummonAllies":
		return models.FeatureEffectSummonAllies
	default:
		return models.FeatureEffectDamage // Default fallback
	}
}

// stringToTargetType converts string to FeatureTargetType
func stringToTargetType(s string) models.FeatureTargetType {
	switch s {
	case "self":
		return models.FeatureTargetSelf
	case "single":
		return models.FeatureTargetSingle
	case "aoe":
		return models.FeatureTargetAOE
	case "allies":
		return models.FeatureTargetAllies
	default:
		return models.FeatureTargetSingle
	}
}
