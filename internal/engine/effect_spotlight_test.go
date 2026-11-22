package engine

import (
	"strings"
	"testing"

	"github.com/skarlso/combat-simulator/internal/models"
	"github.com/skarlso/combat-simulator/internal/strategy"
)

// featureDM uses an adversary's first active feature when it has one, else a basic attack
type featureDM struct{}

func (featureDM) ChooseAction(state *models.CombatState, actor *models.Combatant) *strategy.ActionDecision {
	enemies := state.GetEnemiesOf(actor)
	if len(enemies) == 0 {
		return &strategy.ActionDecision{}
	}
	d := &strategy.ActionDecision{Target: enemies[0]}
	for i := range actor.AdversaryFeatures {
		if !actor.AdversaryFeatures[i].Passive {
			d.AdversaryFeature = &actor.AdversaryFeatures[i]
			break
		}
	}
	return d
}

func rallyFeature(effect models.FeatureEffect) models.AdversaryFeature {
	return models.AdversaryFeature{Name: "Rally", Effects: []models.FeatureEffect{effect}}
}

func leaderWithGuards(effect models.FeatureEffect) (*models.Combatant, *models.Combatant, []*models.Combatant, *models.CombatState) {
	hero := tPlayer("p1", "Hero")
	leader := tEnemy("e1", "Captain", 10)
	leader.AdversaryFeatures = []models.AdversaryFeature{rallyFeature(effect)}
	guards := []*models.Combatant{tEnemy("e2", "Guard A", 10), tEnemy("e3", "Guard B", 10)}
	state := newStateWith([]*models.Combatant{hero}, append([]*models.Combatant{leader}, guards...), 0, 0)
	return hero, leader, guards, state
}

func TestAllySpotlightLetsAlliesActInTheSameGMTurn(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 5, coinFlip: 1}, &firstPicker{})
	hero, leader, guards, state := leaderWithGuards(models.FeatureEffect{Category: models.FeatureEffectAllySpotlight, Magnitude: 2})
	state.BeginGMTurn()

	ce.executeAdversarySpotlight(leader, state, featureDM{})
	if len(state.PendingSpotlights) != 2 {
		t.Fatalf("queued %d spotlights, want 2", len(state.PendingSpotlights))
	}

	turns := 0
	ce.drainPendingSpotlights(state, featureDM{}, &turns, 10)

	if turns != 2 || len(state.PendingSpotlights) != 0 {
		t.Errorf("turns=%d pending=%d, want 2 / 0", turns, len(state.PendingSpotlights))
	}
	if hero.HP != 8 {
		t.Errorf("hero HP = %d, want 8 (one Minor hit from each guard)", hero.HP)
	}
	for _, g := range guards {
		if state.AdversarySpotlightCount(g.ID) != 1 {
			t.Errorf("%s spotlight count = %d, want 1", g.Name, state.AdversarySpotlightCount(g.ID))
		}
	}
	var spotlightLog string
	for _, a := range state.ActionHistory {
		if strings.HasPrefix(a.Result, "Spotlights ") {
			spotlightLog = a.Result
		}
	}
	if !strings.Contains(spotlightLog, "Guard A") || !strings.Contains(spotlightLog, "Guard B") {
		t.Errorf("log = %q, want both guards named", spotlightLog)
	}
}

func TestAllySpotlightHalfDamageOnlyLastsForThatSpotlight(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 8, coinFlip: 1}, &firstPicker{})
	_, leader, guards, state := leaderWithGuards(models.FeatureEffect{Category: models.FeatureEffectAllySpotlight, SpotlightAll: true, HalfDamage: true})
	state.BeginGMTurn()

	ce.executeAdversarySpotlight(leader, state, featureDM{})
	turns := 0
	ce.drainPendingSpotlights(state, featureDM{}, &turns, 10)

	attacks := 0
	for _, a := range state.ActionHistory {
		if a.ActionType == models.ActionAttack && strings.HasPrefix(a.Actor, "Guard") {
			attacks++
			if a.DamageRoll != 4 {
				t.Errorf("%s rolled %d damage, want 4 (8 halved)", a.Actor, a.DamageRoll)
			}
		}
	}
	if attacks != 2 {
		t.Errorf("guard attacks = %d, want 2", attacks)
	}
	for _, g := range guards {
		if g.HalfDamage {
			t.Errorf("%s still flagged HalfDamage after acting", g.Name)
		}
	}
}

func TestAllySpotlightNeedsSomeoneToSpotlight(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{}, &firstPicker{})
	leader := tEnemy("e1", "Captain", 10)
	leader.AdversaryFeatures = []models.AdversaryFeature{rallyFeature(models.FeatureEffect{Category: models.FeatureEffectAllySpotlight, Magnitude: 2})}
	state := newStateWith([]*models.Combatant{tPlayer("p1", "Hero")}, []*models.Combatant{leader}, 0, 0)

	if ce.featureResolver.CanActivateAdversaryFeature(&leader.AdversaryFeatures[0], leader, state) {
		t.Error("a lone leader should not be able to spotlight allies")
	}
}

func TestAllySpotlightSkipsAlliesAlreadySpotlightedThisGMTurn(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 5, coinFlip: 1}, &firstPicker{})
	_, leader, guards, state := leaderWithGuards(models.FeatureEffect{Category: models.FeatureEffectAllySpotlight, SpotlightAll: true})
	state.BeginGMTurn()
	state.RecordAdversarySpotlight(guards[0].ID)

	ce.executeAdversarySpotlight(leader, state, featureDM{})

	if len(state.PendingSpotlights) != 1 || state.PendingSpotlights[0].Combatant != guards[1] {
		t.Errorf("pending = %+v, want only Guard B", state.PendingSpotlights)
	}
}

func TestSummonedAlliesCanTakeTheSpotlightAtOnce(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{damage: 1, coinFlip: 1})
	caller := tEnemy("e1", "Necromancer", 10)
	state := newStateWith([]*models.Combatant{tPlayer("p1", "Hero")}, []*models.Combatant{caller}, 0, 0)
	template := tEnemy("tpl", "Zombie", 5)

	effect := models.FeatureEffect{
		Category:       models.FeatureEffectSummonAllies,
		Magnitude:      2,
		SummonPool:     []*models.Combatant{template},
		SpotlightCount: 1,
	}
	if err := fr.ResolveEffect(effect, caller, nil, state, nil); err != nil {
		t.Fatal(err)
	}

	if len(state.Enemies) != 3 {
		t.Fatalf("enemies = %d, want 3", len(state.Enemies))
	}
	if len(state.PendingSpotlights) != 1 || !strings.HasPrefix(state.PendingSpotlights[0].Combatant.Name, "Zombie #") {
		t.Errorf("pending = %+v, want one summoned Zombie", state.PendingSpotlights)
	}
}

func TestRunCombatChainsSpotlightsInsideTheGMTurn(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 5, coinFlip: 1}, &firstPicker{})
	_, _, _, state := leaderWithGuards(models.FeatureEffect{Category: models.FeatureEffectAllySpotlight, SpotlightAll: true})
	state.FearPool = 1

	ce.RunCombat(state, featureDM{}, 8, strategy.ResourceGreedy)

	// Expect: seize spotlight, Captain's Rally, then both guards attack within that GM turn
	seen := ""
	for _, a := range state.ActionHistory {
		switch {
		case strings.HasPrefix(a.Result, "Spends 1 Fear"):
			seen += "S"
		case strings.HasPrefix(a.Result, "Spotlights "):
			seen += "R"
		case a.ActionType == models.ActionAttack && strings.HasPrefix(a.Actor, "Guard"):
			seen += "G"
		}
	}
	if !strings.Contains(seen, "SRGG") {
		t.Errorf("action sequence %q, want a seize, rally, two guard attacks in order", seen)
	}
}
