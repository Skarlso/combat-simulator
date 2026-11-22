package components

import (
	"fmt"
	"strings"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/combat-simulator/internal/engine"
	"github.com/skarlso/combat-simulator/internal/models"
	"github.com/skarlso/combat-simulator/internal/simulation"
)

// CombatLog shows one simulated combat row by row, with where every number came from
type CombatLog struct {
	app.Compo

	Result *simulation.SimulationResult

	hidden bool
	// Which runs to page through ("" = all, or one of the outcome filters) and where among them
	filter   string
	position int
}

const (
	filterPlayerWins = "players"
	filterEnemyWins  = "enemies"
	filterDraws      = "draws"
)

func outcomeKind(outcome *engine.CombatOutcome) string {
	switch {
	case outcome.PlayerVictory:
		return filterPlayerWins
	case outcome.EnemyVictory:
		return filterEnemyWins
	default:
		return filterDraws
	}
}

// matching lists the runs, by index, that ended the way the filter asks
func (cl *CombatLog) matching(filter string) []int {
	var runs []int
	for i, outcome := range cl.Result.CombatOutcomes {
		if filter == "" || outcomeKind(outcome) == filter {
			runs = append(runs, i)
		}
	}
	return runs
}

// Render renders the outcome filter, the run picker and the log table
func (cl *CombatLog) Render() app.UI {
	if cl.Result == nil || len(cl.Result.CombatOutcomes) == 0 {
		return app.Div()
	}
	runs := cl.matching(cl.filter)
	cl.position = max(min(cl.position, len(runs)-1), 0)

	step := func(label string, by int) app.UI {
		return app.Button().
			Class("btn btn-small").
			Text(label).
			Disabled(cl.position+by < 0 || cl.position+by >= len(runs)).
			OnClick(func(ctx app.Context, e app.Event) {
				cl.position += by
				ctx.Update()
			}, app.EventScope(label))
	}
	indicator := "No runs"
	if len(runs) > 0 {
		indicator = fmt.Sprintf("Run %d of %d", runs[cl.position]+1, len(cl.Result.CombatOutcomes))
		if cl.filter != "" {
			indicator += fmt.Sprintf(" · %d/%d", cl.position+1, len(runs))
		}
	}

	return app.Div().
		Class("combat-log-section").
		Body(
			app.Div().
				Class("combat-log-header").
				Body(
					app.H2().Text("Combat Log"),
					app.Div().
						Class("log-controls").
						Body(
							app.Div().Class("tabs tabs-compact").Body(
								cl.renderFilter("", "All"),
								cl.renderFilter(filterPlayerWins, "Player wins"),
								cl.renderFilter(filterEnemyWins, "Enemy wins"),
								app.If(len(cl.matching(filterDraws)) > 0, func() app.UI { return cl.renderFilter(filterDraws, "Draws") }),
							),
							step("◀ Prev", -1),
							app.Span().Class("run-indicator").Text(indicator),
							step("Next ▶", 1),
							app.Button().
								Class("btn btn-small").
								Text(logToggleLabel(!cl.hidden)).
								OnClick(func(ctx app.Context, e app.Event) {
									cl.hidden = !cl.hidden
									ctx.Update()
								}),
						),
				),
			app.If(!cl.hidden && len(runs) > 0, func() app.UI {
				return cl.renderLog(cl.Result.CombatOutcomes[runs[cl.position]])
			}),
			app.If(!cl.hidden && len(runs) == 0, func() app.UI {
				return app.Div().Class("empty-state").Text("No run ended that way.")
			}),
		)
}

// renderFilter is one outcome tab with the number of runs that ended that way
func (cl *CombatLog) renderFilter(filter, label string) app.UI {
	return app.Button().
		Class(toggleClass("tab", cl.filter == filter, "active")).
		Body(
			app.Text(label),
			app.Span().Class("tab-count").Text(fmt.Sprintf("%d", len(cl.matching(filter)))),
		).
		OnClick(func(ctx app.Context, e app.Event) {
			cl.filter, cl.position = filter, 0
			ctx.Update()
		}, app.EventScope(label))
}

// renderLog renders the detailed action log
func (cl *CombatLog) renderLog(outcome *engine.CombatOutcome) app.UI {
	outcomeText := "Draw"
	outcomeClass := "outcome-draw"
	if outcome.PlayerVictory {
		outcomeText = "✓ Players Victory"
		outcomeClass = "outcome-player-win"
	} else if outcome.EnemyVictory {
		outcomeText = "✓ Enemies Victory"
		outcomeClass = "outcome-enemy-win"
	}

	rows := []app.UI{
		app.Div().Class("log-row log-header").Body(
			app.Span().Class("log-col-turn").Text("Turn"),
			app.Span().Class("log-col-actor").Text("Actor"),
			app.Span().Class("log-col-action").Text("Action"),
			app.Span().Class("log-col-target").Text("Target"),
			app.Span().Class("log-col-dice").Text("Roll"),
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
		if action.FeatureName != "" {
			actionText += ": " + action.FeatureName
		}

		targetText := action.Target
		if targetText == "" {
			targetText = "-"
		}

		rows = append(rows, app.Div().Class(rowClass).Body(
			app.Span().Class("log-col-turn").Text(fmt.Sprintf("R%d T%d", action.Round, action.Turn)),
			app.Span().Class("log-col-actor").Body(app.Text(action.Actor), stateLine(action.ActorState)),
			app.Span().Class("log-col-action").Text(actionText),
			app.Span().Class("log-col-target").Body(app.Text(targetText), stateLine(action.TargetState)),
			app.Span().Class("log-col-dice").Body(diceRollCell(action)),
			app.Span().Class("log-col-damage").Body(damageCell(action)),
			app.Span().Class("log-col-result").Body(
				app.Text(resultText(action)),
				app.Range(action.Notes).Slice(func(i int) app.UI {
					return app.Span().Class("log-note").Text(action.Notes[i])
				}),
			),
			app.Span().Class("log-col-pools").Body(
				app.Text(poolsText(action)),
				app.If(len(action.FearChanges) > 0, func() app.UI {
					return app.Span().Class("log-state").Text(fearChangesText(action.FearChanges))
				}),
			),
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

// stateLine shows where a combatant stands after the action: a PC's HP, Stress, Hope and
// Armor Slots, an adversary's HP and Stress
func stateLine(state *models.CombatantState) app.UI {
	if state == nil {
		return app.Text("")
	}
	text := fmt.Sprintf("HP %d/%d · Stress %d/%d", state.HP, state.HPMax, state.Stress, state.StressMax)
	if state.IsPC {
		text += fmt.Sprintf(" · Hope %d · Armor %d/%d", state.Hope, state.Armor, state.ArmorMax)
	}
	return app.Span().Class("log-state").Title("After this action: HP and Armor Slots left, Stress marked").Text(text)
}

func logToggleLabel(shown bool) string {
	if shown {
		return "Hide Log"
	}
	return "Show Log"
}

// signedParts writes named parts as " + 2 Strength - 3 disadvantage d6". The first part always
// shows, even at 0; a later one worth nothing is left out, unless it is the GM's advantage or
// disadvantage, which changes which d20 is kept rather than adding a number.
func signedParts(parts []models.LogPart) string {
	var b strings.Builder
	for i, part := range parts {
		switch {
		case part.Value == 0 && (part.Label == "advantage" || part.Label == "disadvantage"):
			b.WriteString(", " + part.Label)
		case part.Value == 0 && i > 0:
		case part.Value < 0:
			b.WriteString(strings.TrimRight(fmt.Sprintf(" - %d %s", -part.Value, part.Label), " "))
		default:
			b.WriteString(strings.TrimRight(fmt.Sprintf(" + %d %s", part.Value, part.Label), " "))
		}
	}
	return b.String()
}

// rollTarget names what a roll had to meet: the target's Evasion or Difficulty, or a fixed one
func rollTarget(action models.ActionRecord) string {
	switch {
	case action.TotalRoll == 0 && action.TargetDefense == 0:
		// A death move rolls the dice bare: no total, nothing to beat
		return ""
	case !action.VsDefense:
		return fmt.Sprintf(" = %d (vs Difficulty %d)", action.TotalRoll, action.TargetDefense)
	case action.TargetState != nil && action.TargetState.IsPC:
		return fmt.Sprintf(" = %d (vs Evasion %d)", action.TotalRoll, action.TargetDefense)
	default:
		return fmt.Sprintf(" = %d (vs %s's Difficulty %d)", action.TotalRoll, action.Target, action.TargetDefense)
	}
}

// diceRollCell shows the dice, everything added to them by name, and what the total had to meet
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
			app.Span().Class(hopeClass).Text(fmt.Sprintf("Hope %d", action.HopeDie)),
			app.Text(" + "),
			app.Span().Class(fearClass).Text(fmt.Sprintf("Fear %d", action.FearDie)),
			app.Text(signedParts(action.RollParts)+rollTarget(action)),
		)
	case action.D20Roll > 0:
		return app.Text(fmt.Sprintf("d20 %d%s%s", action.D20Roll, signedParts(action.RollParts), rollTarget(action)))
	default:
		return app.Text("-")
	}
}

// damageSum writes the damage roll out: the dice and the weapon's bonus from its formula,
// then every named addition, summing to the total
func damageSum(action models.ActionRecord) string {
	var terms []string
	for _, die := range action.DamageDice {
		terms = append(terms, fmt.Sprintf("%d", die))
	}
	if action.DamageBonus > 0 {
		terms = append(terms, fmt.Sprintf("%d", action.DamageBonus))
	}
	sum := strings.Join(terms, " + ")
	parts := signedParts(action.DamageParts)
	if sum == "" {
		parts = strings.TrimPrefix(parts, " + ")
	}
	sum += parts
	if sum == "" || sum == fmt.Sprintf("%d", action.DamageRoll) {
		return fmt.Sprintf("%d", action.DamageRoll)
	}
	return fmt.Sprintf("%s = %d", sum, action.DamageRoll)
}

// damageCell shows the damage roll with its sources, what reduced it, the Hit Points it marked
// and how the severity was reached
func damageCell(action models.ActionRecord) app.UI {
	if action.DamageRoll <= 0 && len(action.DamageParts) == 0 {
		return app.Text("-")
	}

	roll := damageSum(action)
	if action.DamageFormula != "" {
		roll = action.DamageFormula + ": " + roll
	}
	if len(action.Reduced) > 0 {
		roll += fmt.Sprintf(", %d after %s", action.ProcessedDamage, strings.Join(action.Reduced, ", "))
	}

	marked := fmt.Sprintf("→ %d HP", action.HPDamage)
	if action.TargetsHit > 1 {
		marked += fmt.Sprintf(" across %d targets", action.TargetsHit)
	}
	severity := action.Threshold
	if len(action.Severity) > 0 {
		severity = strings.Join(action.Severity, " → ")
	}

	return app.Span().Body(
		app.Span().Class("log-damage-roll").Text(roll),
		app.Span().Class("log-damage-marked").Text(marked+" "),
		app.Span().Class("log-severity").Title("Severity by the target's Major/Severe thresholds, then each reduction").Text("["+severity+"]"),
	)
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

// fearChangesText spells out what moved the Fear pool on this row: "Fear -1 for another
// spotlight, +2 no Armor Slot to mark". The same reason on several targets is added up.
func fearChangesText(changes []models.LogPart) string {
	var order []string
	totals := map[string]int{}
	for _, c := range changes {
		if _, seen := totals[c.Label]; !seen {
			order = append(order, c.Label)
		}
		totals[c.Label] += c.Value
	}
	parts := make([]string, len(order))
	for i, label := range order {
		parts[i] = fmt.Sprintf("%+d %s", totals[label], label)
	}
	return "Fear " + strings.Join(parts, ", ")
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
