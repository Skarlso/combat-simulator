package models

// ActiveCondition represents a status effect currently on a combatant
type ActiveCondition struct {
	Type          ConditionType
	Duration      DurationType
	RoundsLeft    int // For DurationRounds type
	DamagePerTick int // For burning, poison, etc.
	DamageType    DamageType
	// Difficulty of the action roll a PC makes to clear this (0 = DefaultClearDifficulty)
	ClearDifficulty int
}

// DefaultClearDifficulty is the SRD Tier 1 adversary Difficulty benchmark
const DefaultClearDifficulty = 11

// Tick decrements duration for round-based conditions
func (c *ActiveCondition) Tick() bool {
	if c.Duration == DurationRounds {
		c.RoundsLeft--
		return c.RoundsLeft <= 0 // Returns true if expired
	}
	return false
}

// ShouldExpire checks if condition should be removed
func (c *ActiveCondition) ShouldExpire(trigger TriggerType) bool {
	switch c.Duration {
	case DurationInstant:
		return true
	case DurationEndOfTurn:
		return trigger == TriggerTurnEnd
	case DurationStartNextTurn:
		return trigger == TriggerTurnStart
	case DurationRounds:
		return c.RoundsLeft <= 0
	case DurationUntilCleared:
		return false // Must be manually cleared
	default:
		return false
	}
}
