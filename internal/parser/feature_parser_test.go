package parser

import (
	"os"
	"strings"
	"testing"

	"github.com/skarlso/combat-simulator/internal/models"
)

const testMappingJSON = `{
	"mappings": {
		"Savage Strike": {
			"passive": false,
			"fear_cost": 1,
			"stress_cost": 0,
			"effects": [
				{
					"category": "FeatureEffectDamage",
					"dice_formula": "2d6",
					"damage_type": "physical",
					"target_type": "single"
				}
			]
		},
		"Thick Hide": {
			"passive": true,
			"fear_cost": 0,
			"stress_cost": 0,
			"effects": [
				{
					"category": "FeatureEffectDamageResistance",
					"magnitude": 50,
					"damage_type": "physical",
					"target_type": "self"
				}
			]
		}
	},
	"metadata": {
		"version": "1.0",
		"mapped_features": 2,
		"total_features": 2,
		"coverage": "100%",
		"last_updated": "2026-01-01"
	}
}`

func TestNewFeatureParserFromBytes(t *testing.T) {
	fp, err := NewFeatureParserFromBytes([]byte(testMappingJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp == nil {
		t.Fatal("parser should not be nil")
	}
	if len(fp.mappings) != 2 {
		t.Errorf("mappings len = %d, want 2", len(fp.mappings))
	}
}

func TestNewFeatureParserFromBytesInvalid(t *testing.T) {
	_, err := NewFeatureParserFromBytes([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseFeatureMapped(t *testing.T) {
	fp, err := NewFeatureParserFromBytes([]byte(testMappingJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	feature := fp.ParseFeature("Savage Strike", "A brutal attack")
	if feature == nil {
		t.Fatal("feature should not be nil for mapped name")
	}
	if feature.Name != "Savage Strike" {
		t.Errorf("Name = %q, want %q", feature.Name, "Savage Strike")
	}
	if feature.Passive {
		t.Error("should not be passive")
	}
	if feature.FearCost != 1 {
		t.Errorf("FearCost = %d, want 1", feature.FearCost)
	}
	if len(feature.Effects) != 1 {
		t.Fatalf("Effects len = %d, want 1", len(feature.Effects))
	}
	if feature.Effects[0].Category != models.FeatureEffectDamage {
		t.Errorf("Category = %q, want %q", feature.Effects[0].Category, models.FeatureEffectDamage)
	}
}

func TestParseFeaturePassive(t *testing.T) {
	fp, err := NewFeatureParserFromBytes([]byte(testMappingJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	feature := fp.ParseFeature("Thick Hide", "Tough skin")
	if feature == nil {
		t.Fatal("feature should not be nil")
	}
	if !feature.Passive {
		t.Error("should be passive")
	}
	if feature.Effects[0].Category != models.FeatureEffectDamageResistance {
		t.Errorf("Category = %q, want damage_resistance", feature.Effects[0].Category)
	}
	if feature.Effects[0].Magnitude != 50 {
		t.Errorf("Magnitude = %d, want 50", feature.Effects[0].Magnitude)
	}
}

func TestFeatureNamesReturnsAllMappedKeys(t *testing.T) {
	fp, err := NewFeatureParserFromBytes([]byte(testMappingJSON))
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	names := fp.FeatureNames()
	if len(names) != 2 {
		t.Errorf("FeatureNames len = %d, want 2", len(names))
	}

	got := map[string]bool{}
	for _, n := range names {
		got[n] = true
	}
	if !got["Savage Strike"] || !got["Thick Hide"] {
		t.Errorf("FeatureNames = %v, want to include Savage Strike and Thick Hide", names)
	}
}

func TestParseFeatureUnmapped(t *testing.T) {
	fp, err := NewFeatureParserFromBytes([]byte(testMappingJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	feature := fp.ParseFeature("Unknown Ability", "mystery")
	if feature != nil {
		t.Error("unmapped feature should return nil")
	}
}

func TestStringToEffectCategory(t *testing.T) {
	tests := []struct {
		input string
		want  models.FeatureEffectCategory
	}{
		{"FeatureEffectDamage", models.FeatureEffectDamage},
		{"FeatureEffectDirectDamage", models.FeatureEffectDirectDamage},
		{"FeatureEffectHPHeal", models.FeatureEffectHPHeal},
		{"FeatureEffectSummonAllies", models.FeatureEffectSummonAllies},
		{"FeatureEffectDamageResistance", models.FeatureEffectDamageResistance},
		{"FeatureEffectTokenPlace", models.FeatureEffectTokenPlace},
		{"FeatureEffectTokenSpend", models.FeatureEffectTokenSpend},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := stringToEffectCategory(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("stringToEffectCategory(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}

	for _, input := range []string{"unknown", "", "FeatureEffectDamageAmplifier"} {
		if _, err := stringToEffectCategory(input); err == nil {
			t.Errorf("stringToEffectCategory(%q) should fail", input)
		}
	}
}

func TestNewFeatureParserRejectsUnknownEffectCategory(t *testing.T) {
	blob := `{"mappings": {"Backstab": {"effects": [{"category": "FeatureEffectDamageAmplifier", "dice_formula": "2d6"}]}}}`
	_, err := NewFeatureParserFromBytes([]byte(blob))
	if err == nil {
		t.Fatal("expected error for unknown effect category")
	}
	if !strings.Contains(err.Error(), "Backstab") || !strings.Contains(err.Error(), "FeatureEffectDamageAmplifier") {
		t.Errorf("error should name the feature and category, got %q", err.Error())
	}
}

func TestParseFeatureCostOnSuccess(t *testing.T) {
	blob := `{"mappings": {"Detain": {"stress_cost": 1, "cost_on_success": true, "dc_type": "target",
		"effects": [{"category": "FeatureEffectApplyRestrained", "duration": "until_cleared"}]}}}`
	fp, err := NewFeatureParserFromBytes([]byte(blob))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f := fp.ParseFeature("Detain", "")
	if f == nil || !f.CostOnSuccess || f.DCType != "target" || f.StressCost != 1 {
		t.Errorf("ParseFeature = %+v, want CostOnSuccess, dc target, stress 1", f)
	}
}

func TestNewFeatureParserRejectsUnknownTargetType(t *testing.T) {
	blob := `{"mappings": {"Bite": {"effects": [{"category": "FeatureEffectDamage", "target_type": "everyone"}]}}}`
	if _, err := NewFeatureParserFromBytes([]byte(blob)); err == nil {
		t.Fatal("expected error for unknown target type")
	}
}

func TestStringToTargetType(t *testing.T) {
	tests := []struct {
		input string
		want  models.FeatureTargetType
	}{
		{"self", models.FeatureTargetSelf},
		{"single", models.FeatureTargetSingle},
		{"aoe", models.FeatureTargetAOE},
		{"allies", models.FeatureTargetAllies},
		{"", models.FeatureTargetSingle},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := stringToTargetType(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("stringToTargetType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFeatureTriggerAndArmorCost(t *testing.T) {
	fp, err := NewFeatureParserFromBytes([]byte(`{"mappings": {
		"Acidic Form": {"passive": true, "trigger": "on_hit", "effects": [{"category": "FeatureEffectArmorDamage", "magnitude": 1}]},
		"Tormenting Lash": {"stress_cost": 1, "effects": [{"category": "FeatureEffectAreaDamage", "dice_formula": "2d8", "armor_cost": 2}]}
	}}`))
	if err != nil {
		t.Fatal(err)
	}

	acid := fp.ParseFeature("Acidic Form", "")
	if !acid.Passive || acid.Trigger.Type != models.TriggerOnHit {
		t.Errorf("Acidic Form passive=%v trigger=%q, want true / on_hit", acid.Passive, acid.Trigger.Type)
	}

	lash := fp.ParseFeature("Tormenting Lash", "")
	if lash.Effects[0].ArmorCost != 2 {
		t.Errorf("ArmorCost = %d, want 2", lash.Effects[0].ArmorCost)
	}
}

func TestEmbeddedArmorFeatureMappings(t *testing.T) {
	data, err := os.ReadFile("../../data/feature_mappings.json")
	if err != nil {
		t.Fatal(err)
	}
	fp, err := NewFeatureParserFromBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	acid := fp.ParseFeature("Spit Acid", "")
	if acid.DCType != "target" {
		t.Errorf("Spit Acid DCType = %q, want target (it is an attack)", acid.DCType)
	}
	shred := acid.Effects[0]
	if shred.Category != models.FeatureEffectArmorDamage || shred.TargetType != models.FeatureTargetAOE || shred.FallbackFear != 1 {
		t.Errorf("Spit Acid shred effect = %+v", shred)
	}

	lash := fp.ParseFeature("Tormenting Lash", "")
	if lash.StressCost != 1 || lash.Effects[0].ArmorCost != 2 || !lash.Effects[0].StandardAttack {
		t.Errorf("Tormenting Lash stress=%d effect=%+v, want stress 1, armor cost 2, standard attack damage", lash.StressCost, lash.Effects[0])
	}

	hellfire := fp.ParseFeature("Hellfire", "").Effects[0]
	if hellfire.Category != models.FeatureEffectAreaDamage || hellfire.ReactionTrait != "agility" || !hellfire.HalfOnSuccess {
		t.Errorf("Hellfire effect = %+v, want area damage halved by an Agility reaction roll", hellfire)
	}

	shards := fp.ParseFeature("Armor-Shredding Shards", "")
	if !shards.Passive || shards.Trigger.Type != models.TriggerOnHitTaken || shards.FearCost != 0 {
		t.Errorf("Armor-Shredding Shards = passive %v, trigger %q, fear %d", shards.Passive, shards.Trigger.Type, shards.FearCost)
	}
}

// The same feature name carries different numbers on different stat blocks.
func TestParseFeatureForPrefersAdversaryKey(t *testing.T) {
	raw := []byte(`{"mappings": {
		"Bite": {"category": "damage", "effects": [{"category": "FeatureEffectDamage", "dice_formula": "1d6", "damage_type": "physical"}]},
		"Dire Wolf::Bite": {"category": "damage", "max_uses": 2, "effects": [{"category": "FeatureEffectDamage", "dice_formula": "3d6", "damage_type": "physical"}]}
	}}`)
	fp, err := NewFeatureParserFromBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	generic := fp.ParseFeatureFor("Bear", "Bite", "")
	specific := fp.ParseFeatureFor("Dire Wolf", "Bite", "")
	if generic.Effects[0].DiceFormula != "1d6" || specific.Effects[0].DiceFormula != "3d6" {
		t.Fatalf("generic %q, specific %q", generic.Effects[0].DiceFormula, specific.Effects[0].DiceFormula)
	}
	if specific.MaxUses != 2 || specific.UsesLeft != 2 {
		t.Fatalf("max_uses not applied: %d/%d", specific.UsesLeft, specific.MaxUses)
	}
}

func TestParseFeatureSpotlightFields(t *testing.T) {
	blob := `{"mappings": {
		"Dance of Death": {"stress_cost": 1, "effects": [{"category": "FeatureEffectAllySpotlight", "dice_formula": "1d4", "half_damage": true}]},
		"Call of the Guahala": {"fear_cost": 2, "effects": [{"category": "FeatureEffectAllySpotlight", "spotlight_all": true, "ally_name": "Guahalan Shifter"}]},
		"Release the Hounds": {"fear_cost": 1, "effects": [{"category": "FeatureEffectSummonAllies", "summon_type": "Demonic Hound Pack", "magnitude": 2, "spotlight_count": 2}]}}}`
	fp, err := NewFeatureParserFromBytes([]byte(blob))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dance := fp.ParseFeature("Dance of Death", "").Effects[0]
	if dance.DiceFormula != "1d4" || !dance.HalfDamage || dance.SpotlightAll {
		t.Errorf("Dance of Death effect = %+v", dance)
	}
	call := fp.ParseFeature("Call of the Guahala", "").Effects[0]
	if !call.SpotlightAll || call.AllyName != "Guahalan Shifter" {
		t.Errorf("Call of the Guahala effect = %+v", call)
	}
	hounds := fp.ParseFeature("Release the Hounds", "").Effects[0]
	if hounds.SpotlightCount != 2 || hounds.SummonType != "Demonic Hound Pack" {
		t.Errorf("Release the Hounds effect = %+v", hounds)
	}
}
