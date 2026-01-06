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

## Next Steps

1. ✅ Parse domain cards (player skills) from data folder
2. ⬜ Create unified feature definition format (adversary + player)
3. ⬜ Design user input system for defining quantized features
4. ⬜ Implement feature application in combat engine
5. ⬜ Add dynamic summoning system to combat state
6. ⬜ Implement resource cost tracking (Hope/Stress/Fear)
7. ⬜ Test with real adversary features from tier 1 data

## Related Files

- `data/daggerheart_adversaries-tier-*.json` - Adversary features to quantize
- `data/daggerheart_domain_*.json` - Player domain card abilities
- `internal/models/combatant.go` - Where features will attach
- `internal/engine/resolver.go` - Where features will be applied
