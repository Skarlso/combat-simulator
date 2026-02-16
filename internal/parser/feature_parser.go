package parser

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// FeatureMapping represents a mapping from feature name to quantized effects
type FeatureMapping struct {
	Category   string                 `json:"category"`
	FearCost   int                    `json:"fear_cost"`
	StressCost int                    `json:"stress_cost"`
	Passive    bool                   `json:"passive"`
	Effects    []FeatureMappingEffect `json:"effects"`
}

// FeatureMappingEffect represents a single effect in the mapping
type FeatureMappingEffect struct {
	Category    string `json:"category"`
	Magnitude   int    `json:"magnitude"`
	DiceFormula string `json:"dice_formula"`
	DamageType  string `json:"damage_type"`
	Condition   string `json:"condition"`
	Duration    string `json:"duration"`
	TargetType  string `json:"target_type"`
	SummonType  string `json:"summon_type"`
	Description string `json:"description"`
}

// FeatureMappingFile represents the JSON structure of feature_mappings.json
type FeatureMappingFile struct {
	Mappings map[string]FeatureMapping `json:"mappings"`
	Metadata struct {
		Version        string `json:"version"`
		MappedFeatures int    `json:"mapped_features"`
		TotalFeatures  int    `json:"total_features"`
		Coverage       string `json:"coverage"`
		LastUpdated    string `json:"last_updated"`
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

	return NewFeatureParserFromBytes(data)
}

// NewFeatureParserFromBytes creates a parser from in-memory JSON data
func NewFeatureParserFromBytes(data []byte) (*FeatureParser, error) {
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
			SummonType:  mappingEffect.SummonType,
		}

		// Set damage type if specified
		if mappingEffect.DamageType != "" {
			effect.DamageType = mappingEffect.DamageType
		}

		feature.Effects = append(feature.Effects, effect)
	}

	return feature
}

var effectCategoryMap = map[string]models.FeatureEffectCategory{
	"FeatureEffectDamage":              models.FeatureEffectDamage,
	"FeatureEffectDirectDamage":        models.FeatureEffectDirectDamage,
	"FeatureEffectApplyVulnerable":     models.FeatureEffectApplyVulnerable,
	"FeatureEffectApplyPoison":         models.FeatureEffectApplyPoison,
	"FeatureEffectApplyIgnite":         models.FeatureEffectApplyIgnite,
	"FeatureEffectApplyTaunt":          models.FeatureEffectApplyTaunt,
	"FeatureEffectApplyHidden":         models.FeatureEffectApplyHidden,
	"FeatureEffectStressInflict":       models.FeatureEffectStressInflict,
	"FeatureEffectStressClear":         models.FeatureEffectStressClear,
	"FeatureEffectHPHeal":              models.FeatureEffectHPHeal,
	"FeatureEffectArmorDamage":         models.FeatureEffectArmorDamage,
	"FeatureEffectHopeDrain":           models.FeatureEffectHopeDrain,
	"FeatureEffectFearGenerate":        models.FeatureEffectFearGenerate,
	"FeatureEffectDamageResistance":    models.FeatureEffectDamageResistance,
	"FeatureEffectDamageVulnerability": models.FeatureEffectDamageVulnerability,
	"FeatureEffectSpotlightModifier":   models.FeatureEffectSpotlightModifier,
	"FeatureEffectAllySpotlight":       models.FeatureEffectAllySpotlight,
	"FeatureEffectSummonAllies":        models.FeatureEffectSummonAllies,
	"FeatureEffectTokenPlace":          models.FeatureEffectTokenPlace,
	"FeatureEffectTokenSpend":          models.FeatureEffectTokenSpend,
	"FeatureEffectAreaDamage":          models.FeatureEffectAreaDamage,
	"FeatureEffectGrantArmor":          models.FeatureEffectGrantArmor,
	"FeatureEffectDamageReduction":     models.FeatureEffectDamageReduction,
}

func stringToEffectCategory(s string) models.FeatureEffectCategory {
	if cat, ok := effectCategoryMap[s]; ok {
		return cat
	}
	return models.FeatureEffectDamage
}

var targetTypeMap = map[string]models.FeatureTargetType{
	"self":   models.FeatureTargetSelf,
	"single": models.FeatureTargetSingle,
	"aoe":    models.FeatureTargetAOE,
	"allies": models.FeatureTargetAllies,
}

func stringToTargetType(s string) models.FeatureTargetType {
	if tt, ok := targetTypeMap[s]; ok {
		return tt
	}
	return models.FeatureTargetSingle
}
