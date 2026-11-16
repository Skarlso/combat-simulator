package parser

import (
	"encoding/json"
	"fmt"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// FeatureMapping represents a mapping from feature name to quantized effects
type FeatureMapping struct {
	Category   string                 `json:"category"`
	FearCost   int                    `json:"fear_cost"`
	StressCost int                    `json:"stress_cost"`
	Passive    bool                   `json:"passive"`
	Trigger    string                 `json:"trigger,omitempty"`
	DCType     string                 `json:"dc_type,omitempty"`
	FixedDC    int                    `json:"fixed_dc,omitempty"`
	UsesTokens bool                   `json:"uses_tokens,omitempty"`
	TokenName  string                 `json:"token_name,omitempty"`
	TokensMax  int                    `json:"tokens_max,omitempty"`
	Effects    []FeatureMappingEffect `json:"effects"`
}

// FeatureMappingEffect represents a single effect in the mapping
type FeatureMappingEffect struct {
	Category       string `json:"category"`
	Magnitude      int    `json:"magnitude"`
	DiceFormula    string `json:"dice_formula"`
	StandardAttack bool   `json:"standard_attack,omitempty"`
	DamageType     string `json:"damage_type"`
	ArmorCost      int    `json:"armor_cost,omitempty"`
	FallbackFear   int    `json:"fallback_fear,omitempty"`
	ReactionTrait  string `json:"reaction_trait,omitempty"`
	ReactionDC     int    `json:"reaction_dc,omitempty"`
	HalfOnSuccess  bool   `json:"half_on_success,omitempty"`
	Condition      string `json:"condition"`
	Duration       string `json:"duration"`
	TargetType     string `json:"target_type"`
	SummonType     string `json:"summon_type"`
	Description    string `json:"description"`
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

// NewFeatureParserFromBytes creates a parser from in-memory JSON data
func NewFeatureParserFromBytes(data []byte) (*FeatureParser, error) {
	var mappingFile FeatureMappingFile
	if err := json.Unmarshal(data, &mappingFile); err != nil {
		return nil, fmt.Errorf("failed to parse mapping file: %w", err)
	}

	for name, mapping := range mappingFile.Mappings {
		for i, effect := range mapping.Effects {
			if _, err := convertMappingEffect(effect); err != nil {
				return nil, fmt.Errorf("mapping %q effect %d: %w", name, i, err)
			}
		}
	}

	return &FeatureParser{
		mappings: mappingFile.Mappings,
	}, nil
}

// FeatureNames returns the names of all mapped features.
func (fp *FeatureParser) FeatureNames() []string {
	names := make([]string, 0, len(fp.mappings))
	for name := range fp.mappings {
		names = append(names, name)
	}
	return names
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
		Trigger:     models.FeatureTriggerCondition{Type: models.FeatureTriggerType(mapping.Trigger)},
		FearCost:    mapping.FearCost,
		StressCost:  mapping.StressCost,
		DCType:      mapping.DCType,
		FixedDC:     mapping.FixedDC,
		UsesTokens:  mapping.UsesTokens,
		TokenName:   mapping.TokenName,
		TokensMax:   mapping.TokensMax,
		Effects:     make([]models.FeatureEffect, 0, len(mapping.Effects)),
	}

	// Effects were validated when the mapping file was loaded
	for _, mappingEffect := range mapping.Effects {
		effect, _ := convertMappingEffect(mappingEffect)
		feature.Effects = append(feature.Effects, effect)
	}

	return feature
}

func convertMappingEffect(m FeatureMappingEffect) (models.FeatureEffect, error) {
	category, err := stringToEffectCategory(m.Category)
	if err != nil {
		return models.FeatureEffect{}, err
	}
	targetType, err := stringToTargetType(m.TargetType)
	if err != nil {
		return models.FeatureEffect{}, err
	}

	return models.FeatureEffect{
		Category:       category,
		Magnitude:      m.Magnitude,
		DiceFormula:    m.DiceFormula,
		StandardAttack: m.StandardAttack,
		Duration:       m.Duration,
		TargetType:     targetType,
		SummonType:     m.SummonType,
		DamageType:     m.DamageType,
		ArmorCost:      m.ArmorCost,
		FallbackFear:   m.FallbackFear,
		ReactionTrait:  m.ReactionTrait,
		ReactionDC:     m.ReactionDC,
		HalfOnSuccess:  m.HalfOnSuccess,
	}, nil
}

var effectCategoryMap = map[string]models.FeatureEffectCategory{
	"FeatureEffectDamage":              models.FeatureEffectDamage,
	"FeatureEffectDirectDamage":        models.FeatureEffectDirectDamage,
	"FeatureEffectApplyVulnerable":     models.FeatureEffectApplyVulnerable,
	"FeatureEffectApplyPoison":         models.FeatureEffectApplyPoison,
	"FeatureEffectApplyIgnite":         models.FeatureEffectApplyIgnite,
	"FeatureEffectApplyTaunt":          models.FeatureEffectApplyTaunt,
	"FeatureEffectApplyHidden":         models.FeatureEffectApplyHidden,
	"FeatureEffectApplyStunned":        models.FeatureEffectApplyStunned,
	"FeatureEffectApplyRestrained":     models.FeatureEffectApplyRestrained,
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

func stringToEffectCategory(s string) (models.FeatureEffectCategory, error) {
	if cat, ok := effectCategoryMap[s]; ok {
		return cat, nil
	}
	return "", fmt.Errorf("unknown effect category %q", s)
}

var targetTypeMap = map[string]models.FeatureTargetType{
	"self":   models.FeatureTargetSelf,
	"single": models.FeatureTargetSingle,
	"aoe":    models.FeatureTargetAOE,
	"allies": models.FeatureTargetAllies,
}

func stringToTargetType(s string) (models.FeatureTargetType, error) {
	if s == "" {
		return models.FeatureTargetSingle, nil
	}
	if tt, ok := targetTypeMap[s]; ok {
		return tt, nil
	}
	return "", fmt.Errorf("unknown target type %q", s)
}
