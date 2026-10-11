package engine

import (
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)

type mockDiceRoller struct {
	dualityQueue []DualityRollResult
	dualityIdx   int
	damage       int
	coinFlip     int
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
	return D20RollResult{D20Roll: 15, Modifier: modifier, Total: 15 + modifier}
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
	player.AddCondition(&models.ActiveCondition{Type: models.ConditionRestrained, Duration: models.DurationUntilCleared, ClearDifficulty: 12})
	state := newStateWith([]*models.Combatant{player}, []*models.Combatant{tEnemy("e1", "Ogre", 10)}, 0, 0)

	ce.ExecuteTurnWithDecision(player, &strategy.ActionDecision{Target: player, ClearCondition: true}, state)

	if !player.HasCondition(models.ConditionRestrained) {
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
	enemy.AddCondition(&models.ActiveCondition{Type: models.ConditionRestrained, Duration: models.DurationUntilCleared})
	state := newStateWith([]*models.Combatant{tPlayer("p1", "Hero")}, []*models.Combatant{enemy}, 0, 0)

	ce.ExecuteTurnWithDecision(enemy, &strategy.ActionDecision{Target: enemy, ClearCondition: true}, state)

	if enemy.HasCondition(models.ConditionRestrained) {
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
