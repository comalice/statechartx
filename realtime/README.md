# StatechartX Real-Time Runtime

A tick-based deterministic runtime for StatechartX state machines.

## Overview

The real-time runtime provides fixed time-step execution with guaranteed determinism, making it ideal for:

- **Game engines** - 60 FPS game logic
- **Physics simulations** - Fixed time-step integration
- **Robotics** - Deterministic control loops  
- **Testing & debugging** - Reproducible scenarios

## Architecture

The `RealtimeRuntime` embeds the existing `statechartx.Runtime` and reuses all core state transition logic (~430 lines). Only the event dispatch mechanism is replaced with tick-based batching (~230 lines of new code).

**Key Benefits:**
- ✅ Zero code duplication
- ✅ Consistent behavior with event-driven runtime
- ✅ Easy maintenance
- ✅ Predictable performance

## Usage

```go
package main

import (
    "context"
    "time"
    
    "github.com/comalice/statechartx"
    "github.com/comalice/statechartx/realtime"
)

func main() {
    // Create state machine
    machine, _ := statechartx.NewMachine(rootState)
    
    // Create tick-based runtime (60 FPS)
    rt := realtime.NewRuntime(machine, realtime.Config{
        TickRate: 16667 * time.Microsecond, // 60 FPS
    })
    
    // Start runtime
    ctx := context.Background()
    rt.Start(ctx)
    defer rt.Stop()
    
    // Send events (non-blocking, batched for next tick)
    rt.SendEvent(statechartx.Event{ID: 1})
    
    // Query state (reads from last completed tick)
    currentState := rt.GetCurrentState()
}
```

## API

### Constructor

```go
func NewRuntime(machine *statechartx.Machine, cfg Config) *RealtimeRuntime
```

### Configuration

```go
type Config struct {
    TickRate         time.Duration // e.g., 16.67ms for 60 FPS
    MaxEventsPerTick int           // Queue capacity (default: 1000)
}
```

### Methods

```go
// Lifecycle
func (rt *RealtimeRuntime) Start(ctx context.Context) error
func (rt *RealtimeRuntime) Stop() error

// Event Sending (non-blocking)
func (rt *RealtimeRuntime) SendEvent(event statechartx.Event) error
func (rt *RealtimeRuntime) SendEventWithPriority(event statechartx.Event, priority int) error

// State Queries (reads from last completed tick)
func (rt *RealtimeRuntime) GetCurrentState() statechartx.StateID
func (rt *RealtimeRuntime) IsInState(stateID statechartx.StateID) bool
func (rt *RealtimeRuntime) GetTickNumber() uint64
```

## Event Ordering

Events are ordered deterministically using:
1. **Priority** - Higher priority processed first
2. **Sequence number** - FIFO for same priority
3. **Stable sorting** - Preserves relative order

This guarantees that given the same sequence of `SendEvent()` calls, the state machine will always execute identically.

## Performance

### At 60 FPS (16.67ms tick rate):
- **Throughput:** ~60,000 events/second
- **Latency:** 0-16.67ms (depends on when event arrives in tick)
- **Memory:** O(max_events_per_tick)

### At 1000 Hz (1ms tick rate):
- **Throughput:** ~1,000,000 events/second
- **Latency:** 0-1ms

## Comparison with Event-Driven Runtime

| Feature | Event-Driven | Tick-Based |
|---------|--------------|------------|
| **Latency** | ~217ns | ~16.67ms @ 60 FPS |
| **Throughput** | ~2M events/sec | ~60K events/sec @ 60 FPS |
| **Determinism** | Best-effort | Guaranteed |
| **Use Cases** | Web servers, microservices, UI | Games, physics, robotics, testing |

## Examples

See the `examples/realtime/` directory for complete examples:

- **game_loop.go** - 60 FPS game state management
- **physics_sim.go** - 1000 Hz physics simulation
- **replay.go** - Deterministic replay scenario

## Testing

```bash
# Run realtime tests
go test ./realtime

# Run benchmarks
go test -bench=. ./benchmarks

# Run example
go run examples/realtime/game_loop.go
```

## Implementation Notes

- The runtime reuses `processEvent()`, `processMicrosteps()`, `computeLCA()`, and all other core methods from the embedded runtime
- Parallel regions are processed sequentially (no goroutines) for determinism
- Panic recovery is built into the tick loop
- Thread-safe event batching with mutex protection

## License

Same as StatechartX project.
