package engine

import (
	"math/rand"
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func newTestResolver() *CombatResolver {
	rng := rand.New(rand.NewSource(42))
	dice := NewStandardDiceRoller(rng)
	fr := NewFeatureResolver(rng)
	return NewCombatResolver(dice, fr)
}

func TestResolveAttackPlayerHit(t *testing.T) {
	resolver := newTestResolver()
	attacker := models.NewCombatant("p1", "Hero", models.SidePlayer, 3, 12, 4, 15, 8)
	defender := models.NewAdversary("e1", "Goblin", "Tier 1", "Standard", 5, 3, 10, 4, []int{4, 7})
	state := models.NewCombatState([]*models.Combatant{attacker}, []*models.Combatant{defender}, 0, 3, 5)

	hitCount := 0
	for i := 0; i < 100; i++ {
		defenderCopy := models.CloneCombatant(defender, "e1")
		stateCopy := models.NewCombatState([]*models.Combatant{attacker}, []*models.Combatant{defenderCopy}, 0, 3, 5)
		result := resolver.ResolveAttack(attacker, defenderCopy, "1d6", stateCopy)
		if result.Hit {
			hitCount++
		}
	}

	if hitCount == 0 {
		t.Error("should have at least one hit in 100 attempts against low defense")
	}
	_ = state
}

func TestResolveAttackEnemyHit(t *testing.T) {
	resolver := newTestResolver()
	attacker := models.NewAdversary("e1", "Orc", "Tier 1", "Bruiser", 12, 5, 15, 6, []int{5, 10})
	defender := models.NewCombatant("p1", "Hero", models.SidePlayer, 3, 12, 2, 15, 8)
	state := models.NewCombatState([]*models.Combatant{defender}, []*models.Combatant{attacker}, 0, 3, 5)

	result := resolver.ResolveAttack(attacker, defender, "1d8+2", state)
	if result.D20Result.Modifier != 5 {
		t.Errorf("enemy attack modifier = %d, want 5 (ATK)", result.D20Result.Modifier)
	}
}

func TestResolveAttackGeneratesHopeOrFear(t *testing.T) {
	resolver := newTestResolver()
	attacker := models.NewCombatant("p1", "Hero", models.SidePlayer, 3, 12, 2, 15, 8)
	defender := models.NewAdversary("e1", "Goblin", "Tier 1", "Standard", 5, 3, 10, 4, []int{4, 7})

	hopeGenerated := false
	fearGenerated := false

	for i := 0; i < 100; i++ {
		defenderCopy := models.CloneCombatant(defender, "e1")
		state := models.NewCombatState([]*models.Combatant{attacker}, []*models.Combatant{defenderCopy}, 0, 0, 5)
		result := resolver.ResolveAttack(attacker, defenderCopy, "1d6", state)
		if result.GeneratedHope {
			hopeGenerated = true
		}
		if result.GeneratedFear {
			fearGenerated = true
		}
		if hopeGenerated && fearGenerated {
			break
		}
	}

	if !hopeGenerated {
		t.Error("should generate hope at least once in 100 rolls")
	}
	if !fearGenerated {
		t.Error("should generate fear at least once in 100 rolls")
	}
}

func TestResolveAttackDamageApplied(t *testing.T) {
	resolver := newTestResolver()
	attacker := models.NewCombatant("p1", "Hero", models.SidePlayer, 3, 12, 20, 15, 8)
	defender := models.NewAdversary("e1", "Goblin", "Tier 1", "Standard", 2, 3, 10, 4, []int{4, 7})
	state := models.NewCombatState([]*models.Combatant{attacker}, []*models.Combatant{defender}, 0, 3, 5)

	result := resolver.ResolveAttack(attacker, defender, "2d6+3", state)
	if result.Hit {
		if result.DamageRoll <= 0 {
			t.Error("hit should have positive damage roll")
		}
		if result.DamageResult == nil {
			t.Error("hit should have DamageResult")
		}
		if defender.HP >= defender.HPMax && !result.CriticalResult {
			t.Error("defender should take damage on hit")
		}
	}
}

func TestResolveAttackDefaultDice(t *testing.T) {
	resolver := newTestResolver()
	attacker := models.NewCombatant("p1", "Hero", models.SidePlayer, 3, 12, 20, 15, 8)
	defender := models.NewAdversary("e1", "Goblin", "Tier 1", "Standard", 2, 3, 10, 4, []int{4, 7})
	state := models.NewCombatState([]*models.Combatant{attacker}, []*models.Combatant{defender}, 0, 3, 5)

	result := resolver.ResolveAttack(attacker, defender, "invalid_dice", state)
	_ = result
}

func TestProcessConditionTicks(t *testing.T) {
	resolver := newTestResolver()
	c := models.NewCombatant("p1", "Hero", models.SidePlayer, 3, 12, 2, 15, 8)
	c.AddCondition(&models.ActiveCondition{
		Type:          models.ConditionBurning,
		Duration:      models.DurationRounds,
		RoundsLeft:    2,
		DamagePerTick: 3,
		DamageType:    models.DamagePhysical,
	})

	hpBefore := c.HP
	resolver.ProcessConditionTicks(c, models.TriggerTurnStart)

	if c.HP >= hpBefore {
		t.Error("burning should deal damage on turn start")
	}
	if !c.HasCondition(models.ConditionBurning) {
		t.Error("burning should still be active (1 round left)")
	}
}

func TestProcessConditionTicksExpiry(t *testing.T) {
	resolver := newTestResolver()
	c := models.NewCombatant("p1", "Hero", models.SidePlayer, 3, 12, 2, 15, 8)
	c.AddCondition(&models.ActiveCondition{
		Type:     models.ConditionVulnerable,
		Duration: models.DurationEndOfTurn,
	})

	resolver.ProcessConditionTicks(c, models.TriggerTurnEnd)

	if c.HasCondition(models.ConditionVulnerable) {
		t.Error("end_of_turn condition should expire on turn end")
	}
}

func TestProcessConditionTicksDoNotExpirePrematurely(t *testing.T) {
	resolver := newTestResolver()
	c := models.NewCombatant("p1", "Hero", models.SidePlayer, 3, 12, 2, 15, 8)
	c.AddCondition(&models.ActiveCondition{
		Type:     models.ConditionVulnerable,
		Duration: models.DurationEndOfTurn,
	})

	resolver.ProcessConditionTicks(c, models.TriggerTurnStart)

	if !c.HasCondition(models.ConditionVulnerable) {
		t.Error("end_of_turn condition should not expire on turn start")
	}
}
