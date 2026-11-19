package models

import "slices"

// ExperienceSkill represents an adversary's trained skill
type ExperienceSkill struct {
	Name  string
	Bonus int
}

// Combatant represents any character in combat (player or enemy)
type Combatant struct {
	ID   string
	Name string
	Side Side

	// Player-specific stats
	Level       int // PC level for threshold calculation
	Evasion     int
	Proficiency int // number of weapon damage dice
	AttackTrait int // trait modifier added to attack and action rolls
	Traits      Traits
	Hope        int

	// Adversary-specific stats
	IsAdversary   bool
	Tier          string // "Tier 1", "Tier 2", etc.
	AdversaryType string // "Leader", "Bruiser", "Solo", etc.
	Difficulty    int    // Replaces Evasion for adversaries
	ATK           int    // Replaces Proficiency for adversaries
	Motives       string // Free text field
	Tactics       string // Free text field
	Experience    []ExperienceSkill

	// Custom damage thresholds (for both Players and Adversaries)
	CustomThresholds []int // [Major, Severe]; 0 means that tier is unreachable ("None" in the stat block)
	// Empty = PC default 3+level / 6+level, adversary default [4, 7]

	// Resources
	HP         int
	HPMax      int
	Stress     int
	StressMax  int
	ArmorSlots int // How many hits can armor absorb

	// Damage dice notation (e.g., "2d6+2", "1d6+1d6+4")
	DamageDice string

	// Spotlight system
	SpotlightTokens int

	// Feature system
	AdversaryFeatures []AdversaryFeature // For enemies
	DomainAbilities   []DomainAbility    // For players

	// Resistances
	Resistances []Resistance

	// Active conditions
	Conditions []*ActiveCondition

	// Relentless: how many times this adversary can be spotlighted in a single GM turn
	RelentlessLimit int

	// Combat state flags
	IsDefeated       bool
	IsFullyStressed  bool
	IsSummonTemplate bool
	// Fallen PCs have resolved their death move and are out of the fight
	Fallen bool
	// Set for the duration of a spotlight granted by an ally's feature that halves damage
	HalfDamage bool
}

// MaxHope is the most Hope a single PC can hold
const MaxHope = 6

// Traits are a PC's six trait modifiers, used for reaction rolls
type Traits struct {
	Agility   int
	Strength  int
	Finesse   int
	Instinct  int
	Presence  int
	Knowledge int
}

// Modifier returns the modifier for a trait name such as "agility" (0 if unknown)
func (t Traits) Modifier(trait string) int {
	switch trait {
	case "agility":
		return t.Agility
	case "strength":
		return t.Strength
	case "finesse":
		return t.Finesse
	case "instinct":
		return t.Instinct
	case "presence":
		return t.Presence
	case "knowledge":
		return t.Knowledge
	default:
		return 0
	}
}

// Resistance reduces incoming damage to specific type
type Resistance struct {
	Type     DamageType
	IsImmune bool // If true, 100% reduction (immunity). If false, 50% reduction (resistance)
}

func newCombatant(id, name string, side Side, hpMax, stressMax int) *Combatant {
	return &Combatant{
		ID:                id,
		Name:              name,
		Side:              side,
		HP:                hpMax,
		HPMax:             hpMax,
		StressMax:         stressMax,
		AdversaryFeatures: make([]AdversaryFeature, 0),
		DomainAbilities:   make([]DomainAbility, 0),
		Resistances:       make([]Resistance, 0),
		Conditions:        make([]*ActiveCondition, 0),
		Experience:        make([]ExperienceSkill, 0),
	}
}

// NewCombatant creates a player combatant with basic stats
func NewCombatant(id, name string, side Side, level, evasion, proficiency, hpMax, stressMax int) *Combatant {
	c := newCombatant(id, name, side, hpMax, stressMax)
	c.Level = level
	c.Evasion = evasion
	c.Proficiency = proficiency
	return c
}

// NewAdversary creates an adversary combatant
func NewAdversary(id, name, tier, adversaryType string, difficulty, atk, hpMax, stressMax int, thresholds []int) *Combatant {
	c := newCombatant(id, name, SideEnemy, hpMax, stressMax)
	c.IsAdversary = true
	c.Tier = tier
	c.AdversaryType = adversaryType
	c.Difficulty = difficulty
	c.ATK = atk
	c.CustomThresholds = thresholds
	return c
}

func (c *Combatant) MaxSpotlightsPerGMTurn() int {
	if c.RelentlessLimit > 0 {
		return c.RelentlessLimit
	}
	return 1
}

// AddResistance adds 50% damage resistance to a damage type
func (c *Combatant) AddResistance(damageType DamageType) {
	c.Resistances = append(c.Resistances, Resistance{Type: damageType})
}

// AddImmunity adds 100% damage immunity to a damage type
func (c *Combatant) AddImmunity(damageType DamageType) {
	c.Resistances = append(c.Resistances, Resistance{Type: damageType, IsImmune: true})
}

// DamageResult contains the results of damage application
type DamageResult struct {
	ProcessedDamage  int             // Damage after resistance/immunity/vulnerability
	HPDamage         int             // Actual HP damage dealt (1/2/3)
	Threshold        DamageThreshold // Threshold calculated from processed damage
	HPBefore         int             // HP before taking damage
	HPAfter          int             // HP after taking damage
	ArmorUsed        bool            // Was an armor slot used?
	ArmorSlotsBefore int             // Armor slots before
	ArmorSlotsAfter  int             // Armor slots after
}

// TakeDamage applies damage using the threshold system; roller may be nil
func (c *Combatant) TakeDamage(rawDamage int, damageType DamageType, roller DiceRoller) DamageResult {
	return c.takeDamage(rawDamage, damageType, roller, 1)
}

// TakeDirectDamage applies damage that cannot be reduced by armor slots
func (c *Combatant) TakeDirectDamage(rawDamage int, damageType DamageType, roller DiceRoller) DamageResult {
	return c.takeDamage(rawDamage, damageType, roller, 0)
}

// TakeDamageWithArmorCost applies damage where using armor marks armorCost slots instead of 1
func (c *Combatant) TakeDamageWithArmorCost(rawDamage int, damageType DamageType, roller DiceRoller, armorCost int) DamageResult {
	return c.takeDamage(rawDamage, damageType, roller, max(armorCost, 1))
}

// armorCost is how many slots reducing this damage marks; 0 means armor can't be used
func (c *Combatant) takeDamage(rawDamage int, damageType DamageType, roller DiceRoller, armorCost int) DamageResult {
	result := DamageResult{Threshold: ThresholdNone, HPBefore: c.HP, ArmorSlotsBefore: c.ArmorSlots}
	defer func() {
		result.HPAfter = c.HP
		result.ArmorSlotsAfter = c.ArmorSlots
	}()

	// SRD p50: resistance applies before any other reduction
	resistant := c.HasCondition(ConditionResistant)
	for _, resist := range c.Resistances {
		if resist.Type != damageType {
			continue
		}
		if resist.IsImmune {
			return result
		}
		resistant = true
	}

	damage := rawDamage
	if resistant {
		damage = (damage + 1) / 2 // SRD rounds up
	}
	if roller != nil {
		damage -= c.calculateDamageReduction(roller)
	}
	damage = max(damage, 0)
	result.ProcessedDamage = damage

	threshold := DamageThresholdFromAmount(damage, c)
	hpDamage := threshold.HPMarked()
	if armorCost > 0 && threshold != ThresholdNone && c.ArmorSlots >= armorCost {
		c.ArmorSlots -= armorCost
		result.ArmorUsed = true
		// Armor steps the severity down one tier; a Minor hit marks nothing
		hpDamage--
		threshold = max(threshold-1, ThresholdMinor)
	}

	c.markHP(hpDamage)
	result.Threshold = threshold
	result.HPDamage = hpDamage
	return result
}

// passiveEffects lists every effect carried by the combatant's passive features and abilities
func (c *Combatant) passiveEffects() []FeatureEffect {
	var effects []FeatureEffect
	for _, f := range c.AdversaryFeatures {
		if f.Passive {
			effects = append(effects, f.Effects...)
		}
	}
	for _, a := range c.DomainAbilities {
		if a.Passive {
			effects = append(effects, a.Effects...)
		}
	}
	return effects
}

// calculateDamageReduction sums flat Ward-style reductions from passive effects
func (c *Combatant) calculateDamageReduction(roller DiceRoller) int {
	total := 0
	for _, effect := range c.passiveEffects() {
		if effect.Category != FeatureEffectDamageReduction {
			continue
		}
		total += effect.Magnitude
		if notation, err := ParseDiceNotation(effect.DiceFormula); err == nil {
			total += notation.Roll(roller)
		}
	}
	return total
}

func (c *Combatant) markHP(amount int) {
	if amount <= 0 {
		return
	}
	c.HP -= amount
	if c.HP <= 0 {
		c.HP = 0
		c.IsDefeated = true
	}
}

// LoseArmorSlots marks Armor Slots without their benefit. Each slot that
// can't be marked costs 1 HP instead. Returns the HP marked.
func (c *Combatant) LoseArmorSlots(amount int) int {
	marked := min(amount, c.ArmorSlots)
	c.ArmorSlots -= marked
	hpMarked := max(amount-marked, 0)
	c.markHP(hpMarked)
	return hpMarked
}

// Heal restores HP
func (c *Combatant) Heal(amount int) {
	c.HP += amount
	if c.HP > c.HPMax {
		c.HP = c.HPMax
	}
}

// AddStress increases stress level
func (c *Combatant) AddStress(amount int) {
	if amount <= 0 {
		return
	}

	// SRD: Stress that can't be marked costs 1 HP, however much is left over
	free := max(c.StressMax-c.Stress, 0)
	if amount > free {
		amount = free
		c.markHP(1)
	}
	if amount == 0 {
		return
	}

	c.Stress += amount
	if c.Stress >= c.StressMax {
		c.IsFullyStressed = true
		c.AddCondition(&ActiveCondition{
			Type:     ConditionVulnerable,
			Duration: DurationUntilCleared,
		})
	}
}

// GainHope adds Hope up to MaxHope
func (c *Combatant) GainHope(amount int) {
	c.Hope = min(c.Hope+amount, MaxHope)
}

// SpendHope deducts Hope if the combatant has enough
func (c *Combatant) SpendHope(amount int) bool {
	if c.Hope >= amount {
		c.Hope -= amount
		return true
	}
	return false
}

// LoseHope removes up to amount Hope
func (c *Combatant) LoseHope(amount int) {
	c.Hope = max(c.Hope-amount, 0)
}

func (c *Combatant) ClearStress(amount int) {
	c.Stress -= amount
	if c.Stress < 0 {
		c.Stress = 0
	}
	if c.Stress < c.StressMax && c.IsFullyStressed {
		c.IsFullyStressed = false
		c.RemoveCondition(ConditionVulnerable)
	}
}

func (c *Combatant) AddCondition(condition *ActiveCondition) {
	if c.HasCondition(condition.Type) {
		return
	}
	c.Conditions = append(c.Conditions, condition)
}

// RemoveCondition removes a status effect by type
func (c *Combatant) RemoveCondition(condType ConditionType) bool {
	for i, cond := range c.Conditions {
		if cond.Type == condType {
			c.Conditions = append(c.Conditions[:i], c.Conditions[i+1:]...)
			return true
		}
	}
	return false
}

// HasCondition checks if combatant has a specific condition
func (c *Combatant) HasCondition(condType ConditionType) bool {
	for _, cond := range c.Conditions {
		if cond.Type == condType {
			return true
		}
	}
	return false
}

// ClearableCondition returns the first harmful condition an action can clear, or nil.
// Restrained only limits movement, which the simulator does not model, so it is never worth a turn.
func (c *Combatant) ClearableCondition() *ActiveCondition {
	for _, cond := range c.Conditions {
		switch cond.Type {
		case ConditionBurning, ConditionPoisoned:
			return cond
		case ConditionVulnerable:
			// Vulnerable from full Stress only ends by clearing Stress
			if !c.IsFullyStressed {
				return cond
			}
		}
	}
	return nil
}

// IsAlive returns whether combatant is still in the fight
func (c *Combatant) IsAlive() bool {
	return !c.IsDefeated
}

// GetDefense returns appropriate defense stat
func (c *Combatant) GetDefense() int {
	if c.IsAdversary {
		return c.Difficulty
	}
	return c.Evasion
}

// GetAttackBonus returns appropriate attack bonus
func (c *Combatant) GetAttackBonus() int {
	if c.IsAdversary {
		return c.ATK
	}
	return c.AttackTrait
}

// ApplyPassiveResistances turns passive damage_resistance effects into Resistances (100+ = immunity)
func (c *Combatant) ApplyPassiveResistances() {
	for _, effect := range c.passiveEffects() {
		if effect.Category != FeatureEffectDamageResistance {
			continue
		}
		damageType := ParseDamageType(effect.DamageType)
		if effect.Magnitude >= 100 {
			c.AddImmunity(damageType)
		} else {
			c.AddResistance(damageType)
		}
	}
}

// ParseDamageType converts string to DamageType
func ParseDamageType(s string) DamageType {
	if s == "magic" {
		return DamageMagic
	}
	return DamagePhysical
}

// CloneCombatant copies a template at full health with a new ID and fresh combat state
func CloneCombatant(template *Combatant, uniqueID string) *Combatant {
	clone := *template
	clone.ID = uniqueID
	clone.HP = template.HPMax
	clone.Hope = 0
	clone.Stress = 0
	clone.SpotlightTokens = 0
	clone.IsDefeated = false
	clone.IsFullyStressed = false
	clone.Fallen = false
	clone.HalfDamage = false

	clone.CustomThresholds = slices.Clone(template.CustomThresholds)
	clone.AdversaryFeatures = slices.Clone(template.AdversaryFeatures)
	clone.DomainAbilities = slices.Clone(template.DomainAbilities)
	clone.Experience = slices.Clone(template.Experience)
	clone.Resistances = slices.Clone(template.Resistances)
	clone.Conditions = make([]*ActiveCondition, 0)

	clone.ApplyPassiveResistances()
	return &clone
}

// AttackDamageFormula is the dice the basic attack really rolls.
// SRD p50: a PC rolls Proficiency damage dice; the flat modifier is unchanged.
func (c *Combatant) AttackDamageFormula() string {
	formula := c.DamageDice
	if formula == "" {
		formula = "1d6"
	}
	if c.IsAdversary || c.Proficiency <= 0 {
		return formula
	}
	notation, err := ParseDiceNotation(formula)
	if err != nil {
		return formula
	}
	for i := range notation.Terms {
		if notation.Terms[i].Count > 0 {
			notation.Terms[i].Count = c.Proficiency
			break
		}
	}
	return notation.String()
}
