package parser

import (
	"testing"
)

const testAdversaryJSON = `{
	"adversaries": [
		{
			"name": "Goblin Raider",
			"tier": 1,
			"level_range": "1-2",
			"type": "Standard",
			"description": "A sneaky goblin",
			"motives_tactics": ["Attack weakest", "Flee when outnumbered"],
			"difficulty": 10,
			"thresholds": {"minor": 4, "major": 7},
			"hp": 8,
			"stress": 4,
			"attack": {
				"bonus": 3,
				"name": "Rusty Blade",
				"range": "melee",
				"damage": "1d6+1",
				"damage_type": "physical"
			},
			"features": [
				{
					"name": "Savage Strike",
					"type": "active",
					"description": "A brutal attack"
				},
				{
					"name": "Unmapped Feature",
					"type": "passive",
					"description": "Not in mappings"
				}
			]
		}
	]
}`

func TestLoadAdversariesFromBytes(t *testing.T) {
	fp, err := NewFeatureParserFromBytes([]byte(testMappingJSON))
	if err != nil {
		t.Fatalf("feature parser error: %v", err)
	}
	ap := &AdversaryParser{featureParser: fp}

	combatants, err := ap.LoadAdversariesFromBytes([]byte(testAdversaryJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(combatants) != 1 {
		t.Fatalf("combatants len = %d, want 1", len(combatants))
	}

	c := combatants[0]
	if c.Name != "Goblin Raider" {
		t.Errorf("Name = %q, want %q", c.Name, "Goblin Raider")
	}
	if !c.IsAdversary {
		t.Error("should be adversary")
	}
	if c.HP != 8 || c.HPMax != 8 {
		t.Errorf("HP = %d/%d, want 8/8", c.HP, c.HPMax)
	}
	if c.Difficulty != 10 {
		t.Errorf("Difficulty = %d, want 10", c.Difficulty)
	}
	if c.DamageDice != "1d6+1" {
		t.Errorf("DamageDice = %q, want %q", c.DamageDice, "1d6+1")
	}
}

func TestLoadAdversariesMappedFeatures(t *testing.T) {
	fp, err := NewFeatureParserFromBytes([]byte(testMappingJSON))
	if err != nil {
		t.Fatalf("feature parser error: %v", err)
	}
	ap := &AdversaryParser{featureParser: fp}

	combatants, err := ap.LoadAdversariesFromBytes([]byte(testAdversaryJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := combatants[0]

	if len(c.AdversaryFeatures) != 1 {
		t.Errorf("AdversaryFeatures len = %d, want 1 (unmapped should be skipped)", len(c.AdversaryFeatures))
	}
	if c.AdversaryFeatures[0].Name != "Savage Strike" {
		t.Errorf("Feature name = %q, want %q", c.AdversaryFeatures[0].Name, "Savage Strike")
	}
}

func TestLoadAdversariesInvalidJSON(t *testing.T) {
	fp, err := NewFeatureParserFromBytes([]byte(testMappingJSON))
	if err != nil {
		t.Fatalf("feature parser error: %v", err)
	}
	ap := &AdversaryParser{featureParser: fp}

	_, err = ap.LoadAdversariesFromBytes([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseAdversaryThresholds(t *testing.T) {
	fp, err := NewFeatureParserFromBytes([]byte(testMappingJSON))
	if err != nil {
		t.Fatalf("feature parser error: %v", err)
	}
	ap := &AdversaryParser{featureParser: fp}

	combatants, err := ap.LoadAdversariesFromBytes([]byte(testAdversaryJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := combatants[0]

	if len(c.CustomThresholds) != 2 {
		t.Fatalf("CustomThresholds len = %d, want 2", len(c.CustomThresholds))
	}
	if c.CustomThresholds[0] != 4 || c.CustomThresholds[1] != 7 {
		t.Errorf("CustomThresholds = %v, want [4, 7]", c.CustomThresholds)
	}
}
