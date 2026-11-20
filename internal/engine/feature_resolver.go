package engine

import (
	"errors"
	"fmt"

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
	// Damage tracking (populated during effect resolution)
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
		return fr.resolveGrantArmor(effect, target)
	case models.FeatureEffectAllySpotlight:
		return fr.resolveAllySpotlight(effect, actor, state)
	case models.FeatureEffectSummonAllies:
		return fr.resolveSummon(effect, actor, state)

	// Passives read elsewhere (damage pipeline, stat block load, shielding) and tokens handled by the activation
	case models.FeatureEffectDamageResistance, models.FeatureEffectDamageVulnerability,
		models.FeatureEffectDamageReduction, models.FeatureEffectSpotlightModifier,
		models.FeatureEffectTokenPlace, models.FeatureEffectTokenSpend, models.FeatureEffectRedirectDamage:
		return nil

	default:
		return fmt.Errorf("unknown effect category: %s", effect.Category)
	}
}

// rollDCCheck makes the actor's roll against a target's defense or a fixed Difficulty.
// Players roll Duality Dice and generate Hope or Fear; adversaries roll a d20.
func (fr *FeatureResolver) rollDCCheck(actor, target *models.Combatant, dcType string, fixedDC int, state *models.CombatState) DCCheckResult {
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

	if actor.Side != models.SidePlayer {
		roll := rollAdversaryD20(fr.dice, actor.GetAttackBonus(), hasAdv, hasDisadv)
		result.D20Roll = roll.D20Roll
		result.TotalRoll = roll.Total
		result.IsCritical = roll.IsCritical
		result.Success = roll.IsCritical || roll.Total >= result.DC
		return result
	}

	hasAdv = hasAdv || actor.HasCondition(models.ConditionAdvantage)
	hasDisadv = hasDisadv || actor.HasCondition(models.ConditionDisadvantage)
	hopeModifier := actor.GetAttackBonus()
	if hasAdv && !hasDisadv {
		hopeModifier += fr.dice.RollDice(1, 6, 0)
	} else if hasDisadv && !hasAdv {
		hopeModifier -= fr.dice.RollDice(1, 6, 0)
	}

	roll := fr.dice.RollDuality(hopeModifier, 0)
	result.HopeDie = roll.HopeDie
	result.FearDie = roll.FearDie
	result.TotalRoll = roll.WithSpirit
	result.IsCritical = roll.IsCritical

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
func placeTokens(effects []models.FeatureEffect, held *int, limit int) {
	for _, effect := range effects {
		if effect.Category == models.FeatureEffectTokenPlace {
			*held += effect.Magnitude
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

	dcResult = fr.rollDCCheck(actor, target, feature.DCType, feature.FixedDC, state)

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
	placeTokens(feature.Effects, &feature.TokensHeld, feature.TokensMax)
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

	dcResult = fr.rollDCCheck(actor, target, ability.DCType, ability.FixedDC, state)

	if err := spendTokens(ability.Effects, &ability.TokensHeld); err != nil {
		return err, dcResult
	}
	if err := fr.resolveEffects("ability "+ability.Name, ability.Effects, actor, target, state, &dcResult); err != nil {
		return err, dcResult
	}
	placeTokens(ability.Effects, &ability.TokensHeld, ability.TokensMax)
	return nil, dcResult
}

// ErrFeatureMissed marks a feature whose attack or DC roll failed: a normal outcome, not a fault.
var ErrFeatureMissed = errors.New("failed DC check")

// recordTriggered logs a passive that fired, so reactions show up in the combat log
func recordTriggered(state *models.CombatState, actor *models.Combatant, name string, err error, dcResult DCCheckResult) {
	recordReactions(state, name, dcResult)
	result := "Triggered"
	switch {
	case errors.Is(err, ErrFeatureMissed):
		result = "Reaction missed"
	case err != nil:
		result = fmt.Sprintf("Triggered: %v", err)
	}
	state.RecordAction(models.ActionParams{
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
	})
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
		err, dcResult := fr.ActivateAdversaryFeature(feature, actor, target, state)
		recordTriggered(state, actor, feature.Name, err, dcResult)
	}

	for i := range actor.DomainAbilities {
		ability := &actor.DomainAbilities[i]
		if !ability.Passive || !fr.shouldTrigger(ability.Trigger, triggerType, isCritical, actor, target) {
			continue
		}
		if !fr.CanActivateDomainAbility(*ability, actor, state) {
			continue
		}
		err, dcResult := fr.ActivateDomainAbility(ability, actor, target, state)
		recordTriggered(state, actor, ability.Name, err, dcResult)
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
	if trigger.Filter != "" {
		return fr.evaluateFilter(trigger.Filter, actor, target)
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
	default:
		return true
	}
}
