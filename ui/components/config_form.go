package components

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/combat-simulator/internal/simulation"
	"github.com/skarlso/combat-simulator/internal/strategy"
)

// ConfigForm allows configuring simulation parameters
type ConfigForm struct {
	app.Compo

	Config *simulation.SimulationConfig
}

// Render renders the configuration form
func (cf *ConfigForm) Render() app.UI {
	return app.Div().Class("config-form").Body(
		section("Basic Settings",
			settingField("Number of Simulations", "Recommended: 100-1000", &cf.Config.RunCount, 1, 10000),
			settingField("Max Turns Per Combat", "Prevents infinite loops", &cf.Config.MaxTurnsPerCombat, 10, 500),
		),
		section("Resource Pools",
			app.Div().Class("form-row").Body(
				settingField("Initial Hope (per PC)", "", &cf.Config.InitialHope, 0, 6),
				settingField("Initial Fear Pool (-1 = 1 per PC)", "", &cf.Config.InitialFearPool, -1, 12),
			),
			settingField("Drink a Health Potion at or below (% HP)", "0 never drinks. Stamina Potions are drunk one Stress short of full.", &cf.Config.PotionHPThreshold, 0, 100),
		),
		section("Spotlight System",
			settingField("Spotlight Threshold", "Tokens required to force selection", &cf.Config.SpotlightThreshold, 1, 10),
			settingField("Area Attack Targets", "How many targets an area effect catches, since ranges are not modeled (0 = everyone)", &cf.Config.AreaTargets, 0, 10),
		),
		section("AI Strategies",
			app.Div().Class("strategy-row").Body(
				app.Div().Class("strategy-column").Body(
					app.H4().Text("Player Strategy"),
					cf.renderStrategySelector(&cf.Config.PlayerStrategy),
				),
				app.Div().Class("strategy-column").Body(
					app.H4().Text("Enemy Strategy"),
					cf.renderStrategySelector(&cf.Config.EnemyStrategy),
				),
			),
		),
		section("Death Moves", cf.renderLastManStanding()),
	)
}

var lastManStandingOptions = []selectOption{
	{string(strategy.DeathMoveBlazeOfGlory), "Blaze of Glory (take it down with them)"},
	{string(strategy.DeathMoveRiskItAll), "Risk It All (roll to stay up)"},
}

func (cf *ConfigForm) renderLastManStanding() app.UI {
	return app.Div().Class("form-group").Body(
		app.Label().Text("Last Man Standing"),
		app.Select().
			OnChange(func(ctx app.Context, e app.Event) {
				cf.Config.LastManStanding = strategy.DeathMove(selectedOption(ctx, lastManStandingOptions))
			}).
			Body(optionElements(lastManStandingOptions, string(cf.Config.LastManStanding))...),
		app.Span().Class("form-help").Text("For the last PC standing when a Blaze of Glory would finish the last adversary. The kill ends the fight as a draw, the roll can still win it. Every other death move is picked from the state of the fight."),
	)
}

func section(title string, body ...app.UI) app.UI {
	return app.Div().Class("form-section").Body(append([]app.UI{app.H3().Text(title)}, body...)...)
}

// settingField is a number input with an optional help line
func settingField(label, help string, target *int, minVal, maxVal int) app.UI {
	return app.Div().Class("form-group").Body(
		app.Label().Text(label),
		app.Input().Type("number").Value(*target).Min(minVal).Max(maxVal).OnInput(handleIntInput(target), fieldScope(target)),
		app.If(help != "", func() app.UI { return app.Span().Class("form-help").Text(help) }),
	)
}

// strategyPresets are keyed by the ResourceUsage they set, which is how the current one is recognised
var strategyPresets = []struct {
	key, label, description string
	config                  func() strategy.StrategyConfig
}{
	{"greedy", "Greedy (Aggressive)", "Uses resources aggressively, focuses fire on weakest targets", strategy.DefaultGreedyConfig},
	{"conservative", "Conservative (Defensive)", "Hoards resources, clears conditions immediately, targets threats", strategy.DefaultConservativeConfig},
	{"desperate", "Desperate (Contextual)", "Uses resources when low HP, contextual ability usage", strategy.DefaultDesperateConfig},
	{"random", "Random (Unpredictable)", "Unpredictable behavior with random decisions", strategy.DefaultRandomConfig},
}

func (cf *ConfigForm) renderStrategySelector(target *strategy.StrategyConfig) app.UI {
	current := string(target.ResourceUsage)
	options := make([]selectOption, len(strategyPresets))
	description := ""
	for i, p := range strategyPresets {
		options[i] = selectOption{p.key, p.label}
		if p.key == current {
			description = p.description
		}
	}

	return app.Div().Class("form-group").Body(
		app.Label().Text("Preset"),
		app.Select().
			OnChange(func(ctx app.Context, e app.Event) {
				cf.applyStrategyPreset(target, ctx.JSSrc().Get("value").String())
				ctx.Update()
			}).
			Body(optionElements(options, current)...),
		app.Span().Class("form-help").Text(description),
	)
}

func (cf *ConfigForm) applyStrategyPreset(target *strategy.StrategyConfig, key string) {
	for _, p := range strategyPresets {
		if p.key == key {
			*target = p.config()
			return
		}
	}
	*target = strategy.DefaultDesperateConfig()
}
