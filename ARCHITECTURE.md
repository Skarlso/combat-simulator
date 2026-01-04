# Daggerheart Combat Simulator - Architecture

## Project Structure

```
daggerheart-combat-simulator/
├── cmd/
│   └── server/          # WASM server entry point
├── internal/
│   ├── models/          # Core data structures
│   ├── engine/          # Combat resolution engine
│   ├── strategy/        # AI decision strategies
│   ├── simulation/      # Monte Carlo runner
│   └── export/          # Report generation (CSV/PDF)
├── ui/
│   ├── components/      # Reusable UI components
│   ├── pages/           # Main app pages
│   └── app.go           # go-app entry point
└── web/                 # Static assets (CSS, icons)
```

## Data Flow

```
User Input (UI Forms)
    ↓
Character/Adversary Models
    ↓
Simulation Configuration
    ↓
Monte Carlo Runner
    ↓
    ├→ Combat Engine (per simulation)
    │   ├→ Turn Resolution
    │   ├→ Ability System
    │   └→ AI Decision Strategies
    ↓
Aggregated Results
    ↓
Visualization + Export
```

## Core Packages

### `internal/models`

Character sheets, abilities, combat state.

**Key types:**
- `Character` - Base stats, abilities, resources
- `Ability` - Triggers, conditions, effects
- `CombatState` - Active encounter state (HP, stress, conditions)
- `Condition` - Status effects (burning, cloaked, vulnerable)

### `internal/engine`

Combat resolution logic. Stateless where possible.

**Key interfaces:**
- `DiceRoller` - Duality dice, damage dice
- `CombatResolver` - Attack resolution, damage calculation
- `EffectApplier` - Apply ability effects to combat state
- `SpotlightSelector` - Choose next actor based on spotlight tokens

**Responsibilities:**
- Resolve duality rolls (d12 Hope vs d12 Fear)
- Apply damage thresholds (Minor/Major/Severe)
- Track Hope/Fear tokens
- Manage stress and HP
- Calculate evasion checks
- Manage spotlight token accumulation

### `internal/strategy`

AI decision-making for resource usage and action selection.

**Key interfaces:**
- `DecisionStrategy` - Choose actions each turn
- `FearStrategy` - When to spend Fear tokens
- `HopeStrategy` - When to spend Hope tokens
- `ConditionStrategy` - When to clear/ignore conditions

**Implementations:**
- Greedy: Use resources aggressively
- Conservative: Hoard resources for emergencies
- Desperate: Context-aware (use when low HP)
- Random: Stochastic decisions

### `internal/simulation`

Monte Carlo orchestration.

**Key types:**
- `SimulationConfig` - Number of runs, strategies, participants, spotlight threshold
- `SimulationRunner` - Execute N simulations
- `SimulationResult` - Win/loss rates, average duration, key statistics

**Responsibilities:**
- Run N independent simulations
- Aggregate results
- Track statistics (win rate, average rounds, resource usage)

### `internal/export`

Report generation.

**Key interfaces:**
- `Exporter` - Export results to format

**Implementations:**
- CSV exporter
- PDF exporter (future)

### `ui/`

go-app WASM UI components.

**Key pages:**
- Character builder
- Adversary builder
- Simulation configuration
- Results dashboard

## Core Abstractions

### Ability System

Abilities are declarative:

```go
type Ability struct {
    Name        string
    Trigger     Trigger      // When can this activate?
    Condition   Condition    // What must be true?
    Costs       ResourceCost // What does it cost?
    Effects     []Effect     // What happens?
}

type Trigger string
// Examples: on_turn_start, on_attack, on_hit, on_damaged, on_ally_downed

type Condition interface {
    Evaluate(state *CombatState, actor *Combatant) bool
}

type Effect interface {
    Apply(state *CombatState, actor *Combatant, target *Combatant) error
}
```

### Effect Types

All effects reduce to resource changes or state modifications:

- `AddDamageEffect` - Add dice to damage pool
- `HealEffect` - Restore HP
- `AddConditionEffect` - Apply status effect
- `RemoveConditionEffect` - Clear status effect
- `ModifyResourceEffect` - Change Hope/Fear/Stress/tokens
- `AddDiceEffect` - Grant advantage/disadvantage

### Combat Resolution Flow

```
1. Initialize all combatants with 0 spotlight tokens
2. Loop until one side eliminated:
   a. Select next actor using SpotlightSelector (based on tokens)
   b. Grant +1 token to all non-acting combatants
   c. Reset acting combatant's tokens to 0
   d. Check active conditions on actor (burning, etc.)
   e. AI chooses action (attack, ability, clear condition)
   f. Resolve action:
      - Roll duality dice (d12 Hope vs d12 Fear)
      - Determine outcome (Hope/Fear with Spirit)
      - Apply effects
      - Handle damage thresholds
   g. Check for defeat conditions
```

### Decision Strategy

AI picks actions each turn:

```go
type DecisionStrategy interface {
    ChooseAction(state *CombatState, actor *Combatant) Action
}

type Action struct {
    Type    ActionType  // Attack, UseAbility, ClearCondition, Pass
    Target  *Combatant  // Who is targeted (if applicable)
    Ability *Ability    // Which ability (if UseAbility)
}
```

## Key Design Decisions

### 1. Spotlight Token System

**Problem:** Daggerheart uses narrative spotlight selection, not rigid turn order. How to simulate this?

**Solution:** Token accumulation system with configurable forced threshold.

**Mechanics:**
- Each combatant tracks spotlight tokens (starts at 0)
- After each action, all non-acting combatants gain +1 token
- Acting combatant resets to 0 tokens

**Selection algorithm (hybrid):**
```go
type SpotlightSelector interface {
    ChooseNext(combatants []*Combatant, threshold int) *Combatant
}

// Hybrid implementation:
// 1. If any combatant has tokens >= threshold, randomly pick from those
// 2. Else, weighted random based on token count
// 3. Base probability + (tokens * weight_factor)
```

**Configuration:**
- `spotlight_threshold` - Tokens required to force selection (user configurable, default: 3)
- `weight_factor` - How much tokens bias selection (default: 0.2)

**Example:**
```
Round 1: Rogue acts (tokens: 0) → Fighter +1 (1), Wizard +1 (1)
Round 2: Fighter acts (tokens: 1) → Rogue +1 (1), Wizard +1 (2)
Round 3: Wizard acts (tokens: 2) → Rogue +1 (2), Fighter +1 (2)
Round 4: Any could act (weighted toward higher tokens)
Round 5: If Rogue still at 3+ tokens, must act
```

**Benefits:**
- Preserves narrative randomness
- Prevents spotlight starvation
- User controls balance via threshold

### 2. AoE Targeting

**Problem:** Without positioning, AoE might always hit everyone.

**Solution:**
- AoE abilities specify `max_targets`
- Actual hits = random(1, min(max_targets, enemy_count))
- Penalty for repeated AoE spam (diminishing returns)
- AI avoids AoE when low enemy count or friendly fire risk

### 2. Condition Duration

**Problem:** Narrative timing is fuzzy.

**Solution:**
- Conditions have duration: `instant`, `end_of_turn`, `start_of_next_turn`, `until_cleared`
- AI uses strategy to decide when to clear
- Probability-based: high damage conditions → more likely to clear

### 3. Damage Types and Resistances

```go
type DamageType int
const (
    Physical DamageType = iota
    Magic
    Fire
    Frost
    // etc
)

type Resistance struct {
    Type   DamageType
    Amount int  // Flat reduction
}
```

Applied during damage calculation.

### 4. Action Economy

**Simplification:** Ignore action/movement costs in simulation. Assume everything is in range and actors can use one ability per turn.

**Rationale:** Movement and positioning are narrative. Focus on combat math.

## Daggerheart Core Mechanics (Baked In)

### Duality Dice System

- Roll d12 Hope + d12 Fear + modifiers
- Highest die wins (with Spirit)
- Hope wins: PC benefits (damage, success)
- Fear wins: GM benefits (Fear token, complications)

### Damage Thresholds

Based on proficiency tier:
- Minor: 1-3 damage
- Major: 4-6 damage
- Severe: 7+ damage

### Hope/Fear Tokens

- Hope: Players spend for advantages
- Fear: GM spends for enemy abilities or complications
- Pool sizes configurable

### Stress

- Max stress = 6 (typically)
- Fully stressed = disadvantage on rolls
- Clear stress via downtime or abilities

### Hit Points

- HP = character/adversary specific
- 0 HP = defeated

## Next Steps

1. Implement core models (`internal/models`)
2. Build combat engine (`internal/engine`)
3. Create basic decision strategies (`internal/strategy`)
4. Wire up Monte Carlo runner (`internal/simulation`)
5. Build WASM UI (`ui/`)
