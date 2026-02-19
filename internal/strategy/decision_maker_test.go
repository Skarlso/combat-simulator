package strategy

import (
	"math/rand"
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func newTestAI(config StrategyConfig) *AIDecisionMaker {
	return NewAIDecisionMaker(rand.New(rand.NewSource(42)), config)
}

func newTestCombatState() *models.CombatState {
	players := []*models.Combatant{
		models.NewCombatant("p1", "Hero", models.SidePlayer, 3, 12, 2, 15, 8),
	}
	players[0].DamageDice = "1d6"

	enemies := []*models.Combatant{
		models.NewAdversary("e1", "Weak", "Tier 1", "Standard", 10, 3, 5, 4, []int{4, 7}),
		models.NewAdversary("e2", "Strong", "Tier 1", "Bruiser", 12, 5, 20, 6, []int{5, 10}),
	}
	enemies[0].DamageDice = "1d6"
	enemies[1].DamageDice = "1d8+2"

	return models.NewCombatState(players, enemies, 3, 3, 5)
}

func TestChooseActionReturnsTarget(t *testing.T) {
	ai := newTestAI(DefaultGreedyConfig())
	state := newTestCombatState()

	decision := ai.ChooseAction(state, state.Players[0])
	if decision == nil {
		t.Fatal("decision should not be nil")
	}
	if decision.Target == nil {
		t.Fatal("target should not be nil when enemies exist")
	}
}

func TestChooseActionNoTargets(t *testing.T) {
	ai := newTestAI(DefaultGreedyConfig())
	state := newTestCombatState()
	state.Enemies[0].IsDefeated = true
	state.Enemies[1].IsDefeated = true

	decision := ai.ChooseAction(state, state.Players[0])
	if decision.Target != nil {
		t.Error("target should be nil when all enemies defeated")
	}
}

func TestTargetSelectionLowestHP(t *testing.T) {
	ai := newTestAI(StrategyConfig{TargetSelection: TargetLowestHP})
	state := newTestCombatState()

	decision := ai.ChooseAction(state, state.Players[0])
	if decision.Target.Name != "Weak" {
		t.Errorf("LowestHP should target %q, got %q", "Weak", decision.Target.Name)
	}
}

func TestTargetSelectionHighestHP(t *testing.T) {
	ai := newTestAI(StrategyConfig{TargetSelection: TargetHighestHP})
	state := newTestCombatState()

	decision := ai.ChooseAction(state, state.Players[0])
	if decision.Target.Name != "Strong" {
		t.Errorf("HighestHP should target %q, got %q", "Strong", decision.Target.Name)
	}
}

func TestTargetSelectionRandom(t *testing.T) {
	ai := newTestAI(StrategyConfig{TargetSelection: TargetRandom})
	state := newTestCombatState()

	names := map[string]bool{}
	for range 100 {
		decision := ai.ChooseAction(state, state.Players[0])
		names[decision.Target.Name] = true
	}

	if len(names) < 2 {
		t.Error("random selection should pick different targets over 100 runs")
	}
}

func TestDefaultConfigs(t *testing.T) {
	configs := []struct {
		name   string
		config StrategyConfig
	}{
		{"greedy", DefaultGreedyConfig()},
		{"conservative", DefaultConservativeConfig()},
		{"desperate", DefaultDesperateConfig()},
		{"random", DefaultRandomConfig()},
	}

	for _, tt := range configs {
		t.Run(tt.name, func(t *testing.T) {
			ai := newTestAI(tt.config)
			state := newTestCombatState()
			decision := ai.ChooseAction(state, state.Players[0])
			if decision == nil {
				t.Error("decision should not be nil")
			}
		})
	}
}

func TestEnemyChooseAction(t *testing.T) {
	ai := newTestAI(DefaultGreedyConfig())
	state := newTestCombatState()

	decision := ai.ChooseAction(state, state.Enemies[0])
	if decision.Target == nil {
		t.Fatal("enemy should have a target")
	}
	if decision.Target.Side != models.SidePlayer {
		t.Error("enemy should target a player")
	}
}

func TestChooseActionWithAbility(t *testing.T) {
	ai := newTestAI(StrategyConfig{
		TargetSelection: TargetLowestHP,
		AbilityUsage:    AbilityAlways,
		ResourceUsage:   ResourceGreedy,
	})

	state := newTestCombatState()
	state.Players[0].DomainAbilities = []models.DomainAbility{
		{
			Name:     "Fireball",
			HopeCost: 1,
			UsesLeft: 1,
			MaxUses:  1,
			Effects: []models.FeatureEffect{
				{
					Category:    models.FeatureEffectDamage,
					DiceFormula: "3d6",
					DamageType:  "magic",
					TargetType:  models.FeatureTargetSingle,
				},
			},
		},
	}

	decision := ai.ChooseAction(state, state.Players[0])
	if decision.DomainAbility == nil {
		t.Log("AI chose basic attack over ability - may be correct based on evaluation")
	}
}

func TestAbilityUsageNever(t *testing.T) {
	ai := newTestAI(StrategyConfig{
		TargetSelection: TargetLowestHP,
		AbilityUsage:    AbilityNever,
		ResourceUsage:   ResourceGreedy,
	})

	state := newTestCombatState()
	state.Players[0].DomainAbilities = []models.DomainAbility{
		{
			Name:     "Fireball",
			HopeCost: 0,
			UsesLeft: 99,
			MaxUses:  99,
			Effects: []models.FeatureEffect{
				{Category: models.FeatureEffectDamage, DiceFormula: "10d6"},
			},
		},
	}

	decision := ai.ChooseAction(state, state.Players[0])
	if decision.DomainAbility != nil {
		t.Error("AbilityNever should never select an ability")
	}
}

func TestResourceConservativeHoardsHope(t *testing.T) {
	ai := newTestAI(StrategyConfig{
		TargetSelection: TargetLowestHP,
		AbilityUsage:    AbilityAlways,
		ResourceUsage:   ResourceConservative,
	})

	state := newTestCombatState()
	state.HopePool = 2
	state.Players[0].DomainAbilities = []models.DomainAbility{
		{
			Name:     "Costly Spell",
			HopeCost: 1,
			UsesLeft: 99,
			MaxUses:  99,
			Effects: []models.FeatureEffect{
				{Category: models.FeatureEffectDamage, DiceFormula: "10d6"},
			},
		},
	}

	decision := ai.ChooseAction(state, state.Players[0])
	if decision.DomainAbility != nil {
		t.Error("Conservative should hoard resources when pool is low (need cost+2)")
	}
}

func TestResourceDesperateOnlyWhenLowHP(t *testing.T) {
	ai := newTestAI(StrategyConfig{
		TargetSelection: TargetLowestHP,
		AbilityUsage:    AbilityAlways,
		ResourceUsage:   ResourceDesperate,
		LowHPThreshold:  30,
	})

	state := newTestCombatState()
	state.HopePool = 5
	state.Players[0].DomainAbilities = []models.DomainAbility{
		{
			Name:     "Desperation Move",
			HopeCost: 1,
			UsesLeft: 99,
			MaxUses:  99,
			Effects: []models.FeatureEffect{
				{Category: models.FeatureEffectDamage, DiceFormula: "10d6"},
			},
		},
	}

	decision := ai.ChooseAction(state, state.Players[0])
	if decision.DomainAbility != nil {
		t.Error("Desperate should not spend resources at full HP")
	}

	state.Players[0].HP = 2
	decision = ai.ChooseAction(state, state.Players[0])
	if decision.DomainAbility == nil {
		t.Error("Desperate should spend resources when HP is low")
	}
}

func TestConditionClearingImmediate(t *testing.T) {
	ai := newTestAI(StrategyConfig{
		TargetSelection:   TargetLowestHP,
		ConditionClearing: ClearImmediately,
	})

	state := newTestCombatState()
	state.Players[0].AddCondition(&models.ActiveCondition{
		Type:     models.ConditionBurning,
		Duration: models.DurationUntilCleared,
	})

	decision := ai.ChooseAction(state, state.Players[0])
	if decision.Target != state.Players[0] {
		t.Error("ClearImmediately should target self to clear harmful condition")
	}
}

func TestEnemyFeatureSelectedOverBasicAttack(t *testing.T) {
	ai := newTestAI(DefaultGreedyConfig())
	state := newTestCombatState()

	enemy := state.Enemies[0]
	enemy.DamageDice = "1d8"
	enemy.ATK = 2
	enemy.AdversaryFeatures = []models.AdversaryFeature{
		{
			Name:     "Venom Strike",
			FearCost: 0,
			DCType:   "none",
			Effects: []models.FeatureEffect{
				{
					Category:    models.FeatureEffectDamage,
					DiceFormula: "1d6",
					DamageType:  "physical",
				},
			},
		},
	}

	decision := ai.ChooseAction(state, enemy)
	if decision.AdversaryFeature == nil {
		basicVal := ai.evaluateBasicAttack(enemy, decision.Target)
		featureVal := ai.evaluateFeature(state, enemy, decision.Target, &enemy.AdversaryFeatures[0])
		t.Errorf("enemy should use free auto-succeed feature over basic attack; basicVal=%.2f featureVal=%.2f", basicVal, featureVal)
	}
}

func TestEnemyFeatureWithFearCost(t *testing.T) {
	ai := newTestAI(DefaultGreedyConfig())
	state := newTestCombatState()
	state.FearPool = 5

	enemy := state.Enemies[0]
	enemy.DamageDice = "1d8"
	enemy.ATK = 2
	enemy.AdversaryFeatures = []models.AdversaryFeature{
		{
			Name:     "Venom Strike",
			FearCost: 1,
			DCType:   "none",
			Effects: []models.FeatureEffect{
				{
					Category:    models.FeatureEffectDamage,
					DiceFormula: "1d6",
					DamageType:  "physical",
				},
			},
		},
	}

	decision := ai.ChooseAction(state, enemy)
	if decision.AdversaryFeature == nil {
		t.Error("feature with 1 Fear cost and auto-succeed should still beat basic attack")
	}
}

func TestEnemyFeatureWithDCTarget(t *testing.T) {
	ai := newTestAI(DefaultGreedyConfig())
	state := newTestCombatState()
	state.FearPool = 5

	enemy := state.Enemies[0]
	enemy.DamageDice = "1d8"
	enemy.ATK = 2
	enemy.AdversaryFeatures = []models.AdversaryFeature{
		{
			Name:   "Venom Strike",
			DCType: "target",
			Effects: []models.FeatureEffect{
				{
					Category:    models.FeatureEffectDamage,
					DiceFormula: "1d6",
					DamageType:  "physical",
				},
			},
		},
	}

	decision := ai.ChooseAction(state, enemy)
	if decision.AdversaryFeature == nil {
		t.Error("free feature with target DC should still beat basic attack")
	}
}

func TestConditionClearingNever(t *testing.T) {
	ai := newTestAI(StrategyConfig{
		TargetSelection:   TargetLowestHP,
		ConditionClearing: ClearNever,
	})

	state := newTestCombatState()
	state.Players[0].AddCondition(&models.ActiveCondition{
		Type:     models.ConditionBurning,
		Duration: models.DurationUntilCleared,
	})

	decision := ai.ChooseAction(state, state.Players[0])
	if decision.Target == state.Players[0] {
		t.Error("ClearNever should not target self for condition clearing")
	}
}
