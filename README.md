# Daggerheart Combat Simulator

Monte Carlo combat simulation for the Daggerheart TTRPG. Run thousands of simulated combats to analyze encounter balance with configurable AI strategies.

Built with Go and WebAssembly ([go-app/v10](https://github.com/maxence-charriere/go-app)).

## Features

### Combat Engine
- **Duality Dice** -- Players roll 2d12 (Hope + Fear), adversaries roll d20
- **Hope/Fear Pools** -- Shared resource pools (Hope capped at 6, Fear at 12)
- **Damage Thresholds** -- Minor/Major/Severe tiers with level/tier scaling and custom thresholds
- **Armor System** -- Slot-based threshold reduction (Severe->Major->Minor->None)
- **Stress** -- Overflow to HP damage, fully stressed applies Vulnerable
- **Spotlight System** -- Player phase / GM turn loop with Fear spending and Relentless(X)
- **Conditions** -- Burning, poisoned, vulnerable, resistant, stunned, restrained, advantage, disadvantage, taunted, hidden
- **Resistances/Immunities** -- Physical and magic damage types, feature-based

### Feature System
- **Adversary Features** -- Active/passive abilities with Fear or stress costs
- **Domain Abilities** -- Player abilities with Hope or stress costs
- **24 Effect Types** -- Damage, healing, conditions, AOE, summoning, tokens, armor, stress, and more
- **DC Checks** -- None, target defense, or fixed DC with success probability
- **Token Mechanics** -- Place/spend tokens on abilities (e.g. Unleash Chaos pattern)
- **Trigger System** -- on_attack, on_hit, on_critical, on_damage_taken, start_turn, end_turn
- **Usage Limits** -- Per-scene, per-session, per-rest tracking

### AI Strategies
- **Greedy** -- Aggressive resource spending, focus fire
- **Conservative** -- Hoards resources, clears conditions early
- **Desperate** -- Spends resources only when low HP
- **Random** -- Unpredictable behavior
- **Clever** -- Context-aware decisions based on combat state

Each strategy configures resource usage, ability usage, condition clearing, target selection, GM spotlight seizure, and Fear spending for additional adversary turns.

### Simulation
- Monte Carlo analysis (configurable run count)
- Win rate, average turns/rounds, min/max tracking
- Deterministic replay via seed
- Per-strategy comparison

## Quick Start

```bash
make help       # Show all available commands
make serve      # Build WASM + start web server at http://localhost:8080
make test       # Run all tests
```

### Web UI

```bash
make serve
```

Open http://localhost:8080. Create players and adversaries, configure strategies, run simulations.

### Static Deployment

```bash
make static-pages   # Generate static site in docs/
```

Deploy `docs/` to GitHub Pages or any static host.

### Programmatic Usage

```go
config := simulation.DefaultSimulationConfig()
config.RunCount = 1000
config.PlayerTemplates = []*models.Combatant{
    models.NewCombatant("p1", "Rogue", models.SidePlayer, 3, 12, 2, 15, 6),
}
config.PlayerTemplates[0].DamageDice = "2d6+2"
config.EnemyTemplates = []*models.Combatant{
    models.NewAdversary("e1", "Goblin", "Tier 1", "Standard", 10, 3, 8, 4, []int{4, 7}),
}
config.EnemyTemplates[0].DamageDice = "1d6"

runner := simulation.NewSimulationRunner(config)
result, err := runner.Run()
fmt.Println(result.Summary())
```

## Project Structure

```
daggerheart-combat-simulator/
├── internal/
│   ├── models/        # Combatant, features, conditions, combat state, dice notation
│   ├── engine/        # Combat engine, resolver, feature resolver, dice, spotlight
│   ├── strategy/      # AI decision maker with pluggable strategy configs
│   ├── parser/        # JSON adversary/feature parsers, adversary registry
│   └── simulation/    # Monte Carlo runner
├── ui/
│   ├── app.go
│   ├── components/    # WASM UI forms and display components
│   └── pages/
├── cmd/
│   ├── wasm/          # Web server entry point
│   └── static-gen/    # Static site generator
├── data/              # Embedded adversary JSON and feature mappings
└── web/static/        # CSS and assets
```

## Makefile

| Command | Description |
|---|---|
| `make serve` | Build WASM + server, start at localhost:8080 |
| `make build` | Build all binaries |
| `make build-wasm` | Build WASM binary only |
| `make static-pages` | Generate static site for deployment |
| `make test` | Run all tests |
| `make fmt` | Format all Go code |
| `make lint` | Run linter |
| `make clean` | Remove build artifacts |
| `make all` | Clean, deps, build, test |

## Requirements

- Go 1.26+
- Modern web browser (for WASM UI)

## License

MIT
