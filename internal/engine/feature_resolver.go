package engine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/skarlso/daggerheart-combat-simulator/internal/models"
)

// DCCheckResult contains the details of a DC check roll
type DCCheckResult struct {
	Success        bool
	HopeDie        int // Player's hope die (0 for enemies)
	FearDie        int // Player's fear die (0 for enemies)
	D20Roll        int // Enemy's d20 roll (0 for players)
	TotalRoll      int
	DC             int
	SpotlightShift bool // True if roll causes spotlight to shift (Fear result for players)
	IsCritical     bool
	VsDefense      bool // True if this was an attack roll against a target's Evasion/Difficulty
	// Who this activation's area effects can reach, picked once so every effect agrees
	areaCandidates []*models.Combatant
	// Reaction rolls made during this activation, one per target, in the order rolled
	reactions     map[*models.Combatant]reactionResult
	Reactions     []ReactionRoll
	HopeGenerated int
	FearGenerated int
	// RedirectedTo is the ally who stepped in and took the damage instead of the target
	RedirectedTo *models.Combatant
	// What was added to the dice, and what made up the damage roll beyond its own dice
	RollParts     []models.LogPart
	DamageParts   []models.LogPart
	DamageFormula string
	// Damage tracking (populated during effect resolution)
	TargetsHit      int                    // Targets an area effect damaged
	DamageDealt     int                    // HP damage actually dealt to target
	DamageRoll      int                    // Raw damage before resistance/threshold
	ProcessedDamage int                    // Damage after resistance/immunity/vulnerability
	Threshold       models.DamageThreshold // Threshold category of damage
	DamageResult    *models.DamageResult   // Full damage result with armor/HP tracking
}

// hits reports whether the roll succeeds against one target of a multi-target attack.
// SRD: one attack roll, compared to each target separately.
func (r *DCCheckResult) hits(target *models.Combatant) bool {
	if r == nil || !r.VsDefense {
		return true
	}
	return r.IsCritical || r.TotalRoll >= target.GetDefense()
}

func isAreaEffect(effect models.FeatureEffect) bool {
	return effect.Category == models.FeatureEffectAreaDamage || effect.TargetType == models.FeatureTargetAOE
}

// pickAreaCandidates chooses who an area effect can reach. There are no ranges, so
// the area is assumed to catch the chosen target plus random others, up to areaTargetLimit.
func (fr *FeatureResolver) pickAreaCandidates(actor, target *models.Combatant, state *models.CombatState) []*models.Combatant {
	pool := make([]*models.Combatant, 0)
	for _, t := range state.GetEnemiesOf(actor) {
		if t.IsAlive() {
			pool = append(pool, t)
		}
	}
	if fr.areaTargetLimit <= 0 || len(pool) <= fr.areaTargetLimit {
		return pool
	}

	picked := make([]*models.Combatant, 0, fr.areaTargetLimit)
	for i, t := range pool {
		if t == target {
			picked = append(picked, t)
			pool = append(pool[:i], pool[i+1:]...)
			break
		}
	}
	for len(picked) < fr.areaTargetLimit {
		n := len(pool)
		i := ((fr.dice.RollDice(1, n, 0)-1)%n + n) % n
		picked = append(picked, pool[i])
		pool = append(pool[:i], pool[i+1:]...)
	}
	return picked
}

// areaTargets returns the area candidates that the activation's attack roll hits
func (fr *FeatureResolver) areaTargets(actor, target *models.Combatant, state *models.CombatState, result *DCCheckResult) []*models.Combatant {
	var candidates []*models.Combatant
	if result != nil {
		if result.areaCandidates == nil {
			result.areaCandidates = fr.pickAreaCandidates(actor, target, state)
		}
		candidates = result.areaCandidates
	} else {
		candidates = fr.pickAreaCandidates(actor, target, state)
	}

	targets := make([]*models.Combatant, 0, len(candidates))
	for _, t := range candidates {
		if t.IsAlive() && result.hits(t) {
			targets = append(targets, t)
		}
	}
	return targets
}

// resolveEffects applies a feature's effects after its roll. When the roll missed the
// chosen target, only area effects go ahead, and only if they hit someone else.
func (fr *FeatureResolver) resolveEffects(
	name string,
	effects []models.FeatureEffect,
	actor, target *models.Combatant,
	state *models.CombatState,
	result *DCCheckResult,
) error {
	primaryHit := result.Success
	if !primaryHit {
		hasArea := false
		for _, effect := range effects {
			hasArea = hasArea || isAreaEffect(effect)
		}
		if !hasArea || !result.VsDefense || len(fr.areaTargets(actor, target, state, result)) == 0 {
			if result.VsDefense && target != nil {
				fr.ProcessTriggeredFeatures(target, actor, state, models.TriggerOnEvaded, false)
			}
			return fmt.Errorf("%s: %w", name, ErrFeatureMissed)
		}
		// The roll still succeeded against someone, so only a Fear result passes the spotlight
		result.SpotlightShift = result.FearGenerated > 0
	}

	for _, effect := range effects {
		if !primaryHit && !isAreaEffect(effect) {
			continue
		}
		if err := fr.ResolveEffect(effect, actor, target, state, result); err != nil {
			return err
		}
	}

	if primaryHit && result.VsDefense && target != nil {
		fr.ProcessTriggeredFeatures(target, actor, state, models.TriggerOnHitTaken, false)
	}
	return nil
}

// FeatureResolver handles resolution of adversary features and domain abilities
type FeatureResolver struct {
	dice     DiceRoller
	registry AdversaryRegistry // Optional: for summoning support
	// Most targets one area effect reaches (0 = everyone)
	areaTargetLimit int
}

// SetAreaTargetLimit caps how many targets a single area effect reaches
func (fr *FeatureResolver) SetAreaTargetLimit(limit int) {
	fr.areaTargetLimit = limit
}

// SetAdversaryRegistry sets the registry for summoning support
func (fr *FeatureResolver) SetAdversaryRegistry(registry AdversaryRegistry) {
	fr.registry = registry
}

// ResolveEffect applies a single effect to target(s)
func (fr *FeatureResolver) ResolveEffect(
	effect models.FeatureEffect,
	actor *models.Combatant,
	target *models.Combatant,
	state *models.CombatState,
	result *DCCheckResult,
) error {
	// An effect flagged as area lands on everyone the area catches
	if effect.TargetType == models.FeatureTargetAOE && effect.Category != models.FeatureEffectAreaDamage {
		single := effect
		single.TargetType = models.FeatureTargetSingle
		for _, t := range fr.areaTargets(actor, target, state, result) {
			if err := fr.ResolveEffect(single, actor, t, state, result); err != nil {
				return err
			}
		}
		return nil
	}

	switch effect.Category {
	case models.FeatureEffectDamage, models.FeatureEffectDirectDamage, models.FeatureEffectAreaDamage:
		// Damage effects apply the reaction roll themselves, since success may only halve
	default:
		if fr.avoidsEffect(effect, actor, target, result) {
			return nil
		}
	}

	switch effect.Category {
	case models.FeatureEffectDamage:
		return fr.resolveDamage(effect, actor, target, state, result, false)
	case models.FeatureEffectDirectDamage:
		return fr.resolveDamage(effect, actor, target, state, result, true)
	case models.FeatureEffectAreaDamage:
		return fr.resolveAreaDamage(effect, actor, target, state, result)

	case models.FeatureEffectApplyVulnerable:
		return fr.applyCondition(actor, target, models.ConditionVulnerable, effect.Duration)
	case models.FeatureEffectApplyPoison:
		return fr.applyDamagingCondition(actor, target, models.ConditionPoisoned, effect)
	case models.FeatureEffectApplyIgnite:
		return fr.applyDamagingCondition(actor, target, models.ConditionBurning, effect)
	case models.FeatureEffectApplyStunned:
		return fr.applyCondition(actor, target, models.ConditionStunned, effect.Duration)
	case models.FeatureEffectApplyRestrained:
		return fr.applyCondition(actor, target, models.ConditionRestrained, effect.Duration)
	case models.FeatureEffectApplyAdvantage:
		return fr.applyCondition(actor, actor, models.ConditionAdvantage, effect.Duration)
	case models.FeatureEffectApplyDisadvantage:
		return fr.applyCondition(actor, target, models.ConditionDisadvantage, effect.Duration)
	case models.FeatureEffectDefensePenalty:
		if target != nil {
			// Does not stack: the largest penalty stands
			target.DefensePenalty = max(target.DefensePenalty, actor.RiderAmount(effect, 0, nil))
		}
		return nil
	case models.FeatureEffectMarkHP:
		return fr.resolveMarkHP(effect, target, result)
	case models.FeatureEffectApplyTaunt:
		return fr.applyCondition(actor, target, models.ConditionTaunted, effect.Duration)
	case models.FeatureEffectApplyHidden:
		return fr.applyCondition(actor, actor, models.ConditionHidden, effect.Duration)

	case models.FeatureEffectStressInflict:
		return fr.resolveStressInflict(effect, target)
	case models.FeatureEffectStressClear:
		return fr.resolveStressClear(effect, actor, state)
	case models.FeatureEffectHPHeal:
		return fr.resolveHPHeal(effect, actor, state)
	case models.FeatureEffectArmorDamage:
		return fr.resolveArmorDamage(effect, target, state)
	case models.FeatureEffectHopeDrain:
		return fr.resolveHopeDrain(effect, target)
	case models.FeatureEffectFearGenerate:
		state.AddFear(effect.Magnitude)
		return nil
	case models.FeatureEffectGrantArmor:
		return fr.resolveGrantArmor(effect, actor, state)
	case models.FeatureEffectArmorClear:
		return fr.resolveArmorClear(effect, actor, state)
	case models.FeatureEffectAllySpotlight:
		return fr.resolveAllySpotlight(effect, actor, state)
	case models.FeatureEffectSummonAllies:
		return fr.resolveSummon(effect, actor, state)

	// Passives read elsewhere (damage pipeline, stat block load, shielding) and tokens handled by the activation
	case models.FeatureEffectDamageResistance, models.FeatureEffectDamageVulnerability,
		models.FeatureEffectDamageReduction, models.FeatureEffectSpotlightModifier,
		models.FeatureEffectTokenPlace, models.FeatureEffectTokenSpend, models.FeatureEffectRedirectDamage,
		models.FeatureEffectDamageBonus, models.FeatureEffectSeverityReduction,
		models.FeatureEffectConditionImmunity, models.FeatureEffectTokenOnDamage,
		models.FeatureEffectDamageMultiplier, models.FeatureEffectAttackBonus, models.FeatureEffectEvasionBonus:
		return nil

	default:
		return fmt.Errorf("unknown effect category: %s", effect.Category)
	}
}

// rollDCCheck makes the actor's roll against a target's defense or a fixed Difficulty.
// Players roll Duality Dice and generate Hope or Fear; adversaries roll a d20.
// rollTrait names the trait a PC rolls with: "" for the weapon trait, "spellcast", or a trait.
func (fr *FeatureResolver) rollDCCheck(actor, target *models.Combatant, dcType string, fixedDC int, rollTrait string, state *models.CombatState) DCCheckResult {
	result := DCCheckResult{Success: true}

	switch dcType {
	case "target":
		if target == nil {
			return result
		}
		result.DC = target.GetDefense()
		result.VsDefense = true
	case "fixed":
		result.DC = fixedDC
	default:
		return result
	}

	hasAdv := target != nil && target.HasCondition(models.ConditionVulnerable)
	hasDisadv := target != nil && target.HasCondition(models.ConditionHidden)
	hasAdv = hasAdv || actor.HasCondition(models.ConditionAdvantage)
	hasDisadv = hasDisadv || actor.HasCondition(models.ConditionDisadvantage)

	result.RollParts = []models.LogPart{{Label: actor.RollLabel(rollTrait), Value: actor.RollModifier(rollTrait)}}
	if actor.Side != models.SidePlayer {
		roll := rollAdversaryD20(fr.dice, actor.GetAttackBonus(), hasAdv, hasDisadv)
		// The GM rolls a second d20 and keeps the better or worse one, so there is no number to add
		if hasAdv && !hasDisadv {
			result.RollParts = append(result.RollParts, models.LogPart{Label: "advantage"})
		} else if hasDisadv && !hasAdv {
			result.RollParts = append(result.RollParts, models.LogPart{Label: "disadvantage"})
		}
		result.D20Roll = roll.D20Roll
		result.TotalRoll = roll.Total
		result.IsCritical = roll.IsCritical
		result.Success = roll.IsCritical || roll.Total >= result.DC
		if result.Success && !roll.IsCritical && result.VsDefense && fr.evades(target, actor, roll.Total, state) {
			result.Success = false
		}
		return result
	}

	result.RollParts = append(result.RollParts, actor.AttackRollParts(fr.dice)...)
	if hasAdv && !hasDisadv {
		result.RollParts = append(result.RollParts, models.LogPart{Label: "advantage d6", Value: fr.dice.RollDice(1, 6, 0)})
	} else if hasDisadv && !hasAdv {
		result.RollParts = append(result.RollParts, models.LogPart{Label: "disadvantage d6", Value: -fr.dice.RollDice(1, 6, 0)})
	}

	roll := fr.dice.RollDuality(models.SumParts(result.RollParts), 0)
	result.HopeDie = roll.HopeDie
	result.FearDie = roll.FearDie
	result.TotalRoll = roll.WithSpirit
	result.IsCritical = roll.IsCritical

	actor.RolledFear = roll.Result == models.ResultFear
	actor.LastRollTotal = roll.WithSpirit
	if roll.IsCritical {
		actor.LastRollTotal = criticalRollTotal
	}
	if roll.Result == models.ResultFear {
		state.AddFear(1)
		result.FearGenerated = 1
	} else {
		actor.GainHope(1)
		result.HopeGenerated = 1
	}
	if roll.IsCritical {
		actor.ClearStress(1)
	}

	result.Success = roll.IsCritical || roll.WithSpirit >= result.DC
	// A Fear result or a failure passes the spotlight to the GM
	result.SpotlightShift = result.FearGenerated > 0 || !result.Success
	return result
}

// spendTokens pays every token_spend effect from held, refusing if any would overdraw
func spendTokens(effects []models.FeatureEffect, held *int) error {
	for _, effect := range effects {
		if effect.Category != models.FeatureEffectTokenSpend {
			continue
		}
		if *held < effect.Magnitude {
			return fmt.Errorf("not enough tokens: have %d, need %d", *held, effect.Magnitude)
		}
		*held -= effect.Magnitude
	}
	return nil
}

// placeTokens adds every token_place effect to held, capped at limit when limit is set
func placeTokens(effects []models.FeatureEffect, actor *models.Combatant, held *int, limit int) {
	for _, effect := range effects {
		if effect.Category == models.FeatureEffectTokenPlace {
			*held += effect.Magnitude * actor.ScaleValue(effect.ScaleBy)
		}
	}
	if limit > 0 && *held > limit {
		*held = limit
	}
}

// CanActivateAdversaryFeature checks if an adversary can use a feature
func (fr *FeatureResolver) CanActivateAdversaryFeature(feature *models.AdversaryFeature, actor *models.Combatant, state *models.CombatState) bool {
	if feature.FearCost > 0 && state.FearPool < feature.FearCost {
		return false
	}

	if feature.StressCost > 0 && (actor.Stress+feature.StressCost) > actor.StressMax {
		return false
	}

	if feature.MaxUses > 0 && feature.UsesLeft <= 0 {
		return false
	}

	held := feature.TokensHeld
	if spendTokens(feature.Effects, &held) != nil || lacksRecipient(feature.Effects, actor, state) {
		return false
	}

	// No point paying to spotlight allies who are not there
	for _, effect := range feature.Effects {
		if effect.Category == models.FeatureEffectAllySpotlight && len(state.EligibleSpotlightAllies(actor, effect)) == 0 {
			return false
		}
	}
	return true
}

// CanActivateDomainAbility checks if a player can use an ability
func (fr *FeatureResolver) CanActivateDomainAbility(ability models.DomainAbility, actor *models.Combatant, state *models.CombatState) bool {
	if ability.HopeCost > 0 && actor.Hope < ability.HopeCost {
		return false
	}
	if ability.StressCost > 0 && (actor.Stress+ability.StressCost) > actor.StressMax {
		return false
	}
	if ability.MaxUses > 0 && ability.UsesLeft <= 0 {
		return false
	}
	for _, effect := range ability.Effects {
		// Taking Fear from the GM needs Fear to take
		if effect.Category == models.FeatureEffectFearGenerate && effect.Magnitude < 0 && state != nil && state.FearPool == 0 {
			return false
		}
		// "All other targets" needs someone besides the target
		if effect.OthersOnly && state != nil && len(state.GetEnemiesOf(actor)) < 2 {
			return false
		}
	}
	held := ability.TokensHeld
	return spendTokens(ability.Effects, &held) == nil && !lacksRecipient(ability.Effects, actor, state)
}

// ActivateAdversaryFeature triggers an adversary feature
// Returns error and DC check result (if DC check was performed)
func (fr *FeatureResolver) ActivateAdversaryFeature(
	feature *models.AdversaryFeature,
	actor *models.Combatant,
	target *models.Combatant,
	state *models.CombatState,
) (error, DCCheckResult) {
	dcResult := DCCheckResult{Success: true} // Default for features without DC checks

	if !fr.CanActivateAdversaryFeature(feature, actor, state) {
		return fmt.Errorf("cannot activate feature %s", feature.Name), dcResult
	}

	if feature.MaxUses > 0 {
		feature.UsesLeft--
	}

	dcResult = fr.rollDCCheck(actor, target, feature.DCType, feature.FixedDC, "", state)

	if dcResult.Success || !feature.CostOnSuccess {
		state.SpendFear(feature.FearCost)
		actor.AddStress(feature.StressCost)
	}

	if err := spendTokens(feature.Effects, &feature.TokensHeld); err != nil {
		return err, dcResult
	}
	if err := fr.resolveEffects("feature "+feature.Name, feature.Effects, actor, target, state, &dcResult); err != nil {
		return err, dcResult
	}
	placeTokens(feature.Effects, actor, &feature.TokensHeld, feature.TokensMax)
	actor.DropBlockedConditions()
	return nil, dcResult
}

// ActivateDomainAbility triggers a player ability
// Returns error and DC check result (if DC check was performed)
func (fr *FeatureResolver) ActivateDomainAbility(
	ability *models.DomainAbility,
	actor *models.Combatant,
	target *models.Combatant,
	state *models.CombatState,
) (error, DCCheckResult) {
	dcResult := DCCheckResult{Success: true} // Default for abilities without DC checks

	if !fr.CanActivateDomainAbility(*ability, actor, state) {
		return fmt.Errorf("cannot activate ability %s", ability.Name), dcResult
	}

	// Costs are paid to attempt, whatever the roll says
	actor.SpendHope(ability.HopeCost)
	actor.AddStress(ability.StressCost)
	if ability.MaxUses > 0 {
		ability.UsesLeft--
	}

	dcResult = fr.rollDCCheck(actor, target, ability.DCType, ability.FixedDC, ability.RollTrait, state)

	if err := spendTokens(ability.Effects, &ability.TokensHeld); err != nil {
		return err, dcResult
	}
	if err := fr.resolveEffects("ability "+ability.Name, ability.Effects, actor, target, state, &dcResult); err != nil {
		return err, dcResult
	}
	placeTokens(ability.Effects, actor, &ability.TokensHeld, ability.TokensMax)
	actor.DropBlockedConditions()
	return nil, dcResult
}

// afterDamageDealt steps the actor's token dice once a damage roll has marked a Hit Point,
// and logs any feature whose die ran out
func (fr *FeatureResolver) afterDamageDealt(actor *models.Combatant, state *models.CombatState) {
	if actor == nil {
		return
	}
	for _, name := range actor.StepTokensOnDamage() {
		if state == nil {
			continue
		}
		state.MarkConsequence()
		state.RecordAction(models.ActionParams{
			Actor:       actor,
			ActionType:  models.ActionUseAbility,
			FeatureName: name,
			IsFeature:   true,
			Result:      "Ends: its die passed the maximum",
		})
	}
}

// ErrFeatureMissed marks a feature whose attack or DC roll failed: a normal outcome, not a fault.
var ErrFeatureMissed = errors.New("failed DC check")

// recordTriggered logs a passive that fired, so reactions show up in the combat log
func recordTriggered(state *models.CombatState, actor, target *models.Combatant, name string, err error, dcResult DCCheckResult) {
	recordReactions(state, name, dcResult)
	result := "Triggered"
	switch {
	case errors.Is(err, ErrFeatureMissed):
		result = "Reaction missed"
	case err != nil:
		result = fmt.Sprintf("Triggered: %v", err)
	}
	params := models.ActionParams{
		Actor:         actor,
		ActionType:    models.ActionUseAbility,
		FeatureName:   name,
		IsFeature:     true,
		Result:        result,
		HopeDie:       dcResult.HopeDie,
		FearDie:       dcResult.FearDie,
		D20Roll:       dcResult.D20Roll,
		TotalRoll:     dcResult.TotalRoll,
		TargetDefense: dcResult.DC,
		RollParts:     dcResult.RollParts,
		VsDefense:     dcResult.VsDefense,
	}
	// A passive that dealt damage shows whom it hit and how the damage came about
	if dcResult.DamageRoll > 0 || dcResult.DamageDealt > 0 {
		params.Target = target
		if dcResult.RedirectedTo != nil {
			params.Target = dcResult.RedirectedTo
		}
		params.DamageRoll, params.DamageParts, params.DamageFormula = dcResult.DamageRoll, dcResult.DamageParts, dcResult.DamageFormula
		params.ProcessedDamage, params.HPDamage, params.Threshold = dcResult.ProcessedDamage, dcResult.DamageDealt, dcResult.Threshold
		params.TargetsHit, params.DamageResult = dcResult.TargetsHit, dcResult.DamageResult
	}
	state.RecordAction(params)
}

// ProcessTriggeredFeatures fires the actor's passives whose trigger matches
func (fr *FeatureResolver) ProcessTriggeredFeatures(
	actor *models.Combatant,
	target *models.Combatant,
	state *models.CombatState,
	triggerType models.FeatureTriggerType,
	isCritical bool,
) {
	for i := range actor.AdversaryFeatures {
		feature := &actor.AdversaryFeatures[i]
		if !feature.Passive || !fr.shouldTrigger(feature.Trigger, triggerType, isCritical, actor, target) {
			continue
		}
		if !fr.CanActivateAdversaryFeature(feature, actor, state) {
			continue
		}
		state.MarkConsequence()
		err, dcResult := fr.ActivateAdversaryFeature(feature, actor, target, state)
		recordTriggered(state, actor, target, feature.Name, err, dcResult)
	}

	for i := range actor.DomainAbilities {
		ability := &actor.DomainAbilities[i]
		if !ability.Passive || !fr.shouldTrigger(ability.Trigger, triggerType, isCritical, actor, target) {
			continue
		}
		if !actor.KeepsHalfStress(ability.StressCost) || alreadyInPlace(ability.Effects, actor, target) || !fr.CanActivateDomainAbility(*ability, actor, state) {
			continue
		}
		state.MarkConsequence()
		err, dcResult := fr.ActivateDomainAbility(ability, actor, target, state)
		recordTriggered(state, actor, target, ability.Name, err, dcResult)
	}
}

// alreadyInPlace reports a passive that would only repeat what its target already suffers:
// every effect is a condition the target has or a penalty at least as large as the new one
func alreadyInPlace(effects []models.FeatureEffect, actor, target *models.Combatant) bool {
	if target == nil || len(effects) == 0 {
		return false
	}
	for _, effect := range effects {
		switch effect.Category {
		case models.FeatureEffectApplyVulnerable:
			if !target.HasCondition(models.ConditionVulnerable) && !target.ImmuneTo(models.ConditionVulnerable) {
				return false
			}
		case models.FeatureEffectApplyDisadvantage:
			if !target.HasCondition(models.ConditionDisadvantage) {
				return false
			}
		case models.FeatureEffectDefensePenalty:
			if target.DefensePenalty < actor.RiderAmount(effect, 0, nil) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// damageRiders fires the attacker's on_damage_roll passives for a weapon attack that hit. Each
// pays its cost; they return what each adds to the damage roll, and what multiplies it and why.
func (fr *FeatureResolver) damageRiders(actor, target *models.Combatant, state *models.CombatState) (parts []models.LogPart, factor int, multipliers []string) {
	factor = 1
	if actor == nil || state == nil {
		return parts, factor, multipliers
	}
	for i := range actor.DomainAbilities {
		ability := &actor.DomainAbilities[i]
		if !ability.Passive || !fr.shouldTrigger(ability.Trigger, models.TriggerOnDamageRoll, false, actor, target) {
			continue
		}
		if !actor.KeepsHalfStress(ability.StressCost) || !fr.CanActivateDomainAbility(*ability, actor, state) {
			continue
		}
		tokens := ability.TokensHeld
		err, dcResult := fr.ActivateDomainAbility(ability, actor, target, state)
		recordTriggered(state, actor, target, ability.Name, err, dcResult)
		if err != nil {
			continue
		}
		for _, effect := range ability.Effects {
			switch effect.Category {
			case models.FeatureEffectDamageBonus:
				parts = append(parts, models.LogPart{Label: ability.Name, Value: actor.RiderAmount(effect, tokens, fr.dice)})
			case models.FeatureEffectDamageMultiplier:
				factor *= max(effect.Magnitude, 1)
				multipliers = append(multipliers, ability.Name)
			}
		}
	}
	return parts, factor, multipliers
}

// finishDamage applies the multipliers and halvings that come after everything is added up,
// recording each as a part so the log still sums to the total
func finishDamage(total int, parts []models.LogPart, factor int, multipliers []string, halvings ...string) (int, []models.LogPart) {
	if factor > 1 {
		parts = append(parts, models.LogPart{Label: fmt.Sprintf("%s x%d", strings.Join(multipliers, ", "), factor), Value: total * (factor - 1)})
		total *= factor
	}
	for _, why := range halvings {
		parts = append(parts, models.LogPart{Label: why, Value: total/2 - total})
		total /= 2
	}
	return total, parts
}

// evades lets a PC's on_attacked passives turn an attack roll that would hit into a miss.
// A reaction is only paid for when its best result could still beat the roll.
func (fr *FeatureResolver) evades(defender, attacker *models.Combatant, attackTotal int, state *models.CombatState) bool {
	if defender == nil || state == nil {
		return false
	}
	short := attackTotal - defender.GetDefense() + 1
	for i := range defender.DomainAbilities {
		ability := &defender.DomainAbilities[i]
		if !ability.Passive || !fr.shouldTrigger(ability.Trigger, models.TriggerOnAttacked, false, defender, attacker) {
			continue
		}
		best := 0
		for _, effect := range ability.Effects {
			if effect.Category == models.FeatureEffectEvasionBonus {
				best += defender.RiderAmount(effect, ability.TokensHeld, maxRoller{})
			}
		}
		if best < short || !defender.KeepsStressReserve(ability.StressCost) || !fr.CanActivateDomainAbility(*ability, defender, state) {
			continue
		}
		tokens := ability.TokensHeld
		err, dcResult := fr.ActivateDomainAbility(ability, defender, attacker, state)
		recordTriggered(state, defender, attacker, ability.Name, err, dcResult)
		if err != nil {
			continue
		}
		for _, effect := range ability.Effects {
			if effect.Category == models.FeatureEffectEvasionBonus {
				short -= defender.RiderAmount(effect, tokens, fr.dice)
			}
		}
		if short <= 0 {
			return true
		}
	}
	return false
}

// criticalRollTotal stands for a critical success when a roll is reused: it beats any Difficulty
const criticalRollTotal = 1000

// maxRoller rolls every die at its highest face, to see the most a bonus could be
type maxRoller struct{}

func (maxRoller) RollDice(diceCount, diceSize, flatBonus int) int {
	return diceCount*diceSize + flatBonus
}

// logReactions records the on_damage_incoming passives a target paid for to soften a hit
func logReactions(state *models.CombatState, target *models.Combatant, damageRes models.DamageResult) {
	if state == nil {
		return
	}
	for _, name := range damageRes.Reactions {
		state.MarkConsequence()
		recordTriggered(state, target, nil, name, nil, DCCheckResult{})
	}
}

// shouldTrigger checks if a trigger condition is met
func (fr *FeatureResolver) shouldTrigger(
	trigger models.FeatureTriggerCondition,
	triggerType models.FeatureTriggerType,
	isCritical bool,
	actor *models.Combatant,
	target *models.Combatant,
) bool {
	if trigger.Type != triggerType {
		return false
	}
	if triggerType == models.TriggerOnCritical && !isCritical {
		return false
	}
	for _, part := range strings.Split(trigger.Filter, "+") {
		if part != "" && !fr.evaluateFilter(part, actor, target) {
			return false
		}
	}
	return true
}

// evaluateFilter checks additional trigger constraints
func (fr *FeatureResolver) evaluateFilter(filter string, actor *models.Combatant, target *models.Combatant) bool {
	switch filter {
	case "if_target_below_50_hp":
		if target == nil {
			return false
		}
		return (target.HP * 100 / target.HPMax) < 50
	case "if_hidden":
		return actor.HasCondition(models.ConditionHidden)
	case "if_ally_adjacent":
		return false
	case "if_target_vulnerable":
		return target != nil && target.HasCondition(models.ConditionVulnerable)
	case "if_target_fully_stressed":
		return target != nil && target.MeetsFilter("if_fully_stressed")
	case "if_dealt_severe":
		return target != nil && target.LastSeverity == models.ThresholdSevere
	case "if_took_severe":
		return actor.LastSeverity == models.ThresholdSevere
	default:
		return actor.MeetsFilter(filter)
	}
}
