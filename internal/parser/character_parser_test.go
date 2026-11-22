package parser

import (
	"strings"
	"testing"

	"github.com/skarlso/combat-simulator/internal/models"
)

const testCharacterJSON = `{
	"characters": [
		{
			"name": "Kira",
			"level": 3,
			"evasion": 12,
			"proficiency": 2,
			"traits": {"finesse": 3},
			"weapon_trait": "finesse",
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

func TestLoadCharactersFromBytesUnknownEffectCategory(t *testing.T) {
	blob := `{"characters": [{"name": "Kira", "level": 1, "evasion": 10, "proficiency": 1, "hp": 6, "stress": 6,
		"domain_abilities": [{"name": "Mend", "effects": [{"category": "FeatureEffectHeal", "magnitude": 2}]}]}]}`
	_, err := LoadCharactersFromBytes([]byte(blob))
	if err == nil {
		t.Fatal("expected error for unknown effect category")
	}
	if !strings.Contains(err.Error(), "Mend") || !strings.Contains(err.Error(), "FeatureEffectHeal") {
		t.Errorf("error should name the ability and category, got %q", err.Error())
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

func TestLoadCharactersFromBytesWeaponTrait(t *testing.T) {
	combatants, err := LoadCharactersFromBytes([]byte(testCharacterJSON))
	if err != nil {
		t.Fatal(err)
	}
	if combatants[0].GetAttackBonus() != 3 {
		t.Errorf("GetAttackBonus = %d, want 3 (Finesse, the weapon trait)", combatants[0].GetAttackBonus())
	}
	if combatants[1].GetAttackBonus() != 0 {
		t.Errorf("GetAttackBonus = %d, want 0 when no weapon trait is named", combatants[1].GetAttackBonus())
	}
	if _, err := LoadCharactersFromBytes([]byte(`{"characters": [{"name": "Kira", "weapon_trait": "luck"}]}`)); err == nil {
		t.Error("an unknown weapon trait should be rejected")
	}
}

func TestLoadCharactersFromBytesTraits(t *testing.T) {
	combatants, err := LoadCharactersFromBytes([]byte(`{"characters": [
		{"name": "Kira", "traits": {"agility": 2, "knowledge": -1}}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	traits := combatants[0].Traits
	if traits.Modifier("agility") != 2 || traits.Modifier("knowledge") != -1 || traits.Modifier("strength") != 0 {
		t.Errorf("Traits = %+v, want agility 2, knowledge -1, the rest 0", traits)
	}
}

func TestCharacterImportPotionsAreCapped(t *testing.T) {
	blob := `{"characters":[{"name":"Kira","level":2,"hp":6,"stress":5,"evasion":11,"health_potions":9,"stamina_potions":2}]}`
	pcs, err := LoadCharactersFromBytes([]byte(blob))
	if err != nil {
		t.Fatal(err)
	}
	if pcs[0].HealthPotions != 5 || pcs[0].StaminaPotions != 2 {
		t.Errorf("potions = %d health, %d stamina; want 5 / 2", pcs[0].HealthPotions, pcs[0].StaminaPotions)
	}
}
