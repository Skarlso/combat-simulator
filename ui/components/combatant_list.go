package components

import (
	"fmt"
	"net/url"

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

	showForm           bool
	sectionOpen        map[string]bool
	newName            string
	newLevel           int
	newEvasion         int
	newProficiency     int
	newAttackTrait     int
	newTraits          models.Traits
	newHP              int
	newStress          int
	newArmor           int
	newDamageDice      string
	newMinorThreshold  int
	newMajorThreshold  int
	newHealthPotions   int
	newStaminaPotions  int
	newDomainAbilities []*models.DomainAbility

	importer   importPanel
	exportText string
}

// OnMount initializes default values
func (cl *CombatantList) OnMount(ctx app.Context) {
	cl.resetForm()
}

// Render renders the combatant list
func (cl *CombatantList) Render() app.UI {
	return app.Div().Class("combatant-list").Body(
		app.Div().Class("list-header").Body(
			app.H3().Text(cl.Title),
			app.Button().Class("btn btn-small").Text("+ Add").OnClick(func(ctx app.Context, e app.Event) {
				cl.showForm = !cl.showForm
				ctx.Update()
			}),
			app.Button().Class("btn btn-small").Text("Import JSON").OnClick(func(ctx app.Context, e app.Event) {
				cl.importer.toggle(ctx)
			}),
			app.Button().Class("btn btn-small").Text("Export JSON").Disabled(len(cl.Combatants) == 0).OnClick(func(ctx app.Context, e app.Event) {
				cl.toggleExport(ctx)
			}),
		),
		app.If(cl.exportText != "", func() app.UI { return cl.renderExportPanel() }),
		app.If(cl.importer.show, func() app.UI {
			return cl.importer.render(
				"Import Characters from JSON",
				`Paste a JSON document matching {"characters":[...]}. Each entry requires a name; level, evasion, proficiency, attack_trait, traits, hp, stress, thresholds, and domain_abilities are optional.`,
				`{"characters": [{"name": "...", "level": 3, ...}]}`,
				cl.runImport,
			)
		}),
		app.If(cl.showForm, func() app.UI { return cl.renderAddForm() }),
		cl.renderList(),
	)
}

func (cl *CombatantList) runImport(ctx app.Context) {
	load := func(raw []byte) ([]*models.Combatant, error) {
		imported, err := parser.LoadCharactersFromBytes(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		return imported, nil
	}
	cl.importer.run(ctx, "characters", load, func(ctx app.Context, c *models.Combatant) {
		c.Side = cl.Side
		if cl.OnAdd != nil {
			cl.OnAdd(ctx, c)
		}
	})
}

func (cl *CombatantList) toggleExport(ctx app.Context) {
	if cl.exportText != "" {
		cl.exportText = ""
		ctx.Update()
		return
	}
	raw, err := parser.ExportCharacters(cl.Combatants)
	if err != nil {
		app.Log("party export failed:", err)
		return
	}
	cl.exportText = string(raw)
	ctx.Update()
}

// renderExportPanel shows the party as import-ready JSON, to copy or download
func (cl *CombatantList) renderExportPanel() app.UI {
	return app.Div().Class("add-form").Body(
		app.H4().Text("Party as JSON"),
		app.P().Text("Paste this into Import JSON later to restore the party, or download it."),
		app.Div().Class("form-group").Body(
			app.Textarea().
				Style("width", "100%").
				Style("min-height", "200px").
				Style("font-family", "monospace").
				ReadOnly(true).
				Rows(12).
				Text(cl.exportText),
		),
		app.Div().Class("form-actions").Body(
			app.Button().Class("btn btn-primary").Text("Download party.json").OnClick(func(ctx app.Context, e app.Event) {
				downloadText("party.json", cl.exportText)
			}),
			app.Button().Class("btn btn-secondary").Text("Close").OnClick(func(ctx app.Context, e app.Event) {
				cl.exportText = ""
				ctx.Update()
			}),
		),
	)
}

// downloadText saves text as a file through a temporary data-URL link
func downloadText(filename, text string) {
	doc := app.Window().Get("document")
	link := doc.Call("createElement", "a")
	link.Set("href", "data:application/json;charset=utf-8,"+url.PathEscape(text))
	link.Set("download", filename)
	doc.Get("body").Call("appendChild", link)
	link.Call("click")
	doc.Get("body").Call("removeChild", link)
}

// renderList renders the list of combatants
func (cl *CombatantList) renderList() app.UI {
	if len(cl.Combatants) == 0 {
		return app.Div().Class("empty-state").Text("No combatants added yet")
	}

	items := make([]app.UI, len(cl.Combatants))
	for i, combatant := range cl.Combatants {
		items[i] = app.Div().Class("combatant-item").Body(
			app.Div().Class("combatant-info").Body(
				app.Div().Class("item-title").Body(
					app.Text(combatant.Name),
					app.Span().Class("tag tag-muted").Text(fmt.Sprintf("Level %d", combatant.Level)),
				),
				app.Div().Class("chip-row").Body(
					statChip("HP", fmt.Sprintf("%d", combatant.HPMax)),
					statChip("Stress", fmt.Sprintf("%d", combatant.StressMax)),
					statChip("Evasion", fmt.Sprintf("%d", combatant.Evasion)),
					statChip("Prof", fmt.Sprintf("%d", combatant.Proficiency)),
					statChip("Trait", fmt.Sprintf("%+d", combatant.AttackTrait)),
					app.If(combatant.ArmorSlots > 0, func() app.UI {
						return statChip("Armor", fmt.Sprintf("%d", combatant.ArmorSlots))
					}),
					app.If(len(combatant.DomainAbilities) > 0, func() app.UI {
						return statChip("Abilities", fmt.Sprintf("%d", len(combatant.DomainAbilities)))
					}),
					app.If(combatant.HealthPotions+combatant.StaminaPotions > 0, func() app.UI {
						return statChip("Potions", fmt.Sprintf("%d health, %d stamina", combatant.HealthPotions, combatant.StaminaPotions))
					}),
				),
			),
			app.Div().Class("item-actions").Body(
				app.Button().Class("btn btn-small").Text("Duplicate").OnClick(func(ctx app.Context, e app.Event) {
					cl.duplicateCombatant(ctx, i)
				}),
				app.Button().Class("btn btn-danger btn-small").Text("Remove").OnClick(func(ctx app.Context, e app.Event) {
					if cl.OnRemove != nil {
						cl.OnRemove(ctx, i)
					}
				}),
			),
		)
	}

	return app.Div().Class("combatant-items").Body(items...)
}

func (cl *CombatantList) section(key, label string, body ...app.UI) []app.UI {
	return accordion(label, cl.sectionOpen[key], func(ctx app.Context, e app.Event) {
		cl.sectionOpen[key] = !cl.sectionOpen[key]
		ctx.Update()
	}, body...)
}

// renderAddForm renders the form for adding a new combatant
func (cl *CombatantList) renderAddForm() app.UI {
	body := []app.UI{app.H4().Text("Add New Combatant")}
	body = append(body, cl.section("identity", "Identity",
		textField("Name", "Character name", &cl.newName, false),
		app.Div().Class("form-row").Body(numberField("Level", &cl.newLevel, 1, 10)),
	)...)
	body = append(body, cl.section("stats", "Stats",
		app.Div().Class("form-row-3").Body(
			numberField("HP Max", &cl.newHP, 1, 50),
			numberField("Evasion", &cl.newEvasion, 1, 20),
			numberField("Proficiency", &cl.newProficiency, 0, 5),
			numberField("Attack Trait", &cl.newAttackTrait, -5, 10),
		),
		app.Div().Class("form-row-3").Body(
			numberField("Agility", &cl.newTraits.Agility, -5, 10),
			numberField("Strength", &cl.newTraits.Strength, -5, 10),
			numberField("Finesse", &cl.newTraits.Finesse, -5, 10),
		),
		app.Div().Class("form-row-3").Body(
			numberField("Instinct", &cl.newTraits.Instinct, -5, 10),
			numberField("Presence", &cl.newTraits.Presence, -5, 10),
			numberField("Knowledge", &cl.newTraits.Knowledge, -5, 10),
		),
		app.Div().Class("form-row-3").Body(
			numberField("Stress Max", &cl.newStress, 1, 10),
			numberField("Armor Slots", &cl.newArmor, 0, 5),
		),
	)...)
	body = append(body, cl.section("combat", "Combat",
		textField("Damage Dice (e.g., 2d6+2, 1d6+1d6+4)", "e.g., 2d6+2 or 1d6+1d4+3", &cl.newDamageDice, false),
		app.Div().Class("form-row").Body(
			numberField("Minor Threshold", &cl.newMinorThreshold, 1, 20),
			numberField("Major Threshold", &cl.newMajorThreshold, 1, 30),
		),
		app.Div().Class("form-row").Body(
			numberField("Health Potions", &cl.newHealthPotions, 0, models.MaxPotionsEach),
			numberField("Stamina Potions", &cl.newStaminaPotions, 0, models.MaxPotionsEach),
		),
	)...)
	body = append(body, cl.section("abilities", "Abilities",
		&DomainAbilityForm{
			Abilities: cl.newDomainAbilities,
			OnChange:  func(updated []*models.DomainAbility) { cl.newDomainAbilities = updated },
		},
	)...)
	body = append(body, formActions("Add Combatant", cl.newName == "",
		func(ctx app.Context, e app.Event) { cl.addCombatant(ctx) },
		func(ctx app.Context, e app.Event) {
			cl.showForm = false
			cl.resetForm()
			ctx.Update()
		},
	))

	return app.Div().Class("add-form").Body(body...)
}

func (cl *CombatantList) duplicateCombatant(ctx app.Context, idx int) {
	if idx < 0 || idx >= len(cl.Combatants) {
		return
	}
	original := cl.Combatants[idx]
	clone := models.CloneCombatant(original, fmt.Sprintf("%s-%d", cl.Side, len(cl.Combatants)))
	clone.Name = original.Name + " (Copy)"
	if cl.OnAdd != nil {
		cl.OnAdd(ctx, clone)
	}
}

// addCombatant creates a new combatant and adds it to the list
func (cl *CombatantList) addCombatant(ctx app.Context) {
	combatant := models.NewCombatant(
		fmt.Sprintf("%s-%d", cl.Side, len(cl.Combatants)),
		cl.newName, cl.Side, cl.newLevel, cl.newEvasion, cl.newProficiency, cl.newHP, cl.newStress,
	)
	combatant.AttackTrait = cl.newAttackTrait
	combatant.Traits = cl.newTraits
	combatant.ArmorSlots = cl.newArmor
	combatant.DamageDice = cl.newDamageDice
	combatant.CustomThresholds = []int{cl.newMinorThreshold, cl.newMajorThreshold}
	combatant.HealthPotions = cl.newHealthPotions
	combatant.StaminaPotions = cl.newStaminaPotions
	for _, ability := range cl.newDomainAbilities {
		combatant.DomainAbilities = append(combatant.DomainAbilities, *ability)
	}
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
	cl.newAttackTrait = 2
	cl.newTraits = models.Traits{}
	cl.newHP = 5
	cl.newStress = 6
	cl.newArmor = 0
	cl.newDamageDice = "1d6"
	cl.newMinorThreshold = 4
	cl.newMajorThreshold = 7
	cl.newHealthPotions = 0
	cl.newStaminaPotions = 0
	cl.newDomainAbilities = make([]*models.DomainAbility, 0)
	cl.sectionOpen = map[string]bool{"identity": true}
}
