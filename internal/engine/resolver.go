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
	if featureResolver == nil {
		featureResolver = &FeatureResolver{dice: dice}
	}
	return &CombatResolver{dice: dice, featureResolver: featureResolver}
}

// AttackResult contains outcome of an attack
type AttackResult struct {
	Hit             bool
	Roll            DCCheckResult // the attack roll, including Hope/Fear generated and spotlight shift
	DamageRoll      int           // Raw damage rolled (total)
	DamageDice      []int         // Individual dice values
	DamageBonus     int           // Flat bonus
	ProcessedDamage int           // Damage after resistance/immunity/vulnerability
	HPDamage        int           // Actual HP damage (1/2/3 based on threshold)
	Threshold       models.DamageThreshold
	DamageResult    *models.DamageResult
}

// ResolveAttack handles a basic attack from attacker to defender
func (cr *CombatResolver) ResolveAttack(attacker, defender *models.Combatant, damageDice string, state *models.CombatState) AttackResult {
	roll := cr.featureResolver.rollDCCheck(attacker, defender, "target", 0, state)
	result := AttackResult{Hit: roll.Success, Roll: roll}
	if !roll.Success {
		return result
	}

	notation, err := models.ParseDiceNotation(damageDice)
	if err != nil || len(notation.Terms) == 0 {
		notation, _ = models.ParseDiceNotation("1d6")
	}

	damage := rollNotationDetailed(cr.dice, notation)
	if roll.IsCritical {
		damage.Total += notation.MaxDice()
	}
	if attacker.HalfDamage {
		damage.Total /= 2
	}

	// Resistance and Vulnerability are applied inside TakeDamage so every damage source agrees
	damageRes := defender.TakeDamage(damage.Total, models.DamagePhysical, cr.dice)
	result.DamageRoll = damage.Total
	result.DamageDice = damage.DiceValues
	result.DamageBonus = damage.FlatBonus
	result.ProcessedDamage = damageRes.ProcessedDamage
	result.HPDamage = damageRes.HPDamage
	result.Threshold = damageRes.Threshold
	result.DamageResult = &damageRes

	if damageRes.HPDamage > 0 {
		cr.featureResolver.ProcessTriggeredFeatures(defender, attacker, state, models.TriggerOnDamageTaken, false)
	}
	return result
}

// ProcessConditionTicks handles damage-over-time and condition expiry
func (cr *CombatResolver) ProcessConditionTicks(combatant *models.Combatant, trigger models.TriggerType) {
	kept := combatant.Conditions[:0]
	for _, condition := range combatant.Conditions {
		if condition.DamagePerTick > 0 && trigger == models.TriggerTurnStart {
			_ = combatant.TakeDamage(condition.DamagePerTick, condition.DamageType, cr.dice)
		}
		if condition.ShouldExpire(trigger) {
			continue
		}
		if condition.Duration == models.DurationRounds {
			condition.Tick()
		}
		kept = append(kept, condition)
	}
	combatant.Conditions = kept
}
