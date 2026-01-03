# StatechartX Real-Time Runtime: Architecture Design & Implementation Plan

**Date:** January 2, 2026  
**Status:** Design Document for Discussion  
**Purpose:** Explore tick-based real-time runtime as alternative to event-driven implementation

---

## Executive Summary

This document presents a comprehensive design for a **tick-based real-time runtime** for StatechartX, offering deterministic event ordering guarantees as an alternative to the current high-performance event-driven implementation. The real-time runtime targets use cases requiring temporal consistency, deterministic execution, and frame-locked behavior (games, simulations, robotics, real-time control systems).

**Key Design Principles:**
- **Tick-based execution**: Fixed time-step game loop model
- **Event ordering guarantees**: Deterministic FIFO processing per tick
- **Double-buffered state**: Read from tick N-1, write to tick N
- **No goroutines needed**: Sequential processing within each tick
- **Parallel state support**: Deterministic region processing order
- **Adjacent implementation**: Lives alongside event-driven runtime in `realtime/` package

---

## Part 1: Current Event-Driven Runtime Analysis

### 1.1 Event Ordering Characteristics

**Current Implementation (Event-Driven):**

```go
// Event processing in runtime.go
func (rt *Runtime) eventLoop() {
    defer rt.wg.Done()
    for {
        select {
        case <-rt.ctx.Done():
            return
        case event := <-rt.eventQueue:
            rt.processEvent(event)  // Process immediately as received
        }
    }
}
```

**Event Ordering Behavior:**
- Events processed in **arrival order** from buffered channel (100 capacity)
- **Non-deterministic** when multiple goroutines send events concurrently
- Parallel regions receive events via separate channels with independent ordering
- No guarantee of processing order across parallel regions

**Example Non-Deterministic Scenario:**

```go
// Two goroutines sending events concurrently
go rt.SendEvent(ctx, Event{ID: 1})  // May arrive first or second
go rt.SendEvent(ctx, Event{ID: 2})  // May arrive first or second

// Parallel regions - independent event queues
region1.events <- Event{ID: 1}  // Processed independently
region2.events <- Event{ID: 1}  // No synchronization guarantee
```

### 1.2 Parallel State Event Routing

**Current Architecture:**

```go
type parallelRegion struct {
    stateID      StateID
    events       chan Event      // Independent event queue per region
    done         chan struct{}
    ctx          context.Context
    cancel       context.CancelFunc
    runtime      *Runtime
    currentState StateID
    mu           sync.RWMutex
}

func (rt *Runtime) sendEventToRegions(ctx context.Context, event Event) error {
    if event.Address == 0 {
        // Broadcast to all regions - no ordering guarantee
        for _, region := range rt.parallelRegions {
            select {
            case region.events <- event:  // Concurrent sends
            case <-sendCtx.Done():
                return errors.New("broadcast timeout")
            }
        }
    }
}
```

**Characteristics:**
- Each parallel region runs in its own goroutine
- Independent event queues per region (10 capacity each)
- Broadcast events sent concurrently to all regions
- **No guarantee** which region processes event first
- **No synchronization** between region state updates

### 1.3 Non-Deterministic Scenarios

**Scenario 1: Concurrent Event Submission**
```go
// Multiple sources sending events
go sensor1.SendEvent(ctx, Event{ID: SENSOR_1_DATA})
go sensor2.SendEvent(ctx, Event{ID: SENSOR_2_DATA})
go timer.SendEvent(ctx, Event{ID: TICK})

// Processing order depends on goroutine scheduling
// Result: Non-deterministic state transitions
```

**Scenario 2: Parallel Region Race Conditions**
```go
// Parallel state with shared external state
parallelState := &State{
    IsParallel: true,
    Children: map[StateID]*State{
        REGION_A: regionA,  // Modifies shared counter
        REGION_B: regionB,  // Reads shared counter
    },
}

// Event broadcast to both regions
rt.SendEvent(ctx, Event{ID: UPDATE})

// Race: Which region processes first?
// Region A increments counter, Region B reads counter
// Result: Non-deterministic depending on goroutine scheduling
```

**Scenario 3: Microstep Non-Determinism**
```go
// Eventless transitions after state entry
// Current: Processed immediately in single thread (deterministic)
// But: If entry actions send events, those events are queued
//      and may interleave with external events non-deterministically
```

### 1.4 Current Performance Characteristics

**From Performance Report (January 2, 2026):**

| Metric | Performance | Notes |
|--------|-------------|-------|
| **Event Throughput** | 1.44M - 2M events/sec | Stable across 10K-10M events |
| **State Transition** | 518 ns | Sub-microsecond latency |
| **Event Sending** | 217 ns | Minimal overhead |
| **LCA Computation** | 38 ns (shallow), 4.58 μs (deep) | Efficient hierarchy traversal |
| **Parallel Region Spawn** | 19.7 μs (10 regions) | 50x faster than target |
| **Event Routing (10 regions)** | 3.05 μs | Concurrent delivery |
| **Memory per Machine** | 0.61 KB | Minimal footprint |
| **Concurrent Machines** | 10,000 machines @ 3.4M events/sec | Excellent scalability |

**Bottlenecks (from CPU profiling):**
1. **Memory allocation** (12% CPU) - Go runtime overhead
2. **Channel operations** (10% CPU) - Event queue management
3. **Map lookups** (6.5% CPU) - State/transition lookups

**Strengths:**
- ✅ Exceptional throughput (2M events/sec)
- ✅ Sub-microsecond latency
- ✅ Linear scalability (1M states, 10K parallel regions)
- ✅ No data races detected
- ✅ Minimal memory footprint

**Weaknesses for Real-Time Use Cases:**
- ❌ Non-deterministic event ordering under concurrent load
- ❌ No temporal consistency guarantees
- ❌ Parallel regions process events independently (no sync points)
- ❌ Difficult to reproduce exact execution sequences for debugging
- ❌ No frame-locked behavior for game loops

---

## Part 2: Real-Time Runtime Concept Analysis

### 2.1 Tick-Based Execution Model

**Core Concept:**
```
Tick N-1          Tick N            Tick N+1
┌─────────┐      ┌─────────┐      ┌─────────┐
│ State A │─────▶│ State B │─────▶│ State C │
│ Events: │      │ Events: │      │ Events: │
│  [1,2]  │      │  [3,4]  │      │  [5]    │
└─────────┘      └─────────┘      └─────────┘
    │                 │                 │
    └─Read from───────┴─Write to────────┘
```

**Execution Flow:**
1. **Collect Phase**: Gather all events submitted during tick N-1
2. **Sort Phase**: Order events deterministically (submission order, priority, etc.)
3. **Process Phase**: Execute events sequentially, reading state from tick N-1
4. **Commit Phase**: Write all state changes to tick N atomically
5. **Advance Phase**: Tick N becomes tick N-1, repeat

### 2.2 Event Ordering Guarantees

**Guarantee 1: FIFO Within Source**
- Events from same source processed in submission order
- Example: `SendEvent(E1); SendEvent(E2)` → E1 always before E2

**Guarantee 2: Deterministic Cross-Source Ordering**
- Events from different sources ordered by:
  1. **Priority** (if specified)
  2. **Submission timestamp** (within tick)
  3. **Source ID** (tie-breaker)

**Guarantee 3: Parallel Region Synchronization**
- All parallel regions process events in same order
- Synchronization point at end of each tick
- No region advances to tick N+1 until all complete tick N

### 2.3 Read-From-Previous-Tick Architecture

**Double-Buffered State:**

```go
type TickState struct {
    current    StateID           // Current active state
    history    map[StateID]StateID
    datamodel  map[string]any    // Extended state
    timestamp  time.Time
}

type RealtimeRuntime struct {
    tickN_1    *TickState  // Read-only during tick N
    tickN      *TickState  // Write-only during tick N
    tickNumber uint64
}
```

**Benefits:**
- **No race conditions**: Reads and writes to different buffers
- **Atomic commits**: All changes visible simultaneously at tick boundary
- **Rollback capability**: Can discard tick N and retry
- **Deterministic replay**: Save tick N-1 state, replay events

**Example:**
```go
// Tick N processing
func (rt *RealtimeRuntime) ProcessTick() {
    // Read current state from tick N-1
    currentState := rt.tickN_1.current
    
    // Process all events, writing to tick N
    for _, event := range rt.eventQueue {
        newState := rt.processEvent(event, currentState)
        rt.tickN.current = newState  // Write to tick N
        currentState = newState      // Update local view
    }
    
    // Commit: Swap buffers atomically
    rt.tickN_1, rt.tickN = rt.tickN, rt.tickN_1
    rt.tickNumber++
}
```

### 2.4 Goroutine Usage Analysis

**Question: Are goroutines needed?**

**Option A: No Goroutines (Recommended)**
- Process all parallel regions sequentially within tick
- Deterministic region processing order (e.g., by StateID)
- Simpler implementation, easier debugging
- No synchronization overhead
- **Trade-off**: Cannot utilize multiple CPU cores per tick

**Option B: Goroutines with Barriers**
- Spawn goroutines for parallel regions
- Synchronization barrier at end of tick
- Can utilize multiple cores
- **Trade-off**: More complex, synchronization overhead, harder to debug

**Recommendation: Start with Option A**
- Real-time systems often run on single core anyway (determinism)
- Tick processing must complete within fixed time budget
- Parallelism across multiple state machines (not within one machine)
- Can add goroutines later if profiling shows bottleneck

---

## Part 3: Real-Time Runtime Design

### 3.1 Core Architecture

```go
package realtime

import (
    "context"
    "time"
    "github.com/comalice/statechartx"
)

// TickConfig configures tick-based execution
type TickConfig struct {
    TickRate      time.Duration  // Fixed time step (e.g., 16.67ms for 60 FPS)
    MaxEventsPerTick int         // Event queue capacity per tick
    EventOrdering EventOrderingStrategy
}

type EventOrderingStrategy int

const (
    OrderFIFO EventOrderingStrategy = iota  // Simple FIFO
    OrderPriority                            // Priority + FIFO
    OrderTimestamp                           // Timestamp + FIFO
)

// RealtimeRuntime provides tick-based deterministic execution
type RealtimeRuntime struct {
    machine    *statechartx.Machine
    config     TickConfig
    
    // Double-buffered state
    tickN_1    *TickState
    tickN      *TickState
    tickNumber uint64
    
    // Event queue for current tick
    eventQueue []EventWithMetadata
    
    // Parallel region state
    regionStates map[statechartx.StateID]*RegionState
    
    // Control
    ctx        context.Context
    cancel     context.CancelFunc
    ticker     *time.Ticker
    running    bool
}

// TickState represents complete state at a single tick
type TickState struct {
    current       statechartx.StateID
    regionStates  map[statechartx.StateID]statechartx.StateID
    history       map[statechartx.StateID]statechartx.StateID
    deepHistory   map[statechartx.StateID][]statechartx.StateID
    datamodel     map[string]any
    timestamp     time.Time
    tickNumber    uint64
}

// EventWithMetadata wraps events with ordering metadata
type EventWithMetadata struct {
    Event      statechartx.Event
    Priority   int
    Timestamp  time.Time
    SourceID   uint64
    SequenceNum uint64
}

// RegionState tracks state for a parallel region
type RegionState struct {
    stateID      statechartx.StateID
    currentState statechartx.StateID
    eventQueue   []EventWithMetadata
}
```

### 3.2 Tick-Based Execution Model

```go
// Start begins tick-based execution
func (rt *RealtimeRuntime) Start(ctx context.Context) error {
    if rt.running {
        return errors.New("runtime already running")
    }
    
    rt.ctx, rt.cancel = context.WithCancel(ctx)
    rt.ticker = time.NewTicker(rt.config.TickRate)
    rt.running = true
    
    // Enter initial state (writes to tickN_1)
    if err := rt.enterInitialState(); err != nil {
        return err
    }
    
    // Start tick loop
    go rt.tickLoop()
    
    return nil
}

// tickLoop is the main execution loop
func (rt *RealtimeRuntime) tickLoop() {
    for {
        select {
        case <-rt.ctx.Done():
            rt.ticker.Stop()
            return
            
        case <-rt.ticker.C:
            rt.processTick()
        }
    }
}

// processTick executes one complete tick
func (rt *RealtimeRuntime) processTick() {
    // Phase 1: Collect events submitted since last tick
    events := rt.collectEvents()
    
    // Phase 2: Sort events deterministically
    sortedEvents := rt.sortEvents(events)
    
    // Phase 3: Process events sequentially
    rt.processEvents(sortedEvents)
    
    // Phase 4: Process parallel regions
    rt.processParallelRegions()
    
    // Phase 5: Process microsteps (eventless transitions)
    rt.processMicrosteps()
    
    // Phase 6: Commit state changes (swap buffers)
    rt.commitTick()
    
    // Phase 7: Advance tick counter
    rt.tickNumber++
}
```

### 3.3 Event Ordering Mechanism

```go
// SendEvent queues an event for next tick
func (rt *RealtimeRuntime) SendEvent(event statechartx.Event) error {
    return rt.SendEventWithPriority(event, 0)
}

// SendEventWithPriority queues event with priority
func (rt *RealtimeRuntime) SendEventWithPriority(event statechartx.Event, priority int) error {
    metadata := EventWithMetadata{
        Event:       event,
        Priority:    priority,
        Timestamp:   time.Now(),
        SourceID:    rt.getSourceID(),
        SequenceNum: rt.nextSequenceNum(),
    }
    
    // Thread-safe append to event queue
    rt.mu.Lock()
    defer rt.mu.Unlock()
    
    if len(rt.eventQueue) >= rt.config.MaxEventsPerTick {
        return errors.New("event queue full")
    }
    
    rt.eventQueue = append(rt.eventQueue, metadata)
    return nil
}

// sortEvents orders events deterministically
func (rt *RealtimeRuntime) sortEvents(events []EventWithMetadata) []EventWithMetadata {
    sorted := make([]EventWithMetadata, len(events))
    copy(sorted, events)
    
    switch rt.config.EventOrdering {
    case OrderFIFO:
        // Already in order (append order preserved)
        
    case OrderPriority:
        sort.SliceStable(sorted, func(i, j int) bool {
            if sorted[i].Priority != sorted[j].Priority {
                return sorted[i].Priority > sorted[j].Priority  // Higher priority first
            }
            return sorted[i].SequenceNum < sorted[j].SequenceNum  // FIFO within priority
        })
        
    case OrderTimestamp:
        sort.SliceStable(sorted, func(i, j int) bool {
            if !sorted[i].Timestamp.Equal(sorted[j].Timestamp) {
                return sorted[i].Timestamp.Before(sorted[j].Timestamp)
            }
            return sorted[i].SequenceNum < sorted[j].SequenceNum  // Tie-breaker
        })
    }
    
    return sorted
}
```

### 3.4 Double-Buffered State Implementation

```go
// processEvents processes all events for current tick
func (rt *RealtimeRuntime) processEvents(events []EventWithMetadata) {
    for _, eventMeta := range events {
        rt.processEvent(eventMeta)
    }
}

// processEvent processes single event (reads tickN_1, writes tickN)
func (rt *RealtimeRuntime) processEvent(eventMeta EventWithMetadata) {
    event := eventMeta.Event
    
    // Read current state from tick N-1
    currentState := rt.tickN_1.current
    state := rt.machine.states[currentState]
    
    // Find matching transition (reads from tickN_1)
    transition := rt.pickTransitionHierarchical(state, event)
    if transition == nil {
        return  // No matching transition
    }
    
    // Internal transition
    if transition.Target == 0 {
        if transition.Action != nil {
            // Action can modify datamodel (writes to tickN)
            transition.Action(rt.ctx, &event, currentState, currentState)
        }
        return
    }
    
    // External transition
    from := currentState
    to := transition.Target
    
    // Compute LCA (reads from tickN_1)
    lca := rt.computeLCA(from, to)
    
    // Exit states (reads tickN_1, writes tickN)
    rt.exitToLCA(&event, from, to, lca)
    
    // Execute transition action (writes to tickN)
    if transition.Action != nil {
        transition.Action(rt.ctx, &event, from, to)
    }
    
    // Enter states (reads tickN_1, writes tickN)
    rt.enterFromLCA(&event, from, to, lca)
    
    // Update current state in tickN
    rt.tickN.current = to
    
    // Update local view for next event in this tick
    rt.tickN_1.current = to
}

// commitTick atomically commits all state changes
func (rt *RealtimeRuntime) commitTick() {
    // Swap buffers: tickN becomes tickN_1 for next tick
    rt.tickN_1, rt.tickN = rt.tickN, rt.tickN_1
    
    // Clear tickN (now the write buffer) for next tick
    rt.tickN.timestamp = time.Now()
    rt.tickN.tickNumber = rt.tickNumber + 1
    
    // Note: We don't clear current state, history, etc.
    // They are copied from tickN_1 at start of next tick
}
```

### 3.5 Parallel State Handling (No Goroutines)

```go
// processParallelRegions processes all parallel regions sequentially
func (rt *RealtimeRuntime) processParallelRegions() {
    // Check if current state is parallel
    currentState := rt.tickN_1.current
    state := rt.machine.states[currentState]
    
    if !state.IsParallel {
        return
    }
    
    // Process each region in deterministic order (sorted by StateID)
    regionIDs := rt.getSortedRegionIDs(state)
    
    for _, regionID := range regionIDs {
        rt.processRegion(regionID)
    }
}

// processRegion processes events for a single parallel region
func (rt *RealtimeRuntime) processRegion(regionID statechartx.StateID) {
    regionState := rt.regionStates[regionID]
    if regionState == nil {
        return
    }
    
    // Get events for this region (from broadcast or targeted)
    events := rt.getRegionEvents(regionID)
    
    // Sort events deterministically
    sortedEvents := rt.sortEvents(events)
    
    // Process each event in region context
    for _, eventMeta := range sortedEvents {
        rt.processRegionEvent(regionID, eventMeta)
    }
}

// processRegionEvent processes event in region context
func (rt *RealtimeRuntime) processRegionEvent(regionID statechartx.StateID, eventMeta EventWithMetadata) {
    event := eventMeta.Event
    
    // Read region's current state from tickN_1
    currentState := rt.tickN_1.regionStates[regionID]
    state := rt.machine.states[currentState]
    
    // Find matching transition
    transition := rt.pickTransitionHierarchical(state, event)
    if transition == nil {
        return
    }
    
    // Process transition (similar to main state machine)
    // ... (exit, action, enter logic)
    
    // Write new state to tickN
    rt.tickN.regionStates[regionID] = transition.Target
    
    // Update local view for next event in this region
    rt.tickN_1.regionStates[regionID] = transition.Target
}

// getSortedRegionIDs returns region IDs in deterministic order
func (rt *RealtimeRuntime) getSortedRegionIDs(parallelState *statechartx.State) []statechartx.StateID {
    ids := make([]statechartx.StateID, 0, len(parallelState.Children))
    for id := range parallelState.Children {
        ids = append(ids, id)
    }
    sort.Slice(ids, func(i, j int) bool {
        return ids[i] < ids[j]
    })
    return ids
}
```

### 3.6 API Differences from Event-Driven Runtime

| Feature | Event-Driven Runtime | Real-Time Runtime |
|---------|---------------------|-------------------|
| **Event Submission** | `SendEvent(ctx, event)` - async | `SendEvent(event)` - queued for next tick |
| **Event Processing** | Immediate (goroutine) | Batched per tick |
| **State Query** | `IsInState(id)` - current | `IsInState(id)` - as of last tick |
| **Parallel Regions** | Concurrent goroutines | Sequential processing |
| **Timing** | Real-time, async | Fixed time step |
| **Context** | Per-event context | Per-tick context |
| **Cancellation** | Context cancellation | Stop() method |
| **Determinism** | Best-effort | Guaranteed |

**API Example:**

```go
// Event-Driven Runtime
rt := statechartx.NewRuntime(machine, nil)
rt.Start(ctx)
rt.SendEvent(ctx, statechartx.Event{ID: 1})  // Async, processed immediately

// Real-Time Runtime
rt := realtime.NewRuntime(machine, realtime.TickConfig{
    TickRate: 16 * time.Millisecond,  // 60 FPS
    MaxEventsPerTick: 100,
    EventOrdering: realtime.OrderFIFO,
})
rt.Start(ctx)
rt.SendEvent(statechartx.Event{ID: 1})  // Queued for next tick
```

---

## Part 4: Architecture Comparison

### 4.1 Event-Driven vs Tick-Based Trade-offs

| Aspect | Event-Driven | Tick-Based | Winner |
|--------|--------------|------------|--------|
| **Throughput** | 2M events/sec | ~60K events/sec @ 60 FPS | Event-Driven (33x) |
| **Latency** | 217 ns | 16.67 ms (1 tick @ 60 FPS) | Event-Driven (76,000x) |
| **Determinism** | Non-deterministic | Guaranteed deterministic | Tick-Based |
| **Reproducibility** | Difficult | Easy (save/replay ticks) | Tick-Based |
| **Debugging** | Hard (race conditions) | Easy (step through ticks) | Tick-Based |
| **CPU Utilization** | High (goroutines) | Moderate (sequential) | Event-Driven |
| **Memory** | Low (0.61 KB/machine) | Higher (double buffering) | Event-Driven |
| **Complexity** | Moderate (sync primitives) | Low (sequential logic) | Tick-Based |
| **Real-time Guarantees** | None | Fixed time budget | Tick-Based |
| **Parallel Regions** | True parallelism | Sequential simulation | Event-Driven |

### 4.2 Performance Implications

**Event-Driven Runtime:**
- **Strengths**: Maximum throughput, minimal latency, true parallelism
- **Weaknesses**: Non-deterministic, hard to debug, no temporal guarantees
- **Best for**: High-throughput async workflows, I/O-bound systems, microservices

**Tick-Based Runtime:**
- **Strengths**: Deterministic, reproducible, debuggable, temporal consistency
- **Weaknesses**: Lower throughput, higher latency, no true parallelism
- **Best for**: Games, simulations, robotics, real-time control, testing

**Throughput Calculation:**
```
Event-Driven: 2,000,000 events/sec
Tick-Based @ 60 FPS: 60 ticks/sec × 1000 events/tick = 60,000 events/sec
Tick-Based @ 1000 Hz: 1000 ticks/sec × 100 events/tick = 100,000 events/sec

Ratio: Event-Driven is 20-33x faster
```

**Latency Calculation:**
```
Event-Driven: 217 ns (immediate processing)
Tick-Based @ 60 FPS: 16.67 ms average (0-33.33 ms range)
Tick-Based @ 1000 Hz: 1 ms average (0-2 ms range)

Ratio: Event-Driven is 4,600-76,000x faster
```

### 4.3 Use Case Suitability

**Event-Driven Runtime - Best For:**
1. **Microservices**: High-throughput request processing
2. **IoT Systems**: Async sensor data processing
3. **Workflow Engines**: Business process automation
4. **UI State Management**: Responsive user interfaces
5. **Network Protocols**: Async message handling

**Tick-Based Runtime - Best For:**
1. **Game Engines**: Frame-locked game logic (60 FPS)
2. **Physics Simulations**: Fixed time-step integration
3. **Robotics**: Deterministic control loops (100-1000 Hz)
4. **Real-Time Systems**: Hard real-time guarantees
5. **Testing/Debugging**: Reproducible test scenarios
6. **Multiplayer Games**: Deterministic lockstep networking
7. **Replay Systems**: Record/replay functionality

### 4.4 Determinism Guarantees

**Event-Driven Runtime:**
- ❌ Event ordering non-deterministic under concurrent load
- ❌ Parallel regions process independently (no sync)
- ❌ Goroutine scheduling non-deterministic
- ✅ Single-threaded microstep processing (eventless transitions)
- ✅ Deterministic transition selection (document order)

**Tick-Based Runtime:**
- ✅ Event ordering guaranteed (FIFO, priority, or timestamp)
- ✅ Parallel regions synchronized at tick boundaries
- ✅ Sequential processing (no goroutine scheduling)
- ✅ Deterministic microstep processing
- ✅ Deterministic transition selection
- ✅ Reproducible execution (save/replay ticks)

**Determinism Example:**

```go
// Event-Driven: Non-deterministic
go rt.SendEvent(ctx, Event{ID: 1})
go rt.SendEvent(ctx, Event{ID: 2})
// Result: E1 or E2 may process first (depends on goroutine scheduling)

// Tick-Based: Deterministic
rt.SendEvent(Event{ID: 1})  // Queued at sequence 1
rt.SendEvent(Event{ID: 2})  // Queued at sequence 2
// Result: E1 always processes before E2 (FIFO guarantee)
```

### 4.5 Complexity Comparison

**Event-Driven Runtime:**
- **Concurrency**: Goroutines, channels, mutexes, context cancellation
- **Synchronization**: RWMutex for state access, channel timeouts
- **Error Handling**: Context errors, channel errors, goroutine panics
- **Testing**: Race detector, concurrent test scenarios
- **Lines of Code**: ~1,200 lines (statechart.go)

**Tick-Based Runtime:**
- **Concurrency**: None (single-threaded per tick)
- **Synchronization**: None (sequential processing)
- **Error Handling**: Simple error returns
- **Testing**: Deterministic test scenarios, easy replay
- **Lines of Code**: ~800 lines (estimated)

**Complexity Winner: Tick-Based** (simpler, easier to understand and maintain)

---

## Part 5: Implementation Plan

### 5.1 Directory Structure

```
statechartx/
├── statechart.go              # Core types (State, Transition, Machine)
├── runtime.go                 # Event-driven runtime (existing)
├── realtime/                  # New package for tick-based runtime
│   ├── runtime.go             # RealtimeRuntime implementation
│   ├── tick.go                # Tick processing logic
│   ├── events.go              # Event ordering and queueing
│   ├── state.go               # TickState and double buffering
│   ├── parallel.go            # Parallel region handling
│   ├── config.go              # TickConfig and options
│   └── realtime_test.go       # Tests
├── examples/
│   ├── game_loop/             # Game loop example
│   │   └── main.go
│   ├── simulation/            # Physics simulation example
│   │   └── main.go
│   └── comparison/            # Event-driven vs tick-based comparison
│       └── main.go
└── benchmarks/
    └── realtime_bench_test.go # Benchmarks comparing both runtimes
```

### 5.2 Core Types and Interfaces

```go
// realtime/config.go
package realtime

type TickConfig struct {
    TickRate          time.Duration
    MaxEventsPerTick  int
    EventOrdering     EventOrderingStrategy
    EnableProfiling   bool
    TickBudget        time.Duration  // Max time per tick (for overrun detection)
}

type EventOrderingStrategy int

const (
    OrderFIFO EventOrderingStrategy = iota
    OrderPriority
    OrderTimestamp
)

// realtime/state.go
type TickState struct {
    current       statechartx.StateID
    regionStates  map[statechartx.StateID]statechartx.StateID
    history       map[statechartx.StateID]statechartx.StateID
    deepHistory   map[statechartx.StateID][]statechartx.StateID
    datamodel     map[string]any
    timestamp     time.Time
    tickNumber    uint64
}

func (ts *TickState) Clone() *TickState {
    // Deep copy for double buffering
}

// realtime/events.go
type EventWithMetadata struct {
    Event       statechartx.Event
    Priority    int
    Timestamp   time.Time
    SourceID    uint64
    SequenceNum uint64
}

type EventQueue struct {
    events      []EventWithMetadata
    mu          sync.Mutex
    sequenceNum uint64
}

func (eq *EventQueue) Enqueue(event statechartx.Event, priority int) error
func (eq *EventQueue) DequeueAll() []EventWithMetadata
func (eq *EventQueue) Sort(strategy EventOrderingStrategy)

// realtime/runtime.go
type RealtimeRuntime struct {
    machine      *statechartx.Machine
    config       TickConfig
    tickN_1      *TickState
    tickN        *TickState
    tickNumber   uint64
    eventQueue   *EventQueue
    regionStates map[statechartx.StateID]*RegionState
    ctx          context.Context
    cancel       context.CancelFunc
    ticker       *time.Ticker
    running      bool
    mu           sync.Mutex
    
    // Profiling
    tickDurations []time.Duration
    overruns      int
}

func NewRuntime(machine *statechartx.Machine, config TickConfig) *RealtimeRuntime
func (rt *RealtimeRuntime) Start(ctx context.Context) error
func (rt *RealtimeRuntime) Stop() error
func (rt *RealtimeRuntime) SendEvent(event statechartx.Event) error
func (rt *RealtimeRuntime) SendEventWithPriority(event statechartx.Event, priority int) error
func (rt *RealtimeRuntime) IsInState(stateID statechartx.StateID) bool
func (rt *RealtimeRuntime) GetTickNumber() uint64
func (rt *RealtimeRuntime) GetTickState() *TickState  // For debugging/replay

// realtime/tick.go
func (rt *RealtimeRuntime) processTick()
func (rt *RealtimeRuntime) collectEvents() []EventWithMetadata
func (rt *RealtimeRuntime) sortEvents(events []EventWithMetadata) []EventWithMetadata
func (rt *RealtimeRuntime) processEvents(events []EventWithMetadata)
func (rt *RealtimeRuntime) processEvent(eventMeta EventWithMetadata)
func (rt *RealtimeRuntime) processMicrosteps()
func (rt *RealtimeRuntime) commitTick()

// realtime/parallel.go
func (rt *RealtimeRuntime) processParallelRegions()
func (rt *RealtimeRuntime) processRegion(regionID statechartx.StateID)
func (rt *RealtimeRuntime) processRegionEvent(regionID statechartx.StateID, eventMeta EventWithMetadata)
```

### 5.3 Implementation Steps

**Phase 1: Core Infrastructure (Week 1)**
1. Create `realtime/` package structure
2. Implement `TickState` with double buffering
3. Implement `EventQueue` with ordering strategies
4. Implement `TickConfig` and configuration options
5. Write unit tests for core types

**Phase 2: Basic Tick Processing (Week 1-2)**
1. Implement `RealtimeRuntime` struct
2. Implement `Start()` and `Stop()` methods
3. Implement `tickLoop()` and `processTick()`
4. Implement event collection and sorting
5. Implement basic event processing (no parallel states)
6. Write tests for simple state machines

**Phase 3: State Transitions (Week 2)**
1. Implement `processEvent()` with double buffering
2. Implement LCA computation (reuse from event-driven)
3. Implement `exitToLCA()` and `enterFromLCA()`
4. Implement microstep processing
5. Write tests for hierarchical state machines

**Phase 4: Parallel States (Week 2-3)**
1. Implement `RegionState` tracking
2. Implement `processParallelRegions()`
3. Implement `processRegion()` and `processRegionEvent()`
4. Implement deterministic region ordering
5. Write tests for parallel state machines

**Phase 5: Advanced Features (Week 3)**
1. Implement history state support
2. Implement done events
3. Implement tick profiling and overrun detection
4. Implement state snapshot/restore for replay
5. Write tests for advanced features

**Phase 6: Examples and Documentation (Week 3-4)**
1. Create game loop example (60 FPS)
2. Create physics simulation example
3. Create comparison example (event-driven vs tick-based)
4. Write comprehensive documentation
5. Write migration guide

**Phase 7: Benchmarking (Week 4)**
1. Implement benchmark suite
2. Compare throughput vs event-driven
3. Compare latency vs event-driven
4. Measure determinism overhead
5. Profile and optimize hot paths

### 5.4 Integration with Existing Code

**Shared Components:**
- ✅ `State`, `Transition`, `Machine` types (no changes needed)
- ✅ `Action`, `Guard` function types (no changes needed)
- ✅ LCA computation logic (can be extracted to shared package)
- ✅ Transition selection logic (can be extracted to shared package)

**Separate Components:**
- ❌ Event queue (different implementation)
- ❌ State tracking (double buffering vs single state)
- ❌ Parallel region handling (sequential vs goroutines)
- ❌ Event processing loop (tick-based vs async)

**Refactoring Opportunities:**
```go
// Extract shared logic to internal package
statechartx/
├── internal/
│   ├── lca.go          # LCA computation
│   ├── transitions.go  # Transition selection
│   └── hierarchy.go    # Hierarchy traversal
```

### 5.5 Test Strategy

**Unit Tests:**
```go
// realtime/runtime_test.go
func TestTickProcessing(t *testing.T)
func TestEventOrdering(t *testing.T)
func TestDoubleBuffering(t *testing.T)
func TestParallelRegions(t *testing.T)
func TestDeterminism(t *testing.T)
func TestMicrosteps(t *testing.T)
func TestHistoryStates(t *testing.T)
```

**Determinism Tests:**
```go
func TestDeterministicReplay(t *testing.T) {
    // Run same event sequence twice, verify identical results
    machine := createTestMachine()
    
    // Run 1
    rt1 := realtime.NewRuntime(machine, config)
    rt1.Start(ctx)
    for _, event := range testEvents {
        rt1.SendEvent(event)
    }
    time.Sleep(100 * time.Millisecond)
    state1 := rt1.GetTickState()
    
    // Run 2
    rt2 := realtime.NewRuntime(machine, config)
    rt2.Start(ctx)
    for _, event := range testEvents {
        rt2.SendEvent(event)
    }
    time.Sleep(100 * time.Millisecond)
    state2 := rt2.GetTickState()
    
    // Verify identical
    assert.Equal(t, state1, state2)
}
```

**Comparison Tests:**
```go
func TestEventDrivenVsTickBased(t *testing.T) {
    // Verify both runtimes produce same final state
    // (when event ordering is controlled)
}
```

**Stress Tests:**
```go
func TestTickOverrun(t *testing.T)
func TestMaxEventsPerTick(t *testing.T)
func TestLongRunningTicks(t *testing.T)
```

---

## Part 6: Benchmarking Plan

### 6.1 Comparison Benchmarks

**Throughput Benchmarks:**
```go
// benchmarks/realtime_bench_test.go

func BenchmarkEventDriven_Throughput(b *testing.B) {
    rt := statechartx.NewRuntime(machine, nil)
    rt.Start(ctx)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        rt.SendEvent(ctx, Event{ID: 1})
    }
}

func BenchmarkTickBased_Throughput_60FPS(b *testing.B) {
    rt := realtime.NewRuntime(machine, realtime.TickConfig{
        TickRate: 16667 * time.Microsecond,  // 60 FPS
    })
    rt.Start(ctx)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        rt.SendEvent(Event{ID: 1})
    }
}

func BenchmarkTickBased_Throughput_1000Hz(b *testing.B) {
    rt := realtime.NewRuntime(machine, realtime.TickConfig{
        TickRate: 1 * time.Millisecond,  // 1000 Hz
    })
    rt.Start(ctx)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        rt.SendEvent(Event{ID: 1})
    }
}
```

**Latency Benchmarks:**
```go
func BenchmarkEventDriven_Latency(b *testing.B) {
    rt := statechartx.NewRuntime(machine, nil)
    rt.Start(ctx)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        start := time.Now()
        rt.SendEvent(ctx, Event{ID: 1})
        // Wait for processing
        time.Sleep(1 * time.Microsecond)
        latency := time.Since(start)
        b.ReportMetric(float64(latency.Nanoseconds()), "ns/event")
    }
}

func BenchmarkTickBased_Latency_60FPS(b *testing.B) {
    rt := realtime.NewRuntime(machine, realtime.TickConfig{
        TickRate: 16667 * time.Microsecond,
    })
    rt.Start(ctx)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        start := time.Now()
        rt.SendEvent(Event{ID: 1})
        // Wait for next tick
        time.Sleep(17 * time.Millisecond)
        latency := time.Since(start)
        b.ReportMetric(float64(latency.Nanoseconds()), "ns/event")
    }
}
```

**Parallel Region Benchmarks:**
```go
func BenchmarkEventDriven_ParallelRegions_10(b *testing.B)
func BenchmarkTickBased_ParallelRegions_10(b *testing.B)
func BenchmarkEventDriven_ParallelRegions_100(b *testing.B)
func BenchmarkTickBased_ParallelRegions_100(b *testing.B)
```

### 6.2 Scenarios to Test

**Scenario 1: Game Loop (60 FPS)**
```go
func BenchmarkGameLoop_EventDriven(b *testing.B)
func BenchmarkGameLoop_TickBased(b *testing.B)

// Metrics:
// - Events per frame
// - Frame time consistency
// - Determinism (replay test)
```

**Scenario 2: Physics Simulation (1000 Hz)**
```go
func BenchmarkPhysicsSimulation_EventDriven(b *testing.B)
func BenchmarkPhysicsSimulation_TickBased(b *testing.B)

// Metrics:
// - Simulation steps per second
// - Numerical stability
// - Determinism
```

**Scenario 3: Real-Time Control (100 Hz)**
```go
func BenchmarkRealtimeControl_EventDriven(b *testing.B)
func BenchmarkRealtimeControl_TickBased(b *testing.B)

// Metrics:
// - Control loop frequency
// - Jitter
// - Worst-case latency
```

**Scenario 4: Concurrent Event Submission**
```go
func BenchmarkConcurrentEvents_EventDriven(b *testing.B) {
    // Multiple goroutines sending events
}

func BenchmarkConcurrentEvents_TickBased(b *testing.B) {
    // Multiple goroutines sending events
}

// Metrics:
// - Throughput under contention
// - Event ordering consistency
```

**Scenario 5: Parallel State Synchronization**
```go
func BenchmarkParallelSync_EventDriven(b *testing.B)
func BenchmarkParallelSync_TickBased(b *testing.B)

// Metrics:
// - Synchronization overhead
// - Determinism
```

### 6.3 Metrics to Measure

| Metric | Event-Driven | Tick-Based | Comparison |
|--------|--------------|------------|------------|
| **Throughput** | events/sec | events/sec | Ratio |
| **Latency (avg)** | nanoseconds | milliseconds | Ratio |
| **Latency (p99)** | nanoseconds | milliseconds | Ratio |
| **Memory per Machine** | bytes | bytes | Ratio |
| **CPU Usage** | % | % | Difference |
| **Tick Consistency** | N/A | stdev of tick duration | Absolute |
| **Determinism** | % reproducible | % reproducible | Difference |
| **Parallel Region Overhead** | μs per region | μs per region | Ratio |
| **Event Ordering Overhead** | N/A | μs per sort | Absolute |

### 6.4 Success Criteria

**Performance Targets:**
- ✅ Tick-based runtime processes **1000 events/tick @ 60 FPS** (60K events/sec)
- ✅ Tick processing completes within **80% of tick budget** (13.3ms @ 60 FPS)
- ✅ **100% deterministic** replay (identical state after same event sequence)
- ✅ Parallel regions process in **< 1ms** for 10 regions
- ✅ Event ordering overhead **< 100μs** for 1000 events

**Comparison Targets:**
- ✅ Tick-based throughput **> 50K events/sec** (acceptable for real-time use cases)
- ✅ Tick-based latency **< 20ms** @ 60 FPS (acceptable for games)
- ✅ Tick-based memory **< 2x** event-driven (acceptable overhead)
- ✅ Tick-based CPU **< 1.5x** event-driven (acceptable overhead)

**Determinism Targets:**
- ✅ **100% reproducible** execution with same event sequence
- ✅ **0 race conditions** detected (race detector)
- ✅ **Bit-identical** state after replay

### 6.5 Profiling Strategy

**CPU Profiling:**
```bash
# Profile tick-based runtime
go test -bench=BenchmarkTickBased -cpuprofile=tick_cpu.prof
go tool pprof -http=:8080 tick_cpu.prof

# Compare with event-driven
go test -bench=BenchmarkEventDriven -cpuprofile=event_cpu.prof
go tool pprof -http=:8081 event_cpu.prof
```

**Memory Profiling:**
```bash
go test -bench=BenchmarkTickBased -memprofile=tick_mem.prof
go tool pprof -http=:8080 tick_mem.prof
```

**Trace Analysis:**
```bash
go test -bench=BenchmarkTickBased -trace=tick_trace.out
go tool trace tick_trace.out
```

**Expected Hotspots (Tick-Based):**
1. Event sorting (O(n log n) per tick)
2. State cloning (double buffering)
3. Parallel region iteration
4. LCA computation (same as event-driven)

**Optimization Opportunities:**
1. Object pooling for TickState
2. Pre-allocated event slices
3. Incremental sorting (insertion sort for small queues)
4. Copy-on-write for datamodel

---

## Part 7: Discussion Points

### 7.1 Key Design Decisions

**Decision 1: No Goroutines for Parallel Regions**
- **Rationale**: Determinism, simplicity, real-time systems often single-core
- **Trade-off**: Cannot utilize multiple CPU cores per tick
- **Alternative**: Add goroutine option later if profiling shows bottleneck

**Decision 2: Double Buffering**
- **Rationale**: Eliminates race conditions, enables rollback/replay
- **Trade-off**: 2x memory overhead for state
- **Alternative**: Single buffer with copy-on-write (more complex)

**Decision 3: FIFO Default Ordering**
- **Rationale**: Simplest, most intuitive, matches event-driven behavior
- **Trade-off**: No priority support by default
- **Alternative**: Priority ordering (configurable)

**Decision 4: Fixed Tick Rate**
- **Rationale**: Deterministic timing, matches game loop pattern
- **Trade-off**: Cannot adapt to varying load
- **Alternative**: Variable tick rate (more complex, less deterministic)

**Decision 5: Adjacent Implementation (realtime/ package)**
- **Rationale**: No changes to existing code, easy to compare
- **Trade-off**: Some code duplication
- **Alternative**: Refactor existing runtime to support both modes (risky)

### 7.2 Open Questions

**Question 1: Should we support hybrid mode?**
- Use tick-based for deterministic core logic
- Use event-driven for I/O and async operations
- **Complexity**: High, but potentially very powerful

**Question 2: Should we support variable tick rates?**
- Adapt tick rate based on load
- **Benefit**: Better CPU utilization
- **Cost**: Less deterministic

**Question 3: Should we support tick interpolation?**
- Render at higher rate than simulation tick rate
- **Benefit**: Smoother visuals in games
- **Cost**: More complex API

**Question 4: Should we support distributed ticks?**
- Synchronize ticks across multiple machines (multiplayer)
- **Benefit**: Deterministic multiplayer
- **Cost**: Network latency, synchronization complexity

**Question 5: Should we support tick recording/replay?**
- Save tick state and events for debugging
- **Benefit**: Powerful debugging tool
- **Cost**: Storage overhead

### 7.3 Future Enhancements

**Enhancement 1: Tick Profiler**
- Visualize tick timeline
- Identify slow events/transitions
- Detect tick overruns

**Enhancement 2: Replay Debugger**
- Step through ticks
- Inspect state at each tick
- Modify events and re-run

**Enhancement 3: Distributed Tick Synchronization**
- Lockstep networking for multiplayer
- Rollback/predict for latency hiding

**Enhancement 4: Tick Interpolation**
- Smooth rendering between ticks
- Extrapolate state for low-latency rendering

**Enhancement 5: Adaptive Tick Rate**
- Adjust tick rate based on load
- Maintain determinism within variable rate

---

## Part 8: Conclusion

### 8.1 Summary

The **tick-based real-time runtime** offers a compelling alternative to the event-driven runtime for use cases requiring **deterministic execution, temporal consistency, and reproducible behavior**. While it sacrifices throughput (33x slower) and latency (76,000x slower), it provides **guaranteed event ordering, deterministic parallel region processing, and easy debugging/replay**.

**Key Benefits:**
1. ✅ **100% deterministic** execution
2. ✅ **Reproducible** behavior (save/replay)
3. ✅ **Temporal consistency** (fixed time steps)
4. ✅ **Simpler implementation** (no goroutines, no sync primitives)
5. ✅ **Easy debugging** (step through ticks)
6. ✅ **Frame-locked** behavior (game loops)

**Key Trade-offs:**
1. ❌ **Lower throughput** (60K vs 2M events/sec)
2. ❌ **Higher latency** (16.67ms vs 217ns)
3. ❌ **No true parallelism** (sequential region processing)
4. ❌ **Higher memory** (double buffering)

### 8.2 Recommendation

**Proceed with implementation** of the tick-based runtime as an **adjacent package** (`realtime/`) alongside the existing event-driven runtime. This approach:

1. **Preserves existing performance** - No changes to event-driven runtime
2. **Enables comparison** - Users can choose based on use case
3. **Reduces risk** - No refactoring of proven code
4. **Provides learning** - Understand trade-offs through real implementation

**Target Use Cases:**
- Game engines (60 FPS game logic)
- Physics simulations (fixed time-step integration)
- Robotics (deterministic control loops)
- Testing/debugging (reproducible scenarios)
- Multiplayer games (lockstep networking)

**Implementation Timeline:**
- **Week 1-2**: Core infrastructure and basic tick processing
- **Week 2-3**: State transitions and parallel states
- **Week 3-4**: Advanced features, examples, documentation
- **Week 4**: Benchmarking and optimization

**Success Metrics:**
- ✅ 60K events/sec @ 60 FPS
- ✅ 100% deterministic replay
- ✅ < 20ms latency @ 60 FPS
- ✅ < 2x memory overhead

### 8.3 Next Steps

1. **Review this design document** with stakeholders
2. **Validate use cases** - Confirm target domains
3. **Prototype core tick loop** - Validate performance assumptions
4. **Implement Phase 1** - Core infrastructure
5. **Create game loop example** - Validate API design
6. **Benchmark against event-driven** - Measure actual trade-offs
7. **Iterate based on feedback** - Refine design

---

## Appendix A: Code Examples

### A.1 Game Loop Example (60 FPS)

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/comalice/statechartx"
    "github.com/comalice/statechartx/realtime"
)

const (
    IDLE StateID = iota
    RUNNING
    JUMPING
)

const (
    START EventID = iota
    JUMP
    LAND
    STOP
)

func main() {
    // Create state machine
    machine := createPlayerStateMachine()
    
    // Create tick-based runtime (60 FPS)
    rt := realtime.NewRuntime(machine, realtime.TickConfig{
        TickRate:         16667 * time.Microsecond,  // 60 FPS
        MaxEventsPerTick: 100,
        EventOrdering:    realtime.OrderFIFO,
    })
    
    // Start runtime
    ctx := context.Background()
    rt.Start(ctx)
    
    // Game loop
    for i := 0; i < 600; i++ {  // 10 seconds @ 60 FPS
        // Simulate player input
        if i == 60 {  // 1 second
            rt.SendEvent(statechartx.Event{ID: START})
        }
        if i == 120 {  // 2 seconds
            rt.SendEvent(statechartx.Event{ID: JUMP})
        }
        if i == 180 {  // 3 seconds
            rt.SendEvent(statechartx.Event{ID: LAND})
        }
        if i == 240 {  // 4 seconds
            rt.SendEvent(statechartx.Event{ID: STOP})
        }
        
        // Wait for next frame
        time.Sleep(16667 * time.Microsecond)
        
        // Query state
        if i % 60 == 0 {  // Every second
            fmt.Printf("Tick %d: State = %v\n", rt.GetTickNumber(), rt.GetTickState().current)
        }
    }
    
    rt.Stop()
}
```

### A.2 Physics Simulation Example (1000 Hz)

```go
func main() {
    machine := createPhysicsStateMachine()
    
    rt := realtime.NewRuntime(machine, realtime.TickConfig{
        TickRate:         1 * time.Millisecond,  // 1000 Hz
        MaxEventsPerTick: 10,
        EventOrdering:    realtime.OrderFIFO,
    })
    
    ctx := context.Background()
    rt.Start(ctx)
    
    // Simulation loop
    for i := 0; i < 10000; i++ {  // 10 seconds
        // Send physics update event
        rt.SendEvent(statechartx.Event{
            ID: PHYSICS_UPDATE,
            Data: PhysicsData{
                DeltaTime: 0.001,  // 1ms
            },
        })
        
        time.Sleep(1 * time.Millisecond)
    }
    
    rt.Stop()
}
```

### A.3 Deterministic Replay Example

```go
func main() {
    machine := createTestMachine()
    
    // Record events
    events := []statechartx.Event{
        {ID: 1}, {ID: 2}, {ID: 3}, {ID: 1}, {ID: 2},
    }
    
    // Run 1
    rt1 := realtime.NewRuntime(machine, config)
    rt1.Start(ctx)
    for _, event := range events {
        rt1.SendEvent(event)
    }
    time.Sleep(100 * time.Millisecond)
    state1 := rt1.GetTickState()
    rt1.Stop()
    
    // Run 2
    rt2 := realtime.NewRuntime(machine, config)
    rt2.Start(ctx)
    for _, event := range events {
        rt2.SendEvent(event)
    }
    time.Sleep(100 * time.Millisecond)
    state2 := rt2.GetTickState()
    rt2.Stop()
    
    // Verify determinism
    if state1.current == state2.current {
        fmt.Println("✅ Deterministic: Both runs ended in same state")
    } else {
        fmt.Println("❌ Non-deterministic: Different final states")
    }
}
```

---

## Appendix B: Performance Projections

### B.1 Throughput Projections

| Tick Rate | Events/Tick | Throughput | vs Event-Driven |
|-----------|-------------|------------|-----------------|
| 60 FPS | 100 | 6,000 events/sec | 333x slower |
| 60 FPS | 1000 | 60,000 events/sec | 33x slower |
| 120 FPS | 1000 | 120,000 events/sec | 16x slower |
| 1000 Hz | 100 | 100,000 events/sec | 20x slower |
| 1000 Hz | 1000 | 1,000,000 events/sec | 2x slower |

**Conclusion**: Tick-based can approach event-driven throughput at high tick rates (1000 Hz), but at cost of higher CPU usage.

### B.2 Latency Projections

| Tick Rate | Avg Latency | Max Latency | vs Event-Driven |
|-----------|-------------|-------------|-----------------|
| 60 FPS | 8.3 ms | 16.67 ms | 38,000x slower |
| 120 FPS | 4.2 ms | 8.33 ms | 19,000x slower |
| 1000 Hz | 0.5 ms | 1 ms | 2,300x slower |
| 10000 Hz | 0.05 ms | 0.1 ms | 230x slower |

**Conclusion**: Tick-based latency is always higher, but acceptable for real-time systems at appropriate tick rates.

### B.3 Memory Projections

| Component | Event-Driven | Tick-Based | Ratio |
|-----------|--------------|------------|-------|
| Base Runtime | 0.61 KB | 1.2 KB | 2x |
| State Storage | 150 bytes/state | 300 bytes/state | 2x (double buffer) |
| Event Queue | 100 events × 96 bytes | 1000 events × 128 bytes | 13x |
| Parallel Regions | 15 KB/10 regions | 20 KB/10 regions | 1.3x |
| **Total (1000 states, 10 regions)** | **165 KB** | **330 KB** | **2x** |

**Conclusion**: Tick-based uses ~2x memory due to double buffering, acceptable overhead.

---

**Document Version:** 1.0  
**Last Updated:** January 2, 2026  
**Author:** StatechartX Design Team  
**Status:** Ready for Review
