package parser

import (
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

const testCharacterJSON = `{
	"characters": [
		{
			"name": "Kira",
			"level": 3,
			"evasion": 12,
			"proficiency": 2,
			"hp": 15,
			"stress": 8,
			"armor": 2,
			"damage_dice": "2d6+3",
			"thresholds": {"minor": 7, "major": 13},
			"domain_abilities": [
				{
					"name": "Blade Dance",
					"description": "A flourishing strike",
					"hope_cost": 2,
					"refresh_type": "rest",
					"max_uses": 1,
					"dc_type": "target",
					"effects": [
						{
							"category": "FeatureEffectDamage",
							"dice_formula": "2d8",
							"damage_type": "physical"
						}
					]
				}
			]
		},
		{
			"name": "Ruven",
			"level": 1,
			"evasion": 10,
			"proficiency": 1,
			"hp": 10,
			"stress": 6
		}
	]
}`

func TestLoadCharactersFromBytes(t *testing.T) {
	combatants, err := LoadCharactersFromBytes([]byte(testCharacterJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(combatants) != 2 {
		t.Fatalf("len = %d, want 2", len(combatants))
	}

	kira := combatants[0]
	if kira.Name != "Kira" {
		t.Errorf("name = %q, want Kira", kira.Name)
	}
	if kira.Side != models.SidePlayer {
		t.Errorf("side = %v, want SidePlayer", kira.Side)
	}
	if kira.HPMax != 15 || kira.StressMax != 8 {
		t.Errorf("HP/Stress = %d/%d, want 15/8", kira.HPMax, kira.StressMax)
	}
	if kira.ArmorSlots != 2 {
		t.Errorf("ArmorSlots = %d, want 2", kira.ArmorSlots)
	}
	if kira.DamageDice != "2d6+3" {
		t.Errorf("DamageDice = %q, want 2d6+3", kira.DamageDice)
	}
	if len(kira.CustomThresholds) != 2 || kira.CustomThresholds[0] != 7 || kira.CustomThresholds[1] != 13 {
		t.Errorf("CustomThresholds = %v, want [7 13]", kira.CustomThresholds)
	}
	if len(kira.DomainAbilities) != 1 {
		t.Fatalf("DomainAbilities len = %d, want 1", len(kira.DomainAbilities))
	}

	ability := kira.DomainAbilities[0]
	if ability.Name != "Blade Dance" {
		t.Errorf("ability name = %q, want Blade Dance", ability.Name)
	}
	if ability.HopeCost != 2 {
		t.Errorf("HopeCost = %d, want 2", ability.HopeCost)
	}
	if ability.MaxUses != 1 || ability.UsesLeft != 1 {
		t.Errorf("MaxUses/UsesLeft = %d/%d, want 1/1", ability.MaxUses, ability.UsesLeft)
	}
	if ability.DCType != "target" {
		t.Errorf("DCType = %q, want target", ability.DCType)
	}
	if len(ability.Effects) != 1 || ability.Effects[0].Category != models.FeatureEffectDamage {
		t.Errorf("ability.Effects = %v, want one damage effect", ability.Effects)
	}
	if ability.Effects[0].DiceFormula != "2d8" {
		t.Errorf("effect DiceFormula = %q, want 2d8", ability.Effects[0].DiceFormula)
	}
}

func TestLoadCharactersFromBytesMinimal(t *testing.T) {
	combatants, err := LoadCharactersFromBytes([]byte(testCharacterJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ruven := combatants[1]
	if ruven.Name != "Ruven" {
		t.Errorf("name = %q, want Ruven", ruven.Name)
	}
	if len(ruven.DomainAbilities) != 0 {
		t.Errorf("DomainAbilities len = %d, want 0", len(ruven.DomainAbilities))
	}
	if len(ruven.CustomThresholds) != 0 {
		t.Errorf("CustomThresholds len = %d, want 0 (omitted)", len(ruven.CustomThresholds))
	}
}

func TestLoadCharactersFromBytesInvalidJSON(t *testing.T) {
	_, err := LoadCharactersFromBytes([]byte("not json"))
	if err == nil {
		t.Error("expected error on invalid JSON")
	}
}

func TestLoadCharactersFromBytesMissingName(t *testing.T) {
	blob := `{"characters": [{"level": 3}]}`
	_, err := LoadCharactersFromBytes([]byte(blob))
	if err == nil {
		t.Error("expected error when name missing")
	}
}

func TestLoadCharactersFromBytesEmptyArray(t *testing.T) {
	blob := `{"characters": []}`
	combatants, err := LoadCharactersFromBytes([]byte(blob))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(combatants) != 0 {
		t.Errorf("len = %d, want 0", len(combatants))
	}
}
