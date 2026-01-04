package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// CombatantList displays and manages a list of combatants
type CombatantList struct {
	app.Compo

	Title      string
	Combatants []*models.Combatant
	Side       models.Side
	OnAdd      func(app.Context, *models.Combatant)
	OnRemove   func(app.Context, int)

	// Form state for adding new combatant
	showForm       bool
	newName        string
	newEvasion     int
	newProficiency int
	newHP          int
	newStress      int
	newArmor       int
}

// OnMount initializes default values
func (cl *CombatantList) OnMount(ctx app.Context) {
	cl.resetForm()
}

// Render renders the combatant list
func (cl *CombatantList) Render() app.UI {
	return app.Div().
		Class("combatant-list").
		Body(
			app.Div().
				Class("list-header").
				Body(
					app.H3().Text(cl.Title),
					app.Button().
						Class("btn btn-small").
						Text("+ Add").
						OnClick(func(ctx app.Context, e app.Event) {
							cl.showForm = !cl.showForm
						}),
				),
			app.If(cl.showForm, func() app.UI {
				return cl.renderAddForm()
			}),
			cl.renderList(),
		)
}

// renderList renders the list of combatants
func (cl *CombatantList) renderList() app.UI {
	if len(cl.Combatants) == 0 {
		return app.Div().
			Class("empty-state").
			Text("No combatants added yet")
	}

	items := make([]app.UI, len(cl.Combatants))
	for i, combatant := range cl.Combatants {
		idx := i // Capture for closure
		items[i] = app.Div().
			Class("combatant-item").
			Body(
				app.Div().
					Class("combatant-info").
					Body(
						app.Strong().Text(combatant.Name),
						app.Span().Text(fmt.Sprintf(" | HP: %d | Evasion: %d | Prof: %d",
							combatant.HPMax, combatant.Evasion, combatant.Proficiency)),
						app.If(combatant.ArmorSlots > 0, func() app.UI {
							return app.Span().Text(fmt.Sprintf(" | Armor: %d", combatant.ArmorSlots))
						}),
					),
				app.Button().
					Class("btn btn-danger btn-small").
					Text("Remove").
					OnClick(func(ctx app.Context, e app.Event) {
						if cl.OnRemove != nil {
							cl.OnRemove(ctx, idx)
						}
					}),
			)
	}

	return app.Div().Class("combatant-items").Body(items...)
}

// renderAddForm renders the form for adding a new combatant
func (cl *CombatantList) renderAddForm() app.UI {
	return app.Div().
		Class("add-form").
		Body(
			app.H4().Text("Add New Combatant"),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Name:"),
					app.Input().
						Type("text").
						Value(cl.newName).
						Placeholder("Character name").
						OnInput(func(ctx app.Context, e app.Event) {
							cl.newName = ctx.JSSrc().Get("value").String()
						}),
				),
			app.Div().
				Class("form-row").
				Body(
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("HP Max:"),
							app.Input().
								Type("number").
								Value(cl.newHP).
								Min(1).
								Max(50).
								OnInput(cl.handleIntInput(&cl.newHP)),
						),
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Evasion:"),
							app.Input().
								Type("number").
								Value(cl.newEvasion).
								Min(1).
								Max(20).
								OnInput(cl.handleIntInput(&cl.newEvasion)),
						),
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Proficiency:"),
							app.Input().
								Type("number").
								Value(cl.newProficiency).
								Min(0).
								Max(5).
								OnInput(cl.handleIntInput(&cl.newProficiency)),
						),
				),
			app.Div().
				Class("form-row").
				Body(
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Stress Max:"),
							app.Input().
								Type("number").
								Value(cl.newStress).
								Min(1).
								Max(10).
								OnInput(cl.handleIntInput(&cl.newStress)),
						),
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Armor Slots:"),
							app.Input().
								Type("number").
								Value(cl.newArmor).
								Min(0).
								Max(5).
								OnInput(cl.handleIntInput(&cl.newArmor)),
						),
				),
			app.Div().
				Class("form-actions").
				Body(
					app.Button().
						Class("btn btn-primary").
						Text("Add Combatant").
						Disabled(cl.newName == "").
						OnClick(func(ctx app.Context, e app.Event) {
							cl.addCombatant(ctx)
						}),
					app.Button().
						Class("btn btn-secondary").
						Text("Cancel").
						OnClick(func(ctx app.Context, e app.Event) {
							cl.showForm = false
							cl.resetForm()
						}),
				),
		)
}

// handleIntInput creates an input handler for integer fields
func (cl *CombatantList) handleIntInput(target *int) func(app.Context, app.Event) {
	return func(ctx app.Context, e app.Event) {
		value := ctx.JSSrc().Get("valueAsNumber").Int()
		*target = value
	}
}

// addCombatant creates a new combatant and adds it to the list
func (cl *CombatantList) addCombatant(ctx app.Context) {
	combatant := models.NewCombatant(
		fmt.Sprintf("%s-%d", cl.Side, len(cl.Combatants)),
		cl.newName,
		cl.Side,
		cl.newEvasion,
		cl.newProficiency,
		cl.newHP,
		cl.newStress,
	)
	combatant.ArmorSlots = cl.newArmor

	if cl.OnAdd != nil {
		cl.OnAdd(ctx, combatant)
	}

	cl.showForm = false
	cl.resetForm()
}

// resetForm resets the form to default values
func (cl *CombatantList) resetForm() {
	cl.newName = ""
	cl.newEvasion = 10
	cl.newProficiency = 1
	cl.newHP = 5
	cl.newStress = 6
	cl.newArmor = 0
}
