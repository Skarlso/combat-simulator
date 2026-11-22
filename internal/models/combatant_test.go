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
	player.Traits = Traits{Strength: 1, Knowledge: 3, Presence: -1}
	player.WeaponTrait, player.SpellcastTrait = "strength", "knowledge"
	if player.GetAttackBonus() != 1 {
		t.Errorf("player GetAttackBonus() = %d, want 1 (the weapon trait, not proficiency)", player.GetAttackBonus())
	}
	if got := player.RollModifier("spellcast"); got != 3 || player.RollLabel("spellcast") != "Spellcast (Knowledge)" {
		t.Errorf("spellcast roll = %+d %q, want +3 Spellcast (Knowledge)", got, player.RollLabel("spellcast"))
	}
	if got := player.RollModifier("presence"); got != -1 || player.RollLabel("presence") != "Presence" {
		t.Errorf("presence roll = %+d %q, want -1 Presence", got, player.RollLabel("presence"))
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

func TestTakeDirectDamageIgnoresArmor(t *testing.T) {
	c := newTestAdversary()
	c.ArmorSlots = 2

	result := c.TakeDirectDamage(12, DamagePhysical, nil)

	if result.ArmorUsed || c.ArmorSlots != 2 {
		t.Errorf("ArmorSlots = %d, want 2 (direct damage bypasses armor)", c.ArmorSlots)
	}
	if result.HPDamage != 3 {
		t.Errorf("HPDamage = %d, want 3 (severe)", result.HPDamage)
	}
}

func TestClearableCondition(t *testing.T) {
	c := newTestPlayer()
	if c.ClearableCondition() != nil {
		t.Error("no conditions should mean nothing to clear")
	}

	c.AddStress(c.StressMax)
	if c.ClearableCondition() != nil {
		t.Error("Vulnerable from full Stress should not be clearable by an action")
	}

	c.AddCondition(&ActiveCondition{Type: ConditionBurning, Duration: DurationUntilCleared})
	if cond := c.ClearableCondition(); cond == nil || cond.Type != ConditionBurning {
		t.Errorf("ClearableCondition = %v, want burning", cond)
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

func TestTakeDamageResistanceRoundsUp(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{4, 7})
	c.AddResistance(DamagePhysical)

	result := c.TakeDamage(7, DamagePhysical, nil)
	if result.ProcessedDamage != 4 || result.Threshold != ThresholdMajor {
		t.Errorf("7 resisted = %d (%v), want 4 Major (SRD rounds up)", result.ProcessedDamage, result.Threshold)
	}
}

func TestTakeDamageResistanceBeforeFlatReduction(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{5, 10})
	c.AddResistance(DamagePhysical)
	c.DomainAbilities = []DomainAbility{{
		Name: "Ward", Passive: true,
		Effects: []FeatureEffect{{Category: FeatureEffectDamageReduction, Magnitude: 2}},
	}}

	// SRD: halve first (12 -> 6), then other reductions (6 - 2 = 4), not (12 - 2) / 2 = 5
	result := c.TakeDamage(12, DamagePhysical, &mockRoller{})
	if result.ProcessedDamage != 4 || result.Threshold != ThresholdMinor {
		t.Errorf("ProcessedDamage = %d (%v), want 4 Minor", result.ProcessedDamage, result.Threshold)
	}
}

func TestTakeDamageVulnerableNoLongerAffectsDamage(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{5, 10})
	c.AddCondition(&ActiveCondition{Type: ConditionVulnerable, Duration: DurationUntilCleared})

	result := c.TakeDamage(10, DamagePhysical, nil)
	if result.ProcessedDamage != 10 {
		t.Errorf("ProcessedDamage = %d, want 10 (Vulnerable is accuracy bonus, not damage)", result.ProcessedDamage)
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

func withThresholds(major, severe int) *Combatant {
	c := newTestAdversary()
	c.CustomThresholds = []int{major, severe}
	return c
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
		{"zero damage", 0, newTestAdversary(), ThresholdNone},
		{"severe None caps at major", 40, withThresholds(5, 0), ThresholdMajor},
		{"severe None still minor below major", 4, withThresholds(5, 0), ThresholdMinor},
		{"both None is always minor", 40, withThresholds(0, 0), ThresholdMinor},
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

func TestTakeDamageZeroDamageNoHPMarked(t *testing.T) {
	c := newTestAdversary()
	hpBefore := c.HP
	result := c.TakeDamage(0, DamagePhysical, nil)
	if result.HPDamage != 0 {
		t.Errorf("HPDamage = %d, want 0 for zero damage", result.HPDamage)
	}
	if c.HP != hpBefore {
		t.Errorf("HP = %d, want %d (unchanged)", c.HP, hpBefore)
	}
	if result.Threshold != ThresholdNone {
		t.Errorf("Threshold = %v, want ThresholdNone", result.Threshold)
	}
}

func TestTakeDamageZeroDamageNoArmorUsed(t *testing.T) {
	c := newTestAdversary()
	c.ArmorSlots = 2
	result := c.TakeDamage(0, DamagePhysical, nil)
	if result.ArmorUsed {
		t.Error("armor should not be used for zero damage")
	}
	if c.ArmorSlots != 2 {
		t.Errorf("ArmorSlots = %d, want 2 (unchanged)", c.ArmorSlots)
	}
}

func TestAddConditionDedup(t *testing.T) {
	c := newTestPlayer()
	c.AddCondition(&ActiveCondition{Type: ConditionVulnerable, Duration: DurationUntilCleared})
	c.AddCondition(&ActiveCondition{Type: ConditionVulnerable, Duration: DurationUntilCleared})
	count := 0
	for _, cond := range c.Conditions {
		if cond.Type == ConditionVulnerable {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Vulnerable count = %d, want 1 (no duplicates)", count)
	}
}

func TestAddStressFullAppliesVulnerable(t *testing.T) {
	c := newTestPlayer()
	c.AddStress(c.StressMax)
	if !c.IsFullyStressed {
		t.Fatal("should be fully stressed")
	}
	if !c.HasCondition(ConditionVulnerable) {
		t.Error("fully stressed should apply Vulnerable condition")
	}
}

func TestAddStressFullVulnerableNoDuplicate(t *testing.T) {
	c := newTestPlayer()
	c.AddCondition(&ActiveCondition{Type: ConditionVulnerable, Duration: DurationUntilCleared})
	c.AddStress(c.StressMax)
	count := 0
	for _, cond := range c.Conditions {
		if cond.Type == ConditionVulnerable {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Vulnerable count = %d, want 1 (dedup)", count)
	}
}

func TestClearStressRemovesVulnerable(t *testing.T) {
	c := newTestPlayer()
	c.AddStress(c.StressMax)
	if !c.HasCondition(ConditionVulnerable) {
		t.Fatal("should have Vulnerable after full stress")
	}

	c.ClearStress(1)
	if c.HasCondition(ConditionVulnerable) {
		t.Error("Vulnerable should be removed when stress drops below max")
	}
	if c.IsFullyStressed {
		t.Error("should no longer be fully stressed")
	}
}

func TestAddStressOverflowMultiple(t *testing.T) {
	c := newTestPlayer()
	c.Stress = c.StressMax

	hpBefore := c.HP
	c.AddStress(5)
	if c.HP != hpBefore-1 {
		t.Errorf("HP after stress overflow of 5 = %d, want %d (1 HP whatever the excess)", c.HP, hpBefore-1)
	}
}

func TestAddStressPartialOverflow(t *testing.T) {
	c := newTestPlayer()
	c.StressMax = 6
	c.Stress = 4

	hpBefore := c.HP
	c.AddStress(5) // 4+5=9, max=6: 2 marked, the rest costs 1 HP
	if c.Stress != c.StressMax {
		t.Errorf("Stress = %d, want %d", c.Stress, c.StressMax)
	}
	if c.HP != hpBefore-1 {
		t.Errorf("HP after partial overflow = %d, want %d", c.HP, hpBefore-1)
	}
	if !c.IsFullyStressed {
		t.Error("should be fully stressed")
	}
	if !c.HasCondition(ConditionVulnerable) {
		t.Error("should have Vulnerable condition")
	}
}

func TestAddStressOverflowDefeatsMultiple(t *testing.T) {
	c := newTestPlayer()
	c.HP = 3
	c.Stress = c.StressMax

	c.AddStress(5)
	if c.HP != 2 {
		t.Errorf("HP = %d, want 2", c.HP)
	}
	if c.IsDefeated {
		t.Error("stress overflow only ever costs 1 HP")
	}
}

func TestCloneCombatantAppliesPassiveResistances(t *testing.T) {
	template := newTestAdversary()
	template.AdversaryFeatures = []AdversaryFeature{
		{
			Name:    "Thick Hide",
			Passive: true,
			Effects: []FeatureEffect{
				{Category: FeatureEffectDamageResistance, Magnitude: 50, DamageType: "physical"},
			},
		},
	}

	clone := CloneCombatant(template, "clone-1")
	if len(clone.Resistances) != 1 {
		t.Fatalf("clone Resistances len = %d, want 1", len(clone.Resistances))
	}
	if clone.Resistances[0].Type != DamagePhysical {
		t.Errorf("clone resistance type = %v, want DamagePhysical", clone.Resistances[0].Type)
	}
}

func TestTakeDamagePassiveReductionOnDomainAbility(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{5, 10})
	c.DomainAbilities = []DomainAbility{
		{
			Name:    "Ward",
			Passive: true,
			Effects: []FeatureEffect{
				{Category: FeatureEffectDamageReduction, Magnitude: 3},
			},
		},
	}

	result := c.TakeDamage(10, DamagePhysical, &mockRoller{})
	if result.ProcessedDamage != 7 {
		t.Errorf("ProcessedDamage = %d, want 7 (10 - 3 reduction)", result.ProcessedDamage)
	}
	if result.Threshold != ThresholdMajor {
		t.Errorf("Threshold = %v, want ThresholdMajor (reduced from Severe)", result.Threshold)
	}
	if result.HPDamage != 2 {
		t.Errorf("HPDamage = %d, want 2", result.HPDamage)
	}
}

func TestTakeDamageNonPassiveAbilityIgnored(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{5, 10})
	c.DomainAbilities = []DomainAbility{
		{
			Name:    "Active Shield",
			Passive: false,
			Effects: []FeatureEffect{
				{Category: FeatureEffectDamageReduction, Magnitude: 5},
			},
		},
	}

	result := c.TakeDamage(10, DamagePhysical, &mockRoller{})
	if result.ProcessedDamage != 10 {
		t.Errorf("ProcessedDamage = %d, want 10 (non-passive should not auto-reduce)", result.ProcessedDamage)
	}
	if result.HPDamage != 3 {
		t.Errorf("HPDamage = %d, want 3", result.HPDamage)
	}
}

func TestTakeDamagePassiveReductionOnAdversaryFeature(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{5, 10})
	c.AdversaryFeatures = []AdversaryFeature{
		{
			Name:    "Thick Plates",
			Passive: true,
			Effects: []FeatureEffect{
				{Category: FeatureEffectDamageReduction, Magnitude: 4},
			},
		},
	}

	result := c.TakeDamage(10, DamagePhysical, &mockRoller{})
	if result.ProcessedDamage != 6 {
		t.Errorf("ProcessedDamage = %d, want 6 (10 - 4 reduction)", result.ProcessedDamage)
	}
}

func TestTakeDamageNonPassiveFeatureIgnored(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{5, 10})
	c.AdversaryFeatures = []AdversaryFeature{
		{
			Name:    "Deflect",
			Passive: false,
			Effects: []FeatureEffect{
				{Category: FeatureEffectDamageReduction, Magnitude: 5},
			},
		},
	}

	result := c.TakeDamage(10, DamagePhysical, &mockRoller{})
	if result.ProcessedDamage != 10 {
		t.Errorf("ProcessedDamage = %d, want 10 (non-passive feature should not auto-reduce)", result.ProcessedDamage)
	}
}

func TestTakeDamageReductionStacks(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{5, 10})
	c.DomainAbilities = []DomainAbility{
		{Name: "Ward A", Passive: true, Effects: []FeatureEffect{{Category: FeatureEffectDamageReduction, Magnitude: 2}}},
	}
	c.AdversaryFeatures = []AdversaryFeature{
		{Name: "Plate", Passive: true, Effects: []FeatureEffect{{Category: FeatureEffectDamageReduction, Magnitude: 3}}},
	}

	result := c.TakeDamage(10, DamagePhysical, &mockRoller{})
	if result.ProcessedDamage != 5 {
		t.Errorf("ProcessedDamage = %d, want 5 (10 - 2 - 3)", result.ProcessedDamage)
	}
}

func TestTakeDamageReductionDiceFormula(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{5, 10})
	c.AdversaryFeatures = []AdversaryFeature{
		{
			Name:    "Runic Ward",
			Passive: true,
			Effects: []FeatureEffect{
				{Category: FeatureEffectDamageReduction, Magnitude: 1, DiceFormula: "1d4"},
			},
		},
	}

	result := c.TakeDamage(10, DamagePhysical, &mockRoller{fixedRoll: 2})
	if result.ProcessedDamage != 7 {
		t.Errorf("ProcessedDamage = %d, want 7 (10 - 1 flat - 2 rolled)", result.ProcessedDamage)
	}
}

func TestTakeDamageReductionFloorsAtZero(t *testing.T) {
	c := NewAdversary("e1", "Target", "Tier 1", "Standard", 10, 3, 10, 6, []int{5, 10})
	c.AdversaryFeatures = []AdversaryFeature{
		{
			Name:    "Invulnerable",
			Passive: true,
			Effects: []FeatureEffect{
				{Category: FeatureEffectDamageReduction, Magnitude: 100},
			},
		},
	}

	hpBefore := c.HP
	result := c.TakeDamage(5, DamagePhysical, &mockRoller{})
	if result.ProcessedDamage != 0 {
		t.Errorf("ProcessedDamage = %d, want 0 (reduction exceeds raw)", result.ProcessedDamage)
	}
	if result.HPDamage != 0 {
		t.Errorf("HPDamage = %d, want 0", result.HPDamage)
	}
	if c.HP != hpBefore {
		t.Errorf("HP = %d, want %d (unchanged)", c.HP, hpBefore)
	}
}

func TestParseDamageType(t *testing.T) {
	if ParseDamageType("magic") != DamageMagic {
		t.Error("'magic' should return DamageMagic")
	}
	if ParseDamageType("physical") != DamagePhysical {
		t.Error("'physical' should return DamagePhysical")
	}
	if ParseDamageType("anything_else") != DamagePhysical {
		t.Error("unknown should default to DamagePhysical")
	}
}

func TestLoseArmorSlots(t *testing.T) {
	c := newTestPlayer()
	c.ArmorSlots = 1
	hpBefore := c.HP

	if hp := c.LoseArmorSlots(1); hp != 0 || c.ArmorSlots != 0 || c.HP != hpBefore {
		t.Errorf("with a slot left: hpMarked=%d ArmorSlots=%d HP=%d, want 0 / 0 / %d", hp, c.ArmorSlots, c.HP, hpBefore)
	}

	if hp := c.LoseArmorSlots(1); hp != 1 || c.HP != hpBefore-1 {
		t.Errorf("with no slot left: hpMarked=%d HP=%d, want 1 / %d", hp, c.HP, hpBefore-1)
	}

	c.HP = 1
	c.LoseArmorSlots(1)
	if !c.IsDefeated {
		t.Error("marking the last HP this way should defeat the combatant")
	}
}

func TestTakeDamageWithArmorCost(t *testing.T) {
	c := newTestAdversary()
	c.ArmorSlots = 3

	result := c.TakeDamageWithArmorCost(12, DamagePhysical, nil, 2)
	if c.ArmorSlots != 1 || result.HPDamage != 2 {
		t.Errorf("ArmorSlots=%d HPDamage=%d, want 1 / 2 (two slots for one step)", c.ArmorSlots, result.HPDamage)
	}

	result = c.TakeDamageWithArmorCost(12, DamagePhysical, nil, 2)
	if c.ArmorSlots != 1 || result.ArmorUsed || result.HPDamage != 3 {
		t.Errorf("ArmorSlots=%d ArmorUsed=%v HPDamage=%d, want 1 / false / 3 (can't pay two slots)", c.ArmorSlots, result.ArmorUsed, result.HPDamage)
	}
}

func TestAttackDamageFormulaUsesProficiency(t *testing.T) {
	pc := NewCombatant("p1", "Hero", SidePlayer, 5, 12, 3, 15, 8)
	for _, tt := range []struct{ dice, want string }{
		{"1d8+2", "3d8+2"},
		{"2d6", "3d6"},
		{"1d8+1d6+1", "3d8+1d6+1"},
		{"", "3d6"}, // the engine's 1d6 default also scales with Proficiency
	} {
		pc.DamageDice = tt.dice
		if got := pc.AttackDamageFormula(); got != tt.want {
			t.Errorf("PC %q = %q, want %q", tt.dice, got, tt.want)
		}
	}

	adv := NewAdversary("e1", "Ogre", "Tier 2", "Bruiser", 14, 3, 20, 6, []int{10, 20})
	adv.DamageDice = "2d12+4"
	if got := adv.AttackDamageFormula(); got != "2d12+4" {
		t.Errorf("adversary = %q, want unchanged 2d12+4", got)
	}
}

// Restrained has no mechanical effect without ranges, so no one spends a turn on it.
func TestRestrainedIsNotWorthClearing(t *testing.T) {
	c := NewCombatant("p1", "Hero", SidePlayer, 3, 10, 2, 10, 8)
	c.AddCondition(&ActiveCondition{Type: ConditionRestrained, Duration: DurationUntilCleared})
	if cond := c.ClearableCondition(); cond != nil {
		t.Errorf("ClearableCondition = %v, want nil", cond)
	}
}

func TestEligibleSpotlightAlliesFiltersByName(t *testing.T) {
	leader := NewAdversary("e1", "Fang Lord", "Tier 2", "Leader", 14, 3, 8, 4, []int{8, 15})
	shifter := NewAdversary("e2", "Guahalan Shifter", "Tier 2", "Standard", 14, 3, 6, 3, []int{8, 15})
	summoned := NewAdversary("e3", "Guahalan Shifter #2", "Tier 2", "Standard", 14, 3, 6, 3, []int{8, 15})
	other := NewAdversary("e4", "Guahalan Spirit Beast", "Tier 2", "Minion", 14, 3, 1, 1, []int{8, 15})
	dead := NewAdversary("e5", "Guahalan Shifter", "Tier 2", "Standard", 14, 3, 6, 3, []int{8, 15})
	dead.HP = 0
	dead.IsDefeated = true
	state := NewCombatState(nil, []*Combatant{leader, shifter, summoned, other, dead}, 0, 0, 5)

	got := state.EligibleSpotlightAllies(leader, FeatureEffect{AllyName: "Guahalan Shifter"})
	if len(got) != 2 || got[0] != shifter || got[1] != summoned {
		t.Errorf("eligible = %v, want the two living Shifters", got)
	}
	if all := state.EligibleSpotlightAllies(leader, FeatureEffect{}); len(all) != 3 {
		t.Errorf("unfiltered eligible = %d, want 3", len(all))
	}
}

func unstoppableDie(held, limit int) DomainAbility {
	return DomainAbility{
		Name: "Unstoppable", Passive: true, UsesTokens: true, TokensHeld: held, TokensMax: limit,
		Effects: []FeatureEffect{
			{Category: FeatureEffectTokenOnDamage, Magnitude: 1},
			{Category: FeatureEffectDamageBonus, Magnitude: 1, PerToken: true},
			{Category: FeatureEffectSeverityReduction, Magnitude: 1, DamageType: "physical"},
			{Category: FeatureEffectConditionImmunity, Conditions: []ConditionType{ConditionRestrained, ConditionVulnerable}},
		},
	}
}

// Level 3 PC: Major 6, Severe 9
func TestSeverityReductionDropsPhysicalDamageOneThreshold(t *testing.T) {
	tests := []struct {
		name       string
		damage     int
		damageType DamageType
		held       int
		wantHP     int
	}{
		{"Severe becomes Major", 9, DamagePhysical, 1, 2},
		{"Major becomes Minor", 6, DamagePhysical, 1, 1},
		{"Minor becomes None", 3, DamagePhysical, 1, 0},
		{"magic damage is untouched", 9, DamageMagic, 1, 3},
		{"no die, no reduction", 9, DamagePhysical, 0, 3},
	}
	for _, tt := range tests {
		pc := newTestPlayer()
		pc.DomainAbilities = []DomainAbility{unstoppableDie(tt.held, 4)}
		if got := pc.TakeDamage(tt.damage, tt.damageType, nil).HPDamage; got != tt.wantHP {
			t.Errorf("%s: marked %d HP, want %d", tt.name, got, tt.wantHP)
		}
	}
}

func TestSeverityReductionToNoneSavesTheArmorSlot(t *testing.T) {
	pc := newTestPlayer()
	pc.ArmorSlots = 2
	pc.DomainAbilities = []DomainAbility{unstoppableDie(1, 4)}

	if res := pc.TakeDamage(3, DamagePhysical, nil); res.ArmorUsed || pc.ArmorSlots != 2 || pc.HP != pc.HPMax {
		t.Errorf("Minor hit: armor used %v, slots %d, HP %d; want no armor, 2 slots, full HP", res.ArmorUsed, pc.ArmorSlots, pc.HP)
	}
	// Severe drops to Major, then armor takes it to Minor
	if res := pc.TakeDamage(9, DamagePhysical, nil); res.HPDamage != 1 || pc.ArmorSlots != 1 {
		t.Errorf("Severe hit: marked %d HP, %d slots left; want 1 HP, 1 slot", res.HPDamage, pc.ArmorSlots)
	}
}

func TestDamageBonusFollowsTheTokenDie(t *testing.T) {
	pc := newTestPlayer()
	pc.DomainAbilities = []DomainAbility{unstoppableDie(3, 4)}
	if got := SumParts(pc.DamageBonusParts(DamagePhysical, nil)); got != 3 {
		t.Errorf("bonus at die value 3 = %d, want 3", got)
	}
	pc.DomainAbilities[0].TokensHeld = 0
	if got := SumParts(pc.DamageBonusParts(DamagePhysical, nil)); got != 0 {
		t.Errorf("bonus with no die = %d, want 0", got)
	}

	// Without tokens a passive bonus is simply always on
	pc.DomainAbilities = []DomainAbility{{Passive: true, Effects: []FeatureEffect{{Category: FeatureEffectDamageBonus, Magnitude: 2}}}}
	if got := SumParts(pc.DamageBonusParts(DamagePhysical, nil)); got != 2 {
		t.Errorf("flat passive bonus = %d, want 2", got)
	}
}

func TestTokenDieStepsUpAndEndsPastItsMaximum(t *testing.T) {
	pc := newTestPlayer()
	pc.DomainAbilities = []DomainAbility{unstoppableDie(3, 4)}

	if ended := pc.StepTokensOnDamage(); len(ended) != 0 || pc.DomainAbilities[0].TokensHeld != 4 {
		t.Fatalf("3 -> %d (ended %v), want 4 and still going", pc.DomainAbilities[0].TokensHeld, ended)
	}
	if ended := pc.StepTokensOnDamage(); len(ended) != 1 || ended[0] != "Unstoppable" || pc.DomainAbilities[0].TokensHeld != 0 {
		t.Fatalf("4 -> %d (ended %v), want the die removed", pc.DomainAbilities[0].TokensHeld, ended)
	}
	// Once removed it stays off: there is no die left to step
	if ended := pc.StepTokensOnDamage(); len(ended) != 0 || pc.DomainAbilities[0].TokensHeld != 0 {
		t.Errorf("a removed die must not restart: held %d, ended %v", pc.DomainAbilities[0].TokensHeld, ended)
	}
}

func TestConditionImmunityLastsWhileTheDieDoes(t *testing.T) {
	pc := newTestPlayer()
	pc.AddCondition(&ActiveCondition{Type: ConditionRestrained, Duration: DurationUntilCleared})
	pc.AddCondition(&ActiveCondition{Type: ConditionBurning, Duration: DurationUntilCleared})
	pc.DomainAbilities = []DomainAbility{unstoppableDie(4, 4)}

	pc.DropBlockedConditions()
	if pc.HasCondition(ConditionRestrained) || !pc.HasCondition(ConditionBurning) {
		t.Errorf("becoming immune should end Restrained and leave Burning: %v", pc.Conditions)
	}
	pc.AddCondition(&ActiveCondition{Type: ConditionVulnerable, Duration: DurationUntilCleared})
	pc.AddStress(pc.StressMax)
	if pc.HasCondition(ConditionVulnerable) || !pc.IsFullyStressed {
		t.Errorf("Vulnerable = %v, fully stressed = %v; want immune but fully stressed", pc.HasCondition(ConditionVulnerable), pc.IsFullyStressed)
	}

	pc.StepTokensOnDamage()
	if !pc.HasCondition(ConditionVulnerable) {
		t.Error("full Stress should make the PC Vulnerable once the die is gone")
	}
}

func TestClearArmorSlotsStopsAtTheArmorScore(t *testing.T) {
	pc := newTestPlayer()
	pc.ArmorSlots = 3
	clone := CloneCombatant(pc, "p1")
	if clone.ArmorMax != 3 {
		t.Fatalf("ArmorMax = %d, want 3 from the template's slots", clone.ArmorMax)
	}

	clone.ArmorSlots = 2
	if cleared := clone.ClearArmorSlots(2); cleared != 1 || clone.ArmorSlots != 3 {
		t.Errorf("cleared %d, slots %d; want 1 cleared, 3 slots", cleared, clone.ArmorSlots)
	}
}

// The log reads HP and armor "after" from the result, so they must be the values left behind.
func TestDamageResultReportsHPAndArmorAfter(t *testing.T) {
	pc := newTestPlayer()
	pc.ArmorSlots = 2

	// Level 3: 9 is Severe, armor makes it Major
	res := pc.TakeDamage(9, DamagePhysical, nil)
	if res.HPBefore != 15 || res.HPAfter != 13 || res.ArmorSlotsBefore != 2 || res.ArmorSlotsAfter != 1 {
		t.Errorf("HP %d -> %d, armor %d -> %d; want 15 -> 13 and 2 -> 1", res.HPBefore, res.HPAfter, res.ArmorSlotsBefore, res.ArmorSlotsAfter)
	}
}
