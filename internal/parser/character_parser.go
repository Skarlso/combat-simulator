package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/skarlso/combat-simulator/internal/models"
)

// CharacterFile represents a bulk PC import document.
type CharacterFile struct {
	Characters []CharacterJSON `json:"characters"`
}

// CharacterJSON is the import schema for a single player character.
type CharacterJSON struct {
	Name            string              `json:"name"`
	Level           int                 `json:"level"`
	Evasion         int                 `json:"evasion"`
	Proficiency     int                 `json:"proficiency"`
	Traits          CharacterTraits     `json:"traits"`
	WeaponTrait     string              `json:"weapon_trait,omitempty"`
	SpellcastTrait  string              `json:"spellcast_trait,omitempty"`
	HP              int                 `json:"hp"`
	Stress          int                 `json:"stress"`
	Armor           int                 `json:"armor,omitempty"`
	DamageDice      string              `json:"damage_dice,omitempty"`
	HealthPotions   int                 `json:"health_potions,omitempty"`
	StaminaPotions  int                 `json:"stamina_potions,omitempty"`
	Thresholds      CharacterThresholds `json:"thresholds"`
	DomainAbilities []DomainAbilityJSON `json:"domain_abilities,omitempty"`
}

// CharacterTraits maps to Combatant.Traits.
type CharacterTraits struct {
	Agility   int `json:"agility,omitempty"`
	Strength  int `json:"strength,omitempty"`
	Finesse   int `json:"finesse,omitempty"`
	Instinct  int `json:"instinct,omitempty"`
	Presence  int `json:"presence,omitempty"`
	Knowledge int `json:"knowledge,omitempty"`
}

// CharacterThresholds maps to Combatant.CustomThresholds.
type CharacterThresholds struct {
	Minor int `json:"minor"`
	Major int `json:"major"`
}

// DomainAbilityJSON is the import schema for a single domain ability.
type DomainAbilityJSON struct {
	Name          string                  `json:"name"`
	Description   string                  `json:"description,omitempty"`
	Passive       bool                    `json:"passive,omitempty"`
	HopeCost      int                     `json:"hope_cost,omitempty"`
	StressCost    int                     `json:"stress_cost,omitempty"`
	RefreshType   string                  `json:"refresh_type,omitempty"`
	MaxUses       int                     `json:"max_uses,omitempty"`
	DCType        string                  `json:"dc_type,omitempty"`
	FixedDC       int                     `json:"fixed_dc,omitempty"`
	RollTrait     string                  `json:"roll_trait,omitempty"`
	UsesTokens    bool                    `json:"uses_tokens,omitempty"`
	TokenName     string                  `json:"token_name,omitempty"`
	TokensMax     int                     `json:"tokens_max,omitempty"`
	Trigger       string                  `json:"trigger,omitempty"`
	TriggerFilter string                  `json:"trigger_filter,omitempty"`
	Effects       []DomainAbilityEffectJS `json:"effects,omitempty"`
}

// DomainAbilityEffectJS is the import schema for a single effect.
type DomainAbilityEffectJS struct {
	Category        string   `json:"category"`
	Magnitude       int      `json:"magnitude,omitempty"`
	DiceFormula     string   `json:"dice_formula,omitempty"`
	ProficiencyDice bool     `json:"proficiency_dice,omitempty"`
	Duration        string   `json:"duration,omitempty"`
	TargetType      string   `json:"target_type,omitempty"`
	DamageType      string   `json:"damage_type,omitempty"`
	ReactionTrait   string   `json:"reaction_trait,omitempty"`
	ReactionDC      int      `json:"reaction_dc,omitempty"`
	HalfOnSuccess   bool     `json:"half_on_success,omitempty"`
	PerToken        bool     `json:"per_token,omitempty"`
	StandardAttack  bool     `json:"standard_attack,omitempty"`
	HalfDamage      bool     `json:"half_damage,omitempty"`
	ScaleBy         string   `json:"scale_by,omitempty"`
	OthersOnly      bool     `json:"others_only,omitempty"`
	Conditions      []string `json:"conditions,omitempty"`
}

// LoadCharactersFromBytes parses a CharacterFile JSON blob and returns combatants.
func LoadCharactersFromBytes(data []byte) ([]*models.Combatant, error) {
	var file CharacterFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse character file: %w", err)
	}

	combatants := make([]*models.Combatant, 0, len(file.Characters))
	for i, c := range file.Characters {
		combatant, err := parseCharacter(c, i)
		if err != nil {
			return nil, fmt.Errorf("character %d (%q): %w", i+1, c.Name, err)
		}
		combatants = append(combatants, combatant)
	}

	return combatants, nil
}

func parseCharacter(c CharacterJSON, idx int) (*models.Combatant, error) {
	if c.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	id := fmt.Sprintf("%s_%d", strings.ToLower(strings.ReplaceAll(c.Name, " ", "_")), idx)

	combatant := models.NewCombatant(id, c.Name, models.SidePlayer, c.Level, c.Evasion, c.Proficiency, c.HP, c.Stress)
	combatant.Traits = models.Traits(c.Traits)
	if err := checkTrait("weapon_trait", c.WeaponTrait); err != nil {
		return nil, err
	}
	if err := checkTrait("spellcast_trait", c.SpellcastTrait); err != nil {
		return nil, err
	}
	combatant.WeaponTrait, combatant.SpellcastTrait = c.WeaponTrait, c.SpellcastTrait
	combatant.ArmorSlots = c.Armor
	combatant.DamageDice = c.DamageDice
	combatant.HealthPotions = min(max(c.HealthPotions, 0), models.MaxPotionsEach)
	combatant.StaminaPotions = min(max(c.StaminaPotions, 0), models.MaxPotionsEach)

	if c.Thresholds.Minor > 0 || c.Thresholds.Major > 0 {
		combatant.CustomThresholds = []int{c.Thresholds.Minor, c.Thresholds.Major}
	}

	for _, a := range c.DomainAbilities {
		ability, err := buildDomainAbility(a)
		if err != nil {
			return nil, fmt.Errorf("domain ability %q: %w", a.Name, err)
		}
		combatant.DomainAbilities = append(combatant.DomainAbilities, ability)
	}

	combatant.ApplyPassiveResistances()
	return combatant, nil
}

func buildDomainAbility(a DomainAbilityJSON) (models.DomainAbility, error) {
	effects := make([]models.FeatureEffect, 0, len(a.Effects))
	for _, e := range a.Effects {
		effect, err := convertMappingEffect(FeatureMappingEffect{
			Category:        e.Category,
			Magnitude:       e.Magnitude,
			DiceFormula:     e.DiceFormula,
			ProficiencyDice: e.ProficiencyDice,
			Duration:        e.Duration,
			TargetType:      e.TargetType,
			DamageType:      e.DamageType,
			ReactionTrait:   e.ReactionTrait,
			ReactionDC:      e.ReactionDC,
			HalfOnSuccess:   e.HalfOnSuccess,
			PerToken:        e.PerToken,
			StandardAttack:  e.StandardAttack,
			HalfDamage:      e.HalfDamage,
			ScaleBy:         e.ScaleBy,
			OthersOnly:      e.OthersOnly,
			Conditions:      e.Conditions,
		})
		if err != nil {
			return models.DomainAbility{}, err
		}
		effects = append(effects, effect)
	}

	if err := checkTrait("roll_trait", a.RollTrait, "spellcast"); err != nil {
		return models.DomainAbility{}, err
	}
	return models.DomainAbility{
		Name:        a.Name,
		Description: a.Description,
		Passive:     a.Passive,
		Trigger:     models.FeatureTriggerCondition{Type: models.FeatureTriggerType(a.Trigger), Filter: a.TriggerFilter},
		HopeCost:    a.HopeCost,
		StressCost:  a.StressCost,
		RefreshType: a.RefreshType,
		MaxUses:     a.MaxUses,
		UsesLeft:    a.MaxUses,
		DCType:      a.DCType,
		FixedDC:     a.FixedDC,
		RollTrait:   a.RollTrait,
		UsesTokens:  a.UsesTokens,
		TokenName:   a.TokenName,
		TokensMax:   a.TokensMax,
		Effects:     effects,
	}, nil
}

// ExportCharacters writes a party in the import schema, so a built party can be saved and reloaded
func ExportCharacters(pcs []*models.Combatant) ([]byte, error) {
	file := CharacterFile{Characters: make([]CharacterJSON, 0, len(pcs))}
	for _, pc := range pcs {
		file.Characters = append(file.Characters, characterToJSON(pc))
	}
	return json.MarshalIndent(file, "", "  ")
}

func characterToJSON(pc *models.Combatant) CharacterJSON {
	c := CharacterJSON{
		Name:           pc.Name,
		Level:          pc.Level,
		Evasion:        pc.Evasion,
		Proficiency:    pc.Proficiency,
		Traits:         CharacterTraits(pc.Traits),
		WeaponTrait:    pc.WeaponTrait,
		SpellcastTrait: pc.SpellcastTrait,
		HP:             pc.HPMax,
		Stress:         pc.StressMax,
		Armor:          pc.ArmorSlots,
		DamageDice:     pc.DamageDice,
		HealthPotions:  pc.HealthPotions,
		StaminaPotions: pc.StaminaPotions,
	}
	if len(pc.CustomThresholds) >= 2 {
		c.Thresholds = CharacterThresholds{Minor: pc.CustomThresholds[0], Major: pc.CustomThresholds[1]}
	}
	for _, a := range pc.DomainAbilities {
		c.DomainAbilities = append(c.DomainAbilities, domainAbilityToJSON(a))
	}
	return c
}

func domainAbilityToJSON(a models.DomainAbility) DomainAbilityJSON {
	out := DomainAbilityJSON{
		Name:          a.Name,
		Description:   a.Description,
		Passive:       a.Passive,
		HopeCost:      a.HopeCost,
		StressCost:    a.StressCost,
		RefreshType:   a.RefreshType,
		MaxUses:       a.MaxUses,
		DCType:        a.DCType,
		FixedDC:       a.FixedDC,
		RollTrait:     a.RollTrait,
		UsesTokens:    a.UsesTokens,
		TokenName:     a.TokenName,
		TokensMax:     a.TokensMax,
		Trigger:       string(a.Trigger.Type),
		TriggerFilter: a.Trigger.Filter,
	}
	for _, e := range a.Effects {
		var conditions []string
		for _, cond := range e.Conditions {
			conditions = append(conditions, string(cond))
		}
		out.Effects = append(out.Effects, DomainAbilityEffectJS{
			Category:        effectCategoryName(e.Category),
			Magnitude:       e.Magnitude,
			DiceFormula:     e.DiceFormula,
			ProficiencyDice: e.ProficiencyDice,
			Duration:        e.Duration,
			TargetType:      string(e.TargetType),
			DamageType:      e.DamageType,
			ReactionTrait:   e.ReactionTrait,
			ReactionDC:      e.ReactionDC,
			HalfOnSuccess:   e.HalfOnSuccess,
			PerToken:        e.PerToken,
			StandardAttack:  e.StandardAttack,
			HalfDamage:      e.HalfDamage,
			ScaleBy:         e.ScaleBy,
			OthersOnly:      e.OthersOnly,
			Conditions:      conditions,
		})
	}
	return out
}

// effectCategoryName is the mapping-file spelling of a category, the inverse of stringToEffectCategory
func effectCategoryName(c models.FeatureEffectCategory) string {
	for name, cat := range effectCategoryMap {
		if cat == c {
			return name
		}
	}
	return string(c)
}
