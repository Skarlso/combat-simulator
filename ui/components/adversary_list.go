package components

import (
	"fmt"
	"slices"
	"strings"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/combat-simulator/data"
	"github.com/skarlso/combat-simulator/internal/models"
	"github.com/skarlso/combat-simulator/internal/parser"
)

// AdversaryList displays and manages a list of adversaries
type AdversaryList struct {
	app.Compo

	Title       string
	Adversaries []*models.Combatant
	Registry    *parser.AdversaryRegistry
	OnAdd       func(app.Context, *models.Combatant)
	OnUpdate    func(app.Context, int, *models.Combatant)
	OnRemove    func(app.Context, int)

	// Library picker state
	showLibrary   bool
	librarySearch string
	libraryTier   string
	libraryType   string
	libraryAdds   int

	showForm bool
	// Row the form is editing; -1 while it builds a new adversary
	editingIndex        int
	sectionOpen         map[string]bool
	newName             string
	newTier             string
	newType             string
	newDifficulty       int
	newATK              int
	newHP               int
	newStress           int
	newMinorThreshold   int
	newMajorThreshold   int
	newDamageDice       string
	newMotives          string
	newFeatures         []*models.AdversaryFeature
	newIsSummonTemplate bool

	importer importPanel
}

// OnMount initializes default values
func (al *AdversaryList) OnMount(ctx app.Context) {
	al.resetForm()
}

// Render renders the adversary list
func (al *AdversaryList) Render() app.UI {
	return app.Div().Class("adversary-list").Body(
		app.Div().Class("list-header").Body(
			app.H3().Text(al.Title),
			app.Button().Class(toggleClass("btn btn-small", al.showLibrary, "active")).Text("Library").OnClick(func(ctx app.Context, e app.Event) {
				al.showLibrary = !al.showLibrary
				ctx.Update()
			}),
			app.Button().Class(toggleClass("btn btn-small", al.showForm && al.editingIndex < 0, "active")).Text("+ Custom").OnClick(func(ctx app.Context, e app.Event) {
				// From an edit, "+ Custom" switches the open form to a blank one
				if al.editingIndex >= 0 {
					al.resetForm()
					al.showForm = true
				} else {
					al.showForm = !al.showForm
				}
				ctx.Update()
			}),
			app.Button().Class(toggleClass("btn btn-small", al.importer.show, "active")).Text("Import JSON").OnClick(func(ctx app.Context, e app.Event) {
				al.importer.toggle(ctx)
			}),
		),
		app.If(al.showLibrary, func() app.UI { return al.renderLibraryPanel() }),
		app.If(al.importer.show, func() app.UI {
			return al.importer.render(
				"Import Adversaries from JSON",
				`Paste a JSON document matching {"adversaries":[...]} (same schema as data/daggerheart_adversaries-tier-1.json).`,
				`{"adversaries": [{"name": "...", "tier": 1, ...}]}`,
				al.runImport,
			)
		}),
		app.If(al.showForm, func() app.UI { return al.renderAddForm() }),
		al.renderList(),
	)
}

var adversaryTypes = []string{"Bruiser", "Horde", "Leader", "Minion", "Ranged", "Skulk", "Social", "Solo", "Standard", "Support"}

// renderLibraryPanel renders the searchable SRD stat block picker
func (al *AdversaryList) renderLibraryPanel() app.UI {
	if al.Registry == nil {
		return app.Div().Class("library-panel").Body(
			app.P().Class("empty-state").Text("Adversary library failed to load."),
		)
	}

	templates := al.Registry.Templates()
	query := strings.ToLower(strings.TrimSpace(al.librarySearch))
	rows := make([]app.UI, 0, len(templates))
	for _, t := range templates {
		if al.libraryTier != "" && t.Tier != al.libraryTier {
			continue
		}
		if al.libraryType != "" && t.AdversaryType != al.libraryType {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(t.Name), query) && !strings.Contains(strings.ToLower(t.Motives), query) {
			continue
		}
		rows = append(rows, al.renderLibraryRow(t))
	}

	tierChips := []app.UI{al.renderTierChip("", "All")}
	for i := 1; i <= 4; i++ {
		tierChips = append(tierChips, al.renderTierChip(fmt.Sprintf("Tier %d", i), fmt.Sprintf("T%d", i)))
	}

	typeOptions := []selectOption{{"", "All types"}}
	for _, t := range adversaryTypes {
		typeOptions = append(typeOptions, selectOption{t, t})
	}

	var list app.UI = app.Div().Class("empty-state").Text("No stat blocks match.")
	if len(rows) > 0 {
		list = app.Div().Class("library-list").Body(rows...)
	}

	return app.Div().Class("library-panel").Body(
		app.Div().Class("library-filters").Body(
			app.Input().
				Type("search").
				Placeholder("Search name or tactics").
				Value(al.librarySearch).
				OnInput(func(ctx app.Context, e app.Event) {
					al.librarySearch = ctx.JSSrc().Get("value").String()
					ctx.Update()
				}),
			app.Div().Class("tabs tabs-compact").Body(tierChips...),
			app.Select().
				OnChange(func(ctx app.Context, e app.Event) {
					al.libraryType = selectedOption(ctx, typeOptions)
					ctx.Update()
				}).
				Body(optionElements(typeOptions, al.libraryType)...),
		),
		app.Div().Class("library-count").Text(fmt.Sprintf("%d of %d stat blocks", len(rows), len(templates))),
		list,
	)
}

func (al *AdversaryList) renderTierChip(value, label string) app.UI {
	return app.Button().
		Class(toggleClass("tab", al.libraryTier == value, "active")).
		Text(label).
		OnClick(func(ctx app.Context, e app.Event) {
			al.libraryTier = value
			ctx.Update()
		}, app.EventScope(value))
}

func (al *AdversaryList) renderLibraryRow(t *models.Combatant) app.UI {
	return app.Div().Class("library-row").Body(
		app.Div().Class("library-main").Body(
			app.Div().Class("item-title").Body(
				app.Text(t.Name),
				app.Span().Class("tag").Text(fmt.Sprintf("%s %s", t.Tier, t.AdversaryType)),
			),
			app.Div().Class("chip-row").Body(
				statChip("Diff", fmt.Sprintf("%d", t.Difficulty)),
				statChip("HP", fmt.Sprintf("%d", t.HPMax)),
				statChip("Stress", fmt.Sprintf("%d", t.StressMax)),
				statChip("ATK", fmt.Sprintf("%+d", t.ATK)),
				statChip("Dmg", t.DamageDice),
				statChip("Thresholds", thresholdLabel(t.CustomThresholds)),
				app.If(len(t.AdversaryFeatures) > 0, func() app.UI {
					return statChip("Modelled", fmt.Sprintf("%d", len(t.AdversaryFeatures)))
				}),
			),
		),
		// go-app matches handlers by function pointer, so a captured name goes stale
		// when filtering reorders rows; the DOM attribute is always current.
		app.Button().
			Class("btn btn-small btn-primary").
			Text("Add").
			DataSet("name", t.Name).
			OnClick(func(ctx app.Context, e app.Event) {
				al.addFromLibrary(ctx, ctx.JSSrc().Get("dataset").Get("name").String())
			}),
	)
}

func (al *AdversaryList) addFromLibrary(ctx app.Context, name string) {
	al.libraryAdds++
	clone, err := al.Registry.CloneTemplate(name, fmt.Sprintf("lib-%d-%s", al.libraryAdds, strings.ToLower(strings.ReplaceAll(name, " ", "-"))))
	if err != nil {
		app.Log("library add failed:", err)
		return
	}
	if al.OnAdd != nil {
		al.OnAdd(ctx, clone)
	}
	ctx.Update()
}

func (al *AdversaryList) runImport(ctx app.Context) {
	load := func(raw []byte) ([]*models.Combatant, error) {
		mappings, err := data.GetFeatureMappings()
		if err != nil {
			return nil, fmt.Errorf("failed to load feature mappings: %w", err)
		}
		ap, err := parser.NewAdversaryParserFromBytes(mappings)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize parser: %w", err)
		}
		imported, err := ap.LoadAdversariesFromBytes(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		return imported, nil
	}
	al.importer.run(ctx, "adversaries", load, al.OnAdd)
}

// renderList renders the list of adversaries
func (al *AdversaryList) renderList() app.UI {
	if len(al.Adversaries) == 0 {
		return app.Div().Class("empty-state").Text("No adversaries added yet")
	}

	items := make([]app.UI, len(al.Adversaries))
	for i, adversary := range al.Adversaries {
		items[i] = app.Div().Class(toggleClass("adversary-item combatant-item", al.editingIndex == i, "editing")).Body(
			app.Div().Class("adversary-info").Body(
				app.Div().Class("item-title").Body(
					app.Text(adversary.Name),
					app.Span().Class("tag").Text(fmt.Sprintf("%s %s", adversary.Tier, adversary.AdversaryType)),
					app.If(adversary.IsSummonTemplate, func() app.UI {
						return app.Span().Class("tag tag-muted").Text("Template")
					}),
				),
				app.Div().Class("chip-row").Body(
					statChip("Diff", fmt.Sprintf("%d", adversary.Difficulty)),
					statChip("HP", fmt.Sprintf("%d", adversary.HPMax)),
					statChip("Stress", fmt.Sprintf("%d", adversary.StressMax)),
					statChip("ATK", fmt.Sprintf("%+d", adversary.ATK)),
					statChip("Thresholds", thresholdLabel(adversary.CustomThresholds)),
					app.If(len(adversary.AdversaryFeatures) > 0, func() app.UI {
						return statChip("Features", fmt.Sprintf("%d", len(adversary.AdversaryFeatures)))
					}),
				),
			),
			app.Div().Class("item-actions").Body(
				app.Button().Class("btn btn-small").Text("Edit").OnClick(func(ctx app.Context, e app.Event) {
					al.startEditing(ctx, i)
				}),
				app.Button().Class("btn btn-small").Text("Duplicate").OnClick(func(ctx app.Context, e app.Event) {
					al.duplicateAdversary(ctx, i)
				}),
				app.Button().Class("btn btn-danger btn-small").Text("Remove").OnClick(func(ctx app.Context, e app.Event) {
					if al.editingIndex == i {
						al.showForm = false
						al.resetForm()
					}
					al.editingIndex = indexAfterRemoval(al.editingIndex, i)
					if al.OnRemove != nil {
						al.OnRemove(ctx, i)
					}
				}),
			),
		)
	}

	return app.Div().Class("adversary-items").Body(items...)
}

func (al *AdversaryList) section(key, label string, body ...app.UI) []app.UI {
	return accordion(label, al.sectionOpen[key], func(ctx app.Context, e app.Event) {
		al.sectionOpen[key] = !al.sectionOpen[key]
		ctx.Update()
	}, body...)
}

var (
	tierOptions = []selectOption{{"Tier 1", "Tier 1"}, {"Tier 2", "Tier 2"}, {"Tier 3", "Tier 3"}, {"Tier 4", "Tier 4"}, {"Tier 5", "Tier 5"}}
	// Every SRD adversary type, so a library stat block can be edited without losing its own
	customTypeOptions = []selectOption{{"Minion", "Minion"}, {"Standard", "Standard"}, {"Elite", "Elite"}, {"Solo", "Solo"}, {"Horde", "Horde"},
		{"Bruiser", "Bruiser"}, {"Leader", "Leader"}, {"Ranged", "Ranged"}, {"Skulk", "Skulk"}, {"Social", "Social"}, {"Support", "Support"}}
	summonTemplateLabel = "Summon Template Only (won't fight unless summoned)"
)

// renderAddForm renders the form for adding a new adversary
func (al *AdversaryList) renderAddForm() app.UI {
	title, saveText := "Add New Adversary", "Add Adversary"
	if al.editingIndex >= 0 {
		title, saveText = "Edit "+al.Adversaries[al.editingIndex].Name, "Save Changes"
	}
	body := []app.UI{app.H4().Text(title)}
	body = append(body, al.section("identity", "Identity",
		textField("Name", "Ancient Witch", &al.newName, false),
		app.Div().Class("form-row").Body(
			selectField("Tier", al.newTier, tierOptions, bindSelect(&al.newTier, false)),
			selectField("Type", al.newType, customTypeOptions, bindSelect(&al.newType, false)),
		),
	)...)
	body = append(body, al.section("stats", "Stats",
		app.Div().Class("form-row").Body(
			numberField("Difficulty", &al.newDifficulty, 1, 25),
			numberField("ATK", &al.newATK, 0, 10),
		),
		app.Div().Class("form-row").Body(
			numberField("HP Max", &al.newHP, 1, 100),
			numberField("Stress Max", &al.newStress, 1, 20),
		),
		app.Div().Class("form-row").Body(
			numberField("Minor Threshold", &al.newMinorThreshold, 1, 30),
			numberField("Major Threshold", &al.newMajorThreshold, 1, 50),
		),
	)...)
	body = append(body, al.section("combat", "Combat",
		textField("Damage Dice (e.g., 2d6+2, 1d6+1d6+4)", "e.g., 2d6+4 or 1d8+1d4+2", &al.newDamageDice, false),
		app.Div().Class("form-group").Body(
			app.Label().Text("Motives & Tactics"),
			app.Textarea().
				Text(al.newMotives).
				Placeholder("Manipulate, curse, deceive...").
				Rows(3).
				OnInput(func(ctx app.Context, e app.Event) { al.newMotives = ctx.JSSrc().Get("value").String() }),
		),
		checkboxField(summonTemplateLabel, &al.newIsSummonTemplate),
	)...)
	body = append(body, al.section("features", "Features",
		&AdversaryFeatureForm{
			Features:             al.newFeatures,
			AvailableAdversaries: al.Adversaries,
			OnChange:             func(updated []*models.AdversaryFeature) { al.newFeatures = updated },
		},
	)...)
	body = append(body, formActions(saveText, al.newName == "" || al.newTier == "" || al.newType == "",
		func(ctx app.Context, e app.Event) { al.saveAdversary(ctx) },
		func(ctx app.Context, e app.Event) {
			al.showForm = false
			al.resetForm()
			ctx.Update()
		},
	))

	return app.Div().Class("add-form").ID(adversaryFormID).Body(body...)
}

const adversaryFormID = "adversary-form"

// startEditing opens the form on an existing adversary. Features are copied, so nothing
// changes until the form is saved.
func (al *AdversaryList) startEditing(ctx app.Context, idx int) {
	if idx < 0 || idx >= len(al.Adversaries) {
		return
	}
	adv := al.Adversaries[idx]
	al.resetForm()
	al.editingIndex = idx
	al.showForm = true
	al.sectionOpen = map[string]bool{"identity": true, "stats": true, "combat": true, "features": true}

	al.newName = adv.Name
	al.newTier, al.newType = adv.Tier, adv.AdversaryType
	al.newDifficulty, al.newATK = adv.Difficulty, adv.ATK
	al.newHP, al.newStress = adv.HPMax, adv.StressMax
	al.newMinorThreshold, al.newMajorThreshold = adv.Thresholds()
	al.newDamageDice = adv.DamageDice
	al.newMotives = adv.Motives
	al.newIsSummonTemplate = adv.IsSummonTemplate
	for _, feature := range adv.AdversaryFeatures {
		feature.Effects = slices.Clone(feature.Effects)
		al.newFeatures = append(al.newFeatures, &feature)
	}

	ctx.Update()
	ctx.ScrollTo(adversaryFormID)
}

func (al *AdversaryList) duplicateAdversary(ctx app.Context, idx int) {
	if idx < 0 || idx >= len(al.Adversaries) {
		return
	}
	original := al.Adversaries[idx]
	clone := models.CloneCombatant(original, fmt.Sprintf("enemy-%d", len(al.Adversaries)))
	clone.Name = original.Name + " (Copy)"
	if al.OnAdd != nil {
		al.OnAdd(ctx, clone)
	}
}

// saveAdversary adds the form's adversary, or replaces the one being edited with it
func (al *AdversaryList) saveAdversary(ctx app.Context) {
	editing := al.editingIndex >= 0 && al.editingIndex < len(al.Adversaries)
	adversary := models.NewAdversary(
		fmt.Sprintf("enemy-%d", len(al.Adversaries)),
		al.newName, al.newTier, al.newType, al.newDifficulty, al.newATK, al.newHP, al.newStress,
		[]int{al.newMinorThreshold, al.newMajorThreshold},
	)
	if editing {
		// Start from the original so its ID, Experiences and spotlight limit are kept
		original := *al.Adversaries[al.editingIndex]
		adversary = &original
		adversary.Name, adversary.Tier, adversary.AdversaryType = al.newName, al.newTier, al.newType
		adversary.Difficulty, adversary.ATK = al.newDifficulty, al.newATK
		adversary.HP, adversary.HPMax, adversary.StressMax = al.newHP, al.newHP, al.newStress
		adversary.CustomThresholds = []int{al.newMinorThreshold, al.newMajorThreshold}
		adversary.AdversaryFeatures, adversary.Resistances = nil, nil
	}
	adversary.DamageDice = al.newDamageDice
	adversary.Motives = al.newMotives
	adversary.Tactics = al.newMotives
	adversary.IsSummonTemplate = al.newIsSummonTemplate
	for _, feature := range al.newFeatures {
		adversary.AdversaryFeatures = append(adversary.AdversaryFeatures, *feature)
	}
	adversary.ApplyPassiveResistances()

	switch {
	case editing && al.OnUpdate != nil:
		al.OnUpdate(ctx, al.editingIndex, adversary)
	case !editing && al.OnAdd != nil:
		al.OnAdd(ctx, adversary)
	}
	al.showForm = false
	al.resetForm()
	ctx.Update()
}

// resetForm resets the form to default values
func (al *AdversaryList) resetForm() {
	al.editingIndex = -1
	al.newName = ""
	al.newTier = "Tier 1"
	al.newType = "Standard"
	al.newDifficulty = 12
	al.newATK = 2
	al.newHP = 7
	al.newStress = 5
	al.newDamageDice = "1d8"
	al.newMinorThreshold = 8
	al.newMajorThreshold = 16
	al.newMotives = ""
	al.newFeatures = make([]*models.AdversaryFeature, 0)
	al.newIsSummonTemplate = false
	al.sectionOpen = map[string]bool{"identity": true}
}

// thresholdLabel renders [Major, Severe]; 0 is the stat block's "None"
func thresholdLabel(thresholds []int) string {
	if len(thresholds) < 2 {
		return "default"
	}
	one := func(v int) string {
		if v <= 0 {
			return "None"
		}
		return fmt.Sprintf("%d", v)
	}
	return one(thresholds[0]) + "/" + one(thresholds[1])
}
