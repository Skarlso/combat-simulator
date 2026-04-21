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

## Priority 0.75: JSON Import (DONE 2026-04-21)

- **Adversaries**: `AdversaryList` now has an `Import JSON` button that accepts `{"adversaries":[...]}` (same schema as bundled tier files). Uses `parser.AdversaryParser` with embedded feature mappings. Unmapped feature names still drop silently (documented limitation, not a regression of this change).
- **Players**: new `parser.LoadCharactersFromBytes` with `CharacterFile` schema supporting name/level/evasion/proficiency/hp/stress/armor/damage_dice/thresholds/domain_abilities. 5 tests. `CombatantList` now has matching `Import JSON` panel wired to the same side (Player/Enemy) as the list it's attached to.
- File upload (vs paste textarea) deferred — not worth the go-app JS bridging yet.



## Needs Testing (Summoning)

- [ ] Multiple summoners summoning same template type (verify unique numbering)
- [ ] Stress cost activation and limits
- [ ] Usage restrictions (once_per_scene, once_per_session, once_per_rest)
- [ ] Mixed summon pools (multiple template types)
- [ ] Edge cases: empty summon pool, 0 quantity rolls
- [ ] Template-only adversaries properly excluded from initial combat
- [ ] Summoned creatures participating correctly in spotlight selection

## Priority 1: Feature Library Dropdown (DONE 2026-04-20)

- `parser.FeatureParser.FeatureNames()` added + tested.
- `AdversaryFeatureForm` now loads `data.GetFeatureMappings()` in `OnMount`, exposes a "Start from template" dropdown above the name field (add-mode only). Selecting a mapped feature populates name/passive/costs/tokens/DC/first effect; user can still tweak before saving.
- "Custom Feature" option retained for manual entry.
- Coverage remains 33/340; Priority 2 (expand mappings) still open.

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
