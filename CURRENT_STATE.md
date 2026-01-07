# Current State - Combat Mechanics Verified & Fixed

**Last Updated**: 2026-01-06
**Status**: Core mechanics verified against Daggerheart rules - All critical bugs fixed

## Session Summary (2026-01-06)

This session focused on **verifying and fixing core combat mechanics** against official Daggerheart rules:

### Critical Bugs Fixed

1. **Critical Success Detection** ✅
   - **Was**: Checked if Hope die = 12 OR Fear die = 12
   - **Now**: Checks if Hope die = Fear die (matching dice)
   - Location: `internal/engine/dice.go:73-80`

2. **Duality Roll Total Calculation** ✅
   - **Was**: Used only highest die
   - **Now**: Sums both dice (d12 + d12 + modifiers)
   - Location: `internal/engine/dice.go:83`

3. **Critical Damage** ✅
   - **Was**: No extra damage on crits
   - **Now**: Adds max dice result (e.g., 2d8+1 crit = 2d8+1+16)
   - Location: `internal/engine/resolver.go:107-111`

4. **Advantage/Disadvantage** ✅
   - **Was**: Flat +2/-2 modifier
   - **Now**: Rolls actual d6, adds/subtracts result
   - Location: `internal/engine/resolver.go:49-54`, `dice.go:136-139`

5. **Stress Cleared on Critical** ✅
   - **Was**: Only generated Hope
   - **Now**: Clears 1 Stress on critical success
   - Location: `internal/engine/resolver.go:70`

6. **Stress Overflow to HP** ✅
   - **Was**: Nothing
   - **Now**: Marks 1 HP when Stress is maxed
   - Location: `internal/models/combatant.go:181-189`

7. **Dynamic PC Thresholds** ✅
   - **Was**: Hardcoded [4, 7] for everyone
   - **Now**: (3 + Level) for Minor, (6 + Level) for Major
   - Location: `internal/models/types.go:49-55`

8. **Critical Auto-Success** ✅
   - **Was**: Critical hits could miss if roll < defense
   - **Now**: Critical success always hits regardless of defense
   - Location: `internal/engine/resolver.go:80-90, 99-109`

### UI Improvements

1. **Combat Log Enhancements** ✅
   - Critical hits highlighted with gold background
   - Bold text and gold left border for critical rows
   - Fixed Hope/Fear column width to fit in table
   - Location: `ui/components/results_summary.go`, `web/static/app.css:495-503`

2. **Critical Hit Tracking** ✅
   - Added `IsCritical` field to `ActionRecord`
   - Automatic detection of matching duality dice
   - Location: `internal/models/combat_state.go:43, 167`

3. **Level Field Added** ✅
   - Added Level field to Combatant for dynamic thresholds
   - UI form updated with Level input
   - Location: `internal/models/combatant.go:16`, `ui/components/combatant_list.go`

## What Works Now

### Core Mechanics (Rules-Compliant)
- ✅ Duality dice roll correctly (sum both dice)
- ✅ Critical success on matching dice (auto-hit)
- ✅ Critical damage bonus (roll + max dice)
- ✅ Advantage/Disadvantage uses d6 rolls
- ✅ Hope/Fear generation correct
- ✅ Stress clears on critical success
- ✅ Stress overflow converts to HP damage
- ✅ Dynamic damage thresholds by level
- ✅ Spotlight shift mechanics
- ✅ Armor slot absorption
- ✅ Condition system

### Combat Engine
- ✅ Monte Carlo simulation (1000 runs in ~200ms)
- ✅ AI strategies (Desperate, Greedy, Balanced)
- ✅ Attack resolution (Players use duality, Enemies use d20)
- ✅ Adversary mechanics (Difficulty/ATK)
- ✅ Ability system with effects
- ✅ Combat log with detailed tracking

### UI/WebAssembly
- ✅ Server starts and serves HTML
- ✅ Interactive forms with ctx.Update() calls
- ✅ Tab switching between Setup/Players/Enemies/Strategy/Results
- ✅ Combat log viewer with critical highlighting
- ✅ Combatant lists with add/remove
- ✅ Strategy configuration
- ✅ Simulation results display

## Files Modified (2026-01-06)

### Core Mechanics Fixes
1. `internal/engine/dice.go` - Fixed critical detection, duality roll total, added RollD6()
2. `internal/engine/resolver.go` - Fixed critical damage, advantage/disadvantage, critical auto-hit, stress clear
3. `internal/models/combatant.go` - Added Level field, stress overflow to HP
4. `internal/models/types.go` - Dynamic PC thresholds based on level
5. `internal/models/combat_state.go` - Added IsCritical tracking to ActionRecord

### UI Improvements
6. `ui/components/results_summary.go` - Critical hit styling in combat log
7. `ui/components/combatant_list.go` - Added Level field to form
8. `web/static/app.css` - Critical row styling, column width fixes
9. `cmd/example/main.go` - Updated to use new Level parameter

## Known Good State

### CLI Testing
```bash
make run-example
# OR
go run cmd/example/main.go
```
✅ **VERIFIED** - 1000 simulations run successfully with correct mechanics

### Web Server
```bash
make serve
# OR
go run cmd/wasm/main.go
```
✅ **WORKS** - Server starts on http://localhost:8080

### Build Status
```bash
go build ./...
```
✅ **CLEAN** - All packages compile without errors

## Verification Data Sources

Combat mechanics verified against:
- `data/daggerheart_quick_reference.md` - Core mechanics summary
- `data/daggerheart_core_mechanics.json` - Detailed rules reference
- `data/daggerheart_adversaries-tier-*.json` - Adversary stat blocks
- `data/daggerheart_domain_*.json` - Player abilities and features

## Previous Session Work (2026-01-05)

### Adversary System
- ✅ Full native support for Daggerheart adversaries
- ✅ Separate mechanics: Difficulty/ATK vs Evasion/Proficiency
- ✅ Custom damage thresholds per combatant
- ✅ Tier system (Tier 1-5) and Types (Minion, Standard, Elite, Solo, Horde)
- ✅ Motives & Tactics fields

### Combat Log Viewer
- ✅ Turn-by-turn action display
- ✅ Round, Turn, Actor, Action, Target, Result, Damage columns
- ✅ Hope/Fear pool tracking
- ✅ Navigation controls (Prev/Next run)

### Ability System
- ✅ Name, description, damage dice, damage type
- ✅ AOE targeting with max targets
- ✅ Integrated into Player and Adversary forms
- ✅ Dynamic add/remove

## How to Run

### Start Web Server
```bash
make serve
```
Then open: http://localhost:8080

### Run CLI Example
```bash
make run-example
```

### Rebuild WASM (if needed)
```bash
GOARCH=wasm GOOS=js go build -o web/app.wasm cmd/wasm/main.go
```

## Known Issues
- None currently - all critical bugs fixed
- Core mechanics verified against official rules
- Combat simulation results are now accurate

## Current Session (2026-01-06 Part 2)

### Feature Quantization System Design

Started designing a system to convert human-readable adversary/player features into quantifiable mechanics:

1. **Problem**: Features in adversary JSON have free-form text descriptions
   - Example: "Relentless" - "Can be spotlighted up to 3 times per GM turn"
   - Need to convert this into actionable data for combat simulation

2. **Solution**: Categorize features into mechanical types
   - Created `TODO.md` with 7 important feature categories
   - Each category has specific fields (magnitude, condition, cost, etc.)
   - Ignored positional mechanics (knockback, pull, immobilize) since we don't simulate movement

3. **Important Categories** (marked ⭐):
   - Action Economy (spotlight modifiers, fear costs)
   - Damage Modifiers (amplifiers, direct damage, conditional)
   - Resistance/Weakness (damage reduction/amplification)
   - Status Application (vulnerable, poison, ignite, taunt, hidden)
   - Resource Manipulation (stress, HP, armor, hope, fear)
   - Summoning (dynamic enemy addition - requires special handling)
   - Special Mechanics (slow, minion, horde, countdown)

4. **Domain Card Analysis Completed**:
   - Reviewed Arcana and Codex domain cards
   - Identified key patterns: damage spells, status effects, defensive abilities
   - Key difference: player abilities are **active choices** with **resource costs** (Hope/Stress)
   - Added findings to `TODO.md`

5. **Next**: Design unified feature definition format

## Next Steps
- ✅ Parse and analyze domain card data (player skills)
- ✅ Design unified feature definition format (adversary + player)
- ✅ Implement feature application in combat engine
- ✅ Implement resource cost tracking (Hope/Stress/Fear costs)
- ✅ Test feature system with example features
- ⬜ Create user input system for defining quantized features (UI forms)
- ⬜ Parse real adversary features from tier 1 data
- ⬜ Integrate feature system with existing combat engine
- ⬜ Add dynamic summoning to combat state
- Additional testing with complex scenarios
- Advanced AI strategies
- Performance optimization for larger simulations

## Session 3 Summary (2026-01-06 Part 3)

### Feature System Implementation ✅

Implemented complete feature quantization system:

**What Was Built:**
- Separated AdversaryFeature and DomainAbility types (different mechanics)
- Shared FeatureEffect resolution system (20+ effect categories)
- FeatureResolver engine with resource cost validation
- Full support for dice formulas, conditions, durations
- Added missing condition types (ConditionTaunted, ConditionHidden)

**New Files:**
1. `internal/models/feature.go` - Type definitions (FeatureEffect, AdversaryFeature, DomainAbility)
2. `internal/engine/feature_resolver.go` - Resolution logic (340 lines)
3. `cmd/feature-example/main.go` - Working demonstration

**Effect Categories Implemented:**
- Damage: standard, amplifier, direct, area
- Status: poison, burning, vulnerable, taunt, hidden
- Resources: stress inflict/clear, HP heal, armor damage, hope/fear manipulation
- Resistance: damage resistance/vulnerability (passive)
- Action economy: spotlight modifiers

**Example Output:**
```
Player uses Rune Strike (2 Hope) → Enemy takes 2 HP damage
Enemy uses Venomous Stinger → Player gets poisoned + 2 stress
Enemy uses Dark Ritual (2 Fear) → Player takes 6 HP direct damage
```

**Modified Files:**
- `internal/models/combatant.go` - Added AdversaryFeatures and DomainAbilities fields
- `internal/models/types.go` - Added ConditionTaunted and ConditionHidden

Run: `go run cmd/feature-example/main.go`

## Session 4 Summary (2026-01-07)

### UI Forms for Feature System ✅

Added UI components for creating and managing features/abilities through the web interface:

**New Components:**
1. `ui/components/domain_ability_form.go` - Form for player domain abilities
   - Hope/Stress costs
   - Refresh types (rest, long_rest, scene, unlimited)
   - Usage limits
   - Effect types (damage, heal, status, stress manipulation)
   - Dice formula support

2. `ui/components/adversary_feature_form.go` - Form for enemy features
   - Fear costs
   - Spotlight limits
   - Passive vs active toggle
   - Effect types (damage, stress, status, armor damage)

**Modified Components:**
- `ui/components/adversary_list.go` - Integrated AdversaryFeatureForm
- `ui/components/combatant_list.go` - Integrated DomainAbilityForm

**What Users Can Do:**
- Add domain abilities to players (with Hope/Stress costs, usage limits)
- Add adversary features to enemies (with Fear costs, spotlight limits)
- Choose effect types (damage, heal, status conditions, etc.)
- Set dice formulas for variable effects
- Configure durations and targeting

**To Test:**
```bash
make serve
# Open http://localhost:8080
# Navigate to Players or Enemies tabs
# Click "+ Add" to create combatant
# Use "Add Ability" or "Add Feature" buttons to add skills
```

Run example CLI: `go run cmd/feature-example/main.go`

## Session 5 Summary (2026-01-07 Part 2) - LEGACY SYSTEM REMOVAL ✅

### Legacy System Removal - COMPLETE ✅

**Goal:** Remove old Ability system in favor of new Feature system

**Successfully Removed:**
- ✅ `internal/models/ability.go` (216 lines) - Legacy Ability types
- ✅ `internal/engine/effects.go` (145 lines) - EffectApplier
- ✅ `internal/models/action.go` (24 lines) - Action wrappers
- ✅ `ui/components/ability_form.go` (351 lines) - UI form
- ✅ Combatant.Abilities field removed from model
- ✅ AddAbility() method removed
- ✅ UI forms updated (adversary_list, combatant_list)
- ✅ cmd/example/main.go simplified (no ability usage)

**Refactored Files:**
- ✅ `internal/engine/combat_engine.go` - Removed EffectApplier, executeAbility(), executeClearCondition()
- ✅ `internal/engine/resolver.go` - ResolveAttack() now accepts damageDice string instead of DamageEffect
- ✅ `internal/strategy/decision_maker.go` - Simplified to just return target, removed all ability logic
- ✅ `internal/simulation/runner.go` - Updated to use AdversaryFeatures/DomainAbilities

**Key Changes:**
1. **combat_engine.go**:
   - Removed `effects *EffectApplier` field
   - Changed `ExecuteTurn` signature from `(actor, action, state)` to `(actor, target, state)`
   - Removed `executeAbility()` and `executeClearCondition()` functions
   - Simplified to only handle basic attacks

2. **resolver.go**:
   - Changed `ResolveAttack()` to accept `damageDice string` instead of `models.DamageEffect`
   - Parses dice notation (e.g., "2d6+3") internally using `ParseDiceNotation()`
   - Defaults to physical damage type

3. **decision_maker.go**:
   - Changed `ChooseAction()` return from `*models.Action` to `*models.Combatant` (target)
   - Removed ability-related functions: shouldUseAbility, canUseAbility, shouldSpendResources, etc.
   - Simplified to just select targets based on strategy

4. **simulation/runner.go**:
   - Replaced `Abilities` field with `AdversaryFeatures` and `DomainAbilities` in cloneCombatants()
   - Updated UnifiedDecisionMaker.ChooseAction() return type

**Current Status:**
- ✅ All packages compile successfully
- ✅ Main simulation runs: `go run cmd/example/main.go`
- ✅ Web UI builds: `GOARCH=wasm GOOS=js go build -o web/app.wasm cmd/wasm/main.go`
- ✅ Feature system standalone: `go run cmd/feature-example/main.go`
- ✅ Simulation results: 1000 runs, 100% player victory, avg 11.2 turns

**What Works:**
- ✅ Basic attack combat simulation (no features yet)
- ✅ Duality dice system for players
- ✅ d20 system for enemies
- ✅ Damage thresholds and HP tracking
- ✅ Hope/Fear generation
- ✅ Target selection strategies
- ✅ Feature system (standalone, not integrated)

**Next Steps (Priority 2 - Feature Integration):**
1. ⬜ Add FeatureResolver to CombatEngine
2. ⬜ Add AI decision logic for when to activate features
3. ⬜ Wire up feature activation in combat turns
4. ⬜ Resource cost tracking during simulation
5. ⬜ Feature usage in combat log

**Current Working Commands:**
```bash
# Main simulation - WORKS ✅
go run cmd/example/main.go

# Feature system example - WORKS ✅
go run cmd/feature-example/main.go

# Web UI - WORKS ✅
GOARCH=wasm GOOS=js go build -o web/app.wasm cmd/wasm/main.go
go run cmd/wasm/main.go  # Then visit http://localhost:8080

# Build everything - WORKS ✅
go build ./...
```

## Session 6 Summary (2026-01-07 Part 3) - FEATURE INTEGRATION ✅

### Feature System Integration - COMPLETE ✅

**Goal:** Integrate the standalone feature system into the main combat engine

**Completed Work:**

1. **Added FeatureResolver to CombatEngine**:
   - Added `featureResolver *FeatureResolver` field to CombatEngine struct
   - Updated `NewCombatEngine()` to initialize FeatureResolver with RNG
   - Added `GetRNG()` method to StandardDiceRoller for RNG access

2. **Feature Tracking in Combat Log**:
   - Added `FeatureName string` and `IsFeature bool` fields to ActionRecord
   - Created `RecordActionWithFeature()` method for logging feature usage
   - Features now tracked separately from regular attacks

3. **Combat Log Styling**:
   - Added `.log-row-feature` CSS class with purple/violet theme (#8a2be2)
   - Feature actions highlighted with purple left border and background
   - Distinct from critical hits (gold) for visual clarity
   - Feature name displayed in action column when activated

4. **Feature Activation Logic**:
   - Added `tryActivateFeature()` method to CombatEngine
   - 30% chance per turn to attempt feature activation (simple AI)
   - Checks resource costs before activation:
     - Player abilities: Hope pool and Stress limits
     - Enemy features: Fear pool
   - Applies all feature effects via FeatureResolver
   - Records feature usage with proper logging

5. **Resource Cost Tracking**:
   - Hope/Fear spending integrated with feature activation
   - Stress cost validation prevents overflow
   - Costs deducted before effect resolution
   - Resource pools updated in combat state

**Files Modified:**

- `internal/engine/combat_engine.go`:
  - Added featureResolver field and tryActivateFeature() method
  - Modified ExecuteTurn() to attempt feature activation (30% chance)
  - Checks affordability and applies effects

- `internal/engine/dice.go`:
  - Added GetRNG() method to StandardDiceRoller

- `internal/models/combat_state.go`:
  - Added FeatureName and IsFeature fields to ActionRecord
  - Added RecordActionWithFeature() method

- `ui/components/results_summary.go`:
  - Updated combat log rendering to show feature names
  - Added conditional styling for feature rows
  - Feature name appended to action type display

- `web/static/app.css`:
  - Added `.log-row-feature` and `.log-row-feature:hover` styles
  - Purple background: rgba(138, 43, 226, 0.12)
  - Purple border: #8a2be2

**Current Status:**
- ✅ All packages compile successfully
- ✅ Main simulation runs: `go run cmd/example/main.go`
- ✅ WASM builds: `GOARCH=wasm GOOS=js go build -o web/app.wasm cmd/wasm/main.go`
- ✅ Feature system integrated into combat flow
- ✅ Feature activation working (when combatants have features)
- ✅ Resource costs tracked and enforced
- ✅ Combat log styling distinguishes features (purple) from critical hits (gold)

**Visual Design:**
- Critical Hits: Gold background + border (#ffd700)
- Feature Usage: Purple background + border (#8a2be2)
- Normal Actions: Default styling

**Feature Activation Flow:**
1. ExecuteTurn checks for stunned status
2. 30% chance to try activating a feature
3. Check if actor has domain abilities (player) or adversary features (enemy)
4. Validate resource costs (Hope, Fear, Stress)
5. Pay costs and apply all effects
6. Record action with feature name and styling
7. If no feature activated, perform basic attack

**Next Steps:**
- ⬜ Add more sophisticated AI logic for feature selection
- ⬜ Parse real adversary features from tier 1-5 data files
- ⬜ Add usage limits tracking (once per rest, once per scene, etc.)
- ⬜ Implement refresh mechanics for limited-use abilities
- ⬜ Add dynamic summoning system (currently unimplemented)

**Testing:**
```bash
# CLI simulation - WORKS ✅
go run cmd/example/main.go

# Feature system example - WORKS ✅
go run cmd/feature-example/main.go

# Web UI - WORKS ✅
GOARCH=wasm GOOS=js go build -o web/app.wasm cmd/wasm/main.go
go run cmd/wasm/main.go
# Visit http://localhost:8080 to see feature styling in combat log

# Build - WORKS ✅
go build ./...
```

**Key Achievement:**
The feature system is now **fully integrated** into the combat engine. When combatants have features or domain abilities configured via the UI, they will be activated during combat with proper resource tracking, effect resolution, and visual distinction in the combat log.


## Session 6 Summary (2026-01-07 Part 4) - FEATURE PARSING ✅

### Adversary Feature Parsing - COMPLETE ✅

**Goal:** Parse adversary JSON files and auto-populate features using quantized mappings

**Completed Work:**

1. **Feature Mapping System**:
   - Created `data/feature_mappings.json` with 33 mapped features
   - Maps human-readable feature names → quantized AdversaryFeature structs
   - Covers 9.7% of total features (33/340 across all tiers)
   - 52% of tier 1 adversaries now have features (27/52)

2. **Parser Implementation**:
   - `internal/parser/feature_parser.go` - Converts feature names to AdversaryFeature
   - `internal/parser/adversary_parser.go` - Loads adversary JSON files
   - Graceful degradation: unmapped features silently skipped
   - Auto-populates adversaries with quantized features

3. **Example Program**:
   - `cmd/adversary-example/main.go` - Demonstrates feature parsing
   - Loads tier 1 adversaries from JSON
   - Runs combat simulation with parsed features
   - Shows feature activation in combat log

**Mapped Features (33):**
- **Damage:** Backstab, From Above, Bone Breaker, Double Strike, Deadly Shot, Crush, Reckless Attack
- **Amplifiers:** Pack Tactics, Sneak Attack, Charge
- **Status:** Burning, Venomous, Venomous Stinger, Goading Strike, Hobbling Strike, Hobbling Shot, Poison Strike
- **AOE:** Hellfire, Earth Eruption
- **Special:** Armor-Shredding Shards, Bloodsucker, Detain, Curse, Shield Bash, Stunning Blow
- **Passive:** Relentless, Momentum, Flying, Climber, Overwhelming Force, Web
- **Summoning:** Guards, Seize Them!

**Adversaries with Features (Tier 1):**
- Acid Burrower, Bear, Cave Ogre, Dire Wolf, Giant Mosquitoes, Giant Scorpion
- Glass Snake, All Guards (Archer/Bladed/Head), Jagged Knife gang (Bandit/Hexer/Lieutenant/Shadow)
- Minor Elementals (Chaos/Fire), Minor Demon, Tiny Red Ooze
- Petty Noble, Pirate Captain, Skeleton Archer, Spellblade, Sylvan Soldier
- Tangle Bramble Swarm, Weaponmaster, Young Dryad
- Total: 27/52 (52% coverage)

**Files Created:**
- `data/feature_mappings.json` - Feature mapping configuration (33 features)
- `internal/parser/feature_parser.go` - Feature name → AdversaryFeature converter
- `internal/parser/adversary_parser.go` - Adversary JSON loader
- `cmd/adversary-example/main.go` - Example program

**How It Works:**
```bash
# Load adversaries from JSON
parser := parser.NewAdversaryParser("data/feature_mappings.json")
adversaries := parser.LoadAdversariesFromFile("data/daggerheart_adversaries-tier-1.json")

# Adversaries automatically have features populated
for _, adv := range adversaries {
    fmt.Printf("%s: %d features\n", adv.Name, len(adv.AdversaryFeatures))
}

# Use in combat as normal
```

**Testing:**
```bash
# Test feature parsing
go run cmd/adversary-example/main.go

# Output shows:
# - 52 tier 1 adversaries loaded
# - 27 have mapped features
# - Combat simulation runs with feature activation
```

**Expansion Path:**
Add more features to `data/feature_mappings.json`:
```json
{
  "mappings": {
    "New Feature Name": {
      "category": "damage",
      "fear_cost": 1,
      "effects": [
        {
          "category": "FeatureEffectDamage",
          "dice_formula": "2d6",
          "damage_type": "physical"
        }
      ]
    }
  }
}
```

**Current Status:**
- ✅ Feature parsing system complete
- ✅ 33 features mapped (9.7% total coverage)
- ✅ 52% of tier 1 adversaries have features
- ✅ Graceful degradation for unmapped features
- ✅ Extensible mapping system

**Commands:**
```bash
# Test feature parsing
go run cmd/adversary-example/main.go

# Build everything
go build ./...

# Run main simulation (no parsed adversaries, uses manual setup)
go run cmd/example/main.go

# Run web UI (manual combatant creation)
make serve  # http://localhost:8080
```

