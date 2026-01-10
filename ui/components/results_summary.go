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
		// Format dice roll
		diceRoll := "-"
		if action.HopeDie > 0 || action.FearDie > 0 || action.D20Roll > 0 {
			// Check if this is a player (both Hope and Fear) or enemy (d20)
			if action.FearDie > 0 {
				// Player attack/ability - duality dice
				diceRoll = fmt.Sprintf("H:%d F:%d = %d (vs %d)",
					action.HopeDie, action.FearDie, action.TotalRoll, action.TargetDefense)
			} else if action.D20Roll > 0 {
				// Enemy attack/ability - d20
				diceRoll = fmt.Sprintf("d20:%d = %d (vs %d)",
					action.D20Roll, action.TotalRoll, action.TargetDefense)
			} else {
				// Old format fallback (for backwards compatibility)
				diceRoll = fmt.Sprintf("d20:%d = %d (vs %d)",
					action.HopeDie, action.TotalRoll, action.TargetDefense)
			}
		}

		// Format damage with individual dice
		damageText := "-"
		if action.DamageRoll > 0 {
			diceStr := ""
			if len(action.DamageDice) > 0 {
				// Show individual dice values
				parts := make([]string, len(action.DamageDice))
				for i, val := range action.DamageDice {
					parts[i] = fmt.Sprintf("%d", val)
				}
				diceStr = fmt.Sprintf("%s", parts[0])
				for i := 1; i < len(parts); i++ {
					diceStr += fmt.Sprintf("+%s", parts[i])
				}
				if action.DamageBonus > 0 {
					diceStr += fmt.Sprintf("+%d", action.DamageBonus)
				}
			} else {
				diceStr = fmt.Sprintf("%d", action.DamageRoll)
			}
			// Show processed damage (after resistance) if different from raw damage
			if action.ProcessedDamage > 0 && action.ProcessedDamage != action.DamageRoll {
				damageText = fmt.Sprintf("%s=%d→%dHP [%s]", diceStr, action.ProcessedDamage, action.HPDamage, action.Threshold)
			} else {
				damageText = fmt.Sprintf("%s=%d→%dHP [%s]", diceStr, action.DamageRoll, action.HPDamage, action.Threshold)
			}
		}

		// Format result with HP and armor info
		resultText := action.Result
		if action.Target != "" && action.TargetHPAfter >= 0 {
			// Show HP before/after if we have before info
			if action.TargetHPBefore > 0 {
				hpChange := fmt.Sprintf("%s: %d HP → %d HP", action.Target, action.TargetHPBefore, action.TargetHPAfter)

				// Add armor info if armor was used
				if action.ArmorUsed {
					hpChange += fmt.Sprintf(" (armor: %d→%d slots)", action.ArmorSlotsBefore, action.ArmorSlotsAfter)
				}

				resultText = fmt.Sprintf("%s - %s", action.Result, hpChange)
			} else {
				// Fallback to old format if no before info
				resultText = fmt.Sprintf("%s (HP:%d)", action.Result, action.TargetHPAfter)
			}
		}

		targetText := action.Target
		if targetText == "" {
			targetText = "-"
		}

		// Format Hope/Fear pools
		poolsText := fmt.Sprintf("H:%d F:%d", action.HopePoolAfter, action.FearPoolAfter)
		if action.HopeGenerated > 0 {
			poolsText = fmt.Sprintf("H:%d(+%d) F:%d", action.HopePoolAfter, action.HopeGenerated, action.FearPoolAfter)
		} else if action.FearGenerated > 0 {
			poolsText = fmt.Sprintf("H:%d F:%d(+%d)", action.HopePoolAfter, action.FearPoolAfter, action.FearGenerated)
		}

		// Add styling classes
		rowClass := "log-row"
		if action.IsCritical {
			rowClass = "log-row log-row-critical"
		} else if action.IsFeature {
			rowClass = "log-row log-row-feature"
		}

		// Show feature name if present
		actionText := action.ActionType.String()
		if action.IsFeature && action.FeatureName != "" {
			actionText = fmt.Sprintf("%s: %s", action.ActionType.String(), action.FeatureName)
		}

		rows = append(rows, app.Div().Class(rowClass).Body(
			app.Span().Class("log-col-turn").Text(fmt.Sprintf("R%d T%d", action.Round, action.Turn)),
			app.Span().Class("log-col-actor").Text(action.Actor),
			app.Span().Class("log-col-action").Text(actionText),
			app.Span().Class("log-col-target").Text(targetText),
			app.Span().Class("log-col-dice").Text(diceRoll),
			app.Span().Class("log-col-damage").Text(damageText),
			app.Span().Class("log-col-result").Text(resultText),
			app.Span().Class("log-col-pools").Text(poolsText),
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
