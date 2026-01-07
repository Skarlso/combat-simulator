# Continuation Prompt for Daggerheart Combat Simulator

Use this prompt to continue development from the current state:

---

**Prompt:**

```
I'm working on a Daggerheart Combat Simulator (Go + WebAssembly UI).

Current state:
- ✅ Core combat mechanics verified against official rules (duality dice, damage thresholds, Hope/Fear)
- ✅ Feature quantization system complete (AdversaryFeature, DomainAbility, FeatureEffect)
- ✅ Feature system integrated into combat engine with resource tracking
- ✅ Combat log displays features with purple styling (distinct from gold critical hits)
- ✅ Feature parsing from JSON: 33 features mapped, 52% of tier 1 adversaries have features
- ✅ UI forms for creating custom features via web interface
- ✅ WebAssembly build works, web UI at http://localhost:8080

Project structure:
- `/data` - Adversary JSON files, feature_mappings.json
- `/internal/models` - Combatant, CombatState, Feature types
- `/internal/engine` - CombatEngine, FeatureResolver, attack resolution
- `/internal/parser` - Feature parser, adversary loader
- `/internal/strategy` - AI decision making
- `/cmd/example` - Basic simulation CLI
- `/cmd/adversary-example` - Feature parsing example
- `/cmd/wasm` - WebAssembly server
- `/web` - Frontend (HTML/CSS/JS)

Check CURRENT_STATE.md and TODO.md for full details.

Ready to continue. What would you like me to work on?
```

---

## Quick Commands

```bash
# Run main simulation
go run cmd/example/main.go

# Test feature parsing
go run cmd/adversary-example/main.go

# Start web UI
make serve  # or: go run cmd/wasm/main.go
# Visit http://localhost:8080

# Build everything
go build ./...

# Build WASM
GOARCH=wasm GOOS=js go build -o web/app.wasm cmd/wasm/main.go
```

## Possible Next Steps

1. **Expand Feature Mappings** - Add more features to `data/feature_mappings.json` (currently 33/340)
2. **Improve AI Logic** - More sophisticated feature selection (currently 30% random)
3. **Usage Limits** - Track "once per rest", "once per scene" ability limits
4. **Dynamic Summoning** - Implement FeatureEffectSummonAllies
5. **Parse Tiers 2-4** - Extend feature parsing to higher tier adversaries
6. **Domain Card Parsing** - Similar system for player domain abilities
7. **Performance** - Optimize simulation for 10k+ runs
8. **Advanced Strategies** - Context-aware AI for feature usage

## Key Files to Reference

- `CURRENT_STATE.md` - Full session history and implementation details
- `TODO.md` - Comprehensive feature list and priorities
- `data/feature_mappings.json` - Feature quantization mappings
- `internal/engine/combat_engine.go` - Main combat loop
- `internal/engine/feature_resolver.go` - Feature effect resolution
