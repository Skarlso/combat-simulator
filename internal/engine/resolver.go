package engine

import (
	"fmt"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// CombatResolver handles attack resolution and damage calculation
type CombatResolver struct {
	dice DiceRoller
}

// NewCombatResolver creates a combat resolver
func NewCombatResolver(dice DiceRoller) *CombatResolver {
	return &CombatResolver{
		dice: dice,
	}
}

// AttackResult contains outcome of an attack
type AttackResult struct {
	Hit            bool
	DamageRoll     int   // Raw damage rolled (total)
	DamageDice     []int // Individual dice values
	DamageBonus    int   // Flat bonus
	HPDamage       int   // Actual HP damage (1/2/3 based on threshold)
	DamageType     models.DamageType
	Threshold      models.DamageThreshold
	DualityResult  DualityRollResult // Player attacks only
	D20Result      D20RollResult     // Enemy attacks only
	AttackRoll     int               // The final attack roll value
	GeneratedHope  bool
	GeneratedFear  bool
	CriticalResult bool
}

// ResolveAttack handles a basic attack from attacker to defender
func (cr *CombatResolver) ResolveAttack(attacker, defender *models.Combatant, damageEffect models.DamageEffect, state *models.CombatState) AttackResult {
	result := AttackResult{}

	// Players use duality dice, enemies use d20
	if attacker.Side == models.SidePlayer {
		// Player attack - use duality dice
		hopeModifier := attacker.GetAttackBonus()
		fearModifier := 0

		// Check for advantage/disadvantage from conditions
		if attacker.HasCondition(models.ConditionAdvantage) {
			hopeModifier += 2
		}
		if attacker.HasCondition(models.ConditionDisadvantage) {
			hopeModifier -= 2
		}
		if attacker.IsFullyStressed {
			hopeModifier -= 2
		}

		dualityRoll := cr.dice.RollDuality(hopeModifier, fearModifier)
		result.DualityResult = dualityRoll
		result.AttackRoll = dualityRoll.WithSpirit

		// Check if attack hits
		if dualityRoll.WithSpirit < defender.GetDefense() {
			result.Hit = false
			return result
		}

		// Handle Hope/Fear generation based on duality result
		if dualityRoll.Result == models.ResultHope || dualityRoll.Result == models.ResultCriticalHope {
			state.AddHope(1)
			result.GeneratedHope = true
			if dualityRoll.IsCritical {
				state.AddHope(1) // Critical Hope generates extra
			}
		} else {
			state.AddFear(1)
			result.GeneratedFear = true
			if dualityRoll.IsCritical {
				state.AddFear(1) // Critical Fear generates extra
			}
		}

		result.CriticalResult = dualityRoll.IsCritical
	} else {
		// Enemy attack - use d20
		attackModifier := attacker.GetAttackBonus()
		d20Roll := cr.dice.RollD20Attack(attackModifier)
		result.D20Result = d20Roll
		result.AttackRoll = d20Roll.Total

		// Check if attack hits
		if d20Roll.Total < defender.GetDefense() {
			result.Hit = false
			return result
		}

		result.CriticalResult = d20Roll.IsCritical
	}

	result.Hit = true

	// Roll damage with individual dice tracking
	damageResult := cr.dice.RollDamageDetailed(damageEffect.DiceCount, damageEffect.DiceSize, damageEffect.FlatBonus)
	damageRoll := damageResult.Total

	// Apply vulnerability/resistance from conditions
	if defender.HasCondition(models.ConditionVulnerable) {
		damageRoll = int(float64(damageRoll) * 1.5)
	}
	if defender.HasCondition(models.ConditionResistant) {
		damageRoll = int(float64(damageRoll) * 0.5)
	}

	result.DamageRoll = damageRoll
	result.DamageDice = damageResult.DiceValues
	result.DamageBonus = damageResult.FlatBonus
	result.DamageType = damageEffect.DamageType
	result.Threshold = models.DamageThresholdFromAmount(damageRoll, defender)

	// Apply damage to defender (returns actual HP damage dealt)
	hpDamage := defender.TakeDamage(damageRoll, damageEffect.DamageType)
	result.HPDamage = hpDamage

	return result
}

// ApplyHealing restores HP to target
func (cr *CombatResolver) ApplyHealing(target *models.Combatant, healEffect models.HealEffect) int {
	healAmount := cr.dice.RollDamage(healEffect.DiceCount, healEffect.DiceSize, healEffect.FlatBonus)
	target.Heal(healAmount)
	return healAmount
}

// CheckAbilityConditions verifies if ability conditions are met
func (cr *CombatResolver) CheckAbilityConditions(actor *models.Combatant, ability *models.Ability, state *models.CombatState) bool {
	for _, condition := range ability.Conditions {
		if !cr.evaluateCondition(actor, condition, state) {
			return false
		}
	}
	return true
}

// evaluateCondition checks a single ability condition
func (cr *CombatResolver) evaluateCondition(actor *models.Combatant, condition models.AbilityCondition, state *models.CombatState) bool {
	switch condition.Type {
	case models.ConditionHasStatus:
		condType, ok := condition.Value.(models.ConditionType)
		if !ok {
			return false
		}
		return actor.HasCondition(condType)

	case models.ConditionAllyInMelee:
		// Simplified: check if any ally is alive
		allies := state.GetAlliesOf(actor)
		return len(allies) > 1 // More than just self

	case models.ConditionHPBelow:
		percentage, ok := condition.Value.(int)
		if !ok {
			return false
		}
		currentPercent := (actor.HP * 100) / actor.HPMax
		return currentPercent < percentage

	case models.ConditionHPAbove:
		percentage, ok := condition.Value.(int)
		if !ok {
			return false
		}
		currentPercent := (actor.HP * 100) / actor.HPMax
		return currentPercent > percentage

	case models.ConditionStressBelow:
		threshold, ok := condition.Value.(int)
		if !ok {
			return false
		}
		return actor.Stress < threshold

	case models.ConditionStressAbove:
		threshold, ok := condition.Value.(int)
		if !ok {
			return false
		}
		return actor.Stress > threshold

	case models.ConditionEnemyCount:
		minCount, ok := condition.Value.(int)
		if !ok {
			return false
		}
		enemies := state.GetEnemiesOf(actor)
		return len(enemies) >= minCount

	case models.ConditionAllyCount:
		minCount, ok := condition.Value.(int)
		if !ok {
			return false
		}
		allies := state.GetAlliesOf(actor)
		return len(allies) >= minCount

	default:
		return true
	}
}

// CanAffordAbility checks if actor has resources to use ability
func (cr *CombatResolver) CanAffordAbility(actor *models.Combatant, ability *models.Ability, state *models.CombatState) bool {
	costs := ability.Costs

	// Check HP cost
	if costs.HP > 0 && actor.HP <= costs.HP {
		return false // Would die
	}

	// Check stress cost
	if costs.Stress > 0 && (actor.Stress+costs.Stress) > actor.StressMax {
		return false
	}

	// Check Hope/Fear costs
	if actor.Side == models.SidePlayer {
		if costs.Hope > 0 && state.HopePool < costs.Hope {
			return false
		}
	} else {
		if costs.Fear > 0 && state.FearPool < costs.Fear {
			return false
		}
	}

	return true
}

// PayAbilityCost deducts resources when ability is used
func (cr *CombatResolver) PayAbilityCost(actor *models.Combatant, ability *models.Ability, state *models.CombatState) error {
	if !cr.CanAffordAbility(actor, ability, state) {
		return fmt.Errorf("cannot afford ability %s", ability.Name)
	}

	costs := ability.Costs

	// Pay HP cost
	if costs.HP > 0 {
		actor.TakeDamage(costs.HP, models.DamagePhysical)
	}

	// Pay stress cost
	if costs.Stress > 0 {
		actor.AddStress(costs.Stress)
	}

	// Pay Hope/Fear
	if actor.Side == models.SidePlayer {
		if costs.Hope > 0 {
			state.SpendHope(costs.Hope)
		}
	} else {
		if costs.Fear > 0 {
			state.SpendFear(costs.Fear)
		}
	}

	return nil
}

// ProcessConditionTicks handles damage-over-time and condition expiry
func (cr *CombatResolver) ProcessConditionTicks(combatant *models.Combatant, trigger models.TriggerType) {
	toRemove := make([]int, 0)

	for i, condition := range combatant.Conditions {
		// Apply damage-over-time
		if condition.DamagePerTick > 0 && trigger == models.TriggerTurnStart {
			combatant.TakeDamage(condition.DamagePerTick, condition.DamageType)
		}

		// Check if condition should expire
		if condition.ShouldExpire(trigger) {
			toRemove = append(toRemove, i)
		} else if condition.Duration == models.DurationRounds {
			condition.Tick()
		}
	}

	// Remove expired conditions (in reverse to preserve indices)
	for i := len(toRemove) - 1; i >= 0; i-- {
		idx := toRemove[i]
		combatant.Conditions = append(combatant.Conditions[:idx], combatant.Conditions[idx+1:]...)
	}
}
