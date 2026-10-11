package engine

import "github.com/skarlso/daggerheart-combat-simulator/internal/models"

type reactionResult struct {
	success  bool
	critical bool
}

// ReactionRoll is one target's reaction roll, kept for the combat log
type ReactionRoll struct {
	Target     *models.Combatant
	Trait      string
	HopeDie    int
	FearDie    int
	D20Roll    int
	Total      int
	Difficulty int
	Outcome    string
}

// recordReactions logs the reaction rolls made against a feature
func recordReactions(state *models.CombatState, featureName string, result DCCheckResult) {
	for _, r := range result.Reactions {
		state.RecordAction(models.ActionParams{
			Actor:         r.Target,
			ActionType:    models.ActionReactionRoll,
			FeatureName:   featureName,
			IsFeature:     true,
			Result:        r.Outcome,
			HopeDie:       r.HopeDie,
			FearDie:       r.FearDie,
			D20Roll:       r.D20Roll,
			TotalRoll:     r.Total,
			TargetDefense: r.Difficulty,
		})
	}
}

func reactionOutcome(effect models.FeatureEffect, r reactionResult) string {
	isDamage := effect.Category == models.FeatureEffectDamage ||
		effect.Category == models.FeatureEffectDirectDamage ||
		effect.Category == models.FeatureEffectAreaDamage

	switch {
	case r.critical:
		return "Critical: ignores the effect"
	case r.success && isDamage && effect.HalfOnSuccess:
		return "Success: half damage"
	case r.success:
		return "Success: avoided"
	default:
		return "Failed"
	}
}

// reactionFor makes the target's reaction roll against an effect. A target rolls
// once per activation, however many of the feature's effects ask for it.
// SRD: reaction rolls generate no Hope or Fear and never move the spotlight.
func (fr *FeatureResolver) reactionFor(effect models.FeatureEffect, actor, target *models.Combatant, result *DCCheckResult) reactionResult {
	if result != nil {
		if r, ok := result.reactions[target]; ok {
			return r
		}
	}

	difficulty := effect.ReactionDC
	if difficulty == 0 && actor != nil {
		difficulty = actor.GetDefense()
	}

	var r reactionResult
	logged := ReactionRoll{Target: target, Trait: effect.ReactionTrait, Difficulty: difficulty}
	if target.IsAdversary {
		// An adversary's critical reaction only succeeds, with no extra benefit
		roll := fr.dice.RollD20Attack(0)
		r.success = roll.IsCritical || roll.Total >= difficulty
		logged.D20Roll, logged.Total = roll.D20Roll, roll.Total
	} else {
		roll := fr.dice.RollDuality(target.Traits.Modifier(effect.ReactionTrait), 0)
		r.critical = roll.IsCritical
		r.success = roll.IsCritical || roll.WithSpirit >= difficulty
		logged.HopeDie, logged.FearDie, logged.Total = roll.HopeDie, roll.FearDie, roll.WithSpirit
	}

	if result != nil {
		if result.reactions == nil {
			result.reactions = make(map[*models.Combatant]reactionResult)
		}
		result.reactions[target] = r
		logged.Outcome = reactionOutcome(effect, r)
		result.Reactions = append(result.Reactions, logged)
	}
	return r
}

// avoidsEffect reports whether the target's reaction roll negates a non-damage effect
func (fr *FeatureResolver) avoidsEffect(effect models.FeatureEffect, actor, target *models.Combatant, result *DCCheckResult) bool {
	if effect.ReactionTrait == "" || target == nil {
		return false
	}
	return fr.reactionFor(effect, actor, target, result).success
}

// damageAfterReaction returns the damage a target takes once their reaction roll is made
func (fr *FeatureResolver) damageAfterReaction(effect models.FeatureEffect, actor, target *models.Combatant, damage int, result *DCCheckResult) int {
	if effect.ReactionTrait == "" {
		return damage
	}

	r := fr.reactionFor(effect, actor, target, result)
	switch {
	case r.critical:
		return 0
	case r.success && effect.HalfOnSuccess:
		// SRD rounds up
		return (damage + 1) / 2
	case r.success:
		return 0
	default:
		return damage
	}
}
