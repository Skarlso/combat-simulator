package engine

import (
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func healthPercent(c *models.Combatant) int {
	if c.HPMax == 0 {
		return 0
	}
	return c.HP * 100 / c.HPMax
}

// wantsToShield: a guardian steps in for an ally at half health or less when they are the
// healthier of the two and the hit will not down them
func wantsToShield(guardian, defender *models.Combatant, damage int) bool {
	if healthPercent(defender) > 50 || healthPercent(guardian) <= healthPercent(defender) {
		return false
	}
	return guardian.HP > models.DamageThresholdFromAmount(damage, guardian).HPMarked()
}

// shieldFor finds an ally with an on_ally_damaged redirect reaction (I Am Your Shield) willing
// to take the hit meant for defender, pays its cost, and returns them; nil means nobody steps in.
func (fr *FeatureResolver) shieldFor(defender *models.Combatant, damage int, state *models.CombatState) *models.Combatant {
	if state == nil || damage <= 0 {
		return nil
	}
	for _, ally := range state.GetAlliesOf(defender) {
		if ally == defender {
			continue
		}
		for i := range ally.DomainAbilities {
			ability := &ally.DomainAbilities[i]
			if !ability.Passive || ability.Trigger.Type != models.TriggerOnAllyDamaged || !hasEffect(ability.Effects, models.FeatureEffectRedirectDamage) {
				continue
			}
			if !wantsToShield(ally, defender, damage) || !fr.CanActivateDomainAbility(*ability, ally, state) {
				continue
			}
			err, dcResult := fr.ActivateDomainAbility(ability, ally, defender, state)
			recordTriggered(state, ally, defender, ability.Name, err, dcResult)
			if err != nil {
				continue
			}
			state.RecordAction(models.ActionParams{
				Actor:       ally,
				ActionType:  models.ActionUseAbility,
				Target:      defender,
				FeatureName: ability.Name,
				IsFeature:   true,
				Result:      "Takes the hit for " + defender.Name,
			})
			return ally
		}
	}
	return nil
}

func hasEffect(effects []models.FeatureEffect, category models.FeatureEffectCategory) bool {
	for _, e := range effects {
		if e.Category == category {
			return true
		}
	}
	return false
}
