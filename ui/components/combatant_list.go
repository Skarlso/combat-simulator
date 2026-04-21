package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/parser"
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
	showForm    bool
	sectionOpen map[string]bool
	newName     string
	newLevel           int
	newEvasion         int
	newProficiency     int
	newHP              int
	newStress          int
	newArmor           int
	newDamageDice      string
	newMinorThreshold  int
	newMajorThreshold  int
	newDomainAbilities []*models.DomainAbility

	showImport  bool
	importText  string
	importError string
	importInfo  string
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
							ctx.Update()
						}),
					app.Button().
						Class("btn btn-small").
						Text("Import JSON").
						OnClick(func(ctx app.Context, e app.Event) {
							cl.showImport = !cl.showImport
							cl.importError = ""
							cl.importInfo = ""
							ctx.Update()
						}),
				),
			app.If(cl.showImport, func() app.UI {
				return cl.renderImportPanel()
			}),
			app.If(cl.showForm, func() app.UI {
				return cl.renderAddForm()
			}),
			cl.renderList(),
		)
}

func (cl *CombatantList) renderImportPanel() app.UI {
	return app.Div().
		Class("add-form").
		Body(
			app.H4().Text("Import Characters from JSON"),
			app.P().Text(`Paste a JSON document matching {"characters":[...]}. Each entry requires a name; level, evasion, proficiency, hp, stress, thresholds, and domain_abilities are optional.`),
			app.Div().
				Class("form-group").
				Body(
					app.Textarea().
						Style("width", "100%").
						Style("min-height", "200px").
						Style("font-family", "monospace").
						Placeholder(`{"characters": [{"name": "...", "level": 3, ...}]}`).
						Rows(12).
						Text(cl.importText).
						OnInput(func(ctx app.Context, e app.Event) {
							cl.importText = ctx.JSSrc().Get("value").String()
						}),
				),
			app.If(cl.importError != "", func() app.UI {
				return app.Div().Class("validation-error").Text(cl.importError)
			}),
			app.If(cl.importInfo != "", func() app.UI {
				return app.Div().Class("validation-info").Text(cl.importInfo)
			}),
			app.Div().
				Class("form-actions").
				Body(
					app.Button().
						Class("btn btn-primary").
						Text("Import").
						OnClick(func(ctx app.Context, e app.Event) {
							cl.runImport(ctx)
						}),
					app.Button().
						Class("btn btn-secondary").
						Text("Close").
						OnClick(func(ctx app.Context, e app.Event) {
							cl.showImport = false
							ctx.Update()
						}),
				),
		)
}

func (cl *CombatantList) runImport(ctx app.Context) {
	cl.importError = ""
	cl.importInfo = ""

	if cl.importText == "" {
		cl.importError = "Paste JSON before importing."
		ctx.Update()
		return
	}

	imported, err := parser.LoadCharactersFromBytes([]byte(cl.importText))
	if err != nil {
		cl.importError = fmt.Sprintf("invalid JSON: %v", err)
		ctx.Update()
		return
	}

	if len(imported) == 0 {
		cl.importError = "no characters found in JSON"
		ctx.Update()
		return
	}

	for _, c := range imported {
		c.Side = cl.Side
	}

	if cl.OnAdd != nil {
		for _, c := range imported {
			cl.OnAdd(ctx, c)
		}
	}

	cl.importInfo = fmt.Sprintf("Imported %d characters.", len(imported))
	cl.importText = ""
	ctx.Update()
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
		idx := i
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
				app.Div().
					Class("item-actions").
					Body(
						app.Button().
							Class("btn btn-small").
							Text("Duplicate").
							OnClick(func(ctx app.Context, e app.Event) {
								cl.duplicateCombatant(ctx, idx)
							}),
						app.Button().
							Class("btn btn-danger btn-small").
							Text("Remove").
							OnClick(func(ctx app.Context, e app.Event) {
								if cl.OnRemove != nil {
									cl.OnRemove(ctx, idx)
								}
							}),
					),
			)
	}

	return app.Div().Class("combatant-items").Body(items...)
}

func (cl *CombatantList) toggleSection(ctx app.Context, name string) {
	cl.sectionOpen[name] = !cl.sectionOpen[name]
	ctx.Update()
}

// renderAddForm renders the form for adding a new combatant
func (cl *CombatantList) renderAddForm() app.UI {
	return app.Div().
		Class("add-form").
		Body(
			app.H4().Text("Add New Combatant"),
			renderAccordionHeader("Identity", cl.sectionOpen["identity"], func(ctx app.Context, e app.Event) {
				cl.toggleSection(ctx, "identity")
			}),
			app.If(cl.sectionOpen["identity"], func() app.UI {
				return app.Div().Class("accordion-body").Body(
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
									app.Label().Text("Level:"),
									app.Input().
										Type("number").
										Value(cl.newLevel).
										Min(1).
										Max(10).
										OnInput(handleIntInput(&cl.newLevel)),
								),
						),
				)
			}),
			renderAccordionHeader("Stats", cl.sectionOpen["stats"], func(ctx app.Context, e app.Event) {
				cl.toggleSection(ctx, "stats")
			}),
			app.If(cl.sectionOpen["stats"], func() app.UI {
				return app.Div().Class("accordion-body").Body(
					app.Div().
						Class("form-row-3").
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
										OnInput(handleIntInput(&cl.newHP)),
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
										OnInput(handleIntInput(&cl.newEvasion)),
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
										OnInput(handleIntInput(&cl.newProficiency)),
								),
						),
					app.Div().
						Class("form-row-3").
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
										OnInput(handleIntInput(&cl.newStress)),
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
										OnInput(handleIntInput(&cl.newArmor)),
								),
						),
				)
			}),
			renderAccordionHeader("Combat", cl.sectionOpen["combat"], func(ctx app.Context, e app.Event) {
				cl.toggleSection(ctx, "combat")
			}),
			app.If(cl.sectionOpen["combat"], func() app.UI {
				return app.Div().Class("accordion-body").Body(
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Damage Dice (e.g., 2d6+2, 1d6+1d6+4):"),
							app.Input().
								Type("text").
								Value(cl.newDamageDice).
								Placeholder("e.g., 2d6+2 or 1d6+1d4+3").
								OnInput(func(ctx app.Context, e app.Event) {
									cl.newDamageDice = ctx.JSSrc().Get("value").String()
								}),
						),
					app.Div().
						Class("form-row").
						Body(
							app.Div().
								Class("form-group").
								Body(
									app.Label().Text("Minor Threshold:"),
									app.Input().
										Type("number").
										Value(cl.newMinorThreshold).
										Min(1).
										Max(20).
										OnInput(handleIntInput(&cl.newMinorThreshold)),
								),
							app.Div().
								Class("form-group").
								Body(
									app.Label().Text("Major Threshold:"),
									app.Input().
										Type("number").
										Value(cl.newMajorThreshold).
										Min(1).
										Max(30).
										OnInput(handleIntInput(&cl.newMajorThreshold)),
								),
						),
				)
			}),
			renderAccordionHeader("Abilities", cl.sectionOpen["abilities"], func(ctx app.Context, e app.Event) {
				cl.toggleSection(ctx, "abilities")
			}),
			app.If(cl.sectionOpen["abilities"], func() app.UI {
				return app.Div().Class("accordion-body").Body(
					&DomainAbilityForm{
						Abilities: cl.newDomainAbilities,
						OnChange: func(updated []*models.DomainAbility) {
							cl.newDomainAbilities = updated
						},
					},
				)
			}),
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
							ctx.Update()
						}),
				),
		)
}


func (cl *CombatantList) duplicateCombatant(ctx app.Context, idx int) {
	if idx < 0 || idx >= len(cl.Combatants) {
		return
	}
	original := cl.Combatants[idx]
	newID := fmt.Sprintf("%s-%d", cl.Side, len(cl.Combatants))
	clone := models.CloneCombatant(original, newID)
	clone.Name = original.Name + " (Copy)"
	if cl.OnAdd != nil {
		cl.OnAdd(ctx, clone)
	}
}

// addCombatant creates a new combatant and adds it to the list
func (cl *CombatantList) addCombatant(ctx app.Context) {
	combatant := models.NewCombatant(
		fmt.Sprintf("%s-%d", cl.Side, len(cl.Combatants)),
		cl.newName,
		cl.Side,
		cl.newLevel,
		cl.newEvasion,
		cl.newProficiency,
		cl.newHP,
		cl.newStress,
	)
	combatant.ArmorSlots = cl.newArmor
	combatant.DamageDice = cl.newDamageDice
	combatant.CustomThresholds = []int{cl.newMinorThreshold, cl.newMajorThreshold}

	// Copy domain abilities (need to dereference pointers)
	for _, ability := range cl.newDomainAbilities {
		combatant.DomainAbilities = append(combatant.DomainAbilities, *ability)
	}

	// Apply passive resistances from features/abilities
	combatant.ApplyPassiveResistances()

	if cl.OnAdd != nil {
		cl.OnAdd(ctx, combatant)
	}

	cl.showForm = false
	cl.resetForm()
	ctx.Update()
}

// resetForm resets the form to default values
func (cl *CombatantList) resetForm() {
	cl.newName = ""
	cl.newLevel = 1
	cl.newEvasion = 10
	cl.newProficiency = 1
	cl.newHP = 5
	cl.newStress = 6
	cl.newArmor = 0
	cl.newDamageDice = "1d6"
	cl.newMinorThreshold = 4
	cl.newMajorThreshold = 7
	cl.newDomainAbilities = make([]*models.DomainAbility, 0)
	cl.sectionOpen = map[string]bool{
		"identity":  true,
		"stats":     false,
		"combat":    false,
		"abilities": false,
	}
}
