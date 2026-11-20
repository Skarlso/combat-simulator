package parser

import (
	"reflect"
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func TestExportedPartyImportsBack(t *testing.T) {
	pc := models.NewCombatant("p1", "Kira", models.SidePlayer, 4, 12, 2, 7, 6)
	pc.AttackTrait = 3
	pc.Traits = models.Traits{Agility: 1, Presence: 2}
	pc.ArmorSlots = 3
	pc.DamageDice = "1d8+2"
	pc.HealthPotions, pc.StaminaPotions = 2, 1
	pc.CustomThresholds = []int{8, 15}
	pc.DomainAbilities = []models.DomainAbility{{
		Name: "Healing Strike", Description: "Heal on hit", Passive: true,
		Trigger:     models.FeatureTriggerCondition{Type: models.TriggerOnHit, Filter: "if_target_below_50_hp"},
		HopeCost:    2,
		RefreshType: "unlimited",
		Effects:     []models.FeatureEffect{{Category: models.FeatureEffectHPHeal, Magnitude: 1, TargetType: models.FeatureTargetAlly}},
	}, {
		Name: "Bolt Beacon", HopeCost: 1, DCType: "target", RefreshType: "unlimited",
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectDamage, DiceFormula: "1d8+2", DamageType: "magic", ProficiencyDice: true, TargetType: models.FeatureTargetSingle},
			{Category: models.FeatureEffectApplyVulnerable, Duration: "until_cleared", TargetType: models.FeatureTargetSingle},
		},
	}}

	raw, err := ExportCharacters([]*models.Combatant{pc})
	if err != nil {
		t.Fatal(err)
	}
	back, err := LoadCharactersFromBytes(raw)
	if err != nil {
		t.Fatalf("%v\n%s", err, raw)
	}
	got := back[0]

	if got.Name != pc.Name || got.Level != pc.Level || got.Evasion != pc.Evasion || got.Proficiency != pc.Proficiency ||
		got.AttackTrait != pc.AttackTrait || got.Traits != pc.Traits || got.HPMax != pc.HPMax || got.StressMax != pc.StressMax ||
		got.ArmorSlots != pc.ArmorSlots || got.DamageDice != pc.DamageDice || got.HealthPotions != 2 || got.StaminaPotions != 1 ||
		!reflect.DeepEqual(got.CustomThresholds, pc.CustomThresholds) {
		t.Errorf("stats changed in the round trip:\n%+v\n%+v", got, pc)
	}
	if !reflect.DeepEqual(got.DomainAbilities, pc.DomainAbilities) {
		t.Errorf("abilities changed in the round trip:\n%+v\n%+v", got.DomainAbilities, pc.DomainAbilities)
	}
}
