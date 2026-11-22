package engine

import (
	"testing"

	"github.com/skarlso/combat-simulator/internal/models"
	"github.com/skarlso/combat-simulator/internal/strategy"
)

func passive(name string, trigger models.FeatureTriggerType, filter string, effects ...models.FeatureEffect) models.DomainAbility {
	return models.DomainAbility{
		Name: name, Passive: true, RefreshType: "unlimited",
		Trigger: models.FeatureTriggerCondition{Type: trigger, Filter: filter},
		Effects: effects,
	}
}

// duel is one PC against one ogre (Difficulty 10, thresholds 4/7); the PC is level 3 with
// Evasion 10, thresholds 6/9, 10 HP and 8 Stress slots
func duel(dice *mockDiceRoller, hope int, abilities ...models.DomainAbility) (*CombatEngine, *models.Combatant, *models.Combatant, *models.CombatState) {
	ce := newMockEngine(dice, &firstPicker{})
	hero := tPlayer("p1", "Hero")
	hero.DomainAbilities = abilities
	ogre := tEnemy("e1", "Ogre", 20)
	state := newStateWith([]*models.Combatant{hero}, []*models.Combatant{ogre}, hope, 0)
	return ce, hero, ogre, state
}

func TestDamageRiderPaysToAddToTheRoll(t *testing.T) {
	tusks := passive("Tusks", models.TriggerOnDamageRoll, "", models.FeatureEffect{Category: models.FeatureEffectDamageBonus, DiceFormula: "1d6"})
	tusks.HopeCost = 1

	// 3 from the weapon and 3 from the rider is Major; alone the weapon would be Minor
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 3}, 1, tusks)
	ce.executeAttack(hero, ogre, state)
	if ogre.HP != 18 || hero.Hope != 1 {
		t.Errorf("ogre HP %d, Hope %d; want 18 and 1 (one spent, one gained from the roll)", ogre.HP, hero.Hope)
	}
	if got := resultsFor(state, "Tusks"); len(got) != 1 || got[0] != "Triggered" {
		t.Errorf("Tusks log = %v, want one Triggered row", got)
	}

	// No Hope, and a roll with Fear brings none: no rider
	ce, hero, ogre, state = duel(&mockDiceRoller{damage: 3, dualityQueue: []DualityRollResult{fearDuality(20)}}, 0, tusks)
	ce.executeAttack(hero, ogre, state)
	if ogre.HP != 19 {
		t.Errorf("without Hope: ogre HP %d, want 19", ogre.HP)
	}

	// A miss rolls no damage, so nothing is paid
	ce, hero, ogre, state = duel(&mockDiceRoller{damage: 3, dualityQueue: []DualityRollResult{hopeDuality(5)}}, 1, tusks)
	ce.executeAttack(hero, ogre, state)
	if ogre.HP != 20 || hero.Hope != 2 {
		t.Errorf("on a miss: ogre HP %d, Hope %d; want 20 and 2", ogre.HP, hero.Hope)
	}
}

func TestDamageMultiplierDoublesOneRoll(t *testing.T) {
	firstStrike := passive("First Strike", models.TriggerOnDamageRoll, "", models.FeatureEffect{Category: models.FeatureEffectDamageMultiplier, Magnitude: 2})
	firstStrike.MaxUses, firstStrike.UsesLeft = 1, 1
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 3}, 0, firstStrike)

	ce.executeAttack(hero, ogre, state)
	ce.executeAttack(hero, ogre, state)
	// 6 is Major (2 HP), then a plain 3 is Minor (1 HP)
	if ogre.HP != 17 {
		t.Errorf("ogre HP %d, want 17", ogre.HP)
	}
}

func TestStandingDamageBonusScalesAndRespectsDamageType(t *testing.T) {
	training := passive("Combat Training", "", "", models.FeatureEffect{Category: models.FeatureEffectDamageBonus, Magnitude: 1, ScaleBy: "level", DamageType: "physical"})
	basher := passive("Body Basher", "", "", models.FeatureEffect{Category: models.FeatureEffectDamageBonus, Magnitude: 2, ScaleBy: "strength"})
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 1}, 0, training, basher)
	hero.Traits.Strength = 2

	// 1 + level 3 + twice Strength 2 = 8: Severe
	ce.executeAttack(hero, ogre, state)
	if ogre.HP != 17 {
		t.Fatalf("ogre HP %d, want 17", ogre.HP)
	}

	// A magic spell gets Body Basher's 4 but not the physical-only 3: 1 + 4 = 5, Major
	bolt := models.FeatureEffect{Category: models.FeatureEffectDamage, DiceFormula: "1d8", DamageType: "magic"}
	if err := ce.featureResolver.ResolveEffect(bolt, hero, ogre, state, nil); err != nil {
		t.Fatal(err)
	}
	if ogre.HP != 15 {
		t.Errorf("ogre HP %d after the spell, want 15", ogre.HP)
	}
}

func TestStandingBonusFollowsItsFilter(t *testing.T) {
	adrenaline := passive("Adrenaline", "", "if_vulnerable", models.FeatureEffect{Category: models.FeatureEffectDamageBonus, Magnitude: 1, ScaleBy: "level"})
	_, hero, _, _ := duel(&mockDiceRoller{}, 0, adrenaline)

	if got := models.SumParts(hero.DamageBonusParts(models.DamagePhysical, nil)); got != 0 {
		t.Errorf("bonus = %d while not Vulnerable, want 0", got)
	}
	hero.AddCondition(&models.ActiveCondition{Type: models.ConditionVulnerable, Duration: models.DurationUntilCleared})
	if got := models.SumParts(hero.DamageBonusParts(models.DamagePhysical, nil)); got != 3 {
		t.Errorf("bonus = %d while Vulnerable, want the level, 3", got)
	}
}

func TestProficiencyBonusRiderRollsOneWeaponDie(t *testing.T) {
	predator := passive("Ruthless Predator", models.TriggerOnDamageRoll, "", models.FeatureEffect{Category: models.FeatureEffectDamageBonus, StandardAttack: true})
	predator.StressCost = 1
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 4}, 0, predator)
	hero.DamageDice = "1d10+2"

	// The mock rolls 4 for the weapon dice and 4 for the extra die: 4 + 2 + 4 = 10
	ce.executeAttack(hero, ogre, state)
	attack := state.ActionHistory[len(state.ActionHistory)-1]
	if attack.DamageRoll != 10 || hero.Stress != 1 {
		t.Errorf("rolled %d with Stress %d, want 10 and 1", attack.DamageRoll, hero.Stress)
	}
}

func TestSevereHitIsSoftenedForAStress(t *testing.T) {
	getBackUp := passive("Get Back Up", models.TriggerOnDamageIncoming, "if_severe", models.FeatureEffect{Category: models.FeatureEffectSeverityReduction, Magnitude: 1})
	getBackUp.StressCost = 1

	tests := []struct {
		name               string
		damage, armor      int
		stress             int
		wantHP, wantStress int
	}{
		{"Severe is taken as Major", 9, 0, 0, 8, 1},
		{"Major is left alone", 6, 0, 0, 8, 0},
		{"Severe with armor: Major, then Minor", 9, 1, 0, 9, 1},
		{"the last Stress slot is never marked", 9, 0, 7, 7, 7},
	}
	for _, tt := range tests {
		ce, hero, ogre, state := duel(&mockDiceRoller{damage: tt.damage}, 0, getBackUp)
		hero.ArmorSlots, hero.Stress = tt.armor, tt.stress
		ce.executeAttack(ogre, hero, state)
		if hero.HP != tt.wantHP || hero.Stress != tt.wantStress {
			t.Errorf("%s: HP %d, Stress %d; want %d / %d", tt.name, hero.HP, hero.Stress, tt.wantHP, tt.wantStress)
		}
		if used := len(resultsFor(state, "Get Back Up")); used != tt.wantStress-tt.stress {
			t.Errorf("%s: logged %d times, want %d", tt.name, used, tt.wantStress-tt.stress)
		}
	}
}

func TestReactionIsNotPaidWhenArmorAlreadySavesTheHitPoint(t *testing.T) {
	thickSkin := passive("Thick Skin", models.TriggerOnDamageIncoming, "if_minor", models.FeatureEffect{Category: models.FeatureEffectSeverityReduction, Magnitude: 1})
	thickSkin.StressCost = 2

	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 3}, 0, thickSkin)
	hero.ArmorSlots = 1
	ce.executeAttack(ogre, hero, state)
	if hero.Stress != 0 || hero.ArmorSlots != 0 || hero.HP != 10 {
		t.Errorf("with armor: Stress %d, armor %d, HP %d; want 0 / 0 / 10", hero.Stress, hero.ArmorSlots, hero.HP)
	}
	// Out of armor, the Minor hit costs 2 Stress instead of the Hit Point
	ce.executeAttack(ogre, hero, state)
	if hero.Stress != 2 || hero.HP != 10 {
		t.Errorf("without armor: Stress %d, HP %d; want 2 / 10", hero.Stress, hero.HP)
	}
}

func TestIncomingDamageIsHalvedOrReducedForHope(t *testing.T) {
	fortitude := passive("Increased Fortitude", models.TriggerOnDamageIncoming, "if_major_or_worse",
		models.FeatureEffect{Category: models.FeatureEffectDamageResistance, Magnitude: 50, DamageType: "physical"})
	fortitude.HopeCost = 3
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 9}, 3, fortitude)

	// 9 halves to 5: Minor instead of Severe
	ce.executeAttack(ogre, hero, state)
	if hero.HP != 9 || hero.Hope != 0 {
		t.Errorf("HP %d, Hope %d; want 9 / 0", hero.HP, hero.Hope)
	}
	// A magic hit is not what it protects against
	hero.Hope = 3
	hero.TakeDamage(9, models.DamageMagic, nil)
	if hero.HP != 6 || hero.Hope != 3 {
		t.Errorf("after magic damage: HP %d, Hope %d; want 6 / 3", hero.HP, hero.Hope)
	}

	ward := passive("Rune Ward", models.TriggerOnDamageIncoming, "if_major_or_worse", models.FeatureEffect{Category: models.FeatureEffectDamageReduction, DiceFormula: "1d8"})
	ward.HopeCost = 1
	ce, hero, ogre, state = duel(&mockDiceRoller{damage: 6}, 1, ward)
	// The mock rolls 6 on the Ward Die too: nothing gets through
	ce.executeAttack(ogre, hero, state)
	if hero.HP != 10 || hero.Hope != 0 {
		t.Errorf("warded: HP %d, Hope %d; want 10 / 0", hero.HP, hero.Hope)
	}
}

func TestReactiveEvasionTurnsAHitIntoAMiss(t *testing.T) {
	seeIt := passive("I See It Coming", models.TriggerOnAttacked, "", models.FeatureEffect{Category: models.FeatureEffectEvasionBonus, DiceFormula: "1d4"})
	seeIt.StressCost = 1

	tests := []struct {
		name            string
		d20, bonusRoll  int
		wantHP, wantStr int
	}{
		// ATK is +5 and Evasion 10
		{"beaten by 2, rolls 3: dodged", 6, 3, 10, 1},
		{"beaten by 2, rolls 1: paid and still hit", 6, 1, 9, 1},
		{"beaten by 10: a d4 cannot help, nothing paid", 15, 3, 9, 0},
		{"a miss needs no reaction", 2, 3, 10, 0},
	}
	for _, tt := range tests {
		ce, hero, ogre, state := duel(&mockDiceRoller{damage: tt.bonusRoll, d20: tt.d20}, 0, seeIt)
		ce.executeAttack(ogre, hero, state)
		if hero.HP != tt.wantHP || hero.Stress != tt.wantStr {
			t.Errorf("%s: HP %d, Stress %d; want %d / %d", tt.name, hero.HP, hero.Stress, tt.wantHP, tt.wantStr)
		}
	}
}

func TestMissesTriggerRiposteAndGlancingBlow(t *testing.T) {
	riposte := passive("Rapid Riposte", models.TriggerOnEvaded, "", models.FeatureEffect{Category: models.FeatureEffectDamage, StandardAttack: true, DamageType: "physical"})
	riposte.StressCost = 1
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 5, d20: 2}, 0, riposte)
	ce.executeAttack(ogre, hero, state)
	if ogre.HP != 18 || hero.Stress != 1 || hero.HP != 10 {
		t.Errorf("riposte: ogre HP %d, Stress %d, hero HP %d; want 18 / 1 / 10", ogre.HP, hero.Stress, hero.HP)
	}

	glancing := passive("Glancing Blow", models.TriggerOnMiss, "", models.FeatureEffect{Category: models.FeatureEffectDamage, StandardAttack: true, HalfDamage: true, DamageType: "physical"})
	glancing.StressCost = 1
	ce, hero, ogre, state = duel(&mockDiceRoller{damage: 9, dualityQueue: []DualityRollResult{hopeDuality(5)}}, 0, glancing)
	ce.executeAttack(hero, ogre, state)
	// Half of 9 is 4: Major
	if ogre.HP != 18 || hero.Stress != 1 {
		t.Errorf("glancing blow: ogre HP %d, Stress %d; want 18 / 1", ogre.HP, hero.Stress)
	}
}

func TestSevereDamageDealtForcesAnExtraHitPoint(t *testing.T) {
	deathStrike := passive("Death Strike", models.TriggerOnDamageDealt, "if_dealt_severe", models.FeatureEffect{Category: models.FeatureEffectMarkHP, Magnitude: 1})
	deathStrike.StressCost = 1

	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 7}, 0, deathStrike)
	ce.executeAttack(hero, ogre, state)
	if ogre.HP != 16 || hero.Stress != 1 {
		t.Errorf("Severe: ogre HP %d, Stress %d; want 16 / 1", ogre.HP, hero.Stress)
	}

	ce, hero, ogre, state = duel(&mockDiceRoller{damage: 5}, 0, deathStrike)
	ce.executeAttack(hero, ogre, state)
	if ogre.HP != 18 || hero.Stress != 0 {
		t.Errorf("Major: ogre HP %d, Stress %d; want 18 / 0", ogre.HP, hero.Stress)
	}
}

func TestTakingFearNeedsFearInThePool(t *testing.T) {
	direStrike := passive("Dire Strike", models.TriggerOnDamageDealt, "", models.FeatureEffect{Category: models.FeatureEffectFearGenerate, Magnitude: -1})
	direStrike.HopeCost = 1
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 5}, 1, direStrike)

	ce.executeAttack(hero, ogre, state)
	if state.FearPool != 0 || hero.Hope != 2 {
		t.Errorf("empty pool: Fear %d, Hope %d; want 0 and Hope kept at 2", state.FearPool, hero.Hope)
	}
	state.FearPool = 2
	ce.executeAttack(hero, ogre, state)
	if state.FearPool != 1 || hero.Hope != 2 {
		t.Errorf("Fear %d, Hope %d; want 1 and 2 (one spent, one gained)", state.FearPool, hero.Hope)
	}
}

func TestRecklessGivesAdvantageForTheTurn(t *testing.T) {
	reckless := passive("Reckless", models.TriggerOnAttack, "", models.FeatureEffect{Category: models.FeatureEffectApplyAdvantage, Duration: "1_turn"})
	reckless.StressCost = 1
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 1}, 0, reckless)

	ce.featureResolver.ProcessTriggeredFeatures(hero, ogre, state, models.TriggerOnAttack, false)
	if !hero.HasCondition(models.ConditionAdvantage) || hero.Stress != 1 {
		t.Fatalf("advantage %v, Stress %d; want advantage for 1 Stress", hero.HasCondition(models.ConditionAdvantage), hero.Stress)
	}
	ce.ExecuteTurnWithDecision(hero, &strategy.ActionDecision{Target: ogre}, state)
	if hero.HasCondition(models.ConditionAdvantage) {
		t.Error("advantage should end with the turn")
	}
}

func TestDefensePenaltyAndDisadvantageLandOnTheTarget(t *testing.T) {
	_, hero, ogre, state := duel(&mockDiceRoller{}, 0)
	fr := newTestFR(&mockDiceRoller{})
	hero.Traits.Knowledge = 3

	for _, effect := range []models.FeatureEffect{
		{Category: models.FeatureEffectDefensePenalty, Magnitude: 1, ScaleBy: "knowledge"},
		{Category: models.FeatureEffectDefensePenalty, Magnitude: 2},
		{Category: models.FeatureEffectApplyDisadvantage, Duration: "1_turn"},
	} {
		if err := fr.ResolveEffect(effect, hero, ogre, state, nil); err != nil {
			t.Fatal(err)
		}
	}
	if ogre.GetDefense() != 7 {
		t.Errorf("Difficulty = %d, want 7: the larger penalty of 3 stands, they do not stack", ogre.GetDefense())
	}
	if !ogre.HasCondition(models.ConditionDisadvantage) {
		t.Error("the ogre should attack with disadvantage")
	}
}

func TestStandingEvasionAndAttackBonuses(t *testing.T) {
	shield := passive("Conjure Shield", "", "if_hope_2", models.FeatureEffect{Category: models.FeatureEffectEvasionBonus, Magnitude: 1, ScaleBy: "proficiency"})
	noMercy := passive("No Mercy", "", "", models.FeatureEffect{Category: models.FeatureEffectAttackBonus, Magnitude: 1})
	_, hero, _, _ := duel(&mockDiceRoller{}, 2, shield, noMercy)

	if hero.GetDefense() != 12 || models.SumParts(hero.AttackRollParts(nil)) != 1 {
		t.Errorf("Evasion %d, attack bonus %d; want 12 and 1", hero.GetDefense(), models.SumParts(hero.AttackRollParts(nil)))
	}
	hero.Hope = 1
	if hero.GetDefense() != 10 {
		t.Errorf("Evasion %d with 1 Hope, want 10", hero.GetDefense())
	}
}

func TestWhirlwindHitsOnlyTheOtherTargets(t *testing.T) {
	whirlwind := passive("Whirlwind", models.TriggerOnHit, "",
		models.FeatureEffect{Category: models.FeatureEffectAreaDamage, StandardAttack: true, HalfDamage: true, OthersOnly: true, DamageType: "physical", TargetType: models.FeatureTargetAOE})
	whirlwind.HopeCost = 1
	ce := newMockEngine(&mockDiceRoller{damage: 8}, &firstPicker{})
	hero := tPlayer("p1", "Hero")
	hero.DomainAbilities = []models.DomainAbility{whirlwind}
	first, second := tEnemy("e1", "Ogre", 20), tEnemy("e2", "Goblin", 20)
	state := newStateWith([]*models.Combatant{hero}, []*models.Combatant{first, second}, 1, 0)

	ce.executeAttack(hero, first, state)
	// 8 is Severe on the target; the other takes half, 4: Major
	if first.HP != 17 || second.HP != 18 {
		t.Errorf("target HP %d, other HP %d; want 17 / 18", first.HP, second.HP)
	}

	// The same roll of 20 does not beat a Difficulty of 25
	second.Difficulty = 25
	hero.Hope = 1
	ce.executeAttack(hero, first, state)
	if second.HP != 18 {
		t.Errorf("other HP %d, want 18: the reused roll misses it", second.HP)
	}
}

func TestChargedTokensAreSpentByTheNextDamageRoll(t *testing.T) {
	upstaged := passive("Never Upstaged", models.TriggerOnDamageTaken, "",
		models.FeatureEffect{Category: models.FeatureEffectTokenPlace, Magnitude: 1},
		models.FeatureEffect{Category: models.FeatureEffectDamageBonus, Magnitude: 5, PerToken: true},
		models.FeatureEffect{Category: models.FeatureEffectTokenOnDamage, Magnitude: -99})
	upstaged.StressCost, upstaged.UsesTokens = 1, true
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 2}, 0, upstaged)

	ce.executeAttack(ogre, hero, state)
	if got := hero.DomainAbilities[0].TokensHeld; got != 1 || hero.Stress != 1 {
		t.Fatalf("tokens %d, Stress %d after being hit; want 1 / 1", got, hero.Stress)
	}
	// 2 + 5 is Severe, and the token is gone afterwards
	ce.executeAttack(hero, ogre, state)
	if ogre.HP != 17 || hero.DomainAbilities[0].TokensHeld != 0 {
		t.Errorf("ogre HP %d, tokens %d; want 17 / 0", ogre.HP, hero.DomainAbilities[0].TokensHeld)
	}
	ce.executeAttack(hero, ogre, state)
	if ogre.HP != 16 {
		t.Errorf("ogre HP %d after an uncharged hit, want 16", ogre.HP)
	}
}

func TestOffensivePassivesStopAtHalfTheStressTrack(t *testing.T) {
	kick := passive("Kick", models.TriggerOnDamageRoll, "", models.FeatureEffect{Category: models.FeatureEffectDamageBonus, Magnitude: 2})
	kick.StressCost = 1
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 1}, 0, kick)

	// 8 Stress slots: four hits pay, the fifth does not
	for range 5 {
		ce.executeAttack(hero, ogre, state)
	}
	if hero.Stress != 4 {
		t.Errorf("Stress = %d after five hits, want 4", hero.Stress)
	}
}

func TestPassiveIsNotPaidForWhatTheTargetAlreadySuffers(t *testing.T) {
	squareUp := passive("Square Up", models.TriggerOnAttack, "", models.FeatureEffect{Category: models.FeatureEffectApplyVulnerable, Duration: "until_cleared"})
	squareUp.HopeCost = 3
	hex := passive("Hex", models.TriggerOnAttack, "", models.FeatureEffect{Category: models.FeatureEffectDefensePenalty, Magnitude: 1, ScaleBy: "tier"})
	hex.StressCost = 1
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 1}, 6, squareUp, hex)

	ce.executeAttack(hero, ogre, state)
	ce.executeAttack(hero, ogre, state)
	// Level 3 is tier 2. Hope: 6 - 3 once, +1 for each of the two rolls with Hope
	if !ogre.HasCondition(models.ConditionVulnerable) || ogre.GetDefense() != 8 || hero.Hope != 5 || hero.Stress != 1 {
		t.Errorf("Vulnerable %v, Difficulty %d, Hope %d, Stress %d; want true / 8 / 5 / 1",
			ogre.HasCondition(models.ConditionVulnerable), ogre.GetDefense(), hero.Hope, hero.Stress)
	}
}

func TestRestoreOnlyPassivesWaitUntilTheyAreNeeded(t *testing.T) {
	determination := passive("Deadly Determination", models.TriggerStartTurn, "", models.FeatureEffect{Category: models.FeatureEffectStressClear, Magnitude: 2, TargetType: models.FeatureTargetSelf})
	determination.HopeCost = 3
	speech := passive("Rousing Speech", models.TriggerStartTurn, "", models.FeatureEffect{Category: models.FeatureEffectStressClear, Magnitude: 2, TargetType: models.FeatureTargetAllies})
	speech.MaxUses, speech.UsesLeft = 1, 1
	ce, hero, _, state := duel(&mockDiceRoller{damage: 1}, 3, determination, speech)

	hero.Stress = 1
	ce.featureResolver.ProcessTriggeredFeatures(hero, nil, state, models.TriggerStartTurn, false)
	// The speech helps anyone with Stress; 3 Hope for 2 Stress waits until 2 are marked
	if hero.Stress != 0 || hero.Hope != 3 || hero.DomainAbilities[1].UsesLeft != 0 {
		t.Fatalf("Stress %d, Hope %d, speech uses %d; want 0 / 3 / 0", hero.Stress, hero.Hope, hero.DomainAbilities[1].UsesLeft)
	}
	hero.Stress = 2
	ce.featureResolver.ProcessTriggeredFeatures(hero, nil, state, models.TriggerStartTurn, false)
	if hero.Stress != 0 || hero.Hope != 0 {
		t.Errorf("Stress %d, Hope %d; want 0 / 0", hero.Stress, hero.Hope)
	}
}

func TestExtraArmorSlotReactionAndCompoundFilter(t *testing.T) {
	ironWill := passive("Iron Will", models.TriggerOnDamageIncoming, "",
		models.FeatureEffect{Category: models.FeatureEffectSeverityReduction, Magnitude: 1, DamageType: "physical"},
		models.FeatureEffect{Category: models.FeatureEffectArmorDamage, Magnitude: 1})
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 9}, 0, ironWill)
	hero.ArmorSlots = 3

	// Severe: one slot makes it Major, the second Minor
	ce.executeAttack(ogre, hero, state)
	if hero.HP != 9 || hero.ArmorSlots != 1 {
		t.Fatalf("HP %d, armor %d; want 9 / 1", hero.HP, hero.ArmorSlots)
	}
	// One slot left: no second slot to mark
	ce.executeAttack(ogre, hero, state)
	if hero.HP != 7 || hero.ArmorSlots != 0 {
		t.Errorf("HP %d, armor %d; want 7 / 0", hero.HP, hero.ArmorSlots)
	}

	brink := passive("On the Brink", models.TriggerOnDamageIncoming, "if_minor+if_low_hp", models.FeatureEffect{Category: models.FeatureEffectSeverityReduction, Magnitude: 1})
	_, hero, _, _ = duel(&mockDiceRoller{}, 0, brink)
	hero.TakeDamage(3, models.DamagePhysical, nil)
	hero.HP = 2
	hero.TakeDamage(3, models.DamagePhysical, nil)
	hero.TakeDamage(6, models.DamagePhysical, nil)
	if hero.HP != 0 {
		t.Errorf("HP %d, want 0: Minor ignored only at 2 HP, Major still lands", hero.HP)
	}
}

func TestTokensAndDiceScaleWithTheActor(t *testing.T) {
	fr := newTestFR(&mockDiceRoller{})
	_, hero, ogre, state := duel(&mockDiceRoller{}, 0)
	hero.Traits.Knowledge, hero.SpellcastTrait = 3, "knowledge"

	// "d8+3 using your Spellcast trait"
	if got := hero.ScaleDice("1d8+3", "spellcast"); got != "3d8+3" {
		t.Errorf("scaled formula = %q, want 3d8+3", got)
	}

	tokens := models.DomainAbility{Name: "Thorns", UsesTokens: true, Effects: []models.FeatureEffect{{Category: models.FeatureEffectTokenPlace, Magnitude: 1, ScaleBy: "spellcast"}}}
	if _, err := fr.ActivateDomainAbility(&tokens, hero, ogre, state); err != nil || tokens.TokensHeld != 3 {
		t.Errorf("tokens = %d (err %v), want 3", tokens.TokensHeld, err)
	}
}

func TestOtherTargetsEffectNeedsAnotherTarget(t *testing.T) {
	whirlwind := passive("Whirlwind", models.TriggerOnHit, "",
		models.FeatureEffect{Category: models.FeatureEffectAreaDamage, StandardAttack: true, OthersOnly: true, TargetType: models.FeatureTargetAOE})
	whirlwind.HopeCost = 1
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 2}, 1, whirlwind)

	ce.executeAttack(hero, ogre, state)
	if hero.Hope != 2 || len(resultsFor(state, "Whirlwind")) != 0 {
		t.Errorf("Hope %d, Whirlwind log %v; want Hope kept with a lone enemy", hero.Hope, resultsFor(state, "Whirlwind"))
	}
}

func TestAdversaryReactsToSevereDamageOnly(t *testing.T) {
	acidBath := models.AdversaryFeature{Name: "Acid Bath", Passive: true,
		Trigger: models.FeatureTriggerCondition{Type: models.TriggerOnDamageTaken, Filter: "if_took_severe"},
		Effects: []models.FeatureEffect{{Category: models.FeatureEffectAreaDamage, DiceFormula: "1d10", DamageType: "physical"}}}

	// The mock rolls the same number for the attack and the splash. 5 is Major on the ogre: no splash
	ce, hero, ogre, state := duel(&mockDiceRoller{damage: 5}, 0)
	ogre.AdversaryFeatures = []models.AdversaryFeature{acidBath}
	ce.executeAttack(hero, ogre, state)
	if hero.HP != 10 || len(resultsFor(state, "Acid Bath")) != 0 {
		t.Errorf("Major hit: hero HP %d, Acid Bath log %v; want 10 and nothing", hero.HP, resultsFor(state, "Acid Bath"))
	}

	// 7 is Severe: the splash's 7 is Major on the hero
	ce, hero, ogre, state = duel(&mockDiceRoller{damage: 7}, 0)
	ogre.AdversaryFeatures = []models.AdversaryFeature{acidBath}
	ce.executeAttack(hero, ogre, state)
	if hero.HP != 8 || len(resultsFor(state, "Acid Bath")) != 1 {
		t.Errorf("Severe hit: hero HP %d, Acid Bath log %v; want 8 and one row", hero.HP, resultsFor(state, "Acid Bath"))
	}
}
