package models

type GMTurnState struct {
	Active         bool
	SpotlightedIDs map[string]int
}

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

	// GM turn state
	GMTurn GMTurnState

	// Configuration
	SpotlightThreshold int

	// Summoning support
	SummonCounter  int
	SummonedCounts map[string]int
}

// ActionRecord logs what happened during combat
type ActionRecord struct {
	Round            int
	Turn             int
	Actor            string
	ActionType       ActionType
	Target           string
	Ability          string
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
		Players:                players,
		Enemies:                activeEnemies,
		FearPool:               fearPool,
		RoundNumber:            0,
		TurnNumber:             0,
		ConsecutivePlayerTurns: 0,
		SpotlightThreshold:     spotlightThreshold,
		ActionHistory:          make([]ActionRecord, 0),
		SummonedCounts:         make(map[string]int),
	}
}

// GetAllCombatants returns all participants
func (cs *CombatState) GetAllCombatants() []*Combatant {
	all := make([]*Combatant, 0, len(cs.Players)+len(cs.Enemies))
	all = append(all, cs.Players...)
	all = append(all, cs.Enemies...)
	return all
}

// GetAliveCombatants returns only living combatants
func (cs *CombatState) GetAliveCombatants() []*Combatant {
	alive := make([]*Combatant, 0)
	for _, c := range cs.GetAllCombatants() {
		if c.IsAlive() {
			alive = append(alive, c)
		}
	}
	return alive
}

// GetAlivePlayers returns living player characters
func (cs *CombatState) GetAlivePlayers() []*Combatant {
	alive := make([]*Combatant, 0)
	for _, p := range cs.Players {
		if p.IsAlive() {
			alive = append(alive, p)
		}
	}
	return alive
}

// GetAliveEnemies returns living enemies
func (cs *CombatState) GetAliveEnemies() []*Combatant {
	alive := make([]*Combatant, 0)
	for _, e := range cs.Enemies {
		if e.IsAlive() {
			alive = append(alive, e)
		}
	}
	return alive
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
	AbilityName     string
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
		Ability:         p.AbilityName,
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

// IncrementTurn advances turn counter
func (cs *CombatState) IncrementTurn() {
	cs.TurnNumber++
}

// IncrementRound advances round counter
func (cs *CombatState) IncrementRound() {
	cs.RoundNumber++
}

func (cs *CombatState) BeginGMTurn() {
	cs.GMTurn = GMTurnState{
		Active:         true,
		SpotlightedIDs: make(map[string]int),
	}
}

func (cs *CombatState) EndGMTurn() {
	cs.GMTurn.Active = false
	cs.GMTurn.SpotlightedIDs = nil
}

func (cs *CombatState) RecordAdversarySpotlight(id string) {
	if cs.GMTurn.SpotlightedIDs == nil {
		cs.GMTurn.SpotlightedIDs = make(map[string]int)
	}
	cs.GMTurn.SpotlightedIDs[id]++
}

func (cs *CombatState) AdversarySpotlightCount(id string) int {
	if cs.GMTurn.SpotlightedIDs == nil {
		return 0
	}
	return cs.GMTurn.SpotlightedIDs[id]
}

// SummonCombatant adds a new enemy combatant mid-combat
func (cs *CombatState) SummonCombatant(summoned *Combatant) {
	cs.Enemies = append(cs.Enemies, summoned)
}
