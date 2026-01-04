package models

// ActiveCondition represents a status effect currently on a combatant
type ActiveCondition struct {
	Type         ConditionType
	Duration     DurationType
	RoundsLeft   int // For DurationRounds type
	DamagePerTick int // For burning, poison, etc.
	DamageType   DamageType
	Source       string // Who/what applied this condition
}

// NewCondition creates a basic condition
func NewCondition(condType ConditionType, duration DurationType) *ActiveCondition {
	return &ActiveCondition{
		Type:     condType,
		Duration: duration,
	}
}

// NewDOTCondition creates a damage-over-time condition (burning, poison)
func NewDOTCondition(condType ConditionType, damagePerTick int, damageType DamageType, rounds int) *ActiveCondition {
	return &ActiveCondition{
		Type:          condType,
		Duration:      DurationRounds,
		RoundsLeft:    rounds,
		DamagePerTick: damagePerTick,
		DamageType:    damageType,
	}
}

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
