package strategy

import "github.com/skarlso/combat-simulator/internal/models"

// ChooseDeathMove picks the death move for a PC who just marked their last Hit Point.
// It reads the fight, not the AI preset: the PC stays alive unless the party needs more.
// lastManStanding is the move of the last PC up when a Blaze of Glory would finish the last adversary.
func ChooseDeathMove(state *models.CombatState, pc *models.Combatant, lastManStanding DeathMove) DeathMove {
	enemies := state.GetAliveEnemies()
	switch {
	case partyHolds(state):
		return DeathMoveAvoidDeath
	case len(enemies) == 1 && blazeFinishes(pc, enemies[0]):
		// With nobody left standing the kill only buys a draw; the roll can still win
		if len(state.GetAlivePlayers()) == 0 && lastManStanding == DeathMoveRiskItAll {
			return DeathMoveRiskItAll
		}
		return DeathMoveBlazeOfGlory
	default:
		return DeathMoveRiskItAll
	}
}

// partyHolds reports whether the PCs still standing can be trusted to finish the fight:
// together they have at least as many Hit Points left as the adversaries
func partyHolds(state *models.CombatState) bool {
	allies := state.GetAlivePlayers()
	standing, opposing := 0, 0
	for _, ally := range allies {
		standing += ally.HP
	}
	for _, enemy := range state.GetAliveEnemies() {
		opposing += enemy.HP
	}
	return len(allies) > 0 && standing >= opposing
}

// blazeFinishes reports whether even the weakest Blaze of Glory marks the target's last Hit Points
func blazeFinishes(pc, target *models.Combatant) bool {
	notation, err := models.ParseDiceNotation(pc.AttackDamageFormula())
	if err != nil {
		return false
	}

	// A critical adds the dice maximum to the roll, so the floor is that plus every die showing 1
	damage := notation.MaxDice()
	for _, term := range notation.Terms {
		switch {
		case term.Count == 0:
			damage += term.Size
		case term.Count > 0:
			damage += term.Count
		default:
			damage += term.Count * term.Size
		}
	}

	resistant := target.HasCondition(models.ConditionResistant)
	for _, resist := range target.Resistances {
		if resist.Type != models.DamagePhysical {
			continue
		}
		if resist.IsImmune {
			return false
		}
		resistant = true
	}
	if resistant {
		damage = (damage + 1) / 2
	}
	return models.DamageThresholdFromAmount(damage, target).HPMarked() >= target.HP
}
