package models

import (
	"testing"
)

func TestConditionTick(t *testing.T) {
	cond := &ActiveCondition{
		Type:       ConditionBurning,
		Duration:   DurationRounds,
		RoundsLeft: 2,
	}

	expired := cond.Tick()
	if expired {
		t.Error("should not be expired after first tick (1 round left)")
	}
	if cond.RoundsLeft != 1 {
		t.Errorf("RoundsLeft = %d, want 1", cond.RoundsLeft)
	}

	expired = cond.Tick()
	if !expired {
		t.Error("should be expired after second tick")
	}
}

func TestConditionTickNonRounds(t *testing.T) {
	cond := &ActiveCondition{
		Type:     ConditionVulnerable,
		Duration: DurationUntilCleared,
	}

	expired := cond.Tick()
	if expired {
		t.Error("non-rounds condition should not expire from Tick()")
	}
}

func TestShouldExpire(t *testing.T) {
	tests := []struct {
		name     string
		duration DurationType
		trigger  TriggerType
		rounds   int
		want     bool
	}{
		{"instant always expires", DurationInstant, TriggerTurnStart, 0, true},
		{"end_of_turn on turn end", DurationEndOfTurn, TriggerTurnEnd, 0, true},
		{"end_of_turn on turn start", DurationEndOfTurn, TriggerTurnStart, 0, false},
		{"start_next_turn on turn start", DurationStartNextTurn, TriggerTurnStart, 0, true},
		{"start_next_turn on turn end", DurationStartNextTurn, TriggerTurnEnd, 0, false},
		{"rounds expired", DurationRounds, TriggerTurnStart, 0, true},
		{"rounds not expired", DurationRounds, TriggerTurnStart, 2, false},
		{"until_cleared never auto-expires", DurationUntilCleared, TriggerTurnStart, 0, false},
		{"until_cleared never auto-expires end", DurationUntilCleared, TriggerTurnEnd, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond := &ActiveCondition{
				Duration:   tt.duration,
				RoundsLeft: tt.rounds,
			}
			got := cond.ShouldExpire(tt.trigger)
			if got != tt.want {
				t.Errorf("ShouldExpire(%v) = %v, want %v", tt.trigger, got, tt.want)
			}
		})
	}
}
