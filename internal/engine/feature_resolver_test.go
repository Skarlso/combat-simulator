package engine

import (
	"testing"

	"github.com/skarlso/combat-simulator/internal/models"
)

func newTestFR(dice *mockDiceRoller) *FeatureResolver {
	return &FeatureResolver{dice: dice}
}

func newStateWith(players, enemies []*models.Combatant, hope, fear int) *models.CombatState {
	return models.NewCombatState(players, enemies, hope, fear, 3)
}

func TestCanActivateAdversaryFeatureFearCost(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Ogre", 10)
	state := newStateWith(nil, []*models.Combatant{actor}, 0, 1)

	feat := &models.AdversaryFeature{FearCost: 2}
	if fr.CanActivateAdversaryFeature(feat, actor, state) {
		t.Error("should not activate when FearPool < FearCost")
	}

	state.FearPool = 2
	if !fr.CanActivateAdversaryFeature(feat, actor, state) {
		t.Error("should activate when FearPool >= FearCost")
	}
}

func TestCanActivateAdversaryFeatureStressOverflow(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Ogre", 10)
	actor.Stress = 5
	actor.StressMax = 6
	state := newStateWith(nil, []*models.Combatant{actor}, 0, 0)

	feat := &models.AdversaryFeature{StressCost: 2}
	if fr.CanActivateAdversaryFeature(feat, actor, state) {
		t.Error("should not activate when stress would overflow")
	}
}

func TestCanActivateAdversaryFeatureUsesExhausted(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Ogre", 10)
	state := newStateWith(nil, []*models.Combatant{actor}, 0, 0)

	feat := &models.AdversaryFeature{MaxUses: 1, UsesLeft: 0}
	if fr.CanActivateAdversaryFeature(feat, actor, state) {
		t.Error("should not activate when UsesLeft is zero")
	}
}

func TestCanActivateAdversaryFeatureTokenGate(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Demon", 10)
	state := newStateWith(nil, []*models.Combatant{actor}, 0, 0)

	feat := &models.AdversaryFeature{
		UsesTokens: true,
		TokensHeld: 1,
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectTokenSpend, Magnitude: 3},
		},
	}
	if fr.CanActivateAdversaryFeature(feat, actor, state) {
		t.Error("should not activate when TokensHeld < spend magnitude")
	}

	feat.TokensHeld = 3
	if !fr.CanActivateAdversaryFeature(feat, actor, state) {
		t.Error("should activate when TokensHeld meets spend magnitude")
	}
}

func TestActivateAdversaryFeatureTokenPlaceAfterEffects(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Demon", 10)
	target := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 0)

	feat := &models.AdversaryFeature{
		Name:       "Build Rage",
		UsesTokens: true,
		TokensHeld: 0,
		TokensMax:  5,
		DCType:     "none",
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectStressInflict, Magnitude: 1},
			{Category: models.FeatureEffectTokenPlace, Magnitude: 2},
		},
	}
	actor.AdversaryFeatures = []models.AdversaryFeature{*feat}

	_, err := fr.ActivateAdversaryFeature(&actor.AdversaryFeatures[0], actor, target, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actor.AdversaryFeatures[0].TokensHeld != 2 {
		t.Errorf("TokensHeld = %d, want 2", actor.AdversaryFeatures[0].TokensHeld)
	}
	if target.Stress != 1 {
		t.Errorf("target Stress = %d, want 1", target.Stress)
	}
}

func TestActivateAdversaryFeatureTokenSpendConsumes(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{damage: 4})
	actor := tEnemy("e1", "Demon", 10)
	target := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 0)

	feat := &models.AdversaryFeature{
		UsesTokens: true,
		TokensHeld: 3,
		DCType:     "none",
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectTokenSpend, Magnitude: 2},
			{Category: models.FeatureEffectStressInflict, Magnitude: 1},
		},
	}
	actor.AdversaryFeatures = []models.AdversaryFeature{*feat}

	_, _ = fr.ActivateAdversaryFeature(&actor.AdversaryFeatures[0], actor, target, state)
	if actor.AdversaryFeatures[0].TokensHeld != 1 {
		t.Errorf("TokensHeld = %d, want 1 (3 - 2)", actor.AdversaryFeatures[0].TokensHeld)
	}
	if target.Stress != 1 {
		t.Errorf("target Stress = %d, want 1", target.Stress)
	}
}

func TestActivateAdversaryFeatureTokenPlaceCapped(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Demon", 10)
	state := newStateWith(nil, []*models.Combatant{actor}, 0, 0)

	feat := &models.AdversaryFeature{
		UsesTokens: true,
		TokensHeld: 4,
		TokensMax:  5,
		DCType:     "none",
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectTokenPlace, Magnitude: 10},
		},
	}
	actor.AdversaryFeatures = []models.AdversaryFeature{*feat}

	_, _ = fr.ActivateAdversaryFeature(&actor.AdversaryFeatures[0], actor, nil, state)
	if actor.AdversaryFeatures[0].TokensHeld != 5 {
		t.Errorf("TokensHeld = %d, want 5 (capped)", actor.AdversaryFeatures[0].TokensHeld)
	}
}

func TestCanActivateDomainAbilityTokenGate(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{actor}, nil, 0, 0)

	ability := models.DomainAbility{
		UsesTokens: true,
		TokensHeld: 1,
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectTokenSpend, Magnitude: 3},
		},
	}
	if fr.CanActivateDomainAbility(ability, actor, state) {
		t.Error("should not activate when TokensHeld < spend magnitude")
	}

	ability.TokensHeld = 3
	if !fr.CanActivateDomainAbility(ability, actor, state) {
		t.Error("should activate when TokensHeld meets spend magnitude")
	}
}

func TestActivateAdversaryFeaturePaysCostsAndDecrementsUses(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{damage: 3})
	actor := tEnemy("e1", "Ogre", 10)
	target := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 3)

	feat := &models.AdversaryFeature{
		Name:       "Cleave",
		FearCost:   2,
		StressCost: 1,
		MaxUses:    2,
		UsesLeft:   2,
		DCType:     "none",
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectStressInflict, Magnitude: 1},
		},
	}

	_, err := fr.ActivateAdversaryFeature(feat, actor, target, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.FearPool != 1 {
		t.Errorf("FearPool = %d, want 1 (3 - 2 cost)", state.FearPool)
	}
	if actor.Stress != 1 {
		t.Errorf("actor Stress = %d, want 1", actor.Stress)
	}
	if feat.UsesLeft != 1 {
		t.Errorf("UsesLeft = %d, want 1", feat.UsesLeft)
	}
	if target.Stress != 1 {
		t.Errorf("target Stress = %d, want 1 (effect applied)", target.Stress)
	}
}

func TestActivateAdversaryFeatureBlockedLeavesStateIntact(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Ogre", 10)
	target := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 0)

	feat := &models.AdversaryFeature{
		Name:     "Cleave",
		FearCost: 2,
		DCType:   "none",
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectStressInflict, Magnitude: 1},
		},
	}

	_, err := fr.ActivateAdversaryFeature(feat, actor, target, state)
	if err == nil {
		t.Fatal("expected error when Fear pool insufficient")
	}
	if state.FearPool != 0 {
		t.Errorf("FearPool = %d, want 0 (no cost spent on block)", state.FearPool)
	}
	if target.Stress != 0 {
		t.Errorf("target Stress = %d, want 0 (effect not applied)", target.Stress)
	}
}

func TestActivateDomainAbilityTokenPlaceAfterEffects(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tPlayer("p1", "Hero")
	target := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{actor}, []*models.Combatant{target}, 3, 0)

	ability := &models.DomainAbility{
		Name:       "Channel Chaos",
		UsesTokens: true,
		TokensHeld: 0,
		TokensMax:  3,
		HopeCost:   1,
		DCType:     "none",
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectStressInflict, Magnitude: 1},
			{Category: models.FeatureEffectTokenPlace, Magnitude: 2},
		},
	}

	_, err := fr.ActivateDomainAbility(ability, actor, target, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ability.TokensHeld != 2 {
		t.Errorf("TokensHeld = %d, want 2", ability.TokensHeld)
	}
	if target.Stress != 1 {
		t.Errorf("target Stress = %d, want 1", target.Stress)
	}
}

func TestActivateDomainAbilityTokenPlaceCapped(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{actor}, nil, 0, 0)

	ability := &models.DomainAbility{
		UsesTokens: true,
		TokensHeld: 4,
		TokensMax:  5,
		DCType:     "none",
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectTokenPlace, Magnitude: 10},
		},
	}

	_, _ = fr.ActivateDomainAbility(ability, actor, nil, state)
	if ability.TokensHeld != 5 {
		t.Errorf("TokensHeld = %d, want 5 (capped at TokensMax)", ability.TokensHeld)
	}
}

func TestActivateDomainAbilityTokenSpendConsumes(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{damage: 4})
	actor := tPlayer("p1", "Hero")
	target := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{actor}, []*models.Combatant{target}, 0, 0)

	ability := &models.DomainAbility{
		UsesTokens: true,
		TokensHeld: 3,
		DCType:     "none",
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectTokenSpend, Magnitude: 2},
			{Category: models.FeatureEffectStressInflict, Magnitude: 1},
		},
	}

	_, _ = fr.ActivateDomainAbility(ability, actor, target, state)
	if ability.TokensHeld != 1 {
		t.Errorf("TokensHeld = %d, want 1 (3 - 2 spend)", ability.TokensHeld)
	}
	if target.Stress != 1 {
		t.Errorf("target Stress = %d, want 1", target.Stress)
	}
}

func TestDomainAbilityTokensPersistAcrossTurns(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tPlayer("p1", "Chaos Mage")
	target := tEnemy("e1", "Ogre", 20)
	state := newStateWith([]*models.Combatant{actor}, []*models.Combatant{target}, 0, 0)

	actor.DomainAbilities = []models.DomainAbility{
		{
			Name:       "Unleash Chaos (Charge)",
			UsesTokens: true,
			TokenName:  "Chaos",
			TokensHeld: 0,
			TokensMax:  3,
			DCType:     "none",
			Effects: []models.FeatureEffect{
				{Category: models.FeatureEffectTokenPlace, Magnitude: 1},
			},
		},
	}

	ability := &actor.DomainAbilities[0]
	for turn := 1; turn <= 4; turn++ {
		if _, err := fr.ActivateDomainAbility(ability, actor, target, state); err != nil {
			t.Fatalf("turn %d: %v", turn, err)
		}
	}

	if ability.TokensHeld != 3 {
		t.Errorf("TokensHeld after 4 charges = %d, want 3 (capped at TokensMax)", ability.TokensHeld)
	}
}

func TestDomainAbilityTokensChargeThenRelease(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{damage: 5})
	actor := tPlayer("p1", "Chaos Mage")
	target := tEnemy("e1", "Ogre", 30)
	state := newStateWith([]*models.Combatant{actor}, []*models.Combatant{target}, 0, 0)

	actor.DomainAbilities = []models.DomainAbility{
		{
			Name:       "Charge Chaos",
			UsesTokens: true,
			TokenName:  "Chaos",
			TokensMax:  5,
			DCType:     "none",
			Effects: []models.FeatureEffect{
				{Category: models.FeatureEffectTokenPlace, Magnitude: 2},
			},
		},
		{
			Name:       "Release Chaos",
			UsesTokens: true,
			TokenName:  "Chaos",
			TokensMax:  5,
			DCType:     "none",
			Effects: []models.FeatureEffect{
				{Category: models.FeatureEffectTokenSpend, Magnitude: 4},
				{Category: models.FeatureEffectStressInflict, Magnitude: 2},
			},
		},
	}

	charge := &actor.DomainAbilities[0]
	release := &actor.DomainAbilities[1]

	if fr.CanActivateDomainAbility(*release, actor, state) {
		t.Fatal("release should be blocked when TokensHeld is 0")
	}

	release.TokensHeld = charge.TokensHeld
	_, _ = fr.ActivateDomainAbility(charge, actor, target, state)
	release.TokensHeld = charge.TokensHeld
	_, _ = fr.ActivateDomainAbility(charge, actor, target, state)
	release.TokensHeld = charge.TokensHeld

	if release.TokensHeld != 4 {
		t.Fatalf("release TokensHeld = %d, want 4 after two charges", release.TokensHeld)
	}

	if _, err := fr.ActivateDomainAbility(release, actor, target, state); err != nil {
		t.Fatalf("release activation failed: %v", err)
	}

	if release.TokensHeld != 0 {
		t.Errorf("release TokensHeld = %d, want 0 after spend", release.TokensHeld)
	}
	if target.Stress != 2 {
		t.Errorf("target Stress = %d, want 2 from release effect", target.Stress)
	}
}

func TestProcessTriggeredFeaturesSkipsNonPassive(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Ogre", 10)
	target := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 0)

	actor.AdversaryFeatures = []models.AdversaryFeature{
		{
			Name:    "Bloodlust",
			Passive: false,
			Trigger: models.FeatureTriggerCondition{Type: models.TriggerOnDamageTaken},
			DCType:  "none",
			Effects: []models.FeatureEffect{
				{Category: models.FeatureEffectStressInflict, Magnitude: 1},
			},
		},
	}

	fr.ProcessTriggeredFeatures(actor, target, state, models.TriggerOnDamageTaken, false)
	if target.Stress != 0 {
		t.Errorf("target Stress = %d, want 0 (non-passive should not auto-trigger)", target.Stress)
	}
}

func TestProcessTriggeredFeaturesFiresPassiveOnMatch(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Ogre", 10)
	target := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 0)

	actor.AdversaryFeatures = []models.AdversaryFeature{
		{
			Name:    "Thorns",
			Passive: true,
			Trigger: models.FeatureTriggerCondition{Type: models.TriggerOnDamageTaken},
			DCType:  "none",
			Effects: []models.FeatureEffect{
				{Category: models.FeatureEffectStressInflict, Magnitude: 1},
			},
		},
	}

	fr.ProcessTriggeredFeatures(actor, target, state, models.TriggerOnDamageTaken, false)
	if target.Stress != 1 {
		t.Errorf("target Stress = %d, want 1 (passive trigger should fire)", target.Stress)
	}
}

func TestProcessTriggeredFeaturesIgnoresDifferentTrigger(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Ogre", 10)
	target := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 0)

	actor.AdversaryFeatures = []models.AdversaryFeature{
		{
			Name:    "Thorns",
			Passive: true,
			Trigger: models.FeatureTriggerCondition{Type: models.TriggerOnDamageTaken},
			DCType:  "none",
			Effects: []models.FeatureEffect{
				{Category: models.FeatureEffectStressInflict, Magnitude: 1},
			},
		},
	}

	fr.ProcessTriggeredFeatures(actor, target, state, models.TriggerStartTurn, false)
	if target.Stress != 0 {
		t.Errorf("target Stress = %d, want 0 (trigger type mismatch)", target.Stress)
	}
}

func TestProcessTriggeredFeaturesCriticalFilter(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Ogre", 10)
	target := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 0)

	actor.AdversaryFeatures = []models.AdversaryFeature{
		{
			Name:    "Execute",
			Passive: true,
			Trigger: models.FeatureTriggerCondition{Type: models.TriggerOnCritical},
			DCType:  "none",
			Effects: []models.FeatureEffect{
				{Category: models.FeatureEffectStressInflict, Magnitude: 1},
			},
		},
	}

	fr.ProcessTriggeredFeatures(actor, target, state, models.TriggerOnCritical, false)
	if target.Stress != 0 {
		t.Errorf("target Stress = %d, want 0 (not a critical)", target.Stress)
	}

	fr.ProcessTriggeredFeatures(actor, target, state, models.TriggerOnCritical, true)
	if target.Stress != 1 {
		t.Errorf("target Stress = %d, want 1 (critical should fire)", target.Stress)
	}
}

func TestShouldTriggerFilterLowHP(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Ogre", 10)
	target := tPlayer("p1", "Hero")

	trigger := models.FeatureTriggerCondition{
		Type:   models.TriggerOnAttack,
		Filter: "if_target_below_50_hp",
	}

	target.HP = target.HPMax * 60 / 100
	if fr.shouldTrigger(trigger, models.TriggerOnAttack, false, actor, target) {
		t.Error("should not fire when target HP >= 50%")
	}

	target.HP = target.HPMax * 40 / 100
	if !fr.shouldTrigger(trigger, models.TriggerOnAttack, false, actor, target) {
		t.Error("should fire when target HP < 50%")
	}
}

func TestShouldTriggerFilterHidden(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tPlayer("p1", "Hero")

	trigger := models.FeatureTriggerCondition{
		Type:   models.TriggerOnAttack,
		Filter: "if_hidden",
	}

	if fr.shouldTrigger(trigger, models.TriggerOnAttack, false, actor, nil) {
		t.Error("should not fire when actor is not hidden")
	}

	actor.AddCondition(&models.ActiveCondition{Type: models.ConditionHidden, Duration: models.DurationUntilCleared})
	if !fr.shouldTrigger(trigger, models.TriggerOnAttack, false, actor, nil) {
		t.Error("should fire when actor has Hidden condition")
	}
}

func TestShouldTriggerUnknownFilterPasses(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tPlayer("p1", "Hero")

	trigger := models.FeatureTriggerCondition{
		Type:   models.TriggerOnAttack,
		Filter: "if_mercury_is_retrograde",
	}
	if !fr.shouldTrigger(trigger, models.TriggerOnAttack, false, actor, nil) {
		t.Error("unknown filter should default to true")
	}
}

func TestResolveEffectUnknownCategoryErrors(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{actor}, nil, 0, 0)

	effect := models.FeatureEffect{Category: "bogus_category"}
	err := fr.ResolveEffect(effect, actor, nil, state, nil)
	if err == nil {
		t.Error("expected error for unknown effect category")
	}
}

func TestResolveEffectStressInflictAndClear(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tPlayer("p1", "Hero")
	target := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{actor}, []*models.Combatant{target}, 0, 0)

	if err := fr.ResolveEffect(models.FeatureEffect{Category: models.FeatureEffectStressInflict, Magnitude: 2}, actor, target, state, nil); err != nil {
		t.Fatalf("stress inflict: %v", err)
	}
	if target.Stress != 2 {
		t.Errorf("target Stress = %d, want 2", target.Stress)
	}

	actor.Stress = 3
	if err := fr.ResolveEffect(models.FeatureEffect{Category: models.FeatureEffectStressClear, Magnitude: 2}, actor, nil, state, nil); err != nil {
		t.Fatalf("stress clear: %v", err)
	}
	if actor.Stress != 1 {
		t.Errorf("actor Stress = %d, want 1 (3 - 2 cleared)", actor.Stress)
	}
}

func TestResolveEffectHopeDrainAndFearGenerate(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Hexer", 10)
	target := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 4, 1)

	if err := fr.ResolveEffect(models.FeatureEffect{Category: models.FeatureEffectHopeDrain, Magnitude: 2}, actor, target, state, nil); err != nil {
		t.Fatalf("hope drain: %v", err)
	}
	if target.Hope != 2 {
		t.Errorf("Hope = %d, want 2 (4 - 2)", target.Hope)
	}

	if err := fr.ResolveEffect(models.FeatureEffect{Category: models.FeatureEffectFearGenerate, Magnitude: 3}, actor, nil, state, nil); err != nil {
		t.Fatalf("fear generate: %v", err)
	}
	if state.FearPool != 4 {
		t.Errorf("FearPool = %d, want 4 (1 + 3)", state.FearPool)
	}
}

func TestResolveEffectApplyConditionPassives(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tPlayer("p1", "Hero")
	target := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{actor}, []*models.Combatant{target}, 0, 0)

	cases := []struct {
		name     string
		cat      models.FeatureEffectCategory
		target   *models.Combatant
		expected models.ConditionType
	}{
		{"vulnerable", models.FeatureEffectApplyVulnerable, target, models.ConditionVulnerable},
		{"poison", models.FeatureEffectApplyPoison, target, models.ConditionPoisoned},
		{"ignite", models.FeatureEffectApplyIgnite, target, models.ConditionBurning},
		{"taunt", models.FeatureEffectApplyTaunt, target, models.ConditionTaunted},
		{"hidden_on_actor", models.FeatureEffectApplyHidden, actor, models.ConditionHidden},
		{"stunned", models.FeatureEffectApplyStunned, target, models.ConditionStunned},
		{"restrained", models.FeatureEffectApplyRestrained, target, models.ConditionRestrained},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			effect := models.FeatureEffect{Category: c.cat, Duration: "until_cleared"}
			if err := fr.ResolveEffect(effect, actor, target, state, nil); err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			if !c.target.HasCondition(c.expected) {
				t.Errorf("%s: target missing condition %v", c.name, c.expected)
			}
		})
	}
}

func TestResolveDirectDamageUsesThresholdsAndSkipsArmor(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tPlayer("p1", "Hero")
	target := tEnemy("e1", "Ogre", 10)
	target.ArmorSlots = 1
	state := newStateWith([]*models.Combatant{actor}, []*models.Combatant{target}, 0, 0)

	result := &DCCheckResult{}
	effect := models.FeatureEffect{Category: models.FeatureEffectDirectDamage, Magnitude: 12}
	if err := fr.ResolveEffect(effect, actor, target, state, result); err != nil {
		t.Fatal(err)
	}

	if target.HP != 7 {
		t.Errorf("HP = %d, want 7 (severe = 3 HP, not raw damage)", target.HP)
	}
	if target.ArmorSlots != 1 {
		t.Errorf("ArmorSlots = %d, want 1 (direct damage bypasses armor)", target.ArmorSlots)
	}
	if result.DamageDealt != 3 || result.Threshold != models.ThresholdSevere {
		t.Errorf("result = %d HP / %v, want 3 HP / Severe", result.DamageDealt, result.Threshold)
	}
}

func TestIgniteAndPoisonDealDamageEachTurn(t *testing.T) {
	dice := &mockDiceRoller{}
	fr := newTestFR(dice)
	resolver := NewCombatResolver(dice, fr)
	actor := tEnemy("e1", "Imp", 10)

	for _, cat := range []models.FeatureEffectCategory{models.FeatureEffectApplyIgnite, models.FeatureEffectApplyPoison} {
		target := tPlayer("p1", "Hero")
		state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 0)

		effect := models.FeatureEffect{Category: cat, Magnitude: 2, Duration: "until_cleared"}
		if err := fr.ResolveEffect(effect, actor, target, state, nil); err != nil {
			t.Fatal(err)
		}

		resolver.ProcessConditionTicks(target, models.TriggerTurnStart)
		resolver.ProcessConditionTicks(target, models.TriggerTurnStart)
		if target.HP != target.HPMax-2 {
			t.Errorf("%s: HP = %d, want %d after two ticks", cat, target.HP, target.HPMax-2)
		}
	}
}

func TestDomainAbilitySpendsOnlyTheActorsHope(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tPlayer("p1", "Hero")
	ally := tPlayer("p2", "Friend")
	target := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{actor, ally}, []*models.Combatant{target}, 0, 0)
	ally.Hope = 6

	ability := &models.DomainAbility{Name: "Flourish", HopeCost: 2}
	if fr.CanActivateDomainAbility(*ability, actor, state) {
		t.Error("an ally's Hope should not pay for the actor's ability")
	}

	actor.Hope = 3
	if _, err := fr.ActivateDomainAbility(ability, actor, target, state); err != nil {
		t.Fatal(err)
	}
	if actor.Hope != 1 || ally.Hope != 6 {
		t.Errorf("Hope = %d / %d, want 1 / 6", actor.Hope, ally.Hope)
	}
}

func TestAppliedConditionRemembersSourceDifficulty(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Spider", 10)
	actor.Difficulty = 14
	target := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 0)

	effect := models.FeatureEffect{Category: models.FeatureEffectApplyVulnerable, Duration: "until_cleared"}
	if err := fr.ResolveEffect(effect, actor, target, state, nil); err != nil {
		t.Fatal(err)
	}

	if cond := target.ClearableCondition(); cond == nil || cond.ClearDifficulty != 14 {
		t.Errorf("ClearableCondition = %+v, want ClearDifficulty 14", cond)
	}
}

func TestOnHitArmorShredFallsBackToHP(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 5}, &firstPicker{})
	ooze := tEnemy("e1", "Ooze", 10)
	ooze.AdversaryFeatures = []models.AdversaryFeature{{
		Name:    "Acidic Form",
		Passive: true,
		Trigger: models.FeatureTriggerCondition{Type: models.TriggerOnHit},
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectArmorDamage, Magnitude: 1}},
	}}
	target := tPlayer("p1", "Hero")
	target.CustomThresholds = []int{4, 7}
	target.ArmorSlots = 2
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{ooze}, 0, 0)

	// Major damage: one slot reduces it to Minor (1 HP), the acid marks the other
	ce.executeAttack(ooze, target, state)
	if target.ArmorSlots != 0 || target.HP != target.HPMax-1 {
		t.Errorf("ArmorSlots=%d HP=%d, want 0 / %d", target.ArmorSlots, target.HP, target.HPMax-1)
	}

	// No slots left: Major damage (2 HP) plus an additional HP for the acid
	ce.executeAttack(ooze, target, state)
	if target.HP != target.HPMax-4 {
		t.Errorf("HP=%d, want %d", target.HP, target.HPMax-4)
	}
}

func TestMomentumGrantsFearOnHit(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 5}, &firstPicker{})
	bear := tEnemy("e1", "Bear", 10)
	bear.AdversaryFeatures = []models.AdversaryFeature{{
		Name:    "Momentum",
		Passive: true,
		Trigger: models.FeatureTriggerCondition{Type: models.TriggerOnHit},
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectFearGenerate, Magnitude: 1}},
	}}
	target := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{bear}, 0, 2)

	ce.executeAttack(bear, target, state)
	if state.FearPool != 3 {
		t.Errorf("FearPool = %d, want 3 (one Fear from Momentum on a hit)", state.FearPool)
	}
}

func TestBurningReactionDamagesAttacker(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 5}, &firstPicker{})
	ooze := tEnemy("e1", "Tiny Red Ooze", 10)
	ooze.AdversaryFeatures = []models.AdversaryFeature{{
		Name:    "Burning",
		Passive: true,
		Trigger: models.FeatureTriggerCondition{Type: models.TriggerOnDamageTaken},
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectDirectDamage, DiceFormula: "1d6", DamageType: "magic"}},
	}}
	hero := tPlayer("p1", "Hero")
	hero.ArmorSlots = 2
	state := newStateWith([]*models.Combatant{hero}, []*models.Combatant{ooze}, 0, 0)

	ce.executeAttack(hero, ooze, state)
	if ooze.HP == ooze.HPMax {
		t.Fatal("hero should have damaged the ooze")
	}
	// 5 direct damage is Minor for a level 3 PC, and armor cannot reduce it
	if hero.HP != hero.HPMax-1 || hero.ArmorSlots != 2 {
		t.Errorf("hero HP=%d armor=%d, want %d / 2", hero.HP, hero.ArmorSlots, hero.HPMax-1)
	}
}

func TestAttackFeatureMissesAgainstHighEvasion(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{damage: 5}) // d20 is always 15
	wolf := tEnemy("e1", "Dire Wolf", 10)
	wolf.ATK = 2
	hero := tPlayer("p1", "Hero")
	hero.Evasion = 18
	state := newStateWith([]*models.Combatant{hero}, []*models.Combatant{wolf}, 0, 0)
	feature := &models.AdversaryFeature{
		Name: "Hobbling Strike", StressCost: 1, DCType: "target",
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectDirectDamage, DiceFormula: "3d4+10"}},
	}

	res, err := fr.ActivateAdversaryFeature(feature, wolf, hero, state)
	if err == nil || res.Success {
		t.Fatalf("17 vs Evasion 18 should miss, got err=%v success=%v", err, res.Success)
	}
	if hero.HP != hero.HPMax {
		t.Errorf("hero HP = %d, want untouched on a miss", hero.HP)
	}
	// "Mark a Stress to make an attack": the Stress is spent even on a miss
	if wolf.Stress != 1 {
		t.Errorf("wolf Stress = %d, want 1", wolf.Stress)
	}
}

func TestCostOnSuccessIsNotPaidOnMiss(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{damage: 5})
	guard := tEnemy("e1", "Bladed Guard", 10)
	guard.ATK = 1
	hero := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{hero}, []*models.Combatant{guard}, 0, 0)
	feature := &models.AdversaryFeature{
		Name: "Detain", StressCost: 1, CostOnSuccess: true, DCType: "target",
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectApplyRestrained, Duration: "until_cleared"}},
	}

	hero.Evasion = 18
	fr.ActivateAdversaryFeature(feature, guard, hero, state)
	if guard.Stress != 0 || hero.HasCondition(models.ConditionRestrained) {
		t.Errorf("on a miss: Stress=%d restrained=%v, want 0 / false", guard.Stress, hero.HasCondition(models.ConditionRestrained))
	}

	hero.Evasion = 10
	if _, err := fr.ActivateAdversaryFeature(feature, guard, hero, state); err != nil {
		t.Fatal(err)
	}
	if guard.Stress != 1 || !hero.HasCondition(models.ConditionRestrained) {
		t.Errorf("on a hit: Stress=%d restrained=%v, want 1 / true", guard.Stress, hero.HasCondition(models.ConditionRestrained))
	}
}

func TestFeatureAttackCriticalAddsMaxDice(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{damage: 5, d20: 20})
	wolf := tEnemy("e1", "Dire Wolf", 10)
	hero := tPlayer("p1", "Hero")
	hero.Evasion = 30 // only the natural 20 can hit
	state := newStateWith([]*models.Combatant{hero}, []*models.Combatant{wolf}, 0, 0)
	feature := &models.AdversaryFeature{
		Name: "Hobbling Strike", DCType: "target",
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectDirectDamage, DiceFormula: "3d4+10"}},
	}

	res, err := fr.ActivateAdversaryFeature(feature, wolf, hero, state)
	if err != nil || !res.IsCritical {
		t.Fatalf("natural 20 should crit, err=%v crit=%v", err, res.IsCritical)
	}
	// rolled 5 + flat 10, plus the 12 maximum of 3d4
	if res.DamageRoll != 27 {
		t.Errorf("DamageRoll = %d, want 27", res.DamageRoll)
	}
}

func TestAbilityCriticalAddsMaxDice(t *testing.T) {
	crit := DualityRollResult{HopeDie: 6, FearDie: 6, IsCritical: true, Result: models.ResultCriticalHope, WithSpirit: 12}
	fr := newTestFR(&mockDiceRoller{damage: 5, dualityQueue: []DualityRollResult{crit}})
	hero := tPlayer("p1", "Hero")
	ogre := tEnemy("e1", "Ogre", 20)
	state := newStateWith([]*models.Combatant{hero}, []*models.Combatant{ogre}, 2, 0)
	ability := &models.DomainAbility{
		Name: "Blast", DCType: "target",
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectDamage, DiceFormula: "2d8+1", DamageType: "magic"}},
	}

	res, err := fr.ActivateDomainAbility(ability, hero, ogre, state)
	if err != nil || !res.IsCritical {
		t.Fatalf("matching dice should crit, err=%v crit=%v", err, res.IsCritical)
	}
	// rolled 5 + flat 1, plus the 16 maximum of 2d8
	if res.DamageRoll != 22 {
		t.Errorf("DamageRoll = %d, want 22", res.DamageRoll)
	}
}

func TestDamageEffectArmorCost(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	actor := tEnemy("e1", "Tormentor", 10)
	target := tPlayer("p1", "Hero")
	target.CustomThresholds = []int{4, 7}
	target.ArmorSlots = 3
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 0)

	effect := models.FeatureEffect{Category: models.FeatureEffectDamage, Magnitude: 9, ArmorCost: 2}
	if err := fr.ResolveEffect(effect, actor, target, state, nil); err != nil {
		t.Fatal(err)
	}
	if target.ArmorSlots != 1 {
		t.Errorf("ArmorSlots = %d, want 1 (this damage costs 2 slots to reduce)", target.ArmorSlots)
	}
}

func TestAreaArmorDamageFallbackGrantsFear(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	burrower := tEnemy("e1", "Acid Burrower", 10)
	armored := tPlayer("p1", "Knight")
	armored.ArmorSlots = 1
	bare := tPlayer("p2", "Mage")
	state := newStateWith([]*models.Combatant{armored, bare}, []*models.Combatant{burrower}, 0, 0)

	effect := models.FeatureEffect{
		Category:     models.FeatureEffectArmorDamage,
		Magnitude:    1,
		TargetType:   models.FeatureTargetAOE,
		FallbackFear: 1,
	}
	if err := fr.ResolveEffect(effect, burrower, armored, state, nil); err != nil {
		t.Fatal(err)
	}

	if armored.ArmorSlots != 0 || armored.HP != armored.HPMax {
		t.Errorf("armored: ArmorSlots=%d HP=%d, want 0 / %d", armored.ArmorSlots, armored.HP, armored.HPMax)
	}
	if bare.HP != bare.HPMax-1 {
		t.Errorf("bare: HP=%d, want %d (no slot to mark)", bare.HP, bare.HPMax-1)
	}
	if state.FearPool != 1 {
		t.Errorf("FearPool = %d, want 1 (one target could not mark a slot)", state.FearPool)
	}
}

func TestAttackerMarksArmorAfterHittingTheSnake(t *testing.T) {
	// Zero damage: the hit alone triggers the shards
	ce := newMockEngine(&mockDiceRoller{damage: 0}, &firstPicker{})
	snake := tEnemy("e1", "Glass Snake", 10)
	snake.AdversaryFeatures = []models.AdversaryFeature{{
		Name:    "Armor-Shredding Shards",
		Passive: true,
		Trigger: models.FeatureTriggerCondition{Type: models.TriggerOnHitTaken},
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectArmorDamage, Magnitude: 1}},
	}}
	attacker := tPlayer("p1", "Hero")
	attacker.ArmorSlots = 1
	state := newStateWith([]*models.Combatant{attacker}, []*models.Combatant{snake}, 0, 0)

	ce.executeAttack(attacker, snake, state)
	if attacker.ArmorSlots != 0 || attacker.HP != attacker.HPMax {
		t.Errorf("first hit: ArmorSlots=%d HP=%d, want 0 / %d", attacker.ArmorSlots, attacker.HP, attacker.HPMax)
	}

	ce.executeAttack(attacker, snake, state)
	if attacker.HP != attacker.HPMax-1 {
		t.Errorf("second hit: HP=%d, want %d", attacker.HP, attacker.HPMax-1)
	}
	if snake.HP != snake.HPMax {
		t.Errorf("snake HP=%d, want %d (the attacks dealt no damage)", snake.HP, snake.HPMax)
	}
}

func TestAreaAttackComparesRollToEachTarget(t *testing.T) {
	// Mock d20 is 15, +5 ATK = 20
	fr := newTestFR(&mockDiceRoller{damage: 9})
	actor := tEnemy("e1", "Warlord", 10)
	nimble := tPlayer("p1", "Rogue")
	nimble.Evasion = 25
	slow := tPlayer("p2", "Knight")
	slow.Evasion = 12
	state := newStateWith([]*models.Combatant{nimble, slow}, []*models.Combatant{actor}, 0, 0)

	feature := &models.AdversaryFeature{
		Name:    "Sweep",
		DCType:  "target",
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectAreaDamage, DiceFormula: "1d8"}},
	}

	// The chosen target is the one the roll misses; the other is still hit
	if _, err := fr.ActivateAdversaryFeature(feature, actor, nimble, state); err != nil {
		t.Fatalf("should succeed against the target it beats: %v", err)
	}
	if nimble.HP != nimble.HPMax {
		t.Errorf("nimble HP=%d, want %d (20 misses Evasion 25)", nimble.HP, nimble.HPMax)
	}
	if slow.HP == slow.HPMax {
		t.Error("slow should be hit (20 beats Evasion 12)")
	}

	slow.Evasion = 25
	hpBefore := slow.HP
	if _, err := fr.ActivateAdversaryFeature(feature, actor, nimble, state); err == nil {
		t.Error("should fail when the roll beats nobody")
	}
	if slow.HP != hpBefore {
		t.Error("nobody should take damage on a total miss")
	}
}

func TestStandardAttackEffectUsesActorDamageDice(t *testing.T) {
	// Mock dice roll 3 per term, so "2d6+6" totals 9 and "1d4" would total 3
	fr := newTestFR(&mockDiceRoller{damage: 3})
	actor := tEnemy("e1", "Warlord", 10)
	actor.DamageDice = "2d6+6"
	target := tPlayer("p1", "Hero")
	target.CustomThresholds = []int{4, 7}
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 0)

	result := &DCCheckResult{Success: true}
	effect := models.FeatureEffect{Category: models.FeatureEffectDamage, StandardAttack: true, DiceFormula: "1d4"}
	if err := fr.ResolveEffect(effect, actor, target, state, result); err != nil {
		t.Fatal(err)
	}
	if result.DamageRoll != 9 {
		t.Errorf("DamageRoll = %d, want 9 (the actor's standard attack, not the effect formula)", result.DamageRoll)
	}
}

func TestAreaEffectReachesLimitedTargets(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{damage: 9})
	fr.SetAreaTargetLimit(2)
	actor := tEnemy("e1", "Warlord", 10)
	players := []*models.Combatant{tPlayer("p1", "A"), tPlayer("p2", "B"), tPlayer("p3", "C"), tPlayer("p4", "D")}
	state := newStateWith(players, []*models.Combatant{actor}, 0, 0)

	feature := &models.AdversaryFeature{
		Name: "Spit Acid",
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectArmorDamage, Magnitude: 1, TargetType: models.FeatureTargetAOE},
			{Category: models.FeatureEffectAreaDamage, DiceFormula: "1d8"},
		},
	}
	chosen := players[2]
	if _, err := fr.ActivateAdversaryFeature(feature, actor, chosen, state); err != nil {
		t.Fatal(err)
	}

	hit := 0
	for _, p := range players {
		switch p.HPMax - p.HP {
		case 0:
		case 4:
			// 1 HP for the missing Armor Slot plus 3 HP of Severe damage: both effects, same targets
			hit++
		default:
			t.Errorf("%s lost %d HP, want 0 or 4 (effects must agree on who is in the area)", p.Name, p.HPMax-p.HP)
		}
	}
	if hit != 2 {
		t.Errorf("%d targets hit, want 2", hit)
	}
	if chosen.HP == chosen.HPMax {
		t.Error("the chosen target is always inside the area")
	}
}

func hellfire() *models.AdversaryFeature {
	return &models.AdversaryFeature{
		Name: "Hellfire",
		Effects: []models.FeatureEffect{{
			Category:      models.FeatureEffectAreaDamage,
			DiceFormula:   "1d20",
			ReactionTrait: "agility",
			HalfOnSuccess: true,
		}},
	}
}

func TestReactionRollHalvesOrNegatesDamage(t *testing.T) {
	cases := []struct {
		name    string
		roll    DualityRollResult
		half    bool
		wantHit int // raw damage 9 against thresholds 4/7
	}{
		{"failure takes it all", hopeDuality(5), true, 3},
		{"success takes half, rounded up", hopeDuality(20), true, 2},
		{"success takes nothing when not a half effect", hopeDuality(20), false, 0},
		{"critical ignores the effect", DualityRollResult{HopeDie: 3, FearDie: 3, IsCritical: true, WithSpirit: 6}, true, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fr := newTestFR(&mockDiceRoller{damage: 9, dualityQueue: []DualityRollResult{c.roll}})
			actor := tEnemy("e1", "Demon", 10)
			actor.Difficulty = 14
			target := tPlayer("p1", "Hero")
			target.CustomThresholds = []int{4, 7}
			state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 2, 3)

			feature := hellfire()
			feature.Effects[0].HalfOnSuccess = c.half
			if _, err := fr.ActivateAdversaryFeature(feature, actor, target, state); err != nil {
				t.Fatal(err)
			}

			if lost := target.HPMax - target.HP; lost != c.wantHit {
				t.Errorf("lost %d HP, want %d", lost, c.wantHit)
			}
			if target.Hope != 2 || state.FearPool != 3 {
				t.Errorf("Hope=%d Fear=%d, want 2 / 3 (reaction rolls generate neither)", target.Hope, state.FearPool)
			}
		})
	}
}

func TestReactionRollUsesTheNamedTrait(t *testing.T) {
	dice := &traitRecorder{mockDiceRoller: &mockDiceRoller{damage: 9}}
	fr := newTestFR(dice.mockDiceRoller)
	fr.dice = dice
	actor := tEnemy("e1", "Demon", 10)
	target := tPlayer("p1", "Hero")
	target.Traits = models.Traits{Agility: 2, Instinct: -1}
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 0)

	if _, err := fr.ActivateAdversaryFeature(hellfire(), actor, target, state); err != nil {
		t.Fatal(err)
	}
	if dice.lastModifier != 2 {
		t.Errorf("reaction roll modifier = %d, want 2 (Agility)", dice.lastModifier)
	}
}

type traitRecorder struct {
	*mockDiceRoller
	lastModifier int
}

func (r *traitRecorder) RollDuality(hopeModifier, fearModifier int) DualityRollResult {
	r.lastModifier = hopeModifier
	return r.mockDiceRoller.RollDuality(hopeModifier, fearModifier)
}

func TestTargetRollsOneReactionPerActivation(t *testing.T) {
	// First roll fails; a second roll would succeed and wrongly dodge the condition
	dice := &mockDiceRoller{damage: 9, dualityQueue: []DualityRollResult{hopeDuality(5)}}
	fr := newTestFR(dice)
	actor := tEnemy("e1", "Burrower", 10)
	actor.Difficulty = 14
	target := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{target}, []*models.Combatant{actor}, 0, 0)

	feature := &models.AdversaryFeature{
		Name: "Eruption",
		Effects: []models.FeatureEffect{
			{Category: models.FeatureEffectAreaDamage, DiceFormula: "1d8", ReactionTrait: "agility"},
			{Category: models.FeatureEffectApplyVulnerable, Duration: "until_cleared", TargetType: models.FeatureTargetAOE, ReactionTrait: "agility"},
		},
	}
	if _, err := fr.ActivateAdversaryFeature(feature, actor, target, state); err != nil {
		t.Fatal(err)
	}

	if target.HP == target.HPMax || !target.HasCondition(models.ConditionVulnerable) {
		t.Errorf("HP=%d vulnerable=%v, want damage and Vulnerable from the one failed roll", target.HP, target.HasCondition(models.ConditionVulnerable))
	}
	if dice.dualityIdx != 1 {
		t.Errorf("rolled %d reaction rolls, want 1", dice.dualityIdx)
	}
}
