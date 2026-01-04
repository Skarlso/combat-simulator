package models

// Combatant represents any character in combat (player or enemy)
type Combatant struct {
	ID   string
	Name string
	Side Side

	// Core stats
	Evasion     int
	Proficiency int // Used for damage thresholds

	// Resources
	HP          int
	HPMax       int
	Stress      int
	StressMax   int
	ArmorSlots  int // How many hits can armor absorb

	// Spotlight system
	SpotlightTokens int

	// Abilities
	Abilities []*Ability

	// Resistances
	Resistances []Resistance

	// Active conditions
	Conditions []*ActiveCondition

	// Combat state flags
	IsDefeated      bool
	IsFullyStressed bool
}

// Resistance reduces incoming damage of specific type
type Resistance struct {
	Type   DamageType
	Amount int // Flat reduction
}

// NewCombatant creates a combatant with basic stats
func NewCombatant(id, name string, side Side, evasion, proficiency, hpMax, stressMax int) *Combatant {
	return &Combatant{
		ID:              id,
		Name:            name,
		Side:            side,
		Evasion:         evasion,
		Proficiency:     proficiency,
		HP:              hpMax,
		HPMax:           hpMax,
		Stress:          0,
		StressMax:       stressMax,
		SpotlightTokens: 0,
		Abilities:       make([]*Ability, 0),
		Resistances:     make([]Resistance, 0),
		Conditions:      make([]*ActiveCondition, 0),
		IsDefeated:      false,
		IsFullyStressed: false,
	}
}

// AddAbility attaches an ability to this combatant
func (c *Combatant) AddAbility(ability *Ability) {
	c.Abilities = append(c.Abilities, ability)
}

// AddResistance adds damage resistance
func (c *Combatant) AddResistance(damageType DamageType, amount int) {
	c.Resistances = append(c.Resistances, Resistance{
		Type:   damageType,
		Amount: amount,
	})
}

// TakeDamage applies damage after resistances
func (c *Combatant) TakeDamage(amount int, damageType DamageType) int {
	// Apply resistances
	actualDamage := amount
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

	// Apply to HP
	c.HP -= actualDamage
	if c.HP <= 0 {
		c.HP = 0
		c.IsDefeated = true
	}

	return actualDamage
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
