package engine

import (
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// CombatEngine orchestrates the entire combat simulation
type CombatEngine struct {
	dice            DiceRoller
	spotlight       SpotlightSelector
	resolver        *CombatResolver
	featureResolver *FeatureResolver
}

// NewCombatEngine creates a combat engine with all components
func NewCombatEngine(dice DiceRoller, spotlight SpotlightSelector) *CombatEngine {
	resolver := NewCombatResolver(dice)

	// Extract RNG from DiceRoller if it's a StandardDiceRoller
	var featureResolver *FeatureResolver
	if stdDice, ok := dice.(*StandardDiceRoller); ok {
		featureResolver = NewFeatureResolver(stdDice.GetRNG())
	} else {
		// Fallback: create new resolver without RNG (will cause issues if needed)
		featureResolver = &FeatureResolver{dice: dice}
	}

	return &CombatEngine{
		dice:            dice,
		spotlight:       spotlight,
		resolver:        resolver,
		featureResolver: featureResolver,
	}
}

// ExecuteTurn runs a single combatant's turn - performs a basic attack or activates features
func (ce *CombatEngine) ExecuteTurn(actor *models.Combatant, target *models.Combatant, state *models.CombatState) error {
	// Process start-of-turn conditions
	ce.resolver.ProcessConditionTicks(actor, models.TriggerTurnStart)

	// Check if stunned or otherwise unable to act
	if actor.HasCondition(models.ConditionStunned) {
		state.RecordAction(actor, models.ActionPass, nil, "", "Stunned", 0, nil, 0, 0, models.ThresholdMinor, 0, 0, 0, 0, true, 0, 0)
		return nil
	}

	// Try to activate a feature/ability first (simple logic: 30% chance)
	if ce.dice.RollDice(1, 10) <= 3 {
		if ce.tryActivateFeature(actor, target, state) {
			// Feature was activated, turn complete
			ce.resolver.ProcessConditionTicks(actor, models.TriggerTurnEnd)
			return nil
		}
	}

	// Execute basic attack if no feature used
	if target != nil && target.IsAlive() {
		ce.executeAttack(actor, target, state)
	}

	// Process end-of-turn conditions
	ce.resolver.ProcessConditionTicks(actor, models.TriggerTurnEnd)

	return nil
}

// tryActivateFeature attempts to activate a feature or domain ability
// Returns true if a feature was successfully activated
func (ce *CombatEngine) tryActivateFeature(actor *models.Combatant, target *models.Combatant, state *models.CombatState) bool {
	// Try player domain abilities first
	if actor.Side == models.SidePlayer && len(actor.DomainAbilities) > 0 {
		for _, ability := range actor.DomainAbilities {
			// Check if can afford
			if ability.HopeCost > state.HopePool {
				continue
			}
			if ability.StressCost+actor.Stress > actor.StressMax {
				continue
			}

			// Pay costs
			state.SpendHope(ability.HopeCost)
			actor.AddStress(ability.StressCost)

			// Apply effects
			for _, effect := range ability.Effects {
				ce.featureResolver.ResolveEffect(effect, actor, target, state)
			}

			// Record action
			state.RecordActionWithFeature(actor, models.ActionUseAbility, target, ability.Name, true, "Success", 0, nil, 0, 0, models.ThresholdMinor, 0, 0, 0, 0, false, 0, 0)
			return true
		}
	}

	// Try enemy adversary features
	if actor.Side == models.SideEnemy && len(actor.AdversaryFeatures) > 0 {
		for _, feature := range actor.AdversaryFeatures {
			// Check if can afford
			if feature.FearCost > state.FearPool {
				continue
			}

			// Pay costs
			state.SpendFear(feature.FearCost)

			// Apply effects
			for _, effect := range feature.Effects {
				ce.featureResolver.ResolveEffect(effect, actor, target, state)
			}

			// Record action
			state.RecordActionWithFeature(actor, models.ActionUseAbility, target, feature.Name, true, "Success", 0, nil, 0, 0, models.ThresholdMinor, 0, 0, 0, 0, false, 0, 0)
			return true
		}
	}

	return false
}

// executeAttack handles basic attack action
func (ce *CombatEngine) executeAttack(actor *models.Combatant, target *models.Combatant, state *models.CombatState) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	// Use DamageDice notation, default to 1d6 if not set
	damageDice := actor.DamageDice
	if damageDice == "" {
		damageDice = "1d6"
	}

	result := ce.resolver.ResolveAttack(actor, target, damageDice, state)

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

	state.RecordAction(actor, models.ActionAttack, target, "", resultStr,
		result.DamageRoll, result.DamageDice, result.DamageBonus, result.HPDamage, result.Threshold,
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

// DecisionMaker is an interface for AI/strategy to choose targets
type DecisionMaker interface {
	ChooseAction(state *models.CombatState, actor *models.Combatant) *models.Combatant
}
