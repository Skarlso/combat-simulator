# TODO - Daggerheart Combat Simulator

**Last Updated**: 2026-01-25
**Status**: Production-ready with web-based summoning, domain ability triggers, AOE damage, armor mechanics, enhanced combat log, and Daggerheart-themed UI

## ✅ Recently Completed (2026-01-25)

### Web-Based Summoning System ✅
- **What**: Full summoning support in web UI with summon pool picker
- **Implementation**:
  - Added `SummonPool []*Combatant` to `FeatureEffect` for inline templates (no registry needed)
  - Added `StressCost`, `UsageType`, `MaxUses`, `UsesLeft` to `AdversaryFeature`
  - Added `IsSummonTemplate` flag to `Combatant` - templates don't fight unless summoned
  - Added `SummonedCounts map[string]int` to `CombatState` for global unique naming (#1, #2, #3...)
  - Added `CloneCombatant()` helper function for deep copying templates
  - Updated `resolveSummon()` to check `SummonPool` first, fall back to registry
  - Added `summonFromPool()` and `summonFromRegistry()` helper methods
  - Consolidated summon log entries ("Summons 3 allies" instead of 3 separate entries)
- **UI Changes**:
  - Cost type dropdown: Fear / Stress / Free
  - Usage limit dropdown: Unlimited / Once per Scene / Once per Session / Once per Rest
  - "Summon Allies" effect type with quantity controls (Fixed or Dice)
  - Summon pool picker with checkboxes for available adversaries
  - "Summon Template Only" checkbox - marked adversaries show [TEMPLATE] tag
  - Feature display shows stress costs, usage limits, and summon info
- **Files**: `feature.go`, `combatant.go`, `combat_state.go`, `feature_resolver.go`, `adversary_feature_form.go`, `adversary_list.go`, `app.css`
- **Status**: Functional but needs more testing

### Data Fix ✅
- Fixed `daggerheart_adversaries_tier4.json` line 564: Changed `"bonus": "2d4"` to `"bonus": 5` (Abomination's Chaotic Form uses variable attack)

## 🧪 Needs Testing (Summoning)

- [ ] Multiple summoners summoning same template type (verify unique numbering)
- [ ] Stress cost activation and limits
- [ ] Usage restrictions (once_per_scene, once_per_session, once_per_rest)
- [ ] Mixed summon pools (multiple template types)
- [ ] Edge cases: empty summon pool, 0 quantity rolls
- [ ] Template-only adversaries properly excluded from initial combat
- [ ] Summoned creatures participating correctly in spotlight selection

## ✅ Previously Completed (2026-01-10)

### Combat Log Enhancement ✅
- **What**: Overhauled combat log UI to display comprehensive battle information
- **Implementation**:
  - Added armor/HP tracking fields to `DamageResult` struct (HPBefore, HPAfter, ArmorUsed, ArmorSlotsBefore, ArmorSlotsAfter)
  - Added armor/HP tracking fields to `ActionRecord` struct (TargetHPBefore, ArmorUsed, ArmorSlotsBefore, ArmorSlotsAfter)
  - Updated `RecordAction()` and `RecordActionWithFeature()` signatures to accept `*DamageResult` parameter
  - Added `DamageResult` field to `AttackResult` and `DCCheckResult` structs
  - Updated all damage resolution functions to populate and pass DamageResult
  - Enhanced UI rendering to show: "Target: 10 HP → 8 HP (armor: 3→2 slots)"
  - Widened combat log columns (Result column now 280px) with horizontal scrolling
  - Added vertical scrolling (max-height: 600px) for long battles
  - Result column can now wrap text to show full information
- **Visual Changes**:
  - Combat log now shows complete damage narrative
  - Example: "Hit - Guardian: 10 HP → 9 HP (armor: 2→1 slots)" instead of "Hit (HP:9)"
  - Armor usage clearly visible when slots are consumed
  - HP changes show exact before/after values with target name
- **Files**: `combatant.go:137-146`, `combat_state.go:42-48,215-227`, `resolver.go:37,172`, `feature_resolver.go:26,154`, `combat_engine.go:59,133,171,239,301`, `feature_resolver.go:780`, `results_summary.go:257-274`, `app.css:554-558,562,616-620,658`

### Armor System Correction ✅
- **What**: Fixed armor to work correctly per Daggerheart rules (CRITICAL FIX)
- **Problem**: Armor was incorrectly absorbing ALL damage (like HP shields)
- **Correct Behavior**: Armor reduces damage THRESHOLD, not damage amount
  - Mark ONE slot per damage to reduce severity: Severe→Major→Minor→None
  - Severe (3 HP) + armor → Major (2 HP)
  - Major (2 HP) + armor → Minor (1 HP)
  - Minor (1 HP) + armor → None (0 HP)
  - Cannot mark multiple slots unless ability explicitly allows (e.g., Iron Will ancestry feature)
  - **Direct Damage bypasses armor completely** (adversary-only feature)
- **Implementation**:
  - Rewrote `TakeDamage()` threshold logic (combatant.go:204-238)
  - Calculate threshold FIRST, then reduce it if armor available
  - Removed "Direct Damage" from player UI dropdown (domain_ability_form.go:383-385)
  - Added "Damage Armor Slots" to player abilities (domain_ability_form.go:387)
  - Fixed `resolveArmorDamage()` to respect magnitude (feature_resolver.go:341-346)
  - Direct damage uses separate code path (feature_resolver.go:164-194)
- **Tests**:
  - `cmd/armor-threshold-test/main.go` - Verifies Severe/Major/Minor threshold reductions
  - `cmd/direct-damage-test/main.go` - Verifies direct damage bypasses armor
- **Files**: `combatant.go:204-238`, `domain_ability_form.go:383-385,387,523-526`, `feature_resolver.go:164-194,341-346`


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

### UI Redesign - Daggerheart Theme ✅
- **What**: Complete visual overhaul with official Daggerheart colors
- **Colors**:
  - Hope (#e7c74b): Gold representing player success, light, opportunity
  - Fear (#24135f): Deep purple representing GM power, danger, darkness
- **Changes**:
  - Header/footer with Fear gradients and Hope accent stripe
  - Hope-themed primary buttons, Fear-themed secondary buttons
  - Section headings use Fear text with Hope left borders
  - Stat cards themed by type (Hope for info, Fear for danger)
  - Combat log highlights (Hope for criticals, Fear for features)
  - Form focus states use Hope glow
  - All elements use smooth gradients and transitions
- **Philosophy**: Duality of Hope/Fear reflected throughout entire interface
- **File**: `web/static/app.css`

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
- `cmd/armor-threshold-test/main.go` - Armor threshold reduction (Severe→Major→Minor→None)
- `cmd/direct-damage-test/main.go` - Direct damage bypasses armor (adversary-only)
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

### 3. Summoning in Web UI ✅ RESOLVED
**Status**: Implemented using inline SummonPool (no registry needed)
- Summon pool picker allows selecting from existing adversaries
- Templates marked with "Summon Template Only" don't fight unless summoned
- Global unique naming across combat (#1, #2, #3...)
- Supports Fear/Stress costs and usage limits

## 📋 Next Priorities

### Current Status
**WASM Build**: Latest at `web/app.wasm` (2026-01-25) ✅
- **Web-based summoning with inline pool picker** ✅ (needs more testing)
- **Summon template system (IsSummonTemplate flag)** ✅
- **Stress costs and usage limits for adversary features** ✅
- Domain ability triggers fully working ✅
- AOE damage implementation complete ✅
- Armor system corrected (threshold reduction, not damage absorption) ✅
- Direct damage is adversary-only (removed from player UI) ✅
- Player armor slot manipulation (grant + damage) ✅
- Combat log enhanced with HP/armor tracking and wider layout ✅
- All core combat mechanics operational ✅
- Feature parity achieved (players = adversaries) ✅

### Priority 1: Feature Library Dropdown (Web UI) 📚
**Why**: Web UI requires manual feature creation. Pre-mapped features exist but only for CLI.

**Current State**:
- 33/340 adversary features mapped in `data/feature_mappings.json`
- ~6k lines of adversary JSON with full stat blocks
- Parser system works in CLI tests
- Web UI has blank forms only (no dropdowns)

**To Implement**:
1. Add feature library dropdown to adversary feature form
2. Load feature_mappings.json into web UI
3. Populate form fields when feature selected from dropdown
4. Still allow manual creation for custom features

**Files to Modify**:
- `ui/components/adversary_feature_form.go` - Add dropdown, load mappings
- May need to bundle `feature_mappings.json` in WASM

### Priority 2: Feature Coverage Expansion 📊
**Why**: Only 33/340 features mapped (9.7%)

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

### Druid Shapeshifting 🐺
**What**: Implement class-specific transformation mechanics
**Complexity**: High - requires alternate stat blocks and form switching
**Examples**:
- Animal forms with different HP/damage/abilities
- Form-specific features that activate/deactivate
- Resource costs for transformation
- Duration tracking and reversion mechanics
**Note**: Requires class system implementation first

### Strategic Spotlight Selection
**Current**: Random GM turn spotlight selection (`combat_engine.go:194`)
**Future**: Strategic selection based on combatant threat/abilities

### Proper Feature Damage Logging
**Current**: Generic "dealt X damage" logs
**Future**: "Player uses Fireball, deals 18 magic damage" with purple styling
**Files**: `feature_resolver.go:125,165` - RecordAction calls

### Advanced Trigger Patterns
**Examples**:
- Slow tokens (requires token before acting)
- Split mechanics (Ooze splitting on damage)

### Countdown/Loop Cost Mechanic 🔄
**What**: Alternative activation cost that ticks down on conditions
**Example**: "Blood and Souls - Countdown (Loop 6). Ticks down when PC takes violent action. When triggers, summon 1d4 Minor Demons."
**Complexity**: Medium - requires tick condition tracking, loop reset logic
**Implementation Notes**:
- Add `CountdownMax`, `CountdownCurrent` fields to AdversaryFeature
- Add `CountdownTrigger` field (e.g., "on_player_attack", "on_damage_dealt")
- Tick down in trigger processing, activate when reaches 0
- Loop resets counter to max after activation
**Deferred**: Implementing simpler summoning first (Fear/Stress/Free + usage limits)

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
- **Web-based summoning needs comprehensive testing** (see "Needs Testing" section above)
- AOE damage needs verification test
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
