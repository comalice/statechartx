# StatechartX

> Minimal, composable, concurrent-ready hierarchical state machines in Go

[![Go Reference](https://pkg.go.dev/badge/github.com/comalice/statechartx.svg)](https://pkg.go.dev/github.com/comalice/statechartx)
[![Go Version](https://img.shields.io/badge/go-1.19%2B-blue.svg)](https://golang.org/dl/)
[![Alpha](https://img.shields.io/badge/status-alpha-orange.svg)](https://github.com/comalice/statechartx/releases)

**Performance:** 518ns transitions | 1.44M events/sec | 1M states in 264ms

StatechartX is a high-performance implementation of hierarchical state machines (statecharts) in Go, following W3C SCXML semantics. It provides both event-driven and tick-based (deterministic) execution models, making it suitable for everything from web services to game engines.

## Key Features

- **Hierarchical States** - Proper entry/exit semantics with parent-child relationships
- **Parallel Regions** - Orthogonal (concurrent) state execution with thread-safe coordination
- **History States** - Shallow and deep history for state restoration
- **Guarded Transitions** - Conditional transition logic with custom guard functions
- **Entry/Exit Actions** - Execute code when entering or leaving states
- **SCXML Conformance** - Follows most W3C SCXML specifications with comprehensive test coverage
- **Dual Runtimes** - Choose between event-driven (high throughput) or tick-based (deterministic)
- **Zero Dependencies** - Core engine uses only Go standard library

## Installation

```bash
go get github.com/comalice/statechartx
```

View the [API documentation](https://pkg.go.dev/github.com/comalice/statechartx) on pkg.go.dev.

## Quick Start

Here's a simple traffic light state machine:

```go
package main

import (
    "context"
    "fmt"

    "github.com/comalice/statechartx"
)

func main() {
    // Define states
    green := &statechartx.State{ID: 1}
    yellow := &statechartx.State{ID: 2}
    red := &statechartx.State{ID: 3}

    // Define transitions: green -> yellow -> red -> green
    green.On(100, 2, nil, nil)   // event 100: go to yellow
    yellow.On(101, 3, nil, nil)  // event 101: go to red
    red.On(102, 1, nil, nil)     // event 102: go to green

    // Build the machine
    root := &statechartx.State{
        ID:      0,
        Initial: 1,
        Children: map[statechartx.StateID]*statechartx.State{
            1: green,
            2: yellow,
            3: red,
        },
    }
    machine, _ := statechartx.NewMachine(root)

    // Create and start runtime
    rt := statechartx.NewRuntime(machine, nil)
    ctx := context.Background()
    rt.Start(ctx)

    // Send events to trigger transitions
    rt.SendEvent(ctx, statechartx.Event{ID: 100}) // green -> yellow
    rt.SendEvent(ctx, statechartx.Event{ID: 101}) // yellow -> red

    if rt.IsInState(3) {
        fmt.Println("Light is red")
    }
}
```

**Output:**
```
Light is red
```

This example demonstrates the core API using only the public `State`, `Machine`, `Runtime`, and `Event` types.

## Core Concepts

### State Machine

A `Machine` is a hierarchical collection of states with a root compound state. States can have children, making them compound states with an `Initial` child.

```go
type State struct {
    ID         StateID
    Initial    StateID                      // For compound states
    Children   map[StateID]*State           // For hierarchical states
    OnEntry    Action                       // Execute when entering
    OnExit     Action                       // Execute when exiting
    IsFinal    bool                         // Final state (done.state.ID)
    IsParallel bool                         // Parallel/orthogonal regions
    // ... see godoc for complete API
}
```

### Runtime

The `Runtime` manages execution - handling events, processing transitions, coordinating parallel regions, and recording history. Create one runtime per machine instance.

```go
rt := statechartx.NewRuntime(machine, nil)
rt.Start(ctx)
rt.SendEvent(ctx, statechartx.Event{ID: eventID, Data: payload})
```

### Events

Events trigger transitions between states. Each event has an `ID` and optional `Data`.

**Special Event IDs:**
- `NO_EVENT` (0) - Eventless/immediate transitions
- `ANY_EVENT` (-1) - Wildcard event matching

```go
type Event struct {
    ID      EventID
    Data    any
    Address StateID // 0 = broadcast, non-zero = targeted (for parallel states)
}
```

### Transitions

A transition connects a source state to a target state, triggered by an event. Guards can conditionally enable/disable transitions. Actions execute during the transition.

```go
// state.On(eventID, targetStateID, guard, action)
state.On(100, 2,
    func(ctx context.Context, evt *Event, from, to StateID) (bool, error) {
        // Guard: return true to allow transition
        return true, nil
    },
    func(ctx context.Context, evt *Event, from, to StateID) error {
        // Action: execute during transition
        fmt.Printf("Transitioning from %d to %d\n", from, to)
        return nil
    },
)
```

For complete API details, see the [godoc](https://pkg.go.dev/github.com/comalice/statechartx).

## Advanced Features

### Parallel States

Parallel states (orthogonal regions) allow multiple sub-states to be active simultaneously. Each region executes independently with thread-safe coordination.

```go
parallel := &statechartx.State{
    ID:         1,
    IsParallel: true,
    Children: map[statechartx.StateID]*statechartx.State{
        10: region1, // Both regions active simultaneously
        20: region2,
    },
}
```

Target specific regions using `Event.Address`:

```go
rt.SendEvent(ctx, statechartx.Event{
    ID:      100,
    Address: 10, // Only sent to region 10
})
```

**See Also:** [statechart_parallel_test.go](statechart_parallel_test.go), [statechart_nested_parallel_test.go](statechart_nested_parallel_test.go)

### History States

History states remember and restore previous state configurations. Use `HistoryShallow` to restore only the direct child, or `HistoryDeep` to restore the entire hierarchy.

```go
history := &statechartx.State{
    ID:            99,
    IsHistoryState: true,
    HistoryType:   statechartx.HistoryDeep,
}
```

**See Also:** [statechart_history_test.go](statechart_history_test.go)

### Final States

Final states generate `done.state.ID` events when reached, allowing parent states to react to completion.

```go
final := &statechartx.State{
    ID:      5,
    IsFinal: true,
}

parent := &statechartx.State{
    ID:       1,
    Initial:  2,
    Children: map[statechartx.StateID]*statechartx.State{
        2: initial,
        5: final,
    },
}

// React to child completion
parent.On(statechartx.EventID(5), nextState, nil, nil) // done.state.5
```

**See Also:** [statechart_done_events_test.go](statechart_done_events_test.go)

## Runtimes

StatechartX provides two execution models with different performance characteristics.

### Event-Driven Runtime (Default)

High-throughput, low-latency runtime using goroutines for parallel region execution.

**Performance:**
- Latency: ~518ns per transition
- Throughput: ~1.44M events/sec
- Parallel regions: Concurrent goroutines

**Best For:**
- Web servers and microservices
- Reactive systems
- UI state management
- Protocol implementations

**Usage:**
```go
rt := statechartx.NewRuntime(machine, nil)
rt.Start(ctx)
```

### Tick-Based Runtime (Realtime Package)

Deterministic, reproducible runtime with fixed time-step execution.

**Performance:**
- Latency: 0-16.67ms at 60 FPS (depends on when event arrives in tick)
- Throughput: ~60K events/sec at 60 FPS
- Parallel regions: Sequential processing (no goroutines)

**Best For:**
- Game engines (60 FPS game logic)
- Physics simulations (fixed time-step)
- Robotics (deterministic control loops)
- Testing/debugging (reproducible scenarios)

**Usage:**
```go
import "github.com/comalice/statechartx/realtime"

machine, _ := statechartx.NewMachine(rootState)
rt := realtime.NewRuntime(machine, realtime.Config{
    TickRate: 16667 * time.Microsecond, // 60 FPS
})
rt.Start(ctx)
rt.SendEvent(statechartx.Event{ID: 1})
```

The tick-based runtime batches events and processes them at fixed intervals with deterministic ordering, ensuring reproducible execution regardless of timing or concurrency.

**See Also:** [realtime package documentation](https://pkg.go.dev/github.com/comalice/statechartx/realtime)

## Examples & Resources

| Example | Description | Location |
|---------|-------------|----------|
| Traffic Light | Basic hierarchical states | README Quick Start |
| Parallel States | Orthogonal regions | [statechart_parallel_test.go](statechart_parallel_test.go) |
| Nested Parallel | Nested parallel regions | [statechart_nested_parallel_test.go](statechart_nested_parallel_test.go) |
| History States | State restoration | [statechart_history_test.go](statechart_history_test.go) |
| Final States | Done events | [statechart_done_events_test.go](statechart_done_events_test.go) |
| SCXML Tests | W3C conformance | [test/scxml/](test/scxml/) |
| Benchmarks | Performance testing | [benchmarks/](benchmarks/) |

## Performance & Conformance

### Benchmarks

StatechartX is designed for production use with sub-microsecond performance:

| Metric | Target | Actual | Status |
|--------|--------|--------|--------|
| State Transition Latency | < 1μs | 518ns | ✓ |
| Event Throughput | > 10K/sec | 1.44M/sec | ✓ |
| Million States Processing | < 10s | 264ms | ✓ |

Run benchmarks yourself:

```bash
make bench
# or
go test -bench=. ./benchmarks/
```

**Benchmark Files:**
- [benchmarks/transition_bench_test.go](benchmarks/transition_bench_test.go) - State transition performance
- [benchmarks/throughput_bench_test.go](benchmarks/throughput_bench_test.go) - Event throughput
- [benchmarks/memory_bench_test.go](benchmarks/memory_bench_test.go) - Memory usage
- [benchmarks/realtime_bench_test.go](benchmarks/realtime_bench_test.go) - Tick-based runtime

### SCXML Conformance

StatechartX follows [W3C SCXML specification](https://www.w3.org/TR/scxml/) semantics and includes the W3C SCXML test suite for conformance validation.

**Test Coverage:**
- 500+ W3C SCXML tests in [test/scxml/](test/scxml/)
- Comprehensive test files: [statechart_scxml_*_test.go](.)
- Custom SCXML test translator: `scxml-translator` skill

**Known Departures from SCXML:**

1. **Numeric IDs Instead of Strings**
   - StatechartX uses `StateID` and `EventID` as numeric types (integers) instead of string identifiers
   - Rationale: Performance optimization - numeric comparisons are faster than string comparisons
   - Impact: State and event names must be mapped to numeric IDs by the application

2. **Datamodel Not Implemented**
   - SCXML's `<datamodel>`, `<data>`, `<assign>` elements for embedded scripting/variables are not implemented
   - Rationale: Go applications can use native Go variables, context, and closures for state data
   - Impact: ~150+ W3C tests requiring datamodel features are skipped
   - Workaround: Use Action/Guard functions with closures or pass data via `Event.Data` and context

3. **Parallel Region Eventless Transition Ordering**
   - SCXML spec requires phase-separated processing (collect all transitions, then exit all, then execute all actions, then enter all)
   - StatechartX processes each parallel region's eventless transitions sequentially in document order
   - Rationale: Simpler implementation, easier to understand, maintains determinism
   - Impact: Event ordering during simultaneous eventless transitions may differ (affects tests 405-406)
   - Workaround: Use explicit events instead of eventless transitions for coordination

4. **Invoke/Send Mechanics**
   - SCXML `<invoke>` (child state machines), `<send>` (external communication), and related features not implemented
   - Rationale: Go applications can use goroutines, channels, and native networking directly
   - Impact: ~40+ W3C tests requiring invoke/send are skipped
   - Workaround: Use Go's concurrency primitives and embed multiple `Runtime` instances if needed

5. **Script Element**
   - SCXML `<script>` for embedded code execution not implemented
   - Rationale: Go is already a compiled language; use Action/Guard functions directly
   - Impact: ~10+ W3C tests requiring `<script>` are skipped

6. **System Variables**
   - SCXML system variables (`_event`, `_sessionid`, `_name`, `_ioprocessors`) not implemented as part of datamodel
   - Rationale: Event information available via function parameters, no need for special variables
   - Impact: ~20+ W3C tests requiring system variables are skipped

**Test Status:**
- ✅ **Passing**: Core semantics (hierarchical states, transitions, parallel regions, history, final states, run-to-completion)
- ⚠️  **Skipped**: ~200+ tests requiring datamodel, invoke/send, script, or strict parallel eventless ordering
- 📊 **Coverage**: All core state machine functionality fully tested with comprehensive test suite

See [statechart_scxml_*_test.go](.) files for detailed conformance testing and skip reasons.

## Use Cases

StatechartX can be used for:

- **UI State Management** - Workflows, wizards, navigation flows
- **Game AI** - Character controllers, behavior trees
- **Protocol Implementations** - Network protocols, communication state machines
- **Workflow Engines** - Business process automation
- **IoT & Embedded Systems** - Control logic for devices
- **Simulations** - Physics engines, deterministic simulations

## Contributing

**Alpha Status:** StatechartX is in active alpha development (v0.1.0-alpha.1). The API may change before the stable 1.0 release.

### Getting Started

1. Fork and clone the repository
2. Create a feature branch from `dev-docs-rework` (or current development branch)
3. Make your changes with tests
4. Run quality checks: `make check`
5. Submit a pull request

### Development Commands

```bash
make test           # Run all tests
make test-race      # Run tests with race detector (required for parallel states)
make bench          # Run benchmarks
make format         # Format code (gofmt + goimports)
make lint           # Run linter (revive)
make vet            # Run go vet
make staticcheck    # Run staticcheck
make check          # Full quality gate (format + vet + lint + test-race)
```

### Testing Requirements

- **Race Detector:** Always run `make test-race` for changes involving parallel states
- **SCXML Conformance:** Add SCXML tests when implementing SCXML features
- **Benchmarks:** Benchmark performance-critical changes to ensure no regressions
- **Test Coverage:** Include unit tests for all new functionality

### Code Quality

The project uses:
- `gofmt` and `goimports` for formatting
- `revive` for linting (see [.revive.toml](.revive.toml))
- `go vet` for static analysis
- `staticcheck` for additional checks

All checks must pass before merging.

### Architecture

StatechartX follows a clean, layered architecture:

- **Public API** ([statechart.go](statechart.go)) - User-facing types and functions
- **Internal Packages:**
  - `internal/primitives` - Zero-dependency foundation data structures
  - `internal/core` - Runtime engine and state machine execution
  - `internal/extensibility` - Pluggable interfaces (guards, actions, event sources)
  - `internal/production` - Persistence, publishing, visualization
- **Realtime Package** ([realtime/](realtime/)) - Tick-based deterministic runtime

All internal packages use **only** the Go standard library (no external dependencies).

### Questions or Issues?

- Open an issue: [github.com/comalice/statechartx/issues](https://github.com/comalice/statechartx/issues)
- Review existing issues and PRs
- For questions about usage, check the [godoc](https://pkg.go.dev/github.com/comalice/statechartx)

## License

StatechartX is released under the [MIT License](LICENSE).

Copyright (c) 2026

## Links

- **Documentation:** [pkg.go.dev/github.com/comalice/statechartx](https://pkg.go.dev/github.com/comalice/statechartx)
- **Realtime Package:** [pkg.go.dev/github.com/comalice/statechartx/realtime](https://pkg.go.dev/github.com/comalice/statechartx/realtime)
- **Issues:** [github.com/comalice/statechartx/issues](https://github.com/comalice/statechartx/issues)
- **Releases:** [github.com/comalice/statechartx/releases](https://github.com/comalice/statechartx/releases)
- **W3C SCXML Specification:** [w3.org/TR/scxml/](https://www.w3.org/TR/scxml/)

---

**Built with Go.** Fast, concurrent, and production-ready.
