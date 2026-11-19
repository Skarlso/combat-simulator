package pages

import (
	"fmt"
	"time"

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
	configSet         bool
	adversaryRegistry *parser.AdversaryRegistry
}

// OnNav initializes the page on first navigation
func (p *SimulatorPage) OnNav(ctx app.Context) {
	if !p.initialized {
		p.currentTab = "players"
		p.config = simulation.DefaultSimulationConfig()
		p.configSet = true
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
	// Only seed defaults once: RunCount hits 0 whenever the user clears the input to retype it
	if !p.configSet {
		p.config = simulation.DefaultSimulationConfig()
		p.configSet = true
	}

	workspaceClass := "workspace"
	if p.result != nil {
		workspaceClass += " has-results"
	}

	return app.Div().
		Class("simulator-page").
		Body(
			p.renderToolbar(),
			app.Div().
				Class(workspaceClass).
				Body(
					app.Div().
						Class("builder").
						Body(
							app.If(p.errorMessage != "", func() app.UI {
								return app.Div().Class("error-message").Text(p.errorMessage)
							}),
							p.renderTabContent(),
						),
					app.If(p.result != nil, func() app.UI {
						return p.renderResults()
					}),
				),
		)
}

// renderToolbar renders the sticky bar with tabs and run controls
func (p *SimulatorPage) renderToolbar() app.UI {
	return app.Div().
		Class("toolbar").
		Body(
			app.Div().
				Class("tabs").
				Body(
					p.renderTab("players", "Party", len(p.players)),
					p.renderTab("enemies", "Adversaries", len(p.enemies)),
					p.renderTab("config", "Settings", -1),
				),
			app.Div().
				Class("toolbar-actions").
				Body(
					p.renderStatusChip(),
					app.Button().
						Class("btn btn-primary").
						Text(fmt.Sprintf("Run %d", p.config.RunCount)).
						Disabled(p.isRunning || len(p.players) == 0 || len(p.enemies) == 0).
						OnClick(func(ctx app.Context, e app.Event) {
							p.runSimulation(ctx)
						}),
				),
		)
}

func (p *SimulatorPage) renderStatusChip() app.UI {
	switch {
	case p.isRunning:
		return app.Span().Class("status-chip running").Text("Running")
	case len(p.players) == 0 || len(p.enemies) == 0:
		return app.Span().Class("status-chip").Text("Add a party and adversaries to run")
	default:
		return app.Span().Class("status-chip").Text("Ready")
	}
}

// renderTab renders a single tab button; count < 0 hides the badge
func (p *SimulatorPage) renderTab(id, label string, count int) app.UI {
	class := "tab"
	if p.currentTab == id {
		class += " active"
	}

	return app.Button().
		Class(class).
		Body(
			app.Text(label),
			app.If(count >= 0, func() app.UI {
				return app.Span().Class("tab-count").Text(fmt.Sprintf("%d", count))
			}),
		).
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
				&components.CombatantList{
					Title:      "Party",
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
				&components.AdversaryList{
					Title:       "Adversaries",
					Adversaries: p.enemies,
					Registry:    p.adversaryRegistry,
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
				app.H2().Text("Simulation Settings"),
				&components.ConfigForm{Config: &p.config},
			)

	default:
		return app.Div().Text("Unknown tab")
	}
}

// renderResults renders the simulation results panel
func (p *SimulatorPage) renderResults() app.UI {
	return app.Div().
		Class("results-panel").
		Body(
			app.Div().
				Class("panel-header").
				Body(
					app.H2().Text("Results"),
					app.Button().
						Class("btn btn-ghost btn-small").
						Text("Clear").
						OnClick(func(ctx app.Context, e app.Event) {
							p.result = nil
							p.errorMessage = ""
							ctx.Update()
						}),
				),
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

	go func() {
		// Sleeping yields to the browser so the "Running" state paints before the CPU-bound run
		time.Sleep(50 * time.Millisecond)

		p.config.PlayerTemplates = p.players
		p.config.EnemyTemplates = p.enemies
		p.config.AdversaryRegistry = p.adversaryRegistry
		result, err := simulation.NewSimulationRunner(p.config).Run()

		ctx.Dispatch(func(ctx app.Context) {
			p.isRunning = false
			if err != nil {
				p.errorMessage = fmt.Sprintf("Error: %v", err)
			} else {
				p.result = result
			}
		})
	}()
}
