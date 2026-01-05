package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// AbilityForm manages adding/editing abilities
type AbilityForm struct {
	app.Compo

	Abilities []*models.Ability
	OnChange  func()

	// Form state
	showForm       bool
	newName        string
	newDescription string
	newDiceCount   int
	newDiceSize    int
	newFlatBonus   int
	newDamageType  string
	newIsAOE       bool
	newAOETargets  int
}

// OnMount initializes default values
func (af *AbilityForm) OnMount(ctx app.Context) {
	af.resetForm()
}

// Render renders the ability list and form
func (af *AbilityForm) Render() app.UI {
	return app.Div().
		Class("ability-form").
		Body(
			app.Div().
				Class("ability-header").
				Body(
					app.Label().Text("Features/Abilities:"),
					app.Button().
						Class("btn btn-small").
						Text("+ Add Ability").
						OnClick(func(ctx app.Context, e app.Event) {
							af.showForm = !af.showForm
							ctx.Update()
						}),
				),
			af.renderAbilityList(),
			app.If(af.showForm, func() app.UI {
				return af.renderAddForm()
			}),
		)
}

// renderAbilityList shows current abilities
func (af *AbilityForm) renderAbilityList() app.UI {
	if len(af.Abilities) == 0 {
		return app.Div().
			Class("empty-abilities").
			Text("No abilities added")
	}

	items := make([]app.UI, len(af.Abilities))
	for i, ability := range af.Abilities {
		idx := i
		items[i] = af.renderAbilityItem(ability, idx)
	}

	return app.Div().
		Class("ability-list").
		Body(items...)
}

// renderAbilityItem renders a single ability
func (af *AbilityForm) renderAbilityItem(ability *models.Ability, idx int) app.UI {
	damageEffect := af.getDamageEffect(ability)
	damageText := ""
	if damageEffect != nil {
		damageText = fmt.Sprintf("%dd%d", damageEffect.DiceCount, damageEffect.DiceSize)
		if damageEffect.FlatBonus > 0 {
			damageText += fmt.Sprintf("+%d", damageEffect.FlatBonus)
		}
		damageText += fmt.Sprintf(" %s", damageEffect.DamageType.String())
	}

	targeting := "Single"
	if ability.TargetAOE {
		targeting = fmt.Sprintf("AOE (%d targets)", ability.AOEMaxTargets)
	}

	return app.Div().
		Class("ability-item").
		Body(
			app.Div().
				Class("ability-info").
				Body(
					app.Strong().Text(ability.Name),
					app.If(damageText != "", func() app.UI {
						return app.Span().Text(fmt.Sprintf(" | %s | %s", damageText, targeting))
					}),
					app.If(ability.Description != "", func() app.UI {
						return app.Div().Class("ability-desc").Text(ability.Description)
					}),
				),
			app.Button().
				Class("btn btn-danger btn-small").
				Text("Remove").
				OnClick(func(ctx app.Context, e app.Event) {
					af.removeAbility(ctx, idx)
				}),
		)
}

// getDamageEffect extracts damage effect from ability
func (af *AbilityForm) getDamageEffect(ability *models.Ability) *models.DamageEffect {
	for _, effect := range ability.Effects {
		if dmg, ok := effect.(models.DamageEffect); ok {
			return &dmg
		}
	}
	return nil
}

// renderAddForm renders the ability creation form
func (af *AbilityForm) renderAddForm() app.UI {
	return app.Div().
		Class("ability-add-form").
		Body(
			app.H4().Text("Add Ability"),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Name:"),
					app.Input().
						Type("text").
						Value(af.newName).
						Placeholder("Fireball").
						OnInput(func(ctx app.Context, e app.Event) {
							af.newName = ctx.JSSrc().Get("value").String()
						}),
				),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Description (optional):"),
					app.Input().
						Type("text").
						Value(af.newDescription).
						Placeholder("Hurl a ball of flame at your foes").
						OnInput(func(ctx app.Context, e app.Event) {
							af.newDescription = ctx.JSSrc().Get("value").String()
						}),
				),
			app.Div().
				Class("form-row").
				Body(
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Dice Count:"),
							app.Input().
								Type("number").
								Value(af.newDiceCount).
								Min(0).
								Max(10).
								OnInput(af.handleIntInput(&af.newDiceCount)),
						),
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Dice Size:"),
							app.Select().
								OnChange(func(ctx app.Context, e app.Event) {
									af.newDiceSize = ctx.JSSrc().Get("valueAsNumber").Int()
								}).
								Body(
									app.Option().Value("4").Text("d4").Selected(af.newDiceSize == 4),
									app.Option().Value("6").Text("d6").Selected(af.newDiceSize == 6),
									app.Option().Value("8").Text("d8").Selected(af.newDiceSize == 8),
									app.Option().Value("10").Text("d10").Selected(af.newDiceSize == 10),
									app.Option().Value("12").Text("d12").Selected(af.newDiceSize == 12),
									app.Option().Value("20").Text("d20").Selected(af.newDiceSize == 20),
								),
						),
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Flat Bonus:"),
							app.Input().
								Type("number").
								Value(af.newFlatBonus).
								Min(0).
								Max(20).
								OnInput(af.handleIntInput(&af.newFlatBonus)),
						),
				),
			app.Div().
				Class("form-row").
				Body(
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Damage Type:"),
							app.Select().
								OnChange(func(ctx app.Context, e app.Event) {
									af.newDamageType = ctx.JSSrc().Get("value").String()
								}).
								Body(
									app.Option().Value("Physical").Text("Physical").Selected(af.newDamageType == "Physical"),
									app.Option().Value("Magic").Text("Magic").Selected(af.newDamageType == "Magic"),
									app.Option().Value("Fire").Text("Fire").Selected(af.newDamageType == "Fire"),
									app.Option().Value("Frost").Text("Frost").Selected(af.newDamageType == "Frost"),
									app.Option().Value("Poison").Text("Poison").Selected(af.newDamageType == "Poison"),
								),
						),
				),
			app.Div().
				Class("form-row").
				Body(
					app.Div().
						Class("form-group").
						Body(
							app.Label().
								Body(
									app.Input().
										Type("checkbox").
										Checked(af.newIsAOE).
										OnChange(func(ctx app.Context, e app.Event) {
											af.newIsAOE = ctx.JSSrc().Get("checked").Bool()
											ctx.Update()
										}),
									app.Text(" AOE (Area of Effect)"),
								),
						),
					app.If(af.newIsAOE, func() app.UI {
						return app.Div().
							Class("form-group").
							Body(
								app.Label().Text("Max Targets:"),
								app.Input().
									Type("number").
									Value(af.newAOETargets).
									Min(2).
									Max(10).
									OnInput(af.handleIntInput(&af.newAOETargets)),
							)
					}),
				),
			app.Div().
				Class("form-actions").
				Body(
					app.Button().
						Class("btn btn-primary").
						Text("Add Ability").
						Disabled(af.newName == "").
						OnClick(func(ctx app.Context, e app.Event) {
							af.addAbility(ctx)
						}),
					app.Button().
						Class("btn btn-secondary").
						Text("Cancel").
						OnClick(func(ctx app.Context, e app.Event) {
							af.showForm = false
							af.resetForm()
							ctx.Update()
						}),
				),
		)
}

// handleIntInput creates an input handler for integer fields
func (af *AbilityForm) handleIntInput(target *int) func(app.Context, app.Event) {
	return func(ctx app.Context, e app.Event) {
		value := ctx.JSSrc().Get("valueAsNumber").Int()
		*target = value
	}
}

// addAbility creates and adds a new ability
func (af *AbilityForm) addAbility(ctx app.Context) {
	damageType := models.DamagePhysical
	switch af.newDamageType {
	case "Magic":
		damageType = models.DamageMagic
	case "Fire":
		damageType = models.DamageFire
	case "Frost":
		damageType = models.DamageFrost
	case "Poison":
		damageType = models.DamagePoison
	}

	ability := &models.Ability{
		Name:          af.newName,
		Description:   af.newDescription,
		Trigger:       models.TriggerManual,
		Conditions:    []models.AbilityCondition{},
		Costs:         models.ResourceCost{},
		TargetSingle:  !af.newIsAOE,
		TargetAOE:     af.newIsAOE,
		AOEMaxTargets: af.newAOETargets,
		TargetEnemies: true,
		Effects: []models.Effect{
			models.DamageEffect{
				DiceCount:  af.newDiceCount,
				DiceSize:   af.newDiceSize,
				FlatBonus:  af.newFlatBonus,
				DamageType: damageType,
			},
		},
	}

	af.Abilities = append(af.Abilities, ability)

	if af.OnChange != nil {
		af.OnChange()
	}

	af.showForm = false
	af.resetForm()
	ctx.Update()
}

// removeAbility removes an ability by index
func (af *AbilityForm) removeAbility(ctx app.Context, idx int) {
	if idx >= 0 && idx < len(af.Abilities) {
		af.Abilities = append(af.Abilities[:idx], af.Abilities[idx+1:]...)

		if af.OnChange != nil {
			af.OnChange()
		}

		ctx.Update()
	}
}

// resetForm resets the form to default values
func (af *AbilityForm) resetForm() {
	af.newName = ""
	af.newDescription = ""
	af.newDiceCount = 1
	af.newDiceSize = 6
	af.newFlatBonus = 0
	af.newDamageType = "Physical"
	af.newIsAOE = false
	af.newAOETargets = 2
}
