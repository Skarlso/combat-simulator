package strategy

import (
	"testing"

	"github.com/skarlso/combat-simulator/internal/models"
)

func TestChooseDeathMove(t *testing.T) {
	ogre := func(hp int) *models.Combatant {
		e := models.NewAdversary("e", "Ogre", "Tier 1", "Standard", 12, 2, 10, 3, []int{5, 9})
		e.HP = hp
		return e
	}
	ally := func(hp int) *models.Combatant {
		p := models.NewCombatant("p2", "Friend", models.SidePlayer, 1, 10, 1, 6, 6)
		p.HP = hp
		return p
	}
	resistant := ogre(3)
	resistant.AddResistance(models.DamagePhysical)

	blaze, risk := DeathMoveBlazeOfGlory, DeathMoveRiskItAll
	cases := []struct {
		name            string
		allies          []*models.Combatant
		enemies         []*models.Combatant
		lastManStanding DeathMove
		want            DeathMove
	}{
		{"allies can finish it", []*models.Combatant{ally(6)}, []*models.Combatant{ogre(5)}, blaze, DeathMoveAvoidDeath},
		{"allies are losing", []*models.Combatant{ally(2)}, []*models.Combatant{ogre(10)}, blaze, risk},
		{"last PC against several", nil, []*models.Combatant{ogre(1), ogre(1)}, blaze, risk},
		{"last PC against one too healthy to finish", nil, []*models.Combatant{ogre(4)}, blaze, risk},
		// d8+2 criticals for at least 8+1+2 = 11: Severe, 3 Hit Points
		{"last PC can finish the last enemy", nil, []*models.Combatant{ogre(3)}, blaze, blaze},
		{"last PC told to roll instead", nil, []*models.Combatant{ogre(3)}, risk, risk},
		{"losing allies, the critical ends it", []*models.Combatant{ally(1)}, []*models.Combatant{ogre(3)}, blaze, blaze},
		{"the setting is only for the last PC", []*models.Combatant{ally(1)}, []*models.Combatant{ogre(3)}, risk, blaze},
		// Halved to 6 the same critical is only Major
		{"resistance takes the kill away", nil, []*models.Combatant{resistant}, blaze, risk},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pc := models.NewCombatant("p1", "Hero", models.SidePlayer, 1, 10, 1, 6, 6)
			pc.DamageDice = "1d8+2"
			pc.HP, pc.IsDefeated = 0, true
			state := models.NewCombatState(append([]*models.Combatant{pc}, c.allies...), c.enemies, 0, 0, 3)

			if got := ChooseDeathMove(state, pc, c.lastManStanding); got != c.want {
				t.Errorf("ChooseDeathMove = %s, want %s", got, c.want)
			}
		})
	}
}
