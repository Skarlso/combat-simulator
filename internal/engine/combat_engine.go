package engine

import (
	"fmt"
	"slices"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)

// AdversaryRegistry hands out fresh copies of stat blocks for summoning
type AdversaryRegistry interface {
	CloneTemplate(name string, uniqueID string) (*models.Combatant, error)
}

// CombatEngine orchestrates the entire combat simulation
type CombatEngine struct {
	dice            DiceRoller
	spotlight       SpotlightSelector
	resolver        *CombatResolver
	featureResolver *FeatureResolver
	deathMove       strategy.DeathMoveStrategy
}

// NewCombatEngine creates a combat engine with all components
func NewCombatEngine(dice DiceRoller, spotlight SpotlightSelector) *CombatEngine {
	featureResolver := &FeatureResolver{dice: dice}
	return &CombatEngine{
		dice:            dice,
		spotlight:       spotlight,
		resolver:        NewCombatResolver(dice, featureResolver),
		featureResolver: featureResolver,
	}
}

// SetAdversaryRegistry sets the registry for summoning support
func (ce *CombatEngine) SetAdversaryRegistry(registry AdversaryRegistry) {
	ce.featureResolver.SetAdversaryRegistry(registry)
}

// SetAreaTargetLimit caps how many targets a single area effect reaches (0 = everyone)
func (ce *CombatEngine) SetAreaTargetLimit(limit int) {
	ce.featureResolver.SetAreaTargetLimit(limit)
}

// SetDeathMove sets which death move PCs make when they mark their last Hit Point
func (ce *CombatEngine) SetDeathMove(deathMove strategy.DeathMoveStrategy) {
	ce.deathMove = deathMove
}

// ExecuteTurnWithDecision runs a single combatant's turn based on AI decision
func (ce *CombatEngine) ExecuteTurnWithDecision(actor *models.Combatant, decision *strategy.ActionDecision, state *models.CombatState) {
	ce.executeTurn(actor, decision, state)
	ce.resolveDeathMoves(state)
}

func (ce *CombatEngine) executeTurn(actor *models.Combatant, decision *strategy.ActionDecision, state *models.CombatState) {
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

	if decision.Potion != "" {
		ce.drinkPotion(actor, decision.Potion, state)
		ce.resolver.ProcessConditionTicks(actor, models.TriggerTurnEnd)
		return
	}

	if decision.ClearCondition {
		ce.executeClearCondition(actor, state)
		ce.resolver.ProcessConditionTicks(actor, models.TriggerTurnEnd)
		return
	}

	target := decision.Target

	// An activation spends the turn even when its roll fails; an unaffordable one falls back to an attack
	actionTaken := false
	switch {
	case decision.DomainAbility != nil && ce.featureResolver.CanActivateDomainAbility(*decision.DomainAbility, actor, state):
		ce.executeAbility(actor, target, decision.DomainAbility, state)
		actionTaken = true
	case decision.AdversaryFeature != nil && ce.featureResolver.CanActivateAdversaryFeature(decision.AdversaryFeature, actor, state):
		ce.executeFeature(actor, target, decision.AdversaryFeature, state)
		actionTaken = true
	}

	if !actionTaken && target != nil && target.IsAlive() {
		ce.executeAttack(actor, target, state)
	}

	// Process end-of-turn conditions
	ce.resolver.ProcessConditionTicks(actor, models.TriggerTurnEnd)

	// Process end-of-turn triggered features
	ce.featureResolver.ProcessTriggeredFeatures(actor, nil, state, models.TriggerEndTurn, false)
}

// drinkPotion spends the turn on an SRD Health or Stamina Potion: 1d4+1 cleared, no roll, spotlight kept
func (ce *CombatEngine) drinkPotion(actor *models.Combatant, potion models.PotionType, state *models.CombatState) {
	amount := ce.dice.RollDice(1, 4, 1)
	result := "No potion left"
	switch {
	case potion == models.PotionHealth && actor.HealthPotions > 0:
		actor.HealthPotions--
		before := actor.HP
		actor.Heal(amount)
		result = fmt.Sprintf("Drinks a Health Potion: clears %d HP", actor.HP-before)
	case potion == models.PotionStamina && actor.StaminaPotions > 0:
		actor.StaminaPotions--
		before := actor.Stress
		actor.ClearStress(amount)
		result = fmt.Sprintf("Drinks a Stamina Potion: clears %d Stress", before-actor.Stress)
	}
	state.RecordAction(models.ActionParams{Actor: actor, ActionType: models.ActionUseItem, Result: result})
}

func (ce *CombatEngine) recordActivation(actor, target *models.Combatant, before *models.CombatantState, name string, isFeature bool, err error, dcResult DCCheckResult, state *models.CombatState) {
	result := "Success"
	if dcResult.IsCritical {
		result = "Critical Success"
	}
	spotlightShift := dcResult.SpotlightShift
	if dcResult.RedirectedTo != nil {
		target, before = dcResult.RedirectedTo, nil
	}

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
		DamageParts:     dcResult.DamageParts,
		DamageFormula:   dcResult.DamageFormula,
		RollParts:       dcResult.RollParts,
		VsDefense:       dcResult.VsDefense,
		ProcessedDamage: dcResult.ProcessedDamage,
		TargetsHit:      dcResult.TargetsHit,
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
		TargetBefore:    before,
		Main:            true,
	})
	recordReactions(state, name, dcResult)
}

func (ce *CombatEngine) executeAbility(actor *models.Combatant, target *models.Combatant, ability *models.DomainAbility, state *models.CombatState) {
	before := models.StateOf(target)
	state.BeginAction()
	err, dcResult := ce.featureResolver.ActivateDomainAbility(ability, actor, target, state)
	ce.recordActivation(actor, target, before, ability.Name, false, err, dcResult, state)
}

func (ce *CombatEngine) executeFeature(actor *models.Combatant, target *models.Combatant, feature *models.AdversaryFeature, state *models.CombatState) {
	before := models.StateOf(target)
	state.BeginAction()
	err, dcResult := ce.featureResolver.ActivateAdversaryFeature(feature, actor, target, state)
	ce.recordActivation(actor, target, before, feature.Name, true, err, dcResult, state)
}

// executeClearCondition spends the actor's turn clearing one harmful condition.
// PCs need a successful action roll; adversaries only spend their spotlight.
func (ce *CombatEngine) executeClearCondition(actor *models.Combatant, state *models.CombatState) {
	cond := actor.ClearableCondition()
	if cond == nil {
		state.RecordAction(models.ActionParams{Actor: actor, ActionType: models.ActionClearCondition, Result: "Nothing to clear"})
		return
	}

	if actor.Side != models.SidePlayer {
		actor.RemoveCondition(cond.Type)
		state.RecordAction(models.ActionParams{Actor: actor, ActionType: models.ActionClearCondition, Result: fmt.Sprintf("Cleared %s", cond.Type)})
		return
	}

	difficulty := cond.ClearDifficulty
	if difficulty == 0 {
		difficulty = models.DefaultClearDifficulty
	}
	roll := ce.featureResolver.rollDCCheck(actor, nil, "fixed", difficulty, "", state)

	result := fmt.Sprintf("Failed to clear %s", cond.Type)
	if roll.Success {
		actor.RemoveCondition(cond.Type)
		result = fmt.Sprintf("Cleared %s", cond.Type)
	}
	state.RecordAction(models.ActionParams{
		Actor:          actor,
		ActionType:     models.ActionClearCondition,
		Result:         result,
		HopeDie:        roll.HopeDie,
		FearDie:        roll.FearDie,
		TotalRoll:      roll.TotalRoll,
		TargetDefense:  roll.DC,
		RollParts:      roll.RollParts,
		SpotlightShift: roll.SpotlightShift,
		HopeGenerated:  roll.HopeGenerated,
		FearGenerated:  roll.FearGenerated,
	})
}

// resolveDeathMoves makes a death move for every PC who just marked their last Hit Point
func (ce *CombatEngine) resolveDeathMoves(state *models.CombatState) {
	for _, pc := range state.Players {
		if pc.IsAlive() || pc.Fallen {
			continue
		}

		switch ce.deathMove {
		case strategy.DeathMoveRiskItAll:
			ce.riskItAll(pc, state)
		case strategy.DeathMoveBlazeOfGlory:
			ce.blazeOfGlory(pc, state)
		default:
			pc.Fallen = true
			state.RecordAction(models.ActionParams{Actor: pc, ActionType: models.ActionDeathMove, Result: "Avoid Death: unconscious"})
		}
	}
}

// riskItAll rolls the Duality Dice: Hope higher keeps the PC up, Fear higher kills them
func (ce *CombatEngine) riskItAll(pc *models.Combatant, state *models.CombatState) {
	roll := ce.dice.RollDuality(0, 0)

	result := "Risk It All: dies"
	switch {
	case roll.IsCritical:
		pc.IsDefeated = false
		pc.HP = pc.HPMax
		pc.ClearStress(pc.Stress)
		result = "Risk It All: clears all Hit Points and Stress"
	case roll.Result == models.ResultHope:
		// The Hope Die value is split between Hit Points and Stress, Hit Points first
		healed := min(roll.HopeDie, pc.HPMax)
		pc.IsDefeated = false
		pc.HP = healed
		pc.ClearStress(roll.HopeDie - healed)
		result = fmt.Sprintf("Risk It All: stays up, clears %d", roll.HopeDie)
	default:
		pc.Fallen = true
	}

	state.RecordAction(models.ActionParams{
		Actor:      pc,
		ActionType: models.ActionDeathMove,
		Result:     result,
		HopeDie:    roll.HopeDie,
		FearDie:    roll.FearDie,
	})
}

// blazeOfGlory gives the PC one final attack that automatically crits, then they die
func (ce *CombatEngine) blazeOfGlory(pc *models.Combatant, state *models.CombatState) {
	pc.Fallen = true

	var target *models.Combatant
	for _, e := range state.GetAliveEnemies() {
		if target == nil || e.HP < target.HP {
			target = e
		}
	}
	if target == nil {
		state.RecordAction(models.ActionParams{Actor: pc, ActionType: models.ActionDeathMove, Result: "Blaze of Glory"})
		return
	}

	// Both the attack roll and any triggered reactions use the forced-critical dice
	critDice := critRoller{ce.dice}
	critFeatures := *ce.featureResolver
	critFeatures.dice = critDice
	state.BeginAction()
	attack := NewCombatResolver(critDice, &critFeatures).ResolveAttack(pc, target, pc.AttackDamageFormula(), state)

	state.RecordAction(models.ActionParams{
		Actor:           pc,
		ActionType:      models.ActionDeathMove,
		Target:          target,
		Result:          "Blaze of Glory",
		DamageRoll:      attack.DamageRoll,
		ProcessedDamage: attack.ProcessedDamage,
		DamageDice:      attack.DamageDice,
		DamageBonus:     attack.DamageBonus,
		DamageParts:     attack.DamageParts,
		DamageFormula:   attack.DamageFormula,
		RollParts:       attack.Roll.RollParts,
		VsDefense:       true,
		HPDamage:        attack.HPDamage,
		Threshold:       attack.Threshold,
		HopeDie:         attack.Roll.HopeDie,
		FearDie:         attack.Roll.FearDie,
		TotalRoll:       attack.Roll.TotalRoll,
		DamageResult:    attack.DamageResult,
		Main:            true,
	})
}

// executeAttack handles basic attack action
func (ce *CombatEngine) executeAttack(actor *models.Combatant, target *models.Combatant, state *models.CombatState) {
	before := models.StateOf(target)
	ce.featureResolver.ProcessTriggeredFeatures(actor, target, state, models.TriggerOnAttack, false)

	// What the attack sets off (reactions, on-hit passives, a die running out) is logged after its own row
	state.BeginAction()
	result := ce.resolver.ResolveAttack(actor, target, actor.AttackDamageFormula(), state)

	if result.Roll.IsCritical {
		ce.featureResolver.ProcessTriggeredFeatures(actor, target, state, models.TriggerOnCritical, true)
	}

	resultStr := "Miss"
	if result.Hit {
		resultStr = "Hit"
		if result.Roll.IsCritical {
			resultStr = "Critical Hit"
		}
		ce.featureResolver.ProcessTriggeredFeatures(actor, target, state, models.TriggerOnHit, false)
		ce.featureResolver.ProcessTriggeredFeatures(target, actor, state, models.TriggerOnHitTaken, false)
	} else {
		ce.featureResolver.ProcessTriggeredFeatures(actor, target, state, models.TriggerOnMiss, false)
		ce.featureResolver.ProcessTriggeredFeatures(target, actor, state, models.TriggerOnEvaded, false)
	}
	hitBy := target
	if result.Redirected != nil {
		hitBy, before = result.Redirected, nil
	}

	state.RecordAction(models.ActionParams{
		Actor:           actor,
		ActionType:      models.ActionAttack,
		Target:          hitBy,
		Result:          resultStr,
		DamageRoll:      result.DamageRoll,
		ProcessedDamage: result.ProcessedDamage,
		DamageDice:      result.DamageDice,
		DamageBonus:     result.DamageBonus,
		DamageParts:     result.DamageParts,
		DamageFormula:   result.DamageFormula,
		RollParts:       result.Roll.RollParts,
		VsDefense:       true,
		HPDamage:        result.HPDamage,
		Threshold:       result.Threshold,
		HopeDie:         result.Roll.HopeDie,
		FearDie:         result.Roll.FearDie,
		D20Roll:         result.Roll.D20Roll,
		TotalRoll:       result.Roll.TotalRoll,
		TargetDefense:   target.GetDefense(),
		SpotlightShift:  !result.Hit || result.Roll.SpotlightShift,
		HopeGenerated:   result.Roll.HopeGenerated,
		FearGenerated:   result.Roll.FearGenerated,
		DamageResult:    result.DamageResult,
		TargetBefore:    before,
		Main:            true,
	})
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

	decision := dm.ChooseAction(state, enemy)
	if decision.Target == nil {
		return false
	}

	ce.ExecuteTurnWithDecision(enemy, decision, state)
	state.RecordAdversarySpotlight(enemy.ID)
	state.TurnNumber++
	return true
}

// drainPendingSpotlights lets every ally handed a spotlight by a feature act now, at no
// Fear cost. Their own features may queue more; the GM-turn spotlight limit bounds the chain.
func (ce *CombatEngine) drainPendingSpotlights(state *models.CombatState, dm DecisionMaker, turnCount *int, maxTurns int) {
	for !state.IsCombatOver() && *turnCount < maxTurns {
		next, ok := state.PopPendingSpotlight()
		if !ok {
			return
		}
		ally := next.Combatant
		if !ally.IsAlive() {
			continue
		}
		ally.HalfDamage = next.HalfDamage
		ce.executeAdversarySpotlight(ally, state, dm)
		ally.HalfDamage = false
		*turnCount++
	}
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
			// Tokens measure waiting: everyone passed over gains one, the actor's clear after the turn
			for _, waiting := range state.GetAlivePlayers() {
				if waiting != player {
					waiting.SpotlightTokens++
				}
			}

			decision := decisionMaker.ChooseAction(state, player)
			if decision.Target == nil {
				break
			}

			actionsBeforeTurn := len(state.ActionHistory)
			ce.ExecuteTurnWithDecision(player, decision, state)
			player.SpotlightTokens = 0
			state.TurnNumber++
			turnCount++
			state.ConsecutivePlayerTurns++

			if slices.ContainsFunc(state.ActionHistory[actionsBeforeTurn:], func(a models.ActionRecord) bool { return a.SpotlightShift }) {
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

		// Spotlights granted by reactions during the PC phase resolve first
		ce.drainPendingSpotlights(state, decisionMaker, &turnCount, maxTurns)

		firstEnemy := ce.chooseGMAdversary(state)
		if firstEnemy != nil {
			ce.executeAdversarySpotlight(firstEnemy, state, decisionMaker)
			turnCount++
			ce.drainPendingSpotlights(state, decisionMaker, &turnCount, maxTurns)
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
			ce.drainPendingSpotlights(state, decisionMaker, &turnCount, maxTurns)
		}

		state.EndGMTurn()
		state.RoundNumber++
		roundCount++
	}

	outcome := &CombatOutcome{
		PlayerVictory: state.IsPlayerVictory(),
		EnemyVictory:  state.IsEnemyVictory(),
		Turns:         turnCount,
		Rounds:        roundCount,
		FinalHopePool: state.TotalHope(),
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
