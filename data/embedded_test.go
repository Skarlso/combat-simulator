package data

import (
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
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
