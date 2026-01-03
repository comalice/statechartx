# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**statechartx** is a minimal, composable, concurrent-ready hierarchical state machine implementation in Go (~1,333 LOC). It provides:

- Hierarchical state nesting with proper entry/exit order
- Initial states and shallow/deep history support
- Guarded transitions with actions
- Thread-safe event dispatch via `sync.RWMutex`
- Explicit composition for concurrent state machines via goroutines/channels
- Parallel state support with independent region execution
- Real-time deterministic runtime for games, physics, and robotics

## Key Architecture

### Core Types (statechart.go)

- **State**: Hierarchical state node with ID, parent/children, transitions, initial/history states, entry/exit actions
- **Runtime**: Executable state machine instance managing active state configuration, thread-safe event processing
- **Transition**: Event-triggered edges with optional guards and actions
- **Event**: Struct with ID, data, and optional address for targeted delivery to parallel regions
- **Machine**: CompoundState wrapper with helper functions for chart evaluation

### Key Concepts

1. **Hierarchical State Management**: States form a tree. `Runtime.current` (or active configuration) tracks all active states (compound states + their active leaf descendants).

2. **Entry/Exit Order**: Transitions compute LCA (Lowest Common Ancestor) to exit states bottom-up and enter states top-down, preserving SCXML-like semantics.

3. **History States**: Both shallow and deep history supported. Parent states remember their last active child/descendants via history mechanism.

4. **Parallel States**: States marked with `IsParallel: true` spawn independent regions that execute concurrently. Each region runs in its own goroutine with separate event channels.

5. **Concurrency Model**: Single `Runtime` uses mutex for thread-safe `SendEvent`. For orthogonal regions in parallel states, the runtime manages multiple goroutines internally. For completely independent state machines, run multiple `Runtime` instances.

6. **Extended State**: User context passed to all actions/guards via function parameters for application-specific data.

### File Structure

```
statechart.go                  - Core state machine implementation (~1,333 lines)
statechart_test.go             - Unit tests for basic functionality
statechart_scxml_*_test.go     - SCXML conformance tests (grouped by test number ranges)
statechart_*_test.go           - Specialized tests (parallel, history, done events, stress, etc.)
realtime/                      - Tick-based deterministic runtime (~230 lines)
  runtime.go                   - RealtimeRuntime implementation
  event.go, tick.go           - Event batching and tick management
  README.md                    - Real-time runtime documentation
builder/                       - Fluent builder API (functional options pattern)
  helpers.go                   - Builder implementation
  README.md                    - Builder documentation
testutil/                      - Test utilities and adapters
examples/                      - Example implementations
  realtime/                    - Game loop, physics sim, replay examples
benchmarks/                    - Performance benchmarking
cmd/
  scxml_dowloader/             - Downloads W3C SCXML test suite
  examples/basic/              - Basic usage example
doc/                          - Design docs, implementation summaries, performance reports
```

## Development Commands

### Building
```bash
go build ./...
```

### Testing
```bash
make test           # Run all tests
make test-race      # Run with race detector (STRONGLY RECOMMENDED for parallel state work)
go test -v ./...    # Verbose test output
go test -v -run TestName ./...  # Run specific test
```

### Test Coverage
```bash
make coverage       # Generates coverage.html
```

### Linting & Analysis
```bash
make lint           # Run revive linter
make vet            # Run go vet
make staticcheck    # Run staticcheck
make check          # All-in-one: format, vet, staticcheck, lint, test-race
```

### Formatting
```bash
make format         # Run gofmt and goimports
```

### Benchmarking
```bash
make bench          # Run benchmarks with memory stats
```

### Fuzzing
```bash
make fuzz           # Run fuzz tests for 30s each
```

### SCXML Test Suite
```bash
# Download W3C SCXML conformance tests
go run cmd/scxml_dowloader/main.go

# Force re-download
go run cmd/scxml_dowloader/main.go -f

# Download to custom directory
go run cmd/scxml_dowloader/main.go --filepath ./tests
```

## SCXML Conformance Testing

The project includes a custom skill (`scxml-translator`) for translating W3C SCXML test cases into Go unit tests:

- **Test Source**: W3C SCXML IRP test suite (downloaded via scxml_dowloader)
- **Target**: Generate tests in `statechart_scxml_*_test.go` files (grouped by test number ranges)
- **Approach**: Map SCXML `<state>`, `<transition>`, `<onentry>` to equivalent Go `State` trees
- **Validation**: Use `conf:pass` states to assert correct final configuration

### Translation Mapping

| SCXML Element | statechartx Equivalent |
|---------------|------------------------|
| `<state id="s1">` | `&State{ID: StateID(hashString("s1"))}` |
| `<transition event="e" target="s2"/>` | `Transitions: []*Transition{{Event: EventID(hashString("e")), Target: StateID(hashString("s2"))}}` |
| `<onentry><raise event="foo"/></onentry>` | `EntryAction: func(ctx, evt, from, to) error { return rt.SendEvent(ctx, Event{ID: EventID(hashString("foo"))}) }` |
| `initial="s0"` attribute | `Initial: StateID(hashString("s0"))` |
| `conf:pass` final state | Assert `rt.IsInState(StateID(hashString("pass")))` |

### Limitations

- No datamodel/ECMAScript expressions (stub guards where needed)
- No `<invoke>`, `<send>`, or external communication beyond internal event raising
- Tests grouped in files by number ranges (100-199, 200-299, etc.) for maintainability

## Code Patterns

### Creating a State Machine

```go
import "github.com/comalice/statechartx"

// Using raw State construction
root := &State{ID: 1}
idle := &State{ID: 2, Parent: root}
active := &State{ID: 3, Parent: root}
root.Children = map[StateID]*State{2: idle, 3: active}
root.Initial = 2

idle.Transitions = []*Transition{
    {Event: 10, Target: 3},  // "activate" event
}

machine, err := NewMachine(root)
if err != nil {
    log.Fatal(err)
}

rt := NewRuntime(machine, nil)
ctx := context.Background()
rt.Start(ctx)
rt.SendEvent(ctx, Event{ID: 10})
```

### Using the Builder API

```go
import "github.com/comalice/statechartx/builder"

// Fluent builder with functional options
idle := builder.New("IDLE",
    builder.WithEntry(func(ctx context.Context, evt *Event, from, to StateID) error {
        log.Println("Entering IDLE")
        return nil
    }),
    builder.On("START", "RUNNING"),
)

root := builder.Composite("ROOT",
    idle,
    builder.New("RUNNING",
        builder.On("STOP", "IDLE"),
        builder.On("ERROR", "FAULT", builder.WithGuard(isRecoverable)),
    ),
)
```

### Parallel States

```go
// Create a parallel state with multiple concurrent regions
parallel := &State{
    ID:         1,
    IsParallel: true,
    Children:   make(map[StateID]*State),
}

// Add independent regions
region1 := &State{ID: 10, Parent: parallel, Initial: 11}
region2 := &State{ID: 20, Parent: parallel, Initial: 21}
parallel.Children[10] = region1
parallel.Children[20] = region2

// Runtime automatically manages region goroutines and synchronization
machine, _ := NewMachine(parallel)
rt := NewRuntime(machine, nil)
rt.Start(ctx)

// Events can be broadcast or targeted to specific regions
rt.SendEvent(ctx, Event{ID: 100})                    // Broadcast to all regions
rt.SendEvent(ctx, Event{ID: 100, Address: 10})       // Target region1 only
```

### Guards and Actions

```go
transition := &Transition{
    Event:  100,
    Target: 5,
    Guard: func(ctx context.Context, evt *Event, from, to StateID) (bool, error) {
        // Access event data for decision logic
        if data, ok := evt.Data.(*MyData); ok {
            return data.IsValid(), nil
        }
        return false, nil
    },
    Action: func(ctx context.Context, evt *Event, from, to StateID) error {
        // Perform side effects during transition
        log.Printf("Transitioning from %d to %d", from, to)
        return nil
    },
}
```

### Real-Time Deterministic Runtime

```go
import "github.com/comalice/statechartx/realtime"

// Create tick-based runtime for games/physics (60 FPS)
machine, _ := statechartx.NewMachine(rootState)

rt := realtime.NewRuntime(machine, realtime.Config{
    TickRate:         16667 * time.Microsecond, // 60 FPS
    MaxEventsPerTick: 1000,
})

ctx := context.Background()
rt.Start(ctx)
defer rt.Stop()

// Send events (non-blocking, batched for next tick)
rt.SendEvent(statechartx.Event{ID: 1})

// Query state (reads from last completed tick - deterministic)
currentState := rt.GetCurrentState()
tickNum := rt.GetTickNumber()
```

## Testing Strategy

1. **Unit Tests** (`statechart_test.go`): Core Runtime behavior, transitions, guards, history
2. **Parallel Tests** (`statechart_parallel_test.go`, `statechart_nested_parallel_test.go`): Concurrent region execution
3. **Race Detection**: **ALWAYS use `make test-race`** for parallel state work - this is critical
4. **SCXML Conformance**: W3C test suite translation validates standards compliance
5. **Stress Tests** (`statechart_stress_test.go`): High load and concurrent access validation
6. **Benchmarks**: Performance testing for transition speed and memory allocation

## Real-Time Runtime

The `realtime` package provides a tick-based deterministic runtime ideal for:

- **Game engines** - Fixed 60 FPS game logic
- **Physics simulations** - Deterministic time-step integration
- **Robotics** - Predictable control loops
- **Testing & debugging** - Reproducible scenarios

Key differences from event-driven runtime:
- Events batched and processed at fixed tick intervals
- Guaranteed deterministic execution order
- Higher latency (~16.67ms @ 60 FPS) but predictable
- Lower throughput (~60K events/sec @ 60 FPS) vs event-driven (~2M events/sec)
- No goroutines per region - sequential processing for determinism

See `realtime/README.md` for detailed API and examples.

## Custom Skills Available

- **golang-development**: Go best practices, testing guidance, fuzzing reference
- **scxml-translator**: SCXML test suite translation to Go tests
- **context-gathering**: Codebase exploration for large projects
- **skill-builder**: Creating new custom skills

## Important Implementation Details

### State ID System
States and events use numeric IDs (`StateID` and `EventID` are `int` types). For SCXML tests, string names are hashed to IDs using `hashString()` helper function.

### Parallel State Execution
- Each region in a parallel state runs in its own goroutine
- Regions communicate via channels managed by the runtime
- Region lifecycle (start/stop) is managed automatically
- Timeouts protect against deadlocks (configurable via DefaultEntryTimeout, DefaultExitTimeout, etc.)
- **CRITICAL**: Always test parallel states with race detector (`make test-race`)

### History State Behavior
- Shallow history: Restores direct child only
- Deep history: Restores entire sub-hierarchy
- History pseudo-states have `IsHistoryState: true` and specify `HistoryType`
- `HistoryDefault` specifies fallback state when no history exists

### Microstep Loop Protection
The runtime limits microstep iterations (eventless transitions) to `MAX_MICROSTEPS` (100) to prevent infinite loops.
