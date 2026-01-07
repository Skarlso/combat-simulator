package engine

import (
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
func (cr *CombatResolver) ResolveAttack(attacker, defender *models.Combatant, damageDice string, state *models.CombatState) AttackResult {
	result := AttackResult{}

	// Parse damage dice notation
	notation, err := models.ParseDiceNotation(damageDice)
	if err != nil || len(notation.Terms) == 0 {
		// Default to 1d6 if parsing fails
		notation, _ = models.ParseDiceNotation("1d6")
	}

	// Extract dice count, size, and bonus from first term (simple notation)
	// For now, support simple notation like "2d6+3" (single dice type + bonus)
	diceCount := 0
	diceSize := 6
	flatBonus := 0

	for _, term := range notation.Terms {
		if term.Count == 0 {
			// Flat bonus
			flatBonus += term.Size
		} else {
			// Dice roll - use the first dice term found
			if diceCount == 0 {
				diceCount = term.Count
				diceSize = term.Size
			} else {
				// Additional dice terms - add as bonus (approximation)
				flatBonus += term.Count * (term.Size / 2)
			}
		}
	}

	// Default to 1d6 if no dice found
	if diceCount == 0 {
		diceCount = 1
		diceSize = 6
	}

	// Players use duality dice, enemies use d20
	if attacker.Side == models.SidePlayer {
		// Player attack - use duality dice
		hopeModifier := attacker.GetAttackBonus()
		fearModifier := 0

		// Check for advantage/disadvantage from conditions
		if attacker.HasCondition(models.ConditionAdvantage) {
			hopeModifier += cr.dice.RollD6()
		}
		if attacker.HasCondition(models.ConditionDisadvantage) {
			hopeModifier -= cr.dice.RollD6()
		}
		if attacker.IsFullyStressed {
			hopeModifier -= 2
		}

		dualityRoll := cr.dice.RollDuality(hopeModifier, fearModifier)
		result.DualityResult = dualityRoll
		result.AttackRoll = dualityRoll.WithSpirit
		result.CriticalResult = dualityRoll.IsCritical

		// Handle Hope/Fear generation based on duality result
		if dualityRoll.Result == models.ResultHope || dualityRoll.Result == models.ResultCriticalHope {
			state.AddHope(1)
			result.GeneratedHope = true
			if dualityRoll.IsCritical {
				state.AddHope(1) // Critical Hope generates extra
				attacker.ClearStress(1) // Critical success clears a Stress
			}
		} else {
			state.AddFear(1)
			result.GeneratedFear = true
			if dualityRoll.IsCritical {
				state.AddFear(1) // Critical Fear generates extra
			}
		}

		// Critical success = auto-hit regardless of roll vs defense
		if dualityRoll.IsCritical {
			result.Hit = true
		} else {
			// Normal roll: check if attack hits
			if dualityRoll.WithSpirit < defender.GetDefense() {
				result.Hit = false
				return result
			}
			result.Hit = true
		}
	} else {
		// Enemy attack - use d20
		attackModifier := attacker.GetAttackBonus()
		d20Roll := cr.dice.RollD20Attack(attackModifier)
		result.D20Result = d20Roll
		result.AttackRoll = d20Roll.Total
		result.CriticalResult = d20Roll.IsCritical

		// Natural 20 = auto-hit regardless of total vs defense
		if d20Roll.IsCritical {
			result.Hit = true
		} else {
			// Normal roll: check if attack hits
			if d20Roll.Total < defender.GetDefense() {
				result.Hit = false
				return result
			}
			result.Hit = true
		}
	}

	// Roll damage with individual dice tracking
	damageResult := cr.dice.RollDamageDetailed(diceCount, diceSize, flatBonus)
	damageRoll := damageResult.Total

	// Apply critical damage bonus
	if result.CriticalResult {
		maxDiceResult := diceCount * diceSize
		damageRoll += maxDiceResult
	}

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
	result.DamageType = models.DamagePhysical // Default to physical damage
	result.Threshold = models.DamageThresholdFromAmount(damageRoll, defender)

	// Apply damage to defender (returns actual HP damage dealt)
	hpDamage := defender.TakeDamage(damageRoll, models.DamagePhysical)
	result.HPDamage = hpDamage

	return result
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
