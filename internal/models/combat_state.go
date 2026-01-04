package models

// CombatState tracks the current state of an encounter
type CombatState struct {
	// Participants
	Players []*Combatant
	Enemies []*Combatant

	// Resource pools
	HopePool int
	FearPool int

	// Turn tracking
	RoundNumber   int
	TurnNumber    int
	CurrentActor  *Combatant
	ActionHistory []ActionRecord

	// Configuration
	SpotlightThreshold int
}

// ActionRecord logs what happened during combat
type ActionRecord struct {
	Round      int
	Turn       int
	Actor      string
	ActionType ActionType
	Target     string
	Ability    string
	Result     string
	Damage     int
}

// NewCombatState initializes combat
func NewCombatState(players, enemies []*Combatant, hopePool, fearPool, spotlightThreshold int) *CombatState {
	return &CombatState{
		Players:            players,
		Enemies:            enemies,
		HopePool:           hopePool,
		FearPool:           fearPool,
		RoundNumber:        0,
		TurnNumber:         0,
		SpotlightThreshold: spotlightThreshold,
		ActionHistory:      make([]ActionRecord, 0),
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

// IsCombatOver returns true if one side is eliminated
func (cs *CombatState) IsCombatOver() bool {
	return cs.IsPlayerVictory() || cs.IsEnemyVictory()
}

// AddHope adds to Hope pool
func (cs *CombatState) AddHope(amount int) {
	cs.HopePool += amount
}

// SpendHope deducts from Hope pool
func (cs *CombatState) SpendHope(amount int) bool {
	if cs.HopePool >= amount {
		cs.HopePool -= amount
		return true
	}
	return false
}

// AddFear adds to Fear pool
func (cs *CombatState) AddFear(amount int) {
	cs.FearPool += amount
}

// SpendFear deducts from Fear pool
func (cs *CombatState) SpendFear(amount int) bool {
	if cs.FearPool >= amount {
		cs.FearPool -= amount
		return true
	}
	return false
}

// RecordAction logs an action taken during combat
func (cs *CombatState) RecordAction(actor *Combatant, actionType ActionType, target *Combatant, ability *Ability, result string, damage int) {
	record := ActionRecord{
		Round:      cs.RoundNumber,
		Turn:       cs.TurnNumber,
		Actor:      actor.Name,
		ActionType: actionType,
		Result:     result,
		Damage:     damage,
	}

	if target != nil {
		record.Target = target.Name
	}

	if ability != nil {
		record.Ability = ability.Name
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
