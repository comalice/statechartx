# StatechartX Decision Guide

Quick reference for choosing between runtime modes, state patterns, and implementation strategies.

## Runtime Selection

### Event-Driven vs Real-Time Runtime

Choose the runtime that matches your application's requirements:

| Consideration | Event-Driven (Default) | Real-Time (Tick-Based) |
|--------------|------------------------|------------------------|
| **Best for** | Web servers, APIs, microservices, UI state management | Games, simulations, robotics, replay systems |
| **Event processing** | Asynchronous, as-received | Batched at fixed tick intervals |
| **Throughput** | ~2M events/sec | ~60K events/sec @ 60 FPS |
| **Latency** | ~217ns (median) | 0-16.67ms @ 60 FPS (depends on tick) |
| **Determinism** | Best-effort (goroutine scheduling) | **Guaranteed** (sequence numbers, stable sort) |
| **Parallel regions** | Concurrent (goroutines) | Sequential (deterministic order) |
| **Memory** | ~1KB per runtime | ~2KB per runtime (double buffering) |
| **Use when** | Low latency critical | Reproducibility critical |
| **Race detection** | Run `make test-race` | Not needed (sequential) |

**Decision flowchart:**
```
Need reproducibility/replay? → YES → Real-Time
                             ↓ NO
Need <1μs latency?          → YES → Event-Driven
                             ↓ NO
Game/simulation?            → YES → Real-Time
                             ↓ NO
Default: Event-Driven
```

See [realtime/README.md](../realtime/README.md) for tick-based details.

## State Pattern Selection

### Parallel vs Sequential States

| Pattern | Use When | Example |
|---------|----------|---------|
| **Parallel states** | Orthogonal concerns running independently | Audio system (playback + recording), UI (theme + layout), Robot (navigation + sensors) |
| **Sequential states** | Mutually exclusive states | Login flow (idle → authenticating → authenticated), Traffic light (red → yellow → green) |

**Implementation:**
```go
// Parallel: IsParallel = true
parallel := &statechartx.State{
    IsParallel: true,
    Children: map[statechartx.StateID]*statechartx.State{
        101: audioRegion,
        102: videoRegion,
    },
}

// Sequential: Regular hierarchy
sequential := &statechartx.State{
    Initial: idle,
    Children: map[statechartx.StateID]*statechartx.State{
        1: idle,
        2: active,
    },
}
```

**Gotchas:**
- Parallel regions run in separate goroutines (event-driven) or sequentially (real-time)
- Use `make test-race` to detect data races in parallel state machines
- Event targeting: `Address: 0` broadcasts, non-zero targets specific region

### History: Shallow vs Deep

| Type | Restores | Use When | Example |
|------|----------|----------|---------|
| **Shallow** | Direct child state only | Single-level undo | Editor mode (text → visual), Tabbed UI (last active tab) |
| **Deep** | Entire state path | Multi-level undo | Wizard (step 1.2.3 → step 2.1.1), Nested nav (section → subsection → item) |
| **None** | N/A | No restoration needed | One-way flows, stateless operations |

**Implementation:**
```go
// Shallow history
shallowHistory := &statechartx.State{
    ID:             10,
    IsHistoryState: true,
    HistoryType:    statechartx.HistoryShallow,
    HistoryDefault: idleState,  // If no history exists
}

// Deep history
deepHistory := &statechartx.State{
    ID:             11,
    IsHistoryState: true,
    HistoryType:    statechartx.HistoryDeep,
    HistoryDefault: idleState,
}
```

**Gotcha**: Deep history can overwrite shallow history if both present in same hierarchy. Use one type per branch.

## Transition Pattern Selection

### Guarded vs Unguarded Transitions

| Pattern | Use When | Example |
|---------|----------|---------|
| **Guarded** | Conditional logic needed | Count threshold, permission check, state validation |
| **Unguarded** | Always allow transition | UI button clicks, timeout events, error recovery |

```go
// Guarded
guard := func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) (bool, error) {
    return count >= 5, nil  // Only transition if count >= 5
}
idle.On(100, active, &guard, nil)

// Unguarded
idle.On(100, active, nil, nil)
```

### Eventless vs Event-Triggered

| Type | Trigger | Use When | Example |
|------|---------|----------|---------|
| **Eventless** (NO_EVENT) | Immediate (microstep loop) | State refinement, conditional routing | Auto-advance wizard, Error recovery |
| **Event-triggered** | External event | User action, timer, network | Button click, Timeout, API response |

```go
// Eventless - triggers immediately on entry
idle.On(statechartx.NO_EVENT, active, &guard, nil)

// Event-triggered
idle.On(100, active, nil, nil)
```

**Gotcha**: Eventless transitions with always-true guards create infinite loops. MAX_MICROSTEPS = 100 prevents runaway.

### Internal vs External Transitions

| Type | Target | Exit/Entry Actions | Use When |
|------|--------|-------------------|----------|
| **Internal** | `0` | **Not executed** | Update state without full exit/re-entry | Self-transition without overhead |
| **External** | Same state ID | **Executed** | Full state reset needed | Re-initialize state on self-loop |

```go
// Internal transition - no exit/entry actions
idle.On(200, 0, nil, &action)  // Target = 0

// External self-transition - exit/re-entry
idle.On(201, idle.ID, nil, &action)  // Full cycle
```

## Event Design Patterns

### Event Addressing for Parallel States

| Address | Behavior | Use When |
|---------|----------|----------|
| `0` (broadcast) | All regions receive | Global events (pause, reset), State sync |
| Non-zero (targeted) | Specific region only | Region-specific events (audio mute, video play) |

```go
// Broadcast to all regions
rt.SendEvent(ctx, statechartx.Event{
    ID:      100,
    Address: 0,  // Broadcast
})

// Target specific region
rt.SendEvent(ctx, statechartx.Event{
    ID:      101,
    Address: audioRegionID,  // Targeted
})
```

### Event Data Patterns

| Pattern | Use When | Example |
|---------|----------|---------|
| **Typed struct** | Structured data | User actions, API responses |
| **Map/interface{}** | Dynamic data | Generic handlers, JSON passthrough |
| **No data (nil)** | Signal-only events | Simple state changes, acks |

```go
// Typed struct
type LoginEvent struct {
    Username string
    Password string
}
rt.SendEvent(ctx, statechartx.Event{ID: 100, Data: LoginEvent{...}})

// Map
rt.SendEvent(ctx, statechartx.Event{
    ID: 101,
    Data: map[string]interface{}{"count": 42},
})

// No data
rt.SendEvent(ctx, statechartx.Event{ID: 102})
```

## Action Design Patterns

### Error Handling in Actions

Actions can return errors to abort transitions:

```go
action := func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
    if err := validateTransition(); err != nil {
        return err  // Aborts transition, stays in current state
    }
    // Perform action
    return nil  // Success, transition proceeds
}
```

**Best practice**: Use error returns for validation, not business logic failures. Business logic should succeed and emit error events.

### Entry vs Exit vs Transition Actions

| Action Type | Timing | Use For | Example |
|-------------|--------|---------|---------|
| **Entry** | After entering state | Initialize resources | Open connection, start timer |
| **Exit** | Before leaving state | Cleanup resources | Close connection, cancel timer |
| **Transition** | Between exit and entry | Business logic | Validate data, log transition |

**Execution order**: Exit source → Transition action → Enter target

## Performance Tuning

### When to Optimize

| Scenario | Optimization | Impact |
|----------|-------------|--------|
| **>100K events/sec** | Profile with `pprof` | CPU bottlenecks |
| **>1M states** | Reduce StateID allocations | Memory pressure |
| **Deep hierarchies (>10 levels)** | Cache LCA computations | CPU (repeated walks) |
| **Parallel states with shared data** | Use channels/mutexes | Race prevention |

See [performance.md](performance.md) for benchmarks.

### Event Queue Sizing

Default queue: 100 events buffered.

**Increase queue size** when:
- Burst traffic (web servers)
- Slow action processing
- High-throughput pipelines

**Keep default** when:
- Low traffic
- Fast actions (<1ms)
- Memory constrained

```go
// Custom queue size (not exposed in current API - use channels directly)
// Current: Hardcoded to 100 in NewRuntime()
```

## Testing Strategies

### Race Detection (Critical for Parallel States)

**Always run** for parallel state machines:
```bash
make test-race
```

**When to suspect races:**
- Parallel states accessing shared variables
- Actions modifying external state
- Custom hooks with goroutines

### Determinism Testing (Real-Time)

Verify reproducibility with event replay:
```go
// Record events
events := []statechartx.Event{...}

// Replay multiple times
for i := 0; i < 10; i++ {
    rt := realtime.NewRuntime(machine, cfg)
    for _, evt := range events {
        rt.SendEvent(evt)
    }
    // Assert same final state
}
```

See [examples/realtime/replay](../examples/realtime/replay/) for full pattern.

## Migration Guides

### From Other State Machine Libraries

Common patterns when migrating:

| From | To StatechartX | Notes |
|------|----------------|-------|
| **XState** | Similar hierarchy model | Guards/actions map 1:1, parallel states supported |
| **Go FSM libraries** | Add hierarchy support | Flat FSMs → nested states with proper LCA |
| **Custom switch-case** | Extract to State structs | Explicit state graph, testable transitions |

See [README_CORE.md](../README_CORE.md) for API patterns.

### From Event-Driven to Real-Time

**Step 1**: Wrap machine in RealtimeRuntime
```go
// Before: Event-driven
rt := statechartx.NewRuntime(machine, nil)

// After: Real-time
rt := realtime.NewRuntime(machine, realtime.Config{
    TickRate: 16667 * time.Microsecond,  // 60 FPS
})
```

**Step 2**: Remove goroutine-based parallelism (automatic in real-time mode)

**Step 3**: Test determinism with replay

See [realtime/README.md](../realtime/README.md#migrating-from-event-driven) for details.

## Common Pitfalls

### Microstep Infinite Loops

**Problem**: Eventless transitions with always-true guards
```go
// BAD: Infinite loop
state.On(statechartx.NO_EVENT, nextState, &alwaysTrueGuard, nil)
```

**Solution**: Ensure guard eventually returns false
```go
// GOOD: Guard changes based on state
var visited bool
guard := func(...) (bool, error) {
    if !visited {
        visited = true
        return true, nil
    }
    return false, nil
}
```

### Forgetting to Call Stop()

**Problem**: Goroutine leaks
```go
// BAD
rt.Start(ctx)
// Forgot to stop
```

**Solution**: Always defer
```go
// GOOD
rt.Start(ctx)
defer rt.Stop()
```

### Nil Pointer Guards/Actions

**Problem**: Passing `&guard` when guard is nil
```go
var guard Guard  // nil
state.On(100, target, &guard, nil)  // PANIC
```

**Solution**: Pass nil directly or use helper
```go
state.On(100, target, nil, nil)  // Correct
```

## Further Reading

- [Core Package Guide](../README_CORE.md) - Comprehensive API patterns
- [Architecture](architecture.md) - System design deep-dive
- [Performance](performance.md) - Benchmarks and limits
- [Real-Time Runtime](../realtime/README.md) - Tick-based execution
- [Examples](../examples/README.md) - Runnable code samples
