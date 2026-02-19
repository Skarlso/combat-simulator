package engine

import (
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// CombatResolver handles attack resolution and damage calculation
type CombatResolver struct {
	dice            DiceRoller
	featureResolver *FeatureResolver
}

// NewCombatResolver creates a combat resolver
func NewCombatResolver(dice DiceRoller, featureResolver *FeatureResolver) *CombatResolver {
	return &CombatResolver{
		dice:            dice,
		featureResolver: featureResolver,
	}
}

// AttackResult contains outcome of an attack
type AttackResult struct {
	Hit             bool
	DamageRoll      int   // Raw damage rolled (total)
	ProcessedDamage int   // Damage after resistance/immunity/vulnerability
	DamageDice      []int // Individual dice values
	DamageBonus     int   // Flat bonus
	HPDamage        int   // Actual HP damage (1/2/3 based on threshold)
	DamageType      models.DamageType
	Threshold       models.DamageThreshold
	DualityResult   DualityRollResult // Player attacks only
	D20Result       D20RollResult     // Enemy attacks only
	AttackRoll      int               // The final attack roll value
	GeneratedHope   bool
	GeneratedFear   bool
	CriticalResult  bool
	DamageResult    *models.DamageResult // Full damage result with armor/HP tracking
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

	if diceCount == 0 {
		diceCount = 1
		diceSize = 6
	}

	if !attacker.IsAdversary && attacker.Proficiency > 0 {
		diceCount = attacker.Proficiency
	}

	if attacker.Side == models.SidePlayer {
		hopeModifier := attacker.GetAttackBonus()
		fearModifier := 0

		hasAdv := attacker.HasCondition(models.ConditionAdvantage) || defender.HasCondition(models.ConditionVulnerable)
		hasDisadv := attacker.HasCondition(models.ConditionDisadvantage) || defender.HasCondition(models.ConditionHidden)

		if hasAdv && !hasDisadv {
			hopeModifier += cr.dice.RollD6()
		} else if hasDisadv && !hasAdv {
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
				state.AddHope(1)        // Critical Hope generates extra
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
		attackModifier := attacker.GetAttackBonus()
		hasAdv := defender.HasCondition(models.ConditionVulnerable)
		hasDisadv := defender.HasCondition(models.ConditionHidden)
		if hasAdv && !hasDisadv {
			attackModifier += cr.dice.RollD6()
		} else if hasDisadv && !hasAdv {
			attackModifier -= cr.dice.RollD6()
		}
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

	// Note: Vulnerability/Resistance are now handled in TakeDamage() to prevent stacking
	// This ensures consistent application regardless of damage source (attack, ability, effect)

	result.DamageRoll = damageRoll
	result.DamageDice = damageResult.DiceValues
	result.DamageBonus = damageResult.FlatBonus
	result.DamageType = models.DamagePhysical // Default to physical damage

	// Apply damage to defender (TakeDamage calculates threshold from processed damage)
	damageRes := defender.TakeDamage(damageRoll, models.DamagePhysical, cr.dice)
	result.ProcessedDamage = damageRes.ProcessedDamage
	result.HPDamage = damageRes.HPDamage
	result.Threshold = damageRes.Threshold
	result.DamageResult = &damageRes // Store full damage result for action logging

	// Trigger on_damage_taken features for the defender if damage was dealt
	if damageRes.HPDamage > 0 && cr.featureResolver != nil {
		cr.featureResolver.ProcessTriggeredFeatures(defender, attacker, state, models.TriggerOnDamageTaken, false)
	}

	return result
}

// ProcessConditionTicks handles damage-over-time and condition expiry
func (cr *CombatResolver) ProcessConditionTicks(combatant *models.Combatant, trigger models.TriggerType) {
	toRemove := make([]int, 0)

	for i, condition := range combatant.Conditions {
		// Apply damage-over-time
		if condition.DamagePerTick > 0 && trigger == models.TriggerTurnStart {
			_ = combatant.TakeDamage(condition.DamagePerTick, condition.DamageType, cr.dice)
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
