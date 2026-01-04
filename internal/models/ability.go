package models

// Ability represents a skill, attack, or special action
type Ability struct {
	Name        string
	Description string

	// When can this ability be used?
	Trigger TriggerType

	// What must be true to use it?
	Conditions []AbilityCondition

	// What does it cost?
	Costs ResourceCost

	// What happens when used?
	Effects []Effect

	// Targeting
	TargetSelf       bool
	TargetSingle     bool
	TargetAOE        bool
	AOEMaxTargets    int
	TargetAllies     bool
	TargetEnemies    bool
	TargetSelfOption bool // Can target self OR others
}

// ResourceCost defines what using an ability costs
type ResourceCost struct {
	Stress int
	HP     int
	Hope   int // For player abilities
	Fear   int // For enemy abilities
}

// IsEmpty returns true if ability has no cost
func (r ResourceCost) IsEmpty() bool {
	return r.Stress == 0 && r.HP == 0 && r.Hope == 0 && r.Fear == 0
}

// AbilityCondition defines prerequisites for ability usage
type AbilityCondition struct {
	Type  ConditionCheckType
	Value interface{} // Type depends on ConditionCheckType
}

// ConditionCheckType defines what kind of condition check to perform
type ConditionCheckType string

const (
	ConditionHasStatus       ConditionCheckType = "has_condition"     // Value: ConditionType
	ConditionAllyInMelee     ConditionCheckType = "ally_in_melee"     // Value: nil
	ConditionHPBelow         ConditionCheckType = "hp_below"          // Value: int (percentage)
	ConditionHPAbove         ConditionCheckType = "hp_above"          // Value: int (percentage)
	ConditionStressBelow     ConditionCheckType = "stress_below"      // Value: int
	ConditionStressAbove     ConditionCheckType = "stress_above"      // Value: int
	ConditionEnemyCount      ConditionCheckType = "enemy_count"       // Value: int (minimum)
	ConditionAllyCount       ConditionCheckType = "ally_count"        // Value: int (minimum)
	ConditionResourceAtLeast ConditionCheckType = "resource_at_least" // Value: ResourceCost
)

// Effect defines what happens when ability is used
type Effect interface {
	GetType() EffectType
}

// EffectType categorizes effects
type EffectType string

const (
	EffectDamage          EffectType = "damage"
	EffectHeal            EffectType = "heal"
	EffectAddCondition    EffectType = "add_condition"
	EffectRemoveCondition EffectType = "remove_condition"
	EffectModifyResource  EffectType = "modify_resource"
	EffectAddDice         EffectType = "add_dice"
	EffectModifyStat      EffectType = "modify_stat"
)

// DamageEffect adds damage to an attack or deals direct damage
type DamageEffect struct {
	DiceCount  int
	DiceSize   int        // d6, d8, d12, etc.
	FlatBonus  int        // Additional flat damage
	DamageType DamageType
}

func (d DamageEffect) GetType() EffectType {
	return EffectDamage
}

// HealEffect restores HP
type HealEffect struct {
	DiceCount int
	DiceSize  int
	FlatBonus int
}

func (h HealEffect) GetType() EffectType {
	return EffectHeal
}

// AddConditionEffect applies a status effect
type AddConditionEffect struct {
	Condition *ActiveCondition
	Target    EffectTarget // Who gets the condition
}

func (a AddConditionEffect) GetType() EffectType {
	return EffectAddCondition
}

// RemoveConditionEffect clears a status effect
type RemoveConditionEffect struct {
	ConditionType ConditionType
	Target        EffectTarget
}

func (r RemoveConditionEffect) GetType() EffectType {
	return EffectRemoveCondition
}

// ModifyResourceEffect changes Hope/Fear/Stress/HP
type ModifyResourceEffect struct {
	ResourceType ResourceType
	Amount       int  // Positive = gain, negative = lose
	TargetSelf   bool // If false, targets global pool (Hope/Fear)
}

func (m ModifyResourceEffect) GetType() EffectType {
	return EffectModifyResource
}

// ResourceType identifies which resource to modify
type ResourceType string

const (
	ResourceHP     ResourceType = "hp"
	ResourceStress ResourceType = "stress"
	ResourceHope   ResourceType = "hope"
	ResourceFear   ResourceType = "fear"
	ResourceArmor  ResourceType = "armor"
)

// AddDiceEffect grants advantage/disadvantage or extra dice
type AddDiceEffect struct {
	DiceCount    int
	DiceSize     int
	ApplyToRoll  bool // True = affects action roll, false = damage roll
	IsAdvantage  bool // Grants advantage on roll
	IsDisadvantage bool
}

func (a AddDiceEffect) GetType() EffectType {
	return EffectAddDice
}

// ModifyStatEffect temporarily changes a stat
type ModifyStatEffect struct {
	Stat     StatType
	Amount   int
	Duration DurationType
}

func (m ModifyStatEffect) GetType() EffectType {
	return EffectModifyStat
}

// StatType identifies which stat to modify
type StatType string

const (
	StatEvasion     StatType = "evasion"
	StatProficiency StatType = "proficiency"
)

// EffectTarget defines who an effect applies to
type EffectTarget string

const (
	TargetSelf   EffectTarget = "self"
	TargetOther  EffectTarget = "target"
	TargetAll    EffectTarget = "all"
	TargetAllies EffectTarget = "allies"
	TargetEnemies EffectTarget = "enemies"
)

// NewBasicAttack creates a standard melee/ranged attack ability
func NewBasicAttack(name string, diceCount, diceSize int, damageType DamageType) *Ability {
	return &Ability{
		Name:          name,
		Description:   "Basic attack",
		Trigger:       TriggerManual,
		Conditions:    []AbilityCondition{},
		Costs:         ResourceCost{},
		TargetSingle:  true,
		TargetEnemies: true,
		Effects: []Effect{
			DamageEffect{
				DiceCount:  diceCount,
				DiceSize:   diceSize,
				DamageType: damageType,
			},
		},
	}
}
