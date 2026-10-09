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
func (ce *CombatEngine) ExecuteTurnWithDecision(actor *models.Combatant, decision *strategy.ActionDecision, state *models.CombatState) {
	// Process start-of-turn conditions
	ce.resolver.ProcessConditionTicks(actor, models.TriggerTurnStart)

	// Process start-of-turn triggered features
	ce.featureResolver.ProcessTriggeredFeatures(actor, nil, state, models.TriggerStartTurn, false)

	// Burning/poison ticks can defeat the actor before they act
	if !actor.IsAlive() {
		return
	}

	// Check if stunned or otherwise unable to act
	if actor.HasCondition(models.ConditionStunned) {
		state.RecordAction(models.ActionParams{Actor: actor, ActionType: models.ActionPass, Result: "Stunned", SpotlightShift: true})
		// A stun costs exactly one turn, whatever its duration
		actor.RemoveCondition(models.ConditionStunned)
		ce.resolver.ProcessConditionTicks(actor, models.TriggerTurnEnd)
		return
	}

	if decision.ClearCondition {
		ce.executeClearCondition(actor, state)
		ce.resolver.ProcessConditionTicks(actor, models.TriggerTurnEnd)
		return
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
		if ce.featureResolver.CanActivateAdversaryFeature(decision.AdversaryFeature, actor, state) {
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
	ce.featureResolver.ProcessTriggeredFeatures(actor, nil, state, models.TriggerEndTurn, false)
}

func (ce *CombatEngine) recordActivation(actor, target *models.Combatant, name string, isFeature bool, err error, dcResult DCCheckResult, state *models.CombatState) {
	result := "Success"
	spotlightShift := dcResult.SpotlightShift

	if err != nil {
		result = fmt.Sprintf("Failed: %v", err)
		spotlightShift = true
	}

	state.RecordAction(models.ActionParams{
		Actor:           actor,
		ActionType:      models.ActionUseAbility,
		Target:          target,
		FeatureName:     name,
		IsFeature:       isFeature,
		Result:          result,
		DamageRoll:      dcResult.DamageRoll,
		ProcessedDamage: dcResult.ProcessedDamage,
		HPDamage:        dcResult.DamageDealt,
		Threshold:       dcResult.Threshold,
		HopeDie:         dcResult.HopeDie,
		FearDie:         dcResult.FearDie,
		D20Roll:         dcResult.D20Roll,
		TotalRoll:       dcResult.TotalRoll,
		TargetDefense:   dcResult.DC,
		SpotlightShift:  spotlightShift,
		HopeGenerated:   dcResult.HopeGenerated,
		FearGenerated:   dcResult.FearGenerated,
		DamageResult:    dcResult.DamageResult,
	})
}

func (ce *CombatEngine) executeAbility(actor *models.Combatant, target *models.Combatant, ability *models.DomainAbility, state *models.CombatState) error {
	err, dcResult := ce.featureResolver.ActivateDomainAbility(ability, actor, target, state)
	ce.recordActivation(actor, target, ability.Name, false, err, dcResult, state)
	return err
}

func (ce *CombatEngine) executeFeature(actor *models.Combatant, target *models.Combatant, feature *models.AdversaryFeature, state *models.CombatState) error {
	err, dcResult := ce.featureResolver.ActivateAdversaryFeature(feature, actor, target, state)
	ce.recordActivation(actor, target, feature.Name, true, err, dcResult, state)
	return err
}

// executeClearCondition spends the actor's turn removing one harmful condition
func (ce *CombatEngine) executeClearCondition(actor *models.Combatant, state *models.CombatState) {
	result := "Nothing to clear"
	if condType, ok := actor.ClearableCondition(); ok {
		actor.RemoveCondition(condType)
		result = fmt.Sprintf("Cleared %s", condType)
	}
	state.RecordAction(models.ActionParams{Actor: actor, ActionType: models.ActionClearCondition, Result: result})
}

// executeAttack handles basic attack action
func (ce *CombatEngine) executeAttack(actor *models.Combatant, target *models.Combatant, state *models.CombatState) error {
	if target == nil || !target.IsAlive() {
		return nil
	}

	// Process on_attack triggered features BEFORE attack resolves
	ce.featureResolver.ProcessTriggeredFeatures(actor, target, state, models.TriggerOnAttack, false)

	// Use DamageDice notation, default to 1d6 if not set
	damageDice := actor.DamageDice
	if damageDice == "" {
		damageDice = "1d6"
	}

	result := ce.resolver.ResolveAttack(actor, target, damageDice, state)

	// Process on_critical triggered features if attack was a critical
	if result.CriticalResult {
		ce.featureResolver.ProcessTriggeredFeatures(actor, target, state, models.TriggerOnCritical, true)
	}

	resultStr := "Miss"
	spotlightShift := false
	if result.Hit {
		resultStr = "Hit"
		// Process on_hit triggered features (e.g., spend Hope/Fear/Stress for bonus effects)
		ce.featureResolver.ProcessTriggeredFeatures(actor, target, state, models.TriggerOnHit, false)
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
		if result.DualityResult.Result == models.ResultFear {
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

	state.RecordAction(models.ActionParams{
		Actor:           actor,
		ActionType:      models.ActionAttack,
		Target:          target,
		Result:          resultStr,
		DamageRoll:      result.DamageRoll,
		ProcessedDamage: result.ProcessedDamage,
		DamageDice:      result.DamageDice,
		DamageBonus:     result.DamageBonus,
		HPDamage:        result.HPDamage,
		Threshold:       result.Threshold,
		HopeDie:         hopeDie,
		FearDie:         fearDie,
		TotalRoll:       totalRoll,
		TargetDefense:   target.GetDefense(),
		SpotlightShift:  spotlightShift,
		HopeGenerated:   hopeGenerated,
		FearGenerated:   fearGenerated,
		DamageResult:    result.DamageResult,
	})

	return nil
}

func (ce *CombatEngine) chooseNextPlayer(state *models.CombatState) *models.Combatant {
	players := state.GetAlivePlayers()
	if len(players) == 0 {
		return nil
	}
	return ce.spotlight.ChooseNext(players, state.SpotlightThreshold)
}

func (ce *CombatEngine) chooseGMAdversary(state *models.CombatState) *models.Combatant {
	eligible := ce.getEligibleAdversaries(state)
	if len(eligible) == 0 {
		return nil
	}
	return ce.spotlight.ChooseNext(eligible, 0)
}

func (ce *CombatEngine) getEligibleAdversaries(state *models.CombatState) []*models.Combatant {
	alive := state.GetAliveEnemies()
	eligible := make([]*models.Combatant, 0, len(alive))
	for _, e := range alive {
		if state.AdversarySpotlightCount(e.ID) < e.MaxSpotlightsPerGMTurn() {
			eligible = append(eligible, e)
		}
	}
	return eligible
}

func (ce *CombatEngine) shouldGMSeizeSpotlight(state *models.CombatState, fearUsage strategy.ResourceUsageStrategy) bool {
	if state.FearPool < 1 || len(state.GetAliveEnemies()) == 0 {
		return false
	}

	consecutive := state.ConsecutivePlayerTurns

	switch fearUsage {
	case strategy.ResourceGreedy:
		return consecutive >= 3
	case strategy.ResourceConservative:
		return consecutive >= 5
	case strategy.ResourceDesperate:
		return false
	case strategy.ResourceRandom:
		return consecutive >= 3 && ce.dice.RollDice(1, 2, 0) == 1
	case strategy.ResourceClever:
		if consecutive >= 3 && state.FearPool >= 3 {
			return true
		}
		return consecutive >= 4
	default:
		return consecutive >= 3
	}
}

func (ce *CombatEngine) shouldSpendFearForAdditional(state *models.CombatState, fearUsage strategy.ResourceUsageStrategy) bool {
	if state.FearPool < 1 || len(ce.getEligibleAdversaries(state)) == 0 {
		return false
	}

	switch fearUsage {
	case strategy.ResourceGreedy:
		return true
	case strategy.ResourceConservative:
		return state.FearPool > 3
	case strategy.ResourceDesperate:
		return len(state.GetAlivePlayers()) > len(state.GetAliveEnemies())
	case strategy.ResourceRandom:
		return ce.dice.RollDice(1, 2, 0) == 1
	case strategy.ResourceClever:
		return state.FearPool > 4 || len(state.GetAlivePlayers()) > len(state.GetAliveEnemies())
	default:
		return false
	}
}

func (ce *CombatEngine) executeAdversarySpotlight(enemy *models.Combatant, state *models.CombatState, dm DecisionMaker) bool {
	state.CurrentActor = enemy
	enemy.AddSpotlightToken()

	decision := dm.ChooseAction(state, enemy)
	if decision.Target == nil {
		return false
	}

	ce.ExecuteTurnWithDecision(enemy, decision, state)
	enemy.ResetSpotlightTokens()
	state.RecordAdversarySpotlight(enemy.ID)
	state.IncrementTurn()
	return true
}

func (ce *CombatEngine) RunCombat(state *models.CombatState, decisionMaker DecisionMaker, maxTurns int, enemyFearUsage strategy.ResourceUsageStrategy) *CombatOutcome {
	turnCount := 0
	roundCount := 0

	for !state.IsCombatOver() && turnCount < maxTurns {
		// === PLAYER PHASE ===
		for !state.IsCombatOver() && turnCount < maxTurns {
			if ce.shouldGMSeizeSpotlight(state, enemyFearUsage) {
				state.SpendFear(1)
				state.ConsecutivePlayerTurns = 0
				gmActor := &models.Combatant{Name: "GM"}
				state.RecordAction(models.ActionParams{
					Actor:          gmActor,
					ActionType:     models.ActionPass,
					Result:         "Spends 1 Fear to seize spotlight",
					SpotlightShift: true,
				})
				break
			}

			player := ce.chooseNextPlayer(state)
			if player == nil {
				break
			}

			state.CurrentActor = player
			player.AddSpotlightToken()

			decision := decisionMaker.ChooseAction(state, player)
			if decision.Target == nil {
				break
			}

			actionsBeforeTurn := len(state.ActionHistory)
			ce.ExecuteTurnWithDecision(player, decision, state)
			player.ResetSpotlightTokens()
			state.IncrementTurn()
			turnCount++
			state.ConsecutivePlayerTurns++

			spotlightShifted := false
			for i := actionsBeforeTurn; i < len(state.ActionHistory); i++ {
				if state.ActionHistory[i].SpotlightShift {
					spotlightShifted = true
					break
				}
			}

			if spotlightShifted {
				break
			}
		}

		if state.IsCombatOver() || turnCount >= maxTurns {
			break
		}

		// === GM TURN ===
		if len(state.GetAliveEnemies()) == 0 {
			continue
		}

		state.BeginGMTurn()
		state.ConsecutivePlayerTurns = 0

		firstEnemy := ce.chooseGMAdversary(state)
		if firstEnemy != nil {
			ce.executeAdversarySpotlight(firstEnemy, state, decisionMaker)
			turnCount++
		}

		for !state.IsCombatOver() && turnCount < maxTurns &&
			ce.shouldSpendFearForAdditional(state, enemyFearUsage) {
			state.SpendFear(1)
			nextEnemy := ce.chooseGMAdversary(state)
			if nextEnemy == nil {
				break
			}
			ce.executeAdversarySpotlight(nextEnemy, state, decisionMaker)
			turnCount++
		}

		state.EndGMTurn()
		state.IncrementRound()
		roundCount++
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
	PlayerVictory    bool
	EnemyVictory     bool
	Turns            int
	Rounds           int
	FinalHopePool    int
	FinalFearPool    int
	SurvivingPlayers []string
	SurvivingEnemies []string
	ActionHistory    []models.ActionRecord
}

// DecisionMaker is an interface for AI/strategy to choose actions
type DecisionMaker interface {
	ChooseAction(state *models.CombatState, actor *models.Combatant) *strategy.ActionDecision
}
