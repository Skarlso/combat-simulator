package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// AdversaryFeatureForm manages adding/editing adversary features (enemy abilities)
type AdversaryFeatureForm struct {
	app.Compo

	Features []*models.AdversaryFeature
	OnChange func()

	// Form state
	showForm         bool
	newName          string
	newDescription   string
	newPassive       bool
	newFearCost      int
	newSpotlightLim  int
	newDiceFormula   string
	newDamageType    string
	newEffectType    string
	newMagnitude     int
	newDuration      string
}

// OnMount initializes default values
func (aff *AdversaryFeatureForm) OnMount(ctx app.Context) {
	aff.resetForm()
}

// Render renders the feature list and form
func (aff *AdversaryFeatureForm) Render() app.UI {
	return app.Div().
		Class("adversary-feature-form").
		Body(
			app.Div().
				Class("ability-header").
				Body(
					app.Label().Text("Adversary Features:"),
					app.Button().
						Class("btn btn-small").
						Text("+ Add Feature").
						OnClick(func(ctx app.Context, e app.Event) {
							aff.showForm = !aff.showForm
							ctx.Update()
						}),
				),
			aff.renderFeatureList(),
			app.If(aff.showForm, func() app.UI {
				return aff.renderAddForm()
			}),
		)
}

// renderFeatureList shows current features
func (aff *AdversaryFeatureForm) renderFeatureList() app.UI {
	if len(aff.Features) == 0 {
		return app.Div().
			Class("empty-abilities").
			Text("No features added")
	}

	items := make([]app.UI, len(aff.Features))
	for i := range aff.Features {
		idx := i
		items[i] = aff.renderFeatureItem(aff.Features[idx], idx)
	}

	return app.Div().
		Class("ability-list").
		Body(items...)
}

// renderFeatureItem renders a single feature
func (aff *AdversaryFeatureForm) renderFeatureItem(feature *models.AdversaryFeature, idx int) app.UI {
	typeText := "Active"
	if feature.Passive {
		typeText = "Passive"
	}

	costText := ""
	if feature.FearCost > 0 {
		costText += fmt.Sprintf("%d Fear", feature.FearCost)
	}
	if feature.SpotlightLimit > 0 {
		if costText != "" {
			costText += " | "
		}
		costText += fmt.Sprintf("Max %d/turn", feature.SpotlightLimit)
	}
	if costText == "" {
		costText = "No cost"
	}

	effectText := ""
	if len(feature.Effects) > 0 {
		effect := feature.Effects[0]
		switch effect.Category {
		case models.FeatureEffectDamage, models.FeatureEffectDirectDamage:
			effectText = fmt.Sprintf("%s %s damage", effect.DiceFormula, effect.DamageType)
		case models.FeatureEffectStressInflict:
			effectText = fmt.Sprintf("Inflict %d stress", effect.Magnitude)
		case models.FeatureEffectApplyPoison:
			effectText = "Apply poison"
		case models.FeatureEffectApplyVulnerable:
			effectText = "Apply vulnerable"
		default:
			effectText = string(effect.Category)
		}
	}

	return app.Div().
		Class("ability-item").
		Body(
			app.Div().
				Class("ability-info").
				Body(
					app.Strong().Text(feature.Name),
					app.Span().Text(fmt.Sprintf(" | %s | %s", typeText, costText)),
					app.If(effectText != "", func() app.UI {
						return app.Div().Class("ability-effect").Text("Effect: " + effectText)
					}),
					app.If(feature.Description != "", func() app.UI {
						return app.Div().Class("ability-desc").Text(feature.Description)
					}),
				),
			app.Button().
				Class("btn btn-danger btn-small").
				Text("Remove").
				OnClick(func(ctx app.Context, e app.Event) {
					aff.removeFeature(ctx, idx)
				}),
		)
}

// renderAddForm renders the feature creation form
func (aff *AdversaryFeatureForm) renderAddForm() app.UI {
	return app.Div().
		Class("ability-add-form").
		Body(
			app.H4().Text("Add Adversary Feature"),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Name:"),
					app.Input().
						Type("text").
						Value(aff.newName).
						Placeholder("Venomous Stinger").
						OnInput(func(ctx app.Context, e app.Event) {
							aff.newName = ctx.JSSrc().Get("value").String()
						}),
				),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Description (optional):"),
					app.Input().
						Type("text").
						Value(aff.newDescription).
						Placeholder("Injects venom...").
						OnInput(func(ctx app.Context, e app.Event) {
							aff.newDescription = ctx.JSSrc().Get("value").String()
						}),
				),
			app.Div().
				Class("form-group").
				Body(
					app.Label().
						Body(
							app.Input().
								Type("checkbox").
								Checked(aff.newPassive).
								OnChange(func(ctx app.Context, e app.Event) {
									aff.newPassive = ctx.JSSrc().Get("checked").Bool()
									ctx.Update()
								}),
							app.Text(" Passive (always on)"),
						),
				),
			app.Div().
				Class("form-row").
				Body(
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Fear Cost:"),
							app.Input().
								Type("number").
								Value(aff.newFearCost).
								Min(0).
								Max(10).
								OnInput(aff.handleIntInput(&aff.newFearCost)),
						),
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Spotlight Limit (per turn):"),
							app.Input().
								Type("number").
								Value(aff.newSpotlightLim).
								Min(0).
								Max(5).
								OnInput(aff.handleIntInput(&aff.newSpotlightLim)),
						),
				),
			app.H5().Text("Effect"),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Effect Type:"),
					app.Select().
						OnChange(func(ctx app.Context, e app.Event) {
							aff.newEffectType = ctx.JSSrc().Get("value").String()
							ctx.Update()
						}).
						Body(
							app.Option().Value("damage").Text("Damage").Selected(aff.newEffectType == "damage"),
							app.Option().Value("direct_damage").Text("Direct Damage").Selected(aff.newEffectType == "direct_damage"),
							app.Option().Value("stress_inflict").Text("Inflict Stress").Selected(aff.newEffectType == "stress_inflict"),
							app.Option().Value("apply_vulnerable").Text("Apply Vulnerable").Selected(aff.newEffectType == "apply_vulnerable"),
							app.Option().Value("apply_poison").Text("Apply Poison").Selected(aff.newEffectType == "apply_poison"),
							app.Option().Value("armor_damage").Text("Damage Armor").Selected(aff.newEffectType == "armor_damage"),
						),
				),
			app.If(aff.needsDiceFormula(), func() app.UI {
				return app.Div().
					Class("form-group").
					Body(
						app.Label().Text("Dice Formula (e.g., 2d6+3):"),
						app.Input().
							Type("text").
							Value(aff.newDiceFormula).
							Placeholder("1d8").
							OnInput(func(ctx app.Context, e app.Event) {
								aff.newDiceFormula = ctx.JSSrc().Get("value").String()
							}),
					)
			}),
			app.If(aff.needsMagnitude(), func() app.UI {
				return app.Div().
					Class("form-group").
					Body(
						app.Label().Text("Amount:"),
						app.Input().
							Type("number").
							Value(aff.newMagnitude).
							Min(1).
							Max(10).
							OnInput(aff.handleIntInput(&aff.newMagnitude)),
					)
			}),
			app.If(aff.needsDamageType(), func() app.UI {
				return app.Div().
					Class("form-group").
					Body(
						app.Label().Text("Damage Type:"),
						app.Select().
							OnChange(func(ctx app.Context, e app.Event) {
								aff.newDamageType = ctx.JSSrc().Get("value").String()
							}).
							Body(
								app.Option().Value("physical").Text("Physical").Selected(aff.newDamageType == "physical"),
								app.Option().Value("magic").Text("Magic").Selected(aff.newDamageType == "magic"),
							),
					)
			}),
			app.If(aff.needsDuration(), func() app.UI {
				return app.Div().
					Class("form-group").
					Body(
						app.Label().Text("Duration:"),
						app.Select().
							OnChange(func(ctx app.Context, e app.Event) {
								aff.newDuration = ctx.JSSrc().Get("value").String()
							}).
							Body(
								app.Option().Value("instant").Text("Instant").Selected(aff.newDuration == "instant"),
								app.Option().Value("1_turn").Text("1 Turn").Selected(aff.newDuration == "1_turn"),
								app.Option().Value("until_cleared").Text("Until Cleared").Selected(aff.newDuration == "until_cleared"),
								app.Option().Value("permanent").Text("Permanent").Selected(aff.newDuration == "permanent"),
							),
					)
			}),
			app.Div().
				Class("form-actions").
				Body(
					app.Button().
						Class("btn btn-primary").
						Text("Add Feature").
						Disabled(aff.newName == "").
						OnClick(func(ctx app.Context, e app.Event) {
							aff.addFeature(ctx)
						}),
					app.Button().
						Class("btn btn-secondary").
						Text("Cancel").
						OnClick(func(ctx app.Context, e app.Event) {
							aff.showForm = false
							aff.resetForm()
							ctx.Update()
						}),
				),
		)
}

// Helper methods for conditional rendering
func (aff *AdversaryFeatureForm) needsDiceFormula() bool {
	return aff.newEffectType == "damage" || aff.newEffectType == "direct_damage"
}

func (aff *AdversaryFeatureForm) needsMagnitude() bool {
	return aff.newEffectType == "stress_inflict" || aff.newEffectType == "armor_damage"
}

func (aff *AdversaryFeatureForm) needsDamageType() bool {
	return aff.newEffectType == "damage" || aff.newEffectType == "direct_damage"
}

func (aff *AdversaryFeatureForm) needsDuration() bool {
	return aff.newEffectType == "apply_vulnerable" || aff.newEffectType == "apply_poison"
}

// handleIntInput creates an input handler for integer fields
func (aff *AdversaryFeatureForm) handleIntInput(target *int) func(app.Context, app.Event) {
	return func(ctx app.Context, e app.Event) {
		value := ctx.JSSrc().Get("valueAsNumber").Int()
		*target = value
	}
}

// addFeature creates and adds a new feature
func (aff *AdversaryFeatureForm) addFeature(ctx app.Context) {
	// Create the effect based on type
	var effectCategory models.FeatureEffectCategory
	switch aff.newEffectType {
	case "damage":
		effectCategory = models.FeatureEffectDamage
	case "direct_damage":
		effectCategory = models.FeatureEffectDirectDamage
	case "stress_inflict":
		effectCategory = models.FeatureEffectStressInflict
	case "apply_vulnerable":
		effectCategory = models.FeatureEffectApplyVulnerable
	case "apply_poison":
		effectCategory = models.FeatureEffectApplyPoison
	case "armor_damage":
		effectCategory = models.FeatureEffectArmorDamage
	default:
		effectCategory = models.FeatureEffectDamage
	}

	effect := models.FeatureEffect{
		Category:    effectCategory,
		Magnitude:   aff.newMagnitude,
		DiceFormula: aff.newDiceFormula,
		Duration:    aff.newDuration,
		TargetType:  models.FeatureTargetSingle,
		DamageType:  aff.newDamageType,
	}

	feature := &models.AdversaryFeature{
		Name:        aff.newName,
		Description: aff.newDescription,
		Passive:     aff.newPassive,
		Trigger: models.FeatureTriggerCondition{
			Type: "on_attack",
		},
		FearCost:       aff.newFearCost,
		SpotlightLimit: aff.newSpotlightLim,
		Effects:        []models.FeatureEffect{effect},
	}

	aff.Features = append(aff.Features, feature)

	if aff.OnChange != nil {
		aff.OnChange()
	}

	aff.showForm = false
	aff.resetForm()
	ctx.Update()
}

// removeFeature removes a feature by index
func (aff *AdversaryFeatureForm) removeFeature(ctx app.Context, idx int) {
	if idx >= 0 && idx < len(aff.Features) {
		aff.Features = append(aff.Features[:idx], aff.Features[idx+1:]...)

		if aff.OnChange != nil {
			aff.OnChange()
		}

		ctx.Update()
	}
}

// resetForm resets the form to default values
func (aff *AdversaryFeatureForm) resetForm() {
	aff.newName = ""
	aff.newDescription = ""
	aff.newPassive = true
	aff.newFearCost = 0
	aff.newSpotlightLim = 0
	aff.newDiceFormula = "1d6"
	aff.newDamageType = "physical"
	aff.newEffectType = "damage"
	aff.newMagnitude = 1
	aff.newDuration = "instant"
}
