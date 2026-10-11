package strategy

import (
	"math/rand"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// ActionDecision represents the AI's choice for a turn
type ActionDecision struct {
	Target           *models.Combatant
	ClearCondition   bool                     // spend the turn clearing a harmful condition on Target
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
	// Most targets an area effect is assumed to reach (0 = everyone)
	AreaTargets int

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
	DeathMove           DeathMoveStrategy
	LowHPThreshold      int
	HighThreatThreshold int
}

// DefaultGreedyConfig returns aggressive strategy
func DefaultGreedyConfig() StrategyConfig {
	return StrategyConfig{
		ResourceUsage:       ResourceGreedy,
		ConditionClearing:   ClearImmediately,
		TargetSelection:     TargetLowestHP,
		AbilityUsage:        AbilityAlways,
		DeathMove:           DeathMoveBlazeOfGlory,
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
		DeathMove:           DeathMoveAvoidDeath,
		LowHPThreshold:      50,
		HighThreatThreshold: 8,
	}
}

// DefaultDesperateConfig returns context-aware strategy
func DefaultDesperateConfig() StrategyConfig {
	return StrategyConfig{
		ResourceUsage:       ResourceDesperate,
		ConditionClearing:   ClearImmediately,
		TargetSelection:     TargetLowestHP,
		AbilityUsage:        AbilityContextual,
		DeathMove:           DeathMoveRiskItAll,
		LowHPThreshold:      40,
		HighThreatThreshold: 12,
	}
}

// DefaultRandomConfig returns unpredictable strategy
func DefaultRandomConfig() StrategyConfig {
	return StrategyConfig{
		ResourceUsage:       ResourceRandom,
		ConditionClearing:   ClearImmediately,
		TargetSelection:     TargetRandom,
		AbilityUsage:        AbilityContextual,
		DeathMove:           DeathMoveRiskItAll,
		LowHPThreshold:      35,
		HighThreatThreshold: 10,
	}
}

func (ai *AIDecisionMaker) ChooseAction(state *models.CombatState, actor *models.Combatant) *ActionDecision {
	if ai.shouldClearCondition(actor) {
		return &ActionDecision{Target: actor, ClearCondition: true}
	}

	target := ai.selectTarget(state, actor)
	if target == nil {
		return &ActionDecision{Target: nil}
	}

	decision := ai.evaluateBestAction(state, actor, target)
	return decision
}

// Every preset clears immediately for now: without ranges there is no way to
// judge when a condition like Restrained is harmless enough to ignore.
func (ai *AIDecisionMaker) shouldClearCondition(actor *models.Combatant) bool {
	if actor.ClearableCondition() == nil {
		return false
	}

	switch ai.ConditionClearing {
	case ClearImmediately:
		return true
	case ClearWhenSafe:
		hpPercent := (actor.HP * 100) / actor.HPMax
		return hpPercent > 50
	case ClearProbabilistic:
		hpPercent := (actor.HP * 100) / actor.HPMax
		threshold := 60
		if hpPercent < 30 {
			threshold = 80
		}
		return ai.rng.Intn(100) < threshold
	case ClearNever:
		return false
	default:
		return false
	}
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

func (ai *AIDecisionMaker) evaluateBestAction(state *models.CombatState, actor *models.Combatant, target *models.Combatant) *ActionDecision {
	if !ai.shouldUseAbilities(state, actor) {
		return &ActionDecision{Target: target}
	}

	bestValue := ai.evaluateBasicAttack(actor, target)
	decision := &ActionDecision{Target: target}

	if actor.Side == models.SidePlayer {
		for i := range actor.DomainAbilities {
			ability := &actor.DomainAbilities[i]
			// Passives fire on their trigger; they are not actions to choose
			if ability.Passive {
				continue
			}
			if !ai.canAffordAbility(state, actor, ability) {
				continue
			}
			if !ai.shouldSpendResources(state, actor, ability.HopeCost, ability.StressCost) {
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

	if actor.Side == models.SideEnemy {
		for i := range actor.AdversaryFeatures {
			feature := &actor.AdversaryFeatures[i]
			if feature.Passive {
				continue
			}
			if !ai.canAffordFeature(state, feature) {
				continue
			}
			if !ai.shouldSpendResources(state, actor, feature.FearCost, feature.StressCost) {
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

func (ai *AIDecisionMaker) shouldUseAbilities(state *models.CombatState, actor *models.Combatant) bool {
	switch ai.AbilityUsage {
	case AbilityNever:
		return false
	case AbilityAlways:
		return true
	case AbilitySparingly:
		hpPercent := (actor.HP * 100) / actor.HPMax
		return hpPercent < ai.LowHPThreshold || len(state.GetEnemiesOf(actor)) <= 1
	case AbilityContextual:
		hpPercent := (actor.HP * 100) / actor.HPMax
		if hpPercent < ai.LowHPThreshold {
			return true
		}
		return ai.rng.Intn(100) < 60
	default:
		return true
	}
}

func (ai *AIDecisionMaker) shouldSpendResources(state *models.CombatState, actor *models.Combatant, hopeFearCost, stressCost int) bool {
	if hopeFearCost == 0 && stressCost == 0 {
		return true
	}

	switch ai.ResourceUsage {
	case ResourceGreedy:
		return true
	case ResourceConservative:
		if actor.Side == models.SidePlayer {
			return actor.Hope > hopeFearCost+2
		}
		return state.FearPool > hopeFearCost+2
	case ResourceDesperate:
		hpPercent := (actor.HP * 100) / actor.HPMax
		return hpPercent < ai.LowHPThreshold
	case ResourceRandom:
		return ai.rng.Intn(100) < 50
	case ResourceClever:
		enemiesLeft := len(state.GetEnemiesOf(actor))
		if enemiesLeft <= 1 {
			return true
		}
		hpPercent := (actor.HP * 100) / actor.HPMax
		return hpPercent < 50 || ai.rng.Intn(100) < 40
	default:
		return true
	}
}

// evaluateBasicAttack estimates expected value of a basic attack
func (ai *AIDecisionMaker) evaluateBasicAttack(actor *models.Combatant, target *models.Combatant) float64 {
	avgDamage := ai.estimateAverageDamage(actor.DamageDice)
	hitChance := ai.estimateSuccessProbability(actor, target, "target", 0)
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
		// Average duality roll is about 13 + trait
		avgRoll := 13 + actor.GetAttackBonus()

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
		avgRoll := 10 + actor.GetAttackBonus()

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

	successProb := ai.estimateSuccessProbability(actor, target, feature.DCType, feature.FixedDC)
	value *= successProb

	value -= float64(feature.FearCost) * 0.5
	value -= float64(feature.StressCost) * 0.5

	// Features are the adversary's designed actions; prefer over generic attacks.
	// In Daggerheart's threshold system, raw dice differences matter less than
	// tactical effects, so give features an inherent preference.
	value += 2.0

	return value
}

// evaluateEffect estimates the value of a single effect
func (ai *AIDecisionMaker) evaluateEffect(effect models.FeatureEffect, actor *models.Combatant, target *models.Combatant, state *models.CombatState) float64 {
	value := ai.evaluateUnopposedEffect(effect, actor, target, state)

	// Rough discount for effects the target can react against
	switch {
	case effect.ReactionTrait == "":
		return value
	case effect.HalfOnSuccess:
		return value * 0.75
	default:
		return value * 0.5
	}
}

func (ai *AIDecisionMaker) evaluateUnopposedEffect(effect models.FeatureEffect, actor *models.Combatant, target *models.Combatant, state *models.CombatState) float64 {
	if effect.StandardAttack {
		effect.DiceFormula = actor.DamageDice
	}

	switch effect.Category {
	case models.FeatureEffectDamage, models.FeatureEffectDirectDamage:
		// Direct damage value
		avgDamage := ai.estimateAverageDamage(effect.DiceFormula)
		return avgDamage + float64(effect.Magnitude)

	case models.FeatureEffectAreaDamage:
		// AOE hits multiple targets
		avgDamage := ai.estimateAverageDamage(effect.DiceFormula)
		enemyCount := float64(len(state.GetEnemiesOf(actor)))
		if ai.AreaTargets > 0 && enemyCount > float64(ai.AreaTargets) {
			enemyCount = float64(ai.AreaTargets)
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

	case models.FeatureEffectApplyStunned:
		return 4.0

	case models.FeatureEffectApplyRestrained:
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
		// With no slot left to mark, the target marks a Hit Point instead
		return 3.0

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

	case models.FeatureEffectSummonAllies:
		// Summoning is VERY valuable - adds combatants to your side
		// Estimate based on summon quantity
		quantity := float64(effect.Magnitude)
		if effect.DiceFormula != "" {
			// Rough estimate: assume average of dice
			quantity += 2.5 // ~average of 1d4
		}
		// Each summoned ally is worth ~10 HP worth of value
		return quantity * 10.0

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
	if ability.HopeCost > actor.Hope {
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

type UnifiedDecisionMaker struct {
	playerStrategy *AIDecisionMaker
	enemyStrategy  *AIDecisionMaker
}

func NewUnifiedDecisionMaker(rng *rand.Rand, playerConfig, enemyConfig StrategyConfig) *UnifiedDecisionMaker {
	return &UnifiedDecisionMaker{
		playerStrategy: NewAIDecisionMaker(rng, playerConfig),
		enemyStrategy:  NewAIDecisionMaker(rng, enemyConfig),
	}
}

// SetAreaTargets tells both sides how many targets an area effect is assumed to reach
func (udm *UnifiedDecisionMaker) SetAreaTargets(limit int) {
	udm.playerStrategy.AreaTargets = limit
	udm.enemyStrategy.AreaTargets = limit
}

func (udm *UnifiedDecisionMaker) ChooseAction(state *models.CombatState, actor *models.Combatant) *ActionDecision {
	if actor.Side == models.SidePlayer {
		return udm.playerStrategy.ChooseAction(state, actor)
	}
	return udm.enemyStrategy.ChooseAction(state, actor)
}
