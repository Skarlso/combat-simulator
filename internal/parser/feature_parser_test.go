package parser

import (
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
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
		{"unknown", models.FeatureEffectDamage},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stringToEffectCategory(tt.input)
			if got != tt.want {
				t.Errorf("stringToEffectCategory(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
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
		{"unknown", models.FeatureTargetSingle},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stringToTargetType(tt.input)
			if got != tt.want {
				t.Errorf("stringToTargetType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
