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
	IsAdversary   bool
	Tier          string // "Tier 1", "Tier 2", etc.
	AdversaryType string // "Leader", "Bruiser", "Solo", etc.
	Difficulty    int    // Replaces Evasion for adversaries
	ATK           int    // Replaces Proficiency for adversaries
	Motives       string // Free text field
	Tactics       string // Free text field
	Experience    []ExperienceSkill

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

	// Relentless: how many times this adversary can be spotlighted in a single GM turn
	RelentlessLimit int

	// Combat state flags
	IsDefeated        bool
	IsFullyStressed   bool
	IsSummonTemplate  bool
}

// Resistance reduces incoming damage to specific type
type Resistance struct {
	Type     DamageType
	IsImmune bool // If true, 100% reduction (immunity). If false, 50% reduction (resistance)
}

// NewCombatant creates a player combatant with basic stats
func NewCombatant(id, name string, side Side, level, evasion, proficiency, hpMax, stressMax int) *Combatant {
	return &Combatant{
		ID:                id,
		Name:              name,
		Side:              side,
		IsAdversary:       false,
		Level:             level,
		Evasion:           evasion,
		Proficiency:       proficiency,
		HP:                hpMax,
		HPMax:             hpMax,
		Stress:            0,
		StressMax:         stressMax,
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
		ID:                id,
		Name:              name,
		Side:              SideEnemy,
		IsAdversary:       true,
		Tier:              tier,
		AdversaryType:     adversaryType,
		Difficulty:        difficulty,
		ATK:               atk,
		CustomThresholds:  thresholds,
		HP:                hpMax,
		HPMax:             hpMax,
		Stress:            0,
		StressMax:         stressMax,
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

func (c *Combatant) MaxSpotlightsPerGMTurn() int {
	if c.RelentlessLimit > 0 {
		return c.RelentlessLimit
	}
	return 1
}

// AddResistance adds 50% damage resistance to a damage type
func (c *Combatant) AddResistance(damageType DamageType) {
	c.Resistances = append(c.Resistances, Resistance{
		Type:     damageType,
		IsImmune: false,
	})
}

// AddImmunity adds 100% damage immunity to a damage type
func (c *Combatant) AddImmunity(damageType DamageType) {
	c.Resistances = append(c.Resistances, Resistance{
		Type:     damageType,
		IsImmune: true,
	})
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

// TakeDamage applies damage using threshold system
// Returns damage result with processed damage and HP damage
// roller can be nil if no dice rolling is needed
func (c *Combatant) TakeDamage(rawDamage int, damageType DamageType, roller DiceRoller) DamageResult {
	// Capture state before damage
	hpBefore := c.HP
	armorBefore := c.ArmorSlots

	actualDamage := rawDamage

	// Apply damage reduction from active effects (Ward-style)
	if roller != nil {
		reduction := c.calculateDamageReduction(roller)
		actualDamage -= reduction
	}

	if actualDamage < 0 {
		actualDamage = 0
	}

	// Apply resistance/immunity/vulnerability (after reductions, before armor/threshold)
	hasResistance := false
	hasImmunity := false

	for _, resist := range c.Resistances {
		if resist.Type == damageType {
			if resist.IsImmune {
				hasImmunity = true
				break
			}
			hasResistance = true
		}
	}

	if c.HasCondition(ConditionResistant) {
		hasResistance = true
	}

	if hasImmunity {
		return DamageResult{
			ProcessedDamage:  0,
			HPDamage:         0,
			Threshold:        ThresholdMinor,
			HPBefore:         hpBefore,
			HPAfter:          c.HP,
			ArmorUsed:        false,
			ArmorSlotsBefore: armorBefore,
			ArmorSlotsAfter:  c.ArmorSlots,
		}
	}
	if hasResistance {
		actualDamage = actualDamage / 2
	}

	if actualDamage < 0 {
		actualDamage = 0
	}

	// Store the processed damage (after all modifiers, before armor/threshold)
	processedDamage := actualDamage

	threshold := DamageThresholdFromAmount(actualDamage, c)

	armorReducedMinor := false
	armorUsed := false
	if threshold != ThresholdNone && c.ArmorSlots > 0 {
		c.ArmorSlots--
		armorUsed = true
		// Reduce threshold by one level: Severe→Major→Minor→None (0 HP)
		switch threshold {
		case ThresholdSevere:
			threshold = ThresholdMajor
		case ThresholdMajor:
			threshold = ThresholdMinor
		case ThresholdMinor:
			// Minor reduced by armor becomes 0 HP damage
			armorReducedMinor = true
		}
	}

	hpDamage := 0
	switch threshold {
	case ThresholdNone:
		hpDamage = 0
	case ThresholdMinor:
		if armorReducedMinor {
			hpDamage = 0
		} else {
			hpDamage = 1
		}
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

	return DamageResult{
		ProcessedDamage:  processedDamage,
		HPDamage:         hpDamage,
		Threshold:        threshold,
		HPBefore:         hpBefore,
		HPAfter:          c.HP,
		ArmorUsed:        armorUsed,
		ArmorSlotsBefore: armorBefore,
		ArmorSlotsAfter:  c.ArmorSlots,
	}
}

// calculateDamageReduction checks for active damage reduction effects
func (c *Combatant) calculateDamageReduction(roller DiceRoller) int {
	totalReduction := 0

	// Check domain abilities for damage reduction effects
	for _, ability := range c.DomainAbilities {
		for _, effect := range ability.Effects {
			if effect.Category == FeatureEffectDamageReduction {
				// Add flat magnitude
				totalReduction += effect.Magnitude

				// Roll dice if formula present
				if effect.DiceFormula != "" {
					notation, err := ParseDiceNotation(effect.DiceFormula)
					if err == nil {
						totalReduction += notation.Roll(roller)
					}
				}
			}
		}
	}

	// Check adversary features for damage reduction effects
	for _, feature := range c.AdversaryFeatures {
		for _, effect := range feature.Effects {
			if effect.Category == FeatureEffectDamageReduction {
				// Add flat magnitude
				totalReduction += effect.Magnitude

				// Roll dice if formula present
				if effect.DiceFormula != "" {
					notation, err := ParseDiceNotation(effect.DiceFormula)
					if err == nil {
						totalReduction += notation.Roll(roller)
					}
				}
			}
		}
	}

	return totalReduction
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
		c.AddCondition(&ActiveCondition{
			Type:     ConditionVulnerable,
			Duration: DurationUntilCleared,
		})
	}
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

// ApplyPassiveResistances processes all passive features/abilities and applies resistance effects
func (c *Combatant) ApplyPassiveResistances() {
	// Process adversary features
	for _, feature := range c.AdversaryFeatures {
		if !feature.Passive {
			continue
		}
		for _, effect := range feature.Effects {
			if effect.Category == FeatureEffectDamageResistance {
				damageType := parseDamageType(effect.DamageType)
				if effect.Magnitude >= 100 {
					c.AddImmunity(damageType)
				} else {
					c.AddResistance(damageType)
				}
			}
		}
	}

	// Process domain abilities
	for _, ability := range c.DomainAbilities {
		if !ability.Passive {
			continue
		}
		for _, effect := range ability.Effects {
			if effect.Category == FeatureEffectDamageResistance {
				damageType := parseDamageType(effect.DamageType)
				if effect.Magnitude >= 100 {
					c.AddImmunity(damageType)
				} else {
					c.AddResistance(damageType)
				}
			}
		}
	}
}

// parseDamageType converts string to DamageType
func parseDamageType(s string) DamageType {
	if s == "magic" {
		return DamageMagic
	}
	return DamagePhysical
}

// CloneCombatant creates a deep copy of a combatant template with a new unique ID
func CloneCombatant(template *Combatant, uniqueID string) *Combatant {
	clone := &Combatant{
		ID:                uniqueID,
		Name:              template.Name,
		Side:              template.Side,
		Level:             template.Level,
		Evasion:           template.Evasion,
		Proficiency:       template.Proficiency,
		IsAdversary:       template.IsAdversary,
		Tier:              template.Tier,
		AdversaryType:     template.AdversaryType,
		Difficulty:        template.Difficulty,
		ATK:               template.ATK,
		Motives:           template.Motives,
		Tactics:           template.Tactics,
		HP:                template.HPMax,
		HPMax:             template.HPMax,
		Stress:            0,
		StressMax:         template.StressMax,
		ArmorSlots:        template.ArmorSlots,
		DamageDice:        template.DamageDice,
		SpotlightTokens:   0,
		RelentlessLimit:   template.RelentlessLimit,
		IsDefeated:        false,
		IsFullyStressed:   false,
		IsSummonTemplate:  template.IsSummonTemplate,
		AdversaryFeatures: make([]AdversaryFeature, len(template.AdversaryFeatures)),
		DomainAbilities:   make([]DomainAbility, len(template.DomainAbilities)),
		Resistances:       make([]Resistance, len(template.Resistances)),
		Conditions:        make([]*ActiveCondition, 0),
		Experience:        make([]ExperienceSkill, len(template.Experience)),
	}

	if len(template.CustomThresholds) > 0 {
		clone.CustomThresholds = make([]int, len(template.CustomThresholds))
		copy(clone.CustomThresholds, template.CustomThresholds)
	}

	copy(clone.AdversaryFeatures, template.AdversaryFeatures)
	copy(clone.DomainAbilities, template.DomainAbilities)
	copy(clone.Resistances, template.Resistances)
	copy(clone.Experience, template.Experience)

	return clone
}
