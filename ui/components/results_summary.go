package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/internal/simulation"
)

// ResultsSummary displays simulation results
type ResultsSummary struct {
	app.Compo

	Result          *simulation.SimulationResult
	showCombatLog   bool
	selectedOutcome int
}

// Render renders the results summary
func (rs *ResultsSummary) Render() app.UI {
	if rs.Result == nil {
		return app.Div()
	}

	return app.Div().
		Class("results-summary").
		Body(
			rs.renderOverview(),
			rs.renderStatistics(),
			rs.renderWinRateChart(),
			rs.renderCombatLogSection(),
		)
}

// renderOverview renders the high-level summary
func (rs *ResultsSummary) renderOverview() app.UI {
	return app.Div().
		Class("results-overview").
		Body(
			app.H3().Text("Overview"),
			app.Div().
				Class("stat-grid").
				Body(
					rs.renderStatCard("Total Simulations", fmt.Sprintf("%d", rs.Result.TotalRuns), "info"),
					rs.renderStatCard("Player Wins", fmt.Sprintf("%d (%.1f%%)", rs.Result.PlayerWins, rs.Result.PlayerWinRate*100), "success"),
					rs.renderStatCard("Enemy Wins", fmt.Sprintf("%d (%.1f%%)", rs.Result.EnemyWins, rs.Result.EnemyWinRate*100), "danger"),
				),
		)
}

// renderStatistics renders detailed statistics
func (rs *ResultsSummary) renderStatistics() app.UI {
	return app.Div().
		Class("results-statistics").
		Body(
			app.H3().Text("Combat Statistics"),
			app.Div().
				Class("stat-grid").
				Body(
					rs.renderStatCard("Average Turns", fmt.Sprintf("%.1f", rs.Result.AverageTurns), "info"),
					rs.renderStatCard("Average Rounds", fmt.Sprintf("%.1f", rs.Result.AverageRounds), "info"),
					rs.renderStatCard("Min Turns", fmt.Sprintf("%d", rs.Result.MinTurns), "info"),
					rs.renderStatCard("Max Turns", fmt.Sprintf("%d", rs.Result.MaxTurns), "info"),
				),
		)
}

// renderStatCard renders a single statistic card
func (rs *ResultsSummary) renderStatCard(label, value, variant string) app.UI {
	return app.Div().
		Class("stat-card stat-" + variant).
		Body(
			app.Div().Class("stat-label").Text(label),
			app.Div().Class("stat-value").Text(value),
		)
}

// renderWinRateChart renders a simple visual representation of win rates
func (rs *ResultsSummary) renderWinRateChart() app.UI {
	playerWidth := int(rs.Result.PlayerWinRate * 100)
	enemyWidth := int(rs.Result.EnemyWinRate * 100)

	return app.Div().
		Class("results-chart").
		Body(
			app.H3().Text("Win Rate Distribution"),
			app.Div().
				Class("chart-container").
				Body(
					app.Div().
						Class("chart-row").
						Body(
							app.Div().Class("chart-label").Text("Players"),
							app.Div().
								Class("chart-bar").
								Body(
									app.Div().
										Class("chart-fill chart-fill-success").
										Style("width", fmt.Sprintf("%d%%", playerWidth)).
										Text(fmt.Sprintf("%.1f%%", rs.Result.PlayerWinRate*100)),
								),
						),
					app.Div().
						Class("chart-row").
						Body(
							app.Div().Class("chart-label").Text("Enemies"),
							app.Div().
								Class("chart-bar").
								Body(
									app.Div().
										Class("chart-fill chart-fill-danger").
										Style("width", fmt.Sprintf("%d%%", enemyWidth)).
										Text(fmt.Sprintf("%.1f%%", rs.Result.EnemyWinRate*100)),
								),
						),
				),
		)
}

// renderCombatLogSection renders collapsible combat log viewer
func (rs *ResultsSummary) renderCombatLogSection() app.UI {
	if len(rs.Result.CombatOutcomes) == 0 {
		return app.Div()
	}

	return app.Div().
		Class("combat-log-section").
		Body(
			app.Div().
				Class("combat-log-header").
				Body(
					app.H3().Text("Combat Log (Sample Run)"),
					app.Button().
						Class("btn btn-small").
						Text(func() string {
							if rs.showCombatLog {
								return "Hide Log"
							}
							return "Show Log"
						}()).
						OnClick(func(ctx app.Context, e app.Event) {
							rs.showCombatLog = !rs.showCombatLog
							ctx.Update()
						}),
				),
			app.If(rs.showCombatLog, func() app.UI {
				return rs.renderCombatLog()
			}),
		)
}

// renderCombatLog renders the detailed action log
func (rs *ResultsSummary) renderCombatLog() app.UI {
	outcome := rs.Result.CombatOutcomes[rs.selectedOutcome]

	logEntries := make([]app.UI, len(outcome.ActionHistory))
	for i, action := range outcome.ActionHistory {
		logEntries[i] = app.Div().
			Class("log-entry").
			Body(
				app.Span().Class("log-turn").Text(fmt.Sprintf("R%d T%d", action.Round, action.Turn)),
				app.Span().Class("log-actor").Text(action.Actor),
				app.Span().Class("log-action").Text(action.ActionType.String()),
				app.If(action.Target != "", func() app.UI {
					return app.Span().Class("log-target").Text(fmt.Sprintf("→ %s", action.Target))
				}),
				app.If(action.Ability != "", func() app.UI {
					return app.Span().Class("log-ability").Text(fmt.Sprintf("(%s)", action.Ability))
				}),
				app.Span().Class("log-result").Text(action.Result),
				app.If(action.Damage > 0, func() app.UI {
					return app.Span().Class("log-damage").Text(fmt.Sprintf("%d dmg", action.Damage))
				}),
			)
	}

	return app.Div().
		Class("combat-log-viewer").
		Body(
			app.Div().
				Class("log-info").
				Text(fmt.Sprintf("Showing run #1 of %d (%d actions, %d turns)",
					rs.Result.TotalRuns, len(outcome.ActionHistory), outcome.Turns)),
			app.Div().
				Class("log-entries").
				Body(logEntries...),
		)
}
