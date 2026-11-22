package engine

import (
	"math/rand"
	"slices"
	"strings"
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)

type mockDiceRoller struct {
	dualityQueue []DualityRollResult
	dualityIdx   int
	damage       int
	coinFlip     int
	d20          int // 0 = 15
}

func (m *mockDiceRoller) RollDuality(_, _ int) DualityRollResult {
	if m.dualityIdx < len(m.dualityQueue) {
		r := m.dualityQueue[m.dualityIdx]
		m.dualityIdx++
		return r
	}
	return hopeDuality(20)
}

func (m *mockDiceRoller) RollD20Attack(modifier int) D20RollResult {
	roll := 15
	if m.d20 > 0 {
		roll = m.d20
	}
	return D20RollResult{D20Roll: roll, Modifier: modifier, Total: roll + modifier, IsCritical: roll == 20}
}

func (m *mockDiceRoller) RollDiceDetailed(_, _, _ int) DamageRollResult {
	return DamageRollResult{DiceValues: []int{m.damage}, Total: m.damage}
}

func (m *mockDiceRoller) RollDice(_, diceSize, _ int) int {
	if diceSize == 2 {
		return m.coinFlip
	}
	return m.damage
}

type mockDecisionMaker struct{}

func (m *mockDecisionMaker) ChooseAction(state *models.CombatState, actor *models.Combatant) *strategy.ActionDecision {
	enemies := state.GetEnemiesOf(actor)
	if len(enemies) == 0 {
		return &strategy.ActionDecision{}
	}
	return &strategy.ActionDecision{Target: enemies[0]}
}

type firstPicker struct{}

func (f *firstPicker) ChooseNext(c []*models.Combatant, _ int) *models.Combatant {
	if len(c) == 0 {
		return nil
	}
	return c[0]
}

type roundRobinPicker struct{ idx int }

func (r *roundRobinPicker) ChooseNext(c []*models.Combatant, _ int) *models.Combatant {
	if len(c) == 0 {
		return nil
	}
	result := c[r.idx%len(c)]
	r.idx++
	return result
}

func hopeDuality(total int) DualityRollResult {
	return DualityRollResult{
		HopeDie: 8, FearDie: 4,
		HopeTotal: 8, FearTotal: 4,
		Result: models.ResultHope, WithSpirit: total,
	}
}

func fearDuality(total int) DualityRollResult {
	return DualityRollResult{
		HopeDie: 4, FearDie: 8,
		HopeTotal: 4, FearTotal: 8,
		Result: models.ResultFear, WithSpirit: total,
	}
}

func newMockEngine(dice *mockDiceRoller, spotlight SpotlightSelector) *CombatEngine {
	fr := &FeatureResolver{dice: dice}
	return &CombatEngine{
		dice:            dice,
		spotlight:       spotlight,
		resolver:        NewCombatResolver(dice, fr),
		featureResolver: fr,
	}
}

func tPlayer(id, name string) *models.Combatant {
	return models.NewCombatant(id, name, models.SidePlayer, 3, 10, 2, 10, 8)
}

func tEnemy(id, name string, hp int) *models.Combatant {
	return models.NewAdversary(id, name, "Tier 1", "Standard", 10, 5, hp, 6, []int{4, 7})
}

func countActorActions(history []models.ActionRecord, name string) int {
	n := 0
	for _, a := range history {
		if a.Actor == name {
			n++
		}
	}
	return n
}

func TestPlayerPhaseActsUntilShift(t *testing.T) {
	dice := &mockDiceRoller{
		dualityQueue: []DualityRollResult{hopeDuality(20), fearDuality(20)},
		damage:       2,
	}
	ce := newMockEngine(dice, &roundRobinPicker{})
	dm := &mockDecisionMaker{}

	players := []*models.Combatant{tPlayer("p1", "Hero"), tPlayer("p2", "Mage")}
	enemies := []*models.Combatant{tEnemy("e1", "Goblin", 20)}
	state := models.NewCombatState(players, enemies, 0, 0, 3)

	outcome := ce.RunCombat(state, dm, 3, strategy.ResourceGreedy)

	if len(outcome.ActionHistory) < 3 {
		t.Fatalf("actions = %d, want >= 3", len(outcome.ActionHistory))
	}
	if outcome.ActionHistory[0].Actor != "Hero" {
		t.Errorf("action 0: got %q, want Hero", outcome.ActionHistory[0].Actor)
	}
	if outcome.ActionHistory[1].Actor != "Mage" {
		t.Errorf("action 1: got %q, want Mage", outcome.ActionHistory[1].Actor)
	}
	if outcome.ActionHistory[2].Actor != "Goblin" {
		t.Errorf("action 2: got %q, want Goblin", outcome.ActionHistory[2].Actor)
	}
}

func TestSpotlightTokensAccrueToWaitingPlayers(t *testing.T) {
	dice := &mockDiceRoller{dualityQueue: []DualityRollResult{hopeDuality(20), fearDuality(20)}, damage: 2}
	ce := newMockEngine(dice, &firstPicker{})
	hero, mage, bard := tPlayer("p1", "Hero"), tPlayer("p2", "Mage"), tPlayer("p3", "Bard")
	state := models.NewCombatState([]*models.Combatant{hero, mage, bard}, []*models.Combatant{tEnemy("e1", "Goblin", 50)}, 0, 0, 3)

	// Hero acts twice (Hope keeps the spotlight, then Fear passes it)
	ce.RunCombat(state, &mockDecisionMaker{}, 2, strategy.ResourceGreedy)

	if hero.SpotlightTokens != 0 {
		t.Errorf("acting player tokens = %d, want 0", hero.SpotlightTokens)
	}
	if mage.SpotlightTokens != 2 || bard.SpotlightTokens != 2 {
		t.Errorf("waiting players tokens = %d / %d, want 2 / 2", mage.SpotlightTokens, bard.SpotlightTokens)
	}
}

func TestSpotlightThresholdForcesWaitingPlayer(t *testing.T) {
	dice := &mockDiceRoller{damage: 1} // every roll is Hope: the spotlight never leaves the players
	ce := newMockEngine(dice, NewHybridSpotlightSelector(rand.New(rand.NewSource(7)), 0.2))
	players := []*models.Combatant{tPlayer("p1", "A"), tPlayer("p2", "B"), tPlayer("p3", "C"), tPlayer("p4", "D")}
	state := models.NewCombatState(players, []*models.Combatant{tEnemy("e1", "Wall", 500)}, 0, 0, 3)

	outcome := ce.RunCombat(state, &mockDecisionMaker{}, 40, strategy.ResourceGreedy)

	// With threshold 3 nobody can be skipped more than 3 player turns in a row
	lastSeen := map[string]int{}
	for i, a := range outcome.ActionHistory {
		if a.ActionType != models.ActionAttack {
			continue
		}
		for _, p := range players {
			if p.Name == a.Actor {
				lastSeen[p.Name] = i
			} else if seen, ok := lastSeen[p.Name]; ok && i-seen > 4 {
				t.Fatalf("%s waited %d player turns at action %d", p.Name, i-seen, i)
			}
		}
	}
	if len(lastSeen) != 4 {
		t.Errorf("only %d of 4 players acted", len(lastSeen))
	}
}

func TestGMTurnFirstAdversaryFree(t *testing.T) {
	dice := &mockDiceRoller{
		dualityQueue: []DualityRollResult{fearDuality(20)},
		damage:       2,
	}
	ce := newMockEngine(dice, &firstPicker{})
	dm := &mockDecisionMaker{}

	state := models.NewCombatState(
		[]*models.Combatant{tPlayer("p1", "Hero")},
		[]*models.Combatant{tEnemy("e1", "Goblin", 20)},
		0, 5, 3,
	)

	outcome := ce.RunCombat(state, dm, 2, strategy.ResourceGreedy)

	if outcome.FinalFearPool != 6 {
		t.Errorf("FinalFearPool = %d, want 6 (5 initial + 1 Fear generated, first enemy free)", outcome.FinalFearPool)
	}
}

func TestGMTurnAdditionalCostsFear(t *testing.T) {
	dice := &mockDiceRoller{
		dualityQueue: []DualityRollResult{fearDuality(20)},
		damage:       2,
	}
	ce := newMockEngine(dice, &firstPicker{})
	dm := &mockDecisionMaker{}

	state := models.NewCombatState(
		[]*models.Combatant{tPlayer("p1", "Hero")},
		[]*models.Combatant{tEnemy("e1", "Goblin", 20), tEnemy("e2", "Orc", 20)},
		0, 5, 3,
	)

	outcome := ce.RunCombat(state, dm, 3, strategy.ResourceGreedy)

	if outcome.FinalFearPool != 5 {
		t.Errorf("FinalFearPool = %d, want 5 (5+1 generated -1 additional)", outcome.FinalFearPool)
	}
	goblin := countActorActions(outcome.ActionHistory, "Goblin")
	orc := countActorActions(outcome.ActionHistory, "Orc")
	if goblin+orc != 2 {
		t.Errorf("enemy actions = %d, want 2", goblin+orc)
	}
}

func TestRelentlessMultipleSpotlights(t *testing.T) {
	dice := &mockDiceRoller{
		dualityQueue: []DualityRollResult{fearDuality(20)},
		damage:       2,
	}
	ce := newMockEngine(dice, &firstPicker{})
	dm := &mockDecisionMaker{}

	enemy := tEnemy("e1", "Goblin", 20)
	enemy.RelentlessLimit = 3
	state := models.NewCombatState(
		[]*models.Combatant{tPlayer("p1", "Hero")},
		[]*models.Combatant{enemy},
		0, 5, 3,
	)

	outcome := ce.RunCombat(state, dm, 4, strategy.ResourceGreedy)

	if outcome.FinalFearPool != 4 {
		t.Errorf("FinalFearPool = %d, want 4 (5+1 generated -2 additional)", outcome.FinalFearPool)
	}
	if n := countActorActions(outcome.ActionHistory, "Goblin"); n != 3 {
		t.Errorf("Goblin actions = %d, want 3", n)
	}
}

func TestRelentlessLimitRespected(t *testing.T) {
	dice := &mockDiceRoller{
		dualityQueue: []DualityRollResult{fearDuality(20)},
		damage:       2,
	}
	ce := newMockEngine(dice, &firstPicker{})
	dm := &mockDecisionMaker{}

	enemy := tEnemy("e1", "Goblin", 20)
	enemy.RelentlessLimit = 2
	state := models.NewCombatState(
		[]*models.Combatant{tPlayer("p1", "Hero")},
		[]*models.Combatant{enemy},
		0, 10, 3,
	)

	outcome := ce.RunCombat(state, dm, 3, strategy.ResourceGreedy)

	if outcome.FinalFearPool != 10 {
		t.Errorf("FinalFearPool = %d, want 10 (10+1 generated -1 additional)", outcome.FinalFearPool)
	}
	if n := countActorActions(outcome.ActionHistory, "Goblin"); n != 2 {
		t.Errorf("Goblin actions = %d, want 2 (Relentless limit)", n)
	}
}

func TestNoRelentlessActsOnce(t *testing.T) {
	dice := &mockDiceRoller{
		dualityQueue: []DualityRollResult{fearDuality(20)},
		damage:       2,
	}
	ce := newMockEngine(dice, &firstPicker{})
	dm := &mockDecisionMaker{}

	state := models.NewCombatState(
		[]*models.Combatant{tPlayer("p1", "Hero")},
		[]*models.Combatant{tEnemy("e1", "Goblin", 20)},
		0, 5, 3,
	)

	outcome := ce.RunCombat(state, dm, 2, strategy.ResourceGreedy)

	if outcome.FinalFearPool != 6 {
		t.Errorf("FinalFearPool = %d, want 6", outcome.FinalFearPool)
	}
	if n := countActorActions(outcome.ActionHistory, "Goblin"); n != 1 {
		t.Errorf("Goblin actions = %d, want 1", n)
	}
}

func TestSpotlightAlwaysReturnsToPlayers(t *testing.T) {
	dice := &mockDiceRoller{
		dualityQueue: []DualityRollResult{fearDuality(20), fearDuality(20)},
		damage:       2,
	}
	ce := newMockEngine(dice, &firstPicker{})
	dm := &mockDecisionMaker{}

	state := models.NewCombatState(
		[]*models.Combatant{tPlayer("p1", "Hero")},
		[]*models.Combatant{tEnemy("e1", "Goblin", 20)},
		0, 0, 3,
	)

	outcome := ce.RunCombat(state, dm, 4, strategy.ResourceGreedy)

	expected := []string{"Hero", "Goblin", "Hero", "Goblin"}
	if len(outcome.ActionHistory) < len(expected) {
		t.Fatalf("actions = %d, want >= %d", len(outcome.ActionHistory), len(expected))
	}
	for i, name := range expected {
		if outcome.ActionHistory[i].Actor != name {
			t.Errorf("action %d: got %q, want %q", i, outcome.ActionHistory[i].Actor, name)
		}
	}
}

func TestGMSeizeGreedy(t *testing.T) {
	dice := &mockDiceRoller{
		dualityQueue: []DualityRollResult{hopeDuality(20), hopeDuality(20), hopeDuality(20)},
		damage:       2,
	}
	ce := newMockEngine(dice, &firstPicker{})
	dm := &mockDecisionMaker{}

	state := models.NewCombatState(
		[]*models.Combatant{tPlayer("p1", "Hero")},
		[]*models.Combatant{tEnemy("e1", "Goblin", 20)},
		0, 1, 3,
	)

	outcome := ce.RunCombat(state, dm, 5, strategy.ResourceGreedy)

	seizeFound := false
	for _, a := range outcome.ActionHistory {
		if a.Actor == "GM" {
			seizeFound = true
			break
		}
	}
	if !seizeFound {
		t.Error("expected GM seize action")
	}
	if outcome.FinalFearPool != 0 {
		t.Errorf("FinalFearPool = %d, want 0 (1 initial - 1 seize)", outcome.FinalFearPool)
	}
}

func TestGMSeizeDesperate(t *testing.T) {
	queue := make([]DualityRollResult, 10)
	for i := range queue {
		queue[i] = hopeDuality(20)
	}
	dice := &mockDiceRoller{dualityQueue: queue, damage: 2}
	ce := newMockEngine(dice, &firstPicker{})
	dm := &mockDecisionMaker{}

	state := models.NewCombatState(
		[]*models.Combatant{tPlayer("p1", "Hero")},
		[]*models.Combatant{tEnemy("e1", "Goblin", 20)},
		0, 5, 3,
	)

	outcome := ce.RunCombat(state, dm, 10, strategy.ResourceDesperate)

	for _, a := range outcome.ActionHistory {
		if a.Actor == "GM" {
			t.Error("Desperate should never seize spotlight")
		}
		if a.Actor == "Goblin" {
			t.Error("enemy should never act (no shift, no seize)")
		}
	}
}

func TestGMSeizeNoFear(t *testing.T) {
	queue := make([]DualityRollResult, 5)
	for i := range queue {
		queue[i] = hopeDuality(20)
	}
	dice := &mockDiceRoller{dualityQueue: queue, damage: 2}
	ce := newMockEngine(dice, &firstPicker{})
	dm := &mockDecisionMaker{}

	state := models.NewCombatState(
		[]*models.Combatant{tPlayer("p1", "Hero")},
		[]*models.Combatant{tEnemy("e1", "Goblin", 20)},
		0, 0, 3,
	)

	outcome := ce.RunCombat(state, dm, 5, strategy.ResourceGreedy)

	for _, a := range outcome.ActionHistory {
		if a.Actor == "GM" {
			t.Error("should not seize with 0 Fear")
		}
		if a.Actor == "Goblin" {
			t.Error("enemy should never act (no Fear for seize, no shift)")
		}
	}
}

func TestFearSeizePlusAdditional(t *testing.T) {
	dice := &mockDiceRoller{
		dualityQueue: []DualityRollResult{hopeDuality(20), hopeDuality(20), hopeDuality(20)},
		damage:       2,
	}
	ce := newMockEngine(dice, &firstPicker{})
	dm := &mockDecisionMaker{}

	state := models.NewCombatState(
		[]*models.Combatant{tPlayer("p1", "Hero")},
		[]*models.Combatant{tEnemy("e1", "Goblin", 20), tEnemy("e2", "Orc", 20)},
		0, 5, 3,
	)

	outcome := ce.RunCombat(state, dm, 6, strategy.ResourceGreedy)

	if outcome.FinalFearPool != 3 {
		t.Errorf("FinalFearPool = %d, want 3 (5 - 1 seize - 1 additional)", outcome.FinalFearPool)
	}
	goblin := countActorActions(outcome.ActionHistory, "Goblin")
	orc := countActorActions(outcome.ActionHistory, "Orc")
	if goblin+orc < 2 {
		t.Errorf("enemy actions in first GM turn = %d, want >= 2", goblin+orc)
	}
}

func TestCombatEndsCleanlyMidGMTurn(t *testing.T) {
	dice := &mockDiceRoller{
		dualityQueue: []DualityRollResult{fearDuality(20)},
		damage:       2,
	}
	ce := newMockEngine(dice, &firstPicker{})
	dm := &mockDecisionMaker{}

	player := tPlayer("p1", "Hero")
	player.HP = 1
	state := models.NewCombatState(
		[]*models.Combatant{player},
		[]*models.Combatant{tEnemy("e1", "Goblin", 20), tEnemy("e2", "Orc", 20)},
		0, 5, 3,
	)

	outcome := ce.RunCombat(state, dm, 10, strategy.ResourceGreedy)

	if !outcome.EnemyVictory {
		t.Error("expected enemy victory")
	}
	goblin := countActorActions(outcome.ActionHistory, "Goblin")
	orc := countActorActions(outcome.ActionHistory, "Orc")
	if goblin+orc != 1 {
		t.Errorf("enemy actions = %d, want 1 (combat ends after player death)", goblin+orc)
	}
}

func TestClearConditionDoesNotAttackSelf(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 20}, &firstPicker{})
	player := tPlayer("p1", "Hero")
	player.AddCondition(&models.ActiveCondition{Type: models.ConditionBurning, Duration: models.DurationUntilCleared})
	state := newStateWith([]*models.Combatant{player}, []*models.Combatant{tEnemy("e1", "Ogre", 10)}, 0, 0)

	ce.ExecuteTurnWithDecision(player, &strategy.ActionDecision{Target: player, ClearCondition: true}, state)

	if player.HP != player.HPMax {
		t.Errorf("HP = %d, want %d (clearing must not damage the actor)", player.HP, player.HPMax)
	}
	if player.HasCondition(models.ConditionBurning) {
		t.Error("burning should be cleared")
	}
	last := state.ActionHistory[len(state.ActionHistory)-1]
	if last.ActionType != models.ActionClearCondition {
		t.Errorf("ActionType = %v, want Clear Condition", last.ActionType)
	}
}

func TestStunnedSkipsExactlyOneTurn(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 5}, &firstPicker{})
	player := tPlayer("p1", "Hero")
	enemy := tEnemy("e1", "Ogre", 10)
	player.AddCondition(&models.ActiveCondition{Type: models.ConditionStunned, Duration: models.DurationUntilCleared})
	state := newStateWith([]*models.Combatant{player}, []*models.Combatant{enemy}, 0, 0)

	ce.ExecuteTurnWithDecision(player, &strategy.ActionDecision{Target: enemy}, state)
	if enemy.HP != enemy.HPMax {
		t.Error("stunned actor should not attack")
	}
	if player.HasCondition(models.ConditionStunned) {
		t.Error("stun should end after the skipped turn")
	}

	ce.ExecuteTurnWithDecision(player, &strategy.ActionDecision{Target: enemy}, state)
	if enemy.HP == enemy.HPMax {
		t.Error("actor should attack on the turn after the stun")
	}
}

func TestDefeatedByConditionTickDoesNotAct(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 5}, &firstPicker{})
	player := tPlayer("p1", "Hero")
	player.HP = 1
	enemy := tEnemy("e1", "Ogre", 10)
	player.AddCondition(&models.ActiveCondition{Type: models.ConditionBurning, Duration: models.DurationUntilCleared, DamagePerTick: 2})
	state := newStateWith([]*models.Combatant{player}, []*models.Combatant{enemy}, 0, 0)

	ce.ExecuteTurnWithDecision(player, &strategy.ActionDecision{Target: enemy}, state)

	if player.IsAlive() {
		t.Fatal("burning tick should defeat the player")
	}
	if enemy.HP != enemy.HPMax {
		t.Error("defeated actor should not attack")
	}
}

func TestPlayerFailsToClearCondition(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{dualityQueue: []DualityRollResult{fearDuality(8)}}, &firstPicker{})
	player := tPlayer("p1", "Hero")
	player.AddCondition(&models.ActiveCondition{Type: models.ConditionVulnerable, Duration: models.DurationUntilCleared, ClearDifficulty: 12})
	state := newStateWith([]*models.Combatant{player}, []*models.Combatant{tEnemy("e1", "Ogre", 10)}, 0, 0)

	ce.ExecuteTurnWithDecision(player, &strategy.ActionDecision{Target: player, ClearCondition: true}, state)

	if !player.HasCondition(models.ConditionVulnerable) {
		t.Error("a failed action roll should leave the condition in place")
	}
	last := state.ActionHistory[len(state.ActionHistory)-1]
	if !last.SpotlightShift {
		t.Error("a failed roll should pass the spotlight to the GM")
	}
	if state.FearPool != 1 {
		t.Errorf("FearPool = %d, want 1 (rolled with Fear)", state.FearPool)
	}
}

func TestAdversaryClearsConditionWithoutRoll(t *testing.T) {
	dice := &mockDiceRoller{dualityQueue: []DualityRollResult{fearDuality(1)}}
	ce := newMockEngine(dice, &firstPicker{})
	enemy := tEnemy("e1", "Ogre", 10)
	enemy.AddCondition(&models.ActiveCondition{Type: models.ConditionVulnerable, Duration: models.DurationUntilCleared})
	state := newStateWith([]*models.Combatant{tPlayer("p1", "Hero")}, []*models.Combatant{enemy}, 0, 0)

	ce.ExecuteTurnWithDecision(enemy, &strategy.ActionDecision{Target: enemy, ClearCondition: true}, state)

	if enemy.HasCondition(models.ConditionVulnerable) {
		t.Error("an adversary clears a condition by spending its spotlight")
	}
	if dice.dualityIdx != 0 {
		t.Error("an adversary should not roll to clear a condition")
	}
}

func fallenPlayerState(deathMove strategy.DeathMoveStrategy, dice *mockDiceRoller) (*CombatEngine, *models.Combatant, *models.Combatant, *models.CombatState) {
	ce := newMockEngine(dice, &firstPicker{})
	ce.SetDeathMove(deathMove)
	player := tPlayer("p1", "Hero")
	player.DamageDice = "1d8"
	player.Stress = 3
	player.HP = 0
	player.IsDefeated = true
	enemy := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{player}, []*models.Combatant{enemy}, 0, 0)
	return ce, player, enemy, state
}

func TestDeathMoveAvoidDeath(t *testing.T) {
	ce, player, enemy, state := fallenPlayerState(strategy.DeathMoveAvoidDeath, &mockDiceRoller{damage: 5})

	ce.resolveDeathMoves(state)

	if player.IsAlive() || !player.Fallen {
		t.Error("Avoid Death should leave the PC out of the fight")
	}
	if enemy.HP != enemy.HPMax {
		t.Error("Avoid Death should not attack")
	}

	ce.resolveDeathMoves(state)
	if len(state.ActionHistory) != 1 {
		t.Errorf("death move recorded %d times, want 1", len(state.ActionHistory))
	}
}

func TestDeathMoveRiskItAll(t *testing.T) {
	cases := []struct {
		name       string
		roll       DualityRollResult
		wantAlive  bool
		wantHP     int
		wantStress int
	}{
		{"hope higher stays up", DualityRollResult{HopeDie: 12, FearDie: 4, Result: models.ResultHope}, true, 10, 1},
		{"fear higher dies", DualityRollResult{HopeDie: 4, FearDie: 9, Result: models.ResultFear}, false, 0, 3},
		{"matching clears everything", DualityRollResult{HopeDie: 2, FearDie: 2, Result: models.ResultCriticalHope, IsCritical: true}, true, 10, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dice := &mockDiceRoller{dualityQueue: []DualityRollResult{c.roll}}
			ce, player, _, state := fallenPlayerState(strategy.DeathMoveRiskItAll, dice)

			ce.resolveDeathMoves(state)

			if player.IsAlive() != c.wantAlive {
				t.Fatalf("IsAlive = %v, want %v", player.IsAlive(), c.wantAlive)
			}
			if player.HP != c.wantHP || player.Stress != c.wantStress {
				t.Errorf("HP / Stress = %d / %d, want %d / %d", player.HP, player.Stress, c.wantHP, c.wantStress)
			}
		})
	}
}

func TestDeathMoveBlazeOfGlory(t *testing.T) {
	// A roll that would miss on its own: Blaze of Glory still crits
	dice := &mockDiceRoller{damage: 3, dualityQueue: []DualityRollResult{fearDuality(1)}}
	ce, player, enemy, state := fallenPlayerState(strategy.DeathMoveBlazeOfGlory, dice)

	ce.resolveDeathMoves(state)

	if player.IsAlive() || !player.Fallen {
		t.Error("the PC dies after a Blaze of Glory")
	}
	if enemy.HP == enemy.HPMax {
		t.Error("the final action should hit automatically")
	}
	last := state.ActionHistory[len(state.ActionHistory)-1]
	if last.ActionType != models.ActionDeathMove || !last.IsCritical {
		t.Errorf("last action = %v critical=%v, want a critical Death Move", last.ActionType, last.IsCritical)
	}
}

func TestMutualWipeEndsCombat(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 5}, &firstPicker{})
	player := tPlayer("p1", "Hero")
	enemy := tEnemy("e1", "Ogre", 10)
	player.IsDefeated, player.Fallen, enemy.IsDefeated = true, true, true
	state := newStateWith([]*models.Combatant{player}, []*models.Combatant{enemy}, 0, 0)

	outcome := ce.RunCombat(state, &mockDecisionMaker{}, 50, strategy.ResourceGreedy)

	if outcome.PlayerVictory || outcome.EnemyVictory {
		t.Error("nobody wins a mutual wipe")
	}
}

func TestReactionRollsAppearInTheLog(t *testing.T) {
	dice := &mockDiceRoller{damage: 9, dualityQueue: []DualityRollResult{hopeDuality(5), hopeDuality(20)}}
	ce := newMockEngine(dice, &firstPicker{})
	demon := tEnemy("e1", "Demon", 10)
	demon.Difficulty = 14
	feature := hellfire()
	demon.AdversaryFeatures = []models.AdversaryFeature{*feature}
	players := []*models.Combatant{tPlayer("p1", "Rogue"), tPlayer("p2", "Knight")}
	state := newStateWith(players, []*models.Combatant{demon}, 0, 0)

	ce.ExecuteTurnWithDecision(demon, &strategy.ActionDecision{Target: players[0], AdversaryFeature: &demon.AdversaryFeatures[0]}, state)

	var log []models.ActionRecord
	for _, a := range state.ActionHistory {
		if a.ActionType == models.ActionUseAbility || a.ActionType == models.ActionReactionRoll {
			log = append(log, a)
		}
	}
	if len(log) != 3 || log[0].ActionType != models.ActionUseAbility {
		t.Fatalf("log = %+v, want the feature followed by two reaction rolls", log)
	}

	rogue, knight := log[1], log[2]
	if rogue.Actor != "Rogue" || rogue.Result != "Failed" || rogue.TotalRoll != 5 || rogue.TargetDefense != 14 {
		t.Errorf("first reaction = %+v, want Rogue failing 5 vs 14", rogue)
	}
	if knight.Actor != "Knight" || knight.Result != "Success: half damage" || knight.FeatureName != "Hellfire" {
		t.Errorf("second reaction = %+v, want Knight halving Hellfire", knight)
	}
	if rogue.SpotlightShift || knight.SpotlightShift {
		t.Error("reaction rolls never move the spotlight")
	}
}

func TestAttackLogShowsCriticalsAndHPAfter(t *testing.T) {
	crit := DualityRollResult{HopeDie: 4, FearDie: 4, Result: models.ResultCriticalHope, WithSpirit: 10, IsCritical: true}
	ce := newMockEngine(&mockDiceRoller{damage: 5, d20: 20, dualityQueue: []DualityRollResult{crit}}, &firstPicker{})
	hero := tPlayer("p1", "Hero")
	hero.DamageDice = "1d8+1"
	ogre := tEnemy("e1", "Ogre", 20)
	ogre.Difficulty = 14
	state := newStateWith([]*models.Combatant{hero}, []*models.Combatant{ogre}, 0, 0)

	// Proficiency 2: 2d8 rolls 5, +1 flat, +16 for the critical = 22, Severe
	ce.executeAttack(hero, ogre, state)
	row := state.ActionHistory[len(state.ActionHistory)-1]
	if row.Result != "Critical Hit" || !row.IsCritical || row.DamageRoll != 22 ||
		len(row.DamageParts) != 1 || row.DamageParts[0] != (models.LogPart{Label: "critical", Value: 16}) {
		t.Errorf("PC row: %q, critical %v, parts %v, roll %d; want Critical Hit / true / [critical 16] / 22", row.Result, row.IsCritical, row.DamageParts, row.DamageRoll)
	}
	if row.TargetHPBefore != 20 || row.TargetHPAfter != 17 {
		t.Errorf("PC row: target HP %d -> %d, want 20 -> 17", row.TargetHPBefore, row.TargetHPAfter)
	}

	// The GM's natural 20 is a critical too
	ce.executeAttack(ogre, hero, state)
	row = state.ActionHistory[len(state.ActionHistory)-1]
	if row.Result != "Critical Hit" || !row.IsCritical {
		t.Errorf("GM row: %q, critical %v; want Critical Hit / true", row.Result, row.IsCritical)
	}
}

func TestAreaDamageLogShowsTheTargetsOwnSeverity(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 3}, &firstPicker{})
	hero, friend := tPlayer("p1", "Hero"), tPlayer("p2", "Friend")
	hero.ArmorSlots = 1
	ogre := tEnemy("e1", "Ogre", 20)
	spit := models.AdversaryFeature{Name: "Spit Acid", DCType: "target", Effects: []models.FeatureEffect{
		{Category: models.FeatureEffectAreaDamage, DiceFormula: "2d6", DamageType: "physical"},
	}}
	ogre.AdversaryFeatures = []models.AdversaryFeature{spit}
	state := newStateWith([]*models.Combatant{hero, friend}, []*models.Combatant{ogre}, 0, 0)

	// 3 is Minor for both: the hero's armor absorbs it, the friend marks 1 HP
	ce.executeFeature(ogre, hero, &ogre.AdversaryFeatures[0], state)
	row := state.ActionHistory[len(state.ActionHistory)-1]
	if row.Threshold != "Minor" || row.TargetsHit != 2 || row.HPDamage != 1 || row.ProcessedDamage != 3 {
		t.Errorf("row: %s, %d targets, %d HP, %d damage; want Minor / 2 / 1 / 3", row.Threshold, row.TargetsHit, row.HPDamage, row.ProcessedDamage)
	}
	if row.Target != "Hero" || row.TargetHPBefore != 10 || row.TargetHPAfter != 10 || !row.ArmorUsed || row.ArmorSlotsAfter != 0 {
		t.Errorf("row: %s HP %d -> %d, armor used %v, slots after %d; want Hero 10 -> 10, armor used, 0 left",
			row.Target, row.TargetHPBefore, row.TargetHPAfter, row.ArmorUsed, row.ArmorSlotsAfter)
	}
}

// Spit Acid strips an Armor Slot and then deals damage: the row must show both, not only the
// slot the damage roll used.
func TestLogRowCoversEverythingTheActionCostTheTarget(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 6}, &firstPicker{})
	hero := tPlayer("p1", "Hero")
	hero.ArmorSlots = 3
	ogre := tEnemy("e1", "Ogre", 20)
	ogre.AdversaryFeatures = []models.AdversaryFeature{{Name: "Spit Acid", DCType: "target", Effects: []models.FeatureEffect{
		{Category: models.FeatureEffectArmorDamage, Magnitude: 1, TargetType: models.FeatureTargetAOE, FallbackFear: 1},
		{Category: models.FeatureEffectAreaDamage, DiceFormula: "2d6", DamageType: "physical"},
	}}}
	state := newStateWith([]*models.Combatant{hero}, []*models.Combatant{ogre}, 0, 0)
	spit := func() models.ActionRecord {
		ce.executeFeature(ogre, hero, &ogre.AdversaryFeatures[0], state)
		return state.ActionHistory[len(state.ActionHistory)-1]
	}

	// 6 is Major: one slot to the acid, one to take the hit down to Minor
	if row := spit(); row.ArmorSlotsBefore != 3 || row.ArmorSlotsAfter != 1 || row.TargetHPBefore != 10 || row.TargetHPAfter != 9 {
		t.Errorf("first: armor %d -> %d, HP %d -> %d; want 3 -> 1 and 10 -> 9", row.ArmorSlotsBefore, row.ArmorSlotsAfter, row.TargetHPBefore, row.TargetHPAfter)
	}
	// The last slot goes to the acid, so the hit lands in full
	if row := spit(); row.ArmorSlotsBefore != 1 || row.ArmorSlotsAfter != 0 || row.TargetHPBefore != 9 || row.TargetHPAfter != 7 {
		t.Errorf("second: armor %d -> %d, HP %d -> %d; want 1 -> 0 and 9 -> 7", row.ArmorSlotsBefore, row.ArmorSlotsAfter, row.TargetHPBefore, row.TargetHPAfter)
	}
	// No slot left: the acid costs a Hit Point on top of the 2 from the damage
	row := spit()
	if row.ArmorUsed || row.TargetHPBefore != 7 || row.TargetHPAfter != 4 {
		t.Errorf("third: armor used %v, HP %d -> %d; want no armor and 7 -> 4", row.ArmorUsed, row.TargetHPBefore, row.TargetHPAfter)
	}
	if len(row.Notes) != 1 || row.Notes[0] != "Hero marks 1 HP for having no Armor Slot to mark" {
		t.Errorf("notes = %v, want the acid's Hit Point named", row.Notes)
	}
}

// The log must be able to say where every number came from: the trait on the roll, each
// feature's share of the damage, and each step from the rolled severity to the Hit Points marked.
func TestLogRowsNameWhereTheNumbersComeFrom(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 9}, &firstPicker{})
	hero := tPlayer("p1", "Hero")
	hero.Traits.Strength, hero.WeaponTrait = 2, "strength"
	hero.DamageDice = "1d8+4"
	hero.ArmorSlots = 1
	hero.DomainAbilities = []models.DomainAbility{unstoppable(4)}
	hero.DomainAbilities[0].TokensHeld = 3
	ogre := tEnemy("e1", "Ogre", 20)
	state := newStateWith([]*models.Combatant{hero}, []*models.Combatant{ogre}, 0, 0)

	ce.executeAttack(hero, ogre, state)
	row := state.ActionHistory[len(state.ActionHistory)-1]
	if len(row.RollParts) != 1 || row.RollParts[0] != (models.LogPart{Label: "Strength", Value: 2}) {
		t.Errorf("roll parts = %v, want [Strength 2]", row.RollParts)
	}
	// 2d8 rolls 9, +4 from the weapon, +3 from the Unstoppable Die
	if row.DamageFormula != "2d8+4" || row.DamageBonus != 4 || row.DamageRoll != 16 ||
		len(row.DamageParts) != 1 || row.DamageParts[0] != (models.LogPart{Label: "Unstoppable", Value: 3}) {
		t.Errorf("damage: %s, weapon %d, parts %v, total %d; want 2d8+4 / 4 / [Unstoppable 3] / 16", row.DamageFormula, row.DamageBonus, row.DamageParts, row.DamageRoll)
	}
	if len(row.Severity) != 1 || row.Severity[0] != "Severe at 4/7" {
		t.Errorf("severity = %v, want [Severe at 4/7]", row.Severity)
	}

	// The ogre hits back for 9: Severe for a level 3 PC, Major while Unstoppable, Minor after armor
	ce.executeAttack(ogre, hero, state)
	row = state.ActionHistory[len(state.ActionHistory)-1]
	if len(row.RollParts) != 1 || row.RollParts[0] != (models.LogPart{Label: "ATK", Value: 5}) || !row.VsDefense {
		t.Errorf("GM roll parts = %v, vs defense %v; want [ATK 5] against Evasion", row.RollParts, row.VsDefense)
	}
	want := []string{"Severe at 6/9", "Major by Unstoppable", "Minor by an Armor Slot"}
	if strings.Join(row.Severity, " | ") != strings.Join(want, " | ") || row.HPDamage != 1 {
		t.Errorf("severity = %v with %d HP, want %v with 1 HP", row.Severity, row.HPDamage, want)
	}
}

func TestLogKeepsAFixedDifficultyAndNamesWhatCutTheDamage(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 6}, &firstPicker{})
	hero := tPlayer("p1", "Hero")
	hero.Traits.Knowledge, hero.SpellcastTrait = 3, "knowledge"
	ward := passive("Rune Ward", models.TriggerOnDamageIncoming, "if_major_or_worse", models.FeatureEffect{Category: models.FeatureEffectDamageReduction, DiceFormula: "1d8"})
	ward.HopeCost = 1
	barrier := models.DomainAbility{Name: "Barrier", DCType: "fixed", FixedDC: 15, RollTrait: "spellcast",
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectHPHeal, Magnitude: 1, TargetType: models.FeatureTargetAllies}}}
	hero.DomainAbilities = []models.DomainAbility{ward, barrier}
	ogre := tEnemy("e1", "Ogre", 20)
	state := newStateWith([]*models.Combatant{hero}, []*models.Combatant{ogre}, 1, 0)

	ce.executeAttack(ogre, hero, state)
	row := lastRow(state, models.ActionAttack)
	if len(row.Reduced) != 1 || row.Reduced[0] != "Rune Ward -6" || row.ProcessedDamage != 0 {
		t.Errorf("reduced = %v, %d left; want [Rune Ward -6] and 0", row.Reduced, row.ProcessedDamage)
	}

	hero.HP = 5
	ce.executeAbility(hero, ogre, &hero.DomainAbilities[1], state)
	row = state.ActionHistory[len(state.ActionHistory)-1]
	if row.TargetDefense != 15 || row.VsDefense || row.RollParts[0] != (models.LogPart{Label: "Spellcast (Knowledge)", Value: 3}) {
		t.Errorf("fixed roll: vs %d, vs defense %v, parts %v; want 15 / false / [Spellcast (Knowledge) 3]", row.TargetDefense, row.VsDefense, row.RollParts)
	}
}

func lastRow(state *models.CombatState, kind models.ActionType) models.ActionRecord {
	for i := len(state.ActionHistory) - 1; i >= 0; i-- {
		if state.ActionHistory[i].ActionType == kind {
			return state.ActionHistory[i]
		}
	}
	return models.ActionRecord{}
}

// An attack's row comes first, as things stood when it landed; what it set off follows.
func TestAttackRowComesBeforeWhatItSetsOff(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 7}, &firstPicker{})
	hero := tPlayer("p1", "Hero")
	hero.DomainAbilities = []models.DomainAbility{unstoppable(4)}
	hero.DomainAbilities[0].TokensHeld, hero.DomainAbilities[0].UsesLeft = 4, 0
	ogre := tEnemy("e1", "Ogre", 20)
	ogre.AdversaryFeatures = []models.AdversaryFeature{{Name: "Acid Bath", Passive: true,
		Trigger: models.FeatureTriggerCondition{Type: models.TriggerOnDamageTaken, Filter: "if_took_severe"},
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectAreaDamage, DiceFormula: "1d10", DamageType: "physical"}}}}
	state := newStateWith([]*models.Combatant{hero}, []*models.Combatant{ogre}, 0, 0)

	// 7 + 4 from the die is Severe: the die runs out, and the splash of 7 is Major on the hero
	ce.executeAttack(hero, ogre, state)

	var order []string
	for _, row := range state.ActionHistory {
		order = append(order, strings.TrimSpace(row.ActionType.String()+" "+row.FeatureName))
	}
	want := "Attack | Use Ability Unstoppable | Use Ability Acid Bath"
	if got := strings.Join(order, " | "); got != want {
		t.Fatalf("rows = %s, want %s", got, want)
	}
	attack, bath := state.ActionHistory[0], state.ActionHistory[2]
	if attack.DamageRoll != 11 || attack.TargetHPAfter != 17 || attack.ActorState.HP != 10 {
		t.Errorf("attack row: rolled %d, ogre at %d HP, hero at %d HP; want 11 / 17 / 10 (before the splash)", attack.DamageRoll, attack.TargetHPAfter, attack.ActorState.HP)
	}
	if bath.Target != "Hero" || bath.TargetHPBefore != 10 || bath.TargetHPAfter != 8 || hero.HP != 8 {
		t.Errorf("Acid Bath row: %s %d -> %d HP (hero now %d); want Hero 10 -> 8", bath.Target, bath.TargetHPBefore, bath.TargetHPAfter, hero.HP)
	}
}

// A row whose Fear total did not move can still have spent and gained Fear: the log names both.
func TestLogRowNamesWhatMovedTheFearPool(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 3}, &firstPicker{})
	hero, friend := tPlayer("p1", "Hero"), tPlayer("p2", "Friend")
	ogre := tEnemy("e1", "Ogre", 20)
	ogre.AdversaryFeatures = []models.AdversaryFeature{{Name: "Spit Acid", DCType: "target", Effects: []models.FeatureEffect{
		{Category: models.FeatureEffectArmorDamage, Magnitude: 1, TargetType: models.FeatureTargetAOE, FallbackFear: 1},
		{Category: models.FeatureEffectAreaDamage, DiceFormula: "2d6", DamageType: "physical"},
	}}}
	state := newStateWith([]*models.Combatant{hero, friend}, []*models.Combatant{ogre}, 0, 2)

	// The GM pays for another spotlight; neither PC has an Armor Slot for the acid
	state.SpendFearFor(1, "for another spotlight")
	ce.executeFeature(ogre, hero, &ogre.AdversaryFeatures[0], state)
	row := lastRow(state, models.ActionUseAbility)
	want := []models.LogPart{{Label: "for another spotlight", Value: -1}, {Label: "no Armor Slot to mark", Value: 1}, {Label: "no Armor Slot to mark", Value: 1}}
	if row.FearPoolAfter != 3 || !slices.Equal(row.FearChanges, want) {
		t.Errorf("Fear %d, changes %v; want 3 and %v", row.FearPoolAfter, row.FearChanges, want)
	}

	// A dodge is logged while the attack is still resolving: the spotlight's Fear stays on the attack row
	dodge := passive("I See It Coming", models.TriggerOnAttacked, "", models.FeatureEffect{Category: models.FeatureEffectEvasionBonus, Magnitude: 99})
	hero.DomainAbilities = []models.DomainAbility{dodge}
	state.SpendFearFor(1, "for another spotlight")
	ce.executeAttack(ogre, hero, state)
	attack, dodged := lastRow(state, models.ActionAttack), state.ActionHistory[len(state.ActionHistory)-2]
	if dodged.FeatureName != "I See It Coming" || len(dodged.FearChanges) != 0 || len(attack.FearChanges) != 1 || attack.FearChanges[0].Value != -1 {
		t.Errorf("dodge row %q changes %v, attack row changes %v; want the spend on the attack row only", dodged.FeatureName, dodged.FearChanges, attack.FearChanges)
	}
}
