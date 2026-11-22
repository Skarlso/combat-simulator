package models

import (
	"fmt"
	"slices"
	"strings"
)

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
	Traits      Traits
	// Traits rolled for weapon attacks and for Spellcast Rolls, by name ("strength"); empty adds nothing
	WeaponTrait    string
	SpellcastTrait string
	Hope           int

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
	ArmorMax   int // Armor Score, the most slots a clear can restore; taken from ArmorSlots when cloned for combat

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
	// Taken off Evasion or Difficulty by defense_penalty effects, for the rest of the fight
	DefensePenalty int
	// Severity of the last damage taken, for filters such as "when you deal Severe damage"
	LastSeverity DamageThreshold
	// Whether the PC's last action roll was with Fear, for "on a success with Fear" filters
	RolledFear bool
	// Total of the PC's last action roll, for features that reuse it against other targets
	LastRollTotal int

	// SRD consumables: Health and Stamina Potions clear 1d4+1 Hit Points or Stress, up to five of each
	HealthPotions  int
	StaminaPotions int
}

// PotionType names the two consumables the simulator models
type PotionType string

const (
	PotionHealth  PotionType = "health"
	PotionStamina PotionType = "stamina"
	// MaxPotionsEach is the SRD cap on how many of one consumable a character can carry
	MaxPotionsEach = 5
)

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

// Thresholds returns the Major and Severe damage thresholds: the stat block's own, or the
// SRD defaults (PC 3+level / 6+level, adversary 4 / 7)
func (c *Combatant) Thresholds() (major, severe int) {
	if len(c.CustomThresholds) >= 2 {
		return c.CustomThresholds[0], c.CustomThresholds[1]
	}
	if c.IsAdversary {
		return 4, 7
	}
	return 3 + c.Level, 6 + c.Level
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

// LogPart is one named piece of a roll or damage total, kept so the log can say where numbers come from
type LogPart struct {
	Label string
	Value int
}

// SumParts adds up the values of named parts
func SumParts(parts []LogPart) int {
	total := 0
	for _, p := range parts {
		total += p.Value
	}
	return total
}

// DamageResult contains the results of damage application
type DamageResult struct {
	Reactions        []string        // on_damage_incoming passives the target paid for to soften this hit
	Reduced          []string        // what cut the damage before thresholds: "resistance", "Rune Ward -6"
	Severity         []string        // the severity by thresholds, then each step down with its cause
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
func (c *Combatant) takeDamage(rawDamage int, damageType DamageType, roller DiceRoller, armorCost int) (result DamageResult) {
	result = DamageResult{Threshold: ThresholdNone, HPBefore: c.HP, ArmorSlotsBefore: c.ArmorSlots}
	// result is a named return so this still reaches the caller
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
			result.Reduced = append(result.Reduced, "immunity")
			return result
		}
		resistant = true
	}

	damage := rawDamage
	if resistant {
		damage = (damage + 1) / 2 // SRD rounds up
		result.Reduced = append(result.Reduced, "resistance halves it")
	}
	if roller != nil {
		for _, part := range c.passiveParts(FeatureEffectDamageReduction, damageType, roller) {
			damage -= part.Value
			result.Reduced = append(result.Reduced, fmt.Sprintf("%s -%d", part.Label, part.Value))
		}
	}
	damage = max(damage, 0)

	damage, drops := c.reactToDamage(&result, damage, damageType, armorCost, roller)
	result.ProcessedDamage = damage

	threshold := DamageThresholdFromAmount(damage, c)
	major, severe := c.Thresholds()
	result.Severity = []string{fmt.Sprintf("%s at %d/%d", threshold, major, severe)}
	for _, drop := range drops {
		if threshold == ThresholdNone {
			break
		}
		threshold = max(threshold-DamageThreshold(drop.Value), ThresholdNone)
		result.Severity = append(result.Severity, fmt.Sprintf("%s by %s", threshold, drop.Label))
	}
	hpDamage := threshold.HPMarked()
	if armorCost > 0 && threshold != ThresholdNone && c.ArmorSlots >= armorCost {
		c.ArmorSlots -= armorCost
		result.ArmorUsed = true
		// Armor steps the severity down one tier; a Minor hit marks nothing
		hpDamage--
		threshold--
		result.Severity = append(result.Severity, fmt.Sprintf("%s by an Armor Slot", threshold))
	}

	c.markHP(hpDamage)
	c.LastSeverity = threshold
	result.Threshold = threshold
	result.HPDamage = hpDamage
	return result
}

// severityMatches checks an on_damage_incoming filter against how severe the hit is right now
func severityMatches(filter string, severity DamageThreshold) bool {
	for _, part := range strings.Split(filter, "+") {
		switch part {
		case "if_severe":
			if severity != ThresholdSevere {
				return false
			}
		case "if_major_or_worse":
			if severity < ThresholdMajor {
				return false
			}
		case "if_minor":
			if severity != ThresholdMinor {
				return false
			}
		}
	}
	return true
}

// reactToDamage lets a PC's on_damage_incoming passives soften a hit that would still mark a
// Hit Point after armor, paying their costs. It returns the damage left and the thresholds
// to drop with their causes, standing severity reductions first.
func (c *Combatant) reactToDamage(result *DamageResult, damage int, damageType DamageType, armorCost int, roller DiceRoller) (int, []LogPart) {
	drops := c.passiveParts(FeatureEffectSeverityReduction, damageType, nil)
	for i := range c.DomainAbilities {
		a := &c.DomainAbilities[i]
		if !a.Passive || a.Trigger.Type != TriggerOnDamageIncoming {
			continue
		}
		armorReady := armorCost > 0 && c.ArmorSlots >= armorCost
		severity := max(DamageThresholdFromAmount(damage, c)-DamageThreshold(SumParts(drops)), ThresholdNone)
		hp := severity.HPMarked()
		if armorReady && hp > 0 {
			hp--
		}
		if hp <= 0 {
			break
		}
		if !severityMatches(a.Trigger.Filter, severity) || !c.MeetsFilter(a.Trigger.Filter) || !softens(a.Effects, damageType) || !c.canReact(a) {
			continue
		}
		// armor_damage on a reaction is extra Armor Slots marked on top of the usual one
		extraArmor := 0
		for _, e := range a.Effects {
			if e.Category == FeatureEffectArmorDamage {
				extraArmor += e.Magnitude
			}
		}
		if extraArmor > 0 && (!armorReady || c.ArmorSlots < armorCost+extraArmor) {
			continue
		}

		tokens := a.TokensHeld
		c.payReaction(a)
		c.ArmorSlots -= extraArmor
		result.Reactions = append(result.Reactions, a.Name)
		for _, e := range a.Effects {
			amount := c.scaled(a.Name, e, tokens).amount(roller)
			switch e.Category {
			case FeatureEffectSeverityReduction:
				drops = append(drops, LogPart{a.Name, amount})
			case FeatureEffectDamageReduction:
				damage = max(damage-amount, 0)
				result.Reduced = append(result.Reduced, fmt.Sprintf("%s -%d", a.Name, amount))
			case FeatureEffectDamageResistance:
				if e.Magnitude >= 100 {
					damage = 0
					result.Reduced = append(result.Reduced, a.Name+" negates it")
				} else {
					damage = (damage + 1) / 2
					result.Reduced = append(result.Reduced, a.Name+" halves it")
				}
			}
		}
	}
	return damage, drops
}

// softens reports whether a reaction's effects apply to this damage type (no type means any)
func softens(effects []FeatureEffect, damageType DamageType) bool {
	for _, e := range effects {
		switch e.Category {
		case FeatureEffectSeverityReduction, FeatureEffectDamageReduction, FeatureEffectDamageResistance:
			if e.DamageType == "" || ParseDamageType(e.DamageType) == damageType {
				return true
			}
		}
	}
	return false
}

// KeepsStressReserve reports whether an optional Stress cost leaves the last slot unmarked.
// Defensive reactions never mark it: full Stress makes a PC Vulnerable.
func (c *Combatant) KeepsStressReserve(cost int) bool {
	return cost <= 0 || c.Stress+cost < c.StressMax
}

// KeepsHalfStress reports whether an optional Stress cost stays within half the Stress track,
// the most a PC marks for extra damage and other offensive passives
func (c *Combatant) KeepsHalfStress(cost int) bool {
	return cost <= 0 || c.Stress+cost <= c.StressMax/2
}

func tokenCost(effects []FeatureEffect) int {
	cost := 0
	for _, e := range effects {
		if e.Category == FeatureEffectTokenSpend {
			cost += e.Magnitude
		}
	}
	return cost
}

func (c *Combatant) canReact(a *DomainAbility) bool {
	if a.HopeCost > c.Hope || !c.KeepsStressReserve(a.StressCost) {
		return false
	}
	if a.MaxUses > 0 && a.UsesLeft <= 0 {
		return false
	}
	return tokenCost(a.Effects) <= a.TokensHeld
}

func (c *Combatant) payReaction(a *DomainAbility) {
	c.SpendHope(a.HopeCost)
	c.AddStress(a.StressCost)
	if a.MaxUses > 0 {
		a.UsesLeft--
	}
	a.TokensHeld -= tokenCost(a.Effects)
}

// PCTier is the tier a PC's level falls in: 1, 2-4, 5-7, 8-10
func (c *Combatant) PCTier() int {
	switch {
	case c.Level >= 8:
		return 4
	case c.Level >= 5:
		return 3
	case c.Level >= 2:
		return 2
	default:
		return 1
	}
}

// ScaleValue is the number a ScaleBy name stands for, never below 0; no name means 1
func (c *Combatant) ScaleValue(by string) int {
	switch by {
	case "":
		return 1
	case "level":
		return max(c.Level, 0)
	case "tier":
		return c.PCTier()
	case "proficiency":
		return max(c.Proficiency, 0)
	case "spellcast":
		return max(c.Traits.Modifier(c.SpellcastTrait), 0)
	default:
		return max(c.Traits.Modifier(by), 0)
	}
}

// MeetsFilter checks the filters that only look at the combatant's own state; any other passes
func (c *Combatant) MeetsFilter(filter string) bool {
	for _, part := range strings.Split(filter, "+") {
		ok := true
		switch part {
		case "if_vulnerable":
			ok = c.HasCondition(ConditionVulnerable)
		case "if_low_hp":
			ok = c.HP <= 2
		case "if_fully_stressed":
			ok = c.StressMax > 0 && c.Stress >= c.StressMax
		case "if_hope_2":
			ok = c.Hope >= 2
		case "if_out_of_armor":
			ok = c.ArmorSlots == 0
		case "if_rolled_fear":
			ok = c.RolledFear
		case "if_rolled_hope":
			ok = !c.RolledFear
		}
		if !ok {
			return false
		}
	}
	return true
}

// standingEffect is a passive's effect in force right now, with the feature it comes from, the
// tokens that feature holds and the value its ScaleBy stands for
type standingEffect struct {
	FeatureEffect
	source string
	tokens int
	scale  int
}

// amount is the Magnitude plus the dice, both taken once per scale step (and per held token
// with PerToken); a nil roller skips the dice
func (e standingEffect) amount(roller DiceRoller) int {
	times := e.scale
	if e.PerToken {
		times *= e.tokens
	}
	total := e.Magnitude * times
	if roller != nil && e.DiceFormula != "" {
		if notation, err := ParseDiceNotation(e.DiceFormula); err == nil {
			for range times {
				total += notation.Roll(roller)
			}
		}
	}
	return total
}

// passiveEffects lists every effect in force from the combatant's passive features and abilities.
// A passive that tracks tokens only counts while it holds some, one without a trigger only while
// its filter holds, and one with a reactive trigger never: it applies to single rolls instead.
func (c *Combatant) passiveEffects() []standingEffect {
	var effects []standingEffect
	add := func(name string, passive bool, trigger FeatureTriggerCondition, usesTokens bool, tokens int, list []FeatureEffect) {
		if !passive || trigger.Type.IsReactive() || (usesTokens && tokens <= 0) {
			return
		}
		if trigger.Type == "" && !c.MeetsFilter(trigger.Filter) {
			return
		}
		for _, e := range list {
			effects = append(effects, c.scaled(name, e, tokens))
		}
	}
	for _, f := range c.AdversaryFeatures {
		add(f.Name, f.Passive, f.Trigger, f.UsesTokens, f.TokensHeld, f.Effects)
	}
	for _, a := range c.DomainAbilities {
		add(a.Name, a.Passive, a.Trigger, a.UsesTokens, a.TokensHeld, a.Effects)
	}
	return effects
}

// passiveParts lists what each standing effect of a category contributes, named after its
// feature. An effect with a damage type only counts for that type; a nil roller skips dice.
func (c *Combatant) passiveParts(category FeatureEffectCategory, damageType DamageType, roller DiceRoller) []LogPart {
	var parts []LogPart
	for _, effect := range c.passiveEffects() {
		if effect.Category != category {
			continue
		}
		if effect.DamageType == "" || ParseDamageType(effect.DamageType) == damageType {
			parts = append(parts, LogPart{effect.source, effect.amount(roller)})
		}
	}
	return parts
}

func (c *Combatant) sumPassive(category FeatureEffectCategory, roller DiceRoller) int {
	return SumParts(c.passiveParts(category, DamagePhysical, roller))
}

// DamageBonusParts is what each standing damage_bonus effect adds to a damage roll of this type
func (c *Combatant) DamageBonusParts(damageType DamageType, roller DiceRoller) []LogPart {
	return c.passiveParts(FeatureEffectDamageBonus, damageType, roller)
}

// DamageBonus is the total standing damage_bonus effects add to a damage roll of this type
func (c *Combatant) DamageBonus(damageType DamageType, roller DiceRoller) int {
	return SumParts(c.DamageBonusParts(damageType, roller))
}

// AttackRollParts is what each standing attack_bonus effect adds to attack and action rolls
func (c *Combatant) AttackRollParts(roller DiceRoller) []LogPart {
	return c.passiveParts(FeatureEffectAttackBonus, DamagePhysical, roller)
}

// AttackRollBonus is the total standing attack_bonus effects add to attack and action rolls
func (c *Combatant) AttackRollBonus(roller DiceRoller) int {
	return SumParts(c.AttackRollParts(roller))
}

// RiderAmount rolls one effect of a reactive passive for its owner: Magnitude and dice, scaled
func (c *Combatant) RiderAmount(effect FeatureEffect, tokens int, roller DiceRoller) int {
	return c.scaled("", effect, tokens).amount(roller)
}

// scaled binds an effect to its owner: the scale value, and for StandardAttack one weapon die
// ("+1 bonus to your Proficiency" on a damage roll)
func (c *Combatant) scaled(source string, effect FeatureEffect, tokens int) standingEffect {
	if effect.StandardAttack {
		effect.DiceFormula = c.weaponDie()
	}
	return standingEffect{effect, source, tokens, c.ScaleValue(effect.ScaleBy)}
}

// weaponDie is a single die of the combatant's attack, such as "1d10" for "2d10+3"
func (c *Combatant) weaponDie() string {
	notation, err := ParseDiceNotation(c.DamageDice)
	if err != nil {
		return "1d6"
	}
	for _, term := range notation.Terms {
		if term.Count > 0 {
			return (&DiceNotation{Terms: []DiceTerm{{Count: 1, Size: term.Size}}}).String()
		}
	}
	return "1d6"
}

// MarkHP marks Hit Points outright: no thresholds, armor or resistance
func (c *Combatant) MarkHP(amount int) {
	c.markHP(amount)
}

// ScaleDice multiplies the first dice count of a formula by the value ScaleBy stands for
func (c *Combatant) ScaleDice(formula, by string) string {
	if by == "" {
		return formula
	}
	notation, err := ParseDiceNotation(formula)
	if err != nil {
		return formula
	}
	for i := range notation.Terms {
		if notation.Terms[i].Count > 0 {
			notation.Terms[i].Count *= max(c.ScaleValue(by), 1)
			break
		}
	}
	return notation.String()
}

// ImmuneTo reports whether a standing condition_immunity effect keeps the condition off
func (c *Combatant) ImmuneTo(condType ConditionType) bool {
	for _, effect := range c.passiveEffects() {
		if effect.Category == FeatureEffectConditionImmunity && slices.Contains(effect.Conditions, condType) {
			return true
		}
	}
	return false
}

// DropBlockedConditions ends conditions the combatant has just become immune to
func (c *Combatant) DropBlockedConditions() {
	c.Conditions = slices.DeleteFunc(c.Conditions, func(cond *ActiveCondition) bool { return c.ImmuneTo(cond.Type) })
}

// stepTokenDie advances a token_on_damage die; one that would pass its maximum is removed
func stepTokenDie(effects []FeatureEffect, usesTokens bool, held *int, limit int) (ended bool) {
	if !usesTokens || *held <= 0 {
		return false
	}
	for _, effect := range effects {
		if effect.Category != FeatureEffectTokenOnDamage {
			continue
		}
		// A negative step spends the tokens the damage roll used
		*held = max(*held+effect.Magnitude, 0)
		if limit > 0 && *held > limit {
			*held = 0
			return true
		}
	}
	return false
}

// StepTokensOnDamage advances the combatant's token_on_damage dice after a damage roll that
// marked a Hit Point, and returns the names of the features whose die ran out.
func (c *Combatant) StepTokensOnDamage() []string {
	var ended []string
	for i := range c.AdversaryFeatures {
		f := &c.AdversaryFeatures[i]
		if stepTokenDie(f.Effects, f.UsesTokens, &f.TokensHeld, f.TokensMax) {
			ended = append(ended, f.Name)
		}
	}
	for i := range c.DomainAbilities {
		a := &c.DomainAbilities[i]
		if stepTokenDie(a.Effects, a.UsesTokens, &a.TokensHeld, a.TokensMax) {
			ended = append(ended, a.Name)
		}
	}
	// Full Stress makes a PC Vulnerable again once nothing blocks it
	if len(ended) > 0 && c.IsFullyStressed {
		c.AddCondition(&ActiveCondition{Type: ConditionVulnerable, Duration: DurationUntilCleared})
	}
	return ended
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

// ClearArmorSlots unmarks up to amount Armor Slots, never past the Armor Score. Returns the slots cleared.
func (c *Combatant) ClearArmorSlots(amount int) int {
	cleared := min(amount, MarkedArmor(c))
	c.ArmorSlots += cleared
	return cleared
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
	if c.HasCondition(condition.Type) || c.ImmuneTo(condition.Type) {
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

// GetDefense returns appropriate defense stat, with standing Evasion bonuses and penalties applied
func (c *Combatant) GetDefense() int {
	base := c.Evasion
	if c.IsAdversary {
		base = c.Difficulty
	}
	return base + c.sumPassive(FeatureEffectEvasionBonus, nil) - c.DefensePenalty
}

// GetAttackBonus is what the combatant adds to a weapon attack roll
func (c *Combatant) GetAttackBonus() int {
	return c.RollModifier("")
}

// rollTrait resolves a roll's trait: "" is the weapon trait, "spellcast" the Spellcast trait
func (c *Combatant) rollTrait(trait string) string {
	switch trait {
	case "":
		return c.WeaponTrait
	case "spellcast":
		return c.SpellcastTrait
	}
	return trait
}

// RollModifier is what the combatant adds to a roll made with the named trait.
// Adversaries add their ATK to every roll.
func (c *Combatant) RollModifier(trait string) int {
	if c.IsAdversary {
		return c.ATK
	}
	return c.Traits.Modifier(c.rollTrait(trait))
}

// RollLabel names that modifier for the log: "Strength", "Spellcast (Knowledge)", "ATK"
func (c *Combatant) RollLabel(trait string) string {
	if c.IsAdversary {
		return "ATK"
	}
	name := c.rollTrait(trait)
	if name == "" {
		return "no trait"
	}
	label := strings.ToUpper(name[:1]) + name[1:]
	if trait == "spellcast" {
		return "Spellcast (" + label + ")"
	}
	return label
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
	clone.DefensePenalty = 0
	clone.ArmorMax = max(template.ArmorMax, template.ArmorSlots)

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
	return c.ScaleToProficiency(formula)
}

// ScaleToProficiency rolls a PC's Proficiency in place of the first dice count ("d8+2 using your Proficiency")
func (c *Combatant) ScaleToProficiency(formula string) string {
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
