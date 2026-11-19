package components

import (
	"fmt"
	"strings"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
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
			rs.renderCombatLogSection(),
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
					rs.renderStatCard("Draws", fmt.Sprintf("%d", draws), "hit the turn limit", "info"),
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
					app.H3().Text("Combat Log"),
					app.Div().
						Class("log-controls").
						Body(
							app.Button().
								Class("btn btn-small").
								Text("◀ Prev").
								Disabled(rs.selectedOutcome <= 0).
								OnClick(func(ctx app.Context, e app.Event) {
									if rs.selectedOutcome > 0 {
										rs.selectedOutcome--
										ctx.Update()
									}
								}),
							app.Span().
								Class("run-indicator").
								Text(fmt.Sprintf("Run %d/%d", rs.selectedOutcome+1, len(rs.Result.CombatOutcomes))),
							app.Button().
								Class("btn btn-small").
								Text("Next ▶").
								Disabled(rs.selectedOutcome >= len(rs.Result.CombatOutcomes)-1).
								OnClick(func(ctx app.Context, e app.Event) {
									if rs.selectedOutcome < len(rs.Result.CombatOutcomes)-1 {
										rs.selectedOutcome++
										ctx.Update()
									}
								}),
							app.Button().
								Class("btn btn-small").
								Text(logToggleLabel(rs.showCombatLog)).
								OnClick(func(ctx app.Context, e app.Event) {
									rs.showCombatLog = !rs.showCombatLog
									ctx.Update()
								}),
						),
				),
			app.If(rs.showCombatLog, func() app.UI {
				return rs.renderCombatLog()
			}),
		)
}

// renderCombatLog renders the detailed action log
func (rs *ResultsSummary) renderCombatLog() app.UI {
	outcome := rs.Result.CombatOutcomes[rs.selectedOutcome]

	// Determine winner
	outcomeText := "Draw"
	outcomeClass := "outcome-draw"
	if outcome.PlayerVictory {
		outcomeText = "✓ Players Victory"
		outcomeClass = "outcome-player-win"
	} else if outcome.EnemyVictory {
		outcomeText = "✓ Enemies Victory"
		outcomeClass = "outcome-enemy-win"
	}

	// Build table rows
	rows := []app.UI{
		// Header row
		app.Div().Class("log-row log-header").Body(
			app.Span().Class("log-col-turn").Text("Round/Turn"),
			app.Span().Class("log-col-actor").Text("Actor"),
			app.Span().Class("log-col-action").Text("Action"),
			app.Span().Class("log-col-target").Text("Target"),
			app.Span().Class("log-col-dice").Text("Dice Roll"),
			app.Span().Class("log-col-damage").Text("Damage"),
			app.Span().Class("log-col-result").Text("Result"),
			app.Span().Class("log-col-pools").Text("Hope/Fear"),
		),
	}

	for _, action := range outcome.ActionHistory {
		rowClass := "log-row"
		if action.IsCritical {
			rowClass += " log-row-critical"
		} else if action.IsFeature {
			rowClass += " log-row-feature"
		}

		actionText := action.ActionType.String()
		if action.IsFeature && action.FeatureName != "" {
			actionText += ": " + action.FeatureName
		}

		targetText := action.Target
		if targetText == "" {
			targetText = "-"
		}

		rows = append(rows, app.Div().Class(rowClass).Body(
			app.Span().Class("log-col-turn").Text(fmt.Sprintf("R%d T%d", action.Round, action.Turn)),
			app.Span().Class("log-col-actor").Text(action.Actor),
			app.Span().Class("log-col-action").Text(actionText),
			app.Span().Class("log-col-target").Text(targetText),
			app.Span().Class("log-col-dice").Body(diceRollCell(action)),
			app.Span().Class("log-col-damage").Text(damageText(action)),
			app.Span().Class("log-col-result").Text(resultText(action)),
			app.Span().Class("log-col-pools").Text(poolsText(action)),
		))
	}

	return app.Div().
		Class("combat-log-viewer").
		Body(
			app.Div().
				Class("log-info").
				Body(
					app.Span().Text(fmt.Sprintf("%d actions, %d turns - ", len(outcome.ActionHistory), outcome.Turns)),
					app.Span().Class(outcomeClass).Text(outcomeText),
				),
			app.Div().
				Class("log-table").
				Body(rows...),
		)
}

func logToggleLabel(shown bool) string {
	if shown {
		return "Hide Log"
	}
	return "Show Log"
}

// diceRollCell shows Duality Dice with the winning die marked, or the GM's d20
func diceRollCell(action models.ActionRecord) app.UI {
	switch {
	case action.FearDie > 0:
		hopeClass, fearClass := "dice-hope", "dice-fear"
		if action.FearDie > action.HopeDie {
			fearClass += " dice-winner"
		} else {
			hopeClass += " dice-winner"
		}
		return app.Span().Body(
			app.Span().Class(hopeClass).Text(fmt.Sprintf("H:%d", action.HopeDie)),
			app.Text(" "),
			app.Span().Class(fearClass).Text(fmt.Sprintf("F:%d", action.FearDie)),
			app.Text(fmt.Sprintf(" = %d (vs %d)", action.TotalRoll, action.TargetDefense)),
		)
	case action.D20Roll > 0:
		return app.Text(fmt.Sprintf("d20:%d = %d (vs %d)", action.D20Roll, action.TotalRoll, action.TargetDefense))
	default:
		return app.Text("-")
	}
}

// damageText lists the dice rolled, the damage that counted after resistance, and the HP marked
func damageText(action models.ActionRecord) string {
	if action.DamageRoll <= 0 {
		return "-"
	}
	dice := fmt.Sprintf("%d", action.DamageRoll)
	if len(action.DamageDice) > 0 {
		parts := make([]string, 0, len(action.DamageDice)+1)
		for _, val := range action.DamageDice {
			parts = append(parts, fmt.Sprintf("%d", val))
		}
		if action.DamageBonus > 0 {
			parts = append(parts, fmt.Sprintf("%d", action.DamageBonus))
		}
		dice = strings.Join(parts, "+")
	}
	counted := action.DamageRoll
	if action.ProcessedDamage > 0 {
		counted = action.ProcessedDamage
	}
	return fmt.Sprintf("%s=%d→%dHP [%s]", dice, counted, action.HPDamage, action.Threshold)
}

// resultText appends the target's HP change, and armor use, to the outcome
func resultText(action models.ActionRecord) string {
	if action.Target == "" {
		return action.Result
	}
	if action.TargetHPBefore <= 0 {
		return fmt.Sprintf("%s (HP:%d)", action.Result, action.TargetHPAfter)
	}
	text := fmt.Sprintf("%s - %s: %d HP → %d HP", action.Result, action.Target, action.TargetHPBefore, action.TargetHPAfter)
	if action.ArmorUsed {
		text += fmt.Sprintf(" (armor: %d→%d slots)", action.ArmorSlotsBefore, action.ArmorSlotsAfter)
	}
	return text
}

func poolsText(action models.ActionRecord) string {
	switch {
	case action.HopeGenerated > 0:
		return fmt.Sprintf("H:%d(+%d) F:%d", action.HopePoolAfter, action.HopeGenerated, action.FearPoolAfter)
	case action.FearGenerated > 0:
		return fmt.Sprintf("H:%d F:%d(+%d)", action.HopePoolAfter, action.FearPoolAfter, action.FearGenerated)
	default:
		return fmt.Sprintf("H:%d F:%d", action.HopePoolAfter, action.FearPoolAfter)
	}
}
