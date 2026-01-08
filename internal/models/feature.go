package models

// FeatureEffectCategory defines the type of effect a feature can have
type FeatureEffectCategory string

const (
	// Damage categories
	FeatureEffectDamage          FeatureEffectCategory = "damage"
	FeatureEffectDamageAmplifier FeatureEffectCategory = "damage_amplifier"
	FeatureEffectDirectDamage    FeatureEffectCategory = "direct_damage"
	FeatureEffectAreaDamage      FeatureEffectCategory = "area_damage"

	// Status application
	FeatureEffectApplyVulnerable FeatureEffectCategory = "apply_vulnerable"
	FeatureEffectApplyPoison     FeatureEffectCategory = "apply_poison"
	FeatureEffectApplyIgnite     FeatureEffectCategory = "apply_ignite"
	FeatureEffectApplyTaunt      FeatureEffectCategory = "apply_taunt"
	FeatureEffectApplyHidden     FeatureEffectCategory = "apply_hidden"

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

// FeatureTriggerCondition defines when a feature activates
type FeatureTriggerCondition struct {
	Type   string // "on_attack", "on_damage_taken", "on_critical", "start_turn", "end_turn", "always"
	Filter string // Additional constraints: "if_target_below_50_hp", "if_hidden", "if_ally_adjacent"
}

// FeatureEffect is the shared resolution system for both adversary features and player abilities
type FeatureEffect struct {
	Category    FeatureEffectCategory
	Magnitude   int    // Flat value (damage bonus, stress amount, etc.)
	DiceFormula string // "2d6+3" for variable effects
	Duration    string // "instant", "1_turn", "until_cleared", "permanent", "scene"
	TargetType  FeatureTargetType
	DamageType  string // "physical", "magic", empty for non-damage effects
}

// AdversaryFeature represents passive or AI-triggered abilities for enemies
type AdversaryFeature struct {
	Name           string
	Description    string
	Passive        bool // true = always on, false = AI decides when to trigger
	Trigger        FeatureTriggerCondition
	FearCost       int // Fear required to activate (0 = free)
	SpotlightLimit int // Max times per GM turn (0 = unlimited)

	// Difficulty Check
	DCType  string // "none" (auto-succeed), "target" (roll vs target's Evasion), "fixed" (roll vs FixedDC)
	FixedDC int    // DC value if DCType is "fixed"

	Effects        []FeatureEffect
}

// DomainAbility represents player-chosen abilities from domain cards
type DomainAbility struct {
	Name        string
	Description string
	HopeCost    int    // Hope tokens required
	StressCost  int    // Stress marks required
	RefreshType string // "rest", "long_rest", "scene", "unlimited"
	MaxUses     int    // Uses before needing refresh (0 = unlimited if RefreshType allows)
	UsesLeft    int    // Current remaining uses (runtime state)

	// Difficulty Check
	DCType  string // "none" (auto-succeed), "target" (roll vs target's Difficulty), "fixed" (roll vs FixedDC)
	FixedDC int    // DC value if DCType is "fixed"

	// Token mechanics (for abilities like Unleash Chaos)
	UsesTokens     bool   // Whether this ability uses tokens
	TokenName      string // Name of tokens (e.g., "Chaos", "Ward")
	TokensHeld     int    // Current tokens held (runtime state)
	TokensMax      int    // Maximum tokens (0 = unlimited)

	Effects     []FeatureEffect
}

// SummonData contains information about summoned allies
type SummonData struct {
	AdversaryType string // Type of adversary to summon (reference to adversary definition)
	Quantity      int    // How many to summon
	DiceFormula   string // "1d4" if quantity is rolled
}
