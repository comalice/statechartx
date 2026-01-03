# StatechartX Examples

This directory contains runnable examples demonstrating various features of the StatechartX state machine library.

## Basic Example

**Location**: [basic/main.go](basic/main.go)

**Description**: Simple state machine demonstrating core concepts including:
- State hierarchy creation
- Transition configuration
- Event dispatching
- Entry/exit actions
- Basic runtime usage

**Run**:
```bash
go run examples/basic/main.go
```

**What it demonstrates**:
- Creating a hierarchical state machine
- Defining states with parent-child relationships
- Adding transitions between states
- Sending events to trigger transitions
- Using entry and exit actions

## Real-Time Examples

All real-time examples use the tick-based deterministic runtime from the `realtime/` package, providing fixed time-step execution ideal for games, simulations, and robotics.

### Game Loop

**Location**: [realtime/game_loop/game_loop.go](realtime/game_loop/game_loop.go)

**Description**: 60 FPS game state management demonstrating:
- Tick-based state machine for game states (Menu, Playing, Paused, GameOver)
- Fixed time-step execution (16.67ms per tick)
- Deterministic event processing
- Frame-locked state transitions

**Run**:
```bash
go run examples/realtime/game_loop/game_loop.go
```

**Use Cases**:
- Game state management
- Menu systems
- Game session control
- Deterministic gameplay logic

### Physics Simulation

**Location**: [realtime/physics_sim/physics_sim.go](realtime/physics_sim/physics_sim.go)

**Description**: Fixed time-step physics simulation demonstrating:
- Deterministic physics state machine
- Collision detection states
- Time-step integration
- Reproducible simulation results

**Run**:
```bash
go run examples/realtime/physics_sim/physics_sim.go
```

**Use Cases**:
- Physics engines
- Simulation systems
- Rigid body dynamics
- Deterministic physics

### Replay System

**Location**: [realtime/replay/replay.go](realtime/replay/replay.go)

**Description**: Deterministic replay functionality demonstrating:
- Event recording during gameplay
- Deterministic replay of recorded events
- Tick-based synchronization
- State verification

**Run**:
```bash
go run examples/realtime/replay/replay.go
```

**Use Cases**:
- Game replays
- Testing and debugging
- Demo recording
- Tournament systems

## Example Categories

### By Feature

- **Hierarchical States**: basic, all real-time examples
- **Parallel States**: (Future example - see test files for now)
- **History States**: (Future example - see test files for now)
- **Guarded Transitions**: basic
- **Entry/Exit Actions**: basic, all real-time examples

### By Runtime Type

- **Event-Driven**: basic
- **Tick-Based (Real-Time)**: game_loop, physics_sim, replay

### By Use Case

- **Learning**: basic
- **Games**: game_loop, replay
- **Simulations**: physics_sim
- **Robotics**: (Future examples - use real-time runtime)

## Running All Examples

```bash
# Run all examples sequentially
for example in basic realtime/game_loop realtime/physics_sim realtime/replay; do
    echo "Running $example..."
    go run examples/$example/*.go
    echo "---"
done
```

## Example Code Structure

All examples follow a similar structure:

```go
package main

import (
    "context"
    "github.com/comalice/statechartx"
    // Additional imports...
)

func main() {
    // 1. Create state hierarchy
    root := &statechartx.State{...}
    // Build state tree...

    // 2. Create machine
    machine, err := statechartx.NewMachine(root)
    if err != nil {
        panic(err)
    }

    // 3. Create runtime (event-driven or real-time)
    rt := statechartx.NewRuntime(machine, nil)
    // OR: rt := realtime.NewRuntime(machine, config)

    // 4. Start runtime
    ctx := context.Background()
    rt.Start(ctx)
    defer rt.Stop()

    // 5. Send events and observe behavior
    rt.SendEvent(ctx, statechartx.Event{...})
}
```

## Contributing Examples

We welcome additional examples! When contributing:

1. **Focus on one concept** - Keep examples focused on demonstrating specific features
2. **Include comments** - Explain what each section does
3. **Provide output** - Show expected console output in comments
4. **Keep it simple** - Avoid unnecessary complexity
5. **Add to this README** - Document your example in the appropriate section

See [../CONTRIBUTING.md](../CONTRIBUTING.md) for full contribution guidelines.

## Additional Resources

- [Main Documentation](../docs/README.md)
- [Architecture Overview](../docs/architecture.md)
- [Real-Time Runtime Guide](../docs/realtime-runtime.md)
- [API Reference](https://pkg.go.dev/github.com/comalice/statechartx)

## Learning Path

**Recommended order for learning**:

1. Start with [basic/](basic/) - Understand core concepts
2. Read [Architecture Overview](../docs/architecture.md) - Grasp the design
3. Try [game_loop/](realtime/game_loop/) - See real-time runtime
4. Explore [physics_sim/](realtime/physics_sim/) - Understand determinism
5. Check [replay/](realtime/replay/) - Learn advanced patterns
6. Read test files in root - See comprehensive usage

## Questions?

- Check the [documentation](../docs/)
- Review test files in the project root
- Open an issue for clarification
- See [CONTRIBUTING.md](../CONTRIBUTING.md) for help
