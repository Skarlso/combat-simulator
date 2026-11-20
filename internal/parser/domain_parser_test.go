package parser

import (
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func TestLoadDomainCardsReadsBothDataLayouts(t *testing.T) {
	mappings := `{"mappings": {
		"Healing Hands": {"stress_cost": 1, "dc_type": "fixed", "fixed_dc": 13, "max_uses": 1, "refresh_type": "scene",
			"effects": [{"category": "FeatureEffectHPHeal", "magnitude": 2, "target_type": "ally"}]},
		"I Am Your Shield": {"passive": true, "trigger": "on_ally_damaged", "stress_cost": 1,
			"effects": [{"category": "FeatureEffectRedirectDamage"}]}}}`
	single := `{"domain": "Splendor", "cards": [{"name": "Healing Hands", "level": 2, "type": "Spell", "description": "Heal."}]}`
	nested := `{"domains": {"Valor": {"cards": [{"name": "I Am Your Shield", "level": 1, "type": "Ability", "description": "Shield."}]}}}`
	listed := `{"domains": [{"domain": "Blade", "cards": [{"name": "Reckless", "level": 2, "type": "Ability", "description": "Adv."}]}]}`

	cards, err := LoadDomainCards([]byte(mappings), [][]byte{[]byte(single), []byte(nested), []byte(listed)})
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 2 || cards[0].Name != "Healing Hands" || cards[1].Name != "I Am Your Shield" {
		t.Fatalf("cards = %+v", cards)
	}
	hands := cards[0]
	if hands.Domain != "Splendor" || hands.Level != 2 || hands.Ability.FixedDC != 13 || hands.Ability.UsesLeft != 1 ||
		hands.Ability.Effects[0].TargetType != models.FeatureTargetAlly || hands.Ability.Description != "Heal." {
		t.Errorf("Healing Hands = %+v", hands)
	}
	shield := cards[1].Ability
	if !shield.Passive || shield.Trigger.Type != models.TriggerOnAllyDamaged || shield.Effects[0].Category != models.FeatureEffectRedirectDamage || shield.RefreshType != "unlimited" {
		t.Errorf("I Am Your Shield = %+v", shield)
	}
}

func TestLoadDomainCardsRejectsUnknownCard(t *testing.T) {
	mappings := `{"mappings": {"Made Up": {"effects": [{"category": "FeatureEffectDamage"}]}}}`
	if _, err := LoadDomainCards([]byte(mappings), [][]byte{[]byte(`{"domain": "Blade", "cards": []}`)}); err == nil {
		t.Fatal("expected an error for a mapping that names no card")
	}
}
