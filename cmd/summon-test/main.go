package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/skarlso/daggerheart-combat-simulator/internal/engine"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/parser"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)

type UnifiedDecisionMaker struct {
	playerStrategy *strategy.AIDecisionMaker
	enemyStrategy  *strategy.AIDecisionMaker
}

func (udm *UnifiedDecisionMaker) ChooseAction(state *models.CombatState, actor *models.Combatant) *strategy.ActionDecision {
	if actor.Side == models.SidePlayer {
		return udm.playerStrategy.ChooseAction(state, actor)
	}
	return udm.enemyStrategy.ChooseAction(state, actor)
}

func main() {
	testInlinePoolSummoning()
	fmt.Println()
	fmt.Println()
	testRegistrySummoning()
}

func testInlinePoolSummoning() {
	fmt.Println("Testing Inline Pool Summoning (No Registry)")
	fmt.Println("============================================")

	minionTemplate := models.NewAdversary("minion-template", "Summoned Minion", "Tier 1", "Minion", 10, 1, 3, 3, []int{5, 10})
	minionTemplate.DamageDice = "1d4"

	summonFeature := models.AdversaryFeature{
		Name:        "Call Reinforcements",
		Description: "Summons minions from the void",
		Passive:     false,
		FearCost:    1,
		StressCost:  0,
		UsageType:   "once_per_scene",
		MaxUses:     1,
		UsesLeft:    1,
		Effects: []models.FeatureEffect{
			{
				Category:    models.FeatureEffectSummonAllies,
				Magnitude:   0,
				DiceFormula: "1d4",
				SummonPool:  []*models.Combatant{minionTemplate},
			},
		},
	}

	summoner := models.NewAdversary("summoner", "Void Caller", "Tier 2", "Standard", 14, 3, 10, 6, []int{6, 12})
	summoner.DamageDice = "1d8"
	summoner.AdversaryFeatures = append(summoner.AdversaryFeatures, summonFeature)

	players := []*models.Combatant{
		models.NewCombatant("p1", "Hero", models.SidePlayer, 3, 12, 2, 15, 8),
	}
	players[0].DamageDice = "1d6"

	state := models.NewCombatState(players, []*models.Combatant{summoner}, 0, 3, 5)

	rng := rand.New(rand.NewSource(42))
	featureResolver := engine.NewFeatureResolver(rng)

	fmt.Printf("Initial enemies: %d\n", len(state.GetAliveEnemies()))
	fmt.Printf("Fear Pool: %d\n", state.FearPool)
	fmt.Printf("Summoner has %d features\n", len(summoner.AdversaryFeatures))

	feature := summoner.AdversaryFeatures[0]
	fmt.Printf("Feature: %s (Fear: %d, MaxUses: %d, UsesLeft: %d)\n",
		feature.Name, feature.FearCost, feature.MaxUses, feature.UsesLeft)

	canActivate := featureResolver.CanActivateAdversaryFeature(feature, summoner, state)
	fmt.Printf("Can activate: %v\n", canActivate)

	if canActivate {
		err, result := featureResolver.ActivateAdversaryFeature(feature, summoner, nil, state)
		if err != nil {
			fmt.Printf("Activation failed: %v\n", err)
		} else {
			fmt.Printf("Activation succeeded (DC check: %v)\n", result.Success)
			fmt.Printf("Fear Pool after: %d\n", state.FearPool)
			fmt.Printf("Enemies after summon: %d\n", len(state.GetAliveEnemies()))
			for i, enemy := range state.GetAliveEnemies() {
				fmt.Printf("  %d. %s (HP: %d/%d)\n", i+1, enemy.Name, enemy.HP, enemy.HPMax)
			}
		}
	}

	state.FearPool = 3
	summoner.AdversaryFeatures[0].UsesLeft = 1
	fmt.Println("\n--- Second summon activation ---")
	err, _ := featureResolver.ActivateAdversaryFeature(summoner.AdversaryFeatures[0], summoner, nil, state)
	if err != nil {
		fmt.Printf("Second activation failed: %v\n", err)
	} else {
		fmt.Printf("Enemies after second summon: %d\n", len(state.GetAliveEnemies()))
		for i, enemy := range state.GetAliveEnemies() {
			fmt.Printf("  %d. %s (HP: %d/%d)\n", i+1, enemy.Name, enemy.HP, enemy.HPMax)
		}
	}
}

func testRegistrySummoning() {
	fmt.Println("Testing Registry-Based Summoning - Guards, Seize Them!")
	fmt.Println("=======================================================")

	// Load adversary registry
	registry, err := parser.NewAdversaryRegistry(
		"data/feature_mappings.json",
		[]string{"data/daggerheart_adversaries-tier-1.json"},
	)
	if err != nil {
		fmt.Printf("Failed to load adversary registry: %v\n", err)
		return
	}

	fmt.Println("✓ Adversary registry loaded")

	// Check if Bladed Guard template exists
	_, err = registry.GetTemplate("Bladed Guard")
	if err != nil {
		fmt.Printf("✗ Bladed Guard template not found: %v\n", err)
		fmt.Println("\nAvailable templates:")
		for _, name := range registry.ListTemplates() {
			fmt.Printf("  - %s\n", name)
		}
		return
	}
	fmt.Println("✓ Bladed Guard template found")

	// Load Petty Noble adversary (has "Guards, Seize Them!" feature)
	advParser, err := parser.NewAdversaryParser("data/feature_mappings.json")
	if err != nil {
		fmt.Printf("Failed to create adversary parser: %v\n", err)
		return
	}

	adversaries, err := advParser.LoadAdversariesFromFile("data/daggerheart_adversaries-tier-1.json")
	if err != nil {
		fmt.Printf("Failed to load adversaries: %v\n", err)
		return
	}

	var pettyNoble *models.Combatant
	for _, adv := range adversaries {
		if adv.Name == "Petty Noble" {
			pettyNoble = adv
			break
		}
	}

	if pettyNoble == nil {
		fmt.Println("✗ Petty Noble not found in adversaries")
		return
	}

	fmt.Printf("✓ Found Petty Noble with %d features\n", len(pettyNoble.AdversaryFeatures))

	// Find "Guards, Seize Them!" feature
	var summonFeature *models.AdversaryFeature
	for i, feature := range pettyNoble.AdversaryFeatures {
		fmt.Printf("  Feature %d: %s (Fear: %d, %d effects)\n", i+1, feature.Name, feature.FearCost, len(feature.Effects))
		if feature.Name == "Guards, Seize Them!" {
			summonFeature = &pettyNoble.AdversaryFeatures[i]
		}
	}

	if summonFeature == nil {
		fmt.Println("✗ 'Guards, Seize Them!' feature not found or not mapped")
		return
	}

	fmt.Printf("✓ Found summon feature: %s\n", summonFeature.Name)
	fmt.Printf("  Fear Cost: %d\n", summonFeature.FearCost)
	fmt.Printf("  Effects: %d\n", len(summonFeature.Effects))
	if len(summonFeature.Effects) > 0 {
		effect := summonFeature.Effects[0]
		fmt.Printf("  Effect Category: %s\n", effect.Category)
		fmt.Printf("  Summon Type: %s\n", effect.SummonType)
		fmt.Printf("  Dice Formula: %s\n", effect.DiceFormula)
	}

	// Create players (weaker for testing)
	players := []*models.Combatant{
		{
			ID:          "p1",
			Name:        "Fighter",
			Side:        models.SidePlayer,
			HP:          20,
			HPMax:       20,
			Stress:      0,
			StressMax:   10,
			Evasion:     12,
			Proficiency: 1, // Lower proficiency
			ArmorSlots:  1,
			DamageDice:  "1d4", // Lower damage
		},
	}

	// Clone Petty Noble for combat (buffed for testing)
	pettyNobleClone := &models.Combatant{
		ID:                pettyNoble.ID,
		Name:              pettyNoble.Name,
		Side:              models.SideEnemy,
		HP:                25, // Buffed from original
		HPMax:             25,
		Stress:            0,
		StressMax:         pettyNoble.StressMax,
		Evasion:           pettyNoble.Evasion,
		Proficiency:       pettyNoble.Proficiency,
		IsAdversary:       true,
		Tier:              pettyNoble.Tier,
		AdversaryType:     pettyNoble.AdversaryType,
		Difficulty:        pettyNoble.Difficulty,
		ATK:               pettyNoble.ATK,
		DamageDice:        pettyNoble.DamageDice,
		CustomThresholds:  append([]int{}, pettyNoble.CustomThresholds...),
		AdversaryFeatures: append([]models.AdversaryFeature{}, pettyNoble.AdversaryFeatures...),
		Conditions:        make([]*models.ActiveCondition, 0),
	}

	enemies := []*models.Combatant{pettyNobleClone}

	// Create combat state with enough Fear to use summon
	state := models.NewCombatState(players, enemies, 3, 5, 5)

	// Setup combat engine
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	dice := engine.NewStandardDiceRoller(rng)
	spotlight := engine.NewHybridSpotlightSelector(rng, 0.5)
	combatEngine := engine.NewCombatEngine(dice, spotlight)

	// Set adversary registry for summoning
	combatEngine.SetAdversaryRegistry(registry)

	playerStrategy := strategy.NewAIDecisionMaker(rng, strategy.DefaultDesperateConfig())
	enemyStrategy := strategy.NewAIDecisionMaker(rng, strategy.DefaultGreedyConfig())

	decisionMaker := &UnifiedDecisionMaker{
		playerStrategy: playerStrategy,
		enemyStrategy:  enemyStrategy,
	}

	// Run combat
	fmt.Println("\n=== Running Combat ===")
	fmt.Printf("Initial enemies: %d\n", len(state.GetAliveEnemies()))
	fmt.Printf("Fear Pool: %d\n\n", state.FearPool)

	outcome := combatEngine.RunCombat(state, decisionMaker, 50, strategy.ResourceGreedy)

	// Display results
	fmt.Printf("\n=== Combat Result ===\n")
	if outcome.PlayerVictory {
		fmt.Println("Winner: Players")
	} else if outcome.EnemyVictory {
		fmt.Println("Winner: Enemies")
	} else {
		fmt.Println("Winner: Draw")
	}

	fmt.Printf("Turns: %d\n", outcome.Turns)
	fmt.Printf("Final Hope: %d, Fear: %d\n", outcome.FinalHopePool, outcome.FinalFearPool)

	// Check for summons in action history
	fmt.Println("\n=== Summoning Events ===")
	summonCount := 0
	for i, action := range outcome.ActionHistory {
		if action.ActionType == models.ActionUseAbility && action.Result != "" {
			if action.Result[:7] == "Summons" || action.FeatureName == "Guards, Seize Them!" {
				summonCount++
				fmt.Printf("Turn %d: %s - %s\n", i+1, action.Actor, action.Result)
			}
		}
	}

	if summonCount == 0 {
		fmt.Println("No summons occurred during combat")
		fmt.Println("(This may be expected if enemy didn't use the feature or died quickly)")
	} else {
		fmt.Printf("\nTotal summons: %d\n", summonCount)
	}

	// Show final enemy count
	finalEnemyCount := 0
	for _, enemy := range outcome.SurvivingEnemies {
		fmt.Printf("Surviving: %s\n", enemy)
		finalEnemyCount++
	}

	fmt.Println("\n=== Action Log (first 20 actions) ===")
	for i, action := range outcome.ActionHistory {
		if i >= 20 {
			break
		}
		if action.ActionType == models.ActionUseAbility {
			fmt.Printf("Turn %d: %s uses %s -> %s (%s)\n",
				i+1, action.Actor, action.FeatureName, action.Target, action.Result)
		} else {
			fmt.Printf("Turn %d: %s -> %s (%s)\n",
				i+1, action.Actor, action.Target, action.Result)
		}
	}
}
