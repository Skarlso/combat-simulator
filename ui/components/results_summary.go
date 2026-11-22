package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/combat-simulator/internal/simulation"
)

// ResultsSummary displays simulation results
type ResultsSummary struct {
	app.Compo

	Result *simulation.SimulationResult
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
		)
}

// renderOverview renders the high-level summary
func (rs *ResultsSummary) renderOverview() app.UI {
	r := rs.Result
	draws := r.TotalRuns - r.PlayerWins - r.EnemyWins
	return app.Div().
		Class("results-overview").
		Body(
			app.H3().Text("Outcome"),
			app.Div().
				Class("stat-grid").
				Body(
					rs.renderStatCard("Player wins", fmt.Sprintf("%.1f%%", r.PlayerWinRate*100), fmt.Sprintf("%d of %d", r.PlayerWins, r.TotalRuns), "success"),
					rs.renderStatCard("Enemy wins", fmt.Sprintf("%.1f%%", r.EnemyWinRate*100), fmt.Sprintf("%d of %d", r.EnemyWins, r.TotalRuns), "danger"),
					rs.renderStatCard("Draws", fmt.Sprintf("%d", draws), "both sides down, or the turn limit", "info"),
				),
			rs.renderWinRateChart(),
		)
}

// renderStatistics renders detailed statistics
func (rs *ResultsSummary) renderStatistics() app.UI {
	return app.Div().
		Class("results-statistics").
		Body(
			app.H3().Text("Combat length"),
			app.Div().
				Class("stat-grid").
				Body(
					rs.renderStatCard("Avg turns", fmt.Sprintf("%.1f", rs.Result.AverageTurns), "", "info"),
					rs.renderStatCard("Avg rounds", fmt.Sprintf("%.1f", rs.Result.AverageRounds), "", "info"),
					rs.renderStatCard("Shortest", fmt.Sprintf("%d", rs.Result.MinTurns), "turns", "info"),
					rs.renderStatCard("Longest", fmt.Sprintf("%d", rs.Result.MaxTurns), "turns", "info"),
				),
		)
}

// renderStatCard renders a single statistic card
func (rs *ResultsSummary) renderStatCard(label, value, sub, variant string) app.UI {
	return app.Div().
		Class("stat-card stat-"+variant).
		Body(
			app.Div().Class("stat-label").Text(label),
			app.Div().Class("stat-value").Text(value),
			app.If(sub != "", func() app.UI {
				return app.Div().Class("stat-sub").Text(sub)
			}),
		)
}

// renderWinRateChart renders the outcome split as one stacked bar with a legend
func (rs *ResultsSummary) renderWinRateChart() app.UI {
	players := rs.Result.PlayerWinRate * 100
	enemies := rs.Result.EnemyWinRate * 100
	other := 100 - players - enemies
	if other < 0 {
		other = 0
	}

	segment := func(class string, pct float64) app.UI {
		return app.Div().
			Class("stacked-seg "+class).
			Style("width", fmt.Sprintf("%.2f%%", pct))
	}
	legendItem := func(class, label string, pct float64) app.UI {
		return app.Span().
			Class("legend-item").
			Body(
				app.Span().Class("legend-swatch "+class),
				app.Text(label+" "),
				app.Span().Class("legend-value").Text(fmt.Sprintf("%.1f%%", pct)),
			)
	}

	return app.Div().
		Class("results-chart").
		Body(
			app.Div().
				Class("stacked-bar").
				Body(
					segment("seg-players", players),
					segment("seg-enemies", enemies),
					segment("seg-other", other),
				),
			app.Div().
				Class("legend").
				Body(
					legendItem("seg-players", "Players", players),
					legendItem("seg-enemies", "Enemies", enemies),
					app.If(other > 0, func() app.UI {
						return legendItem("seg-other", "Draws", other)
					}),
				),
		)
}
