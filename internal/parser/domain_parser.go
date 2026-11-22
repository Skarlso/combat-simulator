package parser

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// AbilityMapping is how the mapping files describe one player ability the engine can run
type AbilityMapping struct {
	HopeCost      int                    `json:"hope_cost,omitempty"`
	StressCost    int                    `json:"stress_cost,omitempty"`
	Passive       bool                   `json:"passive,omitempty"`
	Trigger       string                 `json:"trigger,omitempty"`
	TriggerFilter string                 `json:"trigger_filter,omitempty"`
	DCType        string                 `json:"dc_type,omitempty"`
	FixedDC       int                    `json:"fixed_dc,omitempty"`
	RollTrait     string                 `json:"roll_trait,omitempty"`
	MaxUses       int                    `json:"max_uses,omitempty"`
	RefreshType   string                 `json:"refresh_type,omitempty"`
	UsesTokens    bool                   `json:"uses_tokens,omitempty"`
	TokenName     string                 `json:"token_name,omitempty"`
	TokensMax     int                    `json:"tokens_max,omitempty"`
	LevelUpgrades []LevelUpgrade         `json:"level_upgrades,omitempty"`
	Note          string                 `json:"note,omitempty"`
	Effects       []FeatureMappingEffect `json:"effects"`
}

// LevelUpgrade replaces a mapped ability's numbers once the PC reaches Level
type LevelUpgrade struct {
	Level     int `json:"level"`
	TokensMax int `json:"tokens_max,omitempty"`
}

type abilityMappingFile struct {
	Mappings map[string]AbilityMapping `json:"mappings"`
}

// Where a library ability comes from
const (
	SourceDomainCard     = "Domain Card"
	SourceClass          = "Class"
	SourceSubclass       = "Subclass"
	SourceAncestry       = "Ancestry"
	SourceCommunity      = "Community"
	SourceTransformation = "Transformation"
)

// AbilitySources lists the sources in the order the library sorts them
var AbilitySources = []string{SourceClass, SourceSubclass, SourceDomainCard, SourceAncestry, SourceCommunity, SourceTransformation}

// LibraryAbility is one pickable player ability: a domain card or a class, subclass, ancestry,
// community or transformation feature. Modelled ones carry the ability the engine runs for them.
type LibraryAbility struct {
	Name   string
	Source string
	Group  string // the domain, class, subclass, ancestry, community or transformation it belongs to
	// Classes that can take it: the owner of a class or subclass feature, or every class with the card's domain
	Classes     []string
	Level       int    // domain cards only
	Type        string // "Spell", "Hope Feature", "Foundation", ...
	Description string
	Note        string
	Modelled    bool
	Ability     models.DomainAbility
	upgrades    []LevelUpgrade
}

// AbilityAt is the ability as a PC of this level has it
func (l LibraryAbility) AbilityAt(level int) models.DomainAbility {
	ability := l.Ability
	ability.Effects = slices.Clone(l.Ability.Effects)
	for _, up := range l.upgrades {
		if level >= up.Level && up.TokensMax > 0 {
			ability.TokensMax = up.TokensMax
		}
	}
	return ability
}

type domainJSONSpell struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type domainJSONCard struct {
	Name        string            `json:"name"`
	Level       int               `json:"level"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Mechanics   string            `json:"mechanics"`
	Spells      []domainJSONSpell `json:"spells"`
}

// text is the whole card: grimoires list their spells, some cards split off a mechanics paragraph
func (c domainJSONCard) text() string {
	parts := []string{c.Description}
	for _, s := range c.Spells {
		parts = append(parts, s.Name+": "+s.Description)
	}
	parts = append(parts, c.Mechanics)
	return strings.Join(slices.DeleteFunc(parts, func(p string) bool { return p == "" }), " ")
}

type domainJSONDomain struct {
	Domain string           `json:"domain"`
	Cards  []domainJSONCard `json:"cards"`
}

// domainJSONFile covers both shapes the data ships in: one domain per file, or several under "domains"
type domainJSONFile struct {
	Domain  string           `json:"domain"`
	Cards   []domainJSONCard `json:"cards"`
	Domains json.RawMessage  `json:"domains"`
}

type featureJSON struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type featureGroupJSON struct {
	Name       string             `json:"name"`
	Domains    []string           `json:"domains"`
	Features   []featureJSON      `json:"features"`
	Subclasses []featureGroupJSON `json:"subclasses"`
}

// playerFeaturesFile is daggerheart_player_features.json: every player feature that is not a domain card
type playerFeaturesFile struct {
	Classes         []featureGroupJSON `json:"classes"`
	Ancestries      []featureGroupJSON `json:"ancestries"`
	Communities     []featureGroupJSON `json:"communities"`
	Transformations []featureGroupJSON `json:"transformations"`
}

// LoadAbilityLibrary builds the player ability library: every domain card and player feature in
// the SRD data, with the mapped ones runnable. A mapping key is the ability's name, or
// "Group::Name" when two groups share a name.
func LoadAbilityLibrary(mappingData, domainData, featureData [][]byte) ([]LibraryAbility, error) {
	mappings := map[string]AbilityMapping{}
	for i, raw := range mappingData {
		var file abilityMappingFile
		if err := json.Unmarshal(raw, &file); err != nil {
			return nil, fmt.Errorf("ability mappings %d: %w", i, err)
		}
		for name, mapping := range file.Mappings {
			mappings[name] = mapping
		}
	}

	var library []LibraryAbility
	for i, raw := range domainData {
		cards, err := readDomainCards(raw)
		if err != nil {
			return nil, fmt.Errorf("domain data %d: %w", i, err)
		}
		library = append(library, cards...)
	}
	for i, raw := range featureData {
		var file playerFeaturesFile
		if err := json.Unmarshal(raw, &file); err != nil {
			return nil, fmt.Errorf("player features %d: %w", i, err)
		}
		for c := range library {
			if card := &library[c]; card.Source == SourceDomainCard {
				card.Classes = append(card.Classes, classesWithDomain(file, card.Group)...)
			}
		}
		library = append(library, playerFeatures(file)...)
	}

	used := map[string]bool{}
	for i := range library {
		entry := &library[i]
		key := entry.Group + "::" + entry.Name
		mapping, ok := mappings[key]
		if !ok {
			key = entry.Name
			mapping, ok = mappings[key]
		}
		if !ok {
			entry.Ability = models.DomainAbility{Name: entry.Name, Description: entry.Description, RefreshType: "unlimited"}
			continue
		}
		ability, err := buildMappedAbility(entry.Name, entry.Description, mapping)
		if err != nil {
			return nil, fmt.Errorf("ability mapping %q: %w", key, err)
		}
		// Spells and grimoires roll with the Spellcast trait unless their mapping names another
		if ability.RollTrait == "" && (entry.Type == "Spell" || entry.Type == "Grimoire") {
			ability.RollTrait = "spellcast"
		}
		used[key] = true
		entry.Modelled = true
		entry.Note = mapping.Note
		entry.Ability = ability
		entry.upgrades = mapping.LevelUpgrades
	}
	for name := range mappings {
		if !used[name] {
			return nil, fmt.Errorf("ability mapping %q names nothing in the SRD data", name)
		}
	}

	// Features keep their SRD order inside a group; cards go by level
	sort.SliceStable(library, func(i, j int) bool {
		a, b := library[i], library[j]
		if a.Source != b.Source {
			return slices.Index(AbilitySources, a.Source) < slices.Index(AbilitySources, b.Source)
		}
		if a.Source != SourceDomainCard {
			return false
		}
		if a.Group != b.Group {
			return a.Group < b.Group
		}
		if a.Level != b.Level {
			return a.Level < b.Level
		}
		return a.Name < b.Name
	})
	return library, nil
}

func readDomainCards(raw []byte) ([]LibraryAbility, error) {
	var file domainJSONFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, err
	}
	var out []LibraryAbility
	add := func(domain string, cards []domainJSONCard) {
		for _, c := range cards {
			card := LibraryAbility{Name: c.Name, Source: SourceDomainCard, Group: domain, Level: c.Level, Type: c.Type, Description: c.text()}
			if len(c.Spells) == 0 {
				out = append(out, card)
				continue
			}
			// A grimoire is several spells; each is picked and run on its own
			for _, spell := range c.Spells {
				card.Name, card.Description = c.Name+": "+spell.Name, spell.Description
				out = append(out, card)
			}
		}
	}
	add(file.Domain, file.Cards)
	if len(file.Domains) == 0 {
		return out, nil
	}

	var byName map[string]domainJSONDomain
	if err := json.Unmarshal(file.Domains, &byName); err == nil {
		for _, name := range slices.Sorted(maps.Keys(byName)) {
			add(name, byName[name].Cards)
		}
		return out, nil
	}
	var list []domainJSONDomain
	if err := json.Unmarshal(file.Domains, &list); err != nil {
		return nil, fmt.Errorf("unrecognised domains layout: %w", err)
	}
	for _, d := range list {
		add(d.Domain, d.Cards)
	}
	return out, nil
}

func classesWithDomain(file playerFeaturesFile, domain string) []string {
	var classes []string
	for _, class := range file.Classes {
		if slices.Contains(class.Domains, domain) {
			classes = append(classes, class.Name)
		}
	}
	return classes
}

func playerFeatures(file playerFeaturesFile) []LibraryAbility {
	var out []LibraryAbility
	add := func(source string, classes []string, group featureGroupJSON, defaultType string) {
		for _, f := range group.Features {
			kind := f.Type
			if kind == "" {
				kind = defaultType
			}
			out = append(out, LibraryAbility{Name: f.Name, Source: source, Group: group.Name, Classes: classes, Type: kind, Description: f.Description})
		}
	}
	for _, class := range file.Classes {
		add(SourceClass, []string{class.Name}, class, "Class Feature")
		for _, sub := range class.Subclasses {
			add(SourceSubclass, []string{class.Name}, sub, "Subclass Feature")
		}
	}
	for _, g := range file.Ancestries {
		add(SourceAncestry, nil, g, "Ancestry Feature")
	}
	for _, g := range file.Communities {
		add(SourceCommunity, nil, g, "Community Feature")
	}
	for _, g := range file.Transformations {
		add(SourceTransformation, nil, g, "Transformation Feature")
	}
	return out
}

// Triggers and filters a mapping may name; anything else is a typo that would never fire
var (
	knownTriggers = []models.FeatureTriggerType{"",
		models.TriggerOnAttack, models.TriggerOnHit, models.TriggerOnDamageTaken, models.TriggerOnHitTaken,
		models.TriggerOnCritical, models.TriggerStartTurn, models.TriggerEndTurn, models.TriggerOnAllyDamaged,
		models.TriggerOnMiss, models.TriggerOnDamageDealt, models.TriggerOnEvaded,
		models.TriggerOnDamageRoll, models.TriggerOnDamageIncoming, models.TriggerOnAttacked,
	}
	knownFilters = []string{"", "if_target_below_50_hp", "if_hidden", "if_target_vulnerable", "if_target_fully_stressed",
		"if_dealt_severe", "if_took_severe", "if_severe", "if_major_or_worse", "if_minor",
		"if_vulnerable", "if_low_hp", "if_fully_stressed", "if_hope_2", "if_out_of_armor", "if_rolled_fear", "if_rolled_hope"}
)

func buildMappedAbility(name, description string, m AbilityMapping) (models.DomainAbility, error) {
	if !slices.Contains(knownTriggers, models.FeatureTriggerType(m.Trigger)) {
		return models.DomainAbility{}, fmt.Errorf("unknown trigger %q", m.Trigger)
	}
	for _, part := range strings.Split(m.TriggerFilter, "+") {
		if !slices.Contains(knownFilters, part) {
			return models.DomainAbility{}, fmt.Errorf("unknown trigger filter %q", part)
		}
	}
	if err := checkTrait("roll_trait", m.RollTrait, "spellcast"); err != nil {
		return models.DomainAbility{}, err
	}
	effects := make([]models.FeatureEffect, 0, len(m.Effects))
	for i, e := range m.Effects {
		effect, err := convertMappingEffect(e)
		if err != nil {
			return models.DomainAbility{}, fmt.Errorf("effect %d: %w", i, err)
		}
		effects = append(effects, effect)
	}
	refresh := m.RefreshType
	if refresh == "" {
		refresh = "unlimited"
	}
	return models.DomainAbility{
		Name:        name,
		Description: description,
		Passive:     m.Passive,
		Trigger:     models.FeatureTriggerCondition{Type: models.FeatureTriggerType(m.Trigger), Filter: m.TriggerFilter},
		HopeCost:    m.HopeCost,
		StressCost:  m.StressCost,
		RefreshType: refresh,
		MaxUses:     m.MaxUses,
		UsesLeft:    m.MaxUses,
		DCType:      m.DCType,
		FixedDC:     m.FixedDC,
		RollTrait:   m.RollTrait,
		UsesTokens:  m.UsesTokens,
		TokenName:   m.TokenName,
		TokensMax:   m.TokensMax,
		Effects:     effects,
	}, nil
}
