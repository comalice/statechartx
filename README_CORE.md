# StatechartX Core Package

Event-driven hierarchical state machine runtime for Go.

## Overview

The core `statechartx` package provides the foundation for building hierarchical state machines (statecharts) in Go. It implements SCXML-compliant state machine semantics with support for:

- **Hierarchical states** - Nested state hierarchies with proper entry/exit ordering
- **Guarded transitions** - Conditional state changes based on predicates
- **Actions** - Executable code during transitions and state entry/exit
- **Parallel states** - Concurrent orthogonal regions
- **History states** - Shallow and deep history for state restoration
- **Final states** - Completion detection with done events

For deterministic tick-based execution, see the [realtime package](realtime/README.md).

## Quick Start

```go
package main

import (
    "context"
    "github.com/comalice/statechartx"
)

func main() {
    // Define states
    idle := &statechartx.State{ID: 1}
    active := &statechartx.State{ID: 2}

    // Add transition: idle -> active on event 100
    idle.On(100, 2, nil, nil)

    // Build machine with hierarchy
    root := &statechartx.State{
        ID:       0,
        Initial:  1,
        Children: map[statechartx.StateID]*statechartx.State{
            1: idle,
            2: active,
        },
    }
    machine, _ := statechartx.NewMachine(root)

    // Create and start runtime
    rt := statechartx.NewRuntime(machine, nil)
    ctx := context.Background()
    rt.Start(ctx)
    defer rt.Stop()

    // Send event to trigger transition
    rt.SendEvent(ctx, statechartx.Event{ID: 100})

    // Check current state
    if rt.IsInState(2) {
        println("Now in active state")
    }
}
```

## Core API

### State Machine Components

#### State
Represents a node in the state hierarchy. States can be atomic (leaf), compound (with children), parallel, or final.

```go
type State struct {
    ID            StateID                    // Unique identifier
    Parent        *State                     // Parent state (nil for root)
    Children      map[StateID]*State         // Child states (for compound states)
    Initial       StateID                    // Initial child state ID
    Transitions   []*Transition              // Outgoing transitions
    EntryAction   Action                     // Executed on state entry
    ExitAction    Action                     // Executed on state exit
    IsFinal       bool                       // Final state marker
    IsParallel    bool                       // Parallel state marker (advanced)
    IsHistoryState bool                      // History pseudo-state (advanced)
    HistoryType    HistoryType               // Shallow or Deep (advanced)
}
```

#### Transition
Defines a state change triggered by an event, with optional guard and action.

```go
type Transition struct {
    Event  EventID  // Triggering event (0 = eventless/immediate)
    Source *State   // Source state
    Target StateID  // Target state (0 = internal transition)
    Guard  Guard    // Conditional predicate (nil = always true)
    Action Action   // Transition action (nil = none)
}
```

#### Machine
Top-level state container with validation and state lookup.

```go
machine, err := statechartx.NewMachine(rootState)
// Returns error if hierarchy has cycles, duplicate IDs, or missing children
```

#### Runtime
Manages state machine execution with event queue and transition processing.

```go
rt := statechartx.NewRuntime(machine, nil)
rt.Start(ctx)              // Enter initial state, spawn event loop
rt.SendEvent(ctx, event)   // Queue event for async processing
rt.IsInState(stateID)      // Check if state is active
rt.Stop()                  // Graceful shutdown
```

## Building State Machines

### Pattern 1: Simple Hierarchy

Most basic pattern - parent state with child states and transitions.

```go
// Define state IDs
const (
    RootID   statechartx.StateID = 0
    IdleID   statechartx.StateID = 1
    ActiveID statechartx.StateID = 2
)

// Create states
root := &statechartx.State{ID: RootID, Initial: IdleID}
idle := &statechartx.State{ID: IdleID, Parent: root}
active := &statechartx.State{ID: ActiveID, Parent: root}

// Build hierarchy
root.Children = map[statechartx.StateID]*statechartx.State{
    IdleID:   idle,
    ActiveID: active,
}

// Add transition with On() helper
idle.On(100, ActiveID, nil, nil)  // Event 100: idle -> active

machine, _ := statechartx.NewMachine(root)
```

### Pattern 2: Entry/Exit Actions

Execute code when entering or exiting states.

```go
idle := &statechartx.State{ID: IdleID}
idle.OnEntry(func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
    fmt.Println("Entering idle state")
    return nil
})
idle.OnExit(func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
    fmt.Println("Exiting idle state")
    return nil  // Return error to abort transition
})
```

**Execution order**: Exit previous state → Transition action → Enter new state

### Pattern 3: Guarded Transitions

Conditionally enable/disable transitions based on runtime state.

```go
// Guard function checks if transition is allowed
var count int
guard := func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) (bool, error) {
    return count >= 5, nil  // Only allow transition if count >= 5
}

// Add guarded transition
idle.Transitions = append(idle.Transitions, &statechartx.Transition{
    Event:  100,
    Target: ActiveID,
    Guard:  guard,
})
```

**Note**: If guard returns `(false, nil)`, transition is blocked. If `(false, error)`, error is propagated.

### Pattern 4: Transition Actions

Execute code during the transition itself (between exit and entry).

```go
action := func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
    fmt.Printf("Transitioning from %d to %d\n", from, to)
    // Access event data
    if evt.Data != nil {
        fmt.Printf("Event data: %v\n", evt.Data)
    }
    return nil  // Return error to abort transition
}

idle.On(100, ActiveID, nil, &action)
```

### Pattern 5: Event Data

Pass data with events for use in guards and actions.

```go
// Send event with data
rt.SendEvent(ctx, statechartx.Event{
    ID:   100,
    Data: map[string]interface{}{"count": 42, "user": "alice"},
})

// Access in action
action := func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
    data := evt.Data.(map[string]interface{})
    fmt.Printf("Count: %v, User: %v\n", data["count"], data["user"])
    return nil
}
```

### Pattern 6: Nested Hierarchies

Multi-level state nesting for complex state machines.

```go
// Three-level hierarchy: root -> operational -> (idle, active)
root := &statechartx.State{ID: 0, Initial: 10}
operational := &statechartx.State{ID: 10, Parent: root, Initial: 11}
idle := &statechartx.State{ID: 11, Parent: operational}
active := &statechartx.State{ID: 12, Parent: operational}
error := &statechartx.State{ID: 20, Parent: root}

root.Children = map[statechartx.StateID]*statechartx.State{
    10: operational,
    20: error,
}
operational.Children = map[statechartx.StateID]*statechartx.State{
    11: idle,
    12: active,
}

// Transition from nested state to top-level state
active.On(999, 20, nil, nil)  // error event -> error state
// Runtime will exit: active -> operational -> root -> enter error
```

**LCA (Lowest Common Ancestor)**: The runtime computes the LCA to determine which states to exit/enter. Transitions within the same parent are cheaper than cross-branch transitions.

### Pattern 7: Internal Transitions

Execute transitions without exiting/entering states (target = 0).

```go
// Internal transition - no exit/entry actions fire
idle.On(200, 0, nil, &action)  // Target 0 = internal

// vs External transition - exit/entry actions fire
idle.On(201, IdleID, nil, &action)  // Target = self, full exit/re-entry
```

**Use case**: Update internal state or trigger side effects without state change overhead.

## Advanced Patterns

### Parallel States

Execute multiple orthogonal regions concurrently. Each child region runs independently with its own state configuration.

```go
// Create parallel state with two regions
parallel := &statechartx.State{
    ID:         100,
    IsParallel: true,
    Children: map[statechartx.StateID]*statechartx.State{
        101: region1Root,  // Region 1 hierarchy
        102: region2Root,  // Region 2 hierarchy
    },
}

// Each region has its own state machine
region1Root := &statechartx.State{ID: 101, Initial: 111}
region1Idle := &statechartx.State{ID: 111, Parent: region1Root}
region1Active := &statechartx.State{ID: 112, Parent: region1Root}
region1Root.Children = map[statechartx.StateID]*statechartx.State{
    111: region1Idle,
    112: region1Active,
}

// Region 2 similar...

// Target specific region with Event.Address
rt.SendEvent(ctx, statechartx.Event{
    ID:      200,
    Address: 101,  // Send only to region 1
})

// Broadcast to all regions
rt.SendEvent(ctx, statechartx.Event{
    ID:      201,
    Address: 0,  // 0 = broadcast
})
```

**Threading**: By default, each region runs in its own goroutine. Use `make test-race` to detect data races. For sequential processing (determinism), see [realtime package](realtime/README.md).

### History States

Record and restore previous state configurations.

#### Shallow History
Restores only the immediate child state.

```go
history := &statechartx.State{
    ID:             50,
    IsHistoryState: true,
    HistoryType:    statechartx.HistoryShallow,
    HistoryDefault: IdleID,  // Default if no history exists
    Parent:         root,
}
root.Children[50] = history

// After active state was visited, transitioning to history restores active
idle.On(100, ActiveID, nil, nil)   // idle -> active
active.On(101, OtherID, nil, nil)  // active -> other
other.On(102, 50, nil, nil)        // other -> history (restores active)
```

#### Deep History
Restores the entire state hierarchy path.

```go
deepHistory := &statechartx.State{
    ID:             51,
    IsHistoryState: true,
    HistoryType:    statechartx.HistoryDeep,
    HistoryDefault: IdleID,
    Parent:         root,
}

// Restores full path, e.g., operational.active.processing
```

**Gotcha**: Deep history can overwrite shallow history if both are present. See [parallel_state_implementation_status.md](docs/parallel_state_implementation_status.md) for details.

### Final States and Done Events

Mark states as final to trigger completion detection.

```go
success := &statechartx.State{
    ID:             30,
    IsFinal:        true,
    FinalStateData: map[string]interface{}{"result": "ok"},
}

// When runtime enters final state, it generates a done event
// Event ID: DoneEventID(parentStateID)
doneEventID := statechartx.DoneEventID(root.ID)

// Parent can react to child completion
root.On(doneEventID, NextStateID, nil, nil)
```

**Use case**: Workflow completion, async operation signaling, composite state completion.

### Eventless Transitions

Trigger transitions immediately without waiting for external events.

```go
// NO_EVENT constant (0) means immediate/eventless transition
idle.On(statechartx.NO_EVENT, ActiveID, &guard, nil)

// Processed during microstep loop after state entry
// Useful for: state refinement, conditional routing, cleanup
```

**Microstep limit**: MAX_MICROSTEPS = 100. If exceeded, runtime stops to prevent infinite loops. See [SCXML_EVENTLESS_TRANSITION_SEMANTICS.md](docs/SCXML_EVENTLESS_TRANSITION_SEMANTICS.md).

### Wildcard Events

Catch-all transitions for any event.

```go
// ANY_EVENT constant (-1) matches all events
errorState.On(statechartx.ANY_EVENT, IdleID, nil, nil)

// Useful for error recovery, logging, default handlers
```

## Performance Characteristics

From `docs/performance.md`:

| Metric | Value | Notes |
|--------|-------|-------|
| Event throughput | ~2M events/sec | Event-driven runtime |
| Transition latency | ~217ns | Median, simple transitions |
| Memory per machine | ~1KB | Excludes user state |
| Goroutines per parallel region | 1 | Default implementation |

**For deterministic fixed time-step**: See [realtime package](realtime/README.md) (60K events/sec @ 60 FPS).

## Common Pitfalls

### 1. Race Conditions in Parallel States
**Problem**: Parallel regions run in separate goroutines; shared state access can race.

**Solution**: Always run tests with race detector:
```bash
make test-race
```

Use synchronization (mutexes) or message passing for shared data.

### 2. Microstep Infinite Loops
**Problem**: Eventless transitions with guards that always return true create loops.

**Symptom**: "MAX_MICROSTEPS exceeded" error after 100 iterations.

**Solution**: Ensure eventless transition guards eventually return false, or use regular events.

### 3. Deep History Overwriting Shallow
**Problem**: If both shallow and deep history are used, deep history restoration can overwrite shallow.

**Solution**: Use one history type per hierarchy branch. See [docs/hooks_implementation_status.md](docs/hooks_implementation_status.md).

### 4. Forgetting to Call Stop()
**Problem**: Runtime goroutines leak if `Stop()` not called.

**Solution**: Always defer stop:
```go
rt.Start(ctx)
defer rt.Stop()
```

### 5. Nil Guard/Action Pointers
**Problem**: Passing `&guard` when guard is nil causes panic.

**Solution**: Pass nil directly:
```go
idle.On(100, ActiveID, nil, nil)  // Correct
// NOT: idle.On(100, ActiveID, &guard, &action) when nil
```

Use the helper method `State.On()` which handles nil correctly.

## API Reference

### Types

- `StateID` - Unique state identifier (int)
- `EventID` - Event type identifier (int, special: NO_EVENT=0, ANY_EVENT=-1)
- `Event` - Event with ID, Data, and Address (for parallel states)
- `Action` - Function executed during transitions/entry/exit
- `Guard` - Predicate function for conditional transitions
- `State` - State node with hierarchy, transitions, actions
- `Transition` - Event-triggered state change with guard/action
- `Machine` - Top-level state machine with validation
- `Runtime` - Execution engine with event queue

### Functions

- `NewMachine(root *State) (*Machine, error)` - Create and validate machine
- `NewRuntime(machine *Machine, hooks *ParallelStateHooks) *Runtime` - Create runtime

### Runtime Methods

- `Start(ctx context.Context) error` - Enter initial state, spawn event loop
- `Stop() error` - Graceful shutdown, wait for goroutines
- `SendEvent(ctx context.Context, event Event) error` - Queue event
- `IsInState(stateID StateID) bool` - Check if state active

### State Methods

- `OnEntry(action Action)` - Set entry action
- `OnExit(action Action)` - Set exit action
- `On(event EventID, target StateID, guard *Guard, action *Action)` - Add transition

### Machine Methods

- `GetState(stateID StateID) *State` - Lookup state by ID

## Examples

See [examples/](examples/) for runnable code:

- **[examples/basic](examples/basic/)** - Hierarchy, transitions, actions
- **[examples/realtime/game_loop](examples/realtime/game_loop/)** - 60 FPS game logic
- **[examples/realtime/physics_sim](examples/realtime/physics_sim/)** - Deterministic physics
- **[examples/realtime/replay](examples/realtime/replay/)** - Event recording/replay

For test-driven examples of advanced features:
- Parallel states: `statechart_parallel_test.go`, `statechart_nested_parallel_test.go`
- History: `statechart_history_test.go`
- Final states: `statechart_done_events_test.go`

## Further Reading

- [Architecture](docs/architecture.md) - Core design and algorithms
- [SCXML Conformance](docs/scxml-conformance.md) - W3C test suite results
- [Performance](docs/performance.md) - Benchmarks and stress tests
- [Realtime Package](realtime/README.md) - Tick-based deterministic runtime
- [SCXML Eventless Semantics](docs/SCXML_EVENTLESS_TRANSITION_SEMANTICS.md) - Microstep details

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development workflow, code style, and testing requirements.
