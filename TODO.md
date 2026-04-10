# TODO - Daggerheart Combat Simulator

**Last Updated**: 2026-04-10

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
