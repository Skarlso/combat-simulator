package data

import (
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/parser"
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
