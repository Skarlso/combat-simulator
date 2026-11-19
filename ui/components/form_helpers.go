package components

import (
	"fmt"
	"slices"
	"strings"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func validateDiceNotation(s string) string {
	if s == "" {
		return ""
	}
	if _, err := models.ParseDiceNotation(s); err != nil {
		return "Invalid dice notation (e.g. 2d6+3)"
	}
	return ""
}

func renderValidationMsg(msg string) app.UI {
	if msg == "" {
		return app.Text("")
	}
	return app.Span().Class("validation-error").Text(msg)
}

func formGroupClass(hasError bool) string {
	if hasError {
		return "form-group has-error"
	}
	return "form-group"
}

func handleIntInput(target *int) func(app.Context, app.Event) {
	return func(ctx app.Context, e app.Event) {
		*target = ctx.JSSrc().Get("valueAsNumber").Int()
	}
}

// fieldScope keys a handler by the field it writes. go-app compares handlers by code
// pointer, so without it a closure from a shared helper survives on a sibling that moved.
func fieldScope(target any) app.EventOption {
	return app.EventScope(fmt.Sprintf("%p", target))
}

// numberField is a labelled number input bound to target
func numberField(label string, target *int, minVal, maxVal int) app.UI {
	return app.Div().Class("form-group").Body(
		app.Label().Text(label),
		app.Input().Type("number").Value(*target).Min(minVal).Max(maxVal).OnInput(handleIntInput(target), fieldScope(target)),
	)
}

// textField is a labelled text input bound to target; rerender redraws the form on each keystroke
func textField(label, placeholder string, target *string, rerender bool) app.UI {
	return app.Div().Class("form-group").Body(
		app.Label().Text(label),
		app.Input().Type("text").Value(*target).Placeholder(placeholder).OnInput(func(ctx app.Context, e app.Event) {
			*target = ctx.JSSrc().Get("value").String()
			if rerender {
				ctx.Update()
			}
		}, fieldScope(target)),
	)
}

type selectOption struct{ Value, Label string }

func optionElements(options []selectOption, current string) []app.UI {
	out := make([]app.UI, len(options))
	for i, o := range options {
		out[i] = app.Option().Value(o.Value).Text(o.Label).Selected(o.Value == current)
	}
	return out
}

// selectField is a labelled select; onChange receives the chosen value
func selectField(label, current string, options []selectOption, onChange func(app.Context, string)) app.UI {
	return app.Div().Class("form-group").Body(
		app.Label().Text(label),
		app.Select().
			OnChange(func(ctx app.Context, e app.Event) { onChange(ctx, ctx.JSSrc().Get("value").String()) }, app.EventScope(label)).
			Body(optionElements(options, current)...),
	)
}

// bindSelect stores the chosen value in target; rerender when other fields depend on it
func bindSelect(target *string, rerender bool) func(app.Context, string) {
	return func(ctx app.Context, v string) {
		*target = v
		if rerender {
			ctx.Update()
		}
	}
}

// checkboxField is a labelled checkbox bound to target; checkboxes gate other fields, so it always redraws
func checkboxField(label string, target *bool) app.UI {
	return app.Div().Class("form-group").Body(
		app.Label().Class("checkbox-label").Body(
			app.Input().Type("checkbox").Checked(*target).OnChange(func(ctx app.Context, e app.Event) {
				*target = ctx.JSSrc().Get("checked").Bool()
				ctx.Update()
			}, fieldScope(target)),
			app.Text(" "+label),
		),
	)
}

// formActions is the save/cancel pair every add form ends with
func formActions(saveText string, disabled bool, onSave, onCancel func(app.Context, app.Event)) app.UI {
	return app.Div().Class("form-actions").Body(
		app.Button().Class("btn btn-primary").Type("button").Text(saveText).Disabled(disabled).OnClick(onSave),
		app.Button().Class("btn btn-secondary").Type("button").Text("Cancel").OnClick(onCancel),
	)
}

// editableItem is a list row with a title, a meta line, optional effect and description, and Edit/Remove
func editableItem(editing bool, name, meta, effectText, description string, onEdit, onRemove func(app.Context, app.Event)) app.UI {
	return app.Div().
		Class(toggleClass("ability-item", editing, "editing")).
		Body(
			app.Div().Class("ability-info").Body(
				app.Strong().Text(name),
				app.Span().Text(" | "+meta),
				app.If(effectText != "", func() app.UI {
					return app.Div().Class("ability-effect").Text("Effect: " + effectText)
				}),
				app.If(description != "", func() app.UI {
					return app.Div().Class("ability-desc").Text(description)
				}),
			),
			app.Div().Class("item-actions").Body(
				app.Button().Class("btn btn-small").Text("Edit").OnClick(onEdit),
				app.Button().Class("btn btn-danger btn-small").Text("Remove").OnClick(onRemove),
			),
		)
}

// toggleClass appends modifier to base while on is set
func toggleClass(base string, on bool, modifier string) string {
	if on {
		return base + " " + modifier
	}
	return base
}

// tokenFields is the token subsection both feature forms share
func tokenFields(uses *bool, name *string, maxTokens *int) app.UI {
	return app.Div().Class("form-subsection").Body(
		app.H5().Text("Tokens"),
		checkboxField("Uses tokens", uses),
		app.If(*uses, func() app.UI {
			return app.Div().Class("form-row").Body(
				textField("Token Name", "Rage", name, false),
				numberField("Max Tokens (0=unlimited)", maxTokens, 0, 20),
			)
		}),
	)
}

// dcFields is the difficulty check subsection; targetLabel names what a "target" roll is against
func dcFields(dcType *string, fixedDC *int, targetLabel string) app.UI {
	options := []selectOption{{"none", "Auto-Succeed"}, {"target", targetLabel}, {"fixed", "Fixed DC"}}
	return app.Div().Class("form-subsection").Body(
		app.H5().Text("Difficulty Check"),
		app.Div().Class("form-row").Body(
			selectField("DC Type", *dcType, options, bindSelect(dcType, true)),
			app.If(*dcType == "fixed", func() app.UI { return numberField("Fixed DC", fixedDC, 1, 30) }),
		),
	)
}

var (
	damageTypeOptions = []selectOption{{"physical", "Physical"}, {"magic", "Magic"}}
	durationOptions   = []selectOption{{"1_turn", "1 Turn"}, {"until_cleared", "Until Cleared"}, {"permanent", "Permanent"}}
)

// effectFields are the inputs an effect type may need: dice, amount, damage type and duration
func effectFields(effectType string, dice *string, diceErr string, magnitude *int, damageType, duration *string) []app.UI {
	return []app.UI{
		app.If(needsDiceFormula(effectType), func() app.UI {
			return app.Div().Class(formGroupClass(diceErr != "")).Body(
				app.Label().Text("Dice Formula"),
				app.Input().Type("text").Value(*dice).Placeholder("2d6+3").OnInput(func(ctx app.Context, e app.Event) {
					*dice = ctx.JSSrc().Get("value").String()
					ctx.Update()
				}, fieldScope(dice)),
				renderValidationMsg(diceErr),
			)
		}),
		app.If(needsMagnitude(effectType), func() app.UI { return numberField("Amount", magnitude, 1, 10) }),
		app.If(needsDamageType(effectType), func() app.UI {
			return selectField("Damage Type", *damageType, damageTypeOptions, bindSelect(damageType, false))
		}),
		app.If(needsDuration(effectType), func() app.UI {
			return selectField("Duration", *duration, durationOptions, bindSelect(duration, false))
		}),
	}
}

var uiEffectCategoryMap = map[string]models.FeatureEffectCategory{
	"damage":             models.FeatureEffectDamage,
	"direct_damage":      models.FeatureEffectDirectDamage,
	"area_damage":        models.FeatureEffectAreaDamage,
	"heal":               models.FeatureEffectHPHeal,
	"stress_inflict":     models.FeatureEffectStressInflict,
	"stress_clear":       models.FeatureEffectStressClear,
	"apply_vulnerable":   models.FeatureEffectApplyVulnerable,
	"apply_poison":       models.FeatureEffectApplyPoison,
	"apply_ignite":       models.FeatureEffectApplyIgnite,
	"apply_stunned":      models.FeatureEffectApplyStunned,
	"apply_restrained":   models.FeatureEffectApplyRestrained,
	"armor_damage":       models.FeatureEffectArmorDamage,
	"grant_armor":        models.FeatureEffectGrantArmor,
	"damage_reduction":   models.FeatureEffectDamageReduction,
	"token_place":        models.FeatureEffectTokenPlace,
	"token_spend":        models.FeatureEffectTokenSpend,
	"resistance":         models.FeatureEffectDamageResistance,
	"immunity":           models.FeatureEffectDamageResistance,
	"summon_allies":      models.FeatureEffectSummonAllies,
	"apply_taunt":        models.FeatureEffectApplyTaunt,
	"fear_generate":      models.FeatureEffectFearGenerate,
	"hope_drain":         models.FeatureEffectHopeDrain,
	"spotlight_modifier": models.FeatureEffectSpotlightModifier,
}

func resolveEffectCategory(effectType string) models.FeatureEffectCategory {
	if cat, ok := uiEffectCategoryMap[effectType]; ok {
		return cat
	}
	return models.FeatureEffectDamage
}

// reverseEffectCategory names the select option for an effect; resistance and immunity share a category
func reverseEffectCategory(eff models.FeatureEffect) string {
	if eff.Category == models.FeatureEffectDamageResistance {
		if eff.Magnitude >= 100 {
			return "immunity"
		}
		return "resistance"
	}
	for key, val := range uiEffectCategoryMap {
		if val == eff.Category {
			return key
		}
	}
	return string(eff.Category)
}

// resistanceMagnitude is the engine's encoding of the resistance/immunity options
func resistanceMagnitude(effectType string, magnitude int) int {
	switch effectType {
	case "resistance":
		return 50
	case "immunity":
		return 100
	}
	return magnitude
}

// describeEffect summarises an effect for list rows
func describeEffect(effect models.FeatureEffect) string {
	switch effect.Category {
	case models.FeatureEffectDamage, models.FeatureEffectDirectDamage:
		return fmt.Sprintf("%s %s damage", effect.DiceFormula, effect.DamageType)
	case models.FeatureEffectHPHeal:
		return fmt.Sprintf("Heal %s", effect.DiceFormula)
	case models.FeatureEffectStressInflict:
		return fmt.Sprintf("Inflict %d stress", effect.Magnitude)
	case models.FeatureEffectStressClear:
		return fmt.Sprintf("Clear %d stress", effect.Magnitude)
	case models.FeatureEffectArmorDamage:
		return fmt.Sprintf("Damage %d armor slot(s)", effect.Magnitude)
	case models.FeatureEffectGrantArmor:
		return fmt.Sprintf("Grant %d armor slot(s)", effect.Magnitude)
	case models.FeatureEffectApplyPoison:
		return "Apply poison"
	case models.FeatureEffectApplyVulnerable:
		return "Apply vulnerable"
	case models.FeatureEffectSummonAllies:
		quantity := fmt.Sprintf("%d", effect.Magnitude)
		if effect.DiceFormula != "" {
			quantity = effect.DiceFormula
		}
		return fmt.Sprintf("Summon %s from %d templates", quantity, len(effect.SummonPool))
	default:
		return string(effect.Category)
	}
}

// joinParts joins the non-empty parts with sep
func joinParts(sep string, parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}

// countText is "3 Fear" style, or empty for zero
func countText(n int, unit string) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("%d %s", n, unit)
}

// statChip renders a compact label/value pair for list items
func statChip(label, value string) app.UI {
	return app.Span().Class("chip").Body(
		app.Span().Class("chip-label").Text(label),
		app.Text(value),
	)
}

func renderAccordionHeader(label string, open bool, toggle func(app.Context, app.Event)) app.UI {
	return app.Div().Class("accordion-header").OnClick(toggle, app.EventScope(label)).Body(
		app.H5().Text(label),
		app.Span().Class(toggleClass("accordion-indicator", open, "open")).Text("▶"),
	)
}

// accordion renders a collapsible section; open controls the body, toggle flips it
func accordion(label string, open bool, toggle func(app.Context, app.Event), body ...app.UI) []app.UI {
	return []app.UI{
		renderAccordionHeader(label, open, toggle),
		app.If(open, func() app.UI { return app.Div().Class("accordion-body").Body(body...) }),
	}
}

func oneOf(s string, options ...string) bool {
	return slices.Contains(options, s)
}

func needsDamageType(effectType string) bool {
	return oneOf(effectType, "damage", "direct_damage", "resistance", "immunity")
}

func needsDuration(effectType string) bool {
	return oneOf(effectType, "apply_vulnerable", "apply_poison", "apply_ignite", "apply_stunned", "apply_restrained", "apply_taunt")
}

func needsDiceFormula(effectType string) bool {
	return oneOf(effectType, "damage", "direct_damage", "area_damage", "heal", "damage_reduction")
}

func needsMagnitude(effectType string) bool {
	return oneOf(effectType, "stress_inflict", "stress_clear", "apply_poison", "apply_ignite", "armor_damage",
		"grant_armor", "token_place", "token_spend", "fear_generate", "hope_drain", "spotlight_modifier")
}

func needsArmorCost(effectType string) bool {
	return oneOf(effectType, "damage", "area_damage")
}

func isDamageEffect(effectType string) bool {
	return oneOf(effectType, "damage", "direct_damage", "area_damage")
}

// needsReaction reports whether the target can be given a reaction roll against the effect
func needsReaction(effectType string) bool {
	return isDamageEffect(effectType) || oneOf(effectType, "stress_inflict", "armor_damage") || needsDuration(effectType)
}

var reactionTraits = []string{"agility", "strength", "finesse", "instinct", "presence", "knowledge"}

func traitLabel(trait string) string {
	if trait == "" {
		return "None"
	}
	return strings.ToUpper(trait[:1]) + trait[1:]
}

// importPanel is the paste-JSON importer both lists share
type importPanel struct {
	show bool
	text string
	err  string
	info string
}

func (ip *importPanel) toggle(ctx app.Context) {
	ip.show = !ip.show
	ip.err, ip.info = "", ""
	ctx.Update()
}

func (ip *importPanel) render(title, help, placeholder string, onImport func(app.Context)) app.UI {
	return app.Div().Class("add-form").Body(
		app.H4().Text(title),
		app.P().Text(help),
		app.Div().Class("form-group").Body(
			app.Textarea().
				Style("width", "100%").
				Style("min-height", "200px").
				Style("font-family", "monospace").
				Placeholder(placeholder).
				Rows(12).
				Text(ip.text).
				OnInput(func(ctx app.Context, e app.Event) { ip.text = ctx.JSSrc().Get("value").String() }),
		),
		app.If(ip.err != "", func() app.UI { return app.Div().Class("validation-error").Text(ip.err) }),
		app.If(ip.info != "", func() app.UI { return app.Div().Class("validation-info").Text(ip.info) }),
		app.Div().Class("form-actions").Body(
			app.Button().Class("btn btn-primary").Text("Import").OnClick(func(ctx app.Context, e app.Event) { onImport(ctx) }),
			app.Button().Class("btn btn-secondary").Text("Close").OnClick(func(ctx app.Context, e app.Event) {
				ip.show = false
				ctx.Update()
			}),
		),
	)
}

// run parses the pasted text with load and hands every result to onAdd
func (ip *importPanel) run(ctx app.Context, noun string, load func([]byte) ([]*models.Combatant, error), onAdd func(app.Context, *models.Combatant)) {
	ip.err, ip.info = "", ""
	defer ctx.Update()

	if ip.text == "" {
		ip.err = "Paste JSON before importing."
		return
	}
	imported, err := load([]byte(ip.text))
	if err != nil {
		ip.err = err.Error()
		return
	}
	if len(imported) == 0 {
		ip.err = "no " + noun + " found in JSON"
		return
	}
	if onAdd != nil {
		for _, c := range imported {
			onAdd(ctx, c)
		}
	}
	ip.info = fmt.Sprintf("Imported %d %s.", len(imported), noun)
	ip.text = ""
}
