# TODO - Daggerheart Combat Simulator

**Last Updated**: 2026-01-10
**Status**: Production-ready with domain ability triggers and AOE damage

## ✅ Recently Completed (2026-01-10)

### Domain Ability Triggers ✅
- **What**: Players can now have reactive abilities like enemies
- **Implementation**:
  - Added `Trigger` field to `DomainAbility` struct
  - Extended `ProcessTriggeredFeatures()` to handle domain abilities
  - Added trigger UI controls (type + filter dropdowns)
  - All 6 trigger types supported: on_attack, on_hit, on_critical, on_damage_taken, start_turn, end_turn
- **Test**: `cmd/trigger-test/main.go` - Retribution ability inflicts stress on attacker when damaged
- **Files**: `feature.go:95`, `domain_ability_form.go:197-235`, `feature_resolver.go:585-605`

### AOE Damage Implementation ✅
- **What**: Area-of-effect damage now hits multiple targets
- **Problem**: `FeatureEffectAreaDamage` was defined but not implemented in `ResolveEffect()`
- **Implementation**:
  - Added `resolveAreaDamage()` function
  - Rolls damage once, applies to all enemies of actor
  - Respects damage type (physical/magic), armor, resistance
  - Triggers on_damage_taken for each target hit
- **Test**: `cmd/aoe-test/main.go` - Fireball hits all 3 goblins with single cast
- **Files**: `feature_resolver.go:61-62,200-254`

## ✅ Previously Completed (2026-01-08 Late Evening)

### Resistance/Immunity System Refactor ✅
- **Old System**: Direct checkboxes on combatant forms (Physical/Magical resistance/immunity)
- **New System**: Feature-based passive effects
  - Added "Resistance (50%)" and "Immunity (100%)" as effect types
  - Available in both adversary features and domain abilities
  - Damage type selection (Physical/Magical) when selected
  - Automatic magnitude setting (50 or 100)
  - Implicit percentage handling (user doesn't see magnitudes)
- **Implementation**:
  - `ApplyPassiveResistances()` method in Combatant
  - Called automatically on combatant creation
  - Scans all passive features/abilities for resistance effects
  - Uses existing `AddResistance()` / `AddImmunity()` methods
- **Benefits**: More flexible, lore-friendly, consistent with other passive effects
- **Files**: `combatant.go:370-413`, `adversary_feature_form.go:232-233,396-410`, `domain_ability_form.go:349-350,497-511`

### Passive Domain Abilities ✅
- Added `Passive` field to `DomainAbility` struct (missed in earlier session)
- Added passive checkbox to domain ability UI form
- Players now have full parity with adversaries for passive abilities
- **Files**: `feature.go:94`, `domain_ability_form.go:180-194,511,554`

## ✅ Previously Completed (2026-01-08 Evening)

### Aggressive Fear Usage ✅
- Implemented spotlight grab mechanic for aggressive strategy
- Tracks consecutive player turns
- GM spends 1 Fear after 3+ player turns
- Fully integrated and tested

### Summoning System ✅
- Created AdversaryRegistry for template management
- Implemented FeatureEffectSummonAllies resolution
- Added SummonCombatant() to combat state
- AI evaluates summon features appropriately
- Tested with "Guards, Seize Them!" feature
- CLI fully functional, web UI gracefully handles missing registry

### Critical Bug Fixes ✅

#### DC Check Auto-Fail Bug
- **Problem**: UI defaulted DCType to "target", failed when target was nil
- **Impact**: All self-targeted abilities failed, 100-round stalemates
- **Fix**: Auto-succeed when target is nil, changed UI defaults to "none"
- **Files**: `feature_resolver.go:292-294`, both UI forms

#### Incorrect Fallback Logic
- **Problem**: Abilities that failed DC checks fell back to basic attack (double actions)
- **Impact**: Turn consumption broken, abilities didn't properly consume turns
- **Fix**: Only fallback when can't activate (insufficient resources), not on DC failure
- **Rationale**: Basic attacks generate Hope (no cost), abilities consume Hope (paid upfront)
- **Files**: `combat_engine.go:63-90`

### Tests Created
- `cmd/fear-test/main.go` - Aggressive Fear usage validation
- `cmd/summon-test/main.go` - Summoning system demonstration
- `cmd/dc-fail-test/main.go` - DC failure turn consumption
- `cmd/fallback-test/main.go` - Resource-based fallback

## ✅ Previously Completed (2026-01-08 Morning)

### Critical Bug Fix
- **FIXED**: `executeAbility()` and `executeFeature()` now properly call validation methods
- All activation paths now enforce: affordability checks, DC rolls, token mechanics, uses decrementing

### Code Cleanup
- Removed ~100 lines of dead code
- Removed unused functions: `NewCondition()`, `NewDOTCondition()`, `resolveDamageAmplifier()`
- Removed unused struct fields: `ActiveCondition.Source`
- Removed unimplemented effects: `FeatureEffectDamageAmplifier`
- Removed unused constants: Fire/Frost/Poison/PhysicalMagic damage types, 7 unused trigger types, ConditionCloaked
- Removed 30-line dead loop for domain ability triggers
- Project now has no unused code or unimplemented features in production

## 🚧 Current Limitations

### 1. Domain Abilities Don't Have Triggers
**Impact**: Medium - Players can't have reactive abilities like enemies do

**What's Missing**:
- Domain abilities lack the `Trigger` field that adversary features have
- Can't implement abilities like "when ally takes damage, heal them"
- Only manual activation supported currently

**To Fix**:
```go
// Add to DomainAbility struct in internal/models/feature.go
Trigger FeatureTriggerCondition
```

Then wire up trigger processing in `feature_resolver.go` similar to adversary features.

### 2. Adjacency Not Implemented ⚫ (Deprioritized)
**Impact**: Very Low - Only affects niche trigger filters

**Reason for Deprioritization**:
- Would require full positional/distance system
- Adds significant complexity for minimal benefit
- Theater-of-mind TTRPG doesn't need precise positioning
- Only affects "if_ally_adjacent" filter (very rare)

**Current Workaround**: Filter returns false, features don't trigger - acceptable limitation

### 3. Summoning in Web UI
**Impact**: Low - Works in CLI, not yet in web

**What's Missing**:
- Web UI doesn't load AdversaryRegistry
- Summon features fail gracefully with error message
- Need to either bundle JSON files or add upload capability

**To Fix** (if needed):
- Bundle adversary JSON files in WASM resources
- Or add UI for uploading adversary data
- Or accept CLI-only for now

## 📋 Next Priorities

### Current Status
**WASM Build**: Latest at `ui/web/app.wasm` (2026-01-10 12:53) ✅
- Domain ability triggers fully working ✅
- AOE damage implementation complete ✅
- UI updated with trigger controls and AOE effect type ✅
- All core combat mechanics operational ✅
- Feature parity achieved (players = adversaries) ✅

### Priority 1: Feature Coverage Expansion 📊
**Why**: Currently only 33/340 features mapped (9.7%)

**Status**: Not urgent - core system works, just needs more data entry

**Tasks**:
1. Parse tier 2-3 adversary features
2. Expand `data/feature_mappings.json`
3. Add more complex patterns (conditional effects, token systems)
4. Test new features in simulation

**Files to Modify**:
- `data/feature_mappings.json` - Add more mappings
- `internal/parser/feature_parser.go` - Handle new patterns if needed

## 🔮 Future Enhancements

### Strategic Spotlight Selection
**Current**: Random GM turn spotlight selection (`combat_engine.go:194`)
**Future**: Strategic selection based on combatant threat/abilities

### Proper Feature Damage Logging
**Current**: Generic "dealt X damage" logs
**Future**: "Player uses Fireball, deals 18 magic damage" with purple styling
**Files**: `feature_resolver.go:125,165` - RecordAction calls

### Advanced Trigger Patterns
**Examples**:
- Countdown mechanics ("On My Signal" countdown:5)
- Slow tokens (requires token before acting)
- Split mechanics (Ooze splitting on damage)

### Monte Carlo Optimization
**Current**: 200-500 simulations per strategy test
**Future**: Parallel processing, adaptive sampling

## 📊 Current Coverage

### Effect Types: 24/24 (100%)
All implemented effect categories are working and tested.

### Adversary Features: 33/340 (9.7%)
Core system works, just needs more feature mappings.

### Domain Abilities: ~20 examples
UI supports full configuration, ready for expansion.

### Trigger Types: 6/7
- ✅ on_attack
- ✅ on_hit
- ✅ on_critical
- ✅ on_damage_taken
- ✅ start_turn
- ✅ end_turn
- ⬜ always (removed - design issue)

## 🛠️ Maintenance Notes

### Code Quality Checklist
- ✅ No unused functions
- ✅ No unused struct fields
- ✅ No dead code loops
- ✅ No unimplemented effects in production code
- ✅ All builds passing (WASM + CLI)
- ✅ Proper error handling throughout
- ✅ Consistent naming conventions

### Known TODO Comments in Code
- `combat_engine.go:194` - Strategic spotlight selection
- `feature_resolver.go:125,165` - Proper feature damage logging via RecordAction
- `feature_resolver.go:613` - Implement adjacency tracking

### Testing Gaps
- AOE damage needs verification test
- Summoning system needs implementation + tests
- Domain ability triggers need test coverage

## 📝 Documentation Status

- ✅ CURRENT_STATE.md - Comprehensive project status
- ✅ TODO.md - This file, updated 2026-01-08
- ✅ Code comments - Key sections documented
- ⬜ API documentation - Could use godoc expansion
- ⬜ User guide - No end-user documentation yet

## 🎯 Success Metrics

**Combat Simulation**:
- ✅ Full Daggerheart combat rules implemented
- ✅ Hope/Fear/Stress resource tracking
- ✅ Threshold damage with armor
- ✅ Spotlight system with shifts
- ✅ 10 condition types with durations

**Feature System**:
- ✅ DC checks with success probability
- ✅ Token place/spend mechanics
- ✅ Damage reduction (Ward-style)
- ✅ Trigger system (6 trigger types)
- ✅ 24 effect categories

**AI & Strategy**:
- ✅ Expected value evaluation
- ✅ Cost-benefit analysis
- ✅ DC success probability factored in
- ✅ 4 strategy profiles (Greedy, Conservative, Desperate, Random)
- ✅ Monte Carlo optimization (200-500 runs)

**Code Quality**:
- ✅ Production-ready codebase
- ✅ No dead code
- ✅ All builds passing
- ✅ Critical bugs fixed

## 🚀 Long-Term Vision

1. **Complete Feature Coverage** - Map all 340 adversary features + domain abilities
2. **Web UI Polish** - Better visualization, combat replay, save/load scenarios
3. **API Endpoints** - RESTful API for external integrations
4. **Strategy Benchmarking** - Comprehensive strategy comparison tool
5. **Character Optimizer** - Monte Carlo-based build recommendations
6. **Campaign Mode** - Multi-combat scenarios with resource persistence
