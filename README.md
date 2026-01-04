# Daggerheart Combat Simulator

Monte Carlo combat simulation for Daggerheart RPG. Analyze encounter balance by running thousands of simulated combats with configurable AI strategies.

## Features

### Core Combat Engine
- **Duality Dice System**: d12 Hope vs d12 Fear with Spirit mechanics
- **Damage Thresholds**: Minor/Major/Severe tiers
- **Hope/Fear Tokens**: Resource pools for special abilities
- **Stress Mechanics**: Fully stressed = disadvantage
- **Spotlight System**: Token-based turn selection (configurable threshold)
- **Conditions**: Burning, poisoned, cloaked, advantage, disadvantage, etc.
- **Resistances**: Damage type resistances (physical, magic, fire, frost, poison)

### AI Strategies
- **Greedy**: Aggressive resource usage, focus fire on weakest targets
- **Conservative**: Hoards resources, clears conditions immediately
- **Desperate**: Context-aware, uses resources when low HP
- **Random**: Unpredictable behavior

### Simulation Features
- Monte Carlo analysis (100-10,000 simulations)
- Win rate statistics
- Average combat duration (turns/rounds)
- Configurable player and enemy strategies
- Results visualization with charts

## Quick Start

### Using Make (Recommended)

```bash
# Show all available commands
make help

# Run the CLI example
make run-example

# Build and run the web server
make serve

# Generate static pages for GitHub Pages
make static-pages
```

### Command Line Example

```bash
# Or run directly with Go
go run cmd/example/main.go
```

Output:
```
Daggerheart Combat Simulator - Example
=======================================

Combat Setup:
  Players: Grimjaw the Rogue (HP: 6), Thora the Fighter (HP: 8)
  Enemies: Goblin Archer (HP: 4), Goblin Warrior (HP: 4), Goblin Boss (HP: 7)
  Simulations: 1000

Results:
========
Simulations: 1000 | Player Wins: 981 (98.1%) | Enemy Wins: 19 (1.9%) | Avg Turns: 14.7 | Avg Rounds: 2.6
```

### Web UI (WASM)

```bash
# Build and run the web server
make serve

# Or manually:
# go build -o bin/wasm-server ./cmd/wasm/
# ./bin/wasm-server
```

Open http://localhost:8080 in your browser.

## Deployment

### GitHub Pages

Generate static files and deploy to GitHub Pages:

```bash
# Generate static website in docs/ directory
make static-pages

# Commit and push
git add docs/
git commit -m "Update GitHub Pages"
git push

# Enable GitHub Pages in repository settings:
# Settings → Pages → Source: Deploy from branch → Branch: main → /docs
```

The app will be available at `https://yourusername.github.io/daggerheart-combat-simulator/`

## Project Structure

```
daggerheart-combat-simulator/
├── internal/
│   ├── models/          # Core data structures
│   │   ├── combatant.go
│   │   ├── ability.go
│   │   ├── condition.go
│   │   └── combat_state.go
│   ├── engine/          # Combat resolution
│   │   ├── dice.go
│   │   ├── resolver.go
│   │   ├── effects.go
│   │   ├── spotlight.go
│   │   └── combat_engine.go
│   ├── strategy/        # AI decision making
│   │   └── decision_maker.go
│   └── simulation/      # Monte Carlo runner
│       └── runner.go
├── ui/                  # WASM UI (go-app)
│   ├── app.go
│   ├── components/
│   │   ├── combatant_list.go
│   │   ├── config_form.go
│   │   └── results_summary.go
│   └── pages/
│       └── simulator.go
├── cmd/
│   ├── example/         # CLI example
│   └── wasm/            # Web server
└── web/
    └── static/          # CSS and assets
```

## Usage

### Building Characters

Characters have:
- **Name**: Identifier
- **HP Max**: Total hit points
- **Evasion**: Difficulty to hit (vs duality roll)
- **Proficiency**: Bonus to rolls and damage
- **Stress Max**: Maximum stress (usually 6)
- **Armor Slots**: Hits absorbed before HP damage
- **Abilities**: Special attacks/skills
- **Resistances**: Damage reduction by type

### Creating Abilities

Abilities are declarative:

```go
ability := &models.Ability{
    Name: "Sneak Attack",
    Trigger: models.TriggerManual,
    TargetSingle: true,
    TargetEnemies: true,
    Costs: models.ResourceCost{
        Stress: 1,
    },
    Conditions: []models.AbilityCondition{
        {
            Type: models.ConditionAllyInMelee,
        },
    },
    Effects: []models.Effect{
        models.DamageEffect{
            DiceCount: 3,
            DiceSize: 6,
            FlatBonus: 2,
            DamageType: models.DamagePhysical,
        },
    },
}
```

### Configuring Simulations

```go
config := simulation.SimulationConfig{
    RunCount: 1000,
    PlayerTemplates: []*models.Combatant{rogue, fighter},
    EnemyTemplates: []*models.Combatant{goblin1, goblin2},
    InitialHopePool: 5,
    InitialFearPool: 5,
    SpotlightThreshold: 3,
    PlayerStrategy: strategy.DefaultDesperateConfig(),
    EnemyStrategy: strategy.DefaultGreedyConfig(),
    MaxTurnsPerCombat: 100,
}

runner := simulation.NewSimulationRunner(config)
result, _ := runner.Run()
```

## Development Status

**Completed:**
- ✅ Core combat engine
- ✅ Data models
- ✅ AI strategies
- ✅ Monte Carlo simulation
- ✅ WASM UI (basic)
- ✅ Character creation forms
- ✅ Config forms
- ✅ Results visualization

**TODO:**
- ⬜ Ability builder UI
- ⬜ Export to CSV/PDF
- ⬜ Advanced charts
- ⬜ Save/load encounters

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed design documentation.

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make help` | Show all available commands |
| `make build` | Build all binaries (WASM + example) |
| `make build-wasm` | Build WASM binary only |
| `make build-server` | Build web server binary |
| `make build-example` | Build CLI example binary |
| `make run-example` | Build and run CLI example |
| `make serve` | Build and start web server |
| `make static-pages` | Generate static site for GitHub Pages |
| `make clean` | Remove all build artifacts |
| `make test` | Run all tests |
| `make fmt` | Format all Go code |
| `make lint` | Run linter (golangci-lint or go vet) |
| `make deps` | Download and tidy dependencies |
| `make all` | Clean, deps, build, and test |

## Requirements

- Go 1.21+
- Modern web browser (for WASM UI)
- Make (optional, for convenience commands)

## License

MIT
