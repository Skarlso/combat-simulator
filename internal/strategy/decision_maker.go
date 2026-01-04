package strategy

import (
	"math/rand"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// AIDecisionMaker implements strategic decision-making for combatants
type AIDecisionMaker struct {
	rng *rand.Rand

	// Strategy configurations
	ResourceUsage     ResourceUsageStrategy
	ConditionClearing ConditionClearStrategy
	TargetSelection   TargetSelectionStrategy
	AbilityUsage      AbilityUsageStrategy

	// Thresholds
	LowHPThreshold      int // Percentage (e.g., 30 = below 30% HP)
	HighThreatThreshold int // Damage threshold to consider enemy "high threat"
}

// NewAIDecisionMaker creates a decision maker with given strategies
func NewAIDecisionMaker(rng *rand.Rand, config StrategyConfig) *AIDecisionMaker {
	return &AIDecisionMaker{
		rng:                 rng,
		ResourceUsage:       config.ResourceUsage,
		ConditionClearing:   config.ConditionClearing,
		TargetSelection:     config.TargetSelection,
		AbilityUsage:        config.AbilityUsage,
		LowHPThreshold:      config.LowHPThreshold,
		HighThreatThreshold: config.HighThreatThreshold,
	}
}

// StrategyConfig bundles strategy settings
type StrategyConfig struct {
	ResourceUsage       ResourceUsageStrategy
	ConditionClearing   ConditionClearStrategy
	TargetSelection     TargetSelectionStrategy
	AbilityUsage        AbilityUsageStrategy
	LowHPThreshold      int
	HighThreatThreshold int
}

// DefaultGreedyConfig returns aggressive strategy
func DefaultGreedyConfig() StrategyConfig {
	return StrategyConfig{
		ResourceUsage:       ResourceGreedy,
		ConditionClearing:   ClearNever,
		TargetSelection:     TargetLowestHP,
		AbilityUsage:        AbilityAlways,
		LowHPThreshold:      30,
		HighThreatThreshold: 10,
	}
}

// DefaultConservativeConfig returns defensive strategy
func DefaultConservativeConfig() StrategyConfig {
	return StrategyConfig{
		ResourceUsage:       ResourceConservative,
		ConditionClearing:   ClearImmediately,
		TargetSelection:     TargetHighestThreat,
		AbilityUsage:        AbilitySparingly,
		LowHPThreshold:      50,
		HighThreatThreshold: 8,
	}
}

// DefaultDesperateConfig returns context-aware strategy
func DefaultDesperateConfig() StrategyConfig {
	return StrategyConfig{
		ResourceUsage:       ResourceDesperate,
		ConditionClearing:   ClearProbabilistic,
		TargetSelection:     TargetLowestHP,
		AbilityUsage:        AbilityContextual,
		LowHPThreshold:      40,
		HighThreatThreshold: 12,
	}
}

// DefaultRandomConfig returns unpredictable strategy
func DefaultRandomConfig() StrategyConfig {
	return StrategyConfig{
		ResourceUsage:       ResourceRandom,
		ConditionClearing:   ClearProbabilistic,
		TargetSelection:     TargetRandom,
		AbilityUsage:        AbilityContextual,
		LowHPThreshold:      35,
		HighThreatThreshold: 10,
	}
}

// ChooseAction selects the best action for the actor
func (ai *AIDecisionMaker) ChooseAction(state *models.CombatState, actor *models.Combatant) *models.Action {
	// Check if should clear condition first
	if ai.shouldClearCondition(actor) {
		return models.NewClearConditionAction()
	}

	// Check if should use an ability
	if ability, targets := ai.shouldUseAbility(state, actor); ability != nil {
		return models.NewAbilityAction(ability, targets)
	}

	// Default to basic attack
	target := ai.selectTarget(state, actor)
	if target == nil {
		return models.NewPassAction()
	}

	return models.NewAttackAction(target)
}

// shouldClearCondition determines if actor should spend turn clearing a condition
func (ai *AIDecisionMaker) shouldClearCondition(actor *models.Combatant) bool {
	if len(actor.Conditions) == 0 {
		return false
	}

	switch ai.ConditionClearing {
	case ClearImmediately:
		// Clear any harmful condition
		return ai.hasHarmfulCondition(actor)

	case ClearWhenSafe:
		// Only clear if HP is high enough
		hpPercent := (actor.HP * 100) / actor.HPMax
		return hpPercent > 60 && ai.hasHarmfulCondition(actor)

	case ClearNever:
		return false

	case ClearProbabilistic:
		// Probability increases as HP decreases
		hpPercent := (actor.HP * 100) / actor.HPMax
		clearChance := 1.0 - (float64(hpPercent) / 100.0)

		// Higher chance if has harmful condition
		if ai.hasHarmfulCondition(actor) {
			clearChance += 0.3
		}

		return ai.rng.Float64() < clearChance

	default:
		return false
	}
}

// hasHarmfulCondition checks if actor has conditions worth clearing
func (ai *AIDecisionMaker) hasHarmfulCondition(actor *models.Combatant) bool {
	harmfulConditions := []models.ConditionType{
		models.ConditionStunned,
		models.ConditionBurning,
		models.ConditionPoisoned,
		models.ConditionDisadvantage,
		models.ConditionRestrained,
	}

	for _, harmfulType := range harmfulConditions {
		if actor.HasCondition(harmfulType) {
			return true
		}
	}

	return false
}

// shouldUseAbility determines if actor should use a special ability
func (ai *AIDecisionMaker) shouldUseAbility(state *models.CombatState, actor *models.Combatant) (*models.Ability, []*models.Combatant) {
	if len(actor.Abilities) == 0 {
		return nil, nil
	}

	switch ai.AbilityUsage {
	case AbilityNever:
		return nil, nil

	case AbilityAlways:
		// Use first available ability
		for _, ability := range actor.Abilities {
			if targets := ai.canUseAbility(state, actor, ability); len(targets) > 0 {
				return ability, targets
			}
		}

	case AbilitySparingly:
		// Only use if low HP or high resource pool
		hpPercent := (actor.HP * 100) / actor.HPMax
		var pool int
		if actor.Side == models.SidePlayer {
			pool = state.HopePool
		} else {
			pool = state.FearPool
		}

		if hpPercent < ai.LowHPThreshold || pool > 5 {
			for _, ability := range actor.Abilities {
				if targets := ai.canUseAbility(state, actor, ability); len(targets) > 0 {
					return ability, targets
				}
			}
		}

	case AbilityContextual:
		// Use damage abilities if enemies low HP, heal/buff if allies low HP
		enemies := state.GetEnemiesOf(actor)
		allies := state.GetAlliesOf(actor)

		// Check for low HP enemies (good time for damage abilities)
		hasLowHPEnemy := false
		for _, enemy := range enemies {
			enemyHPPercent := (enemy.HP * 100) / enemy.HPMax
			if enemyHPPercent < 40 {
				hasLowHPEnemy = true
				break
			}
		}

		// Check for low HP allies (good time for healing/support)
		hasLowHPAlly := false
		for _, ally := range allies {
			if ally == actor {
				continue
			}
			allyHPPercent := (ally.HP * 100) / ally.HPMax
			if allyHPPercent < 50 {
				hasLowHPAlly = true
				break
			}
		}

		for _, ability := range actor.Abilities {
			isDamageAbility := ai.isDamageAbility(ability)
			isHealAbility := ai.isHealAbility(ability)

			// Use damage ability if enemy is low
			if isDamageAbility && hasLowHPEnemy {
				if targets := ai.canUseAbility(state, actor, ability); len(targets) > 0 {
					return ability, targets
				}
			}

			// Use heal ability if ally is low
			if isHealAbility && hasLowHPAlly {
				if targets := ai.canUseAbility(state, actor, ability); len(targets) > 0 {
					return ability, targets
				}
			}
		}
	}

	return nil, nil
}

// canUseAbility checks if ability is usable and returns valid targets
func (ai *AIDecisionMaker) canUseAbility(state *models.CombatState, actor *models.Combatant, ability *models.Ability) []*models.Combatant {
	// Check resource costs
	if !ai.shouldSpendResources(state, actor, ability) {
		return nil
	}

	// Get potential targets
	targets := ai.getAbilityTargets(state, actor, ability)
	if len(targets) == 0 {
		return nil
	}

	return targets
}

// shouldSpendResources determines if actor should spend resources on ability
func (ai *AIDecisionMaker) shouldSpendResources(state *models.CombatState, actor *models.Combatant, ability *models.Ability) bool {
	costs := ability.Costs

	// Always allow abilities with no cost
	if costs.IsEmpty() {
		return true
	}

	// Don't spend if would die
	if costs.HP > 0 && actor.HP <= costs.HP {
		return false
	}

	// Check stress cost
	if costs.Stress > 0 && (actor.Stress+costs.Stress) > actor.StressMax {
		return false
	}

	var pool int
	var cost int
	if actor.Side == models.SidePlayer {
		pool = state.HopePool
		cost = costs.Hope
	} else {
		pool = state.FearPool
		cost = costs.Fear
	}

	if cost > pool {
		return false
	}

	// Strategy-based resource spending
	switch ai.ResourceUsage {
	case ResourceGreedy:
		return true // Always spend if available

	case ResourceConservative:
		// Only spend if pool is large
		return pool >= cost+3

	case ResourceDesperate:
		// Spend if low HP
		hpPercent := (actor.HP * 100) / actor.HPMax
		return hpPercent < ai.LowHPThreshold

	case ResourceRandom:
		return ai.rng.Float64() < 0.5

	case ResourceClever:
		// Spend if enemies are low or allies are in danger
		enemies := state.GetEnemiesOf(actor)
		totalEnemyHP := 0
		for _, e := range enemies {
			totalEnemyHP += e.HP
		}
		avgEnemyHP := totalEnemyHP / max(len(enemies), 1)

		// Spend if average enemy HP is low
		return avgEnemyHP < 3

	default:
		return true
	}
}

// getAbilityTargets selects targets for an ability
func (ai *AIDecisionMaker) getAbilityTargets(state *models.CombatState, actor *models.Combatant, ability *models.Ability) []*models.Combatant {
	if ability.TargetSelf {
		return []*models.Combatant{actor}
	}

	var candidates []*models.Combatant
	if ability.TargetEnemies {
		candidates = state.GetEnemiesOf(actor)
	} else if ability.TargetAllies {
		candidates = state.GetAlliesOf(actor)
	}

	if len(candidates) == 0 {
		return nil
	}

	if ability.TargetAOE {
		// For AoE, select random subset up to max targets
		maxTargets := ability.AOEMaxTargets
		if maxTargets == 0 || maxTargets > len(candidates) {
			maxTargets = len(candidates)
		}

		// Random number of targets (1 to maxTargets)
		numTargets := ai.rng.Intn(maxTargets) + 1

		// Shuffle and take first N
		shuffled := make([]*models.Combatant, len(candidates))
		copy(shuffled, candidates)
		ai.rng.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		return shuffled[:numTargets]
	}

	// Single target selection
	target := ai.selectTargetFrom(candidates)
	if target == nil {
		return nil
	}

	return []*models.Combatant{target}
}

// selectTarget chooses a target from enemies
func (ai *AIDecisionMaker) selectTarget(state *models.CombatState, actor *models.Combatant) *models.Combatant {
	enemies := state.GetEnemiesOf(actor)
	return ai.selectTargetFrom(enemies)
}

// selectTargetFrom chooses from a list of candidates
func (ai *AIDecisionMaker) selectTargetFrom(candidates []*models.Combatant) *models.Combatant {
	if len(candidates) == 0 {
		return nil
	}

	switch ai.TargetSelection {
	case TargetLowestHP:
		lowest := candidates[0]
		for _, c := range candidates {
			if c.HP < lowest.HP {
				lowest = c
			}
		}
		return lowest

	case TargetHighestHP:
		highest := candidates[0]
		for _, c := range candidates {
			if c.HP > highest.HP {
				highest = c
			}
		}
		return highest

	case TargetRandom:
		return candidates[ai.rng.Intn(len(candidates))]

	case TargetHighestThreat:
		// Threat = proficiency + number of abilities
		highestThreat := candidates[0]
		maxThreat := ai.calculateThreat(highestThreat)

		for _, c := range candidates {
			threat := ai.calculateThreat(c)
			if threat > maxThreat {
				highestThreat = c
				maxThreat = threat
			}
		}
		return highestThreat

	default:
		return candidates[0]
	}
}

// calculateThreat estimates how dangerous a combatant is
func (ai *AIDecisionMaker) calculateThreat(combatant *models.Combatant) int {
	threat := combatant.Proficiency * 2
	threat += len(combatant.Abilities)

	// Higher HP = more threat potential
	hpPercent := (combatant.HP * 100) / combatant.HPMax
	threat += hpPercent / 10

	return threat
}

// isDamageAbility checks if ability primarily deals damage
func (ai *AIDecisionMaker) isDamageAbility(ability *models.Ability) bool {
	for _, effect := range ability.Effects {
		if effect.GetType() == models.EffectDamage {
			return true
		}
	}
	return false
}

// isHealAbility checks if ability heals
func (ai *AIDecisionMaker) isHealAbility(ability *models.Ability) bool {
	for _, effect := range ability.Effects {
		if effect.GetType() == models.EffectHeal {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
