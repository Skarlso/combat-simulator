package components

import (
	"fmt"
	"sort"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/data"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/parser"
)

const customLibraryOption = "__custom__"

// AdversaryFeatureForm manages adding/editing adversary features (enemy abilities)
type AdversaryFeatureForm struct {
	app.Compo

	Features             []*models.AdversaryFeature
	OnChange             func([]*models.AdversaryFeature)
	AvailableAdversaries []*models.Combatant

	// Form state
	showForm      bool
	editingIndex  int
	newName       string
	newDescription    string
	newPassive        bool
	newFearCost       int
	newStressCost     int
	newCostType       string
	newSpotlightLim   int
	newUsageType      string
	newMaxUses        int
	newDiceFormula    string
	newDamageType     string
	newEffectType     string
	newMagnitude      int
	newArmorCost      int
	newFallbackFear   int
	newReactionTrait  string
	newReactionDC     int
	newHalfOnSuccess  bool
	newTrigger        models.FeatureTriggerType
	newDuration       string
	newDCType         string
	newFixedDC        int
	newQuantityType   string
	selectedForSummon map[string]bool

	newUsesTokens bool
	newTokenName  string
	newTokensMax  int

	featureLibrary     *parser.FeatureParser
	libraryFeatureKeys []string
	selectedLibraryKey string
}

// OnMount initializes default values
func (aff *AdversaryFeatureForm) OnMount(ctx app.Context) {
	aff.resetForm()
	aff.loadFeatureLibrary()
}

func (aff *AdversaryFeatureForm) loadFeatureLibrary() {
	raw, err := data.GetFeatureMappings()
	if err != nil {
		return
	}
	fp, err := parser.NewFeatureParserFromBytes(raw)
	if err != nil {
		return
	}
	aff.featureLibrary = fp
	aff.libraryFeatureKeys = fp.FeatureNames()
	sort.Strings(aff.libraryFeatureKeys)
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
	if feature.StressCost > 0 {
		if costText != "" {
			costText += ", "
		}
		costText += fmt.Sprintf("%d Stress", feature.StressCost)
	}
	if feature.SpotlightLimit > 0 {
		if costText != "" {
			costText += " | "
		}
		costText += fmt.Sprintf("Max %d/turn", feature.SpotlightLimit)
	}
	if feature.MaxUses > 0 {
		if costText != "" {
			costText += " | "
		}
		costText += fmt.Sprintf("%d uses (%s)", feature.MaxUses, feature.UsageType)
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
		case models.FeatureEffectSummonAllies:
			count := len(effect.SummonPool)
			if effect.DiceFormula != "" {
				effectText = fmt.Sprintf("Summon %s from %d templates", effect.DiceFormula, count)
			} else {
				effectText = fmt.Sprintf("Summon %d from %d templates", effect.Magnitude, count)
			}
		default:
			effectText = string(effect.Category)
		}
	}

	itemClass := "ability-item"
	if aff.editingIndex == idx {
		itemClass += " editing"
	}

	return app.Div().
		Class(itemClass).
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
			app.Div().
				Class("item-actions").
				Body(
					app.Button().
						Class("btn btn-small").
						Text("Edit").
						OnClick(func(ctx app.Context, e app.Event) {
							aff.startEditing(ctx, idx)
						}),
					app.Button().
						Class("btn btn-danger btn-small").
						Text("Remove").
						OnClick(func(ctx app.Context, e app.Event) {
							aff.removeFeature(ctx, idx)
						}),
				),
		)
}

// renderAddForm renders the feature creation form
func (aff *AdversaryFeatureForm) renderAddForm() app.UI {
	formTitle := "Add Adversary Feature"
	saveText := "Add Feature"
	if aff.editingIndex >= 0 {
		formTitle = "Edit Adversary Feature"
		saveText = "Save"
	}

	diceErr := validateDiceNotation(aff.newDiceFormula)

	return app.Div().
		Class("ability-add-form").
		Body(
			app.H4().Text(formTitle),
			app.If(aff.editingIndex < 0 && len(aff.libraryFeatureKeys) > 0, func() app.UI {
				return aff.renderLibraryPicker()
			}),
			app.Div().
				Class("form-row").
				Body(
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
							app.Label().Text("Description:"),
							app.Input().
								Type("text").
								Value(aff.newDescription).
								Placeholder("Injects venom...").
								OnInput(func(ctx app.Context, e app.Event) {
									aff.newDescription = ctx.JSSrc().Get("value").String()
								}),
						),
				),
			app.Div().
				Class("form-group").
				Body(
					app.Label().
						Class("checkbox-label").
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
			app.Div().Class("form-subsection").Body(
				app.H5().Text("Cost & Usage"),
				app.Div().
					Class("form-row-3").
					Body(
						app.Div().
							Class("form-group").
							Body(
								app.Label().Text("Cost Type:"),
								app.Select().
									OnChange(func(ctx app.Context, e app.Event) {
										aff.newCostType = ctx.JSSrc().Get("value").String()
										ctx.Update()
									}).
									Body(
										app.Option().Value("free").Text("Free").Selected(aff.newCostType == "free"),
										app.Option().Value("fear").Text("Fear").Selected(aff.newCostType == "fear"),
										app.Option().Value("stress").Text("Stress").Selected(aff.newCostType == "stress"),
									),
							),
						app.If(aff.newCostType == "fear", func() app.UI {
							return app.Div().
								Class("form-group").
								Body(
									app.Label().Text("Fear Cost:"),
									app.Input().
										Type("number").
										Value(aff.newFearCost).
										Min(1).Max(10).
										OnInput(handleIntInput(&aff.newFearCost)),
								)
						}),
						app.If(aff.newCostType == "stress", func() app.UI {
							return app.Div().
								Class("form-group").
								Body(
									app.Label().Text("Stress Cost:"),
									app.Input().
										Type("number").
										Value(aff.newStressCost).
										Min(1).Max(10).
										OnInput(handleIntInput(&aff.newStressCost)),
								)
						}),
						app.Div().
							Class("form-group").
							Body(
								app.Label().Text("Spotlight Limit:"),
								app.Input().
									Type("number").
									Value(aff.newSpotlightLim).
									Min(0).Max(5).
									OnInput(handleIntInput(&aff.newSpotlightLim)),
							),
					),
				app.Div().
					Class("form-row").
					Body(
						app.Div().
							Class("form-group").
							Body(
								app.Label().Text("Usage Limit:"),
								app.Select().
									OnChange(func(ctx app.Context, e app.Event) {
										aff.newUsageType = ctx.JSSrc().Get("value").String()
										ctx.Update()
									}).
									Body(
										app.Option().Value("unlimited").Text("Unlimited").Selected(aff.newUsageType == "unlimited"),
										app.Option().Value("once_per_scene").Text("Once per Scene").Selected(aff.newUsageType == "once_per_scene"),
										app.Option().Value("once_per_session").Text("Once per Session").Selected(aff.newUsageType == "once_per_session"),
										app.Option().Value("once_per_rest").Text("Once per Rest").Selected(aff.newUsageType == "once_per_rest"),
									),
							),
						app.If(aff.newUsageType != "unlimited", func() app.UI {
							return app.Div().
								Class("form-group").
								Body(
									app.Label().Text("Max Uses:"),
									app.Input().
										Type("number").
										Value(aff.newMaxUses).
										Min(1).Max(10).
										OnInput(handleIntInput(&aff.newMaxUses)),
								)
						}),
					),
			),
			app.Div().Class("form-subsection").Body(
				app.H5().Text("Tokens"),
				app.Div().
					Class("form-group").
					Body(
						app.Label().
							Class("checkbox-label").
							Body(
								app.Input().
									Type("checkbox").
									Checked(aff.newUsesTokens).
									OnChange(func(ctx app.Context, e app.Event) {
										aff.newUsesTokens = ctx.JSSrc().Get("checked").Bool()
										ctx.Update()
									}),
								app.Text(" Uses tokens"),
							),
					),
				app.If(aff.newUsesTokens, func() app.UI {
					return app.Div().
						Class("form-row").
						Body(
							app.Div().
								Class("form-group").
								Body(
									app.Label().Text("Token Name:"),
									app.Input().
										Type("text").
										Value(aff.newTokenName).
										Placeholder("Rage").
										OnInput(func(ctx app.Context, e app.Event) {
											aff.newTokenName = ctx.JSSrc().Get("value").String()
										}),
								),
							app.Div().
								Class("form-group").
								Body(
									app.Label().Text("Max Tokens (0=unlimited):"),
									app.Input().
										Type("number").
										Value(aff.newTokensMax).
										Min(0).Max(20).
										OnInput(handleIntInput(&aff.newTokensMax)),
								),
						)
				}),
			),
			app.Div().Class("form-subsection").Body(
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
								app.Option().Value("area_damage").Text("Area Damage (AOE)").Selected(aff.newEffectType == "area_damage"),
								app.Option().Value("stress_inflict").Text("Inflict Stress").Selected(aff.newEffectType == "stress_inflict"),
								app.Option().Value("apply_vulnerable").Text("Apply Vulnerable").Selected(aff.newEffectType == "apply_vulnerable"),
								app.Option().Value("apply_poison").Text("Apply Poison").Selected(aff.newEffectType == "apply_poison"),
								app.Option().Value("apply_ignite").Text("Apply Burning").Selected(aff.newEffectType == "apply_ignite"),
								app.Option().Value("apply_stunned").Text("Apply Stunned").Selected(aff.newEffectType == "apply_stunned"),
								app.Option().Value("apply_restrained").Text("Apply Restrained").Selected(aff.newEffectType == "apply_restrained"),
								app.Option().Value("armor_damage").Text("Damage Armor").Selected(aff.newEffectType == "armor_damage"),
								app.Option().Value("grant_armor").Text("Grant Armor Slots").Selected(aff.newEffectType == "grant_armor"),
								app.Option().Value("resistance").Text("Resistance (50%)").Selected(aff.newEffectType == "resistance"),
								app.Option().Value("immunity").Text("Immunity (100%)").Selected(aff.newEffectType == "immunity"),
								app.Option().Value("summon_allies").Text("Summon Allies").Selected(aff.newEffectType == "summon_allies"),
								app.Option().Value("token_place").Text("Place Token").Selected(aff.newEffectType == "token_place"),
								app.Option().Value("token_spend").Text("Spend Token").Selected(aff.newEffectType == "token_spend"),
							),
					),
				app.If(needsDiceFormula(aff.newEffectType), func() app.UI {
					return app.Div().
						Class(formGroupClass(diceErr != "")).
						Body(
							app.Label().Text("Dice Formula:"),
							app.Input().
								Type("text").
								Value(aff.newDiceFormula).
								Placeholder("1d8").
								OnInput(func(ctx app.Context, e app.Event) {
									aff.newDiceFormula = ctx.JSSrc().Get("value").String()
									ctx.Update()
								}),
							renderValidationMsg(diceErr),
						)
				}),
				app.If(needsMagnitude(aff.newEffectType), func() app.UI {
					return app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Amount:"),
							app.Input().
								Type("number").
								Value(aff.newMagnitude).
								Min(1).Max(10).
								OnInput(handleIntInput(&aff.newMagnitude)),
						)
				}),
				app.If(needsArmorCost(aff.newEffectType), func() app.UI {
					return app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Armor Slots to Reduce:"),
							app.Input().
								Type("number").
								Value(aff.newArmorCost).
								Min(1).Max(3).
								OnInput(handleIntInput(&aff.newArmorCost)),
						)
				}),
				app.If(aff.newEffectType == "armor_damage", func() app.UI {
					return app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Fear Gained if No Slot Left:"),
							app.Input().
								Type("number").
								Value(aff.newFallbackFear).
								Min(0).Max(3).
								OnInput(handleIntInput(&aff.newFallbackFear)),
						)
				}),
				app.If(needsReaction(aff.newEffectType), func() app.UI {
					options := []app.UI{
						app.Option().Value("").Text(traitLabel("")).Selected(aff.newReactionTrait == ""),
					}
					for _, trait := range reactionTraits {
						options = append(options, app.Option().Value(trait).Text(traitLabel(trait)).Selected(aff.newReactionTrait == trait))
					}
					return app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Reaction Roll to Avoid:"),
							app.Select().
								OnChange(func(ctx app.Context, e app.Event) {
									aff.newReactionTrait = ctx.JSSrc().Get("value").String()
									ctx.Update()
								}).
								Body(options...),
						)
				}),
				app.If(needsReaction(aff.newEffectType) && aff.newReactionTrait != "", func() app.UI {
					return app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Reaction Difficulty (0 = this adversary's Difficulty):"),
							app.Input().
								Type("number").
								Value(aff.newReactionDC).
								Min(0).Max(30).
								OnInput(handleIntInput(&aff.newReactionDC)),
						)
				}),
				app.If(isDamageEffect(aff.newEffectType) && aff.newReactionTrait != "", func() app.UI {
					return app.Div().
						Class("form-group").
						Body(
							app.Label().
								Class("checkbox-label").
								Body(
									app.Input().
										Type("checkbox").
										Checked(aff.newHalfOnSuccess).
										OnChange(func(ctx app.Context, e app.Event) {
											aff.newHalfOnSuccess = ctx.JSSrc().Get("checked").Bool()
										}),
									app.Text(" Success takes half damage (otherwise none)"),
								),
						)
				}),
				app.If(needsDamageType(aff.newEffectType), func() app.UI {
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
				app.If(needsDuration(aff.newEffectType), func() app.UI {
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
				app.If(aff.newEffectType == "summon_allies", func() app.UI {
					return aff.renderSummonControls()
				}),
			),
			app.Div().Class("form-subsection").Body(
				app.H5().Text("Difficulty Check"),
				app.Div().
					Class("form-row").
					Body(
						app.Div().
							Class("form-group").
							Body(
								app.Label().Text("DC Type:"),
								app.Select().
									OnChange(func(ctx app.Context, e app.Event) {
										aff.newDCType = ctx.JSSrc().Get("value").String()
										ctx.Update()
									}).
									Body(
										app.Option().Value("none").Text("Auto-Succeed").Selected(aff.newDCType == "none"),
										app.Option().Value("target").Text("Target's Evasion").Selected(aff.newDCType == "target"),
										app.Option().Value("fixed").Text("Fixed DC").Selected(aff.newDCType == "fixed"),
									),
							),
						app.If(aff.newDCType == "fixed", func() app.UI {
							return app.Div().
								Class("form-group").
								Body(
									app.Label().Text("Fixed DC:"),
									app.Input().
										Type("number").
										Value(aff.newFixedDC).
										Min(1).Max(30).
										OnInput(handleIntInput(&aff.newFixedDC)),
								)
						}),
					),
			),
			app.Div().
				Class("form-actions").
				Body(
					app.Button().
						Class("btn btn-primary").
						Type("button").
						Text(saveText).
						Disabled(aff.newName == "" || diceErr != "").
						OnClick(func(ctx app.Context, e app.Event) {
							e.PreventDefault()
							aff.saveFeature(ctx)
						}),
					app.Button().
						Class("btn btn-secondary").
						Type("button").
						Text("Cancel").
						OnClick(func(ctx app.Context, e app.Event) {
							e.PreventDefault()
							aff.showForm = false
							aff.resetForm()
							ctx.Update()
						}),
				),
		)
}

// renderSummonControls renders the summon-specific form controls
func (aff *AdversaryFeatureForm) renderSummonControls() app.UI {
	return app.Div().
		Class("summon-controls").
		Body(
			app.H5().Text("Summon Configuration"),
			app.Div().
				Class("form-row").
				Body(
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Quantity Type:"),
							app.Select().
								OnChange(func(ctx app.Context, e app.Event) {
									aff.newQuantityType = ctx.JSSrc().Get("value").String()
									ctx.Update()
								}).
								Body(
									app.Option().Value("fixed").Text("Fixed").Selected(aff.newQuantityType == "fixed"),
									app.Option().Value("dice").Text("Dice Roll").Selected(aff.newQuantityType == "dice"),
								),
						),
					app.If(aff.newQuantityType == "fixed", func() app.UI {
						return app.Div().
							Class("form-group").
							Body(
								app.Label().Text("Quantity:"),
								app.Input().
									Type("number").
									Value(aff.newMagnitude).
									Min(1).
									Max(10).
									OnInput(handleIntInput(&aff.newMagnitude)),
							)
					}),
					app.If(aff.newQuantityType == "dice", func() app.UI {
						return app.Div().
							Class("form-group").
							Body(
								app.Label().Text("Dice Formula:"),
								app.Input().
									Type("text").
									Value(aff.newDiceFormula).
									Placeholder("1d4").
									OnInput(func(ctx app.Context, e app.Event) {
										aff.newDiceFormula = ctx.JSSrc().Get("value").String()
									}),
							)
					}),
				),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Summon Pool (select adversaries):"),
					app.If(len(aff.AvailableAdversaries) == 0, func() app.UI {
						return app.Div().Class("empty-pool").Text("No adversaries defined yet. Add adversaries first.")
					}),
					app.If(len(aff.AvailableAdversaries) > 0, func() app.UI {
						return aff.renderSummonPoolPicker()
					}),
				),
		)
}

// renderSummonPoolPicker renders checkboxes for selecting adversaries
func (aff *AdversaryFeatureForm) renderSummonPoolPicker() app.UI {
	items := make([]app.UI, len(aff.AvailableAdversaries))
	for i, adv := range aff.AvailableAdversaries {
		advID := adv.ID
		advName := adv.Name
		items[i] = app.Label().
			Class("checkbox-label").
			Body(
				app.Input().
					Type("checkbox").
					Checked(aff.selectedForSummon[advID]).
					OnChange(func(ctx app.Context, e app.Event) {
						if aff.selectedForSummon == nil {
							aff.selectedForSummon = make(map[string]bool)
						}
						aff.selectedForSummon[advID] = ctx.JSSrc().Get("checked").Bool()
						ctx.Update()
					}),
				app.Text(" "+advName),
			)
	}
	return app.Div().Class("summon-pool-picker").Body(items...)
}



func (aff *AdversaryFeatureForm) renderLibraryPicker() app.UI {
	options := make([]app.UI, 0, len(aff.libraryFeatureKeys)+1)
	options = append(options,
		app.Option().
			Value(customLibraryOption).
			Text("-- Custom Feature --").
			Selected(aff.selectedLibraryKey == "" || aff.selectedLibraryKey == customLibraryOption),
	)
	for _, key := range aff.libraryFeatureKeys {
		k := key
		options = append(options,
			app.Option().Value(k).Text(k).Selected(aff.selectedLibraryKey == k),
		)
	}

	return app.Div().
		Class("form-subsection").
		Body(
			app.H5().Text("Feature Library"),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text(fmt.Sprintf("Start from template (%d available):", len(aff.libraryFeatureKeys))),
					app.Select().
						OnChange(func(ctx app.Context, e app.Event) {
							key := ctx.JSSrc().Get("value").String()
							aff.selectedLibraryKey = key
							aff.populateFromLibrary(key)
							ctx.Update()
						}).
						Body(options...),
				),
		)
}

func (aff *AdversaryFeatureForm) populateFromLibrary(key string) {
	if key == "" || key == customLibraryOption || aff.featureLibrary == nil {
		return
	}
	feat := aff.featureLibrary.ParseFeature(key, "")
	if feat == nil {
		return
	}

	aff.newName = feat.Name
	aff.newDescription = feat.Description
	aff.newPassive = feat.Passive
	aff.newTrigger = feat.Trigger.Type
	if aff.newTrigger == "" {
		aff.newTrigger = models.TriggerOnAttack
	}
	aff.newFearCost = feat.FearCost
	aff.newStressCost = feat.StressCost
	aff.newSpotlightLim = feat.SpotlightLimit
	aff.newUsageType = feat.UsageType
	aff.newMaxUses = feat.MaxUses
	aff.newDCType = feat.DCType
	aff.newFixedDC = feat.FixedDC
	aff.newUsesTokens = feat.UsesTokens
	aff.newTokenName = feat.TokenName
	aff.newTokensMax = feat.TokensMax

	switch {
	case feat.FearCost > 0:
		aff.newCostType = "fear"
	case feat.StressCost > 0:
		aff.newCostType = "stress"
	default:
		aff.newCostType = "free"
	}

	if len(feat.Effects) > 0 {
		eff := feat.Effects[0]
		aff.newEffectType = reverseEffectCategory(eff.Category)
		aff.newDiceFormula = eff.DiceFormula
		aff.newDamageType = eff.DamageType
		aff.newMagnitude = eff.Magnitude
		aff.newDuration = eff.Duration
		aff.newArmorCost = max(eff.ArmorCost, 1)
		aff.newFallbackFear = eff.FallbackFear
		aff.newReactionTrait = eff.ReactionTrait
		aff.newReactionDC = eff.ReactionDC
		aff.newHalfOnSuccess = eff.HalfOnSuccess
	}
}

func (aff *AdversaryFeatureForm) startEditing(ctx app.Context, idx int) {
	if idx < 0 || idx >= len(aff.Features) {
		return
	}
	feature := aff.Features[idx]
	aff.editingIndex = idx
	aff.showForm = true
	aff.newName = feature.Name
	aff.newDescription = feature.Description
	aff.newPassive = feature.Passive
	aff.newTrigger = feature.Trigger.Type
	if aff.newTrigger == "" {
		aff.newTrigger = models.TriggerOnAttack
	}
	aff.newFearCost = feature.FearCost
	aff.newStressCost = feature.StressCost
	aff.newSpotlightLim = feature.SpotlightLimit
	aff.newUsageType = feature.UsageType
	aff.newMaxUses = feature.MaxUses
	aff.newDCType = feature.DCType
	aff.newFixedDC = feature.FixedDC
	aff.newUsesTokens = feature.UsesTokens
	aff.newTokenName = feature.TokenName
	aff.newTokensMax = feature.TokensMax

	switch {
	case feature.FearCost > 0:
		aff.newCostType = "fear"
	case feature.StressCost > 0:
		aff.newCostType = "stress"
	default:
		aff.newCostType = "free"
	}

	if len(feature.Effects) > 0 {
		eff := feature.Effects[0]
		aff.newEffectType = reverseEffectCategory(eff.Category)
		aff.newDiceFormula = eff.DiceFormula
		aff.newDamageType = eff.DamageType
		aff.newMagnitude = eff.Magnitude
		aff.newDuration = eff.Duration
		aff.newArmorCost = max(eff.ArmorCost, 1)
		aff.newFallbackFear = eff.FallbackFear
		aff.newReactionTrait = eff.ReactionTrait
		aff.newReactionDC = eff.ReactionDC
		aff.newHalfOnSuccess = eff.HalfOnSuccess
		if eff.Category == models.FeatureEffectSummonAllies {
			if eff.DiceFormula != "" {
				aff.newQuantityType = "dice"
			} else {
				aff.newQuantityType = "fixed"
			}
			aff.selectedForSummon = make(map[string]bool)
			for _, s := range eff.SummonPool {
				aff.selectedForSummon[s.ID] = true
			}
		}
	}
	ctx.Update()
}

func (aff *AdversaryFeatureForm) buildFeature() *models.AdversaryFeature {
	effectCategory := resolveEffectCategory(aff.newEffectType)

	magnitude := aff.newMagnitude
	diceFormula := aff.newDiceFormula
	switch aff.newEffectType {
	case "resistance":
		magnitude = 50
	case "immunity":
		magnitude = 100
	case "summon_allies":
		if aff.newQuantityType == "fixed" {
			diceFormula = ""
		} else {
			magnitude = 0
		}
	}

	effect := models.FeatureEffect{
		Category:    effectCategory,
		Magnitude:   magnitude,
		DiceFormula: diceFormula,
		Duration:    aff.newDuration,
		TargetType:  models.FeatureTargetSingle,
		DamageType:  aff.newDamageType,
	}
	if needsArmorCost(aff.newEffectType) {
		effect.ArmorCost = aff.newArmorCost
	}
	if effectCategory == models.FeatureEffectArmorDamage {
		effect.FallbackFear = aff.newFallbackFear
	}
	if needsReaction(aff.newEffectType) && aff.newReactionTrait != "" {
		effect.ReactionTrait = aff.newReactionTrait
		effect.ReactionDC = aff.newReactionDC
		effect.HalfOnSuccess = isDamageEffect(aff.newEffectType) && aff.newHalfOnSuccess
	}

	if effectCategory == models.FeatureEffectSummonAllies {
		effect.SummonPool = make([]*models.Combatant, 0)
		for _, adv := range aff.AvailableAdversaries {
			if aff.selectedForSummon[adv.ID] {
				effect.SummonPool = append(effect.SummonPool, adv)
			}
		}
	}

	fearCost := 0
	stressCost := 0
	switch aff.newCostType {
	case "fear":
		fearCost = aff.newFearCost
	case "stress":
		stressCost = aff.newStressCost
	}

	maxUses := 0
	usesLeft := 0
	if aff.newUsageType != "unlimited" {
		maxUses = aff.newMaxUses
		usesLeft = aff.newMaxUses
	}

	return &models.AdversaryFeature{
		Name:        aff.newName,
		Description: aff.newDescription,
		Passive:     aff.newPassive,
		Trigger: models.FeatureTriggerCondition{
			Type: aff.newTrigger,
		},
		FearCost:       fearCost,
		StressCost:     stressCost,
		SpotlightLimit: aff.newSpotlightLim,
		UsageType:      aff.newUsageType,
		MaxUses:        maxUses,
		UsesLeft:       usesLeft,
		DCType:         aff.newDCType,
		FixedDC:        aff.newFixedDC,
		UsesTokens:     aff.newUsesTokens,
		TokenName:      aff.newTokenName,
		TokensHeld:     0,
		TokensMax:      aff.newTokensMax,
		Effects:        []models.FeatureEffect{effect},
	}
}

func (aff *AdversaryFeatureForm) saveFeature(ctx app.Context) {
	feature := aff.buildFeature()

	if aff.editingIndex >= 0 && aff.editingIndex < len(aff.Features) {
		aff.Features[aff.editingIndex] = feature
	} else {
		aff.Features = append(aff.Features, feature)
	}

	if aff.OnChange != nil {
		aff.OnChange(aff.Features)
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
			aff.OnChange(aff.Features)
		}

		ctx.Update()
	}
}

// resetForm resets the form to default values
func (aff *AdversaryFeatureForm) resetForm() {
	aff.editingIndex = -1
	aff.newName = ""
	aff.newDescription = ""
	aff.newPassive = true
	aff.newFearCost = 0
	aff.newStressCost = 0
	aff.newCostType = "free"
	aff.newSpotlightLim = 0
	aff.newUsageType = "unlimited"
	aff.newMaxUses = 0
	aff.newDiceFormula = "1d6"
	aff.newDamageType = "physical"
	aff.newEffectType = "damage"
	aff.newMagnitude = 1
	aff.newArmorCost = 1
	aff.newFallbackFear = 0
	aff.newReactionTrait = ""
	aff.newReactionDC = 0
	aff.newHalfOnSuccess = false
	aff.newTrigger = models.TriggerOnAttack
	aff.newDuration = "instant"
	aff.newDCType = "none"
	aff.newFixedDC = 12
	aff.newQuantityType = "fixed"
	aff.selectedForSummon = make(map[string]bool)
	aff.newUsesTokens = false
	aff.newTokenName = ""
	aff.newTokensMax = 0
	aff.selectedLibraryKey = customLibraryOption
}
