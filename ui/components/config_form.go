package components

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/internal/simulation"
	"github.com/skarlso/daggerheart-combat-simulator/internal/strategy"
)

// ConfigForm allows configuring simulation parameters
type ConfigForm struct {
	app.Compo

	Config   *simulation.SimulationConfig
	OnChange func()
}

// Render renders the configuration form
func (cf *ConfigForm) Render() app.UI {
	return app.Div().
		Class("config-form").
		Body(
			cf.renderBasicSettings(),
			cf.renderResourcePools(),
			cf.renderSpotlightSettings(),
			cf.renderStrategySettings(),
		)
}

// renderBasicSettings renders basic simulation settings
func (cf *ConfigForm) renderBasicSettings() app.UI {
	return app.Div().
		Class("form-section").
		Body(
			app.H3().Text("Basic Settings"),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Number of Simulations:"),
					app.Input().
						Type("number").
						Value(cf.Config.RunCount).
						Min(1).
						Max(10000).
						OnInput(func(ctx app.Context, e app.Event) {
							cf.Config.RunCount = ctx.JSSrc().Get("valueAsNumber").Int()
							if cf.OnChange != nil {
								cf.OnChange()
							}
						}),
					app.Span().
						Class("form-help").
						Text("Recommended: 100-1000"),
				),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Max Turns Per Combat:"),
					app.Input().
						Type("number").
						Value(cf.Config.MaxTurnsPerCombat).
						Min(10).
						Max(500).
						OnInput(func(ctx app.Context, e app.Event) {
							cf.Config.MaxTurnsPerCombat = ctx.JSSrc().Get("valueAsNumber").Int()
							if cf.OnChange != nil {
								cf.OnChange()
							}
						}),
					app.Span().
						Class("form-help").
						Text("Prevents infinite loops"),
				),
		)
}

// renderResourcePools renders Hope/Fear pool settings
func (cf *ConfigForm) renderResourcePools() app.UI {
	return app.Div().
		Class("form-section").
		Body(
			app.H3().Text("Resource Pools"),
			app.Div().
				Class("form-row").
				Body(
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Initial Hope Pool:"),
							app.Input().
								Type("number").
								Value(cf.Config.InitialHopePool).
								Min(0).
								Max(20).
								OnInput(func(ctx app.Context, e app.Event) {
									cf.Config.InitialHopePool = ctx.JSSrc().Get("valueAsNumber").Int()
									if cf.OnChange != nil {
										cf.OnChange()
									}
								}),
						),
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Initial Fear Pool:"),
							app.Input().
								Type("number").
								Value(cf.Config.InitialFearPool).
								Min(0).
								Max(20).
								OnInput(func(ctx app.Context, e app.Event) {
									cf.Config.InitialFearPool = ctx.JSSrc().Get("valueAsNumber").Int()
									if cf.OnChange != nil {
										cf.OnChange()
									}
								}),
						),
				),
		)
}

// renderSpotlightSettings renders spotlight token settings
func (cf *ConfigForm) renderSpotlightSettings() app.UI {
	return app.Div().
		Class("form-section").
		Body(
			app.H3().Text("Spotlight System"),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Spotlight Threshold:"),
					app.Input().
						Type("number").
						Value(cf.Config.SpotlightThreshold).
						Min(1).
						Max(10).
						OnInput(func(ctx app.Context, e app.Event) {
							cf.Config.SpotlightThreshold = ctx.JSSrc().Get("valueAsNumber").Int()
							if cf.OnChange != nil {
								cf.OnChange()
							}
						}),
					app.Span().
						Class("form-help").
						Text("Tokens required to force selection"),
				),
		)
}

// renderStrategySettings renders AI strategy settings
func (cf *ConfigForm) renderStrategySettings() app.UI {
	return app.Div().
		Class("form-section").
		Body(
			app.H3().Text("AI Strategies"),
			app.Div().
				Class("strategy-row").
				Body(
					app.Div().
						Class("strategy-column").
						Body(
							app.H4().Text("Player Strategy"),
							cf.renderStrategySelector("player"),
						),
					app.Div().
						Class("strategy-column").
						Body(
							app.H4().Text("Enemy Strategy"),
							cf.renderStrategySelector("enemy"),
						),
				),
		)
}

// renderStrategySelector renders strategy preset selector
func (cf *ConfigForm) renderStrategySelector(side string) app.UI {
	var currentStrategy strategy.ResourceUsageStrategy
	if side == "player" {
		currentStrategy = cf.Config.PlayerStrategy.ResourceUsage
	} else {
		currentStrategy = cf.Config.EnemyStrategy.ResourceUsage
	}

	return app.Div().
		Class("form-group").
		Body(
			app.Label().Text("Preset:"),
			app.Select().
				OnChange(func(ctx app.Context, e app.Event) {
					value := ctx.JSSrc().Get("value").String()
					cf.applyStrategyPreset(side, value)
					if cf.OnChange != nil {
						cf.OnChange()
					}
				}).
				Body(
					cf.renderStrategyOption("greedy", "Greedy (Aggressive)", currentStrategy),
					cf.renderStrategyOption("conservative", "Conservative (Defensive)", currentStrategy),
					cf.renderStrategyOption("desperate", "Desperate (Contextual)", currentStrategy),
					cf.renderStrategyOption("random", "Random (Unpredictable)", currentStrategy),
				),
			app.Span().
				Class("form-help").
				Text(cf.getStrategyDescription(currentStrategy)),
		)
}

// renderStrategyOption renders a strategy option
func (cf *ConfigForm) renderStrategyOption(value, label string, current strategy.ResourceUsageStrategy) app.UI {
	option := app.Option().
		Value(value).
		Text(label)

	if string(current) == value {
		option = option.Selected(true)
	}

	return option
}

// getStrategyDescription returns description for a strategy
func (cf *ConfigForm) getStrategyDescription(strat strategy.ResourceUsageStrategy) string {
	switch strat {
	case strategy.ResourceGreedy:
		return "Uses resources aggressively, focuses fire on weakest targets"
	case strategy.ResourceConservative:
		return "Hoards resources, clears conditions immediately, targets threats"
	case strategy.ResourceDesperate:
		return "Uses resources when low HP, contextual ability usage"
	case strategy.ResourceRandom:
		return "Unpredictable behavior with random decisions"
	default:
		return ""
	}
}

// applyStrategyPreset applies a strategy preset to player or enemy
func (cf *ConfigForm) applyStrategyPreset(side, preset string) {
	var config strategy.StrategyConfig

	switch preset {
	case "greedy":
		config = strategy.DefaultGreedyConfig()
	case "conservative":
		config = strategy.DefaultConservativeConfig()
	case "desperate":
		config = strategy.DefaultDesperateConfig()
	case "random":
		config = strategy.DefaultRandomConfig()
	default:
		config = strategy.DefaultDesperateConfig()
	}

	if side == "player" {
		cf.Config.PlayerStrategy = config
	} else {
		cf.Config.EnemyStrategy = config
	}
}
