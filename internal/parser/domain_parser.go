package parser

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// DomainCardMapping is how domain_mappings.json describes one card the engine can run
type DomainCardMapping struct {
	HopeCost    int                    `json:"hope_cost,omitempty"`
	StressCost  int                    `json:"stress_cost,omitempty"`
	Passive     bool                   `json:"passive,omitempty"`
	Trigger     string                 `json:"trigger,omitempty"`
	DCType      string                 `json:"dc_type,omitempty"`
	FixedDC     int                    `json:"fixed_dc,omitempty"`
	MaxUses     int                    `json:"max_uses,omitempty"`
	RefreshType string                 `json:"refresh_type,omitempty"`
	Note        string                 `json:"note,omitempty"`
	Effects     []FeatureMappingEffect `json:"effects"`
}

type domainMappingFile struct {
	Mappings map[string]DomainCardMapping `json:"mappings"`
}

// DomainCard is a library entry: the SRD card text plus the ability the engine runs for it
type DomainCard struct {
	Name        string
	Domain      string
	Level       int
	Type        string
	Description string
	Note        string
	Ability     models.DomainAbility
}

type domainJSONCard struct {
	Name        string `json:"name"`
	Level       int    `json:"level"`
	Type        string `json:"type"`
	Description string `json:"description"`
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

// LoadDomainCards builds the domain card library: every mapped card, described from the SRD card data
func LoadDomainCards(mappingData []byte, domainData [][]byte) ([]DomainCard, error) {
	var mappingFile domainMappingFile
	if err := json.Unmarshal(mappingData, &mappingFile); err != nil {
		return nil, fmt.Errorf("failed to parse domain mappings: %w", err)
	}

	cards := map[string]DomainCard{}
	for i, raw := range domainData {
		if err := collectDomainCards(raw, cards); err != nil {
			return nil, fmt.Errorf("domain data %d: %w", i, err)
		}
	}

	library := make([]DomainCard, 0, len(mappingFile.Mappings))
	for name, mapping := range mappingFile.Mappings {
		card, ok := cards[name]
		if !ok {
			return nil, fmt.Errorf("domain mapping %q names no card in the domain data", name)
		}
		ability, err := buildMappedAbility(name, card.Description, mapping)
		if err != nil {
			return nil, fmt.Errorf("domain mapping %q: %w", name, err)
		}
		card.Note = mapping.Note
		card.Ability = ability
		library = append(library, card)
	}

	sort.Slice(library, func(i, j int) bool {
		a, b := library[i], library[j]
		if a.Domain != b.Domain {
			return a.Domain < b.Domain
		}
		if a.Level != b.Level {
			return a.Level < b.Level
		}
		return a.Name < b.Name
	})
	return library, nil
}

func collectDomainCards(raw []byte, into map[string]DomainCard) error {
	var file domainJSONFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return err
	}
	add := func(domain string, cards []domainJSONCard) {
		for _, c := range cards {
			into[c.Name] = DomainCard{Name: c.Name, Domain: domain, Level: c.Level, Type: c.Type, Description: c.Description}
		}
	}
	add(file.Domain, file.Cards)
	if len(file.Domains) == 0 {
		return nil
	}

	var byName map[string]domainJSONDomain
	if err := json.Unmarshal(file.Domains, &byName); err == nil {
		for name, d := range byName {
			add(name, d.Cards)
		}
		return nil
	}
	var list []domainJSONDomain
	if err := json.Unmarshal(file.Domains, &list); err != nil {
		return fmt.Errorf("unrecognised domains layout: %w", err)
	}
	for _, d := range list {
		add(d.Domain, d.Cards)
	}
	return nil
}

func buildMappedAbility(name, description string, m DomainCardMapping) (models.DomainAbility, error) {
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
		Trigger:     models.FeatureTriggerCondition{Type: models.FeatureTriggerType(m.Trigger)},
		HopeCost:    m.HopeCost,
		StressCost:  m.StressCost,
		RefreshType: refresh,
		MaxUses:     m.MaxUses,
		UsesLeft:    m.MaxUses,
		DCType:      m.DCType,
		FixedDC:     m.FixedDC,
		Effects:     effects,
	}, nil
}
