package pages

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/data"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/parser"
	"github.com/skarlso/daggerheart-combat-simulator/internal/simulation"
	"github.com/skarlso/daggerheart-combat-simulator/ui/components"
)

// SimulatorPage is the main page for running simulations
type SimulatorPage struct {
	app.Compo

	// State
	players           []*models.Combatant
	enemies           []*models.Combatant
	config            simulation.SimulationConfig
	result            *simulation.SimulationResult
	isRunning         bool
	errorMessage      string
	currentTab        string
	initialized       bool
	adversaryRegistry *parser.AdversaryRegistry
}

// OnNav initializes the page on first navigation
func (p *SimulatorPage) OnNav(ctx app.Context) {
	if !p.initialized {
		p.currentTab = "players"
		p.config = simulation.DefaultSimulationConfig()
		p.players = make([]*models.Combatant, 0)
		p.enemies = make([]*models.Combatant, 0)

		// Initialize adversary registry for summoning support
		registry, err := data.NewEmbeddedAdversaryRegistry()
		if err != nil {
			app.Log("Warning: Failed to initialize adversary registry:", err)
			app.Log("Summoning features will not work in this session")
		} else {
			p.adversaryRegistry = registry
			app.Log("Adversary registry initialized successfully")
		}

		p.initialized = true
	}
}

// Render renders the simulator page
func (p *SimulatorPage) Render() app.UI {
	// Initialize state on first render if needed
	if p.currentTab == "" {
		p.currentTab = "players"
	}
	if p.players == nil {
		p.players = make([]*models.Combatant, 0)
	}
	if p.enemies == nil {
		p.enemies = make([]*models.Combatant, 0)
	}
	if p.config.RunCount == 0 {
		p.config = simulation.DefaultSimulationConfig()
	}

	return app.Div().
		Class("simulator-page").
		Body(
			p.renderTabs(),
			p.renderTabContent(),
			p.renderActionButtons(),
			p.renderResults(),
		)
}

// renderTabs renders the tab navigation
func (p *SimulatorPage) renderTabs() app.UI {
	return app.Div().
		Class("tabs").
		Body(
			p.renderTab("players", "Players"),
			p.renderTab("enemies", "Enemies"),
			p.renderTab("config", "Configuration"),
		)
}

// renderTab renders a single tab button
func (p *SimulatorPage) renderTab(id, label string) app.UI {
	class := "tab"
	if p.currentTab == id {
		class += " active"
	}

	return app.Button().
		Class(class).
		Text(label).
		OnClick(func(ctx app.Context, e app.Event) {
			p.currentTab = id
			ctx.Update()
		})
}

// renderTabContent renders the active tab's content
func (p *SimulatorPage) renderTabContent() app.UI {
	switch p.currentTab {
	case "players":
		return app.Div().
			Class("tab-content").
			Body(
				app.H2().Text("Player Characters"),
				&components.CombatantList{
					Title:      "Players",
					Combatants: p.players,
					Side:       models.SidePlayer,
					OnAdd: func(ctx app.Context, c *models.Combatant) {
						p.players = append(p.players, c)
						ctx.Update()
					},
					OnRemove: func(ctx app.Context, idx int) {
						if idx >= 0 && idx < len(p.players) {
							p.players = append(p.players[:idx], p.players[idx+1:]...)
							ctx.Update()
						}
					},
				},
			)

	case "enemies":
		return app.Div().
			Class("tab-content").
			Body(
				app.H2().Text("Adversaries"),
				&components.AdversaryList{
					Title:       "Adversaries",
					Adversaries: p.enemies,
					OnAdd: func(ctx app.Context, c *models.Combatant) {
						p.enemies = append(p.enemies, c)
						ctx.Update()
					},
					OnRemove: func(ctx app.Context, idx int) {
						if idx >= 0 && idx < len(p.enemies) {
							p.enemies = append(p.enemies[:idx], p.enemies[idx+1:]...)
							ctx.Update()
						}
					},
				},
			)

	case "config":
		return app.Div().
			Class("tab-content").
			Body(
				app.H2().Text("Simulation Configuration"),
				&components.ConfigForm{
					Config: &p.config,
					OnChange: func() {
					},
				},
			)

	default:
		return app.Div().Text("Unknown tab")
	}
}

// renderActionButtons renders the run/clear buttons
func (p *SimulatorPage) renderActionButtons() app.UI {
	return app.Div().
		Class("action-buttons").
		Body(
			app.Button().
				Class("btn btn-primary").
				Text("Run Simulation").
				Disabled(p.isRunning || len(p.players) == 0 || len(p.enemies) == 0).
				OnClick(func(ctx app.Context, e app.Event) {
					p.runSimulation(ctx)
				}),
			app.Button().
				Class("btn btn-secondary").
				Text("Clear Results").
				Disabled(p.result == nil).
				OnClick(func(ctx app.Context, e app.Event) {
					p.result = nil
					p.errorMessage = ""
					ctx.Update()
				}),
			app.If(p.isRunning, func() app.UI {
				return app.Span().
					Class("status-text").
					Text("Running simulations...")
			}),
			app.If(p.errorMessage != "", func() app.UI {
				return app.Div().
					Class("error-message").
					Text(p.errorMessage)
			}),
		)
}

// renderResults renders the simulation results
func (p *SimulatorPage) renderResults() app.UI {
	if p.result == nil {
		return app.Div()
	}

	return app.Div().
		Class("results-section").
		Body(
			app.H2().Text("Simulation Results"),
			&components.ResultsSummary{
				Result: p.result,
			},
		)
}

// runSimulation executes the simulation in a goroutine
func (p *SimulatorPage) runSimulation(ctx app.Context) {
	p.isRunning = true
	p.errorMessage = ""
	p.result = nil

	// Run in goroutine to avoid blocking UI
	go func() {
		defer func() {
			p.isRunning = false
			ctx.Dispatch(func(ctx app.Context) {
			})
		}()

		// Configure simulation
		p.config.PlayerTemplates = p.players
		p.config.EnemyTemplates = p.enemies
		p.config.AdversaryRegistry = p.adversaryRegistry

		// Run simulation
		runner := simulation.NewSimulationRunner(p.config)
		result, err := runner.Run()

		ctx.Dispatch(func(ctx app.Context) {
			if err != nil {
				p.errorMessage = fmt.Sprintf("Error: %v", err)
			} else {
				p.result = result
			}
		})
	}()
}
