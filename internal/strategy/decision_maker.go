package strategy

import (
	"math/rand"

	"github.com/skarlso/combat-simulator/internal/models"
)

// ActionDecision represents the AI's choice for a turn
type ActionDecision struct {
	Target           *models.Combatant
	Potion           models.PotionType        // spend the turn drinking this potion
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
	// PCs drink a Health Potion at or below this HP percentage (0 = never)
	PotionHPThreshold int

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
		ConditionClearing:   ClearImmediately,
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
		ConditionClearing:   ClearImmediately,
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
		ConditionClearing:   ClearImmediately,
		TargetSelection:     TargetRandom,
		AbilityUsage:        AbilityContextual,
		LowHPThreshold:      35,
		HighThreatThreshold: 10,
	}
}

func (ai *AIDecisionMaker) ChooseAction(state *models.CombatState, actor *models.Combatant) *ActionDecision {
	if potion := ai.wantsPotion(actor); potion != "" {
		return &ActionDecision{Target: actor, Potion: potion}
	}
	if ai.shouldClearCondition(actor) {
		return &ActionDecision{Target: actor, ClearCondition: true}
	}

	target := ai.selectTargetFrom(state.GetEnemiesOf(actor))
	if target == nil {
		return &ActionDecision{}
	}
	return ai.evaluateBestAction(state, actor, target)
}

func hpPercent(c *models.Combatant) int {
	return c.HP * 100 / c.HPMax
}

// wantsPotion picks a Health Potion below the HP threshold, or a Stamina Potion when one
// more Stress would cost a Hit Point. Only PCs carry potions.
func (ai *AIDecisionMaker) wantsPotion(actor *models.Combatant) models.PotionType {
	if actor.Side != models.SidePlayer {
		return ""
	}
	if ai.PotionHPThreshold > 0 && actor.HealthPotions > 0 && hpPercent(actor) <= ai.PotionHPThreshold {
		return models.PotionHealth
	}
	if actor.StaminaPotions > 0 && actor.StressMax > 0 && actor.Stress >= actor.StressMax-1 {
		return models.PotionStamina
	}
	return ""
}

// Every preset clears immediately for now; ClearableCondition already leaves out
// conditions that do nothing in this simulator.
func (ai *AIDecisionMaker) shouldClearCondition(actor *models.Combatant) bool {
	if actor.ClearableCondition() == nil {
		return false
	}

	switch ai.ConditionClearing {
	case ClearImmediately:
		return true
	case ClearWhenSafe:
		return hpPercent(actor) > 50
	case ClearProbabilistic:
		threshold := 60
		if hpPercent(actor) < 30 {
			threshold = 80
		}
		return ai.rng.Intn(100) < threshold
	default:
		return false
	}
}

// selectTargetFrom chooses from a list of candidates
func (ai *AIDecisionMaker) selectTargetFrom(candidates []*models.Combatant) *models.Combatant {
	if len(candidates) == 0 {
		return nil
	}

	switch ai.TargetSelection {
	case TargetLowestHP:
		return pickMax(candidates, func(c *models.Combatant) int { return -c.HP })
	case TargetHighestHP:
		return pickMax(candidates, func(c *models.Combatant) int { return c.HP })
	case TargetRandom:
		return candidates[ai.rng.Intn(len(candidates))]
	case TargetHighestThreat:
		return pickMax(candidates, calculateThreat)
	default:
		return candidates[0]
	}
}

// pickMax returns the first candidate with the highest score
func pickMax(candidates []*models.Combatant, score func(*models.Combatant) int) *models.Combatant {
	best := candidates[0]
	for _, c := range candidates[1:] {
		if score(c) > score(best) {
			best = c
		}
	}
	return best
}

// calculateThreat rates a combatant by weapon dice, remaining health and number of tricks
func calculateThreat(c *models.Combatant) int {
	return c.Proficiency*2 + hpPercent(c)/10 + len(c.AdversaryFeatures) + len(c.DomainAbilities)
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
		return hpPercent(actor) < ai.LowHPThreshold || len(state.GetEnemiesOf(actor)) <= 1
	case AbilityContextual:
		return hpPercent(actor) < ai.LowHPThreshold || ai.rng.Intn(100) < 60
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
		return hpPercent(actor) < ai.LowHPThreshold
	case ResourceRandom:
		return ai.rng.Intn(100) < 50
	case ResourceClever:
		return len(state.GetEnemiesOf(actor)) <= 1 || hpPercent(actor) < 50 || ai.rng.Intn(100) < 40
	default:
		return true
	}
}

// evaluateBasicAttack estimates expected value of a basic attack
func (ai *AIDecisionMaker) evaluateBasicAttack(actor *models.Combatant, target *models.Combatant) float64 {
	avgDamage := ai.estimateAverageDamage(actor.AttackDamageFormula())
	hitChance := ai.estimateSuccessProbability(actor, target, "target", 0, "")
	return avgDamage * hitChance
}

// estimateSuccessProbability estimates the chance of success against a DC
func (ai *AIDecisionMaker) estimateSuccessProbability(actor *models.Combatant, target *models.Combatant, dcType string, fixedDC int, rollTrait string) float64 {
	if dcType == "" || dcType == "none" {
		return 1.0 // Auto-succeed
	}

	var dc int
	switch dcType {
	case "target":
		if target == nil {
			return 0.5
		}
		dc = target.GetDefense()
	case "fixed":
		dc = fixedDC
	default:
		return 1.0
	}

	// Duality Dice average 13, a d20 averages 10; the bands are a rough curve around that
	avgRoll, odds := 13, [4]float64{0.85, 0.65, 0.45, 0.25}
	if actor.Side != models.SidePlayer {
		avgRoll, odds = 10, [4]float64{0.80, 0.60, 0.40, 0.20}
	}
	switch diff := avgRoll + actor.RollModifier(rollTrait) - dc; {
	case diff >= 5:
		return odds[0]
	case diff >= 0:
		return odds[1]
	case diff >= -3:
		return odds[2]
	default:
		return odds[3]
	}
}

// evaluateEffects values a set of effects, scaled by how likely the roll is to succeed
func (ai *AIDecisionMaker) evaluateEffects(state *models.CombatState, actor, target *models.Combatant, effects []models.FeatureEffect, dcType string, fixedDC int, rollTrait string) float64 {
	value := 0.0
	for _, effect := range effects {
		value += ai.evaluateEffect(effect, actor, target, state)
	}
	return value * ai.estimateSuccessProbability(actor, target, dcType, fixedDC, rollTrait)
}

// evaluateAbility estimates expected value of a domain ability; Hope is dearer than Stress
func (ai *AIDecisionMaker) evaluateAbility(state *models.CombatState, actor *models.Combatant, target *models.Combatant, ability *models.DomainAbility) float64 {
	value := ai.evaluateEffects(state, actor, target, ability.Effects, ability.DCType, ability.FixedDC, ability.RollTrait)
	return value - float64(ability.HopeCost)*1.5 - float64(ability.StressCost)*0.5
}

// evaluateFeature estimates expected value of an adversary feature. Features are the
// stat block's designed moves, so they get a flat preference over a generic attack.
func (ai *AIDecisionMaker) evaluateFeature(state *models.CombatState, actor *models.Combatant, target *models.Combatant, feature *models.AdversaryFeature) float64 {
	value := ai.evaluateEffects(state, actor, target, feature.Effects, feature.DCType, feature.FixedDC, "")
	return value - float64(feature.FearCost)*0.5 - float64(feature.StressCost)*0.5 + 2.0
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
	switch {
	case effect.StandardAttack:
		effect.DiceFormula = actor.AttackDamageFormula()
	case effect.ProficiencyDice:
		effect.DiceFormula = actor.ScaleToProficiency(effect.DiceFormula)
	default:
		effect.DiceFormula = actor.ScaleDice(effect.DiceFormula, effect.ScaleBy)
	}

	avgAmount := ai.estimateAverageDamage(effect.DiceFormula) + float64(effect.Magnitude)

	switch effect.Category {
	case models.FeatureEffectDamage, models.FeatureEffectDirectDamage:
		return avgAmount

	case models.FeatureEffectAreaDamage:
		enemyCount := len(state.GetEnemiesOf(actor))
		if ai.AreaTargets > 0 {
			enemyCount = min(enemyCount, ai.AreaTargets)
		}
		return avgAmount * float64(enemyCount)

	case models.FeatureEffectApplyPoison, models.FeatureEffectApplyIgnite:
		if target.HasCondition(models.ConditionPoisoned) || target.HasCondition(models.ConditionBurning) {
			return 0.0
		}
		// Status effects are valuable on high HP targets
		if target.HP > target.HPMax/2 {
			return 3.0 // Worth about 3 damage over time
		}
		return 1.5

	case models.FeatureEffectApplyVulnerable:
		if target.HasCondition(models.ConditionVulnerable) || target.ImmuneTo(models.ConditionVulnerable) {
			return 0.0
		}
		// Very valuable on high HP targets
		if target.HP > target.HPMax/2 {
			return 5.0
		}
		return 2.0

	case models.FeatureEffectApplyStunned:
		if target.HasCondition(models.ConditionStunned) {
			return 0.0
		}
		return 4.0

	case models.FeatureEffectApplyRestrained:
		// Only limits movement, which is not modelled: never worth a spotlight
		return 0.0

	case models.FeatureEffectStressInflict:
		return avgAmount * 0.3

	case models.FeatureEffectMarkHP:
		// A forced Hit Point is worth the damage it takes to mark one: the target's Major threshold
		major, _ := target.Thresholds()
		return float64(min(effect.Magnitude, target.HP) * max(major, 1))

	case models.FeatureEffectApplyDisadvantage:
		if target.HasCondition(models.ConditionDisadvantage) {
			return 0.0
		}
		return 3.0

	case models.FeatureEffectDefensePenalty:
		gain := actor.RiderAmount(effect, 0, nil) - target.DefensePenalty
		return float64(max(gain, 0)) * 1.5

	case models.FeatureEffectHPHeal:
		// Damage effects are valued in raw damage, so a healed Hit Point is worth the
		// damage it takes to mark one: the recipient's Major threshold. Overheal is wasted.
		value := 0.0
		for _, c := range state.Recipients(actor, effect.TargetType, models.MissingHP) {
			major, _ := c.Thresholds()
			value += min(avgAmount, float64(models.MissingHP(c))) * float64(max(major, 1))
		}
		return value

	case models.FeatureEffectStressClear:
		value := 0.0
		for _, c := range state.Recipients(actor, effect.TargetType, models.StressHeld) {
			value += min(avgAmount, float64(c.Stress)) * 0.5
		}
		return value

	case models.FeatureEffectArmorDamage:
		// With no slot left to mark, the target marks a Hit Point instead
		if target.ArmorSlots > 0 {
			return 2.0
		}
		return 3.0

	case models.FeatureEffectArmorClear:
		// A cleared slot will absorb one Hit Point later, so it is valued like a healed one
		value := 0.0
		for _, c := range state.Recipients(actor, effect.TargetType, models.MarkedArmor) {
			major, _ := c.Thresholds()
			value += min(avgAmount, float64(models.MarkedArmor(c))) * float64(max(major, 1))
		}
		return value

	case models.FeatureEffectGrantArmor:
		if actor.ArmorSlots < 2 {
			return float64(effect.Magnitude) * 2.0
		}
		return float64(effect.Magnitude)

	case models.FeatureEffectDamageReduction:
		return avgAmount

	case models.FeatureEffectTokenPlace, models.FeatureEffectTokenSpend:
		return 2.0

	case models.FeatureEffectSummonAllies:
		// Each body is worth about 10 HP; arrivals that act at once also deliver an attack this GM turn
		quantity := avgAmount
		spotlighted := float64(effect.SpotlightCount)
		if effect.SpotlightAll {
			spotlighted = quantity
		}
		return quantity*10.0 + spotlighted*3.0

	case models.FeatureEffectAllySpotlight:
		return ai.evaluateAllySpotlight(effect, actor, target, state)

	default:
		return 1.0
	}
}

// evaluateAllySpotlight sums the basic attacks the chosen allies would make right now
func (ai *AIDecisionMaker) evaluateAllySpotlight(effect models.FeatureEffect, actor *models.Combatant, target *models.Combatant, state *models.CombatState) float64 {
	allies := state.EligibleSpotlightAllies(actor, effect)
	if len(allies) == 0 {
		return 0.0
	}

	count := len(allies)
	if !effect.SpotlightAll {
		count = min(effect.Magnitude+int(ai.estimateAverageDamage(effect.DiceFormula)), len(allies))
	}

	value := 0.0
	for _, ally := range allies[:count] {
		value += ai.evaluateBasicAttack(ally, target)
	}
	if effect.HalfDamage {
		value /= 2
	}
	return value
}

// estimateAverageDamage is the expected total of a dice formula such as "2d6+3"
func (ai *AIDecisionMaker) estimateAverageDamage(diceFormula string) float64 {
	parsed, err := models.ParseDiceNotation(diceFormula)
	if err != nil {
		return 0.0
	}
	total := 0.0
	for _, term := range parsed.Terms {
		if term.Count == 0 {
			total += float64(term.Size)
		} else {
			total += float64(term.Count) * float64(term.Size+1) / 2.0
		}
	}
	return total
}

// canAffordAbility checks if actor can pay costs for ability
func (ai *AIDecisionMaker) canAffordAbility(state *models.CombatState, actor *models.Combatant, ability *models.DomainAbility) bool {
	return ability.HopeCost <= actor.Hope && ability.StressCost+actor.Stress <= actor.StressMax
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

// SetPotionThreshold sets the HP percentage at or below which PCs drink a Health Potion
func (udm *UnifiedDecisionMaker) SetPotionThreshold(percent int) {
	udm.playerStrategy.PotionHPThreshold = percent
}

func (udm *UnifiedDecisionMaker) ChooseAction(state *models.CombatState, actor *models.Combatant) *ActionDecision {
	if actor.Side == models.SidePlayer {
		return udm.playerStrategy.ChooseAction(state, actor)
	}
	return udm.enemyStrategy.ChooseAction(state, actor)
}
