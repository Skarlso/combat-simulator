# TODO - Daggerheart Combat Simulator

**Last Updated**: 2026-04-20

## Priority 0: Feature Resolver Correctness (DONE 2026-04-20)

- DamageReduction passive gate fixed (`calculateDamageReduction` now respects `.Passive`); 7 tests added.
- FeatureResolver test suite added (20 tests covering activate, process, shouldTrigger, resolve).
- Adversary tokens wired: `AdversaryFeature` gains `UsesTokens`/`TokenName`/`TokensHeld`/`TokensMax`; spend/place mirrors domain ability logic; parser + UI form updated; 4 tests.
- Unleash Chaos multi-turn regression: 2 tests (persist across turns, cap, charge-then-release).

### Still outstanding
- `effect_damage.go`, `effect_resource.go`, `effect_summon.go` direct tests (covered indirectly by resolver tests, but gap remains).

## Priority 0.5: Spotlight Transfer Features (NOT MODELED)

Two common SRD patterns currently unrepresented:

1. **Minion group spotlight** ("Spend a Fear to spotlight all X within Close range; shared attack roll"). Present on 15+ adversaries across all tiers: Giant Rats, Jagged Knife Lackeys, Minor Treants, Sellswords, Dredges, Tangle Brambles, Rotted Zombies, Apprentice Assassins, Conscripts, Cult Initiates, Giant Recruits, Elemental Sparks, Treant Saplings, Fallen Shock Troops, Hallowed Soldiers, Outer Realm Thralls.
2. **Master/minion spotlight chain** ("When the Beastmaster is spotlighted, you can also spotlight a Tier 1 animal adversary currently under their control"). Beastmaster (tier2).

**To implement:**
- Add `FeatureEffectSpotlightMinions` category (cost: Fear, effect: find minions of named type within range, shared attack vs target, combined damage).
- Add `FeatureEffectSpotlightAlly` category for master/minion chains (triggered on master's spotlight, free extra spotlight).
- Wire into `combat_engine.go` spotlight loop so GM turn recognizes these before the standard 1-free + 1-Fear-each rule.

## Priority 0.75: JSON Character Import (Multi-Character)

UI forms for building PCs are painful for repeat setup. Add bulk JSON import.

- One file, multiple PCs: `{"characters": [{...}, {...}]}`
- Reuse the adversary JSON schema style where possible, but PC-specific (class, domains, hope max, etc.).
- Import button in web UI that parses and adds all characters to the combatant list.
- Validate on import (HP/stress bounds, known domains, attack dice format).
- Consider same for adversaries (multi-adversary JSON) for scenario packs.



## Needs Testing (Summoning)

- [ ] Multiple summoners summoning same template type (verify unique numbering)
- [ ] Stress cost activation and limits
- [ ] Usage restrictions (once_per_scene, once_per_session, once_per_rest)
- [ ] Mixed summon pools (multiple template types)
- [ ] Edge cases: empty summon pool, 0 quantity rolls
- [ ] Template-only adversaries properly excluded from initial combat
- [ ] Summoned creatures participating correctly in spotlight selection

## Priority 1: Feature Library Dropdown (Web UI)

Web UI requires manual feature creation. Pre-mapped features exist but only for CLI.

- 33/340 adversary features mapped in `data/feature_mappings.json`
- ~6k lines of adversary JSON with full stat blocks
- Parser system works in CLI tests
- Web UI has blank forms only (no dropdowns)

**To implement**:
1. Add feature library dropdown to adversary feature form
2. Load `feature_mappings.json` into web UI
3. Populate form fields when feature selected from dropdown
4. Still allow manual creation for custom features

**Files**: `ui/components/adversary_feature_form.go`, possibly bundle `feature_mappings.json` in WASM

## Priority 2: Feature Coverage Expansion

Only 33/340 features mapped (9.7%). Core system works, just needs more data entry.

1. Parse tier 2-3 adversary features
2. Expand `data/feature_mappings.json`
3. Add more complex patterns (conditional effects, token systems)
4. Test new features in simulation

## Future Enhancements

### Druid Shapeshifting
Alternate stat blocks and form switching. Requires class system implementation first.
- Animal forms with different HP/damage/abilities
- Form-specific features that activate/deactivate
- Resource costs for transformation
- Duration tracking and reversion mechanics

### Advanced Trigger Patterns
- Slow tokens (requires token before acting)
- Split mechanics (Ooze splitting on damage)

### Countdown/Loop Cost Mechanic
Alternative activation cost that ticks down on conditions.
Example: "Blood and Souls - Countdown (Loop 6). Ticks down when PC takes violent action. When triggers, summon 1d4 Minor Demons."
- Add `CountdownMax`, `CountdownCurrent` fields to `AdversaryFeature`
- Add `CountdownTrigger` field (e.g., "on_player_attack", "on_damage_dealt")
- Tick down in trigger processing, activate when reaches 0
- Loop resets counter to max after activation

### Simulation Parallelism
The simulation runner is a straightforward Monte Carlo simulation (repeated random sampling).
Each combat is independent -- trivial to parallelize with goroutines if run counts increase.
Current performance (~200ms for 1000 runs) doesn't warrant it yet.

## Coverage

| Category | Status |
|---|---|
| Effect Types | 24/24 (100%) |
| Adversary Features | 33/340 (9.7%) |
| Domain Abilities | ~20 examples |
| Trigger Types | 6/6 (on_attack, on_hit, on_critical, on_damage_taken, start_turn, end_turn) |

## Known TODO Comments in Code

- `feature_resolver.go` -- Adjacency tracking (deprioritized, theater-of-mind TTRPG)
