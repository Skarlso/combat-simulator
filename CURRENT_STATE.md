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

## Next Steps
- Additional testing with complex scenarios
- More abilities and features from domain cards
- Advanced AI strategies
- Performance optimization for larger simulations
