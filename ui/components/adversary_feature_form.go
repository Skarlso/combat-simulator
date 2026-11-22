package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/combat-simulator/data"
	"github.com/skarlso/combat-simulator/internal/models"
	"github.com/skarlso/combat-simulator/internal/parser"
)

const customLibraryOption = "__custom__"

// AdversaryFeatureForm manages adding/editing adversary features (enemy abilities)
type AdversaryFeatureForm struct {
	app.Compo

	Features             []*models.AdversaryFeature
	OnChange             func([]*models.AdversaryFeature)
	AvailableAdversaries []*models.Combatant

	showForm         bool
	editingIndex     int
	newName          string
	newDescription   string
	newPassive       bool
	newFearCost      int
	newStressCost    int
	newCostType      string
	newSpotlightLim  int
	newUsageType     string
	newMaxUses       int
	newDiceFormula   string
	newDamageType    string
	newEffectType    string
	newMagnitude     int
	newArmorCost     int
	newFallbackFear  int
	newReactionTrait string
	newReactionDC    int
	newHalfOnSuccess bool
	newTrigger       string
	newDuration      string
	newDCType        string
	newFixedDC       int
	newCostOnSuccess bool
	// The library entry or feature being edited; fields this form cannot show are kept from it
	baseFeature       *models.AdversaryFeature
	newQuantityType   string
	selectedForSummon map[string]bool

	newUsesTokens bool
	newTokenName  string
	newTokensMax  int

	featureLibrary     *parser.FeatureParser
	libraryFeatureKeys []string
	selectedLibraryKey string
}

var (
	featureTriggerOptions = []selectOption{
		{"on_hit", "On Hit (own attack lands)"}, {"on_attack", "On Attack (before rolling)"}, {"on_critical", "On Critical"},
		{"on_hit_taken", "On Being Hit"}, {"on_damage_taken", "On Taking Damage"}, {"start_turn", "Start of Turn"}, {"end_turn", "End of Turn"},
	}
	costTypeOptions  = []selectOption{{"free", "Free"}, {"fear", "Fear"}, {"stress", "Stress"}}
	usageTypeOptions = []selectOption{
		{"unlimited", "Unlimited"}, {"once_per_scene", "Once per Scene"}, {"once_per_session", "Once per Session"}, {"once_per_rest", "Once per Rest"},
	}
	featureEffectOptions = []selectOption{
		{"damage", "Damage"}, {"direct_damage", "Direct Damage"}, {"area_damage", "Area Damage (AOE)"},
		{"stress_inflict", "Inflict Stress"}, {"apply_vulnerable", "Apply Vulnerable"}, {"apply_poison", "Apply Poison"},
		{"apply_ignite", "Apply Burning"}, {"apply_stunned", "Apply Stunned"}, {"apply_restrained", "Apply Restrained"},
		{"armor_damage", "Damage Armor"}, {"grant_armor", "Grant Armor Slots"}, {"apply_taunt", "Apply Taunt"},
		{"fear_generate", "GM Gains Fear"}, {"hope_drain", "Drain Hope"}, {"spotlight_modifier", "Spotlight Modifier"},
		{"resistance", "Resistance (50%)"}, {"immunity", "Immunity (100%)"}, {"summon_allies", "Summon Allies"},
		{"token_place", "Place Token"}, {"token_spend", "Spend Token"},
	}
	quantityTypeOptions = []selectOption{{"fixed", "Fixed"}, {"dice", "Dice Roll"}}
)

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
	return app.Div().Class("adversary-feature-form").Body(
		app.Div().Class("ability-header").Body(
			app.Label().Text("Adversary Features"),
			app.Button().Class("btn btn-small").Text("+ Add Feature").OnClick(func(ctx app.Context, e app.Event) {
				aff.showForm = !aff.showForm
				ctx.Update()
			}),
		),
		aff.renderFeatureList(),
		app.If(aff.showForm, func() app.UI { return aff.renderAddForm() }),
	)
}

func (aff *AdversaryFeatureForm) renderFeatureList() app.UI {
	if len(aff.Features) == 0 {
		return app.Div().Class("empty-abilities").Text("No features added")
	}
	items := make([]app.UI, len(aff.Features))
	for i, feature := range aff.Features {
		items[i] = aff.renderFeatureItem(feature, i)
	}
	return app.Div().Class("ability-list").Body(items...)
}

func (aff *AdversaryFeatureForm) renderFeatureItem(feature *models.AdversaryFeature, idx int) app.UI {
	typeText := "Active"
	if feature.Passive {
		typeText = "Passive"
	}
	limit, uses := "", ""
	if feature.SpotlightLimit > 0 {
		limit = fmt.Sprintf("Max %d/turn", feature.SpotlightLimit)
	}
	if feature.MaxUses > 0 {
		uses = fmt.Sprintf("%d uses (%s)", feature.MaxUses, feature.UsageType)
	}
	cost := joinParts(" | ", joinParts(", ", countText(feature.FearCost, "Fear"), countText(feature.StressCost, "Stress")), limit, uses)
	if cost == "" {
		cost = "No cost"
	}
	effectText := ""
	if len(feature.Effects) > 0 {
		effectText = describeEffect(feature.Effects[0])
	}

	return editableItem(aff.editingIndex == idx, feature.Name, typeText+" | "+cost, effectText, feature.Description,
		func(ctx app.Context, e app.Event) { aff.startEditing(ctx, idx) },
		func(ctx app.Context, e app.Event) { aff.removeFeature(ctx, idx) },
	)
}

// renderAddForm renders the feature creation form
func (aff *AdversaryFeatureForm) renderAddForm() app.UI {
	formTitle, saveText := "Add Adversary Feature", "Add Feature"
	if aff.editingIndex >= 0 {
		formTitle, saveText = "Edit Adversary Feature", "Save"
	}
	diceErr := validateDiceNotation(aff.newDiceFormula)
	costOnSuccessShown := aff.newCostType != "free" && aff.newDCType != "none" && aff.newDCType != ""

	return app.Div().Class("ability-add-form").Body(
		app.H4().Text(formTitle),
		app.If(aff.editingIndex < 0 && len(aff.libraryFeatureKeys) > 0, func() app.UI { return aff.renderLibraryPicker() }),
		app.Div().Class("form-row").Body(
			textField("Name", "Venomous Stinger", &aff.newName, false),
			textField("Description", "Injects venom...", &aff.newDescription, false),
		),
		checkboxField("Passive (fires on its trigger, never chosen as an action)", &aff.newPassive),
		app.If(aff.newPassive, func() app.UI {
			return selectField("Trigger", aff.newTrigger, featureTriggerOptions, bindSelect(&aff.newTrigger, false))
		}),
		app.Div().Class("form-subsection").Body(
			app.H5().Text("Cost & Usage"),
			app.Div().Class("form-row-3").Body(
				selectField("Cost Type", aff.newCostType, costTypeOptions, bindSelect(&aff.newCostType, true)),
				app.If(aff.newCostType == "fear", func() app.UI { return numberField("Fear Cost", &aff.newFearCost, 1, 10) }),
				app.If(aff.newCostType == "stress", func() app.UI { return numberField("Stress Cost", &aff.newStressCost, 1, 10) }),
				app.If(costOnSuccessShown, func() app.UI { return checkboxField("Only paid on a success", &aff.newCostOnSuccess) }),
				numberField("Spotlight Limit", &aff.newSpotlightLim, 0, 5),
			),
			app.Div().Class("form-row").Body(
				selectField("Usage Limit", aff.newUsageType, usageTypeOptions, bindSelect(&aff.newUsageType, true)),
				app.If(aff.newUsageType != "unlimited", func() app.UI { return numberField("Max Uses", &aff.newMaxUses, 1, 10) }),
			),
		),
		tokenFields(&aff.newUsesTokens, &aff.newTokenName, &aff.newTokensMax),
		app.Div().Class("form-subsection").Body(aff.renderEffectSection(diceErr)...),
		dcFields(&aff.newDCType, &aff.newFixedDC, "Target's Evasion"),
		formActions(saveText, aff.newName == "" || diceErr != "" || aff.summonPoolEmpty(),
			func(ctx app.Context, e app.Event) { aff.saveFeature(ctx) },
			func(ctx app.Context, e app.Event) {
				aff.showForm = false
				aff.resetForm()
				ctx.Update()
			},
		),
	)
}

func (aff *AdversaryFeatureForm) renderEffectSection(diceErr string) []app.UI {
	effectType := aff.newEffectType
	reacting := needsReaction(effectType) && aff.newReactionTrait != ""

	body := []app.UI{
		app.H5().Text("Effect"),
		selectField("Effect Type", effectType, featureEffectOptions, bindSelect(&aff.newEffectType, true)),
	}
	body = append(body, effectFields(effectType, &aff.newDiceFormula, diceErr, &aff.newMagnitude, &aff.newDamageType, &aff.newDuration)...)
	body = append(body,
		app.If(needsArmorCost(effectType), func() app.UI { return numberField("Armor Slots to Reduce", &aff.newArmorCost, 1, 3) }),
		app.If(effectType == "armor_damage", func() app.UI { return numberField("Fear Gained if No Slot Left", &aff.newFallbackFear, 0, 3) }),
		app.If(needsReaction(effectType), func() app.UI {
			options := []selectOption{{"", traitLabel("")}}
			for _, trait := range reactionTraits {
				options = append(options, selectOption{trait, traitLabel(trait)})
			}
			return selectField("Reaction Roll to Avoid", aff.newReactionTrait, options, bindSelect(&aff.newReactionTrait, true))
		}),
		app.If(reacting, func() app.UI {
			return numberField("Reaction Difficulty (0 = this adversary's Difficulty)", &aff.newReactionDC, 0, 30)
		}),
		app.If(reacting && isDamageEffect(effectType), func() app.UI {
			return checkboxField("Success takes half damage (otherwise none)", &aff.newHalfOnSuccess)
		}),
		app.If(effectType == "summon_allies", func() app.UI { return aff.renderSummonControls() }),
		app.If(aff.baseFeature != nil && len(aff.baseFeature.Effects) > 1, func() app.UI {
			return app.Div().Class("form-hint").Text(aff.keptEffectsNote())
		}),
	)
	return body
}

// renderSummonControls renders the summon-specific form controls
func (aff *AdversaryFeatureForm) renderSummonControls() app.UI {
	return app.Div().Class("summon-controls").Body(
		app.H5().Text("Summon Configuration"),
		app.Div().Class("form-row").Body(
			selectField("Quantity Type", aff.newQuantityType, quantityTypeOptions, bindSelect(&aff.newQuantityType, true)),
			app.If(aff.newQuantityType == "fixed", func() app.UI { return numberField("Quantity", &aff.newMagnitude, 1, 10) }),
			app.If(aff.newQuantityType == "dice", func() app.UI { return textField("Dice Formula", "1d4", &aff.newDiceFormula, false) }),
		),
		app.Div().Class("form-group").Body(
			app.Label().Text("Summon Pool (select adversaries)"),
			app.If(len(aff.AvailableAdversaries) == 0, func() app.UI {
				return app.Div().Class("empty-pool").Text("No adversaries defined yet. Add adversaries first.")
			}),
			app.If(len(aff.AvailableAdversaries) > 0, func() app.UI { return aff.renderSummonPoolPicker() }),
		),
	)
}

func (aff *AdversaryFeatureForm) renderSummonPoolPicker() app.UI {
	items := make([]app.UI, len(aff.AvailableAdversaries))
	for i, adv := range aff.AvailableAdversaries {
		items[i] = app.Label().Class("checkbox-label").Body(
			app.Input().Type("checkbox").Checked(aff.selectedForSummon[adv.ID]).OnChange(func(ctx app.Context, e app.Event) {
				if aff.selectedForSummon == nil {
					aff.selectedForSummon = make(map[string]bool)
				}
				aff.selectedForSummon[adv.ID] = ctx.JSSrc().Get("checked").Bool()
				ctx.Update()
			}, app.EventScope(adv.ID)),
			app.Text(" "+adv.Name),
		)
	}
	return app.Div().Class("summon-pool-picker").Body(items...)
}

func (aff *AdversaryFeatureForm) renderLibraryPicker() app.UI {
	options := []selectOption{{customLibraryOption, "-- Custom Feature --"}}
	for _, key := range aff.libraryFeatureKeys {
		options = append(options, selectOption{key, key})
	}
	current := aff.selectedLibraryKey
	if current == "" {
		current = customLibraryOption
	}

	return app.Div().Class("form-subsection").Body(
		app.H5().Text("Feature Library"),
		selectField(fmt.Sprintf("Start from template (%d available)", len(aff.libraryFeatureKeys)), current, options,
			func(ctx app.Context, key string) {
				aff.selectedLibraryKey = key
				aff.populateFromLibrary(key)
				ctx.Update()
			}),
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
	aff.loadFeature(feat)
	// Keys may be "Adversary::Feature"; the feature keeps only its own name
	if _, name, found := strings.Cut(key, "::"); found {
		aff.newName = name
	}
}

func (aff *AdversaryFeatureForm) startEditing(ctx app.Context, idx int) {
	if idx < 0 || idx >= len(aff.Features) {
		return
	}
	aff.editingIndex = idx
	aff.showForm = true
	aff.loadFeature(aff.Features[idx])
	ctx.Update()
}

// loadFeature fills the form from a feature and remembers it, so saving keeps what the form cannot show
func (aff *AdversaryFeatureForm) loadFeature(feature *models.AdversaryFeature) {
	base := *feature
	base.Effects = append([]models.FeatureEffect(nil), feature.Effects...)
	aff.baseFeature = &base

	aff.newName = feature.Name
	aff.newDescription = feature.Description
	aff.newPassive = feature.Passive
	aff.newTrigger = string(feature.Trigger.Type)
	if aff.newTrigger == "" {
		aff.newTrigger = string(models.TriggerOnHit)
	}
	aff.newFearCost = feature.FearCost
	aff.newStressCost = feature.StressCost
	aff.newCostOnSuccess = feature.CostOnSuccess
	aff.newSpotlightLim = feature.SpotlightLimit
	aff.newUsageType = feature.UsageType
	if aff.newUsageType == "" {
		aff.newUsageType = "unlimited"
	}
	aff.newMaxUses = feature.MaxUses
	aff.newDCType = feature.DCType
	if aff.newDCType == "" {
		aff.newDCType = "none"
	}
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

	if len(feature.Effects) == 0 {
		return
	}
	eff := feature.Effects[0]
	aff.newEffectType = reverseEffectCategory(eff)
	aff.newDiceFormula = eff.DiceFormula
	aff.newDamageType = eff.DamageType
	aff.newMagnitude = eff.Magnitude
	aff.newDuration = eff.Duration
	if needsDuration(aff.newEffectType) && (aff.newDuration == "" || aff.newDuration == "instant") {
		aff.newDuration = "until_cleared"
	}
	aff.newArmorCost = max(eff.ArmorCost, 1)
	aff.newFallbackFear = eff.FallbackFear
	aff.newReactionTrait = eff.ReactionTrait
	aff.newReactionDC = eff.ReactionDC
	aff.newHalfOnSuccess = eff.HalfOnSuccess
	if eff.Category == models.FeatureEffectSummonAllies {
		aff.newQuantityType = "fixed"
		if eff.DiceFormula != "" {
			aff.newQuantityType = "dice"
		}
		aff.selectedForSummon = make(map[string]bool)
		for _, s := range eff.SummonPool {
			aff.selectedForSummon[s.ID] = true
		}
	}
}

func (aff *AdversaryFeatureForm) keptEffectsNote() string {
	names := make([]string, 0, len(aff.baseFeature.Effects)-1)
	for _, e := range aff.baseFeature.Effects[1:] {
		names = append(names, string(e.Category))
	}
	return fmt.Sprintf("Also keeps %d more effect(s) from the template, not editable here: %s", len(names), strings.Join(names, ", "))
}

// summonPoolEmpty reports a summon effect with nobody to summon, which would fail on every activation
func (aff *AdversaryFeatureForm) summonPoolEmpty() bool {
	if aff.newEffectType != "summon_allies" {
		return false
	}
	for _, picked := range aff.selectedForSummon {
		if picked {
			return false
		}
	}
	return aff.baseFeature == nil || len(aff.baseFeature.Effects) == 0 || aff.baseFeature.Effects[0].SummonType == ""
}

func (aff *AdversaryFeatureForm) buildFeature() *models.AdversaryFeature {
	effectCategory := resolveEffectCategory(aff.newEffectType)

	magnitude := resistanceMagnitude(aff.newEffectType, aff.newMagnitude)
	diceFormula := aff.newDiceFormula
	if aff.newEffectType == "summon_allies" {
		if aff.newQuantityType == "fixed" {
			diceFormula = ""
		} else {
			magnitude = 0
		}
	}

	effect := models.FeatureEffect{TargetType: models.FeatureTargetSingle}
	if aff.baseFeature != nil && len(aff.baseFeature.Effects) > 0 && aff.baseFeature.Effects[0].Category == effectCategory {
		// Same kind of effect: keep target type, standard attack and summon type the form cannot show
		effect = aff.baseFeature.Effects[0]
		if diceFormula != "" {
			effect.StandardAttack = false
		}
	}
	effect.Category = effectCategory
	effect.Magnitude = magnitude
	effect.DiceFormula = diceFormula
	effect.Duration = aff.newDuration
	effect.DamageType = aff.newDamageType
	if effectCategory == models.FeatureEffectAreaDamage {
		effect.TargetType = models.FeatureTargetAOE
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
		if len(effect.SummonPool) > 0 {
			effect.SummonType = ""
		}
	}

	effects := []models.FeatureEffect{effect}
	if aff.baseFeature != nil && len(aff.baseFeature.Effects) > 1 {
		effects = append(effects, aff.baseFeature.Effects[1:]...)
	}

	fearCost, stressCost := 0, 0
	switch aff.newCostType {
	case "fear":
		fearCost = aff.newFearCost
	case "stress":
		stressCost = aff.newStressCost
	}

	maxUses := 0
	if aff.newUsageType != "unlimited" {
		maxUses = aff.newMaxUses
	}

	trigger := models.FeatureTriggerCondition{}
	if aff.newPassive {
		trigger.Type = models.FeatureTriggerType(aff.newTrigger)
		if aff.baseFeature != nil && aff.baseFeature.Trigger.Type == trigger.Type {
			trigger.Filter = aff.baseFeature.Trigger.Filter
		}
	}

	return &models.AdversaryFeature{
		Name:           aff.newName,
		Description:    aff.newDescription,
		Passive:        aff.newPassive,
		Trigger:        trigger,
		FearCost:       fearCost,
		StressCost:     stressCost,
		CostOnSuccess:  aff.newCostOnSuccess && aff.newCostType != "free",
		SpotlightLimit: aff.newSpotlightLim,
		UsageType:      aff.newUsageType,
		MaxUses:        maxUses,
		UsesLeft:       maxUses,
		DCType:         aff.newDCType,
		FixedDC:        aff.newFixedDC,
		UsesTokens:     aff.newUsesTokens,
		TokenName:      aff.newTokenName,
		TokensMax:      aff.newTokensMax,
		Effects:        effects,
	}
}

func (aff *AdversaryFeatureForm) saveFeature(ctx app.Context) {
	feature := aff.buildFeature()
	if aff.editingIndex >= 0 && aff.editingIndex < len(aff.Features) {
		aff.Features[aff.editingIndex] = feature
	} else {
		aff.Features = append(aff.Features, feature)
	}
	aff.notify()
	aff.showForm = false
	aff.resetForm()
	ctx.Update()
}

func (aff *AdversaryFeatureForm) removeFeature(ctx app.Context, idx int) {
	if idx < 0 || idx >= len(aff.Features) {
		return
	}
	aff.Features = append(aff.Features[:idx], aff.Features[idx+1:]...)
	aff.notify()
	ctx.Update()
}

func (aff *AdversaryFeatureForm) notify() {
	if aff.OnChange != nil {
		aff.OnChange(aff.Features)
	}
}

// resetForm resets the form to default values
func (aff *AdversaryFeatureForm) resetForm() {
	aff.editingIndex = -1
	aff.newName = ""
	aff.newDescription = ""
	aff.newPassive = false
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
	aff.newTrigger = string(models.TriggerOnHit)
	aff.newDuration = "until_cleared"
	aff.newDCType = "none"
	aff.newFixedDC = 12
	aff.newCostOnSuccess = false
	aff.baseFeature = nil
	aff.newQuantityType = "fixed"
	aff.selectedForSummon = make(map[string]bool)
	aff.newUsesTokens = false
	aff.newTokenName = ""
	aff.newTokensMax = 0
	aff.selectedLibraryKey = customLibraryOption
}
