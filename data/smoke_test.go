package data

import (
	"strings"
	"testing"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/simulation"
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
			pc.AttackTrait = 2
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
