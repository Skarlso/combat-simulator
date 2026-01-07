package models

// DamageType represents different damage categories
type DamageType int

const (
	DamagePhysical DamageType = iota
	DamageMagic
	DamageFire
	DamageFrost
	DamagePoison
	DamagePhysicalMagic // Hybrid damage type
)

func (d DamageType) String() string {
	switch d {
	case DamagePhysical:
		return "Physical"
	case DamageMagic:
		return "Magic"
	case DamageFire:
		return "Fire"
	case DamageFrost:
		return "Frost"
	case DamagePoison:
		return "Poison"
	case DamagePhysicalMagic:
		return "Physical/Magic"
	default:
		return "Unknown"
	}
}

// DamageThreshold represents the severity tiers for damage
type DamageThreshold int

const (
	ThresholdMinor DamageThreshold = iota // 1-3 damage
	ThresholdMajor                         // 4-6 damage
	ThresholdSevere                        // 7+ damage
)

// DamageThresholdFromAmount determines threshold based on combatant's custom thresholds
func DamageThresholdFromAmount(damage int, combatant *Combatant) DamageThreshold {
	thresholds := combatant.CustomThresholds
	if len(thresholds) == 0 {
		// For PCs: base threshold + level
		// For adversaries: use tier-based default
		if !combatant.IsAdversary {
			// PC thresholds scale with level
			// Base armor thresholds: Minor=3, Major=6
			// Add level to each threshold
			minorThreshold := 3 + combatant.Level
			majorThreshold := 6 + combatant.Level
			thresholds = []int{minorThreshold, majorThreshold}
		} else {
			// Default adversary thresholds: <4=Minor, 4-6=Major, 7+=Severe
			thresholds = []int{4, 7}
		}
	}

	switch {
	case damage >= thresholds[1]:
		return ThresholdSevere
	case damage >= thresholds[0]:
		return ThresholdMajor
	default:
		return ThresholdMinor
	}
}

// TriggerType defines when an ability can activate
type TriggerType string

const (
	TriggerTurnStart     TriggerType = "on_turn_start"
	TriggerTurnEnd       TriggerType = "on_turn_end"
	TriggerAttack        TriggerType = "on_attack"
	TriggerAttackSuccess TriggerType = "on_attack_success"
	TriggerHit           TriggerType = "on_hit"
	TriggerDamaged       TriggerType = "on_damaged"
	TriggerAllyDowned    TriggerType = "on_ally_downed"
	TriggerEnemyDowned   TriggerType = "on_enemy_downed"
	TriggerManual        TriggerType = "manual" // Player chooses when to use
)

// ConditionType represents status effects
type ConditionType string

const (
	ConditionBurning      ConditionType = "burning"
	ConditionPoisoned     ConditionType = "poisoned"
	ConditionCloaked      ConditionType = "cloaked"
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
	default:
		return "Unknown"
	}
}

// DiceResult represents outcome of duality dice roll
type DiceResult int

const (
	ResultHope DiceResult = iota
	ResultFear
	ResultCriticalHope // Natural 12 on Hope die
	ResultCriticalFear // Natural 12 on Fear die
)

func (d DiceResult) String() string {
	switch d {
	case ResultHope:
		return "Hope"
	case ResultFear:
		return "Fear"
	case ResultCriticalHope:
		return "Critical Hope"
	case ResultCriticalFear:
		return "Critical Fear"
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
