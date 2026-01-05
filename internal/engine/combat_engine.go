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
		state.RecordAction(actor, models.ActionPass, nil, nil, "Stunned", 0, nil, 0, 0, models.ThresholdMinor, 0, 0, 0, 0, true, 0, 0)
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
		state.RecordAction(actor, models.ActionPass, nil, nil, "Passed", 0, nil, 0, 0, models.ThresholdMinor, 0, 0, 0, 0, false, 0, 0)
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

	// Use custom damage dice if set, otherwise default
	damageEffect := models.DamageEffect{
		DiceCount:  1,
		DiceSize:   6,
		FlatBonus:  actor.GetAttackBonus(),
		DamageType: models.DamagePhysical,
	}

	// Parse custom damage dice notation if provided
	if actor.DamageDice != "" {
		notation, err := models.ParseDiceNotation(actor.DamageDice)
		if err == nil {
			damageEffect = notation.ConvertToDamageEffect(models.DamagePhysical)
		}
	}

	result := ce.resolver.ResolveAttack(actor, target, damageEffect, state)

	resultStr := "Miss"
	spotlightShift := false
	if result.Hit {
		resultStr = "Hit"
	} else {
		// Miss causes spotlight to shift
		spotlightShift = true
	}

	// Prepare dice roll info for action record
	hopeDie := 0
	fearDie := 0
	totalRoll := result.AttackRoll
	hopeGenerated := 0
	fearGenerated := 0

	// Check if Hope/Fear result causes spotlight shift (player only)
	if actor.Side == models.SidePlayer {
		hopeDie = result.DualityResult.HopeDie
		fearDie = result.DualityResult.FearDie
		if result.DualityResult.Result == models.ResultFear || result.DualityResult.Result == models.ResultCriticalFear {
			spotlightShift = true
		}
		// Track Hope/Fear generation
		if result.GeneratedHope {
			hopeGenerated = 1
			if result.CriticalResult {
				hopeGenerated = 2
			}
		}
		if result.GeneratedFear {
			fearGenerated = 1
			if result.CriticalResult {
				fearGenerated = 2
			}
		}
	} else {
		// Enemy uses d20 - store in HopeDie field for display
		hopeDie = result.D20Result.D20Roll
		fearDie = 0 // Enemies don't use fear die
	}

	state.RecordAction(actor, models.ActionAttack, target, nil, resultStr,
		result.DamageRoll, result.DamageDice, result.DamageBonus, result.HPDamage, result.Threshold,
		hopeDie, fearDie, totalRoll, target.GetDefense(), spotlightShift, hopeGenerated, fearGenerated)

	return nil
}

// executeAbility handles ability usage
func (ce *CombatEngine) executeAbility(actor *models.Combatant, ability *models.Ability, targets []*models.Combatant, state *models.CombatState) error {
	// Check if conditions are met
	if !ce.resolver.CheckAbilityConditions(actor, ability, state) {
		state.RecordAction(actor, models.ActionUseAbility, nil, ability, "Conditions not met", 0, nil, 0, 0, models.ThresholdMinor, 0, 0, 0, 0, false, 0, 0)
		return nil
	}

	// Check if can afford
	if !ce.resolver.CanAffordAbility(actor, ability, state) {
		state.RecordAction(actor, models.ActionUseAbility, nil, ability, "Cannot afford", 0, nil, 0, 0, models.ThresholdMinor, 0, 0, 0, 0, false, 0, 0)
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

	state.RecordAction(actor, models.ActionUseAbility, targets[0], ability, "Success", 0, nil, 0, 0, models.ThresholdMinor, 0, 0, 0, 0, false, 0, 0)
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
			state.RecordAction(actor, models.ActionClearCondition, nil, nil, string(condType)+" cleared", 0, nil, 0, 0, models.ThresholdMinor, 0, 0, 0, 0, false, 0, 0)
			return nil
		}
	}

	state.RecordAction(actor, models.ActionClearCondition, nil, nil, "No conditions to clear", 0, nil, 0, 0, models.ThresholdMinor, 0, 0, 0, 0, false, 0, 0)
	return nil
}

// DetermineNextActor chooses who acts next based on Daggerheart spotlight rules
// Returns (next actor, should shift spotlight)
func (ce *CombatEngine) DetermineNextActor(state *models.CombatState, lastActorSide models.Side, spotlightShifted bool) (*models.Combatant, bool) {
	var candidates []*models.Combatant

	// If spotlight shifted, switch sides
	if spotlightShifted {
		if lastActorSide == models.SidePlayer {
			candidates = state.GetAliveEnemies()
		} else {
			candidates = state.GetAlivePlayers()
		}
	} else {
		// No shift - same side keeps spotlight
		if lastActorSide == models.SidePlayer {
			candidates = state.GetAlivePlayers()
		} else {
			candidates = state.GetAliveEnemies()
		}
	}

	if len(candidates) == 0 {
		return nil, false
	}

	// For now, pick randomly from candidates
	// TODO: Add strategic selection
	next := candidates[ce.dice.RollDice(1, len(candidates))-1]
	state.CurrentActor = next

	return next, spotlightShifted
}

// RunCombat executes full combat until victory/defeat
func (ce *CombatEngine) RunCombat(state *models.CombatState, decisionMaker DecisionMaker, maxTurns int) *CombatOutcome {
	turnCount := 0
	roundCount := 0

	// Players start with spotlight
	currentSide := models.SidePlayer
	spotlightShifted := false

	for !state.IsCombatOver() && turnCount < maxTurns {
		// Determine next actor based on spotlight
		actor, _ := ce.DetermineNextActor(state, currentSide, spotlightShifted)
		if actor == nil {
			break
		}

		// Update current side
		currentSide = actor.Side

		// AI chooses action
		action := decisionMaker.ChooseAction(state, actor)

		// Store action history length before action
		actionsBeforeTurn := len(state.ActionHistory)

		// Execute turn
		ce.ExecuteTurn(actor, action, state)

		// Check if spotlight shifted based on last action
		spotlightShifted = false
		if len(state.ActionHistory) > actionsBeforeTurn {
			lastAction := state.ActionHistory[len(state.ActionHistory)-1]
			spotlightShifted = lastAction.SpotlightShift
		}

		// Enemy turn logic: enemy gets ONE turn, then spotlight automatically goes to players
		// (unless enemy has Relentless feature - not implemented yet)
		if actor.Side == models.SideEnemy && !spotlightShifted {
			// Enemy had a turn without causing shift - force spotlight back to players
			spotlightShifted = true
		}

		state.IncrementTurn()
		turnCount++

		// Track rounds (every 2 turns approximately)
		if turnCount%2 == 0 {
			state.IncrementRound()
			roundCount++
		}
	}

	outcome := &CombatOutcome{
		PlayerVictory: state.IsPlayerVictory(),
		EnemyVictory:  state.IsEnemyVictory(),
		Turns:         turnCount,
		Rounds:        roundCount,
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
