package components

import (
	"fmt"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/data"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/parser"
)

// DomainAbilityForm manages adding/editing domain abilities (player skills)
type DomainAbilityForm struct {
	app.Compo

	Abilities []*models.DomainAbility
	OnChange  func([]*models.DomainAbility)

	showForm         bool
	editingIndex     int
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
	newTargetType    string
	// The library card or ability being edited; effects this form cannot show are kept from it
	baseAbility *models.DomainAbility

	library      []parser.DomainCard
	selectedCard string
}

var (
	abilityTriggerOptions = []selectOption{
		{"", "None (manual)"}, {"on_attack", "On Attack"}, {"on_hit", "On Hit"}, {"on_critical", "On Critical"},
		{"on_damage_taken", "On Damage Taken"}, {"on_ally_damaged", "When an ally would take damage"},
		{"start_turn", "Start of Turn"}, {"end_turn", "End of Turn"},
	}
	triggerFilterOptions = []selectOption{
		{"", "None"}, {"if_target_below_50_hp", "Target below 50% HP"}, {"if_hidden", "If hidden"}, {"if_ally_adjacent", "If ally adjacent"},
	}
	refreshOptions = []selectOption{
		{"unlimited", "Unlimited"}, {"rest", "Per Rest"}, {"long_rest", "Per Long Rest"}, {"scene", "Per Scene"},
	}
	abilityEffectOptions = []selectOption{
		{"damage", "Damage"}, {"area_damage", "Area Damage (AOE)"}, {"heal", "Heal HP"},
		{"stress_inflict", "Inflict Stress"}, {"stress_clear", "Clear Stress"},
		{"apply_vulnerable", "Apply Vulnerable"}, {"apply_poison", "Apply Poison"}, {"apply_ignite", "Apply Burning"},
		{"apply_stunned", "Apply Stunned"}, {"apply_restrained", "Apply Restrained"},
		{"armor_damage", "Damage Armor Slots"}, {"grant_armor", "Grant Armor Slots"}, {"damage_reduction", "Damage Reduction (Ward)"},
		{"token_place", "Place Token"}, {"token_spend", "Spend Token"},
		{"resistance", "Resistance (50%)"}, {"immunity", "Immunity (100%)"},
		{"redirect_damage", "Shield an ally (take their hit)"},
	}
)

// OnMount initializes default values and loads the domain card library
func (daf *DomainAbilityForm) OnMount(ctx app.Context) {
	daf.resetForm()
	cards, err := data.LoadDomainCards()
	if err != nil {
		app.Log("domain card library failed to load:", err)
		return
	}
	daf.library = cards
}

// Render renders the ability list and form
func (daf *DomainAbilityForm) Render() app.UI {
	return app.Div().Class("domain-ability-form").Body(
		app.Div().Class("ability-header").Body(
			app.Label().Text("Domain Abilities"),
			app.Button().Class("btn btn-small").Text("+ Add Ability").OnClick(func(ctx app.Context, e app.Event) {
				daf.showForm = !daf.showForm
				ctx.Update()
			}),
		),
		daf.renderAbilityList(),
		app.If(daf.showForm, func() app.UI { return daf.renderAddForm() }),
	)
}

func (daf *DomainAbilityForm) renderAbilityList() app.UI {
	if len(daf.Abilities) == 0 {
		return app.Div().Class("empty-abilities").Text("No abilities added")
	}
	items := make([]app.UI, len(daf.Abilities))
	for i, ability := range daf.Abilities {
		items[i] = daf.renderAbilityItem(ability, i)
	}
	return app.Div().Class("ability-list").Body(items...)
}

func (daf *DomainAbilityForm) renderAbilityItem(ability *models.DomainAbility, idx int) app.UI {
	cost := joinParts(" + ", countText(ability.HopeCost, "Hope"), countText(ability.StressCost, "Stress"))
	if cost == "" {
		cost = "Free"
	}
	uses := ""
	if ability.MaxUses > 0 {
		uses = fmt.Sprintf(" | %d uses per %s", ability.MaxUses, ability.RefreshType)
	} else if ability.RefreshType != "unlimited" {
		uses = fmt.Sprintf(" | Refreshes on %s", ability.RefreshType)
	}
	effectText := ""
	if len(ability.Effects) > 0 {
		effectText = describeEffect(ability.Effects[0])
	}

	return editableItem(daf.editingIndex == idx, ability.Name, cost+uses, effectText, ability.Description,
		func(ctx app.Context, e app.Event) { daf.startEditing(ctx, idx) },
		func(ctx app.Context, e app.Event) { daf.removeAbility(ctx, idx) },
	)
}

// renderAddForm renders the ability creation form
func (daf *DomainAbilityForm) renderAddForm() app.UI {
	formTitle, saveText := "Add Domain Ability", "Add Ability"
	if daf.editingIndex >= 0 {
		formTitle, saveText = "Edit Domain Ability", "Save"
	}
	diceErr := validateDiceNotation(daf.newDiceFormula)

	effect := []app.UI{
		app.H5().Text("Effect"),
		selectField("Effect Type", daf.newEffectType, abilityEffectOptions, bindSelect(&daf.newEffectType, true)),
	}
	effect = append(effect, effectFields(daf.newEffectType, &daf.newDiceFormula, diceErr, &daf.newMagnitude, &daf.newDamageType, &daf.newDuration)...)
	effect = append(effect,
		app.If(needsRecipient(daf.newEffectType), func() app.UI {
			return selectField("Target", daf.newTargetType, recipientOptions, bindSelect(&daf.newTargetType, false))
		}),
		app.If(daf.baseAbility != nil && len(daf.baseAbility.Effects) > 1, func() app.UI {
			return app.Div().Class("form-hint").Text(fmt.Sprintf("Also keeps %d more effect(s) from the card, not editable here", len(daf.baseAbility.Effects)-1))
		}),
	)

	return app.Div().Class("ability-add-form").Body(
		app.H4().Text(formTitle),
		app.If(daf.editingIndex < 0 && len(daf.library) > 0, func() app.UI { return daf.renderLibraryPicker() }),
		app.Div().Class("form-row").Body(
			textField("Name", "Rune Strike", &daf.newName, false),
			textField("Description", "Channel runic power...", &daf.newDescription, false),
		),
		checkboxField("Passive (always on)", &daf.newPassive),
		app.Div().Class("form-subsection").Body(
			app.H5().Text("Trigger"),
			app.Div().Class("form-row").Body(
				selectField("Trigger Type", daf.newTriggerType, abilityTriggerOptions, bindSelect(&daf.newTriggerType, true)),
				selectField("Filter", daf.newTriggerFilter, triggerFilterOptions, bindSelect(&daf.newTriggerFilter, true)),
			),
		),
		app.Div().Class("form-subsection").Body(
			app.H5().Text("Cost & Usage"),
			app.Div().Class("form-row-3").Body(
				numberField("Hope Cost", &daf.newHopeCost, 0, 10),
				numberField("Stress Cost", &daf.newStressCost, 0, 5),
				selectField("Refresh", daf.newRefreshType, refreshOptions, bindSelect(&daf.newRefreshType, true)),
			),
			app.If(daf.newRefreshType != "unlimited", func() app.UI { return numberField("Max Uses", &daf.newMaxUses, 0, 10) }),
		),
		dcFields(&daf.newDCType, &daf.newFixedDC, "Target's Difficulty"),
		tokenFields(&daf.newUsesTokens, &daf.newTokenName, &daf.newTokensMax),
		app.Div().Class("form-subsection").Body(effect...),
		formActions(saveText, daf.newName == "" || diceErr != "",
			func(ctx app.Context, e app.Event) { daf.saveAbility(ctx) },
			func(ctx app.Context, e app.Event) {
				daf.showForm = false
				daf.resetForm()
				ctx.Update()
			},
		),
	)
}

func (daf *DomainAbilityForm) renderLibraryPicker() app.UI {
	options := []selectOption{{customLibraryOption, "-- Custom Ability --"}}
	note := ""
	for _, c := range daf.library {
		options = append(options, selectOption{c.Name, fmt.Sprintf("%s (%s, level %d)", c.Name, c.Domain, c.Level)})
		if c.Name == daf.selectedCard {
			note = c.Note
		}
	}
	current := daf.selectedCard
	if current == "" {
		current = customLibraryOption
	}

	return app.Div().Class("form-subsection").Body(
		app.H5().Text("Domain Card Library"),
		selectField(fmt.Sprintf("Start from card (%d available)", len(daf.library)), current, options, func(ctx app.Context, key string) {
			daf.selectedCard = key
			for i := range daf.library {
				if daf.library[i].Name == key {
					daf.loadAbility(&daf.library[i].Ability)
				}
			}
			ctx.Update()
		}),
		app.If(note != "", func() app.UI { return app.Div().Class("form-hint").Text(note) }),
	)
}

func (daf *DomainAbilityForm) startEditing(ctx app.Context, idx int) {
	if idx < 0 || idx >= len(daf.Abilities) {
		return
	}
	daf.editingIndex = idx
	daf.showForm = true
	daf.loadAbility(daf.Abilities[idx])
	ctx.Update()
}

// loadAbility fills the form from an ability and remembers it, so saving keeps what the form cannot show
func (daf *DomainAbilityForm) loadAbility(ability *models.DomainAbility) {
	base := *ability
	base.Effects = append([]models.FeatureEffect(nil), ability.Effects...)
	daf.baseAbility = &base

	daf.newName = ability.Name
	daf.newDescription = ability.Description
	daf.newPassive = ability.Passive
	daf.newTriggerType = string(ability.Trigger.Type)
	daf.newTriggerFilter = ability.Trigger.Filter
	daf.newHopeCost = ability.HopeCost
	daf.newStressCost = ability.StressCost
	daf.newRefreshType = ability.RefreshType
	daf.newMaxUses = ability.MaxUses
	daf.newDCType = ability.DCType
	daf.newFixedDC = ability.FixedDC
	daf.newUsesTokens = ability.UsesTokens
	daf.newTokenName = ability.TokenName
	daf.newTokensMax = ability.TokensMax
	if daf.newRefreshType == "" {
		daf.newRefreshType = "unlimited"
	}
	if daf.newDCType == "" {
		daf.newDCType = "none"
	}
	if len(ability.Effects) == 0 {
		return
	}
	eff := ability.Effects[0]
	daf.newEffectType = reverseEffectCategory(eff)
	daf.newDiceFormula = eff.DiceFormula
	daf.newDamageType = eff.DamageType
	daf.newMagnitude = eff.Magnitude
	daf.newDuration = eff.Duration
	daf.newTargetType = string(eff.TargetType)
	if daf.newTargetType == "" || daf.newTargetType == "single" {
		daf.newTargetType = "self"
	}
}

func (daf *DomainAbilityForm) buildAbility() *models.DomainAbility {
	category := resolveEffectCategory(daf.newEffectType)
	effect := models.FeatureEffect{TargetType: models.FeatureTargetSingle}
	if daf.baseAbility != nil && len(daf.baseAbility.Effects) > 0 && daf.baseAbility.Effects[0].Category == category {
		// Same kind of effect: keep reaction and target details the form cannot show
		effect = daf.baseAbility.Effects[0]
	}
	effect.Category = category
	effect.Magnitude = resistanceMagnitude(daf.newEffectType, daf.newMagnitude)
	effect.DiceFormula = daf.newDiceFormula
	effect.Duration = daf.newDuration
	effect.DamageType = daf.newDamageType
	if needsRecipient(daf.newEffectType) {
		effect.TargetType = models.FeatureTargetType(daf.newTargetType)
	}
	effects := []models.FeatureEffect{effect}
	if daf.baseAbility != nil && len(daf.baseAbility.Effects) > 1 {
		effects = append(effects, daf.baseAbility.Effects[1:]...)
	}

	return &models.DomainAbility{
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
		TokensMax:   daf.newTokensMax,
		Effects:     effects,
	}
}

func (daf *DomainAbilityForm) saveAbility(ctx app.Context) {
	ability := daf.buildAbility()
	if daf.editingIndex >= 0 && daf.editingIndex < len(daf.Abilities) {
		daf.Abilities[daf.editingIndex] = ability
	} else {
		daf.Abilities = append(daf.Abilities, ability)
	}
	daf.notify()
	daf.showForm = false
	daf.resetForm()
	ctx.Update()
}

func (daf *DomainAbilityForm) removeAbility(ctx app.Context, idx int) {
	if idx < 0 || idx >= len(daf.Abilities) {
		return
	}
	daf.Abilities = append(daf.Abilities[:idx], daf.Abilities[idx+1:]...)
	daf.notify()
	ctx.Update()
}

func (daf *DomainAbilityForm) notify() {
	if daf.OnChange != nil {
		daf.OnChange(daf.Abilities)
	}
}

// resetForm resets the form to default values
func (daf *DomainAbilityForm) resetForm() {
	daf.editingIndex = -1
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
	daf.newDuration = "until_cleared"
	daf.newDCType = "none"
	daf.newFixedDC = 12
	daf.newUsesTokens = false
	daf.newTokenName = ""
	daf.newTokensMax = 5
	daf.newTargetType = "self"
	daf.baseAbility = nil
	daf.selectedCard = customLibraryOption
}
