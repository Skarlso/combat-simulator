package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// DomainAbilityForm manages adding/editing domain abilities (player skills)
type DomainAbilityForm struct {
	app.Compo

	Abilities []*models.DomainAbility
	OnChange  func([]*models.DomainAbility)

	// Form state
	showForm         bool
	newName          string
	newDescription   string
	newPassive       bool
	newTriggerType   string
	newTriggerFilter string
	newHopeCost      int
	newStressCost    int
	newRefreshType   string
	newMaxUses       int
	newDCType        string
	newFixedDC       int
	newDiceFormula   string
	newDamageType    string
	newEffectType    string
	newMagnitude     int
	newDuration      string
	newUsesTokens    bool
	newTokenName     string
	newTokensMax     int
}

// OnMount initializes default values
func (daf *DomainAbilityForm) OnMount(ctx app.Context) {
	daf.resetForm()
}

// Render renders the ability list and form
func (daf *DomainAbilityForm) Render() app.UI {
	return app.Div().
		Class("domain-ability-form").
		Body(
			app.Div().
				Class("ability-header").
				Body(
					app.Label().Text("Domain Abilities:"),
					app.Button().
						Class("btn btn-small").
						Text("+ Add Ability").
						OnClick(func(ctx app.Context, e app.Event) {
							daf.showForm = !daf.showForm
							ctx.Update()
						}),
				),
			daf.renderAbilityList(),
			app.If(daf.showForm, func() app.UI {
				return daf.renderAddForm()
			}),
		)
}

// renderAbilityList shows current abilities
func (daf *DomainAbilityForm) renderAbilityList() app.UI {
	if len(daf.Abilities) == 0 {
		return app.Div().
			Class("empty-abilities").
			Text("No abilities added")
	}

	items := make([]app.UI, len(daf.Abilities))
	for i := range daf.Abilities {
		idx := i
		items[i] = daf.renderAbilityItem(daf.Abilities[idx], idx)
	}

	return app.Div().
		Class("ability-list").
		Body(items...)
}

// renderAbilityItem renders a single ability
func (daf *DomainAbilityForm) renderAbilityItem(ability *models.DomainAbility, idx int) app.UI {
	costText := ""
	if ability.HopeCost > 0 {
		costText += fmt.Sprintf("%d Hope", ability.HopeCost)
	}
	if ability.StressCost > 0 {
		if costText != "" {
			costText += " + "
		}
		costText += fmt.Sprintf("%d Stress", ability.StressCost)
	}
	if costText == "" {
		costText = "Free"
	}

	usesText := ""
	if ability.MaxUses > 0 {
		usesText = fmt.Sprintf(" | %d uses per %s", ability.MaxUses, ability.RefreshType)
	} else if ability.RefreshType != "unlimited" {
		usesText = fmt.Sprintf(" | Refreshes on %s", ability.RefreshType)
	}

	effectText := ""
	if len(ability.Effects) > 0 {
		effect := ability.Effects[0]
		switch effect.Category {
		case models.FeatureEffectDamage, models.FeatureEffectDirectDamage:
			effectText = fmt.Sprintf("%s %s damage", effect.DiceFormula, effect.DamageType)
		case models.FeatureEffectHPHeal:
			effectText = fmt.Sprintf("Heal %s", effect.DiceFormula)
		case models.FeatureEffectStressInflict:
			effectText = fmt.Sprintf("Inflict %d stress", effect.Magnitude)
		case models.FeatureEffectStressClear:
			effectText = fmt.Sprintf("Clear %d stress", effect.Magnitude)
		case models.FeatureEffectArmorDamage:
			effectText = fmt.Sprintf("Damage %d armor slot(s)", effect.Magnitude)
		case models.FeatureEffectGrantArmor:
			effectText = fmt.Sprintf("Grant %d armor slot(s)", effect.Magnitude)
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
					app.Strong().Text(ability.Name),
					app.Span().Text(fmt.Sprintf(" | %s%s", costText, usesText)),
					app.If(effectText != "", func() app.UI {
						return app.Div().Class("ability-effect").Text("Effect: " + effectText)
					}),
					app.If(ability.Description != "", func() app.UI {
						return app.Div().Class("ability-desc").Text(ability.Description)
					}),
				),
			app.Button().
				Class("btn btn-danger btn-small").
				Text("Remove").
				OnClick(func(ctx app.Context, e app.Event) {
					daf.removeAbility(ctx, idx)
				}),
		)
}

// renderAddForm renders the ability creation form
func (daf *DomainAbilityForm) renderAddForm() app.UI {
	return app.Div().
		Class("ability-add-form").
		Body(
			app.H4().Text("Add Domain Ability"),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Name:"),
					app.Input().
						Type("text").
						Value(daf.newName).
						Placeholder("Rune Strike").
						OnInput(func(ctx app.Context, e app.Event) {
							daf.newName = ctx.JSSrc().Get("value").String()
						}),
				),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Description (optional):"),
					app.Input().
						Type("text").
						Value(daf.newDescription).
						Placeholder("Channel runic power...").
						OnInput(func(ctx app.Context, e app.Event) {
							daf.newDescription = ctx.JSSrc().Get("value").String()
						}),
				),
			app.Div().
				Class("form-group").
				Body(
					app.Label().
						Body(
							app.Input().
								Type("checkbox").
								Checked(daf.newPassive).
								OnChange(func(ctx app.Context, e app.Event) {
									daf.newPassive = ctx.JSSrc().Get("checked").Bool()
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
							app.Label().Text("Trigger Type:"),
							app.Select().
								OnChange(func(ctx app.Context, e app.Event) {
									daf.newTriggerType = ctx.JSSrc().Get("value").String()
									ctx.Update()
								}).
								Body(
									app.Option().Value("").Text("None (manual activation)").Selected(daf.newTriggerType == ""),
									app.Option().Value("on_attack").Text("On Attack").Selected(daf.newTriggerType == "on_attack"),
									app.Option().Value("on_hit").Text("On Hit").Selected(daf.newTriggerType == "on_hit"),
									app.Option().Value("on_critical").Text("On Critical").Selected(daf.newTriggerType == "on_critical"),
									app.Option().Value("on_damage_taken").Text("On Damage Taken").Selected(daf.newTriggerType == "on_damage_taken"),
									app.Option().Value("start_turn").Text("Start of Turn").Selected(daf.newTriggerType == "start_turn"),
									app.Option().Value("end_turn").Text("End of Turn").Selected(daf.newTriggerType == "end_turn"),
								),
						),
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Trigger Filter (optional):"),
							app.Select().
								OnChange(func(ctx app.Context, e app.Event) {
									daf.newTriggerFilter = ctx.JSSrc().Get("value").String()
									ctx.Update()
								}).
								Body(
									app.Option().Value("").Text("None").Selected(daf.newTriggerFilter == ""),
									app.Option().Value("if_target_below_50_hp").Text("If target below 50% HP").Selected(daf.newTriggerFilter == "if_target_below_50_hp"),
									app.Option().Value("if_hidden").Text("If hidden").Selected(daf.newTriggerFilter == "if_hidden"),
									app.Option().Value("if_ally_adjacent").Text("If ally adjacent").Selected(daf.newTriggerFilter == "if_ally_adjacent"),
								),
						),
				),
			app.Div().
				Class("form-row").
				Body(
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Hope Cost:"),
							app.Input().
								Type("number").
								Value(daf.newHopeCost).
								Min(0).
								Max(10).
								OnInput(handleIntInput(&daf.newHopeCost)),
						),
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Stress Cost:"),
							app.Input().
								Type("number").
								Value(daf.newStressCost).
								Min(0).
								Max(5).
								OnInput(handleIntInput(&daf.newStressCost)),
						),
				),
			app.Div().
				Class("form-row").
				Body(
					app.Div().
						Class("form-group").
						Body(
							app.Label().Text("Refresh Type:"),
							app.Select().
								OnChange(func(ctx app.Context, e app.Event) {
									daf.newRefreshType = ctx.JSSrc().Get("value").String()
									ctx.Update()
								}).
								Body(
									app.Option().Value("unlimited").Text("Unlimited").Selected(daf.newRefreshType == "unlimited"),
									app.Option().Value("rest").Text("Per Rest").Selected(daf.newRefreshType == "rest"),
									app.Option().Value("long_rest").Text("Per Long Rest").Selected(daf.newRefreshType == "long_rest"),
									app.Option().Value("scene").Text("Per Scene").Selected(daf.newRefreshType == "scene"),
								),
						),
					app.If(daf.newRefreshType != "unlimited", func() app.UI {
						return app.Div().
							Class("form-group").
							Body(
								app.Label().Text("Max Uses:"),
								app.Input().
									Type("number").
									Value(daf.newMaxUses).
									Min(0).
									Max(10).
									OnInput(handleIntInput(&daf.newMaxUses)),
							)
					}),
				),
			app.H5().Text("Difficulty Check"),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("DC Type:"),
					app.Select().
						OnChange(func(ctx app.Context, e app.Event) {
							daf.newDCType = ctx.JSSrc().Get("value").String()
							ctx.Update()
						}).
						Body(
							app.Option().Value("none").Text("Auto-Succeed").Selected(daf.newDCType == "none"),
							app.Option().Value("target").Text("Target's Difficulty").Selected(daf.newDCType == "target"),
							app.Option().Value("fixed").Text("Fixed DC").Selected(daf.newDCType == "fixed"),
						),
				),
			app.If(daf.newDCType == "fixed", func() app.UI {
				return app.Div().
					Class("form-group").
					Body(
						app.Label().Text("Fixed DC:"),
						app.Input().
							Type("number").
							Value(daf.newFixedDC).
							Min(1).
							Max(30).
							OnInput(handleIntInput(&daf.newFixedDC)),
					)
			}),
			app.H5().Text("Token Configuration"),
			app.Div().
				Class("form-group").
				Body(
					app.Label().
						Body(
							app.Input().
								Type("checkbox").
								Checked(daf.newUsesTokens).
								OnChange(func(ctx app.Context, e app.Event) {
									daf.newUsesTokens = ctx.JSSrc().Get("checked").Bool()
									ctx.Update()
								}),
							app.Text(" This ability uses tokens"),
						),
				),
			app.If(daf.newUsesTokens, func() app.UI {
				return app.Div().
					Body(
						app.Div().
							Class("form-group").
							Body(
								app.Label().Text("Token Name (e.g., 'Chaos', 'Ward'):"),
								app.Input().
									Type("text").
									Value(daf.newTokenName).
									Placeholder("Chaos").
									OnInput(func(ctx app.Context, e app.Event) {
										daf.newTokenName = ctx.JSSrc().Get("value").String()
									}),
							),
						app.Div().
							Class("form-group").
							Body(
								app.Label().Text("Max Tokens (0 = unlimited):"),
								app.Input().
									Type("number").
									Value(daf.newTokensMax).
									Min(0).
									Max(20).
									OnInput(handleIntInput(&daf.newTokensMax)),
							),
					)
			}),
			app.H5().Text("Effect"),
			app.Div().
				Class("form-group").
				Body(
					app.Label().Text("Effect Type:"),
					app.Select().
						OnChange(func(ctx app.Context, e app.Event) {
							daf.newEffectType = ctx.JSSrc().Get("value").String()
							ctx.Update()
						}).
						Body(
							app.Option().Value("damage").Text("Damage").Selected(daf.newEffectType == "damage"),
							app.Option().Value("area_damage").Text("Area Damage (AOE)").Selected(daf.newEffectType == "area_damage"),
							app.Option().Value("heal").Text("Heal HP").Selected(daf.newEffectType == "heal"),
							app.Option().Value("stress_inflict").Text("Inflict Stress").Selected(daf.newEffectType == "stress_inflict"),
							app.Option().Value("stress_clear").Text("Clear Stress").Selected(daf.newEffectType == "stress_clear"),
							app.Option().Value("apply_vulnerable").Text("Apply Vulnerable").Selected(daf.newEffectType == "apply_vulnerable"),
							app.Option().Value("apply_poison").Text("Apply Poison").Selected(daf.newEffectType == "apply_poison"),
							app.Option().Value("armor_damage").Text("Damage Armor Slots").Selected(daf.newEffectType == "armor_damage"),
							app.Option().Value("grant_armor").Text("Grant Armor Slots").Selected(daf.newEffectType == "grant_armor"),
							app.Option().Value("damage_reduction").Text("Damage Reduction (Ward)").Selected(daf.newEffectType == "damage_reduction"),
							app.Option().Value("token_place").Text("Place Token").Selected(daf.newEffectType == "token_place"),
							app.Option().Value("token_spend").Text("Spend Token").Selected(daf.newEffectType == "token_spend"),
							app.Option().Value("resistance").Text("Resistance (50%)").Selected(daf.newEffectType == "resistance"),
							app.Option().Value("immunity").Text("Immunity (100%)").Selected(daf.newEffectType == "immunity"),
						),
				),
			app.If(daf.needsDiceFormula(), func() app.UI {
				return app.Div().
					Class("form-group").
					Body(
						app.Label().Text("Dice Formula (e.g., 2d6+3):"),
						app.Input().
							Type("text").
							Value(daf.newDiceFormula).
							Placeholder("2d6+3").
							OnInput(func(ctx app.Context, e app.Event) {
								daf.newDiceFormula = ctx.JSSrc().Get("value").String()
							}),
					)
			}),
			app.If(daf.needsMagnitude(), func() app.UI {
				return app.Div().
					Class("form-group").
					Body(
						app.Label().Text("Amount:"),
						app.Input().
							Type("number").
							Value(daf.newMagnitude).
							Min(1).
							Max(10).
							OnInput(handleIntInput(&daf.newMagnitude)),
					)
			}),
			app.If(needsDamageType(daf.newEffectType), func() app.UI {
				return app.Div().
					Class("form-group").
					Body(
						app.Label().Text("Damage Type:"),
						app.Select().
							OnChange(func(ctx app.Context, e app.Event) {
								daf.newDamageType = ctx.JSSrc().Get("value").String()
							}).
							Body(
								app.Option().Value("physical").Text("Physical").Selected(daf.newDamageType == "physical"),
								app.Option().Value("magic").Text("Magic").Selected(daf.newDamageType == "magic"),
							),
					)
			}),
			app.If(needsDuration(daf.newEffectType), func() app.UI {
				return app.Div().
					Class("form-group").
					Body(
						app.Label().Text("Duration:"),
						app.Select().
							OnChange(func(ctx app.Context, e app.Event) {
								daf.newDuration = ctx.JSSrc().Get("value").String()
							}).
							Body(
								app.Option().Value("instant").Text("Instant").Selected(daf.newDuration == "instant"),
								app.Option().Value("1_turn").Text("1 Turn").Selected(daf.newDuration == "1_turn"),
								app.Option().Value("until_cleared").Text("Until Cleared").Selected(daf.newDuration == "until_cleared"),
								app.Option().Value("permanent").Text("Permanent").Selected(daf.newDuration == "permanent"),
							),
					)
			}),
			app.Div().
				Class("form-actions").
				Body(
					app.Button().
						Class("btn btn-primary").
						Type("button").
						Text("Add Ability").
						Disabled(daf.newName == "").
						OnClick(func(ctx app.Context, e app.Event) {
							e.PreventDefault()
							daf.addAbility(ctx)
						}),
					app.Button().
						Class("btn btn-secondary").
						Type("button").
						Text("Cancel").
						OnClick(func(ctx app.Context, e app.Event) {
							e.PreventDefault()
							daf.showForm = false
							daf.resetForm()
							ctx.Update()
						}),
				),
		)
}

// Helper methods for conditional rendering
func (daf *DomainAbilityForm) needsDiceFormula() bool {
	return daf.newEffectType == "damage" ||
		daf.newEffectType == "direct_damage" ||
		daf.newEffectType == "heal" ||
		daf.newEffectType == "damage_reduction"
}

func (daf *DomainAbilityForm) needsMagnitude() bool {
	return daf.newEffectType == "stress_inflict" ||
		daf.newEffectType == "stress_clear" ||
		daf.newEffectType == "grant_armor" ||
		daf.newEffectType == "token_place" ||
		daf.newEffectType == "token_spend"
}


// addAbility creates and adds a new ability
func (daf *DomainAbilityForm) addAbility(ctx app.Context) {
	effectCategory := resolveEffectCategory(daf.newEffectType)

	// Set magnitude based on effect type
	magnitude := daf.newMagnitude
	if daf.newEffectType == "resistance" {
		magnitude = 50
	} else if daf.newEffectType == "immunity" {
		magnitude = 100
	}

	effect := models.FeatureEffect{
		Category:    effectCategory,
		Magnitude:   magnitude,
		DiceFormula: daf.newDiceFormula,
		Duration:    daf.newDuration,
		TargetType:  models.FeatureTargetSingle,
		DamageType:  daf.newDamageType,
	}

	ability := &models.DomainAbility{
		Name:        daf.newName,
		Description: daf.newDescription,
		Passive:     daf.newPassive,
		Trigger: models.FeatureTriggerCondition{
			Type:   models.FeatureTriggerType(daf.newTriggerType),
			Filter: daf.newTriggerFilter,
		},
		HopeCost:    daf.newHopeCost,
		StressCost:  daf.newStressCost,
		RefreshType: daf.newRefreshType,
		MaxUses:     daf.newMaxUses,
		UsesLeft:    daf.newMaxUses,
		DCType:      daf.newDCType,
		FixedDC:     daf.newFixedDC,
		UsesTokens:  daf.newUsesTokens,
		TokenName:   daf.newTokenName,
		TokensHeld:  0,
		TokensMax:   daf.newTokensMax,
		Effects:     []models.FeatureEffect{effect},
	}

	daf.Abilities = append(daf.Abilities, ability)

	if daf.OnChange != nil {
		daf.OnChange(daf.Abilities)
	}

	daf.showForm = false
	daf.resetForm()
	ctx.Update()
}

// removeAbility removes an ability by index
func (daf *DomainAbilityForm) removeAbility(ctx app.Context, idx int) {
	if idx >= 0 && idx < len(daf.Abilities) {
		daf.Abilities = append(daf.Abilities[:idx], daf.Abilities[idx+1:]...)

		if daf.OnChange != nil {
			daf.OnChange(daf.Abilities)
		}

		ctx.Update()
	}
}

// resetForm resets the form to default values
func (daf *DomainAbilityForm) resetForm() {
	daf.newName = ""
	daf.newDescription = ""
	daf.newPassive = false
	daf.newTriggerType = ""
	daf.newTriggerFilter = ""
	daf.newHopeCost = 1
	daf.newStressCost = 0
	daf.newRefreshType = "rest"
	daf.newMaxUses = 3
	daf.newDiceFormula = "2d6"
	daf.newDamageType = "physical"
	daf.newEffectType = "damage"
	daf.newMagnitude = 1
	daf.newDuration = "instant"
	daf.newDCType = "none"
	daf.newFixedDC = 12
	daf.newUsesTokens = false
	daf.newTokenName = ""
	daf.newTokensMax = 5
}
