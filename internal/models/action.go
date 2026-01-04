package models

// Action represents a decision made by a combatant
type Action struct {
	Type    ActionType
	Ability *Ability
	Targets []*Combatant
}

// NewAttackAction creates a basic attack action
func NewAttackAction(target *Combatant) *Action {
	return &Action{
		Type:    ActionAttack,
		Targets: []*Combatant{target},
	}
}

// NewAbilityAction creates an ability usage action
func NewAbilityAction(ability *Ability, targets []*Combatant) *Action {
	return &Action{
		Type:    ActionUseAbility,
		Ability: ability,
		Targets: targets,
	}
}

// NewClearConditionAction creates a condition-clearing action
func NewClearConditionAction() *Action {
	return &Action{
		Type: ActionClearCondition,
	}
}

// NewPassAction creates a pass action
func NewPassAction() *Action {
	return &Action{
		Type: ActionPass,
	}
}
