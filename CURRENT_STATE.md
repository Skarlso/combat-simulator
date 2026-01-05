# Current State - Adversary System Implemented

## What Works
- ✅ Core combat engine (fully tested via CLI)
- ✅ Monte Carlo simulation (1000 runs in ~200ms)
- ✅ AI strategies
- ✅ Data models
- ✅ Server starts and serves HTML
- ✅ Example CLI program works perfectly
- ✅ Event handlers fixed (ctx.Update() calls added)
- ✅ Adversary system with separate mechanics

## Recent Changes

### UI Event Handler Fixes
- ✅ Added `ctx.Update()` to tab switching
- ✅ Added `ctx.Update()` to form toggles
- ✅ Added `ctx.Update()` to button clicks
- ✅ Fixed missing `app.RunWhenOnBrowser()` call in main.go

### Adversary System Implementation
- ✅ **Models**: Added Adversary-specific fields to Combatant
  - IsAdversary, Tier, AdversaryType, Difficulty, ATK, Motives, Tactics, Experience
  - CustomThresholds for both Players and Adversaries
  - Helper methods: GetDefense(), GetAttackBonus()

- ✅ **Engine Refactoring**: Native Adversary mechanics
  - Hit calculation uses GetDefense() (Difficulty for adversaries, Evasion for players)
  - Attack modifiers use GetAttackBonus() (ATK for adversaries, Proficiency for players)
  - Custom threshold system (user-definable per combatant)

- ✅ **UI Components**:
  - Player form: Added Minor/Major threshold inputs
  - Created AdversaryList component with full form (Tier, Type, Difficulty, ATK, Thresholds, Motives)
  - Enemies tab now uses AdversaryList instead of CombatantList

## Files Modified
1. `/Users/skarlso/goprojects/daggerheart-combat-simulator/internal/models/combatant.go` - Added Adversary fields, NewAdversary(), helper methods
2. `/Users/skarlso/goprojects/daggerheart-combat-simulator/internal/models/types.go` - Custom threshold function
3. `/Users/skarlso/goprojects/daggerheart-combat-simulator/internal/engine/resolver.go` - Uses GetDefense(), GetAttackBonus(), custom thresholds
4. `/Users/skarlso/goprojects/daggerheart-combat-simulator/internal/engine/combat_engine.go` - Uses GetAttackBonus()
5. `/Users/skarlso/goprojects/daggerheart-combat-simulator/ui/components/combatant_list.go` - Added threshold inputs
6. `/Users/skarlso/goprojects/daggerheart-combat-simulator/ui/pages/simulator.go` - Fixed event handlers, switched enemies tab to AdversaryList

## Files Created
7. `/Users/skarlso/goprojects/daggerheart-combat-simulator/ui/components/adversary_list.go` - New Adversary form component

## Known Good State
- ✅ CLI example: `make run-example` - **WORKS PERFECTLY**
- ✅ Server starts: `make serve` - **WORKS**
- ✅ HTML renders: Browser shows content - **WORKS**
- ✅ Interactivity: **FIXED** (ctx.Update() calls added)
- ✅ Build: Both regular and WASM builds compile successfully
- ✅ Adversary mechanics: Engine supports both Player (Evasion/Proficiency) and Adversary (Difficulty/ATK) systems

## Next Steps
- Test the UI to ensure:
  - Player form accepts custom thresholds
  - Adversary form creates adversaries correctly
  - Simulation runs with mixed Player/Adversary combatants
  - Hit calculations respect Difficulty vs Evasion
  - Damage thresholds use custom values
