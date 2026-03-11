package components

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

func validateDiceNotation(s string) string {
	if s == "" {
		return ""
	}
	_, err := models.ParseDiceNotation(s)
	if err != nil {
		return "Invalid dice notation (e.g. 2d6+3)"
	}
	return ""
}

func renderValidationMsg(msg string) app.UI {
	if msg == "" {
		return app.Text("")
	}
	return app.Span().Class("validation-error").Text(msg)
}

func formGroupClass(hasError bool) string {
	if hasError {
		return "form-group has-error"
	}
	return "form-group"
}

func handleIntInput(target *int) func(app.Context, app.Event) {
	return func(ctx app.Context, e app.Event) {
		*target = ctx.JSSrc().Get("valueAsNumber").Int()
	}
}

var uiEffectCategoryMap = map[string]models.FeatureEffectCategory{
	"damage":           models.FeatureEffectDamage,
	"direct_damage":    models.FeatureEffectDirectDamage,
	"area_damage":      models.FeatureEffectAreaDamage,
	"heal":             models.FeatureEffectHPHeal,
	"stress_inflict":   models.FeatureEffectStressInflict,
	"stress_clear":     models.FeatureEffectStressClear,
	"apply_vulnerable": models.FeatureEffectApplyVulnerable,
	"apply_poison":     models.FeatureEffectApplyPoison,
	"armor_damage":     models.FeatureEffectArmorDamage,
	"grant_armor":      models.FeatureEffectGrantArmor,
	"damage_reduction": models.FeatureEffectDamageReduction,
	"token_place":      models.FeatureEffectTokenPlace,
	"token_spend":      models.FeatureEffectTokenSpend,
	"resistance":       models.FeatureEffectDamageResistance,
	"immunity":         models.FeatureEffectDamageResistance,
	"summon_allies":    models.FeatureEffectSummonAllies,
}

func resolveEffectCategory(effectType string) models.FeatureEffectCategory {
	if cat, ok := uiEffectCategoryMap[effectType]; ok {
		return cat
	}
	return models.FeatureEffectDamage
}

func reverseEffectCategory(cat models.FeatureEffectCategory) string {
	for key, val := range uiEffectCategoryMap {
		if val == cat {
			return key
		}
	}
	return "damage"
}

func renderAccordionHeader(label string, open bool, toggle func(app.Context, app.Event)) app.UI {
	indicatorClass := "accordion-indicator"
	if open {
		indicatorClass += " open"
	}
	return app.Div().
		Class("accordion-header").
		OnClick(toggle).
		Body(
			app.H5().Text(label),
			app.Span().Class(indicatorClass).Text("\u25B6"),
		)
}

func needsDamageType(effectType string) bool {
	return effectType == "damage" || effectType == "direct_damage" || effectType == "resistance" || effectType == "immunity"
}

func needsDuration(effectType string) bool {
	return effectType == "apply_vulnerable" || effectType == "apply_poison"
}

func needsDiceFormula(effectType string) bool {
	return effectType == "damage" ||
		effectType == "direct_damage" ||
		effectType == "area_damage" ||
		effectType == "heal" ||
		effectType == "damage_reduction"
}

func needsMagnitude(effectType string) bool {
	return effectType == "stress_inflict" ||
		effectType == "stress_clear" ||
		effectType == "armor_damage" ||
		effectType == "grant_armor" ||
		effectType == "token_place" ||
		effectType == "token_spend"
}
