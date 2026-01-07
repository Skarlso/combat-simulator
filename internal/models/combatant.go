package models

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
	Proficiency int // Used for damage thresholds

	// Adversary-specific stats
	IsAdversary     bool
	Tier            string // "Tier 1", "Tier 2", etc.
	AdversaryType   string // "Leader", "Bruiser", "Solo", etc.
	Difficulty      int    // Replaces Evasion for adversaries
	ATK             int    // Replaces Proficiency for adversaries
	Motives         string // Free text field
	Tactics         string // Free text field
	Experience      []ExperienceSkill

	// Custom damage thresholds (for both Players and Adversaries)
	CustomThresholds []int // e.g., [8, 16] means: <8=Minor, 8-15=Major, 16+=Severe
	// Empty = use default [4, 7]

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

	// Combat state flags
	IsDefeated      bool
	IsFullyStressed bool
}

// Resistance reduces incoming damage to specific type
type Resistance struct {
	Type   DamageType
	Amount int // Flat reduction
}

// NewCombatant creates a player combatant with basic stats
func NewCombatant(id, name string, side Side, level, evasion, proficiency, hpMax, stressMax int) *Combatant {
	return &Combatant{
		ID:              id,
		Name:            name,
		Side:            side,
		IsAdversary:     false,
		Level:           level,
		Evasion:         evasion,
		Proficiency:     proficiency,
		HP:              hpMax,
		HPMax:           hpMax,
		Stress:          0,
		StressMax:       stressMax,
		SpotlightTokens:   0,
		AdversaryFeatures: make([]AdversaryFeature, 0),
		DomainAbilities:   make([]DomainAbility, 0),
		Resistances:       make([]Resistance, 0),
		Conditions:        make([]*ActiveCondition, 0),
		Experience:        make([]ExperienceSkill, 0),
		IsDefeated:        false,
		IsFullyStressed:   false,
	}
}

// NewAdversary creates an adversary combatant
func NewAdversary(id, name, tier, adversaryType string, difficulty, atk, hpMax, stressMax int, thresholds []int) *Combatant {
	return &Combatant{
		ID:               id,
		Name:             name,
		Side:             SideEnemy,
		IsAdversary:      true,
		Tier:             tier,
		AdversaryType:    adversaryType,
		Difficulty:       difficulty,
		ATK:              atk,
		CustomThresholds: thresholds,
		HP:               hpMax,
		HPMax:            hpMax,
		Stress:           0,
		StressMax:        stressMax,
		SpotlightTokens:   0,
		AdversaryFeatures: make([]AdversaryFeature, 0),
		DomainAbilities:   make([]DomainAbility, 0),
		Resistances:       make([]Resistance, 0),
		Conditions:        make([]*ActiveCondition, 0),
		Experience:        make([]ExperienceSkill, 0),
		IsDefeated:        false,
		IsFullyStressed:   false,
	}
}

// AddResistance adds damage resistance
func (c *Combatant) AddResistance(damageType DamageType, amount int) {
	c.Resistances = append(c.Resistances, Resistance{
		Type:   damageType,
		Amount: amount,
	})
}

// TakeDamage applies damage using threshold system
// Returns actual HP damage dealt
func (c *Combatant) TakeDamage(rawDamage int, damageType DamageType) int {
	// Apply resistances to raw damage
	actualDamage := rawDamage
	for _, resist := range c.Resistances {
		if resist.Type == damageType {
			actualDamage -= resist.Amount
		}
	}

	if actualDamage < 0 {
		actualDamage = 0
	}

	// Check armor first
	if c.ArmorSlots > 0 {
		c.ArmorSlots--
		return 0 // Armor absorbed the hit
	}

	// Apply threshold-based HP damage
	threshold := DamageThresholdFromAmount(actualDamage, c)
	hpDamage := 0

	switch threshold {
	case ThresholdMinor:
		hpDamage = 1
	case ThresholdMajor:
		hpDamage = 2
	case ThresholdSevere:
		hpDamage = 3
	}

	c.HP -= hpDamage
	if c.HP <= 0 {
		c.HP = 0
		c.IsDefeated = true
	}

	return hpDamage
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
	if c.Stress >= c.StressMax {
		// Stress overflow: mark HP instead
		c.HP--
		if c.HP <= 0 {
			c.HP = 0
			c.IsDefeated = true
		}
		return
	}

	c.Stress += amount
	if c.Stress >= c.StressMax {
		c.Stress = c.StressMax
		c.IsFullyStressed = true
	}
}

// ClearStress removes stress
func (c *Combatant) ClearStress(amount int) {
	c.Stress -= amount
	if c.Stress < 0 {
		c.Stress = 0
	}
	if c.Stress < c.StressMax {
		c.IsFullyStressed = false
	}
}

// AddCondition applies a status effect
func (c *Combatant) AddCondition(condition *ActiveCondition) {
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

// IsAlive returns whether combatant is still in the fight
func (c *Combatant) IsAlive() bool {
	return !c.IsDefeated
}

// AddSpotlightToken increments spotlight tokens
func (c *Combatant) AddSpotlightToken() {
	c.SpotlightTokens++
}

// ResetSpotlightTokens clears tokens after taking action
func (c *Combatant) ResetSpotlightTokens() {
	c.SpotlightTokens = 0
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
	return c.Proficiency
}

// AddAdversaryFeature attaches a feature to an adversary
func (c *Combatant) AddAdversaryFeature(feature AdversaryFeature) {
	c.AdversaryFeatures = append(c.AdversaryFeatures, feature)
}

// AddDomainAbility attaches a domain ability to a player
func (c *Combatant) AddDomainAbility(ability DomainAbility) {
	c.DomainAbilities = append(c.DomainAbilities, ability)
}
