# StatechartX

> Minimal, composable, concurrent-ready hierarchical state machine implementation in Go

## Features

- **Hierarchical States** - Nested state support with proper entry/exit order
- **History States** - Both shallow and deep history preservation
- **Guarded Transitions** - Conditional transitions with guards and actions
- **Parallel States** - Independent concurrent regions with automatic synchronization
- **Thread-Safe** - Concurrent event dispatch via mutex protection
- **Real-Time Runtime** - Tick-based deterministic execution for games and simulations
- **SCXML Conformance** - Validated against W3C SCXML test suite
- **Lightweight** - ~1,552 LOC core implementation with minimal dependencies

## Installation

```bash
go get github.com/comalice/statechartx
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/comalice/statechartx"
)

func main() {
    // Create states
    root := &statechartx.State{ID: 1, Initial: 2}
    idle := &statechartx.State{ID: 2, Parent: root}
    active := &statechartx.State{ID: 3, Parent: root}

    // Build hierarchy
    root.Children = map[statechartx.StateID]*statechartx.State{
        2: idle,
        3: active,
    }

    // Add transition: "activate" event moves from idle → active
    idle.Transitions = []*statechartx.Transition{
        {Event: 10, Target: 3},
    }

    // Create and start runtime
    machine, _ := statechartx.NewMachine(root)
    rt := statechartx.NewRuntime(machine, nil)

    ctx := context.Background()
    rt.Start(ctx)

    // Send events
    rt.SendEvent(ctx, statechartx.Event{ID: 10})

    fmt.Println("State machine running!")
}
```

For more examples, see the [examples/](examples/) directory.

## Documentation

- [Architecture Overview](docs/architecture.md) - System design and key concepts
- [Real-Time Runtime](docs/realtime-runtime.md) - Tick-based deterministic execution
- [Performance Testing](docs/performance.md) - Benchmarks and optimization
- [SCXML Conformance](docs/scxml-conformance.md) - W3C test suite integration
- [Examples](examples/README.md) - Runnable code examples

## Performance

StatechartX exceeds all performance targets:

| Metric | Target | Actual | Status |
|--------|--------|--------|--------|
| State Transition | < 1 μs | 518 ns | ✅ 1.9x faster |
| Event Throughput | > 10K/sec | 1.44M/sec | ✅ 144x faster |
| Million States | < 10s | 264 ms | ✅ 37x faster |
| Parallel Regions | < 5s | 3.8 ms | ✅ 1,300x faster |

See [docs/performance.md](docs/performance.md) for full benchmarks.

## Real-Time Runtime

For deterministic execution in games, physics simulations, and robotics:

```go
import "github.com/comalice/statechartx/realtime"

machine, _ := statechartx.NewMachine(rootState)
rt := realtime.NewRuntime(machine, realtime.Config{
    TickRate:         16667 * time.Microsecond, // 60 FPS
    MaxEventsPerTick: 1000,
})

rt.Start(ctx)
defer rt.Stop()

// Events are batched and processed at fixed tick intervals
rt.SendEvent(statechartx.Event{ID: 1})
```

See [realtime/README.md](realtime/README.md) for details.

## Development

```bash
# Run tests
make test

# Run with race detector (recommended for parallel states)
make test-race

# Run benchmarks
make bench

# Full validation
make check  # format, vet, staticcheck, lint, test-race
```

## Project Status

- ~1,552 LOC core implementation
- 13 test suites (basic, parallel, history, done events, SCXML conformance, stress tests)
- W3C SCXML conformance validated
- Thread-safe with race detector validation
- Production-ready

## License

[Choose appropriate license - MIT recommended]

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.
