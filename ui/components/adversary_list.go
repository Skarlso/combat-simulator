package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// AdversaryList displays and manages a list of adversaries
type AdversaryList struct {
	app.Compo

	Title       string
	Adversaries []*models.Combatant
	OnAdd       func(app.Context, *models.Combatant)
	OnRemove    func(app.Context, int)

	// Form state for adding new adversary
	showForm           bool
	newName            string
	newTier            string
	newType            string
	newDifficulty      int
	newATK             int
	newHP              int
	newStress          int
	newMinorThreshold  int
	newMajorThreshold  int
	newDamageDice      string
	newMotives         string
	newTactics         string
	newFeatures        []*models.AdversaryFeature
	newIsSummonTemplate bool
}

// OnMount initializes default values
func (al *AdversaryList) OnMount(ctx app.Context) {
	al.resetForm()
}

// Render renders the adversary list
func (al *AdversaryList) Render() app.UI {
	return app.Div().
		Class("adversary-list").
		Body(
			app.Div().
				Class("list-header").
				Body(
					app.H3().Text(al.Title),
					app.Button().
						Class("btn btn-small").
						Text("+ Add").
						OnClick(func(ctx app.Context, e app.Event) {
							al.showForm = !al.showForm
							ctx.Update()
						}),
				),
			app.If(al.showForm, func() app.UI {
				return al.renderAddForm()
			}),
			al.renderList(),
		)
}

// renderList renders the list of adversaries
func (al *AdversaryList) renderList() app.UI {
	if len(al.Adversaries) == 0 {
		return app.Div().
			Class("empty-state").
			Text("No adversaries added yet")
	}

	items := make([]app.UI, len(al.Adversaries))
	for i, adversary := range al.Adversaries {
		idx := i
		templateTag := ""
		if adversary.IsSummonTemplate {
			templateTag = " [TEMPLATE]"
		}
		items[i] = app.Div().
			Class("adversary-item").
			Body(
				app.Div().
					Class("adversary-info").
					Body(
						app.Strong().Text(fmt.Sprintf("%s %s - %s%s", adversary.Tier, adversary.AdversaryType, adversary.Name, templateTag)),
						app.Br(),
						app.Span().Text(fmt.Sprintf("Difficulty: %d | Thresholds: %d/%d | HP: %d | Stress: %d | ATK: +%d",
							adversary.Difficulty,
							adversary.CustomThresholds[0],
							adversary.CustomThresholds[1],
							adversary.HPMax,
							adversary.StressMax,
							adversary.ATK)),
					),
				app.Button().
					Class("btn btn-danger btn-small").
					Text("Remove").
					OnClick(func(ctx app.Context, e app.Event) {
						if al.OnRemove != nil {
							al.OnRemove(ctx, idx)
						}
					}),
			)
	}

	return app.Div().Class("adversary-items").Body(items...)
}

// renderAddForm renders the form for adding a new adversary
func (al *AdversaryList) renderAddForm() app.UI {
	return app.Div().
		Class("add-form").
		Body(
			app.H4().Text("Add New Adversary"),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Name:"),
					app.Input().
						Type("text").
						Value(al.newName).
						Placeholder("Ancient Witch").
						OnInput(func(ctx app.Context, e app.Event) {
							al.newName = ctx.JSSrc().Get("value").String()
						}),
				),
			app.Div().
				Class("form-row").
				Body(
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Tier:"),
							app.Select().
								OnChange(func(ctx app.Context, e app.Event) {
									al.newTier = ctx.JSSrc().Get("value").String()
								}).
								Body(
									app.Option().Value("Tier 1").Text("Tier 1").Selected(al.newTier == "Tier 1"),
									app.Option().Value("Tier 2").Text("Tier 2").Selected(al.newTier == "Tier 2"),
									app.Option().Value("Tier 3").Text("Tier 3").Selected(al.newTier == "Tier 3"),
									app.Option().Value("Tier 4").Text("Tier 4").Selected(al.newTier == "Tier 4"),
									app.Option().Value("Tier 5").Text("Tier 5").Selected(al.newTier == "Tier 5"),
								),
						),
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Type:"),
							app.Select().
								OnChange(func(ctx app.Context, e app.Event) {
									al.newType = ctx.JSSrc().Get("value").String()
								}).
								Body(
									app.Option().Value("Minion").Text("Minion").Selected(al.newType == "Minion"),
									app.Option().Value("Standard").Text("Standard").Selected(al.newType == "Standard"),
									app.Option().Value("Elite").Text("Elite").Selected(al.newType == "Elite"),
									app.Option().Value("Solo").Text("Solo").Selected(al.newType == "Solo"),
									app.Option().Value("Horde").Text("Horde").Selected(al.newType == "Horde"),
								),
						),
				),
			app.Div().
				Class("form-row").
				Body(
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Difficulty:"),
							app.Input().
								Type("number").
								Value(al.newDifficulty).
								Min(1).
								Max(25).
								OnInput(al.handleIntInput(&al.newDifficulty)),
						),
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("ATK:"),
							app.Input().
								Type("number").
								Value(al.newATK).
								Min(0).
								Max(10).
								OnInput(al.handleIntInput(&al.newATK)),
						),
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
								Value(al.newHP).
								Min(1).
								Max(100).
								OnInput(al.handleIntInput(&al.newHP)),
						),
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Stress Max:"),
							app.Input().
								Type("number").
								Value(al.newStress).
								Min(1).
								Max(20).
								OnInput(al.handleIntInput(&al.newStress)),
						),
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
								Value(al.newMinorThreshold).
								Min(1).
								Max(30).
								OnInput(al.handleIntInput(&al.newMinorThreshold)),
						),
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Major Threshold:"),
							app.Input().
								Type("number").
								Value(al.newMajorThreshold).
								Min(1).
								Max(50).
								OnInput(al.handleIntInput(&al.newMajorThreshold)),
						),
				),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Damage Dice (e.g., 2d6+2, 1d6+1d6+4):"),
					app.Input().
						Type("text").
						Value(al.newDamageDice).
						Placeholder("e.g., 2d6+4 or 1d8+1d4+2").
						OnInput(func(ctx app.Context, e app.Event) {
							al.newDamageDice = ctx.JSSrc().Get("value").String()
						}),
				),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Motives & Tactics:"),
					app.Textarea().
						Text(al.newMotives).
						Placeholder("Manipulate, curse, deceive...").
						Rows(3).
						OnInput(func(ctx app.Context, e app.Event) {
							text := ctx.JSSrc().Get("value").String()
							al.newMotives = text
							al.newTactics = text
						}),
				),
			app.Div().
				Class("form-group").
				Body(
					app.Label().
						Class("checkbox-label").
						Body(
							app.Input().
								Type("checkbox").
								Checked(al.newIsSummonTemplate).
								OnChange(func(ctx app.Context, e app.Event) {
									al.newIsSummonTemplate = ctx.JSSrc().Get("checked").Bool()
									ctx.Update()
								}),
							app.Text(" Summon Template Only (won't fight unless summoned)"),
						),
				),
			&AdversaryFeatureForm{
				Features:             al.newFeatures,
				AvailableAdversaries: al.Adversaries,
				OnChange: func(updated []*models.AdversaryFeature) {
					al.newFeatures = updated
				},
			},
			app.Div().
				Class("form-actions").
				Body(
					app.Button().
						Class("btn btn-primary").
						Text("Add Adversary").
						Disabled(al.newName == "" || al.newTier == "" || al.newType == "").
						OnClick(func(ctx app.Context, e app.Event) {
							al.addAdversary(ctx)
						}),
					app.Button().
						Class("btn btn-secondary").
						Text("Cancel").
						OnClick(func(ctx app.Context, e app.Event) {
							al.showForm = false
							al.resetForm()
							ctx.Update()
						}),
				),
		)
}

// handleIntInput creates an input handler for integer fields
func (al *AdversaryList) handleIntInput(target *int) func(app.Context, app.Event) {
	return func(ctx app.Context, e app.Event) {
		value := ctx.JSSrc().Get("valueAsNumber").Int()
		*target = value
	}
}

// addAdversary creates a new adversary and adds it to the list
func (al *AdversaryList) addAdversary(ctx app.Context) {
	adversary := models.NewAdversary(
		fmt.Sprintf("enemy-%d", len(al.Adversaries)),
		al.newName,
		al.newTier,
		al.newType,
		al.newDifficulty,
		al.newATK,
		al.newHP,
		al.newStress,
		[]int{al.newMinorThreshold, al.newMajorThreshold},
	)
	adversary.DamageDice = al.newDamageDice
	adversary.Motives = al.newMotives
	adversary.Tactics = al.newTactics
	adversary.IsSummonTemplate = al.newIsSummonTemplate

	for _, feature := range al.newFeatures {
		adversary.AdversaryFeatures = append(adversary.AdversaryFeatures, *feature)
	}

	adversary.ApplyPassiveResistances()

	if al.OnAdd != nil {
		al.OnAdd(ctx, adversary)
	}

	al.showForm = false
	al.resetForm()
	ctx.Update()
}

// resetForm resets the form to default values
func (al *AdversaryList) resetForm() {
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
	al.newTactics = ""
	al.newFeatures = make([]*models.AdversaryFeature, 0)
	al.newIsSummonTemplate = false
}
