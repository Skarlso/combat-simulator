# TODO - Feature System Implementation

**Created**: 2026-01-06
**Status**: Planning phase for quantizing human-readable features

## Goal

Convert human-readable adversary and player features into quantifiable mechanical data that the combat simulator can process.

## Important Feature Categories to Implement

### 1. Action Economy ⭐
- `spotlight_modifier`: Extra spotlight uses per GM turn (e.g., Relentless: 2-3 uses)
- `fear_cost_modifier`: Changes Fear cost to take spotlight (e.g., Ramp Up: requires Fear)
- `ally_spotlight`: Spotlight allies when activated (e.g., Rally Guards, Tactician)

### 2. Damage Modifiers ⭐⭐⭐
- `damage_amplifier`: Conditional bonus damage (e.g., Pack Tactics, Backstab)
- `direct_damage`: Bypasses armor completely (e.g., Bone Breaker)
- `area_damage`: Hits multiple targets in range
- `conditional_damage`: Bonus under specific conditions (e.g., From Above, Hidden attacks)

### 3. Resistance/Weakness ⭐⭐
- `damage_resistance`: Reduce damage from specific type (physical/magic)
- `damage_vulnerability`: Take extra damage from specific type
- `extra_hp_on_damage`: Mark additional HP when taking damage from specific source

### 4. Status Application ⭐⭐⭐
- `vulnerable`: Target takes increased damage until condition cleared
- `poison`: Ongoing stress/damage per action (e.g., Venomous Stinger)
- `ignite`: Fire damage over time per action
- `taunt`: Forces disadvantage against other targets (e.g., Goading Strike)
- `hidden`: Grants advantage on next attack

**NOT IMPLEMENTED** (movement-related):
- ~~restrain~~ (doesn't affect damage simulation)

### 5. Resource Manipulation ⭐⭐⭐
- `stress_inflict`: Force target to mark Stress (amount, condition)
- `stress_clear`: Clear own Stress (amount, cost)
- `hp_heal`: Clear own HP (amount, cost, condition)
- `armor_damage`: Mark target's Armor Slot (e.g., Acid Bath, Armor-Shredding Shards)
- `hope_drain`: Remove target's Hope tokens (amount)
- `fear_generate`: Gain Fear on specific condition (e.g., Momentum on successful hit)

### 6. Summoning ⭐⭐
- `summon_allies`: Spawn additional enemies (type, quantity, range)
  - Examples: "Guards, Seize Them!" (1d4 Bladed Guards), "More Where That Came From" (3 Lackeys)
  - Must dynamically add to enemy list

**NOT IMPLEMENTED**:
- ~~split~~ (e.g., Ooze splitting - too complex for initial implementation)

### 7. Special Mechanics ⭐⭐
- `slow`: Requires token before acting (special activation pattern)
- `minion`: Dies in one hit, extra defeats per damage threshold
- `horde`: Reduced damage when HP drops below threshold
- `countdown`: Triggers after X events (e.g., "On My Signal" countdown:5)

### 8. NOT IMPORTANT (Positional Mechanics)
These don't affect damage calculations in our simulation:
- ~~knockback~~ (push target away)
- ~~pull~~ (drag target closer)
- ~~immobilize~~ (prevent movement)

## Implementation Structure

Each feature will need:
```go
type FeatureMechanic struct {
    Category    string          // e.g., "damage_amplifier"
    Magnitude   int             // Numeric value (damage, stress, etc.)
    DamageDice  string          // e.g., "2d6+3"
    Condition   string          // When it triggers (on_attack, on_damage, passive)
    Cost        ResourceCost    // Stress/Fear cost to activate
    UsesPerScene int            // Limit (0 = unlimited)
    Duration    string          // How long effect lasts
    TargetType  string          // self, single, aoe
}

type ResourceCost struct {
    Stress int
    Fear   int
    HP     int
}
```

## Domain Card Analysis (Player Abilities)

Domain cards differ from adversary features - they are **active player choices** vs passive/AI-triggered effects.

### Key Domain Card Mechanics (Combat-Relevant)

**Damage Spells** ⭐⭐⭐
- Direct damage: `1d20+3`, `2d6`, `d8+3` magic/physical
- Scaling damage: `Xd10` where X = resources spent (Unleash Chaos)
- AOE damage: `2d8+4` to multiple targets (Chain Lightning)
- Per-stress scaling: `1d20+2 per Stress marked` (Falling Sky)

**Status Application** ⭐⭐
- On Fire: `2d6 magic damage` per action taken while burning
- Vulnerable: Marked permanently or until cleared (Eternal Enervation)
- Asleep: Target disabled until damage taken
- ~~Restrained~~: Movement (ignored in sim)

**Defensive Abilities** ⭐⭐
- Damage reduction: Roll `1d8`, reduce incoming damage (Rune Ward)
- Armor bonus: `+1 to Armor Score` until rest (Tava's Armor)
- Damage reflection: Roll `Xd6`, if any = 6, reflect attack (Arcane Reflection)
- Attack negation: Roll `Xd6`, block on 5+ (Confusing Aura)

**Resource Costs** ⭐⭐⭐
- Hope spending: Most spells cost 1-5 Hope
- Stress marking: `Mark X Stress` to activate (often 1-2)
- Recall cost: Hope cost to swap card from vault (0-4)
- Combo mechanics: Spend Hope to roll extra damage dice

**Scaling Mechanics** ⭐⭐
- Token system: Place tokens, spend for effects (Unleash Chaos)
- Damage storage: Store d8s, add to next attack (Sigil of Retribution)
- Variable cost: Spend any number of Hope/Stress for proportional effect

**NOT IMPORTANT**:
- Teleportation, information gathering, illusions, utility magic

### Key Differences vs Adversaries
1. **Active vs Passive**: Players choose when to use abilities
2. **Resource Costs**: Hope/Stress costs are central to player abilities
3. **Frequency Limits**: "once per rest", "once per long rest"
4. **Scaling Options**: Many abilities let players spend more resources for more effect

## Current Status (2026-01-07)

### ✅ Feature System - COMPLETE
- Separated AdversaryFeature and DomainAbility structures
- FeatureEffect resolution with 20+ effect categories
- FeatureResolver engine with resource validation
- UI forms for adding features to combatants
- Working standalone example

### ⚠️ Legacy Removal - PARTIAL
- Old Ability system partially removed
- Combat engine broken (does not compile)
- Needs integration with new feature system

### 📋 Next Steps

**Priority 1 - Fix Broken Compilation:**
1. ⬜ Refactor combat_engine.go to remove Ability references
2. ⬜ Refactor decision_maker.go to use features instead of abilities
3. ⬜ Update resolver.go attack resolution
4. ⬜ Integrate FeatureResolver into combat flow
5. ⬜ Test basic simulation without features first

**Priority 2 - Feature Integration:**
6. ⬜ Add AI logic for when to activate features
7. ⬜ Wire up feature activation in combat turns
8. ⬜ Resource cost tracking during simulation
9. ⬜ Feature usage in combat log

**Priority 3 - Advanced Features:**
10. ⬜ Add dynamic summoning system to combat state
11. ⬜ Parse real adversary features from tier 1 data
12. ⬜ Feature display in results summary
13. ⬜ Domain ability usage limits and refresh mechanics

## Completed Work (2026-01-06 Part 3)

### Feature System Implementation ✅

Created a complete feature quantization system for converting human-readable abilities into mechanical data:

**New Files:**
- `internal/models/feature.go` - Feature type definitions
- `internal/engine/feature_resolver.go` - Effect resolution logic
- `cmd/feature-example/main.go` - Working example demonstrating the system

**Key Components:**
1. **FeatureEffect** - Shared effect system with 20+ categories (damage, status, resource manipulation)
2. **AdversaryFeature** - Enemy abilities with Fear costs and spotlight limits
3. **DomainAbility** - Player spells with Hope/Stress costs and usage limits
4. **FeatureResolver** - Engine for activating features and resolving effects

**Supported Effect Categories:**
- Damage (standard, amplifier, direct, area)
- Status application (poison, burning, vulnerable, taunt, hidden)
- Resource manipulation (stress, HP, armor, hope, fear)
- Resistance/vulnerability (passive modifiers)
- Action economy (spotlight modifiers)

**Tested Features:**
- Player: "Rune Strike" - 2 Hope, deals 2d6+3 magic damage
- Enemy: "Venomous Stinger" - Applies poison + 2 stress
- Enemy: "Dark Ritual" - 2 Fear, deals 1d8 direct damage

Run example: `go run cmd/feature-example/main.go`

## Completed Work (2026-01-07)

### UI Forms for Feature System ✅

**Components Created:**
- `ui/components/domain_ability_form.go` - Player ability form (300+ lines)
- `ui/components/adversary_feature_form.go` - Enemy feature form (290+ lines)

**Features:**
- Dynamic form fields based on effect type
- Dice formula input (e.g., "2d6+3")
- Resource cost configuration (Hope, Stress, Fear)
- Usage limits and refresh types
- Passive/active toggle for adversaries
- Duration and targeting options

**Integration:**
- Added to `adversary_list.go` for enemies
- Added to `combatant_list.go` for players
- Forms follow existing pattern from `ability_form.go`

Test: `make serve` → http://localhost:8080

## Partial Work (2026-01-07 Part 2)

### Legacy System Removal - INCOMPLETE ⚠️

**Files Deleted:**
- `internal/models/ability.go` (216 lines)
- `internal/engine/effects.go` (145 lines)
- `internal/models/action.go` (24 lines)
- `ui/components/ability_form.go` (351 lines)

**Files Modified:**
- `internal/models/combatant.go` - Removed Abilities field and AddAbility()
- `internal/models/combat_state.go` - RecordAction signature changed (ability → string)
- `internal/models/dice_notation.go` - Removed ConvertToDamageEffect()
- `internal/engine/resolver.go` - Removed ability methods (CheckAbilityConditions, CanAffordAbility, PayAbilityCost, ApplyHealing)
- `ui/components/adversary_list.go` - Removed AbilityForm usage
- `ui/components/combatant_list.go` - Removed AbilityForm usage
- `cmd/example/main.go` - Removed ability creation functions

**Compilation Status:** ❌ BROKEN
- Combat engine references deleted types
- Decision maker expects old ability system
- Main simulation cannot run

**Why Incomplete:**
Legacy system deeply integrated into:
- Combat turn execution logic
- AI decision making for ability usage
- Attack resolution expecting DamageEffect interface

**Next Session Should:**
1. Restore compilation (stub out or rewrite broken parts)
2. Get basic simulation working without features
3. Then incrementally add feature system integration

## Related Files

- `internal/models/feature.go` - Feature type definitions ✅
- `internal/models/combatant.go` - Features attached here ✅
- `internal/engine/feature_resolver.go` - Feature resolution logic ✅
- `cmd/feature-example/main.go` - Working example ✅
- `data/daggerheart_adversaries-tier-*.json` - Source for adversary features
- `data/daggerheart_domain_*.json` - Source for player domain abilities
