package engine

import (
	"slices"
	"strings"
	"testing"

	"github.com/skarlso/combat-simulator/internal/models"
	"github.com/skarlso/combat-simulator/internal/strategy"
)

func healParty() (*models.Combatant, *models.Combatant, *models.Combatant, *models.CombatState) {
	healer := tPlayer("p1", "Healer")
	wounded := tPlayer("p2", "Wounded")
	wounded.HP = 5
	scratched := tPlayer("p3", "Scratched")
	scratched.HP = 8
	state := newStateWith([]*models.Combatant{healer, wounded, scratched}, []*models.Combatant{tEnemy("e1", "Ogre", 10)}, 0, 0)
	return healer, wounded, scratched, state
}

func TestAllyHealGoesToTheMostWounded(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	healer, wounded, scratched, state := healParty()

	effect := models.FeatureEffect{Category: models.FeatureEffectHPHeal, Magnitude: 2, TargetType: models.FeatureTargetAlly}
	if err := fr.ResolveEffect(effect, healer, nil, state, nil); err != nil {
		t.Fatal(err)
	}
	if wounded.HP != 7 || scratched.HP != 8 || healer.HP != 10 {
		t.Errorf("HP = healer %d, wounded %d, scratched %d; want 10 / 7 / 8", healer.HP, wounded.HP, scratched.HP)
	}
}

func TestAlliesHealReachesEveryone(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	healer, wounded, scratched, state := healParty()
	healer.HP = 9

	effect := models.FeatureEffect{Category: models.FeatureEffectHPHeal, Magnitude: 1, TargetType: models.FeatureTargetAllies}
	if err := fr.ResolveEffect(effect, healer, nil, state, nil); err != nil {
		t.Fatal(err)
	}
	if healer.HP != 10 || wounded.HP != 6 || scratched.HP != 9 {
		t.Errorf("HP = healer %d, wounded %d, scratched %d; want 10 / 6 / 9", healer.HP, wounded.HP, scratched.HP)
	}
}

func TestSelfHealStaysOnTheActor(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	healer, wounded, _, state := healParty()
	healer.HP = 6

	effect := models.FeatureEffect{Category: models.FeatureEffectHPHeal, Magnitude: 2}
	if err := fr.ResolveEffect(effect, healer, nil, state, nil); err != nil {
		t.Fatal(err)
	}
	if healer.HP != 8 || wounded.HP != 5 {
		t.Errorf("HP = healer %d, wounded %d; want 8 / 5", healer.HP, wounded.HP)
	}
}

func TestAllyHealNeedsSomeoneWounded(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	healer, wounded, scratched, state := healParty()
	ability := models.DomainAbility{Name: "Healing Hands", StressCost: 1,
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectHPHeal, Magnitude: 2, TargetType: models.FeatureTargetAlly}}}

	if !fr.CanActivateDomainAbility(ability, healer, state) {
		t.Fatal("should be usable while an ally is wounded")
	}
	wounded.HP, scratched.HP = wounded.HPMax, scratched.HPMax
	if fr.CanActivateDomainAbility(ability, healer, state) {
		t.Error("should not be usable when nobody is wounded")
	}
}

func TestAlliesStressClear(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	healer, wounded, _, state := healParty()
	healer.Stress, wounded.Stress = 3, 2

	effect := models.FeatureEffect{Category: models.FeatureEffectStressClear, Magnitude: 1, TargetType: models.FeatureTargetAllies}
	if err := fr.ResolveEffect(effect, healer, nil, state, nil); err != nil {
		t.Fatal(err)
	}
	if healer.Stress != 2 || wounded.Stress != 1 {
		t.Errorf("Stress = healer %d, wounded %d; want 2 / 1", healer.Stress, wounded.Stress)
	}
}

func shieldAbility() models.DomainAbility {
	return models.DomainAbility{
		Name: "I Am Your Shield", Passive: true, StressCost: 1,
		Trigger: models.FeatureTriggerCondition{Type: models.TriggerOnAllyDamaged},
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectRedirectDamage}},
	}
}

func TestGuardianTakesTheHitForAWoundedAlly(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 5}, &firstPicker{})
	hero := tPlayer("p1", "Hero")
	hero.HP = 4
	guardian := tPlayer("p2", "Guardian")
	guardian.DomainAbilities = []models.DomainAbility{shieldAbility()}
	ogre := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{hero, guardian}, []*models.Combatant{ogre}, 0, 0)

	ce.executeAttack(ogre, hero, state)

	if hero.HP != 4 || guardian.HP != 9 || guardian.Stress != 1 {
		t.Errorf("hero HP %d, guardian HP %d, guardian Stress %d; want 4 / 9 / 1", hero.HP, guardian.HP, guardian.Stress)
	}
	last := state.ActionHistory[len(state.ActionHistory)-1]
	if last.Target != "Guardian" || last.Result != "Hit" {
		t.Errorf("attack logged against %q (%s), want Guardian / Hit", last.Target, last.Result)
	}
	shieldLogged := false
	for _, a := range state.ActionHistory {
		shieldLogged = shieldLogged || strings.HasPrefix(a.Result, "Takes the hit for Hero")
	}
	if !shieldLogged {
		t.Error("the shield reaction should be logged")
	}
}

func TestGuardianStaysPutWhenNoHealthierOrWouldDrop(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 5}, &firstPicker{})
	hero := tPlayer("p1", "Hero")
	guardian := tPlayer("p2", "Guardian")
	guardian.DomainAbilities = []models.DomainAbility{shieldAbility()}
	ogre := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{hero, guardian}, []*models.Combatant{ogre}, 0, 0)

	// The ally is still above half health: no reason to step in
	hero.HP = 7
	ce.executeAttack(ogre, hero, state)
	if hero.HP != 6 || guardian.HP != 10 {
		t.Errorf("ally above half: hero HP %d, guardian HP %d; want 6 / 10", hero.HP, guardian.HP)
	}

	// A Major hit (7 vs thresholds 6/9) would down a guardian on 2 HP
	hero.HP, guardian.HP = 1, 2
	heavy := newMockEngine(&mockDiceRoller{damage: 7}, &firstPicker{})
	heavy.executeAttack(ogre, hero, state)
	if guardian.HP != 2 || guardian.Stress != 0 {
		t.Errorf("guardian at 2 HP should not take a Major hit for a 1 HP ally; HP %d Stress %d", guardian.HP, guardian.Stress)
	}
}

func TestFeatureDamageCanBeShielded(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{damage: 5})
	hero := tPlayer("p1", "Hero")
	hero.HP = 4
	guardian := tPlayer("p2", "Guardian")
	guardian.DomainAbilities = []models.DomainAbility{shieldAbility()}
	ogre := tEnemy("e1", "Ogre", 10)
	state := newStateWith([]*models.Combatant{hero, guardian}, []*models.Combatant{ogre}, 0, 0)

	result := DCCheckResult{Success: true}
	effect := models.FeatureEffect{Category: models.FeatureEffectDamage, DiceFormula: "1d6"}
	if err := fr.ResolveEffect(effect, ogre, hero, state, &result); err != nil {
		t.Fatal(err)
	}
	if result.RedirectedTo != guardian || hero.HP != 4 || guardian.HP != 9 {
		t.Errorf("redirected=%v hero HP %d guardian HP %d; want guardian / 4 / 9", result.RedirectedTo != nil, hero.HP, guardian.HP)
	}
}

func TestDrinkingPotions(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{damage: 3}, &firstPicker{})
	hero := tPlayer("p1", "Hero")
	hero.HP, hero.Stress = 4, 5
	hero.HealthPotions, hero.StaminaPotions = 1, 1
	state := newStateWith([]*models.Combatant{hero}, []*models.Combatant{tEnemy("e1", "Ogre", 10)}, 0, 0)

	ce.ExecuteTurnWithDecision(hero, &strategy.ActionDecision{Target: hero, Potion: models.PotionHealth}, state)
	if hero.HP != 7 || hero.HealthPotions != 0 {
		t.Errorf("after Health Potion: HP %d, potions %d; want 7 / 0", hero.HP, hero.HealthPotions)
	}
	ce.ExecuteTurnWithDecision(hero, &strategy.ActionDecision{Target: hero, Potion: models.PotionStamina}, state)
	if hero.Stress != 2 || hero.StaminaPotions != 0 {
		t.Errorf("after Stamina Potion: Stress %d, potions %d; want 2 / 0", hero.Stress, hero.StaminaPotions)
	}
	ce.ExecuteTurnWithDecision(hero, &strategy.ActionDecision{Target: hero, Potion: models.PotionHealth}, state)

	results := []string{}
	for _, a := range state.ActionHistory {
		if a.ActionType == models.ActionUseItem {
			results = append(results, a.Result)
			if a.SpotlightShift {
				t.Error("drinking a potion should not pass the spotlight")
			}
		}
	}
	want := []string{"Drinks a Health Potion: clears 3 HP", "Drinks a Stamina Potion: clears 3 Stress", "No potion left"}
	if strings.Join(results, "|") != strings.Join(want, "|") {
		t.Errorf("log = %v, want %v", results, want)
	}
}

// A heal is aimed at an enemy by the AI but never touches it: the row names whom it reached.
func TestHealRowNamesWhoWasHealed(t *testing.T) {
	ce := newMockEngine(&mockDiceRoller{}, &firstPicker{})
	healer, _, _, state := healParty()
	ogre := state.Enemies[0]
	hands := models.DomainAbility{Name: "Healing Hands", DCType: "fixed", FixedDC: 13,
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectHPHeal, Magnitude: 2, TargetType: models.FeatureTargetAlly}}}
	// The roll is not against the ogre, so the ogre being Vulnerable gives it no advantage
	ogre.AddCondition(&models.ActiveCondition{Type: models.ConditionVulnerable, Duration: models.DurationUntilCleared})

	ce.executeAbility(healer, ogre, &hands, state)
	row := lastRow(state, models.ActionUseAbility)
	if row.Target != "Wounded" || row.TargetHPBefore != 5 || row.TargetHPAfter != 7 || !slices.Equal(row.Notes, []string{"Wounded clears 2 HP"}) {
		t.Errorf("ally heal row: target %q, %d -> %d HP, notes %v; want Wounded 5 -> 7", row.Target, row.TargetHPBefore, row.TargetHPAfter, row.Notes)
	}
	if slices.ContainsFunc(row.RollParts, func(p models.LogPart) bool { return strings.Contains(p.Label, "advantage") }) {
		t.Errorf("heal roll took advantage from the enemy's Vulnerable: %v", row.RollParts)
	}

	healer.HP = 4
	mend := models.DomainAbility{Name: "Mend", Effects: []models.FeatureEffect{{Category: models.FeatureEffectHPHeal, Magnitude: 3}}}
	ce.executeAbility(healer, ogre, &mend, state)
	row = lastRow(state, models.ActionUseAbility)
	if row.Target != "" || !slices.Equal(row.Notes, []string{"Healer clears 3 HP"}) {
		t.Errorf("self heal row: target %q, notes %v; want no target and the healer named", row.Target, row.Notes)
	}
}
