package engine

import (
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
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

	err, _ := fr.ActivateAdversaryFeature(&actor.AdversaryFeatures[0], actor, target, state)
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

	err, _ := fr.ActivateAdversaryFeature(feat, actor, target, state)
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

	err, _ := fr.ActivateAdversaryFeature(feat, actor, target, state)
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

	err, _ := fr.ActivateDomainAbility(ability, actor, target, state)
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
		if err, _ := fr.ActivateDomainAbility(ability, actor, target, state); err != nil {
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

	if err, _ := fr.ActivateDomainAbility(release, actor, target, state); err != nil {
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
	actor := tPlayer("p1", "Hero")
	state := newStateWith([]*models.Combatant{actor}, nil, 4, 1)

	if err := fr.ResolveEffect(models.FeatureEffect{Category: models.FeatureEffectHopeDrain, Magnitude: 2}, actor, nil, state, nil); err != nil {
		t.Fatalf("hope drain: %v", err)
	}
	if state.HopePool != 2 {
		t.Errorf("HopePool = %d, want 2 (4 - 2)", state.HopePool)
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
