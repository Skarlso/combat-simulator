package strategy

import (
	"math/rand"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// ActionDecision represents the AI's choice for a turn
type ActionDecision struct {
	Target           *models.Combatant
	DomainAbility    *models.DomainAbility    // nil = use basic attack
	AdversaryFeature *models.AdversaryFeature // nil = use basic attack
}

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

// ChooseAction selects an action (feature or basic attack) and target
// Returns ActionDecision with target and optional feature choice
func (ai *AIDecisionMaker) ChooseAction(state *models.CombatState, actor *models.Combatant) *ActionDecision {
	// Select target based on strategy
	target := ai.selectTarget(state, actor)
	if target == nil {
		return &ActionDecision{Target: nil}
	}

	// Evaluate features vs basic attack
	decision := ai.evaluateBestAction(state, actor, target)
	return decision
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

	// Higher HP = more threat potential
	hpPercent := (combatant.HP * 100) / combatant.HPMax
	threat += hpPercent / 10

	// Add threat based on features/abilities count
	threat += len(combatant.AdversaryFeatures)
	threat += len(combatant.DomainAbilities)

	return threat
}

// evaluateBestAction compares features vs basic attack and returns the best choice
func (ai *AIDecisionMaker) evaluateBestAction(state *models.CombatState, actor *models.Combatant, target *models.Combatant) *ActionDecision {
	bestValue := ai.evaluateBasicAttack(actor, target)
	decision := &ActionDecision{Target: target} // Default: basic attack

	// Evaluate player domain abilities
	if actor.Side == models.SidePlayer {
		for i := range actor.DomainAbilities {
			ability := &actor.DomainAbilities[i]
			if !ai.canAffordAbility(state, actor, ability) {
				continue
			}

			value := ai.evaluateAbility(state, actor, target, ability)
			if value > bestValue {
				bestValue = value
				decision.DomainAbility = ability
				decision.AdversaryFeature = nil
			}
		}
	}

	// Evaluate enemy adversary features
	if actor.Side == models.SideEnemy {
		for i := range actor.AdversaryFeatures {
			feature := &actor.AdversaryFeatures[i]
			if !ai.canAffordFeature(state, feature) {
				continue
			}

			value := ai.evaluateFeature(state, actor, target, feature)
			if value > bestValue {
				bestValue = value
				decision.AdversaryFeature = feature
				decision.DomainAbility = nil
			}
		}
	}

	return decision
}

// evaluateBasicAttack estimates expected value of a basic attack
func (ai *AIDecisionMaker) evaluateBasicAttack(actor *models.Combatant, target *models.Combatant) float64 {
	// Estimate average damage from actor's damage dice
	avgDamage := ai.estimateAverageDamage(actor.DamageDice)

	// Factor in hit chance (rough estimate based on proficiency vs defense)
	hitChance := 0.6 // Base 60% hit chance
	if actor.Proficiency > target.GetDefense() {
		hitChance = 0.75
	} else if actor.Proficiency < target.GetDefense()-3 {
		hitChance = 0.45
	}

	// Expected value = avgDamage * hitChance
	return avgDamage * hitChance
}

// estimateSuccessProbability estimates the chance of success against a DC
func (ai *AIDecisionMaker) estimateSuccessProbability(actor *models.Combatant, target *models.Combatant, dcType string, fixedDC int) float64 {
	if dcType == "" || dcType == "none" {
		return 1.0 // Auto-succeed
	}

	var dc int
	switch dcType {
	case "target":
		if target == nil {
			return 0.5 // Unknown target, assume 50%
		}
		dc = target.GetDefense()
	case "fixed":
		dc = fixedDC
	default:
		return 1.0 // Unknown DC type, assume success
	}

	// Estimate success probability based on actor's stats
	if actor.Side == models.SidePlayer {
		// Player uses duality dice (2d12 + modifiers)
		// Average duality roll is about 13 + proficiency
		avgRoll := 13 + actor.Proficiency

		// Rough probability estimate based on difference
		diff := avgRoll - dc
		if diff >= 5 {
			return 0.85 // Very likely to succeed
		} else if diff >= 0 {
			return 0.65 // Moderately likely
		} else if diff >= -3 {
			return 0.45 // Somewhat unlikely
		} else {
			return 0.25 // Very unlikely
		}
	} else {
		// Enemy uses d20 + ATK
		// Average roll is 10.5 + proficiency
		avgRoll := 10 + actor.Proficiency

		diff := avgRoll - dc
		if diff >= 5 {
			return 0.80
		} else if diff >= 0 {
			return 0.60
		} else if diff >= -3 {
			return 0.40
		} else {
			return 0.20
		}
	}
}

// evaluateAbility estimates expected value of a domain ability
func (ai *AIDecisionMaker) evaluateAbility(state *models.CombatState, actor *models.Combatant, target *models.Combatant, ability *models.DomainAbility) float64 {
	value := 0.0

	for _, effect := range ability.Effects {
		value += ai.evaluateEffect(effect, actor, target, state)
	}

	// Factor in success probability based on DC
	successProb := ai.estimateSuccessProbability(actor, target, ability.DCType, ability.FixedDC)
	value *= successProb

	// Subtract cost penalty (Hope is valuable) - costs paid regardless of success
	value -= float64(ability.HopeCost) * 1.5
	value -= float64(ability.StressCost) * 0.5

	return value
}

// evaluateFeature estimates expected value of an adversary feature
func (ai *AIDecisionMaker) evaluateFeature(state *models.CombatState, actor *models.Combatant, target *models.Combatant, feature *models.AdversaryFeature) float64 {
	value := 0.0

	for _, effect := range feature.Effects {
		value += ai.evaluateEffect(effect, actor, target, state)
	}

	// Factor in success probability based on DC
	successProb := ai.estimateSuccessProbability(actor, target, feature.DCType, feature.FixedDC)
	value *= successProb

	// Subtract fear cost penalty - costs paid regardless of success
	value -= float64(feature.FearCost) * 1.0

	return value
}

// evaluateEffect estimates the value of a single effect
func (ai *AIDecisionMaker) evaluateEffect(effect models.FeatureEffect, actor *models.Combatant, target *models.Combatant, state *models.CombatState) float64 {
	switch effect.Category {
	case models.FeatureEffectDamage, models.FeatureEffectDirectDamage:
		// Direct damage value
		avgDamage := ai.estimateAverageDamage(effect.DiceFormula)
		return avgDamage + float64(effect.Magnitude)

	case models.FeatureEffectAreaDamage:
		// AOE hits multiple targets
		avgDamage := ai.estimateAverageDamage(effect.DiceFormula)
		enemyCount := float64(len(state.GetEnemiesOf(actor)))
		if enemyCount > 3 {
			enemyCount = 3 // Cap at 3 targets
		}
		return (avgDamage + float64(effect.Magnitude)) * enemyCount

	case models.FeatureEffectApplyPoison, models.FeatureEffectApplyIgnite:
		// Status effects are valuable on high HP targets
		if target.HP > target.HPMax/2 {
			return 3.0 // Worth about 3 damage over time
		}
		return 1.5

	case models.FeatureEffectApplyVulnerable:
		// Very valuable on high HP targets
		if target.HP > target.HPMax/2 {
			return 5.0
		}
		return 2.0

	case models.FeatureEffectStressInflict:
		// Stress damage
		return float64(effect.Magnitude) * 0.3 // Less valuable than HP damage

	case models.FeatureEffectHPHeal:
		// Healing is valuable when hurt
		missingHP := actor.HPMax - actor.HP
		healAmount := float64(effect.Magnitude) + ai.estimateAverageDamage(effect.DiceFormula)
		if missingHP > int(healAmount) {
			return healAmount
		}
		return float64(missingHP) * 0.8 // Overheal is wasted

	case models.FeatureEffectStressClear:
		// Clearing stress
		if actor.Stress > 0 {
			return float64(effect.Magnitude) * 0.5
		}
		return 0.0

	case models.FeatureEffectArmorDamage:
		// Armor damage is valuable if target has armor
		if target.ArmorSlots > 0 {
			return 2.0
		}
		return 0.0

	case models.FeatureEffectGrantArmor:
		// Grant armor is valuable when actor is low on armor
		if actor.ArmorSlots < 2 {
			return float64(effect.Magnitude) * 2.0
		}
		return float64(effect.Magnitude) * 1.0

	case models.FeatureEffectDamageReduction:
		// Damage reduction is valuable (defensive ability)
		avgReduction := ai.estimateAverageDamage(effect.DiceFormula)
		return avgReduction + float64(effect.Magnitude)

	case models.FeatureEffectTokenPlace, models.FeatureEffectTokenSpend:
		// Token mechanics - moderate value (future use)
		return 2.0

	default:
		// Unknown effect, give it minimal value
		return 1.0
	}
}

// estimateAverageDamage calculates average damage from dice notation (e.g., "2d6+3")
func (ai *AIDecisionMaker) estimateAverageDamage(diceFormula string) float64 {
	if diceFormula == "" {
		return 0.0
	}

	// Parse dice notation
	parsed, err := models.ParseDiceNotation(diceFormula)
	if err != nil {
		return 0.0
	}

	// Sum up average for each term
	total := 0.0
	for _, term := range parsed.Terms {
		if term.Count == 0 {
			// Flat bonus
			total += float64(term.Size)
		} else {
			// Dice roll: count * ((size + 1) / 2)
			avgPerDie := float64(term.Size+1) / 2.0
			total += float64(term.Count) * avgPerDie
		}
	}

	return total
}

// canAffordAbility checks if actor can pay costs for ability
func (ai *AIDecisionMaker) canAffordAbility(state *models.CombatState, actor *models.Combatant, ability *models.DomainAbility) bool {
	if ability.HopeCost > state.HopePool {
		return false
	}
	if ability.StressCost+actor.Stress > actor.StressMax {
		return false
	}
	return true
}

// canAffordFeature checks if state has enough Fear for feature
func (ai *AIDecisionMaker) canAffordFeature(state *models.CombatState, feature *models.AdversaryFeature) bool {
	return feature.FearCost <= state.FearPool
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
