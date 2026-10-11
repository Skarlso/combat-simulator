package models

// FeatureEffectCategory defines the type of effect a feature can have
type FeatureEffectCategory string

const (
	// Damage categories
	FeatureEffectDamage       FeatureEffectCategory = "damage"
	FeatureEffectDirectDamage FeatureEffectCategory = "direct_damage"
	FeatureEffectAreaDamage   FeatureEffectCategory = "area_damage"

	// Status application
	FeatureEffectApplyVulnerable FeatureEffectCategory = "apply_vulnerable"
	FeatureEffectApplyPoison     FeatureEffectCategory = "apply_poison"
	FeatureEffectApplyIgnite     FeatureEffectCategory = "apply_ignite"
	FeatureEffectApplyTaunt      FeatureEffectCategory = "apply_taunt"
	FeatureEffectApplyHidden     FeatureEffectCategory = "apply_hidden"
	FeatureEffectApplyStunned    FeatureEffectCategory = "apply_stunned"
	FeatureEffectApplyRestrained FeatureEffectCategory = "apply_restrained"

	// Resource manipulation
	FeatureEffectStressInflict FeatureEffectCategory = "stress_inflict"
	FeatureEffectStressClear   FeatureEffectCategory = "stress_clear"
	FeatureEffectHPHeal        FeatureEffectCategory = "hp_heal"
	FeatureEffectArmorDamage   FeatureEffectCategory = "armor_damage"
	FeatureEffectGrantArmor    FeatureEffectCategory = "grant_armor"
	FeatureEffectHopeDrain     FeatureEffectCategory = "hope_drain"
	FeatureEffectFearGenerate  FeatureEffectCategory = "fear_generate"

	// Defensive
	FeatureEffectDamageReduction FeatureEffectCategory = "damage_reduction"

	// Token mechanics
	FeatureEffectTokenPlace FeatureEffectCategory = "token_place"
	FeatureEffectTokenSpend FeatureEffectCategory = "token_spend"

	// Resistance
	FeatureEffectDamageResistance    FeatureEffectCategory = "damage_resistance"
	FeatureEffectDamageVulnerability FeatureEffectCategory = "damage_vulnerability"

	// Action economy
	FeatureEffectSpotlightModifier FeatureEffectCategory = "spotlight_modifier"
	FeatureEffectAllySpotlight     FeatureEffectCategory = "ally_spotlight"

	// Summoning
	FeatureEffectSummonAllies FeatureEffectCategory = "summon_allies"
)

// FeatureTargetType defines who can be targeted by an effect
type FeatureTargetType string

const (
	FeatureTargetSelf   FeatureTargetType = "self"
	FeatureTargetSingle FeatureTargetType = "single"
	FeatureTargetAOE    FeatureTargetType = "aoe"
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
	TriggerEndTurn       FeatureTriggerType = "end_turn"
)

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
	Duration       string // "instant", "1_turn", "until_cleared", "permanent", "scene"
	TargetType     FeatureTargetType
	DamageType     string // "physical", "magic", empty for non-damage effects
	ArmorCost      int    // Armor Slots a target marks to reduce this damage (0 = the usual 1)
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
}

// AdversaryFeature represents passive or AI-triggered abilities for enemies
type AdversaryFeature struct {
	Name           string
	Description    string
	Passive        bool // true = always on, false = AI decides when to trigger
	Trigger        FeatureTriggerCondition
	FearCost       int    // Fear required to activate (0 = free)
	StressCost     int    // Stress required to activate (0 = free)
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

	// Token mechanics (for abilities like Unleash Chaos)
	UsesTokens bool   // Whether this ability uses tokens
	TokenName  string // Name of tokens (e.g., "Chaos", "Ward")
	TokensHeld int    // Current tokens held (runtime state)
	TokensMax  int    // Maximum tokens (0 = unlimited)

	Effects []FeatureEffect
}
