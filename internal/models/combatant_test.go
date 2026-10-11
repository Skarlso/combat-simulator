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
	player.AttackTrait = 1
	if player.GetAttackBonus() != 1 {
		t.Errorf("player GetAttackBonus() = %d, want 1 (trait, not proficiency)", player.GetAttackBonus())
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
