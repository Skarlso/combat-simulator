package engine

import (
	"fmt"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)

// AdversaryRegistry interface for summoning (same as in feature_resolver.go)
type AdversaryRegistry interface {
	CloneTemplate(name string, uniqueID string) (*models.Combatant, error)
}

// CombatEngine orchestrates the entire combat simulation
type CombatEngine struct {
	dice            DiceRoller
	spotlight       SpotlightSelector
	resolver        *CombatResolver
	featureResolver *FeatureResolver
}

// NewCombatEngine creates a combat engine with all components
func NewCombatEngine(dice DiceRoller, spotlight SpotlightSelector) *CombatEngine {
	// Extract RNG from DiceRoller if it's a StandardDiceRoller
	var featureResolver *FeatureResolver
	if stdDice, ok := dice.(*StandardDiceRoller); ok {
		featureResolver = NewFeatureResolver(stdDice.GetRNG())
	} else {
		// Fallback: create new resolver without RNG (will cause issues if needed)
		featureResolver = &FeatureResolver{dice: dice}
	}

	resolver := NewCombatResolver(dice, featureResolver)

	return &CombatEngine{
		dice:            dice,
		spotlight:       spotlight,
		resolver:        resolver,
		featureResolver: featureResolver,
	}
}

// SetAdversaryRegistry sets the registry for summoning support
func (ce *CombatEngine) SetAdversaryRegistry(registry AdversaryRegistry) {
	ce.featureResolver.SetAdversaryRegistry(registry)
}

// ExecuteTurnWithDecision runs a single combatant's turn based on AI decision
func (ce *CombatEngine) ExecuteTurnWithDecision(actor *models.Combatant, decision *strategy.ActionDecision, state *models.CombatState) error {
	// Process start-of-turn conditions
	ce.resolver.ProcessConditionTicks(actor, models.TriggerTurnStart)

	// Process start-of-turn triggered features
	ce.featureResolver.ProcessTriggeredFeatures(actor, nil, state, "start_turn", false)

	// Check if stunned or otherwise unable to act
	if actor.HasCondition(models.ConditionStunned) {
		state.RecordAction(actor, models.ActionPass, nil, "", "Stunned", 0, 0, nil, 0, 0, models.ThresholdMinor, 0, 0, 0, 0, true, 0, 0)
		return nil
	}

	target := decision.Target

	// Track whether an action was taken (ability activation consumes turn even if it fails)
	actionTaken := false

	// Execute feature if AI chose one
	if decision.DomainAbility != nil {
		// Check if can activate BEFORE attempting
		if ce.featureResolver.CanActivateDomainAbility(*decision.DomainAbility, actor, state) {
			// Can activate - attempt it (may still fail DC check, but that's their turn)
			ce.executeAbility(actor, target, decision.DomainAbility, state)
			actionTaken = true
		}
		// If can't activate, fall through to basic attack
	} else if decision.AdversaryFeature != nil {
		// Check if can activate BEFORE attempting
		if ce.featureResolver.CanActivateAdversaryFeature(*decision.AdversaryFeature, actor, state) {
			// Can activate - attempt it (may still fail DC check, but that's their turn)
			ce.executeFeature(actor, target, decision.AdversaryFeature, state)
			actionTaken = true
		}
		// If can't activate, fall through to basic attack
	}

	// Fall back to basic attack if no ability was chosen or couldn't be activated
	if !actionTaken && target != nil && target.IsAlive() {
		ce.executeAttack(actor, target, state)
	}

	// Process end-of-turn conditions
	ce.resolver.ProcessConditionTicks(actor, models.TriggerTurnEnd)

	// Process end-of-turn triggered features
	ce.featureResolver.ProcessTriggeredFeatures(actor, nil, state, "end_turn", false)

	return nil
}

// executeAbility activates a domain ability (player)
// Returns error if ability failed (costs already paid if applicable)
func (ce *CombatEngine) executeAbility(actor *models.Combatant, target *models.Combatant, ability *models.DomainAbility, state *models.CombatState) error {
	// Use proper activation which handles costs, DC checks, tokens, etc.
	err, dcResult := ce.featureResolver.ActivateDomainAbility(ability, actor, target, state)

	result := "Success"
	spotlightShift := dcResult.SpotlightShift // Start with Fear result spotlight shift

	if err != nil {
		result = fmt.Sprintf("Failed: %v", err)
		// Failed DC check shifts spotlight (like a missed attack)
		spotlightShift = true
	}

	// Record action with DC check and damage details
	state.RecordActionWithFeature(
		actor,
		models.ActionUseAbility,
		target,
		ability.Name,
		err == nil,
		result,
		dcResult.DamageRoll, dcResult.ProcessedDamage, nil, 0, dcResult.DamageDealt, dcResult.Threshold,
		dcResult.HopeDie,
		dcResult.FearDie,
		dcResult.D20Roll,
		dcResult.TotalRoll,
		dcResult.DC,
		spotlightShift,
		dcResult.HopeGenerated,
		dcResult.FearGenerated,
	)

	return err
}

// executeFeature activates an adversary feature (enemy)
// Returns error if feature failed (costs already paid if applicable)
func (ce *CombatEngine) executeFeature(actor *models.Combatant, target *models.Combatant, feature *models.AdversaryFeature, state *models.CombatState) error {
	// Use proper activation which handles costs, DC checks, spotlight limits, etc.
	err, dcResult := ce.featureResolver.ActivateAdversaryFeature(*feature, actor, target, state)

	result := "Success"
	spotlightShift := dcResult.SpotlightShift // Start with any spotlight shift from roll

	if err != nil {
		result = fmt.Sprintf("Failed: %v", err)
		// Failed DC check shifts spotlight (like a missed attack)
		spotlightShift = true
	}

	// Record action with DC check and damage details
	state.RecordActionWithFeature(
		actor,
		models.ActionUseAbility,
		target,
		feature.Name,
		err == nil,
		result,
		dcResult.DamageRoll, dcResult.ProcessedDamage, nil, 0, dcResult.DamageDealt, dcResult.Threshold,
		dcResult.HopeDie,
		dcResult.FearDie,
		dcResult.D20Roll,
		dcResult.TotalRoll,
		dcResult.DC,
		spotlightShift,
		dcResult.HopeGenerated,
		dcResult.FearGenerated,
	)

	return err
}

// executeAttack handles basic attack action
func (ce *CombatEngine) executeAttack(actor *models.Combatant, target *models.Combatant, state *models.CombatState) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	// Process on_attack triggered features BEFORE attack resolves
	ce.featureResolver.ProcessTriggeredFeatures(actor, target, state, "on_attack", false)

	// Use DamageDice notation, default to 1d6 if not set
	damageDice := actor.DamageDice
	if damageDice == "" {
		damageDice = "1d6"
	}

	result := ce.resolver.ResolveAttack(actor, target, damageDice, state)

	// Process on_critical triggered features if attack was a critical
	if result.CriticalResult {
		ce.featureResolver.ProcessTriggeredFeatures(actor, target, state, "on_critical", true)
	}

	resultStr := "Miss"
	spotlightShift := false
	if result.Hit {
		resultStr = "Hit"
		// Process on_hit triggered features (e.g., spend Hope/Fear/Stress for bonus effects)
		ce.featureResolver.ProcessTriggeredFeatures(actor, target, state, "on_hit", false)
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
		// Track Hope/Fear generation (always 1 per roll, even on critical)
		if result.GeneratedHope {
			hopeGenerated = 1
		}
		if result.GeneratedFear {
			fearGenerated = 1
		}
	} else {
		// Enemy uses d20 - store in HopeDie field for display
		hopeDie = result.D20Result.D20Roll
		fearDie = 0 // Enemies don't use fear die
	}

	state.RecordAction(actor, models.ActionAttack, target, "", resultStr,
		result.DamageRoll, result.ProcessedDamage, result.DamageDice, result.DamageBonus, result.HPDamage, result.Threshold,
		hopeDie, fearDie, totalRoll, target.GetDefense(), spotlightShift, hopeGenerated, fearGenerated)

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
func (ce *CombatEngine) RunCombat(state *models.CombatState, decisionMaker DecisionMaker, maxTurns int, enemyFearUsage strategy.ResourceUsageStrategy) *CombatOutcome {
	turnCount := 0
	roundCount := 0

	// Players start with spotlight
	currentSide := models.SidePlayer
	spotlightShifted := false

	for !state.IsCombatOver() && turnCount < maxTurns {
		// Check for aggressive Fear usage: if players have gone more than 3 times in a row,
		// GM can spend 1 Fear to grab spotlight
		if state.ConsecutivePlayerTurns > 3 &&
		   enemyFearUsage == strategy.ResourceGreedy &&
		   state.FearPool >= 1 &&
		   len(state.GetAliveEnemies()) > 0 {
			// Force spotlight grab using Fear
			state.SpendFear(1)
			spotlightShifted = true
			state.ConsecutivePlayerTurns = 0
			// Record this special action with a system actor
			gmAction := &models.Combatant{Name: "GM"}
			state.RecordAction(gmAction, models.ActionPass, nil, "",
				"Spends 1 Fear to grab spotlight (aggressive strategy)",
				0, 0, nil, 0, 0, models.ThresholdMinor, 0, 0, 0, 0, true, 0, 0)
		}

		// Determine next actor based on spotlight
		actor, _ := ce.DetermineNextActor(state, currentSide, spotlightShifted)
		if actor == nil {
			break
		}

		// Update current side and track consecutive player turns
		currentSide = actor.Side
		if actor.Side == models.SidePlayer {
			state.ConsecutivePlayerTurns++
		} else {
			state.ConsecutivePlayerTurns = 0
		}

		// AI chooses action
		decision := decisionMaker.ChooseAction(state, actor)
		if decision.Target == nil {
			break // No valid target
		}

		// Store action history length before action
		actionsBeforeTurn := len(state.ActionHistory)

		// Execute turn based on decision
		ce.ExecuteTurnWithDecision(actor, decision, state)

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
	ChooseAction(state *models.CombatState, actor *models.Combatant) *strategy.ActionDecision
}
