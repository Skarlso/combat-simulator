package engine

import (
	"strings"
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)

// unstoppable mirrors the Guardian mapping in data/player_feature_mappings.json
func unstoppable(dieSize int) models.DomainAbility {
	return models.DomainAbility{
		Name: "Unstoppable", Passive: true, MaxUses: 1, UsesLeft: 1, RefreshType: "long_rest",
		Trigger:    models.FeatureTriggerCondition{Type: models.TriggerStartCombat},
		UsesTokens: true, TokenName: "Unstoppable Die", TokensMax: dieSize,
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectTokenPlace, Magnitude: 1},
			{Category: models.FeatureEffectTokenOnDamage, Magnitude: 1},
			{Category: models.FeatureEffectDamageBonus, Magnitude: 1, PerToken: true},
			{Category: models.FeatureEffectSeverityReduction, Magnitude: 1, DamageType: "physical"},
			{Category: models.FeatureEffectConditionImmunity, Conditions: []models.ConditionType{models.ConditionRestrained, models.ConditionVulnerable}},
		},
	}
}

func frontlineTank() models.DomainAbility {
	return models.DomainAbility{
		Name: "Frontline Tank", Passive: true, HopeCost: 3, RefreshType: "unlimited",
		Trigger: models.FeatureTriggerCondition{Type: models.TriggerStartTurn},
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectArmorClear, Magnitude: 2, TargetType: models.FeatureTargetSelf}},
	}
}

func resultsFor(state *models.CombatState, feature string) []string {
	var out []string
	for _, a := range state.ActionHistory {
		if a.FeatureName == feature {
			out = append(out, a.Result)
		}
	}
	return out
}

func TestUnstoppableSwitchesOnAndAddsItsDieToDamage(t *testing.T) {
	// 6 damage is Major against thresholds 4/7; the die's +1 makes it Severe
	ce := newMockEngine(&mockDiceRoller{damage: 6}, &firstPicker{})
	guardian := tPlayer("p1", "Guardian")
	guardian.DomainAbilities = []models.DomainAbility{unstoppable(4)}
	ogre := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{guardian}, []*models.Combatant{ogre}, 0, 0)

	ce.ExecuteTurnWithDecision(guardian, &strategy.ActionDecision{Target: ogre}, state)

	die := guardian.DomainAbilities[0]
	if ogre.HP != 7 || die.TokensHeld != 2 || die.UsesLeft != 0 {
		t.Fatalf("ogre HP %d, die %d, uses left %d; want 7 / 2 / 0", ogre.HP, die.TokensHeld, die.UsesLeft)
	}
	if got := resultsFor(state, "Unstoppable"); len(got) != 1 || got[0] != "Triggered" {
		t.Errorf("Unstoppable log = %v, want one Triggered row", got)
	}
	attack := state.ActionHistory[len(state.ActionHistory)-1]
	if attack.DamageRoll != 7 || attack.DamageBonus != 0 || len(attack.DamageParts) != 1 || attack.DamageParts[0] != (models.LogPart{Label: "Unstoppable", Value: 1}) {
		t.Errorf("attack rolled %d with parts %v and weapon bonus %d, want 7 / [Unstoppable 1] / 0", attack.DamageRoll, attack.DamageParts, attack.DamageBonus)
	}

	// Second turn: no second activation, the die value 2 is added, then it steps to 3
	ce.ExecuteTurnWithDecision(guardian, &strategy.ActionDecision{Target: ogre}, state)
	if got := guardian.DomainAbilities[0].TokensHeld; got != 3 {
		t.Errorf("die after second hit = %d, want 3", got)
	}
	if got := state.ActionHistory[len(state.ActionHistory)-1].DamageRoll; got != 8 {
		t.Errorf("second attack rolled %d, want 8", got)
	}
	if got := resultsFor(state, "Unstoppable"); len(got) != 1 {
		t.Errorf("Unstoppable log = %v, want it to trigger once", got)
	}
}

func TestUnstoppableEndsWhenTheDiePassesItsMaximum(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 2}, &firstPicker{})
	guardian := tPlayer("p1", "Guardian")
	guardian.DomainAbilities = []models.DomainAbility{unstoppable(4)}
	guardian.DomainAbilities[0].TokensHeld, guardian.DomainAbilities[0].UsesLeft = 4, 0
	ogre := tEnemy("e1", "Ogre", 20)
	state := newStateWith([]*models.Combatant{guardian}, []*models.Combatant{ogre}, 0, 0)

	// 2 + 4 from the die is Major: 2 HP, and the die would go to 5
	ce.ExecuteTurnWithDecision(guardian, &strategy.ActionDecision{Target: ogre}, state)
	if ogre.HP != 18 || guardian.DomainAbilities[0].TokensHeld != 0 {
		t.Fatalf("ogre HP %d, die %d; want 18 and the die removed", ogre.HP, guardian.DomainAbilities[0].TokensHeld)
	}
	if got := resultsFor(state, "Unstoppable"); len(got) != 1 || !strings.HasPrefix(got[0], "Ends") {
		t.Errorf("Unstoppable log = %v, want one Ends row", got)
	}

	// No longer Unstoppable: plain 2 damage, and nothing switches it back on
	ce.ExecuteTurnWithDecision(guardian, &strategy.ActionDecision{Target: ogre}, state)
	if ogre.HP != 17 || guardian.DomainAbilities[0].TokensHeld != 0 {
		t.Errorf("ogre HP %d, die %d; want 17 and no die", ogre.HP, guardian.DomainAbilities[0].TokensHeld)
	}
}

func TestUnstoppableDieWaitsForDamageThatMarksHP(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 6}, &firstPicker{})
	guardian := tPlayer("p1", "Guardian")
	guardian.DomainAbilities = []models.DomainAbility{unstoppable(4)}
	ghost := tEnemy("e1", "Ghost", 10)
	ghost.AddImmunity(models.DamagePhysical)
	state := newStateWith([]*models.Combatant{guardian}, []*models.Combatant{ghost}, 0, 0)

	ce.ExecuteTurnWithDecision(guardian, &strategy.ActionDecision{Target: ghost}, state)
	if got := guardian.DomainAbilities[0].TokensHeld; got != 1 {
		t.Errorf("die = %d after a hit that marked nothing, want 1", got)
	}
}

func TestUnstoppableDieCountsAbilityDamage(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{damage: 3})
	guardian := tPlayer("p1", "Guardian")
	guardian.DomainAbilities = []models.DomainAbility{unstoppable(6)}
	guardian.DomainAbilities[0].TokensHeld = 2
	ogre := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{guardian}, []*models.Combatant{ogre}, 0, 0)

	// 3 + 2 from the die crosses the Major threshold of 4
	strike := models.FeatureEffect{Category: models.FeatureEffectDamage, DiceFormula: "1d8", DamageType: "magic"}
	if err := fr.ResolveEffect(strike, guardian, ogre, state, nil); err != nil {
		t.Fatal(err)
	}
	if ogre.HP != 8 || guardian.DomainAbilities[0].TokensHeld != 3 {
		t.Errorf("ogre HP %d, die %d; want 8 / 3", ogre.HP, guardian.DomainAbilities[0].TokensHeld)
	}
}

func TestUnstoppableShrugsOffConditionsAndSeverity(t *testing.T) {
	// tPlayer is level 3: Major 6, Severe 9
	ce := newMockEngine(&mockDiceRoller{damage: 9}, &firstPicker{})
	guardian := tPlayer("p1", "Guardian")
	guardian.DomainAbilities = []models.DomainAbility{unstoppable(4)}
	guardian.DomainAbilities[0].TokensHeld = 1
	ogre := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{guardian}, []*models.Combatant{ogre}, 0, 0)

	ce.executeAttack(ogre, guardian, state)
	if guardian.HP != 8 {
		t.Errorf("guardian HP %d after a Severe hit, want 8 (taken as Major)", guardian.HP)
	}

	for _, category := range []models.FeatureEffectCategory{models.FeatureEffectApplyVulnerable, models.FeatureEffectApplyRestrained} {
		effect := models.FeatureEffect{Category: category, Duration: "until_cleared"}
		if err := ce.featureResolver.ResolveEffect(effect, ogre, guardian, state, nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(guardian.Conditions) != 0 {
		t.Errorf("conditions = %v, want none while Unstoppable", guardian.Conditions)
	}
}

func TestBecomingUnstoppableEndsRestrainedAndVulnerable(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 1}, &firstPicker{})
	guardian := tPlayer("p1", "Guardian")
	guardian.DomainAbilities = []models.DomainAbility{unstoppable(4)}
	guardian.AddCondition(&models.ActiveCondition{Type: models.ConditionVulnerable, Duration: models.DurationUntilCleared})
	guardian.AddCondition(&models.ActiveCondition{Type: models.ConditionRestrained, Duration: models.DurationUntilCleared})
	ogre := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{guardian}, []*models.Combatant{ogre}, 0, 0)

	ce.ExecuteTurnWithDecision(guardian, &strategy.ActionDecision{Target: ogre}, state)
	if len(guardian.Conditions) != 0 {
		t.Errorf("conditions = %v, want none once Unstoppable", guardian.Conditions)
	}
}

func TestFrontlineTankClearsArmorOnceTwoSlotsAreMarked(t *testing.T) {
	tests := []struct {
		name               string
		slots, hope        int
		wantSlots, wantUse int
	}{
		{"two marked, 3 Hope: fires", 1, 3, 3, 1},
		{"one marked: waits", 2, 3, 2, 0},
		{"2 Hope: can't pay", 1, 2, 1, 0},
		{"nothing marked: waits", 3, 6, 3, 0},
	}
	for _, tt := range tests {
		ce := newMockEngine(&mockDiceRoller{damage: 1}, &firstPicker{})
		guardian := tPlayer("p1", "Guardian")
		guardian.ArmorMax, guardian.ArmorSlots = 3, tt.slots
		guardian.DomainAbilities = []models.DomainAbility{frontlineTank()}
		ogre := tEnemy("e1", "Ogre", 10)
		state := newStateWith([]*models.Combatant{guardian}, []*models.Combatant{ogre}, tt.hope, 0)

		ce.ExecuteTurnWithDecision(guardian, &strategy.ActionDecision{Target: ogre}, state)

		// The attack that follows rolls with Hope and gains one back
		wantHope := min(tt.hope-3*tt.wantUse+1, models.MaxHope)
		if guardian.ArmorSlots != tt.wantSlots || guardian.Hope != wantHope || len(resultsFor(state, "Frontline Tank")) != tt.wantUse {
			t.Errorf("%s: slots %d, Hope %d, log %v; want slots %d, Hope %d, %d use(s)",
				tt.name, guardian.ArmorSlots, guardian.Hope, resultsFor(state, "Frontline Tank"), tt.wantSlots, wantHope, tt.wantUse)
		}
	}
}

func TestGrantArmorLandsOnTheActorNotTheTarget(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	guardian := tPlayer("p1", "Guardian")
	ogre := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{guardian}, []*models.Combatant{ogre}, 0, 0)

	effect := models.FeatureEffect{Category: models.FeatureEffectGrantArmor, Magnitude: 2, TargetType: models.FeatureTargetSingle}
	if err := fr.ResolveEffect(effect, guardian, ogre, state, nil); err != nil {
		t.Fatal(err)
	}
	if guardian.ArmorSlots != 2 || ogre.ArmorSlots != 0 {
		t.Errorf("armor: guardian %d, ogre %d; want 2 / 0", guardian.ArmorSlots, ogre.ArmorSlots)
	}
}

// A PC who is hit before their first turn must already be Unstoppable.
func TestStancesSwitchOnWhenTheFightStarts(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 9}, &firstPicker{})
	first, waiting := tPlayer("p1", "First"), tPlayer("p2", "Waiting")
	first.DomainAbilities = []models.DomainAbility{unstoppable(4)}
	waiting.DomainAbilities = []models.DomainAbility{unstoppable(4)}
	noMercy := models.DomainAbility{
		Name: "No Mercy", Passive: true, HopeCost: 3, MaxUses: 1, UsesLeft: 1, UsesTokens: true, TokensMax: 1,
		Trigger: models.FeatureTriggerCondition{Type: models.TriggerStartCombat},
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectTokenPlace, Magnitude: 1}, {Category: models.FeatureEffectAttackBonus, Magnitude: 1}},
	}
	first.DomainAbilities = append(first.DomainAbilities, noMercy)
	ogre := tEnemy("e1", "Ogre", 20)
	state := newStateWith([]*models.Combatant{first, waiting}, []*models.Combatant{ogre}, 2, 0)

	ce.beginCombat(state)
	if first.DomainAbilities[0].TokensHeld != 1 || waiting.DomainAbilities[0].TokensHeld != 1 || len(resultsFor(state, "Unstoppable")) != 2 {
		t.Fatalf("dice %d / %d, log %v; want both PCs Unstoppable before anyone acts", first.DomainAbilities[0].TokensHeld, waiting.DomainAbilities[0].TokensHeld, resultsFor(state, "Unstoppable"))
	}
	// 9 is Severe for a level 3 PC; taken as Major by the one who has not had a turn
	ce.executeAttack(ogre, waiting, state)
	if waiting.HP != 8 {
		t.Errorf("waiting PC at %d HP, want 8", waiting.HP)
	}

	// No Mercy costs 3 Hope: not affordable at the start with 2, so it waits for a turn that can pay
	if first.DomainAbilities[1].TokensHeld != 0 || first.Hope != 2 {
		t.Fatalf("No Mercy on with %d Hope left; want it waiting", first.Hope)
	}
	first.Hope = 3
	ce.ExecuteTurnWithDecision(first, &strategy.ActionDecision{Target: ogre}, state)
	if first.DomainAbilities[1].TokensHeld != 1 || len(resultsFor(state, "Unstoppable")) != 2 {
		t.Errorf("No Mercy tokens %d, Unstoppable log %v; want No Mercy on and no second Unstoppable", first.DomainAbilities[1].TokensHeld, resultsFor(state, "Unstoppable"))
	}
}
