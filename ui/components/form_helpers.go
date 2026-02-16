package components

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

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

func needsDamageType(effectType string) bool {
	return effectType == "damage" || effectType == "direct_damage" || effectType == "resistance" || effectType == "immunity"
}

func needsDuration(effectType string) bool {
	return effectType == "apply_vulnerable" || effectType == "apply_poison"
}
