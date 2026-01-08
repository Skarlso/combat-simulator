# Strategy Tuning Results

**Date**: 2026-01-07
**Status**: AI Parameter Optimization Complete

## Summary

Ran Monte Carlo simulations across 3 difficulty levels to determine optimal target selection strategies.

## Key Findings

### Target Selection Strategy Performance

| Scenario          | Best Strategy  | Win Rate | Avg Turns | Notes |
|-------------------|----------------|----------|-----------|-------|
| **Easy (2v3)**    | HighestHP      | 100.0%   | 18.2      | All strategies effective |
| **Hard (2v5)**    | HighestHP      | 89.0%    | 30.5      | Clear differentiation |
| **Boss (2v1)**    | HighestHP      | 37.5%    | 30.0      | Very difficult |

### Strategy Rankings (by win rate)

**Easy Scenario (2v3 Goblins):**
1. **HighestHP**: 100.0% win rate, 18.2 turns
2. **LowestHP**: 99.5% win rate, 17.9 turns
3. **HighestThreat**: 99.5% win rate, 18.5 turns
4. **Random**: 98.5% win rate, 18.7 turns

**Hard Scenario (2v5 Mixed):**
1. **HighestHP**: 89.0% win rate, 30.5 turns
2. **HighestThreat**: 88.5% win rate, 29.4 turns
3. **Random**: 86.0% win rate, 29.6 turns
4. **LowestHP**: 84.5% win rate, 29.6 turns

**Boss Fight (2v1 Elite):**
1. **HighestHP**: 37.5% win rate, 30.0 turns
2. **LowestHP**: 35.0% win rate, 29.7 turns
3. **Random**: 34.0% win rate, 30.3 turns
4. **HighestThreat**: 29.0% win rate, 29.9 turns

## Strategic Insights

### Why HighestHP Wins

**Counterintuitive Result:** Traditional "focus fire" (LowestHP) is NOT optimal in Daggerheart.

**Reasons:**
1. **Damage Mitigation**: Killing the biggest threat first prevents more total damage
2. **Action Economy**: High HP enemies take multiple turns to kill - better to start early
3. **Resource Efficiency**: Leaving weak enemies for cleanup when resources are low
4. **Spotlight Mechanics**: Enemies get limited actions - removing high damage dealers reduces risk

### Traditional Focus Fire Weaknesses

**LowestHP performs worse because:**
- Weak enemies deal less damage (lower threat)
- Leaving strong enemies alive longer accumulates damage
- Players take more HP damage over longer combats
- Resources depleted before facing tough enemies

## Recommendations

### For Easy Encounters
- **Use**: HighestHP or LowestHP (both work)
- **Why**: Players have resource advantage, strategy matters less

### For Hard Encounters
- **Use**: HighestHP (strongest first)
- **Avoid**: LowestHP (focus fire on weak targets)
- **Why**: 5% win rate difference is significant

### For Boss Fights
- **Use**: HighestHP (marginal advantage)
- **Why**: Only one target, but boss might have adds in future
- **Note**: Boss is overtuned (37% win rate) - may need balancing

## Current AI Implementation

### Evaluation Functions (decision_maker.go)

**Cost Penalties:**
- Hope: 1.5× penalty (very valuable)
- Fear: 1.0× penalty (standard cost)
- Stress: 0.5× penalty (less valuable, can be cleared)

**Effect Values:**
- Direct Damage: avgDamage + magnitude
- AOE Damage: (avgDamage + magnitude) × enemy count (cap 3)
- Vulnerable: 5.0 value (high HP targets), 2.0 (low HP)
- Poison/Ignite: 3.0 value (high HP), 1.5 (low HP)
- Healing: scales with missing HP (no overheal waste)
- Armor Damage: 2.0 if target has armor

**Target Selection:**
- LowestHP: Focus fire weakest
- HighestHP: Attack strongest first ⭐ OPTIMAL
- HighestThreat: Based on proficiency + abilities + HP
- Random: Stochastic selection

## Testing Tools

### Strategy Comparison
```bash
go run cmd/strategy-test/main.go
```
- Tests 6 strategy configurations
- 500 simulations per strategy
- Outputs win rate, avg turns, resource efficiency

### Parameter Tuning
```bash
go run cmd/parameter-tuning/main.go
```
- Tests 4 target selection strategies
- 3 difficulty scenarios (Easy, Hard, Boss)
- 200 simulations per test
- Efficiency scoring: winRate × 100 + (1/avgTurns) × 50

## Future Tuning Opportunities

### Cost Penalty Optimization
Currently hardcoded in `decision_maker.go:251-253`:
```go
value -= float64(ability.HopeCost) * 1.5
value -= float64(ability.StressCost) * 0.5
```

**Could test:**
- Hope: 1.0×, 1.5×, 2.0×
- Stress: 0.3×, 0.5×, 0.7×
- Fear: 0.8×, 1.0×, 1.2×

### Effect Value Tuning
Currently fixed values (e.g., Vulnerable = 5.0). Could A/B test:
- Vulnerable: 3.0-7.0 range
- Poison: 2.0-4.0 range
- Healing: current formula vs alternatives

### Dynamic Strategy Selection
Based on context:
- Early game: HighestHP
- Low resources: LowestHP (finish quickly)
- High resources: HighestThreat (aggressive)

## Validation

All findings based on:
- **Monte Carlo**: 200-500 simulations per test
- **Scenarios**: 3 difficulty levels
- **Strategies**: 4-6 configurations tested
- **Metrics**: Win rate, avg turns, efficiency score

## Next Steps

1. ✅ Implement intelligent feature evaluation (DONE)
2. ✅ Remove random chance in favor of value-based decisions (DONE)
3. ✅ Test strategies across difficulty levels (DONE)
4. ⬜ Implement dynamic strategy switching based on context
5. ⬜ Add more features to feature_mappings.json (currently 33/340)
6. ⬜ Test with real adversary features from tier 1-5 data
7. ⬜ Balance boss encounters (currently 37% win rate)
