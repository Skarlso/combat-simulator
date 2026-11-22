package components

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/data"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
	"github.com/skarlso/daggerheart-combat-simulator/internal/parser"
)

// DomainAbilityForm manages a PC's abilities: domain cards, class and heritage features, or custom ones
type DomainAbilityForm struct {
	app.Compo

	Abilities []*models.DomainAbility
	OnChange  func([]*models.DomainAbility)
	// PCLevel reports the level of the PC being built; some library abilities scale with it
	PCLevel func() int

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
	newRollTrait     string
	newDiceFormula   string
	newDamageType    string
	newEffectType    string
	newMagnitude     int
	newDuration      string
	newUsesTokens    bool
	newTokenName     string
	newTokensMax     int
	newTargetType    string
	// The library entry or ability being edited; effects this form cannot show are kept from it
	baseAbility *models.DomainAbility

	library         []parser.LibraryAbility
	libraryClasses  []string
	showLibrary     bool
	librarySearch   string
	librarySource   string
	libraryClass    string
	libraryModelled bool
}

// Rows the library panel draws at once; the full SRD list is too long to scroll
const libraryRowLimit = 60

var (
	abilityTriggerOptions = []selectOption{
		{"", "None (manual)"}, {"on_attack", "On Attack"}, {"on_hit", "On Hit"}, {"on_critical", "On Critical"},
		{"on_miss", "On a failed attack"}, {"on_damage_roll", "On Damage Roll (adds to it)"},
		{"on_damage_dealt", "When your attack marks HP"},
		{"on_damage_taken", "On Damage Taken"}, {"on_hit_taken", "When an attack succeeds against you"},
		{"on_damage_incoming", "Before marking HP (softens the hit)"}, {"on_attacked", "When an attack would hit you (evade)"},
		{"on_evaded", "When an attack against you fails"}, {"on_ally_damaged", "When an ally would take damage"},
		{"start_combat", "Start of the Fight"}, {"start_turn", "Start of Turn"}, {"end_turn", "End of Turn"},
	}
	triggerFilterOptions = []selectOption{
		{"", "None"}, {"if_target_below_50_hp", "Target below 50% HP"}, {"if_hidden", "If hidden"}, {"if_ally_adjacent", "If ally adjacent"},
		{"if_target_vulnerable", "Target is Vulnerable"}, {"if_target_fully_stressed", "Target has all Stress marked"},
		{"if_dealt_severe", "You dealt Severe damage"}, {"if_took_severe", "You took Severe damage"},
		{"if_severe", "The hit is Severe"}, {"if_major_or_worse", "The hit is Major or worse"}, {"if_minor", "The hit is Minor"},
		{"if_vulnerable", "While you are Vulnerable"}, {"if_low_hp", "While at 2 or fewer HP"},
		{"if_fully_stressed", "While all your Stress is marked"}, {"if_hope_2", "While you hold 2 Hope"},
		{"if_out_of_armor", "While out of Armor Slots"}, {"if_rolled_fear", "You rolled with Fear"}, {"if_rolled_hope", "You rolled with Hope"},
	}
	refreshOptions = []selectOption{
		{"unlimited", "Unlimited"}, {"rest", "Per Rest"}, {"long_rest", "Per Long Rest"}, {"scene", "Per Scene"},
	}
	abilityEffectOptions = []selectOption{
		{"damage", "Damage"}, {"area_damage", "Area Damage (AOE)"}, {"heal", "Heal HP"},
		{"stress_inflict", "Inflict Stress"}, {"stress_clear", "Clear Stress"},
		{"apply_vulnerable", "Apply Vulnerable"}, {"apply_poison", "Apply Poison"}, {"apply_ignite", "Apply Burning"},
		{"apply_stunned", "Apply Stunned"}, {"apply_restrained", "Apply Restrained"},
		{"armor_damage", "Damage Armor Slots"}, {"armor_clear", "Clear Armor Slots"}, {"grant_armor", "Grant Extra Armor Slots"},
		{"damage_reduction", "Damage Reduction (Ward)"},
		{"damage_bonus", "Damage Bonus (passive)"}, {"damage_multiplier", "Damage Multiplier (on damage roll)"},
		{"attack_bonus", "Attack Roll Bonus (passive)"}, {"evasion_bonus", "Evasion Bonus (passive)"},
		{"severity_reduction", "Reduce Damage Severity (passive)"},
		{"mark_hp", "Target Marks HP"}, {"apply_advantage", "Attack with Advantage"},
		{"apply_disadvantage", "Target Attacks with Disadvantage"}, {"defense_penalty", "Lower Target's Difficulty"},
		{"token_place", "Place Token"}, {"token_spend", "Spend Token"}, {"token_on_damage", "Token Die (steps up on damage)"},
		{"resistance", "Resistance (50%)"}, {"immunity", "Immunity (100%)"},
		{"redirect_damage", "Shield an ally (take their hit)"},
	}
	rollTraitOptions  = append([]selectOption{{"", "Weapon trait"}, {"spellcast", "Spellcast trait"}}, traitOptions("")...)
	librarySourceTabs = []selectOption{
		{"", "All"}, {parser.SourceClass, "Class"}, {parser.SourceSubclass, "Subclass"}, {parser.SourceDomainCard, "Cards"},
		{parser.SourceAncestry, "Ancestry"}, {parser.SourceCommunity, "Community"}, {parser.SourceTransformation, "Transformation"},
	}
)

// OnMount initializes default values and loads the ability library
func (daf *DomainAbilityForm) OnMount(ctx app.Context) {
	daf.resetForm()
	library, err := data.LoadAbilityLibrary()
	if err != nil {
		app.Log("ability library failed to load:", err)
		return
	}
	daf.library = library
	for _, entry := range library {
		if entry.Source == parser.SourceClass && !slices.Contains(daf.libraryClasses, entry.Group) {
			daf.libraryClasses = append(daf.libraryClasses, entry.Group)
		}
	}
}

// Render renders the ability list, the library picker and the custom form
func (daf *DomainAbilityForm) Render() app.UI {
	return app.Div().Class("domain-ability-form").Body(
		app.Div().Class("ability-header").Body(
			app.Label().Text("Domain Cards and Features"),
			app.Div().Class("item-actions").Body(
				app.Button().Class(toggleClass("btn btn-small", daf.showLibrary, "active")).Text("Library").OnClick(func(ctx app.Context, e app.Event) {
					daf.showLibrary = !daf.showLibrary
					ctx.Update()
				}),
				app.Button().Class(toggleClass("btn btn-small", daf.showForm, "active")).Text("+ Custom").OnClick(func(ctx app.Context, e app.Event) {
					daf.showForm = !daf.showForm
					ctx.Update()
				}),
			),
		),
		app.If(daf.showLibrary, func() app.UI { return daf.renderLibraryPanel() }),
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

// abilityMeta is the one-line summary of when an ability works and what it costs
func abilityMeta(ability *models.DomainAbility) string {
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
	switch ability.DCType {
	case "target":
		uses += fmt.Sprintf(" | %s roll vs Difficulty", optionLabel(rollTraitOptions, ability.RollTrait))
	case "fixed":
		uses += fmt.Sprintf(" | %s roll (%d)", optionLabel(rollTraitOptions, ability.RollTrait), ability.FixedDC)
	}
	if ability.Passive {
		uses += " | Passive" + triggerSuffix(ability.Trigger)
	}
	if ability.UsesTokens {
		uses += fmt.Sprintf(" | %s, max %d", cmp.Or(ability.TokenName, "Tokens"), ability.TokensMax)
	}
	return cost + uses
}

func (daf *DomainAbilityForm) renderAbilityItem(ability *models.DomainAbility, idx int) app.UI {
	return editableItem(daf.editingIndex == idx, ability.Name, abilityMeta(ability), describeEffects(ability.Effects), ability.Description,
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
		app.If(daf.newDCType != "none", func() app.UI {
			return selectField("Roll with", daf.newRollTrait, rollTraitOptions, bindSelect(&daf.newRollTrait, false))
		}),
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

// libraryMatches applies the panel's search and filters to one entry
func (daf *DomainAbilityForm) libraryMatches(entry parser.LibraryAbility, query string) bool {
	if daf.librarySource != "" && entry.Source != daf.librarySource {
		return false
	}
	if daf.libraryClass != "" && !slices.Contains(entry.Classes, daf.libraryClass) {
		return false
	}
	if daf.libraryModelled && !entry.Modelled {
		return false
	}
	if query == "" {
		return true
	}
	for _, text := range []string{entry.Name, entry.Group, entry.Type, entry.Description} {
		if strings.Contains(strings.ToLower(text), query) {
			return true
		}
	}
	return false
}

// renderLibraryPanel renders the searchable picker over every SRD domain card and player feature
func (daf *DomainAbilityForm) renderLibraryPanel() app.UI {
	if len(daf.library) == 0 {
		return app.Div().Class("library-panel").Body(
			app.P().Class("empty-state").Text("Ability library failed to load."),
		)
	}

	query := strings.ToLower(strings.TrimSpace(daf.librarySearch))
	matches, modelled := 0, 0
	rows := make([]app.UI, 0, libraryRowLimit)
	for i, entry := range daf.library {
		if !daf.libraryMatches(entry, query) {
			continue
		}
		matches++
		if entry.Modelled {
			modelled++
		}
		if len(rows) < libraryRowLimit {
			rows = append(rows, daf.renderLibraryRow(i, entry))
		}
	}

	sourceTabs := make([]app.UI, len(librarySourceTabs))
	for i, tab := range librarySourceTabs {
		sourceTabs[i] = app.Button().
			Class(toggleClass("tab", daf.librarySource == tab.Value, "active")).
			Text(tab.Label).
			OnClick(func(ctx app.Context, e app.Event) {
				daf.librarySource = tab.Value
				ctx.Update()
			}, app.EventScope(tab.Label))
	}
	classOptions := []selectOption{{"", "Any class"}}
	for _, class := range daf.libraryClasses {
		classOptions = append(classOptions, selectOption{class, class})
	}

	count := fmt.Sprintf("%d of %d abilities, %d modelled", matches, len(daf.library), modelled)
	if matches > len(rows) {
		count += fmt.Sprintf(". Showing the first %d: narrow the search to see the rest.", len(rows))
	}
	var list app.UI = app.Div().Class("empty-state").Text("No abilities match.")
	if len(rows) > 0 {
		list = app.Div().Class("library-list").Body(rows...)
	}

	return app.Div().Class("library-panel").Body(
		app.Div().Class("library-filters").Body(
			app.Input().
				Type("search").
				Placeholder("Search name, domain, class or rules text").
				Value(daf.librarySearch).
				OnInput(func(ctx app.Context, e app.Event) {
					daf.librarySearch = ctx.JSSrc().Get("value").String()
					ctx.Update()
				}),
			app.Select().
				OnChange(func(ctx app.Context, e app.Event) {
					daf.libraryClass = selectedOption(ctx, classOptions)
					ctx.Update()
				}).
				Body(optionElements(classOptions, daf.libraryClass)...),
			app.Label().Class("checkbox-label").Body(
				app.Input().Type("checkbox").Checked(daf.libraryModelled).OnChange(func(ctx app.Context, e app.Event) {
					daf.libraryModelled = ctx.JSSrc().Get("checked").Bool()
					ctx.Update()
				}),
				app.Text(" Modelled only"),
			),
		),
		app.Div().Class("library-filters").Body(app.Div().Class("tabs tabs-compact").Body(sourceTabs...)),
		app.Div().Class("library-count").Text(count),
		list,
	)
}

// libraryTag names where an entry comes from: "Valor 1 Ability", "Guardian: Stalwart Foundation"
func libraryTag(entry parser.LibraryAbility) string {
	switch entry.Source {
	case parser.SourceDomainCard:
		return fmt.Sprintf("%s %d %s", entry.Group, entry.Level, entry.Type)
	case parser.SourceSubclass:
		return fmt.Sprintf("%s: %s %s", strings.Join(entry.Classes, "/"), entry.Group, entry.Type)
	default:
		return entry.Group + " " + entry.Type
	}
}

func (daf *DomainAbilityForm) renderLibraryRow(idx int, entry parser.LibraryAbility) app.UI {
	status := app.Span().Class("tag tag-muted").Text("Text only")
	if entry.Modelled {
		status = app.Span().Class("tag tag-modelled").Text("Modelled")
	}
	// go-app matches handlers by function pointer, so a captured index goes stale
	// when filtering reorders rows; the DOM attribute is always current.
	pick := func(use func(app.Context, parser.LibraryAbility)) func(app.Context, app.Event) {
		return func(ctx app.Context, e app.Event) {
			i, err := strconv.Atoi(ctx.JSSrc().Get("dataset").Get("idx").String())
			if err == nil && i >= 0 && i < len(daf.library) {
				use(ctx, daf.library[i])
			}
		}
	}

	return app.Div().Class("library-row").Body(
		app.Div().Class("library-main").Body(
			app.Div().Class("item-title").Body(
				app.Text(entry.Name),
				app.Span().Class("tag").Text(libraryTag(entry)),
				status,
			),
			app.Div().Class("ability-desc").Text(entry.Description),
			app.If(entry.Modelled, func() app.UI {
				return app.Div().Class("ability-effect").Text(joinParts(" ",
					"Simulated as: "+abilityMeta(&entry.Ability)+" | "+describeEffects(entry.Ability.Effects)+".", entry.Note))
			}),
		),
		app.Div().Class("item-actions").Body(
			app.If(entry.Modelled, func() app.UI {
				return app.Button().Class("btn btn-small btn-primary").Text("Add").DataSet("idx", idx).OnClick(pick(daf.addFromLibrary), app.EventScope("add"))
			}),
			app.Button().Class("btn btn-small").Text("Customize").DataSet("idx", idx).OnClick(pick(daf.customizeFromLibrary), app.EventScope("customize")),
		),
	)
}

func (daf *DomainAbilityForm) libraryAbility(entry parser.LibraryAbility) *models.DomainAbility {
	level := 1
	if daf.PCLevel != nil {
		level = daf.PCLevel()
	}
	ability := entry.AbilityAt(level)
	return &ability
}

func (daf *DomainAbilityForm) addFromLibrary(ctx app.Context, entry parser.LibraryAbility) {
	daf.Abilities = append(daf.Abilities, daf.libraryAbility(entry))
	daf.notify()
	ctx.Update()
}

// customizeFromLibrary opens the form on a library entry, so a text-only one can be given an effect
func (daf *DomainAbilityForm) customizeFromLibrary(ctx app.Context, entry parser.LibraryAbility) {
	daf.resetForm()
	daf.loadAbility(daf.libraryAbility(entry))
	daf.showForm = true
	daf.showLibrary = false
	ctx.Update()
}

func optionLabel(options []selectOption, value string) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}
	return value
}

// triggerSuffix names a passive's trigger and filter: ": On Hit, Target is Vulnerable"
func triggerSuffix(trigger models.FeatureTriggerCondition) string {
	var parts []string
	if trigger.Type != "" {
		parts = append(parts, optionLabel(abilityTriggerOptions, string(trigger.Type)))
	}
	if trigger.Filter != "" {
		for _, filter := range strings.Split(trigger.Filter, "+") {
			parts = append(parts, optionLabel(triggerFilterOptions, filter))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return ": " + strings.Join(parts, ", ")
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
	daf.newRollTrait = ability.RollTrait
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
	// Only fields the form shows for this effect type are written; hidden defaults must not leak in
	if needsMagnitude(daf.newEffectType) || oneOf(daf.newEffectType, "resistance", "immunity") {
		effect.Magnitude = resistanceMagnitude(daf.newEffectType, daf.newMagnitude)
	}
	if needsDiceFormula(daf.newEffectType) {
		effect.DiceFormula = daf.newDiceFormula
	}
	if needsDuration(daf.newEffectType) {
		effect.Duration = daf.newDuration
	}
	if needsDamageType(daf.newEffectType) {
		effect.DamageType = daf.newDamageType
	}
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
		RollTrait:   daf.newRollTrait,
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
	daf.newRollTrait = ""
	daf.newUsesTokens = false
	daf.newTokenName = ""
	daf.newTokensMax = 5
	daf.newTargetType = "self"
	daf.baseAbility = nil
}
