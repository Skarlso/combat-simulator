package models

import (
	"testing"
)

func newTestState() *CombatState {
	players := []*Combatant{
		NewCombatant("p1", "Hero", SidePlayer, 3, 12, 2, 15, 8),
		NewCombatant("p2", "Mage", SidePlayer, 3, 10, 3, 12, 8),
	}
	enemies := []*Combatant{
		NewAdversary("e1", "Goblin", "Tier 1", "Standard", 10, 3, 8, 4, []int{4, 7}),
		NewAdversary("e2", "Orc", "Tier 1", "Bruiser", 12, 5, 15, 6, []int{5, 10}),
	}
	return NewCombatState(players, enemies, 0, 3, 5)
}

func TestNewCombatState(t *testing.T) {
	state := newTestState()

	if len(state.Players) != 2 {
		t.Errorf("Players = %d, want 2", len(state.Players))
	}
	if len(state.Enemies) != 2 {
		t.Errorf("Enemies = %d, want 2", len(state.Enemies))
	}
	if state.TotalHope() != 0 {
		t.Errorf("TotalHope = %d, want 0", state.TotalHope())
	}
	if state.FearPool != 3 {
		t.Errorf("FearPool = %d, want 3", state.FearPool)
	}
}

func TestNewCombatStateFiltersSummonTemplates(t *testing.T) {
	template := NewAdversary("t1", "Template", "Tier 1", "Minion", 8, 2, 5, 2, nil)
	template.IsSummonTemplate = true
	normal := NewAdversary("e1", "Normal", "Tier 1", "Standard", 10, 3, 10, 4, nil)

	state := NewCombatState(nil, []*Combatant{template, normal}, 0, 0, 5)
	if len(state.Enemies) != 1 {
		t.Errorf("Enemies = %d, want 1 (template should be filtered)", len(state.Enemies))
	}
	if state.Enemies[0].Name != "Normal" {
		t.Errorf("Enemy name = %q, want %q", state.Enemies[0].Name, "Normal")
	}
}

func TestGetAlivePlayers(t *testing.T) {
	state := newTestState()
	state.Players[0].IsDefeated = true

	alive := state.GetAlivePlayers()
	if len(alive) != 1 {
		t.Errorf("GetAlivePlayers() = %d, want 1", len(alive))
	}
}

func TestGetAliveEnemies(t *testing.T) {
	state := newTestState()
	state.Enemies[0].IsDefeated = true
	state.Enemies[1].IsDefeated = true

	alive := state.GetAliveEnemies()
	if len(alive) != 0 {
		t.Errorf("GetAliveEnemies() = %d, want 0", len(alive))
	}
}

func TestGetAlliesOf(t *testing.T) {
	state := newTestState()

	allies := state.GetAlliesOf(state.Players[0])
	if len(allies) != 2 {
		t.Errorf("GetAlliesOf(player) = %d, want 2", len(allies))
	}

	allies = state.GetAlliesOf(state.Enemies[0])
	if len(allies) != 2 {
		t.Errorf("GetAlliesOf(enemy) = %d, want 2", len(allies))
	}
}

func TestGetEnemiesOf(t *testing.T) {
	state := newTestState()

	enemies := state.GetEnemiesOf(state.Players[0])
	if len(enemies) != 2 {
		t.Errorf("GetEnemiesOf(player) = %d, want 2 enemies", len(enemies))
	}

	enemies = state.GetEnemiesOf(state.Enemies[0])
	if len(enemies) != 2 {
		t.Errorf("GetEnemiesOf(enemy) = %d, want 2 players", len(enemies))
	}
}

func TestVictoryConditions(t *testing.T) {
	state := newTestState()

	if state.IsCombatOver() {
		t.Error("combat should not be over initially")
	}

	state.Enemies[0].IsDefeated = true
	state.Enemies[1].IsDefeated = true

	if !state.IsPlayerVictory() {
		t.Error("should be player victory")
	}
	if state.IsEnemyVictory() {
		t.Error("should not be enemy victory")
	}
	if !state.IsCombatOver() {
		t.Error("combat should be over")
	}
}

func TestEnemyVictory(t *testing.T) {
	state := newTestState()
	state.Players[0].IsDefeated = true
	state.Players[1].IsDefeated = true

	if !state.IsEnemyVictory() {
		t.Error("should be enemy victory")
	}
	if state.IsPlayerVictory() {
		t.Error("should not be player victory")
	}
}

func TestHopeIsPerPC(t *testing.T) {
	state := newTestState()
	pc, other := state.Players[0], state.Players[1]

	pc.GainHope(3)
	if pc.Hope != 3 || other.Hope != 0 {
		t.Errorf("Hope = %d / %d, want 3 / 0 (Hope is not shared)", pc.Hope, other.Hope)
	}

	if !pc.SpendHope(2) {
		t.Error("SpendHope should succeed")
	}
	if pc.Hope != 1 {
		t.Errorf("Hope = %d, want 1", pc.Hope)
	}

	if pc.SpendHope(5) {
		t.Error("SpendHope should fail (insufficient)")
	}
	if pc.Hope != 1 {
		t.Errorf("Hope = %d, want 1 (unchanged after failed spend)", pc.Hope)
	}

	other.GainHope(2)
	if state.TotalHope() != 3 {
		t.Errorf("TotalHope = %d, want 3", state.TotalHope())
	}
}

func TestNewCombatStateGivesEachPCStartingHope(t *testing.T) {
	players := []*Combatant{newTestPlayer(), newTestPlayer()}
	NewCombatState(players, nil, 2, 0, 3)

	for _, p := range players {
		if p.Hope != 2 {
			t.Errorf("Hope = %d, want 2", p.Hope)
		}
	}
}

func TestCombatOverWhenBothSidesFall(t *testing.T) {
	state := newTestState()
	for _, c := range append(state.Players, state.Enemies...) {
		c.IsDefeated = true
	}

	if !state.IsCombatOver() {
		t.Error("combat should be over when nobody is left standing")
	}
	if state.IsPlayerVictory() || state.IsEnemyVictory() {
		t.Error("a mutual wipe is a draw")
	}
}

func TestFearPool(t *testing.T) {
	state := newTestState()

	state.AddFear(2)
	if state.FearPool != 5 {
		t.Errorf("FearPool = %d, want 5 (3 initial + 2)", state.FearPool)
	}

	ok := state.SpendFear(4)
	if !ok {
		t.Error("SpendFear should succeed")
	}
	if state.FearPool != 1 {
		t.Errorf("FearPool = %d, want 1", state.FearPool)
	}
}

func TestHopeCap(t *testing.T) {
	pc := newTestPlayer()
	pc.GainHope(100)
	if pc.Hope != MaxHope {
		t.Errorf("Hope = %d, want %d (capped)", pc.Hope, MaxHope)
	}
}

func TestFearPoolCap(t *testing.T) {
	state := newTestState()
	state.FearPool = 0
	state.AddFear(100)
	if state.FearPool != MaxFearPool {
		t.Errorf("FearPool = %d, want %d (capped)", state.FearPool, MaxFearPool)
	}
}

func TestIncrementCounters(t *testing.T) {
	state := newTestState()

	state.RoundNumber++
	state.TurnNumber += 2

	if state.RoundNumber != 1 {
		t.Errorf("RoundNumber = %d, want 1", state.RoundNumber)
	}
	if state.TurnNumber != 2 {
		t.Errorf("TurnNumber = %d, want 2", state.TurnNumber)
	}
}

func TestSummonCombatant(t *testing.T) {
	state := newTestState()
	summon := NewAdversary("s1", "Skeleton", "Tier 1", "Minion", 8, 2, 5, 2, nil)

	state.SummonCombatant(summon)
	if len(state.Enemies) != 3 {
		t.Errorf("Enemies = %d, want 3 after summon", len(state.Enemies))
	}
}

func TestRecordAction(t *testing.T) {
	state := newTestState()
	state.RoundNumber = 1
	state.TurnNumber = 3

	state.RecordAction(ActionParams{
		Actor:      state.Players[0],
		ActionType: ActionAttack,
		Target:     state.Enemies[0],
		Result:     "Hit",
		DamageRoll: 8, ProcessedDamage: 8,
		DamageDice: []int{5, 3},
		HPDamage:   2,
		Threshold:  ThresholdMajor,
		HopeDie:    7, FearDie: 5,
		TotalRoll: 14, TargetDefense: 10,
		HopeGenerated: 1,
	})

	if len(state.ActionHistory) != 1 {
		t.Fatalf("ActionHistory len = %d, want 1", len(state.ActionHistory))
	}

	record := state.ActionHistory[0]
	if record.Round != 1 {
		t.Errorf("Round = %d, want 1", record.Round)
	}
	if record.Turn != 3 {
		t.Errorf("Turn = %d, want 3", record.Turn)
	}
	if record.Actor != "Hero" {
		t.Errorf("Actor = %q, want %q", record.Actor, "Hero")
	}
	if record.Target != "Goblin" {
		t.Errorf("Target = %q, want %q", record.Target, "Goblin")
	}
	if record.Threshold != "Major" {
		t.Errorf("Threshold = %q, want %q", record.Threshold, "Major")
	}
}

func TestRecordActionCriticalDetection(t *testing.T) {
	state := newTestState()

	state.RecordAction(ActionParams{
		Actor:      state.Players[0],
		ActionType: ActionAttack,
		Target:     state.Enemies[0],
		Result:     "Critical Hit",
		DamageRoll: 16, ProcessedDamage: 16,
		DamageDice: []int{6, 6}, DamageBonus: 4,
		HPDamage:  3,
		Threshold: ThresholdSevere,
		HopeDie:   7, FearDie: 7,
		TotalRoll: 18, TargetDefense: 10,
		HopeGenerated: 1,
	})

	if !state.ActionHistory[0].IsCritical {
		t.Error("should detect critical when hope == fear die")
	}
}

func TestRecordActionWithDamageResult(t *testing.T) {
	state := newTestState()
	dr := &DamageResult{
		HPBefore:         10,
		HPAfter:          8,
		ArmorUsed:        true,
		ArmorSlotsBefore: 2,
		ArmorSlotsAfter:  1,
	}

	state.RecordAction(ActionParams{
		Actor:      state.Players[0],
		ActionType: ActionAttack,
		Target:     state.Enemies[0],
		Result:     "Hit",
		DamageRoll: 8, ProcessedDamage: 8,
		HPDamage:  2,
		Threshold: ThresholdMajor,
		HopeDie:   7, FearDie: 5,
		TotalRoll: 14, TargetDefense: 10,
		HopeGenerated: 1,
		DamageResult:  dr,
	})

	record := state.ActionHistory[0]
	if record.TargetHPBefore != 10 {
		t.Errorf("TargetHPBefore = %d, want 10", record.TargetHPBefore)
	}
	if !record.ArmorUsed {
		t.Error("ArmorUsed should be true")
	}
	if record.ArmorSlotsBefore != 2 {
		t.Errorf("ArmorSlotsBefore = %d, want 2", record.ArmorSlotsBefore)
	}
}

func TestRecordActionSnapshotsActorAndTarget(t *testing.T) {
	state := newTestState()
	pc, enemy := state.Players[0], state.Enemies[0]
	pc.Hope, pc.Stress, pc.ArmorSlots, pc.ArmorMax = 3, 2, 1, 4
	enemy.HP, enemy.Stress = 4, 1

	state.RecordAction(ActionParams{Actor: pc, ActionType: ActionAttack, Target: enemy, Result: "Hit"})
	state.RecordAction(ActionParams{Actor: &Combatant{Name: "GM"}, ActionType: ActionPass, Result: "Spends 1 Fear"})

	row := state.ActionHistory[0]
	want := CombatantState{IsPC: true, HP: pc.HP, HPMax: pc.HPMax, Stress: 2, StressMax: pc.StressMax, Hope: 3, Armor: 1, ArmorMax: 4}
	if row.ActorState == nil || *row.ActorState != want {
		t.Errorf("actor state = %+v, want %+v", row.ActorState, want)
	}
	if row.TargetState == nil || row.TargetState.IsPC || row.TargetState.HP != 4 || row.TargetState.Stress != 1 {
		t.Errorf("target state = %+v, want the adversary at 4 HP and 1 Stress", row.TargetState)
	}
	if gm := state.ActionHistory[1]; gm.ActorState != nil || gm.TargetState != nil {
		t.Errorf("the GM's row should carry no state: %+v / %+v", gm.ActorState, gm.TargetState)
	}
}
