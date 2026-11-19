package models

// DamageType represents different damage categories
type DamageType int

const (
	DamagePhysical DamageType = iota
	DamageMagic
)

func (d DamageType) String() string {
	switch d {
	case DamagePhysical:
		return "Physical"
	case DamageMagic:
		return "Magic"
	default:
		return "Unknown"
	}
}

// DamageThreshold represents the severity tiers for damage
type DamageThreshold int

const (
	ThresholdMinor  DamageThreshold = iota // below the Major threshold
	ThresholdMajor                         // at or above Major, below Severe
	ThresholdSevere                        // at or above Severe
)

// String returns the string representation of the threshold
func (dt DamageThreshold) String() string {
	switch dt {
	case ThresholdNone:
		return "None"
	case ThresholdMinor:
		return "Minor"
	case ThresholdMajor:
		return "Major"
	case ThresholdSevere:
		return "Severe"
	default:
		return "Unknown"
	}
}

// ThresholdNone represents zero damage (no HP marked).
const ThresholdNone DamageThreshold = -1

// HPMarked is how many Hit Points a hit of this severity marks
func (dt DamageThreshold) HPMarked() int {
	return int(dt) + 1
}

// DamageThresholdFromAmount determines threshold based on combatant's custom thresholds.
// Returns ThresholdNone for zero damage per SRD.
func DamageThresholdFromAmount(damage int, combatant *Combatant) DamageThreshold {
	if damage <= 0 {
		return ThresholdNone
	}

	thresholds := combatant.CustomThresholds
	if len(thresholds) < 2 {
		if !combatant.IsAdversary {
			minorThreshold := 3 + combatant.Level
			majorThreshold := 6 + combatant.Level
			thresholds = []int{minorThreshold, majorThreshold}
		} else {
			thresholds = []int{4, 7}
		}
	}

	// A threshold of 0 is the stat block's "None": that tier can never be reached
	switch {
	case thresholds[1] > 0 && damage >= thresholds[1]:
		return ThresholdSevere
	case thresholds[0] > 0 && damage >= thresholds[0]:
		return ThresholdMajor
	default:
		return ThresholdMinor
	}
}

// TriggerType defines when condition ticks should be processed
type TriggerType string

const (
	TriggerTurnStart TriggerType = "on_turn_start"
	TriggerTurnEnd   TriggerType = "on_turn_end"
)

// ConditionType represents status effects
type ConditionType string

const (
	ConditionBurning      ConditionType = "burning"
	ConditionPoisoned     ConditionType = "poisoned"
	ConditionVulnerable   ConditionType = "vulnerable"
	ConditionResistant    ConditionType = "resistant"
	ConditionStunned      ConditionType = "stunned"
	ConditionRestrained   ConditionType = "restrained"
	ConditionAdvantage    ConditionType = "advantage"
	ConditionDisadvantage ConditionType = "disadvantage"
	ConditionTaunted      ConditionType = "taunted"
	ConditionHidden       ConditionType = "hidden"
)

// DurationType defines how long conditions persist
type DurationType string

const (
	DurationInstant       DurationType = "instant"
	DurationEndOfTurn     DurationType = "end_of_turn"
	DurationStartNextTurn DurationType = "start_of_next_turn"
	DurationUntilCleared  DurationType = "until_cleared"
	DurationRounds        DurationType = "rounds" // Specific round count
)

// ActionType defines what action a combatant takes
type ActionType int

const (
	ActionAttack ActionType = iota
	ActionUseAbility
	ActionClearCondition
	ActionPass
	ActionDeathMove
	ActionReactionRoll
)

func (a ActionType) String() string {
	switch a {
	case ActionAttack:
		return "Attack"
	case ActionUseAbility:
		return "Use Ability"
	case ActionClearCondition:
		return "Clear Condition"
	case ActionPass:
		return "Pass"
	case ActionDeathMove:
		return "Death Move"
	case ActionReactionRoll:
		return "Reaction Roll"
	default:
		return "Unknown"
	}
}

// DiceResult represents outcome of duality dice roll
type DiceResult int

const (
	ResultHope DiceResult = iota
	ResultFear
	ResultCriticalHope // Matching dice = critical success (always "with Hope" per SRD)
)

func (d DiceResult) String() string {
	switch d {
	case ResultHope:
		return "Hope"
	case ResultFear:
		return "Fear"
	case ResultCriticalHope:
		return "Critical Hope"
	default:
		return "Unknown"
	}
}

// Side represents combatant alignment
type Side int

const (
	SidePlayer Side = iota
	SideEnemy
)

func (s Side) String() string {
	switch s {
	case SidePlayer:
		return "Player"
	case SideEnemy:
		return "Enemy"
	default:
		return "Unknown"
	}
}
