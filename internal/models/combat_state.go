package models

import "strings"

// CombatState tracks the current state of an encounter
type CombatState struct {
	// Participants
	Players []*Combatant
	Enemies []*Combatant

	// Fear is the GM's pool; Hope is tracked per PC on the Combatant
	FearPool int

	// Turn tracking
	RoundNumber            int
	TurnNumber             int
	CurrentActor           *Combatant
	ActionHistory          []ActionRecord
	ConsecutivePlayerTurns int

	// Spotlights each adversary has used this GM turn, keyed by ID
	gmSpotlights map[string]int

	// Configuration
	SpotlightThreshold int

	// Summoning support
	SummonCounter  int
	SummonedCounts map[string]int

	// Allies granted a spotlight by a feature; they act in the current or next GM turn
	PendingSpotlights []PendingSpotlight
}

// PendingSpotlight is one queued spotlight handed out by an ally_spotlight or summon effect
type PendingSpotlight struct {
	Combatant  *Combatant
	HalfDamage bool
}

// ActionRecord logs what happened during combat
type ActionRecord struct {
	Round            int
	Turn             int
	Actor            string
	ActionType       ActionType
	Target           string
	FeatureName      string // Name of feature/ability used (if any)
	IsFeature        bool   // True if a feature was activated
	Result           string
	DamageRoll       int    // Raw damage rolled (total)
	ProcessedDamage  int    // Damage after resistance/immunity/vulnerability
	DamageDice       []int  // Individual dice values
	DamageBonus      int    // Flat bonus
	HPDamage         int    // Actual HP damage dealt (1/2/3)
	Threshold        string // "Minor", "Major", "Severe"
	TargetHPBefore   int    // Target's HP before this action
	TargetHPAfter    int    // Target's HP after this action
	ArmorUsed        bool   // Did target use an armor slot?
	ArmorSlotsBefore int    // Target's armor slots before
	ArmorSlotsAfter  int    // Target's armor slots after
	HopeDie          int    // Hope die result (d12) - 0 for enemies
	FearDie          int    // Fear die result (d12) - 0 for enemies
	D20Roll          int    // Enemy d20 roll - 0 for players
	TotalRoll        int    // Hope + Fear + modifiers (or d20 + ATK for enemies)
	TargetDefense    int    // Defense value to beat (or DC for abilities)
	SpotlightShift   bool   // Did spotlight change hands?
	IsCritical       bool   // Was this a critical success?
	HopeGenerated    int    // Hope tokens generated this action
	FearGenerated    int    // Fear tokens generated this action
	HopePoolAfter    int    // Party's total Hope after this action
	FearPoolAfter    int    // Fear pool after this action
}

// NewCombatState initializes combat
func NewCombatState(players, enemies []*Combatant, startingHope, fearPool, spotlightThreshold int) *CombatState {
	for _, p := range players {
		p.Hope = min(startingHope, MaxHope)
	}
	activeEnemies := make([]*Combatant, 0, len(enemies))
	for _, e := range enemies {
		if !e.IsSummonTemplate {
			activeEnemies = append(activeEnemies, e)
		}
	}
	return &CombatState{
		Players:            players,
		Enemies:            activeEnemies,
		FearPool:           fearPool,
		SpotlightThreshold: spotlightThreshold,
		ActionHistory:      make([]ActionRecord, 0),
		SummonedCounts:     make(map[string]int),
	}
}

func alive(combatants []*Combatant) []*Combatant {
	out := make([]*Combatant, 0, len(combatants))
	for _, c := range combatants {
		if c.IsAlive() {
			out = append(out, c)
		}
	}
	return out
}

// GetAlivePlayers returns living player characters
func (cs *CombatState) GetAlivePlayers() []*Combatant {
	return alive(cs.Players)
}

// GetAliveEnemies returns living enemies
func (cs *CombatState) GetAliveEnemies() []*Combatant {
	return alive(cs.Enemies)
}

// GetAlliesOf returns living allies of a combatant
func (cs *CombatState) GetAlliesOf(combatant *Combatant) []*Combatant {
	if combatant.Side == SidePlayer {
		return cs.GetAlivePlayers()
	}
	return cs.GetAliveEnemies()
}

// GetEnemiesOf returns living enemies of a combatant
func (cs *CombatState) GetEnemiesOf(combatant *Combatant) []*Combatant {
	if combatant.Side == SidePlayer {
		return cs.GetAliveEnemies()
	}
	return cs.GetAlivePlayers()
}

// IsPlayerVictory returns true if all enemies defeated
func (cs *CombatState) IsPlayerVictory() bool {
	return len(cs.GetAliveEnemies()) == 0 && len(cs.GetAlivePlayers()) > 0
}

// IsEnemyVictory returns true if all players defeated
func (cs *CombatState) IsEnemyVictory() bool {
	return len(cs.GetAlivePlayers()) == 0 && len(cs.GetAliveEnemies()) > 0
}

// IsCombatOver returns true if either side is eliminated (both at once is a draw)
func (cs *CombatState) IsCombatOver() bool {
	return len(cs.GetAlivePlayers()) == 0 || len(cs.GetAliveEnemies()) == 0
}

const MaxFearPool = 12

// TotalHope sums the Hope held by all PCs
func (cs *CombatState) TotalHope() int {
	total := 0
	for _, p := range cs.Players {
		total += p.Hope
	}
	return total
}

func (cs *CombatState) AddFear(amount int) {
	cs.FearPool += amount
	if cs.FearPool > MaxFearPool {
		cs.FearPool = MaxFearPool
	}
}

// SpendFear deducts from Fear pool
func (cs *CombatState) SpendFear(amount int) bool {
	if cs.FearPool >= amount {
		cs.FearPool -= amount
		return true
	}
	return false
}

type ActionParams struct {
	Actor           *Combatant
	ActionType      ActionType
	Target          *Combatant
	FeatureName     string
	IsFeature       bool
	Result          string
	DamageRoll      int
	ProcessedDamage int
	DamageDice      []int
	DamageBonus     int
	HPDamage        int
	Threshold       DamageThreshold
	HopeDie         int
	FearDie         int
	D20Roll         int
	TotalRoll       int
	TargetDefense   int
	SpotlightShift  bool
	HopeGenerated   int
	FearGenerated   int
	DamageResult    *DamageResult
}

func (cs *CombatState) RecordAction(p ActionParams) {
	isCritical := p.HopeDie > 0 && p.FearDie > 0 && p.HopeDie == p.FearDie

	record := ActionRecord{
		Round:           cs.RoundNumber,
		Turn:            cs.TurnNumber,
		Actor:           p.Actor.Name,
		ActionType:      p.ActionType,
		FeatureName:     p.FeatureName,
		IsFeature:       p.IsFeature,
		Result:          p.Result,
		DamageRoll:      p.DamageRoll,
		ProcessedDamage: p.ProcessedDamage,
		DamageDice:      p.DamageDice,
		DamageBonus:     p.DamageBonus,
		HPDamage:        p.HPDamage,
		HopeDie:         p.HopeDie,
		FearDie:         p.FearDie,
		D20Roll:         p.D20Roll,
		TotalRoll:       p.TotalRoll,
		TargetDefense:   p.TargetDefense,
		SpotlightShift:  p.SpotlightShift,
		IsCritical:      isCritical,
		HopeGenerated:   p.HopeGenerated,
		FearGenerated:   p.FearGenerated,
		HopePoolAfter:   cs.TotalHope(),
		FearPoolAfter:   cs.FearPool,
	}

	record.Threshold = p.Threshold.String()

	if p.Target != nil {
		record.Target = p.Target.Name
		record.TargetHPAfter = p.Target.HP
		record.TargetDefense = p.Target.GetDefense()

		if p.DamageResult != nil {
			record.TargetHPBefore = p.DamageResult.HPBefore
			record.TargetHPAfter = p.DamageResult.HPAfter
			record.ArmorUsed = p.DamageResult.ArmorUsed
			record.ArmorSlotsBefore = p.DamageResult.ArmorSlotsBefore
			record.ArmorSlotsAfter = p.DamageResult.ArmorSlotsAfter
		}
	}

	cs.ActionHistory = append(cs.ActionHistory, record)
}

func (cs *CombatState) BeginGMTurn() {
	cs.gmSpotlights = make(map[string]int)
}

func (cs *CombatState) EndGMTurn() {
	cs.gmSpotlights = nil
}

func (cs *CombatState) RecordAdversarySpotlight(id string) {
	if cs.gmSpotlights == nil {
		cs.gmSpotlights = make(map[string]int)
	}
	cs.gmSpotlights[id]++
}

// AdversarySpotlightCount is how often an adversary has been spotlighted this GM turn
func (cs *CombatState) AdversarySpotlightCount(id string) int {
	return cs.gmSpotlights[id]
}

// QueueSpotlight hands a combatant a spotlight to use once the current action resolves
func (cs *CombatState) QueueSpotlight(c *Combatant, halfDamage bool) {
	cs.PendingSpotlights = append(cs.PendingSpotlights, PendingSpotlight{Combatant: c, HalfDamage: halfDamage})
}

// PopPendingSpotlight returns the next queued spotlight, if any
func (cs *CombatState) PopPendingSpotlight() (PendingSpotlight, bool) {
	if len(cs.PendingSpotlights) == 0 {
		return PendingSpotlight{}, false
	}
	next := cs.PendingSpotlights[0]
	cs.PendingSpotlights = cs.PendingSpotlights[1:]
	return next, true
}

func (cs *CombatState) isSpotlightQueued(c *Combatant) bool {
	for _, p := range cs.PendingSpotlights {
		if p.Combatant == c {
			return true
		}
	}
	return false
}

// EligibleSpotlightAllies lists the actor's living allies an ally_spotlight effect may pick:
// not the actor, not already queued, not already at their spotlight limit this GM turn.
func (cs *CombatState) EligibleSpotlightAllies(actor *Combatant, effect FeatureEffect) []*Combatant {
	var allies []*Combatant
	for _, c := range cs.GetAlliesOf(actor) {
		if c == actor || !c.IsAlive() || cs.isSpotlightQueued(c) {
			continue
		}
		if cs.AdversarySpotlightCount(c.ID) >= c.MaxSpotlightsPerGMTurn() {
			continue
		}
		if effect.AllyName != "" && !matchesAllyName(c.Name, effect.AllyName) {
			continue
		}
		allies = append(allies, c)
	}
	return allies
}

// Recipients picks who a self, ally or allies effect lands on. need scores how much a
// combatant would gain from it; an "ally" effect goes to the needful ally, or to nobody.
func (cs *CombatState) Recipients(actor *Combatant, targetType FeatureTargetType, need func(*Combatant) int) []*Combatant {
	switch targetType {
	case FeatureTargetAllies:
		return cs.GetAlliesOf(actor)
	case FeatureTargetAlly:
		var best *Combatant
		for _, c := range cs.GetAlliesOf(actor) {
			if c == actor || need(c) <= 0 {
				continue
			}
			if best == nil || need(c) > need(best) {
				best = c
			}
		}
		if best == nil {
			return nil
		}
		return []*Combatant{best}
	default:
		return []*Combatant{actor}
	}
}

// MissingHP and StressHeld are the need scores for heals and stress clears
func MissingHP(c *Combatant) int  { return c.HPMax - c.HP }
func StressHeld(c *Combatant) int { return c.Stress }

// matchesAllyName accepts the stat block name and the "Name #n" form summons get
func matchesAllyName(name, want string) bool {
	return name == want || strings.HasPrefix(name, want+" ")
}

// SummonCombatant adds a new enemy combatant mid-combat
func (cs *CombatState) SummonCombatant(summoned *Combatant) {
	cs.Enemies = append(cs.Enemies, summoned)
}
