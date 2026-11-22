package models

// FeatureEffectCategory defines the type of effect a feature can have
type FeatureEffectCategory string

const (
	// Damage categories
	FeatureEffectDamage       FeatureEffectCategory = "damage"
	FeatureEffectDirectDamage FeatureEffectCategory = "direct_damage"
	FeatureEffectAreaDamage   FeatureEffectCategory = "area_damage"
	FeatureEffectMarkHP       FeatureEffectCategory = "mark_hp" // the target marks Magnitude Hit Points outright

	// Status application
	FeatureEffectApplyVulnerable FeatureEffectCategory = "apply_vulnerable"
	FeatureEffectApplyPoison     FeatureEffectCategory = "apply_poison"
	FeatureEffectApplyIgnite     FeatureEffectCategory = "apply_ignite"
	FeatureEffectApplyTaunt      FeatureEffectCategory = "apply_taunt"
	FeatureEffectApplyHidden     FeatureEffectCategory = "apply_hidden"
	FeatureEffectApplyStunned    FeatureEffectCategory = "apply_stunned"
	FeatureEffectApplyRestrained FeatureEffectCategory = "apply_restrained"
	// Roll modifiers: the actor attacks with advantage this turn, the target attacks with disadvantage
	FeatureEffectApplyAdvantage    FeatureEffectCategory = "apply_advantage"
	FeatureEffectApplyDisadvantage FeatureEffectCategory = "apply_disadvantage"
	FeatureEffectDefensePenalty    FeatureEffectCategory = "defense_penalty" // lowers the target's Difficulty or Evasion

	// Resource manipulation
	FeatureEffectStressInflict FeatureEffectCategory = "stress_inflict"
	FeatureEffectStressClear   FeatureEffectCategory = "stress_clear"
	FeatureEffectHPHeal        FeatureEffectCategory = "hp_heal"
	FeatureEffectArmorDamage   FeatureEffectCategory = "armor_damage"
	FeatureEffectGrantArmor    FeatureEffectCategory = "grant_armor"
	FeatureEffectArmorClear    FeatureEffectCategory = "armor_clear" // unmark Armor Slots, up to the armor's score
	FeatureEffectHopeDrain     FeatureEffectCategory = "hope_drain"
	FeatureEffectFearGenerate  FeatureEffectCategory = "fear_generate"

	// Defensive
	FeatureEffectDamageReduction FeatureEffectCategory = "damage_reduction"

	// Standing effects: in force while their passive is, never resolved on activation.
	// On a passive with a reactive trigger they apply to the one roll or hit it reacts to instead.
	FeatureEffectDamageBonus       FeatureEffectCategory = "damage_bonus"       // added to the owner's damage rolls
	FeatureEffectDamageMultiplier  FeatureEffectCategory = "damage_multiplier"  // on_damage_roll only: multiplies the roll
	FeatureEffectSeverityReduction FeatureEffectCategory = "severity_reduction" // incoming damage drops Magnitude thresholds
	FeatureEffectConditionImmunity FeatureEffectCategory = "condition_immunity" // the owner can't gain Conditions
	FeatureEffectAttackBonus       FeatureEffectCategory = "attack_bonus"       // added to the owner's attack and action rolls
	FeatureEffectEvasionBonus      FeatureEffectCategory = "evasion_bonus"      // added to the owner's Evasion

	// Token mechanics
	FeatureEffectTokenPlace FeatureEffectCategory = "token_place"
	FeatureEffectTokenSpend FeatureEffectCategory = "token_spend"
	// A die that counts up: +Magnitude after the owner's damage roll marks a Hit Point, removed once it would pass TokensMax
	FeatureEffectTokenOnDamage FeatureEffectCategory = "token_on_damage"

	// Resistance
	FeatureEffectDamageResistance    FeatureEffectCategory = "damage_resistance"
	FeatureEffectDamageVulnerability FeatureEffectCategory = "damage_vulnerability"

	// Action economy
	FeatureEffectSpotlightModifier FeatureEffectCategory = "spotlight_modifier"
	FeatureEffectAllySpotlight     FeatureEffectCategory = "ally_spotlight"

	// Summoning
	FeatureEffectSummonAllies FeatureEffectCategory = "summon_allies"

	// Protection: a passive with this effect steps in front of an ally about to take damage
	FeatureEffectRedirectDamage FeatureEffectCategory = "redirect_damage"
)

// FeatureTargetType defines who can be targeted by an effect
type FeatureTargetType string

const (
	FeatureTargetSelf   FeatureTargetType = "self"
	FeatureTargetSingle FeatureTargetType = "single"
	FeatureTargetAOE    FeatureTargetType = "aoe"
	FeatureTargetAlly   FeatureTargetType = "ally" // the ally who needs it most, never the actor
	FeatureTargetAllies FeatureTargetType = "allies"
)

type FeatureTriggerType string

const (
	TriggerOnAttack      FeatureTriggerType = "on_attack"
	TriggerOnHit         FeatureTriggerType = "on_hit"
	TriggerOnDamageTaken FeatureTriggerType = "on_damage_taken"
	TriggerOnHitTaken    FeatureTriggerType = "on_hit_taken" // a successful attack against this combatant, damage or not
	TriggerOnCritical    FeatureTriggerType = "on_critical"
	TriggerStartTurn     FeatureTriggerType = "start_turn"
	// Declared before the fight: fires when it starts, and at the start of the owner's turns while uses are left
	TriggerStartCombat   FeatureTriggerType = "start_combat"
	TriggerEndTurn       FeatureTriggerType = "end_turn"
	TriggerOnAllyDamaged FeatureTriggerType = "on_ally_damaged" // an ally is about to take damage from an attack
	TriggerOnMiss        FeatureTriggerType = "on_miss"         // the owner's basic attack failed
	TriggerOnDamageDealt FeatureTriggerType = "on_damage_dealt" // the owner's attack marked a Hit Point on its target
	TriggerOnEvaded      FeatureTriggerType = "on_evaded"       // an attack against the owner failed

	// Reactive triggers: the passive pays its cost to change one roll or hit as it happens
	TriggerOnDamageRoll     FeatureTriggerType = "on_damage_roll"     // the owner rolls damage for a weapon attack that hit
	TriggerOnDamageIncoming FeatureTriggerType = "on_damage_incoming" // the owner is about to mark Hit Points
	TriggerOnAttacked       FeatureTriggerType = "on_attacked"        // an attack roll is about to succeed against the owner
)

// IsReactive reports a trigger whose effects apply to a single roll or hit, never as standing effects
func (t FeatureTriggerType) IsReactive() bool {
	return t == TriggerOnDamageRoll || t == TriggerOnDamageIncoming || t == TriggerOnAttacked
}

type FeatureTriggerCondition struct {
	Type   FeatureTriggerType
	Filter string
}

// FeatureEffect is the shared resolution system for both adversary features and player abilities
type FeatureEffect struct {
	Category    FeatureEffectCategory
	Magnitude   int    // Flat value (damage bonus, stress amount, etc.)
	DiceFormula string // "2d6+3" for variable effects
	// Roll the actor's standard attack damage instead of DiceFormula
	StandardAttack bool
	// Roll Proficiency dice of DiceFormula's size, as spells "using your Proficiency" do
	ProficiencyDice bool
	Duration        string // "instant", "1_turn", "until_cleared", "permanent", "scene"
	TargetType      FeatureTargetType
	DamageType      string // "physical", "magic", empty for non-damage effects
	ArmorCost       int    // Armor Slots a target marks to reduce this damage (0 = the usual 1)
	// Fear the GM gains per target that has no Armor Slot left for armor_damage to mark
	FallbackFear int

	// Reaction roll the target makes to avoid this effect ("" = none): the trait
	// they roll, the Difficulty (0 = the actor's own), and whether success halves
	// damage instead of negating it.
	ReactionTrait string
	ReactionDC    int
	HalfOnSuccess bool
	SummonType    string       // Adversary name to summon (for FeatureEffectSummonAllies) - registry lookup
	SummonPool    []*Combatant // Inline summon templates (preferred over registry)

	// Spotlight chaining. ally_spotlight picks Magnitude+DiceFormula allies (or all with
	// SpotlightAll), optionally only those named AllyName; summon_allies spotlights
	// SpotlightCount of the new arrivals (or all). HalfDamage halves what they deal.
	SpotlightAll   bool
	SpotlightCount int
	AllyName       string
	HalfDamage     bool

	// PerToken multiplies a standing effect's Magnitude and dice by the tokens its feature holds
	PerToken bool
	// ScaleBy multiplies the amount (for damage effects, the dice count) by one of the actor's numbers:
	// "level", "tier", "proficiency", "spellcast" (the Spellcast trait) or a trait name. Empty means no scaling.
	ScaleBy string
	// OthersOnly keeps an area effect off the attack's own target ("all other targets")
	OthersOnly bool
	// Conditions a condition_immunity effect keeps off its owner
	Conditions []ConditionType
}

// AdversaryFeature represents passive or AI-triggered abilities for enemies
type AdversaryFeature struct {
	Name           string
	Description    string
	Passive        bool // true = always on, false = AI decides when to trigger
	Trigger        FeatureTriggerCondition
	FearCost       int    // Fear required to activate (0 = free)
	StressCost     int    // Stress required to activate (0 = free)
	CostOnSuccess  bool   // "On a success, mark a Stress to...": costs are only paid when the roll succeeds
	SpotlightLimit int    // Max times per GM turn (0 = unlimited)
	UsageType      string // "unlimited", "once_per_scene", "once_per_session", "once_per_rest"
	MaxUses        int    // Limit count (0 = unlimited)
	UsesLeft       int    // Runtime state

	// Difficulty Check
	DCType  string // "none" (auto-succeed), "target" (roll vs target's Evasion), "fixed" (roll vs FixedDC)
	FixedDC int    // DC value if DCType is "fixed"

	UsesTokens bool
	TokenName  string
	TokensHeld int
	TokensMax  int

	Effects []FeatureEffect
}

// DomainAbility represents player-chosen abilities from domain cards
type DomainAbility struct {
	Name        string
	Description string
	Passive     bool // true = always on, false = AI decides when to trigger
	Trigger     FeatureTriggerCondition
	HopeCost    int    // Hope tokens required
	StressCost  int    // Stress marks required
	RefreshType string // "rest", "long_rest", "scene", "unlimited"
	MaxUses     int    // Uses before needing refresh (0 = unlimited if RefreshType allows)
	UsesLeft    int    // Current remaining uses (runtime state)

	// Difficulty Check
	DCType  string // "none" (auto-succeed), "target" (roll vs target's Difficulty), "fixed" (roll vs FixedDC)
	FixedDC int    // DC value if DCType is "fixed"
	// Trait the roll is made with: "" for the weapon trait, "spellcast", or a trait name
	RollTrait string

	// Token mechanics (for abilities like Unleash Chaos)
	UsesTokens bool   // Whether this ability uses tokens
	TokenName  string // Name of tokens (e.g., "Chaos", "Ward")
	TokensHeld int    // Current tokens held (runtime state)
	TokensMax  int    // Maximum tokens (0 = unlimited)

	Effects []FeatureEffect
}
