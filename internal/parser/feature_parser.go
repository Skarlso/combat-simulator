package parser

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/skarlso/combat-simulator/internal/models"
)

// FeatureMapping represents a mapping from feature name to quantized effects
type FeatureMapping struct {
	Category      string                 `json:"category"`
	FearCost      int                    `json:"fear_cost"`
	StressCost    int                    `json:"stress_cost"`
	CostOnSuccess bool                   `json:"cost_on_success,omitempty"`
	Passive       bool                   `json:"passive"`
	Trigger       string                 `json:"trigger,omitempty"`
	TriggerFilter string                 `json:"trigger_filter,omitempty"`
	DCType        string                 `json:"dc_type,omitempty"`
	FixedDC       int                    `json:"fixed_dc,omitempty"`
	UsesTokens    bool                   `json:"uses_tokens,omitempty"`
	TokenName     string                 `json:"token_name,omitempty"`
	TokensMax     int                    `json:"tokens_max,omitempty"`
	MaxUses       int                    `json:"max_uses,omitempty"`
	Effects       []FeatureMappingEffect `json:"effects"`
}

// FeatureMappingEffect represents a single effect in the mapping
type FeatureMappingEffect struct {
	Category        string `json:"category"`
	Magnitude       int    `json:"magnitude"`
	DiceFormula     string `json:"dice_formula"`
	StandardAttack  bool   `json:"standard_attack,omitempty"`
	ProficiencyDice bool   `json:"proficiency_dice,omitempty"`
	DamageType      string `json:"damage_type"`
	ArmorCost       int    `json:"armor_cost,omitempty"`
	FallbackFear    int    `json:"fallback_fear,omitempty"`
	ReactionTrait   string `json:"reaction_trait,omitempty"`
	ReactionDC      int    `json:"reaction_dc,omitempty"`
	HalfOnSuccess   bool   `json:"half_on_success,omitempty"`
	Condition       string `json:"condition"`
	Duration        string `json:"duration"`
	TargetType      string `json:"target_type"`
	SummonType      string `json:"summon_type"`
	SpotlightAll    bool   `json:"spotlight_all,omitempty"`
	SpotlightCount  int    `json:"spotlight_count,omitempty"`
	AllyName        string `json:"ally_name,omitempty"`
	HalfDamage      bool   `json:"half_damage,omitempty"`
	PerToken        bool   `json:"per_token,omitempty"`
	ScaleBy         string `json:"scale_by,omitempty"`
	OthersOnly      bool   `json:"others_only,omitempty"`
	// Conditions a condition_immunity effect blocks
	Conditions  []string `json:"conditions,omitempty"`
	Description string   `json:"description"`
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
		for part := range strings.SplitSeq(mapping.TriggerFilter, "+") {
			if !slices.Contains(knownFilters, part) {
				return nil, fmt.Errorf("mapping %q: unknown trigger filter %q", name, part)
			}
		}
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
	return fp.ParseFeatureFor("", name, description)
}

// ParseFeatureFor prefers an "Adversary::Feature" mapping, since the same feature
// name carries different numbers on different stat blocks.
func (fp *FeatureParser) ParseFeatureFor(adversaryName, name, description string) *models.AdversaryFeature {
	mapping, exists := fp.mappings[adversaryName+"::"+name]
	if !exists {
		mapping, exists = fp.mappings[name]
	}
	if !exists {
		return nil // Feature not mapped yet
	}

	feature := &models.AdversaryFeature{
		Name:          name,
		Description:   description,
		Passive:       mapping.Passive,
		Trigger:       models.FeatureTriggerCondition{Type: models.FeatureTriggerType(mapping.Trigger), Filter: mapping.TriggerFilter},
		FearCost:      mapping.FearCost,
		StressCost:    mapping.StressCost,
		CostOnSuccess: mapping.CostOnSuccess,
		DCType:        mapping.DCType,
		FixedDC:       mapping.FixedDC,
		UsesTokens:    mapping.UsesTokens,
		TokenName:     mapping.TokenName,
		TokensMax:     mapping.TokensMax,
		MaxUses:       mapping.MaxUses,
		UsesLeft:      mapping.MaxUses,
		Effects:       make([]models.FeatureEffect, 0, len(mapping.Effects)),
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
	conditions, err := stringsToConditions(m.Conditions)
	if err != nil {
		return models.FeatureEffect{}, err
	}
	if !slices.Contains(scaleNames, m.ScaleBy) {
		return models.FeatureEffect{}, fmt.Errorf("unknown scale_by %q", m.ScaleBy)
	}

	return models.FeatureEffect{
		Category:        category,
		Magnitude:       m.Magnitude,
		DiceFormula:     m.DiceFormula,
		StandardAttack:  m.StandardAttack,
		ProficiencyDice: m.ProficiencyDice,
		Duration:        m.Duration,
		TargetType:      targetType,
		SummonType:      m.SummonType,
		DamageType:      m.DamageType,
		ArmorCost:       m.ArmorCost,
		FallbackFear:    m.FallbackFear,
		ReactionTrait:   m.ReactionTrait,
		ReactionDC:      m.ReactionDC,
		HalfOnSuccess:   m.HalfOnSuccess,
		SpotlightAll:    m.SpotlightAll,
		SpotlightCount:  m.SpotlightCount,
		AllyName:        m.AllyName,
		HalfDamage:      m.HalfDamage,
		PerToken:        m.PerToken,
		ScaleBy:         m.ScaleBy,
		OthersOnly:      m.OthersOnly,
		Conditions:      conditions,
	}, nil
}

// scaleNames are the values scale_by accepts; see Combatant.ScaleValue
var scaleNames = []string{"", "level", "tier", "proficiency", "spellcast", "agility", "strength", "finesse", "instinct", "presence", "knowledge"}

// traitNames are the six traits a PC can roll with
var traitNames = []string{"agility", "strength", "finesse", "instinct", "presence", "knowledge"}

// checkTrait accepts a trait name, an empty string, or any of the extra names given
func checkTrait(field, value string, extra ...string) error {
	if value == "" || slices.Contains(traitNames, value) || slices.Contains(extra, value) {
		return nil
	}
	return fmt.Errorf("unknown %s %q", field, value)
}

var knownConditions = []models.ConditionType{
	models.ConditionBurning, models.ConditionPoisoned, models.ConditionVulnerable, models.ConditionResistant,
	models.ConditionStunned, models.ConditionRestrained, models.ConditionAdvantage, models.ConditionDisadvantage,
	models.ConditionTaunted, models.ConditionHidden,
}

func stringsToConditions(names []string) ([]models.ConditionType, error) {
	if len(names) == 0 {
		return nil, nil
	}
	out := make([]models.ConditionType, 0, len(names))
	for _, name := range names {
		cond := models.ConditionType(name)
		if !slices.Contains(knownConditions, cond) {
			return nil, fmt.Errorf("unknown condition %q", name)
		}
		out = append(out, cond)
	}
	return out, nil
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
	"FeatureEffectArmorClear":          models.FeatureEffectArmorClear,
	"FeatureEffectDamageBonus":         models.FeatureEffectDamageBonus,
	"FeatureEffectSeverityReduction":   models.FeatureEffectSeverityReduction,
	"FeatureEffectConditionImmunity":   models.FeatureEffectConditionImmunity,
	"FeatureEffectTokenOnDamage":       models.FeatureEffectTokenOnDamage,
	"FeatureEffectMarkHP":              models.FeatureEffectMarkHP,
	"FeatureEffectApplyAdvantage":      models.FeatureEffectApplyAdvantage,
	"FeatureEffectApplyDisadvantage":   models.FeatureEffectApplyDisadvantage,
	"FeatureEffectDefensePenalty":      models.FeatureEffectDefensePenalty,
	"FeatureEffectDamageMultiplier":    models.FeatureEffectDamageMultiplier,
	"FeatureEffectAttackBonus":         models.FeatureEffectAttackBonus,
	"FeatureEffectEvasionBonus":        models.FeatureEffectEvasionBonus,
	"FeatureEffectDamageReduction":     models.FeatureEffectDamageReduction,
	"FeatureEffectRedirectDamage":      models.FeatureEffectRedirectDamage,
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
	"ally":   models.FeatureTargetAlly,
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
