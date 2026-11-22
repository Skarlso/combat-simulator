package parser

import (
	"testing"

	"github.com/skarlso/combat-simulator/internal/models"
)

func find(t *testing.T, library []LibraryAbility, name string) LibraryAbility {
	t.Helper()
	for _, entry := range library {
		if entry.Name == name {
			return entry
		}
	}
	t.Fatalf("%s is not in the library", name)
	return LibraryAbility{}
}

func TestLoadAbilityLibraryReadsEveryDataLayout(t *testing.T) {
	mappings := `{"mappings": {
		"Healing Hands": {"stress_cost": 1, "dc_type": "fixed", "fixed_dc": 13, "max_uses": 1, "refresh_type": "scene",
			"effects": [{"category": "FeatureEffectHPHeal", "magnitude": 2, "target_type": "ally"}]},
		"I Am Your Shield": {"passive": true, "trigger": "on_ally_damaged", "stress_cost": 1,
			"effects": [{"category": "FeatureEffectRedirectDamage"}]}}}`
	single := `{"domain": "Splendor", "cards": [{"name": "Healing Hands", "level": 2, "type": "Spell", "description": "Heal."}]}`
	nested := `{"domains": {"Valor": {"cards": [{"name": "I Am Your Shield", "level": 1, "type": "Ability", "description": "Shield."}]}}}`
	listed := `{"domains": [{"domain": "Blade", "cards": [{"name": "Reckless", "level": 2, "type": "Ability", "description": "Adv."}]}]}`
	grimoire := `{"domain": "Codex", "cards": [{"name": "Book of Ava", "level": 1, "type": "Grimoire",
		"spells": [{"name": "Power Push", "description": "Push."}, {"name": "Ice Spike", "description": "Spike."}]}]}`

	library, err := LoadAbilityLibrary([][]byte{[]byte(mappings)}, [][]byte{[]byte(single), []byte(nested), []byte(listed), []byte(grimoire)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range library {
		names = append(names, entry.Group+"/"+entry.Name)
	}
	want := []string{"Blade/Reckless", "Codex/Book of Ava: Ice Spike", "Codex/Book of Ava: Power Push", "Splendor/Healing Hands", "Valor/I Am Your Shield"}
	if len(names) != len(want) {
		t.Fatalf("library = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("library = %v, want %v", names, want)
		}
	}

	hands := find(t, library, "Healing Hands")
	if !hands.Modelled || hands.Source != SourceDomainCard || hands.Level != 2 || hands.Ability.FixedDC != 13 || hands.Ability.UsesLeft != 1 ||
		hands.Ability.Effects[0].TargetType != models.FeatureTargetAlly || hands.Ability.Description != "Heal." {
		t.Errorf("Healing Hands = %+v", hands)
	}
	shield := find(t, library, "I Am Your Shield").Ability
	if !shield.Passive || shield.Trigger.Type != models.TriggerOnAllyDamaged || shield.Effects[0].Category != models.FeatureEffectRedirectDamage || shield.RefreshType != "unlimited" {
		t.Errorf("I Am Your Shield = %+v", shield)
	}
	reckless := find(t, library, "Reckless")
	if reckless.Modelled || len(reckless.Ability.Effects) != 0 || reckless.Ability.Name != "Reckless" || reckless.Ability.Description != "Adv." {
		t.Errorf("unmapped card should be text only: %+v", reckless)
	}
	if spike := find(t, library, "Book of Ava: Ice Spike"); spike.Description != "Spike." || spike.Level != 1 || spike.Type != "Grimoire" {
		t.Errorf("grimoire spell = %+v", spike)
	}
}

func TestLoadAbilityLibraryReadsPlayerFeatures(t *testing.T) {
	mappings := `{"mappings": {
		"Unstoppable": {"passive": true, "trigger": "start_turn", "max_uses": 1, "uses_tokens": true, "token_name": "Die", "tokens_max": 4,
			"level_upgrades": [{"level": 5, "tokens_max": 6}],
			"effects": [{"category": "FeatureEffectDamageBonus", "magnitude": 1, "per_token": true},
				{"category": "FeatureEffectConditionImmunity", "conditions": ["restrained", "vulnerable"]}]},
		"Ribbet::Amphibious": {"passive": true, "effects": [{"category": "FeatureEffectDamageReduction", "magnitude": 1}]}}}`
	features := `{
		"classes": [{"name": "Guardian", "domains": ["Valor", "Blade"], "features": [{"name": "Unstoppable", "type": "Class Feature", "description": "Die."}],
			"subclasses": [{"name": "Stalwart", "features": [{"name": "Unwavering", "type": "Foundation", "description": "+1."}]}]}],
		"ancestries": [{"name": "Tidekin", "features": [{"name": "Amphibious", "description": "Swim."}]},
			{"name": "Ribbet", "features": [{"name": "Amphibious", "description": "Swim."}]}],
		"communities": [{"name": "Warborne", "features": [{"name": "Brave Face", "description": "Brave."}]}],
		"transformations": [{"name": "Vampire", "features": [{"name": "Fangs", "description": "Bite."}]}]}`

	cards := `{"domains": [{"domain": "Valor", "cards": [{"name": "Bare Bones", "level": 1, "type": "Ability", "description": "Bones."}]},
		{"domain": "Sage", "cards": [{"name": "Gifted Tracker", "level": 1, "type": "Ability", "description": "Track."}]}]}`

	library, err := LoadAbilityLibrary([][]byte{[]byte(mappings)}, [][]byte{[]byte(cards)}, [][]byte{[]byte(features)})
	if err != nil {
		t.Fatal(err)
	}
	if len(library) != 8 {
		t.Fatalf("library has %d entries, want 8", len(library))
	}
	if got := find(t, library, "Bare Bones").Classes; len(got) != 1 || got[0] != "Guardian" {
		t.Errorf("Bare Bones classes = %v, want Guardian through the Valor domain", got)
	}
	if got := find(t, library, "Gifted Tracker").Classes; len(got) != 0 {
		t.Errorf("Gifted Tracker classes = %v, want none", got)
	}

	unstoppable := find(t, library, "Unstoppable")
	if unstoppable.Source != SourceClass || unstoppable.Group != "Guardian" || unstoppable.Type != "Class Feature" || !unstoppable.Modelled {
		t.Errorf("Unstoppable = %+v", unstoppable)
	}
	immunity := unstoppable.Ability.Effects[1]
	if !unstoppable.Ability.Effects[0].PerToken || len(immunity.Conditions) != 2 || immunity.Conditions[1] != models.ConditionVulnerable {
		t.Errorf("Unstoppable effects = %+v", unstoppable.Ability.Effects)
	}
	if low, high := unstoppable.AbilityAt(4), unstoppable.AbilityAt(5); low.TokensMax != 4 || high.TokensMax != 6 {
		t.Errorf("die size at level 4 = %d, at level 5 = %d, want 4 and 6", low.TokensMax, high.TokensMax)
	}

	unwavering := find(t, library, "Unwavering")
	if unwavering.Source != SourceSubclass || unwavering.Group != "Stalwart" || unwavering.Classes[0] != "Guardian" || unwavering.Modelled {
		t.Errorf("Unwavering = %+v", unwavering)
	}
	for _, entry := range library {
		if entry.Name == "Amphibious" && entry.Modelled != (entry.Group == "Ribbet") {
			t.Errorf("%s Amphibious modelled = %v: a Group::Name mapping must only reach its group", entry.Group, entry.Modelled)
		}
	}
	if fangs := find(t, library, "Fangs"); fangs.Source != SourceTransformation || fangs.Type != "Transformation Feature" {
		t.Errorf("Fangs = %+v", fangs)
	}
}

func TestLoadAbilityLibraryRejectsBadMappings(t *testing.T) {
	blade := [][]byte{[]byte(`{"domain": "Blade", "cards": [{"name": "Reckless", "level": 2}]}`)}
	for name, mappings := range map[string]string{
		"unknown ability":   `{"mappings": {"Made Up": {"effects": [{"category": "FeatureEffectDamage"}]}}}`,
		"unknown condition": `{"mappings": {"Reckless": {"effects": [{"category": "FeatureEffectConditionImmunity", "conditions": ["sleepy"]}]}}}`,
		"unknown trigger":   `{"mappings": {"Reckless": {"passive": true, "trigger": "on_atack", "effects": [{"category": "FeatureEffectApplyAdvantage"}]}}}`,
		"unknown filter":    `{"mappings": {"Reckless": {"passive": true, "trigger_filter": "if_sleepy", "effects": [{"category": "FeatureEffectDamageBonus"}]}}}`,
		"unknown scale":     `{"mappings": {"Reckless": {"passive": true, "effects": [{"category": "FeatureEffectDamageBonus", "scale_by": "luck"}]}}}`,
	} {
		if _, err := LoadAbilityLibrary([][]byte{[]byte(mappings)}, blade, nil); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
