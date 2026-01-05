package engine

import (
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// CombatEngine orchestrates the entire combat simulation
type CombatEngine struct {
	dice      DiceRoller
	spotlight SpotlightSelector
	resolver  *CombatResolver
	effects   *EffectApplier
}

// NewCombatEngine creates a combat engine with all components
func NewCombatEngine(dice DiceRoller, spotlight SpotlightSelector) *CombatEngine {
	resolver := NewCombatResolver(dice)
	effects := NewEffectApplier(resolver)

	return &CombatEngine{
		dice:      dice,
		spotlight: spotlight,
		resolver:  resolver,
		effects:   effects,
	}
}

// ExecuteTurn runs a single combatant's turn
func (ce *CombatEngine) ExecuteTurn(actor *models.Combatant, action *models.Action, state *models.CombatState) error {
	// Process start-of-turn conditions
	ce.resolver.ProcessConditionTicks(actor, models.TriggerTurnStart)

	// Check if stunned or otherwise unable to act
	if actor.HasCondition(models.ConditionStunned) {
		state.RecordAction(actor, models.ActionPass, nil, nil, "Stunned", 0)
		return nil
	}

	// Execute action
	switch action.Type {
	case models.ActionAttack:
		return ce.executeAttack(actor, action.Targets, state)
	case models.ActionUseAbility:
		return ce.executeAbility(actor, action.Ability, action.Targets, state)
	case models.ActionClearCondition:
		return ce.executeClearCondition(actor, state)
	case models.ActionPass:
		state.RecordAction(actor, models.ActionPass, nil, nil, "Passed", 0)
	}

	// Process end-of-turn conditions
	ce.resolver.ProcessConditionTicks(actor, models.TriggerTurnEnd)

	return nil
}

// executeAttack handles basic attack action
func (ce *CombatEngine) executeAttack(actor *models.Combatant, targets []*models.Combatant, state *models.CombatState) error {
	if len(targets) == 0 || targets[0] == nil {
		return nil
	}

	target := targets[0]

	// Basic attack uses attack bonus for damage
	damageEffect := models.DamageEffect{
		DiceCount:  1,
		DiceSize:   6,
		FlatBonus:  actor.GetAttackBonus(),
		DamageType: models.DamagePhysical,
	}

	result := ce.resolver.ResolveAttack(actor, target, damageEffect, state)

	resultStr := "Miss"
	if result.Hit {
		resultStr = "Hit"
	}

	state.RecordAction(actor, models.ActionAttack, target, nil, resultStr, result.Damage)

	return nil
}

// executeAbility handles ability usage
func (ce *CombatEngine) executeAbility(actor *models.Combatant, ability *models.Ability, targets []*models.Combatant, state *models.CombatState) error {
	// Check if conditions are met
	if !ce.resolver.CheckAbilityConditions(actor, ability, state) {
		state.RecordAction(actor, models.ActionUseAbility, nil, ability, "Conditions not met", 0)
		return nil
	}

	// Check if can afford
	if !ce.resolver.CanAffordAbility(actor, ability, state) {
		state.RecordAction(actor, models.ActionUseAbility, nil, ability, "Cannot afford", 0)
		return nil
	}

	// Pay costs
	if err := ce.resolver.PayAbilityCost(actor, ability, state); err != nil {
		return err
	}

	// Apply effects
	if err := ce.effects.ApplyAllEffects(ability, actor, targets, state); err != nil {
		return err
	}

	targetName := ""
	if len(targets) > 0 && targets[0] != nil {
		targetName = targets[0].Name
	}

	state.RecordAction(actor, models.ActionUseAbility, targets[0], ability, "Success", 0)
	_ = targetName // Avoid unused warning

	return nil
}

// executeClearCondition handles clearing a harmful condition
func (ce *CombatEngine) executeClearCondition(actor *models.Combatant, state *models.CombatState) error {
	// Find most harmful condition to clear
	// Priority: Stunned > Burning/Poison > Disadvantage
	priorities := []models.ConditionType{
		models.ConditionStunned,
		models.ConditionBurning,
		models.ConditionPoisoned,
		models.ConditionDisadvantage,
		models.ConditionRestrained,
	}

	for _, condType := range priorities {
		if actor.RemoveCondition(condType) {
			state.RecordAction(actor, models.ActionClearCondition, nil, nil, string(condType)+" cleared", 0)
			return nil
		}
	}

	state.RecordAction(actor, models.ActionClearCondition, nil, nil, "No conditions to clear", 0)
	return nil
}

// SelectNextActor chooses who acts next using spotlight system
func (ce *CombatEngine) SelectNextActor(state *models.CombatState) *models.Combatant {
	alive := state.GetAliveCombatants()
	if len(alive) == 0 {
		return nil
	}

	// Use spotlight selector
	next := ce.spotlight.ChooseNext(alive, state.SpotlightThreshold)

	// Grant spotlight tokens to everyone else
	for _, combatant := range alive {
		if combatant != next {
			combatant.AddSpotlightToken()
		}
	}

	// Reset actor's tokens
	if next != nil {
		next.ResetSpotlightTokens()
		state.CurrentActor = next
	}

	return next
}

// RunCombat executes full combat until victory/defeat
func (ce *CombatEngine) RunCombat(state *models.CombatState, decisionMaker DecisionMaker, maxTurns int) *CombatOutcome {
	turnCount := 0

	for !state.IsCombatOver() && turnCount < maxTurns {
		// Select next actor
		actor := ce.SelectNextActor(state)
		if actor == nil {
			break
		}

		// AI chooses action
		action := decisionMaker.ChooseAction(state, actor)

		// Execute turn
		ce.ExecuteTurn(actor, action, state)

		state.IncrementTurn()
		turnCount++

		// Track rounds (approximate)
		if turnCount%len(state.GetAllCombatants()) == 0 {
			state.IncrementRound()
		}
	}

	outcome := &CombatOutcome{
		PlayerVictory: state.IsPlayerVictory(),
		EnemyVictory:  state.IsEnemyVictory(),
		Turns:         turnCount,
		Rounds:        state.RoundNumber,
		FinalHopePool: state.HopePool,
		FinalFearPool: state.FearPool,
		ActionHistory: state.ActionHistory,
	}

	// Gather survivor stats
	for _, p := range state.Players {
		if p.IsAlive() {
			outcome.SurvivingPlayers = append(outcome.SurvivingPlayers, p.Name)
		}
	}

	for _, e := range state.Enemies {
		if e.IsAlive() {
			outcome.SurvivingEnemies = append(outcome.SurvivingEnemies, e.Name)
		}
	}

	return outcome
}

// CombatOutcome summarizes combat results
type CombatOutcome struct {
	PlayerVictory     bool
	EnemyVictory      bool
	Turns             int
	Rounds            int
	FinalHopePool     int
	FinalFearPool     int
	SurvivingPlayers  []string
	SurvivingEnemies  []string
	ActionHistory     []models.ActionRecord
}

// DecisionMaker is an interface for AI/strategy to choose actions
type DecisionMaker interface {
	ChooseAction(state *models.CombatState, actor *models.Combatant) *models.Action
}
