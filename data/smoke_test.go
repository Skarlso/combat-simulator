package data

import (
	"slices"
	"strings"
	"testing"

	"github.com/skarlso/combat-simulator/internal/models"
	"github.com/skarlso/combat-simulator/internal/simulation"
	"github.com/skarlso/combat-simulator/internal/strategy"
)

// Every stat block must run against a sample party without the engine logging an effect error.
func TestEveryAdversarySimulatesCleanly(t *testing.T) {
	registry, err := NewEmbeddedAdversaryRegistry()
	if err != nil {
		t.Fatal(err)
	}
	party := func() []*models.Combatant {
		var pcs []*models.Combatant
		for i, name := range []string{"Kira", "Tomas", "Mara"} {
			pc := models.NewCombatant("pc"+string(rune('1'+i)), name, models.SidePlayer, 5, 12, 2, 7, 6)
			pc.Traits.Strength, pc.WeaponTrait = 2, "strength"
			pc.DamageDice = "1d8+2"
			pc.ArmorSlots = 3
			pcs = append(pcs, pc)
		}
		return pcs
	}
	problems := map[string]string{}
	for _, tpl := range registry.Templates() {
		enemy, err := registry.CloneTemplate(tpl.Name, "smoke")
		if err != nil {
			t.Fatal(err)
		}
		cfg := simulation.DefaultSimulationConfig()
		cfg.RunCount = 8
		cfg.MaxTurnsPerCombat = 60
		cfg.PlayerTemplates = party()
		cfg.EnemyTemplates = []*models.Combatant{enemy}
		cfg.AdversaryRegistry = registry
		result, err := simulation.NewSimulationRunner(cfg).Run()
		if err != nil {
			problems[tpl.Name] = err.Error()
			continue
		}
		for _, outcome := range result.CombatOutcomes {
			for _, action := range outcome.ActionHistory {
				lower := strings.ToLower(action.Result)
				if strings.Contains(lower, "error") || strings.Contains(lower, "failed to") || strings.Contains(lower, "invalid") || strings.Contains(lower, "triggered:") {
					if !strings.Contains(lower, "failed to clear") {
						problems[tpl.Name] = action.FeatureName + ": " + action.Result
					}
				}
			}
		}
	}
	for name, p := range problems {
		t.Errorf("%s: %s", name, p)
	}
}

// The Guardian's library features must run through whole combats: Unstoppable switches on once
// per fight, and neither feature ever logs a failure.
func TestGuardianLibraryFeaturesSimulate(t *testing.T) {
	library, err := LoadAbilityLibrary()
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewEmbeddedAdversaryRegistry()
	if err != nil {
		t.Fatal(err)
	}
	guardian := models.NewCombatant("pc1", "Gerda", models.SidePlayer, 5, 10, 2, 9, 6)
	guardian.Traits.Strength, guardian.WeaponTrait = 2, "strength"
	guardian.DamageDice = "1d10+3"
	guardian.ArmorSlots = 4
	for _, entry := range library {
		if entry.Name == "Unstoppable" || entry.Name == "Frontline Tank" {
			guardian.DomainAbilities = append(guardian.DomainAbilities, entry.AbilityAt(guardian.Level))
		}
	}
	if len(guardian.DomainAbilities) != 2 {
		t.Fatalf("found %d of the 2 Guardian features in the library", len(guardian.DomainAbilities))
	}
	var enemies []*models.Combatant
	for i, name := range []string{"Jagged Knife Lieutenant", "Bladed Guard", "Bladed Guard"} {
		enemy, err := registry.CloneTemplate(name, "e"+string(rune('1'+i)))
		if err != nil {
			t.Fatal(err)
		}
		enemies = append(enemies, enemy)
	}

	cfg := simulation.DefaultSimulationConfig()
	cfg.RunCount = 200
	cfg.RandomSeed = 7
	cfg.PlayerTemplates = []*models.Combatant{guardian}
	cfg.EnemyTemplates = enemies
	cfg.AdversaryRegistry = registry
	result, err := simulation.NewSimulationRunner(cfg).Run()
	if err != nil {
		t.Fatal(err)
	}

	ended, tanked := 0, 0
	for i, outcome := range result.CombatOutcomes {
		started := 0
		for _, action := range outcome.ActionHistory {
			switch {
			case action.FeatureName == "Unstoppable" && action.Result == "Triggered":
				started++
			case action.FeatureName == "Unstoppable" && strings.HasPrefix(action.Result, "Ends"):
				ended++
			case action.FeatureName == "Frontline Tank" && action.Result == "Triggered":
				tanked++
			case action.FeatureName == "Unstoppable" || action.FeatureName == "Frontline Tank":
				t.Errorf("combat %d: %s logged %q", i, action.FeatureName, action.Result)
			}
		}
		if started != 1 {
			t.Errorf("combat %d: became Unstoppable %d times, want once", i, started)
		}
	}
	if ended == 0 || tanked == 0 {
		t.Errorf("over %d combats the die ran out %d times and Frontline Tank fired %d times; both should happen", cfg.RunCount, ended, tanked)
	}
	t.Logf("%d combats: Unstoppable ran out in %d, Frontline Tank fired %d times, players won %d", cfg.RunCount, ended, tanked, result.PlayerWins)
}

// Abilities the smoke fights never use: the AI won't spend a turn on Stress, Restrained or a
// lower Difficulty alone, and Savor the Anguish needs a Severe hit while Stress is marked.
var rarelyUsed = []string{"Book of Norai: Mystic Tether", "Death Grip", "Glyph of Nightfall", "Mass Enrapture",
	"Savor the Anguish", "Shadowbind", "Terrify", "Troublemaker", "Voice of Dread"}

// Every modelled ability must run through whole combats without the engine logging an effect
// error, and must actually come up, on a hard-hitting PC or on a weak one who marks Stress.
func TestEveryModelledAbilitySimulatesCleanly(t *testing.T) {
	library, err := LoadAbilityLibrary()
	if err != nil {
		t.Fatal(err)
	}
	registry, err := NewEmbeddedAdversaryRegistry()
	if err != nil {
		t.Fatal(err)
	}
	var never, problems []string
	for _, entry := range library {
		if !entry.Modelled {
			continue
		}
		fired := 0
		for build := range 2 {
			hero := models.NewCombatant("pc1", "Hero", models.SidePlayer, 5, 11, 2, 8, 6)
			hero.Traits = models.Traits{Agility: 2, Strength: 3, Finesse: 2, Instinct: 1, Presence: 1, Knowledge: 3}
			hero.WeaponTrait, hero.SpellcastTrait = "strength", "knowledge"
			hero.DamageDice = "1d10+3"
			hero.ArmorSlots = 3
			hero.DomainAbilities = []models.DomainAbility{entry.AbilityAt(5)}
			if build == 1 {
				hero.DamageDice = "1d4"
				hero.Proficiency = 1
				hero.DomainAbilities = append(hero.DomainAbilities, models.DomainAbility{
					Name: "Strain", Passive: true, StressCost: 1, RefreshType: "unlimited",
					Trigger: models.FeatureTriggerCondition{Type: models.TriggerOnAttack},
					Effects: []models.FeatureEffect{{Category: models.FeatureEffectApplyAdvantage, Duration: "1_turn"}},
				})
			}
			var pcs = []*models.Combatant{hero}
			for i, name := range []string{"Tomas", "Mara"} {
				pc := models.NewCombatant("pc"+string(rune('2'+i)), name, models.SidePlayer, 5, 11, 2, 7, 6)
				pc.Traits.Strength, pc.WeaponTrait = 2, "strength"
				pc.DamageDice = "1d8+2"
				pc.ArmorSlots = 2
				pcs = append(pcs, pc)
			}
			var enemies []*models.Combatant
			for i, name := range []string{"Jagged Knife Lieutenant", "Bladed Guard", "Archer Guard", "Jagged Knife Hexer", "Bladed Guard"} {
				e, err := registry.CloneTemplate(name, "e"+string(rune('1'+i)))
				if err != nil {
					t.Fatal(err)
				}
				enemies = append(enemies, e)
			}
			cfg := simulation.DefaultSimulationConfig()
			cfg.RunCount = 60
			cfg.RandomSeed = 3
			cfg.InitialHope = 3
			cfg.PlayerStrategy = strategy.DefaultGreedyConfig()
			cfg.PlayerTemplates = pcs
			cfg.EnemyTemplates = enemies
			cfg.AdversaryRegistry = registry
			result, err := simulation.NewSimulationRunner(cfg).Run()
			if err != nil {
				t.Fatal(err)
			}
			for _, outcome := range result.CombatOutcomes {
				for _, a := range outcome.ActionHistory {
					if a.FeatureName != entry.Name || a.Actor != "Hero" {
						continue
					}
					fired++
					lower := strings.ToLower(a.Result)
					if (strings.Contains(lower, "failed:") && !strings.Contains(lower, "failed dc check")) || strings.Contains(lower, "triggered:") || strings.Contains(lower, "invalid") || strings.Contains(lower, "error") {
						problems = append(problems, entry.Name+": "+a.Result)
					}
				}
			}
		}
		standing := entry.Ability.Passive && entry.Ability.Trigger.Type == ""
		if fired == 0 && !standing {
			never = append(never, entry.Name)
		}
	}
	for _, name := range never {
		if !slices.Contains(rarelyUsed, name) {
			t.Errorf("%s never came up in 120 combats", name)
		}
	}
	seen := map[string]bool{}
	for _, p := range problems {
		if !seen[p] {
			seen[p] = true
			t.Error(p)
		}
	}
}
