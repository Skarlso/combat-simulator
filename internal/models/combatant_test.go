package models

import (
	"testing"
)

func newTestPlayer() *Combatant {
	return NewCombatant("p1", "Hero", SidePlayer, 3, 12, 2, 15, 8)
}

func newTestAdversary() *Combatant {
	return NewAdversary("e1", "Goblin", "Tier 1", "Bruiser", 10, 3, 10, 6, []int{5, 10})
}

func TestNewCombatant(t *testing.T) {
	c := newTestPlayer()

	if c.ID != "p1" {
		t.Errorf("ID = %q, want %q", c.ID, "p1")
	}
	if c.Side != SidePlayer {
		t.Errorf("Side = %v, want SidePlayer", c.Side)
	}
	if c.IsAdversary {
		t.Error("IsAdversary should be false for player")
	}
	if c.HP != 15 || c.HPMax != 15 {
		t.Errorf("HP = %d/%d, want 15/15", c.HP, c.HPMax)
	}
	if c.IsDefeated {
		t.Error("should not be defeated on creation")
	}
}

func TestNewAdversary(t *testing.T) {
	c := newTestAdversary()

	if c.Side != SideEnemy {
		t.Errorf("Side = %v, want SideEnemy", c.Side)
	}
	if !c.IsAdversary {
		t.Error("IsAdversary should be true")
	}
	if c.Difficulty != 10 {
		t.Errorf("Difficulty = %d, want 10", c.Difficulty)
	}
}

func TestGetDefense(t *testing.T) {
	player := newTestPlayer()
	if player.GetDefense() != 12 {
		t.Errorf("player GetDefense() = %d, want 12 (evasion)", player.GetDefense())
	}

	adv := newTestAdversary()
	if adv.GetDefense() != 10 {
		t.Errorf("adversary GetDefense() = %d, want 10 (difficulty)", adv.GetDefense())
	}
}

func TestGetAttackBonus(t *testing.T) {
	player := newTestPlayer()
	if player.GetAttackBonus() != 2 {
		t.Errorf("player GetAttackBonus() = %d, want 2 (proficiency)", player.GetAttackBonus())
	}

	adv := newTestAdversary()
	if adv.GetAttackBonus() != 3 {
		t.Errorf("adversary GetAttackBonus() = %d, want 3 (ATK)", adv.GetAttackBonus())
	}
}

func TestHeal(t *testing.T) {
	c := newTestPlayer()
	c.HP = 5

	c.Heal(3)
	if c.HP != 8 {
		t.Errorf("HP after heal = %d, want 8", c.HP)
	}

	c.Heal(100)
	if c.HP != c.HPMax {
		t.Errorf("HP after overheal = %d, want %d (max)", c.HP, c.HPMax)
	}
}

func TestAddStress(t *testing.T) {
	c := newTestPlayer()

	c.AddStress(3)
	if c.Stress != 3 {
		t.Errorf("Stress = %d, want 3", c.Stress)
	}

	c.AddStress(5)
	if c.Stress != c.StressMax {
		t.Errorf("Stress = %d, want %d (max)", c.Stress, c.StressMax)
	}
	if !c.IsFullyStressed {
		t.Error("should be fully stressed")
	}
}

func TestAddStressOverflow(t *testing.T) {
	c := newTestPlayer()
	c.Stress = c.StressMax

	hpBefore := c.HP
	c.AddStress(1)
	if c.HP != hpBefore-1 {
		t.Errorf("HP after stress overflow = %d, want %d", c.HP, hpBefore-1)
	}
}

func TestAddStressOverflowDefeats(t *testing.T) {
	c := newTestPlayer()
	c.HP = 1
	c.Stress = c.StressMax

	c.AddStress(1)
	if c.HP != 0 {
		t.Errorf("HP = %d, want 0", c.HP)
	}
	if !c.IsDefeated {
		t.Error("should be defeated from stress overflow at 1 HP")
	}
}

func TestClearStress(t *testing.T) {
	c := newTestPlayer()
	c.Stress = c.StressMax
	c.IsFullyStressed = true

	c.ClearStress(2)
	if c.IsFullyStressed {
		t.Error("should not be fully stressed after clearing")
	}

	c.ClearStress(100)
	if c.Stress != 0 {
		t.Errorf("Stress = %d, want 0 (can't go negative)", c.Stress)
	}
}

func TestConditions(t *testing.T) {
	c := newTestPlayer()

	cond := &ActiveCondition{
		Type:     ConditionBurning,
		Duration: DurationRounds,
	}
	c.AddCondition(cond)

	if !c.HasCondition(ConditionBurning) {
		t.Error("should have burning condition")
	}
	if c.HasCondition(ConditionPoisoned) {
		t.Error("should not have poisoned condition")
	}

	removed := c.RemoveCondition(ConditionBurning)
	if !removed {
		t.Error("RemoveCondition should return true")
	}
	if c.HasCondition(ConditionBurning) {
		t.Error("burning should be removed")
	}

	removed = c.RemoveCondition(ConditionBurning)
	if removed {
		t.Error("removing nonexistent condition should return false")
	}
}

func TestIsAlive(t *testing.T) {
	c := newTestPlayer()
	if !c.IsAlive() {
		t.Error("should be alive")
	}

	c.IsDefeated = true
	if c.IsAlive() {
		t.Error("should not be alive when defeated")
	}
}

func TestSpotlightTokens(t *testing.T) {
	c := newTestPlayer()
	c.AddSpotlightToken()
	c.AddSpotlightToken()
	if c.SpotlightTokens != 2 {
		t.Errorf("SpotlightTokens = %d, want 2", c.SpotlightTokens)
	}

	c.ResetSpotlightTokens()
	if c.SpotlightTokens != 0 {
		t.Errorf("SpotlightTokens after reset = %d, want 0", c.SpotlightTokens)
	}
}

func TestTakeDamageThresholds(t *testing.T) {
	tests := []struct {
		name         string
		damage       int
		thresholds   []int
		wantHP       int
		wantHPDamage int
	}{
		{"minor damage", 3, []int{5, 10}, 1, 9},
		{"major damage", 7, []int{5, 10}, 2, 8},
		{"severe damage", 12, []int{5, 10}, 3, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, tt.thresholds)
			result := c.TakeDamage(tt.damage, DamagePhysical, nil)
			if result.HPDamage != tt.wantHP {
				t.Errorf("HPDamage = %d, want %d", result.HPDamage, tt.wantHP)
			}
			if c.HP != tt.wantHPDamage {
				t.Errorf("HP = %d, want %d", c.HP, tt.wantHPDamage)
			}
		})
	}
}

func TestTakeDamageArmor(t *testing.T) {
	c := newTestAdversary()
	c.ArmorSlots = 2

	result := c.TakeDamage(12, DamagePhysical, nil)

	if !result.ArmorUsed {
		t.Error("armor should have been used")
	}
	if c.ArmorSlots != 1 {
		t.Errorf("ArmorSlots = %d, want 1", c.ArmorSlots)
	}
	if result.HPDamage != 2 {
		t.Errorf("HPDamage = %d, want 2 (severe reduced to major by armor)", result.HPDamage)
	}
}

func TestTakeDamageArmorMinor(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{5, 10})
	c.ArmorSlots = 1

	result := c.TakeDamage(3, DamagePhysical, nil)

	if result.HPDamage != 0 {
		t.Errorf("HPDamage = %d, want 0 (minor + armor = negated)", result.HPDamage)
	}
	if c.ArmorSlots != 0 {
		t.Errorf("ArmorSlots = %d, want 0", c.ArmorSlots)
	}
}

func TestTakeDamageImmunity(t *testing.T) {
	c := newTestAdversary()
	c.AddImmunity(DamagePhysical)

	result := c.TakeDamage(20, DamagePhysical, nil)
	if result.HPDamage != 0 {
		t.Errorf("HPDamage = %d, want 0 (immune)", result.HPDamage)
	}
	if c.HP != c.HPMax {
		t.Errorf("HP = %d, want %d (no damage from immunity)", c.HP, c.HPMax)
	}
}

func TestTakeDamageResistance(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{5, 10})
	c.AddResistance(DamagePhysical)

	result := c.TakeDamage(10, DamagePhysical, nil)
	if result.ProcessedDamage != 5 {
		t.Errorf("ProcessedDamage = %d, want 5 (50%% resistance)", result.ProcessedDamage)
	}
}

func TestTakeDamageVulnerability(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{5, 10})
	c.AddCondition(&ActiveCondition{Type: ConditionVulnerable, Duration: DurationUntilCleared})

	result := c.TakeDamage(10, DamagePhysical, nil)
	if result.ProcessedDamage != 15 {
		t.Errorf("ProcessedDamage = %d, want 15 (150%% vulnerability)", result.ProcessedDamage)
	}
}

func TestTakeDamageResistanceBeatsVulnerability(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{5, 10})
	c.AddResistance(DamagePhysical)
	c.AddCondition(&ActiveCondition{Type: ConditionVulnerable, Duration: DurationUntilCleared})

	result := c.TakeDamage(10, DamagePhysical, nil)
	if result.ProcessedDamage != 5 {
		t.Errorf("ProcessedDamage = %d, want 5 (resistance wins over vulnerability)", result.ProcessedDamage)
	}
}

func TestTakeDamageDefeats(t *testing.T) {
	c := newTestAdversary()
	c.HP = 1

	c.TakeDamage(20, DamagePhysical, nil)
	if !c.IsDefeated {
		t.Error("should be defeated")
	}
	if c.HP != 0 {
		t.Errorf("HP = %d, want 0", c.HP)
	}
}

func TestTakeDamageResultFields(t *testing.T) {
	c := newTestAdversary()
	c.ArmorSlots = 1

	result := c.TakeDamage(7, DamagePhysical, nil)

	if result.HPBefore != 10 {
		t.Errorf("HPBefore = %d, want 10", result.HPBefore)
	}
	if result.ArmorSlotsBefore != 1 {
		t.Errorf("ArmorSlotsBefore = %d, want 1", result.ArmorSlotsBefore)
	}
	if result.ArmorSlotsAfter != 0 {
		t.Errorf("ArmorSlotsAfter = %d, want 0", result.ArmorSlotsAfter)
	}
}

func TestAddResistanceAndImmunity(t *testing.T) {
	c := newTestPlayer()
	c.AddResistance(DamagePhysical)
	c.AddImmunity(DamageMagic)

	if len(c.Resistances) != 2 {
		t.Fatalf("Resistances len = %d, want 2", len(c.Resistances))
	}
	if c.Resistances[0].IsImmune {
		t.Error("first resistance should not be immunity")
	}
	if !c.Resistances[1].IsImmune {
		t.Error("second resistance should be immunity")
	}
}

func TestApplyPassiveResistances(t *testing.T) {
	c := newTestAdversary()
	c.AdversaryFeatures = []AdversaryFeature{
		{
			Name:    "Thick Hide",
			Passive: true,
			Effects: []FeatureEffect{
				{Category: FeatureEffectDamageResistance, Magnitude: 50, DamageType: "physical"},
			},
		},
		{
			Name:    "Ethereal",
			Passive: true,
			Effects: []FeatureEffect{
				{Category: FeatureEffectDamageResistance, Magnitude: 100, DamageType: "magic"},
			},
		},
	}

	c.ApplyPassiveResistances()

	if len(c.Resistances) != 2 {
		t.Fatalf("Resistances len = %d, want 2", len(c.Resistances))
	}
	if c.Resistances[0].IsImmune {
		t.Error("physical should be resistance, not immunity")
	}
	if !c.Resistances[1].IsImmune {
		t.Error("magic should be immunity (magnitude >= 100)")
	}
}

func TestApplyPassiveResistancesSkipsActive(t *testing.T) {
	c := newTestAdversary()
	c.AdversaryFeatures = []AdversaryFeature{
		{
			Name:    "Shield Bash",
			Passive: false,
			Effects: []FeatureEffect{
				{Category: FeatureEffectDamageResistance, Magnitude: 50, DamageType: "physical"},
			},
		},
	}

	c.ApplyPassiveResistances()

	if len(c.Resistances) != 0 {
		t.Errorf("Resistances len = %d, want 0 (active features should be skipped)", len(c.Resistances))
	}
}

func TestCloneCombatant(t *testing.T) {
	original := newTestAdversary()
	original.DamageDice = "2d6+3"
	original.ArmorSlots = 2
	original.HP = 5

	clone := CloneCombatant(original, "clone-1")

	if clone.ID != "clone-1" {
		t.Errorf("clone ID = %q, want %q", clone.ID, "clone-1")
	}
	if clone.Name != original.Name {
		t.Errorf("clone Name = %q, want %q", clone.Name, original.Name)
	}
	if clone.HP != original.HPMax {
		t.Errorf("clone HP = %d, want %d (full HP from template)", clone.HP, original.HPMax)
	}
	if clone.IsDefeated {
		t.Error("clone should not be defeated")
	}

	clone.CustomThresholds[0] = 999
	if original.CustomThresholds[0] == 999 {
		t.Error("modifying clone thresholds should not affect original")
	}
}

func TestDamageThresholdFromAmount(t *testing.T) {
	tests := []struct {
		name      string
		damage    int
		combatant *Combatant
		want      DamageThreshold
	}{
		{"adversary minor", 3, newTestAdversary(), ThresholdMinor},
		{"adversary major", 7, newTestAdversary(), ThresholdMajor},
		{"adversary severe", 12, newTestAdversary(), ThresholdSevere},
		{"player minor (level 3)", 4, newTestPlayer(), ThresholdMinor},
		{"player major (level 3)", 6, newTestPlayer(), ThresholdMajor},
		{"player severe (level 3)", 9, newTestPlayer(), ThresholdSevere},
		{"zero damage", 0, newTestAdversary(), ThresholdMinor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DamageThresholdFromAmount(tt.damage, tt.combatant)
			if got != tt.want {
				t.Errorf("DamageThresholdFromAmount(%d) = %v, want %v", tt.damage, got, tt.want)
			}
		})
	}
}

func TestParseDamageType(t *testing.T) {
	if parseDamageType("magic") != DamageMagic {
		t.Error("'magic' should return DamageMagic")
	}
	if parseDamageType("physical") != DamagePhysical {
		t.Error("'physical' should return DamagePhysical")
	}
	if parseDamageType("anything_else") != DamagePhysical {
		t.Error("unknown should default to DamagePhysical")
	}
}
