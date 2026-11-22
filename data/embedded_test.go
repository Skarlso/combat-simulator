package data

import (
	"reflect"
	"testing"

	"github.com/skarlso/combat-simulator/internal/models"
	"github.com/skarlso/combat-simulator/internal/parser"
)

// Guards the shipped JSON: the parsers reject unknown enum strings at load time.
func TestEmbeddedDataLoads(t *testing.T) {
	registry, err := NewEmbeddedAdversaryRegistry()
	if err != nil {
		t.Fatalf("embedded data failed to load: %v", err)
	}
	if registry == nil {
		t.Fatal("registry is nil")
	}
}

// Momentum is a Reaction in every stat block; it must never be offered as a spotlight action.
func TestEmbeddedMomentumIsPassiveOnHit(t *testing.T) {
	registry, err := NewEmbeddedAdversaryRegistry()
	if err != nil {
		t.Fatalf("embedded data failed to load: %v", err)
	}
	bear, err := registry.CloneTemplate("Bear", "bear-1")
	if err != nil {
		t.Fatalf("clone Bear: %v", err)
	}
	for _, f := range bear.AdversaryFeatures {
		if f.Name != "Momentum" {
			continue
		}
		if !f.Passive || f.Trigger.Type != models.TriggerOnHit {
			t.Errorf("Momentum Passive=%v Trigger=%q, want passive on_hit", f.Passive, f.Trigger.Type)
		}
		return
	}
	t.Error("Bear has no Momentum feature")
}

// Stat blocks with "Severe: None" must not treat every hit as Severe.
func TestEmbeddedTinyOozeSurvivesOneDamage(t *testing.T) {
	registry, err := NewEmbeddedAdversaryRegistry()
	if err != nil {
		t.Fatalf("embedded data failed to load: %v", err)
	}
	ooze, err := registry.CloneTemplate("Tiny Red Ooze", "ooze-1")
	if err != nil {
		t.Fatalf("clone Tiny Red Ooze: %v", err)
	}
	ooze.TakeDamage(1, models.DamagePhysical, nil)
	if ooze.HP != ooze.HPMax-1 {
		t.Errorf("HP = %d, want %d (1 damage below Major marks 1 HP)", ooze.HP, ooze.HPMax-1)
	}
}

// Every mapped feature whose stat block says "make an attack" must roll against Evasion.
func TestEmbeddedAttackFeaturesRoll(t *testing.T) {
	raw, err := GetFeatureMappings()
	if err != nil {
		t.Fatal(err)
	}
	fp, err := parser.NewFeatureParserFromBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	attacks := []string{"Bite", "Hobbling Strike", "Hobbling Shot", "Detain", "Goading Strike",
		"Venomous Stinger", "Deadly Shot", "Double Strike", "Spit Acid", "Tormenting Lash"}
	for _, name := range attacks {
		f := fp.ParseFeature(name, "")
		if f == nil {
			t.Errorf("%s: not mapped", name)
			continue
		}
		if f.DCType != "target" {
			t.Errorf("%s: dc_type = %q, want target", name, f.DCType)
		}
	}
}

// SRD 2.0 ships 264 stat blocks; every one needs a standard attack and core stats.
func TestEmbeddedRegistryCoversSRD2(t *testing.T) {
	registry, err := NewEmbeddedAdversaryRegistry()
	if err != nil {
		t.Fatal(err)
	}
	templates := registry.Templates()
	if len(templates) != 264 {
		t.Fatalf("expected 264 adversaries, got %d", len(templates))
	}
	for _, c := range templates {
		if c.AttackDamageFormula() == "" || c.Difficulty == 0 || c.HPMax == 0 {
			t.Errorf("%s: incomplete stats (dmg %q, difficulty %d, hp %d)", c.Name, c.DamageDice, c.Difficulty, c.HPMax)
		}
	}
}

// SRD 2.0 has 210 domain cards (the 13 grimoires listed as their 33 spells) and 229 other
// player features; modelled ones must be runnable.
func TestEmbeddedAbilityLibrary(t *testing.T) {
	library, err := LoadAbilityLibrary()
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	modelled := 0
	for _, entry := range library {
		counts[entry.Source]++
		if entry.Description == "" || entry.Group == "" || entry.Ability.Name != entry.Name {
			t.Errorf("%s: description %q, group %q, ability name %q", entry.Name, entry.Description, entry.Group, entry.Ability.Name)
		}
		if entry.Modelled {
			modelled++
			if len(entry.Ability.Effects) == 0 {
				t.Errorf("%s: modelled with no effects", entry.Name)
			}
		}
	}
	want := map[string]int{
		parser.SourceDomainCard: 230, parser.SourceClass: 36, parser.SourceSubclass: 118,
		parser.SourceAncestry: 48, parser.SourceCommunity: 15, parser.SourceTransformation: 12,
	}
	for source, n := range want {
		if counts[source] != n {
			t.Errorf("%s: %d entries, want %d", source, counts[source], n)
		}
	}
	if modelled < 170 {
		t.Errorf("%d modelled abilities, want at least 170", modelled)
	}
}

// A party built from the library must survive Export JSON and Import JSON unchanged.
func TestLibraryAbilitiesRoundTripThroughCharacterJSON(t *testing.T) {
	library, err := LoadAbilityLibrary()
	if err != nil {
		t.Fatal(err)
	}
	pc := models.NewCombatant("pc1", "Hero", models.SidePlayer, 5, 11, 2, 8, 6)
	for _, entry := range library {
		if entry.Modelled {
			pc.DomainAbilities = append(pc.DomainAbilities, entry.AbilityAt(pc.Level))
		}
	}
	raw, err := parser.ExportCharacters([]*models.Combatant{pc})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := parser.LoadCharactersFromBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range pc.DomainAbilities {
		if got := loaded[0].DomainAbilities[i]; !reflect.DeepEqual(got, want) {
			t.Errorf("%s changed in the round trip:\n got %+v\nwant %+v", want.Name, got, want)
		}
	}
}

// Acid Bath is a reaction to Severe damage only: it needs the trigger filter from the mapping.
func TestEmbeddedAcidBathReactsToSevereDamage(t *testing.T) {
	registry, err := NewEmbeddedAdversaryRegistry()
	if err != nil {
		t.Fatal(err)
	}
	burrower, err := registry.CloneTemplate("Acid Burrower", "e1")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range burrower.AdversaryFeatures {
		if f.Name != "Acid Bath" {
			continue
		}
		if !f.Passive || f.Trigger.Type != models.TriggerOnDamageTaken || f.Trigger.Filter != "if_took_severe" {
			t.Errorf("Acid Bath: passive %v, trigger %q, filter %q; want a passive on_damage_taken / if_took_severe", f.Passive, f.Trigger.Type, f.Trigger.Filter)
		}
		return
	}
	t.Error("Acid Burrower has no Acid Bath feature")
}
