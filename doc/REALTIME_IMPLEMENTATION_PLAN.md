# StatechartX Real-Time Runtime: Implementation Plan

**Date:** January 2, 2026  
**Status:** Ready for Implementation  
**Approach:** Embed-and-Adapt (Minimal New Code)  
**Priority:** Concise > Readable > Performant

---

## Executive Summary

This plan implements a **tick-based real-time runtime** by **embedding the existing Runtime** and adapting only the event dispatch mechanism. This approach reuses ~86% of the existing synchronous core logic, requiring only **~230 lines of new code** instead of the ~1000 lines proposed in the original plan.

**Key Metrics:**
- **New code:** ~230 lines
- **Reused code:** ~430 lines (processEvent, computeLCA, history states, etc.)
- **Code duplication:** 0 lines
- **Time to production:** 3-4 weeks (vs 12 weeks in original plan)

**Core Insight:** The existing `processEvent()`, `processMicrosteps()`, `computeLCA()`, `exitToLCA()`, `enterFromLCA()`, and history methods are **pure synchronous logic** with zero goroutine coupling. We can call them directly from the tick-based runtime.

---

## Part 1: Package Structure

### 1.1 Directory Layout

```
statechartx_review/
├── statechart.go              # Existing event-driven runtime (NO CHANGES)
├── realtime/                  # NEW: Tick-based runtime package
│   ├── runtime.go             # ~150 lines: RealtimeRuntime struct, lifecycle
│   ├── tick.go                # ~50 lines: Tick processing logic
│   ├── event.go               # ~30 lines: Event batching and sorting
│   ├── doc.go                 # ~20 lines: Package documentation
│   └── realtime_test.go       # ~200 lines: Tests
├── examples/
│   └── realtime/              # NEW: Real-time examples
│       ├── game_loop.go       # 60 FPS game loop example
│       ├── physics_sim.go     # 1000 Hz physics simulation
│       └── replay.go          # Deterministic replay example
└── benchmarks/
    └── realtime_bench_test.go # NEW: Comparison benchmarks
```

**Total New Files:** 8  
**Modified Files:** 0 (existing runtime unchanged)

### 1.2 What Goes Where

#### `realtime/runtime.go` (~150 lines)
- `RealtimeRuntime` struct definition
- `NewRuntime()` constructor
- `Start()` / `Stop()` lifecycle methods
- `SendEvent()` event queuing API
- `GetCurrentState()` state query methods
- Tick loop management

#### `realtime/tick.go` (~50 lines)
- `processTick()` orchestration
- `collectEvents()` event batching
- `sortEvents()` deterministic ordering
- Parallel region sequential processing

#### `realtime/event.go` (~30 lines)
- `EventWithMeta` struct (adds sequence number)
- Event sorting logic
- Queue management

#### `realtime/doc.go` (~20 lines)
- Package documentation
- Usage examples
- Trade-off explanations

---

## Part 2: Detailed File Breakdown

### 2.1 realtime/runtime.go

**Purpose:** Main runtime struct and lifecycle management

**Struct Definition:** (~40 lines)

```go
package realtime

import (
	"context"
	"sync"
	"time"
	
	"path/to/statechartx"
)

// RealtimeRuntime provides tick-based deterministic execution by embedding
// the existing event-driven Runtime and adapting only the event dispatch.
type RealtimeRuntime struct {
	// Embed existing runtime to reuse ALL core methods:
	// - processEvent()
	// - processMicrosteps()
	// - computeLCA()
	// - exitToLCA() / enterFromLCA()
	// - pickTransitionHierarchical()
	// - History state methods
	// - Done event methods
	*statechartx.Runtime
	
	// Tick-specific fields
	tickRate      time.Duration      // e.g., 16.67ms for 60 FPS
	ticker        *time.Ticker
	tickNum       uint64
	
	// Event batching (replaces async channel)
	eventBatch    []EventWithMeta
	batchMu       sync.Mutex
	sequenceNum   uint64
	
	// Control
	tickCtx       context.Context
	tickCancel    context.CancelFunc
	stopped       chan struct{}
}

// EventWithMeta adds sequencing metadata for deterministic ordering
type EventWithMeta struct {
	Event       statechartx.Event
	SequenceNum uint64
	Priority    int  // For future priority ordering
}

// Config configures the real-time runtime
type Config struct {
	TickRate         time.Duration  // Fixed tick rate (e.g., 16.67ms for 60 FPS)
	MaxEventsPerTick int            // Event queue capacity (default: 1000)
}
```

**Constructor:** (~15 lines)

```go
// NewRuntime creates a new tick-based runtime by embedding the event-driven runtime
func NewRuntime(machine *statechartx.Machine, cfg Config) *RealtimeRuntime {
	if cfg.MaxEventsPerTick == 0 {
		cfg.MaxEventsPerTick = 1000
	}
	
	return &RealtimeRuntime{
		// Embed existing runtime (THIS IS THE KEY - REUSE EVERYTHING)
		Runtime:     statechartx.NewRuntime(machine, nil),
		tickRate:    cfg.TickRate,
		eventBatch:  make([]EventWithMeta, 0, cfg.MaxEventsPerTick),
		stopped:     make(chan struct{}),
	}
}
```

**Lifecycle Methods:** (~50 lines)

```go
// Start begins tick-based execution
func (rt *RealtimeRuntime) Start(ctx context.Context) error {
	// Enter initial state using EXISTING method
	if err := rt.Runtime.Start(ctx); err != nil {
		return err
	}
	
	// Start tick loop (ONLY DIFFERENCE from event-driven)
	rt.tickCtx, rt.tickCancel = context.WithCancel(ctx)
	rt.ticker = time.NewTicker(rt.tickRate)
	
	go rt.tickLoop()
	
	return nil
}

// Stop gracefully stops the runtime
func (rt *RealtimeRuntime) Stop() error {
	if rt.tickCancel != nil {
		rt.tickCancel()
	}
	if rt.ticker != nil {
		rt.ticker.Stop()
	}
	
	// Wait for tick loop to exit
	<-rt.stopped
	
	// Stop embedded runtime
	return rt.Runtime.Stop()
}

// tickLoop is the main tick execution loop
func (rt *RealtimeRuntime) tickLoop() {
	defer close(rt.stopped)
	
	for {
		select {
		case <-rt.tickCtx.Done():
			return
		case <-rt.ticker.C:
			rt.processTick()
			rt.tickNum++
		}
	}
}
```

**Event Sending API:** (~25 lines)

```go
// SendEvent queues an event for the next tick (thread-safe)
// NOTE: No context parameter - events are queued, not processed immediately
func (rt *RealtimeRuntime) SendEvent(event statechartx.Event) error {
	rt.batchMu.Lock()
	defer rt.batchMu.Unlock()
	
	if len(rt.eventBatch) >= cap(rt.eventBatch) {
		return errors.New("event queue full")
	}
	
	rt.eventBatch = append(rt.eventBatch, EventWithMeta{
		Event:       event,
		SequenceNum: rt.sequenceNum,
		Priority:    0, // Default priority
	})
	rt.sequenceNum++
	
	return nil
}

// SendEventWithPriority queues an event with priority
func (rt *RealtimeRuntime) SendEventWithPriority(event statechartx.Event, priority int) error {
	rt.batchMu.Lock()
	defer rt.batchMu.Unlock()
	
	if len(rt.eventBatch) >= cap(rt.eventBatch) {
		return errors.New("event queue full")
	}
	
	rt.eventBatch = append(rt.eventBatch, EventWithMeta{
		Event:       event,
		SequenceNum: rt.sequenceNum,
		Priority:    priority,
	})
	rt.sequenceNum++
	
	return nil
}
```

**Query Methods:** (~20 lines)

```go
// GetTickNumber returns the current tick count
func (rt *RealtimeRuntime) GetTickNumber() uint64 {
	rt.batchMu.Lock()
	defer rt.batchMu.Unlock()
	return rt.tickNum
}

// IsInState returns true if currently in the given state
// Uses EXISTING method from embedded Runtime
func (rt *RealtimeRuntime) IsInState(stateID statechartx.StateID) bool {
	return rt.Runtime.IsInState(stateID)
}

// GetCurrentState returns the current state
// Uses EXISTING method from embedded Runtime
func (rt *RealtimeRuntime) GetCurrentState() statechartx.StateID {
	rt.Runtime.GetRLock()
	defer rt.Runtime.GetRUnlock()
	return rt.Runtime.GetCurrent()
}
```

---

### 2.2 realtime/tick.go

**Purpose:** Tick processing orchestration

**Main Tick Processor:** (~30 lines)

```go
package realtime

import (
	"context"
	"sort"
)

// processTick processes one complete tick
func (rt *RealtimeRuntime) processTick() {
	// Phase 1: Collect events atomically
	events := rt.collectEvents()
	
	// Phase 2: Sort for deterministic order
	rt.sortEvents(events)
	
	// Phase 3: Process events using EXISTING core methods
	rt.processEvents(events)
	
	// Phase 4: Process microsteps using EXISTING core method
	rt.processMicrostepsIfNeeded()
	
	// Phase 5: Process parallel regions sequentially (if any)
	rt.processParallelRegionsSequentially()
}

// collectEvents atomically retrieves and clears the event batch
func (rt *RealtimeRuntime) collectEvents() []EventWithMeta {
	rt.batchMu.Lock()
	defer rt.batchMu.Unlock()
	
	events := rt.eventBatch
	rt.eventBatch = make([]EventWithMeta, 0, cap(rt.eventBatch))
	
	return events
}
```

**Event Processing:** (~20 lines)

```go
// processEvents processes all events for this tick
func (rt *RealtimeRuntime) processEvents(events []EventWithMeta) {
	for _, eventMeta := range events {
		// CRITICAL: Call EXISTING processEvent method from embedded Runtime
		// This is where we reuse ~430 lines of battle-tested code!
		rt.Runtime.ProcessEvent(eventMeta.Event)
	}
}

// processMicrostepsIfNeeded processes eventless transitions
func (rt *RealtimeRuntime) processMicrostepsIfNeeded() {
	// CRITICAL: Call EXISTING processMicrosteps method
	// Reuses existing microstep logic (lines 784-861 of statechart.go)
	rt.Runtime.ProcessMicrosteps(context.Background())
}

// processParallelRegionsSequentially processes parallel regions in order
func (rt *RealtimeRuntime) processParallelRegionsSequentially() {
	// TODO: Implement when parallel state support is added
	// Will reuse existing transition methods but process sequentially
	// See Phase 3 implementation details below
}
```

---

### 2.3 realtime/event.go

**Purpose:** Event batching and ordering

**Sorting Logic:** (~30 lines)

```go
package realtime

import (
	"sort"
)

// sortEvents orders events deterministically
func (rt *RealtimeRuntime) sortEvents(events []EventWithMeta) {
	// Stable sort preserves insertion order for equal priorities
	sort.SliceStable(events, func(i, j int) bool {
		// Primary: Higher priority first
		if events[i].Priority != events[j].Priority {
			return events[i].Priority > events[j].Priority
		}
		
		// Secondary: Earlier sequence number first (FIFO)
		return events[i].SequenceNum < events[j].SequenceNum
	})
}

// Event ordering guarantees:
// 1. Events from same source processed in submission order (sequence number)
// 2. Higher priority events processed first
// 3. Deterministic tie-breaking via sequence number
// 4. Stable sort preserves relative order
```

---

### 2.4 realtime/doc.go

**Purpose:** Package documentation

```go
// Package realtime provides a tick-based deterministic runtime for StatechartX.
//
// The real-time runtime differs from the event-driven runtime in event dispatch:
// - Events are batched and processed at fixed tick boundaries
// - Deterministic event ordering via sequence numbers
// - Parallel regions processed sequentially (no goroutines)
// - Fixed time-step execution (e.g., 60 FPS)
//
// Example usage:
//
//	machine, _ := statechartx.NewMachine(rootState)
//	rt := realtime.NewRuntime(machine, realtime.Config{
//		TickRate: 16667 * time.Microsecond, // 60 FPS
//	})
//	rt.Start(ctx)
//	rt.SendEvent(statechartx.Event{ID: 1})
//
// Trade-offs vs Event-Driven:
// - Lower throughput (60K vs 2M events/sec at 60 FPS)
// - Higher latency (16.67ms vs 217ns at 60 FPS)
// - Guaranteed determinism and reproducibility
// - Fixed time budget per tick
//
// Use Cases:
// - Game engines (60 FPS game logic)
// - Physics simulations (fixed time-step)
// - Robotics (deterministic control loops)
// - Testing/debugging (reproducible scenarios)
//
package realtime
```

---

## Part 3: Reusing Existing Methods

### 3.1 Methods We Call Directly (NO NEW CODE)

The following methods from `statechart.go` are called directly with **zero modifications**:

#### **State Transition Core** (~300 lines reused)

```go
// Lines 710-779: processEvent - External transition handling
rt.Runtime.ProcessEvent(event)

// Lines 784-861: processMicrosteps - Eventless transitions
rt.Runtime.ProcessMicrosteps(ctx)

// Lines 874-902: computeLCA - Least Common Ancestor
lca := rt.Runtime.ComputeLCA(from, to)

// Lines 904-921: exitToLCA - Exit states to LCA
rt.Runtime.ExitToLCA(ctx, &event, from, to, lca)

// Lines 923-956: enterFromLCA - Enter states from LCA
rt.Runtime.EnterFromLCA(ctx, &event, from, to, lca)

// Lines 1063-1117: pickTransitionHierarchical - Find matching transition
transition := rt.Runtime.PickTransitionHierarchical(state, event)
```

#### **History State Management** (~100 lines reused)

```go
// Lines 1200-1306: History recording and restoration
rt.Runtime.RecordHistory(parentID, childID)
rt.Runtime.RestoreHistory(ctx, state, &event, from)
rt.Runtime.RestoreShallowHistory(...)
rt.Runtime.RestoreDeepHistory(...)
```

#### **Done Event Management** (~95 lines reused, 5 lines adapted)

```go
// Lines 958-979: Check if state is final
rt.Runtime.CheckFinalState(ctx)

// Lines 1023-1031: Should emit done event
rt.Runtime.ShouldEmitDoneEvent(parent)

// Lines 1033-1054: All regions in final state
rt.Runtime.AllRegionsInFinalState(state)
```

**Adaptation Needed:** Lines 1013-1020 of `generateDoneEvent()` need modification:

```go
// CURRENT (async channel send):
go func() {
	select {
	case rt.eventQueue <- doneEvent:
	case <-ctx.Done():
	case <-time.After(100 * time.Millisecond):
	}
}()

// TICK-BASED (append to batch):
rt.SendEvent(doneEvent)  // Uses our SendEvent which batches for next tick
```

### 3.2 Exposing Internal Methods

The existing `Runtime` struct has lowercase (private) methods we need. Two options:

#### **Option A: Make Methods Public (Recommended)**

Add these method aliases to `statechart.go`:

```go
// Public aliases for tick-based runtime (add to statechart.go)

// ProcessEvent exposes processEvent for tick-based runtime
func (rt *Runtime) ProcessEvent(event Event) {
	rt.processEvent(event)
}

// ProcessMicrosteps exposes processMicrosteps for tick-based runtime
func (rt *Runtime) ProcessMicrosteps(ctx context.Context) {
	rt.processMicrosteps(ctx)
}

// GetCurrent exposes current state (add RLock accessor)
func (rt *Runtime) GetCurrent() StateID {
	return rt.current
}

// GetRLock/GetRUnlock for safe concurrent access
func (rt *Runtime) GetRLock() { rt.mu.RLock() }
func (rt *Runtime) GetRUnlock() { rt.mu.RUnlock() }
```

**Lines Added to statechart.go:** ~20 lines  
**Benefit:** Clean API, no reflection hacks

#### **Option B: Use Go Embedding Hack (Not Recommended)**

Access private fields via type assertion. **Don't do this** - it's brittle and defeats type safety.

### 3.3 What We DON'T Need to Reuse

**Event Loop (lines 695-707):** Replaced entirely with tick loop  
**Parallel Region Goroutines (lines 322-575):** Adapted for sequential processing (Phase 3)

---

## Part 4: Code Examples

### 4.1 How Embedding Works

```go
// This is the magic - we get ALL methods for free!
type RealtimeRuntime struct {
	*statechartx.Runtime  // Embeds ALL existing methods
	
	// Only add tick-specific fields
	tickRate    time.Duration
	eventBatch  []EventWithMeta
	// ... etc
}

// Example: Calling embedded methods
rt := NewRuntime(machine, config)

// These work because Runtime is embedded:
rt.IsInState(stateID)       // Calls statechartx.Runtime.IsInState()
rt.GetCurrentState()         // Calls statechartx.Runtime.GetCurrentState()

// Our custom method:
rt.SendEvent(event)          // Calls RealtimeRuntime.SendEvent() (batches for tick)

// But internally we call embedded methods:
rt.Runtime.ProcessEvent(event)  // Calls embedded processEvent()
```

### 4.2 Tick Processing Flow

```go
func (rt *RealtimeRuntime) processTick() {
	// 1. Collect events (NEW CODE - ~5 lines)
	events := rt.collectEvents()
	
	// 2. Sort events (NEW CODE - ~10 lines)
	rt.sortEvents(events)
	
	// 3. Process each event (REUSE EXISTING - calls processEvent)
	for _, eventMeta := range events {
		rt.Runtime.ProcessEvent(eventMeta.Event)  // ← REUSES ~70 lines
	}
	
	// 4. Process microsteps (REUSE EXISTING - calls processMicrosteps)
	rt.Runtime.ProcessMicrosteps(ctx)  // ← REUSES ~80 lines
	
	// 5. Check final states (REUSE EXISTING - calls checkFinalState)
	// This is called within ProcessEvent, so we get it for free!
}
```

### 4.3 Deterministic Event Ordering

```go
// Scenario: Two goroutines send events concurrently
go func() {
	rt.SendEvent(statechartx.Event{ID: 1, Data: "A"})
}()

go func() {
	rt.SendEvent(statechartx.Event{ID: 2, Data: "B"})
}()

// SendEvent implementation:
func (rt *RealtimeRuntime) SendEvent(event statechartx.Event) error {
	rt.batchMu.Lock()
	defer rt.batchMu.Unlock()
	
	// Whoever acquires lock first gets lower sequence number
	rt.eventBatch = append(rt.eventBatch, EventWithMeta{
		Event:       event,
		SequenceNum: rt.sequenceNum,  // ← Determines order
	})
	rt.sequenceNum++
	
	return nil
}

// At next tick:
// - Events sorted by sequence number
// - Event with lower sequence number (acquired lock first) processes first
// - Order is DETERMINISTIC given lock acquisition order
// - Replay: Record sequence numbers, replay in same order
```

### 4.4 Complete Example: 60 FPS Game Loop

```go
package main

import (
	"context"
	"fmt"
	"time"
	
	"path/to/statechartx"
	"path/to/statechartx/realtime"
)

func main() {
	// Create state machine (simplified)
	machine, _ := statechartx.NewMachine(createGameStateMachine())
	
	// Create tick-based runtime (60 FPS)
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate:         16667 * time.Microsecond,  // 60 FPS
		MaxEventsPerTick: 100,
	})
	
	// Start runtime
	ctx := context.Background()
	rt.Start(ctx)
	defer rt.Stop()
	
	// Game loop - send events
	ticker := time.NewTicker(16667 * time.Microsecond)
	defer ticker.Stop()
	
	for i := 0; i < 600; i++ { // 10 seconds @ 60 FPS
		select {
		case <-ticker.C:
			// Simulate player input
			if i == 60 {  // 1 second in
				rt.SendEvent(statechartx.Event{ID: START_GAME})
			}
			if i == 120 { // 2 seconds in
				rt.SendEvent(statechartx.Event{ID: PLAYER_JUMP})
			}
			
			// Query state (non-blocking, reads from last tick)
			if i%60 == 0 {  // Every second
				fmt.Printf("Tick %d: State = %v\n", 
					rt.GetTickNumber(), 
					rt.GetCurrentState())
			}
		}
	}
}
```

---

## Part 5: Phase-by-Phase Implementation

### Phase 1: Foundation (Week 1, Days 1-5)

**Goal:** Get basic tick loop working with simple state machines

#### Day 1-2: Package Setup and Core Structs

**Tasks:**
1. Create `realtime/` package directory
2. Implement `RealtimeRuntime` struct with embedding
3. Implement `EventWithMeta` struct
4. Implement `Config` struct
5. Add public method aliases to `statechart.go`

**Code to Write:** ~60 lines
- `realtime/runtime.go`: RealtimeRuntime struct (~40 lines)
- `realtime/event.go`: EventWithMeta struct (~20 lines)
- `statechart.go`: Public aliases (~20 lines)

**Deliverables:**
- [ ] `realtime/runtime.go` with struct definition
- [ ] `realtime/event.go` with EventWithMeta
- [ ] Public method aliases in `statechart.go`
- [ ] Package compiles without errors

**Test:**
```go
func TestRuntimeCreation(t *testing.T) {
	machine, _ := statechartx.NewMachine(simpleState)
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate: 10 * time.Millisecond,
	})
	assert.NotNil(t, rt)
	assert.NotNil(t, rt.Runtime) // Embedded runtime exists
}
```

#### Day 3-4: Tick Loop and Event Batching

**Tasks:**
1. Implement `NewRuntime()` constructor
2. Implement `Start()` / `Stop()` lifecycle methods
3. Implement `tickLoop()` goroutine
4. Implement `SendEvent()` API
5. Implement `collectEvents()` method

**Code to Write:** ~90 lines
- Constructor: ~15 lines
- Lifecycle: ~50 lines
- SendEvent: ~25 lines

**Deliverables:**
- [ ] Runtime starts and stops cleanly
- [ ] Tick loop runs at correct rate
- [ ] Events can be queued
- [ ] No goroutine leaks

**Test:**
```go
func TestTickLoopTiming(t *testing.T) {
	machine, _ := statechartx.NewMachine(simpleState)
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate: 10 * time.Millisecond,
	})
	
	ctx := context.Background()
	rt.Start(ctx)
	defer rt.Stop()
	
	// Measure 10 ticks
	start := time.Now()
	startTick := rt.GetTickNumber()
	
	time.Sleep(105 * time.Millisecond) // ~10 ticks
	
	endTick := rt.GetTickNumber()
	elapsed := time.Since(start)
	
	// Should be ~10 ticks in ~100ms
	assert.InDelta(t, 10, endTick-startTick, 2)
	assert.InDelta(t, 100*time.Millisecond, elapsed, 20*time.Millisecond)
}
```

#### Day 5: Event Processing Integration

**Tasks:**
1. Implement `processTick()` orchestration
2. Implement `sortEvents()` sorting logic
3. Implement `processEvents()` loop that calls `Runtime.ProcessEvent()`
4. Implement `processMicrostepsIfNeeded()` that calls `Runtime.ProcessMicrosteps()`

**Code to Write:** ~50 lines
- `realtime/tick.go`: ~30 lines
- `realtime/event.go`: ~20 lines (sorting)

**Deliverables:**
- [ ] Events processed at tick boundaries
- [ ] Microsteps work correctly
- [ ] Simple state transitions work

**Test:**
```go
func TestSimpleTransition(t *testing.T) {
	// State machine: A --[event1]--> B
	machine := createSimpleStateMachine()
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate: 10 * time.Millisecond,
	})
	
	ctx := context.Background()
	rt.Start(ctx)
	defer rt.Stop()
	
	// Should start in state A
	assert.Equal(t, STATE_A, rt.GetCurrentState())
	
	// Send event
	rt.SendEvent(statechartx.Event{ID: EVENT_1})
	
	// Wait for next tick
	time.Sleep(15 * time.Millisecond)
	
	// Should now be in state B
	assert.Equal(t, STATE_B, rt.GetCurrentState())
}
```

**Phase 1 Summary:**
- **Lines Written:** ~200 lines
- **Lines Reused:** ~150 lines (processEvent, processMicrosteps, etc.)
- **Time:** 5 days
- **Deliverable:** Basic tick-based runtime working for simple state machines

---

### Phase 2: Testing and Validation (Week 2, Days 6-10)

**Goal:** Ensure correctness and determinism

#### Day 6-7: Unit Tests

**Tasks:**
1. Write unit tests for event batching
2. Write unit tests for event sorting
3. Write unit tests for tick timing
4. Write unit tests for determinism

**Tests to Write:** ~100 lines

**Key Tests:**
```go
func TestEventOrdering(t *testing.T) {
	// Send 100 events from 10 concurrent goroutines
	// Verify all processed in sequence number order
}

func TestDeterminism(t *testing.T) {
	// Run same event sequence twice
	// Verify identical final state
}

func TestMicrosteps(t *testing.T) {
	// State machine with eventless transitions
	// Verify microsteps processed correctly
}
```

#### Day 8-9: SCXML Conformance Tests

**Tasks:**
1. Adapt existing SCXML test runner for tick-based runtime
2. Run W3C SCXML conformance tests
3. Fix any issues
4. Document any intentional deviations

**Adapter Implementation:** ~50 lines

```go
// testutil/adapter.go
type TickBasedAdapter struct {
	runtime *realtime.RealtimeRuntime
}

func (a *TickBasedAdapter) SendEvent(event statechartx.Event) error {
	a.runtime.SendEvent(event)
	// Wait for next tick to process
	time.Sleep(a.runtime.tickRate + 5*time.Millisecond)
	return nil
}

func (a *TickBasedAdapter) IsInState(stateID string) bool {
	return a.runtime.IsInState(parseStateID(stateID))
}

// Run existing SCXML tests with this adapter
```

#### Day 10: Documentation

**Tasks:**
1. Write package documentation (doc.go)
2. Add code examples
3. Document trade-offs
4. Write migration guide

**Documentation to Write:** ~100 lines

**Phase 2 Summary:**
- **Lines Written:** ~250 lines (tests + docs)
- **Time:** 5 days
- **Deliverable:** Tested and documented basic runtime

---

### Phase 3: Parallel States (Week 3, Days 11-15)

**Goal:** Sequential parallel region processing

#### Day 11-12: Parallel Region Sequential Processing

**Tasks:**
1. Detect when current state is parallel
2. Get sorted list of region IDs
3. Process each region sequentially
4. Reuse existing transition methods per region

**Code to Write:** ~50 lines

```go
// realtime/tick.go additions

func (rt *RealtimeRuntime) processParallelRegionsSequentially() {
	// Get current state
	currentStateID := rt.GetCurrentState()
	currentState := rt.machine.states[currentStateID]
	
	if currentState == nil || !currentState.IsParallel {
		return // Not in parallel state
	}
	
	// Get region IDs in deterministic order (sorted)
	regionIDs := rt.getSortedRegionIDs(currentState)
	
	// Process each region sequentially
	for _, regionID := range regionIDs {
		rt.processRegionSequentially(regionID)
	}
}

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

func (rt *RealtimeRuntime) processRegionSequentially(regionID statechartx.StateID) {
	// Get events for this region (broadcast or targeted)
	events := rt.getRegionEvents(regionID)
	
	// Process each event in region context
	for _, event := range events {
		// Use EXISTING transition methods
		// This is where we reuse pickTransitionHierarchical, computeLCA,
		// exitToLCA, enterFromLCA from embedded Runtime
		rt.processRegionEvent(regionID, event)
	}
}
```

#### Day 13-14: Region Event Handling

**Tasks:**
1. Implement region event routing
2. Track region current states
3. Handle broadcast vs targeted events
4. Implement region-level transition processing

**Code to Write:** ~80 lines

#### Day 15: Parallel State Testing

**Tasks:**
1. Test parallel state entry/exit
2. Test broadcast events to all regions
3. Test targeted events to specific regions
4. Test parallel state completion (all regions final)

**Tests to Write:** ~100 lines

**Phase 3 Summary:**
- **Lines Written:** ~230 lines (code + tests)
- **Lines Reused:** ~200 lines (transition methods per region)
- **Time:** 5 days
- **Deliverable:** Full parallel state support

---

### Phase 4: Polish and Examples (Week 4, Days 16-20)

**Goal:** Production-ready release

#### Day 16-17: Error Handling and Edge Cases

**Tasks:**
1. Handle event queue overflow gracefully
2. Add tick overrun detection
3. Handle zero-duration ticks
4. Add panic recovery in tick loop
5. Comprehensive error messages

**Code to Write:** ~50 lines

#### Day 18-19: Examples and Benchmarks

**Tasks:**
1. Create 60 FPS game loop example
2. Create 1000 Hz physics simulation example
3. Create deterministic replay example
4. Create comparison benchmarks vs event-driven

**Examples to Write:** ~300 lines total

```go
// examples/realtime/game_loop.go
func main() {
	machine := createPlayerStateMachine()
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate: 16667 * time.Microsecond, // 60 FPS
	})
	
	// Game loop at 60 FPS
	// ... (see example in Part 4.4)
}

// examples/realtime/physics_sim.go
func main() {
	machine := createPhysicsStateMachine()
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate: 1 * time.Millisecond, // 1000 Hz
	})
	
	// Physics simulation at 1000 Hz
	// ...
}

// examples/realtime/replay.go
func main() {
	// Record events with sequence numbers
	// Replay in same order
	// Verify identical results
	// ...
}
```

#### Day 20: Final Documentation and Release Prep

**Tasks:**
1. Complete README
2. API reference documentation
3. Migration guide from event-driven
4. Performance comparison document
5. Release notes

**Documentation to Write:** ~200 lines

**Phase 4 Summary:**
- **Lines Written:** ~550 lines (error handling + examples + docs)
- **Time:** 5 days
- **Deliverable:** Production-ready release with examples

---

## Phase Summary Table

| Phase | Duration | New Code | Reused Code | Tests | Docs | Deliverable |
|-------|----------|----------|-------------|-------|------|-------------|
| **Phase 1** | Week 1 | ~200 lines | ~150 lines | ~50 lines | - | Basic runtime |
| **Phase 2** | Week 2 | ~50 lines | ~150 lines | ~100 lines | ~100 lines | Tested runtime |
| **Phase 3** | Week 3 | ~130 lines | ~200 lines | ~100 lines | - | Parallel states |
| **Phase 4** | Week 4 | ~100 lines | - | - | ~300 lines | Production ready |
| **TOTAL** | **4 weeks** | **~480 lines** | **~500 lines** | **~250 lines** | **~400 lines** | **Full release** |

---

## Part 6: API Design

### 6.1 Public API

```go
// Constructor
func NewRuntime(machine *statechartx.Machine, cfg Config) *RealtimeRuntime

// Lifecycle
func (rt *RealtimeRuntime) Start(ctx context.Context) error
func (rt *RealtimeRuntime) Stop() error

// Event Sending
func (rt *RealtimeRuntime) SendEvent(event statechartx.Event) error
func (rt *RealtimeRuntime) SendEventWithPriority(event statechartx.Event, priority int) error

// State Queries (non-blocking, reads from last completed tick)
func (rt *RealtimeRuntime) IsInState(stateID statechartx.StateID) bool
func (rt *RealtimeRuntime) GetCurrentState() statechartx.StateID
func (rt *RealtimeRuntime) GetTickNumber() uint64

// Configuration
type Config struct {
	TickRate         time.Duration  // Required: e.g., 16.67ms for 60 FPS
	MaxEventsPerTick int            // Optional: default 1000
}
```

### 6.2 API Comparison: Event-Driven vs Tick-Based

| Feature | Event-Driven | Tick-Based |
|---------|--------------|------------|
| **Constructor** | `NewRuntime(machine, ext)` | `NewRuntime(machine, cfg)` |
| **Start** | `Start(ctx)` | `Start(ctx)` - same signature |
| **Send Event** | `SendEvent(ctx, event)` - blocks until queued | `SendEvent(event)` - no context, batches for next tick |
| **State Query** | `IsInState(id)` - real-time | `IsInState(id)` - as of last tick |
| **Parallel Regions** | Concurrent goroutines | Sequential processing |
| **Event Processing** | Immediate (async) | Next tick boundary |
| **Determinism** | Best-effort | Guaranteed |

### 6.3 Usage Patterns

#### **Event-Driven Pattern**

```go
rt := statechartx.NewRuntime(machine, nil)
rt.Start(ctx)

// Blocks until event queued (buffered channel)
if err := rt.SendEvent(ctx, event); err != nil {
	log.Error(err)
}

// Immediate state query (may be mid-transition)
state := rt.GetCurrentState()
```

#### **Tick-Based Pattern**

```go
rt := realtime.NewRuntime(machine, realtime.Config{
	TickRate: 16667 * time.Microsecond, // 60 FPS
})
rt.Start(ctx)

// Non-blocking, batches for next tick
if err := rt.SendEvent(event); err != nil {
	log.Error(err) // Only fails if queue full
}

// State query reads from last completed tick (stable)
state := rt.GetCurrentState()
```

---

## Part 7: Testing Strategy

### 7.1 Reuse Existing Tests

**Strategy:** Create a `RuntimeAdapter` interface that both runtimes implement, then run the same tests on both.

```go
// testutil/adapter.go

type RuntimeAdapter interface {
	Start(ctx context.Context) error
	Stop() error
	SendEvent(event statechartx.Event) error
	IsInState(stateID statechartx.StateID) bool
	GetCurrentState() statechartx.StateID
	WaitForStability(timeout time.Duration) error
}

// EventDrivenAdapter wraps event-driven runtime
type EventDrivenAdapter struct {
	rt *statechartx.Runtime
}

func (a *EventDrivenAdapter) SendEvent(event statechartx.Event) error {
	return a.rt.SendEvent(context.Background(), event)
}

func (a *EventDrivenAdapter) WaitForStability(timeout time.Duration) error {
	// Event-driven processes immediately, no wait needed
	time.Sleep(1 * time.Millisecond) // Small delay for goroutine scheduling
	return nil
}

// TickBasedAdapter wraps tick-based runtime
type TickBasedAdapter struct {
	rt *realtime.RealtimeRuntime
}

func (a *TickBasedAdapter) SendEvent(event statechartx.Event) error {
	return a.rt.SendEvent(event)
}

func (a *TickBasedAdapter) WaitForStability(timeout time.Duration) error {
	// Wait for next tick to process event
	tickRate := a.rt.tickRate
	time.Sleep(tickRate + 5*time.Millisecond)
	return nil
}

// Run same test on both runtimes
func RunConformanceTest(t *testing.T, adapter RuntimeAdapter, testCase TestCase) {
	adapter.Start(context.Background())
	defer adapter.Stop()
	
	for _, step := range testCase.Steps {
		adapter.SendEvent(step.Event)
		adapter.WaitForStability(1 * time.Second)
		
		if step.ExpectedState != "" {
			assert.True(t, adapter.IsInState(step.ExpectedState))
		}
	}
}

// Test both runtimes
func TestSCXMLConformance(t *testing.T) {
	testCases := loadSCXMLTests()
	
	for _, testCase := range testCases {
		t.Run("EventDriven/"+testCase.Name, func(t *testing.T) {
			machine := buildMachine(testCase)
			adapter := &EventDrivenAdapter{
				rt: statechartx.NewRuntime(machine, nil),
			}
			RunConformanceTest(t, adapter, testCase)
		})
		
		t.Run("TickBased/"+testCase.Name, func(t *testing.T) {
			machine := buildMachine(testCase)
			adapter := &TickBasedAdapter{
				rt: realtime.NewRuntime(machine, realtime.Config{
					TickRate: 10 * time.Millisecond,
				}),
			}
			RunConformanceTest(t, adapter, testCase)
		})
	}
}
```

### 7.2 New Tests Specific to Tick-Based

```go
// Test determinism
func TestDeterministicExecution(t *testing.T) {
	machine := createTestMachine()
	events := generateTestEvents(100)
	
	// Run 1
	rt1 := realtime.NewRuntime(machine, config)
	rt1.Start(ctx)
	for _, event := range events {
		rt1.SendEvent(event)
	}
	time.Sleep(200 * time.Millisecond)
	state1 := rt1.GetCurrentState()
	rt1.Stop()
	
	// Run 2
	rt2 := realtime.NewRuntime(machine, config)
	rt2.Start(ctx)
	for _, event := range events {
		rt2.SendEvent(event)
	}
	time.Sleep(200 * time.Millisecond)
	state2 := rt2.GetCurrentState()
	rt2.Stop()
	
	// Must be identical
	assert.Equal(t, state1, state2, "Non-deterministic execution detected")
}

// Test tick timing accuracy
func TestTickTimingAccuracy(t *testing.T) {
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate: 10 * time.Millisecond,
	})
	rt.Start(ctx)
	defer rt.Stop()
	
	// Measure 100 ticks
	start := time.Now()
	startTick := rt.GetTickNumber()
	
	time.Sleep(1005 * time.Millisecond) // ~100 ticks
	
	endTick := rt.GetTickNumber()
	elapsed := time.Since(start)
	
	// Should be 100 ticks ±5
	assert.InDelta(t, 100, endTick-startTick, 5)
	
	// Should take ~1000ms ±50ms
	assert.InDelta(t, 1000*time.Millisecond, elapsed, 50*time.Millisecond)
}

// Test event ordering
func TestConcurrentEventOrdering(t *testing.T) {
	rt := realtime.NewRuntime(machine, config)
	rt.Start(ctx)
	defer rt.Stop()
	
	// Send 1000 events from 10 goroutines concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rt.SendEvent(statechartx.Event{
					ID: statechartx.EventID(goroutineID*100 + j),
				})
			}
		}(i)
	}
	wg.Wait()
	
	// Wait for all events to process
	time.Sleep(200 * time.Millisecond)
	
	// Verify all 1000 events were processed
	// (specific verification depends on state machine design)
	// Key: No events lost, deterministic order
}

// Test parallel region determinism
func TestParallelRegionDeterminism(t *testing.T) {
	machine := createParallelStateMachine()
	
	// Run same test twice
	state1 := runParallelTest(machine)
	state2 := runParallelTest(machine)
	
	// Both runs should have identical final state
	assert.Equal(t, state1, state2)
}
```

### 7.3 Test Coverage Goals

- **Unit Tests:** 80%+ coverage
- **SCXML Conformance:** 100% pass rate (same as event-driven)
- **Stress Tests:** Pass 1M states, 1M events, 10K parallel regions
- **Determinism Tests:** 100% reproducibility
- **Timing Tests:** ±5% accuracy

---

## Part 8: Migration Considerations

### 8.1 When to Use Tick-Based vs Event-Driven

**Use Event-Driven When:**
- ✅ Low latency critical (< 1ms)
- ✅ High throughput needed (> 1M events/sec)
- ✅ Async I/O and reactive patterns
- ✅ Variable event rates
- ✅ Web servers, microservices, UI state management

**Use Tick-Based When:**
- ✅ Determinism required (simulations, replays)
- ✅ Fixed update rate needed (games, physics)
- ✅ Reproducible behavior critical (testing, debugging)
- ✅ Temporal consistency required (robotics, control systems)
- ✅ Frame-locked execution (game logic at 60 FPS)

### 8.2 Converting Event-Driven Code to Tick-Based

**Minimal Changes Required:**

```go
// BEFORE (Event-Driven)
rt := statechartx.NewRuntime(machine, nil)
rt.Start(ctx)

for {
	event := getEvent()
	if err := rt.SendEvent(ctx, event); err != nil {
		log.Error(err)
	}
}

// AFTER (Tick-Based)
rt := realtime.NewRuntime(machine, realtime.Config{
	TickRate: 16667 * time.Microsecond, // 60 FPS
})
rt.Start(ctx)

for {
	event := getEvent()
	if err := rt.SendEvent(event); err != nil { // No context
		log.Error(err)
	}
	// Event processed at next tick (no blocking)
}
```

**Key Differences:**

1. **Constructor:** Add `Config` with `TickRate`
2. **SendEvent:** Remove `context.Context` parameter
3. **Event Processing:** Batched at tick boundaries (not immediate)
4. **State Queries:** Read from last completed tick (not real-time)

**State Machine Definition:** NO CHANGES - same `Machine`, `State`, `Transition` types

### 8.3 Gradual Migration Path

**Step 1: Proof of Concept**
- Test one state machine with tick-based runtime
- Verify correctness and determinism
- Measure performance impact
- Decision point: Proceed or stick with event-driven

**Step 2: Parallel Deployment**
- Run both runtimes side-by-side
- Compare results for validation
- Measure performance in production
- Decision point: Full migration or hybrid

**Step 3: Full Migration**
- Replace event-driven with tick-based
- Update event sending callsites (remove context)
- Add tick rate configuration
- Monitor for issues

**Step 4: Optimization**
- Profile tick processing
- Optimize hot paths
- Tune tick rate for workload
- Consider parallel region optimizations (if needed)

### 8.4 Hybrid Approach

**Scenario:** Some state machines need determinism, others need low latency

**Solution:** Use both runtimes in same application

```go
// Deterministic game logic (tick-based)
gameLogicRT := realtime.NewRuntime(gameLogicMachine, realtime.Config{
	TickRate: 16667 * time.Microsecond, // 60 FPS
})

// Reactive UI state (event-driven)
uiStateRT := statechartx.NewRuntime(uiStateMachine, nil)

// Both run concurrently
gameLogicRT.Start(ctx)
uiStateRT.Start(ctx)

// Game logic events queued for ticks
gameLogicRT.SendEvent(gameEvent)

// UI events processed immediately
uiStateRT.SendEvent(ctx, uiEvent)
```

---

## Part 9: Performance Expectations

### 9.1 Throughput Projections

| Tick Rate | Max Events/Tick | Throughput | vs Event-Driven |
|-----------|-----------------|------------|-----------------|
| 60 Hz | 1000 | 60,000 events/sec | 33x slower |
| 120 Hz | 1000 | 120,000 events/sec | 16x slower |
| 1000 Hz | 1000 | 1,000,000 events/sec | 2x slower |
| 60 Hz | 100 | 6,000 events/sec | 333x slower |

**Event-Driven Baseline:** 2M events/sec

**Insight:** Throughput scales with tick rate, but at cost of CPU usage

### 9.2 Latency Projections

| Tick Rate | Avg Latency | Max Latency | vs Event-Driven |
|-----------|-------------|-------------|-----------------|
| 60 Hz | 8.3 ms | 16.67 ms | 38,000x higher |
| 120 Hz | 4.2 ms | 8.33 ms | 19,000x higher |
| 1000 Hz | 0.5 ms | 1 ms | 2,300x higher |
| 10000 Hz | 0.05 ms | 0.1 ms | 230x higher |

**Event-Driven Baseline:** 217 ns

**Insight:** Tick-based latency is always higher, but acceptable for fixed-rate use cases

### 9.3 Memory Projections

| Component | Event-Driven | Tick-Based | Difference |
|-----------|--------------|------------|------------|
| Base Runtime | 0.61 KB | 0.7 KB | +15% |
| Event Queue | ~10 KB | ~128 KB | +12x (1000 events) |
| State Storage | Same | Same | No change |
| **Total** | ~11 KB | ~129 KB | +11x |

**Insight:** Larger event batch capacity increases memory, but still modest (< 1 MB per runtime)

### 9.4 CPU Usage

**Tick-Based CPU Pattern:**
- Fixed CPU usage per tick (predictable)
- Idle between ticks (can sleep)
- CPU usage ∝ tick rate × events per tick

**Event-Driven CPU Pattern:**
- Variable CPU usage (depends on event rate)
- Always active (goroutine waiting on channel)
- CPU usage ∝ event rate

**Trade-off:**
- Tick-based: Consistent CPU load, easier to budget
- Event-driven: More efficient at low event rates

### 9.5 Determinism Guarantees

**Event-Driven:**
- ❌ Event order non-deterministic under concurrency
- ❌ Parallel regions race
- ✅ Transition selection deterministic (document order)

**Tick-Based:**
- ✅ Event order deterministic (sequence number)
- ✅ Parallel regions synchronous (sequential processing)
- ✅ Transition selection deterministic (document order)
- ✅ Reproducible execution (replay with same events)

**Value:** For games, simulations, testing - determinism > throughput

---

## Part 10: Success Criteria

### 10.1 Functional Requirements

- [ ] Simple state transitions work correctly
- [ ] Hierarchical states work correctly
- [ ] Parallel states work correctly (sequential processing)
- [ ] History states work correctly (shallow and deep)
- [ ] Done events work correctly
- [ ] Microsteps work correctly
- [ ] Guard conditions work correctly
- [ ] Actions execute correctly

### 10.2 Performance Requirements

- [ ] Tick accuracy: ±5% at 60 Hz (16.67ms ± 0.83ms)
- [ ] Throughput: > 50,000 events/sec at 60 Hz
- [ ] Latency: < 20ms average at 60 Hz
- [ ] Memory: < 1 MB per runtime instance
- [ ] Handle 1000 events per tick without queue overflow

### 10.3 Quality Requirements

- [ ] 80%+ test coverage
- [ ] Pass all W3C SCXML conformance tests (same as event-driven)
- [ ] Zero known bugs in core functionality
- [ ] No goroutine leaks
- [ ] No memory leaks (stable over 24+ hours)
- [ ] No data races (race detector clean)

### 10.4 Determinism Requirements

- [ ] 100% reproducible execution with same event sequence
- [ ] Event ordering consistent across runs
- [ ] Parallel region execution order consistent
- [ ] Replay accuracy: bit-identical state after replay

### 10.5 Documentation Requirements

- [ ] Package documentation (doc.go) complete
- [ ] API reference complete
- [ ] Code examples working
- [ ] Migration guide complete
- [ ] Trade-offs documented
- [ ] Performance comparison documented

---

## Part 11: Risk Mitigation

### 11.1 Technical Risks

**Risk 1: Tick Timing Inaccuracy**

**Impact:** High - Violates fixed time-step guarantee

**Mitigation:**
- Use `time.NewTicker()` (high precision)
- Monitor tick duration
- Warn if tick takes > 80% of tick budget
- Test on multiple platforms (Linux, macOS, Windows)

**Risk 2: Performance Not Acceptable**

**Impact:** Medium - Users prefer event-driven

**Mitigation:**
- Set performance targets early (50K events/sec at 60 Hz)
- Benchmark continuously during development
- Profile and optimize hot paths
- Document trade-offs clearly

**Risk 3: Code Duplication Creeps In**

**Impact:** Low - Maintenance burden

**Mitigation:**
- Regular code reviews
- Lint for duplication
- Keep embed-and-adapt strategy strict
- Refactor if duplication > 5%

### 11.2 Schedule Risks

**Risk 1: Phase 3 (Parallel States) Takes Longer**

**Impact:** Medium - Delays release

**Mitigation:**
- Implement parallel states last (Phases 1-2 still useful)
- Sequential processing simpler than goroutine coordination
- Can release without parallel states (document limitation)

**Risk 2: Testing Takes Longer Than Expected**

**Impact:** Low - Delays release but improves quality

**Mitigation:**
- Reuse existing test suite (RuntimeAdapter pattern)
- Test continuously during development
- Automate testing in CI/CD

### 11.3 Adoption Risks

**Risk 1: Users Don't Need Tick-Based Runtime**

**Impact:** High - Wasted effort

**Mitigation:**
- Validate use cases before starting (already done)
- Build examples showing value (game loop, replay)
- Document trade-offs clearly
- Keep event-driven as default recommendation

**Risk 2: API Too Different from Event-Driven**

**Impact:** Medium - Migration friction

**Mitigation:**
- Minimize API differences (only SendEvent signature changes)
- Provide migration guide
- Show side-by-side code examples
- Support both runtimes in same application

---

## Part 12: Next Steps

### 12.1 Immediate Actions (Day 1)

1. **Review this plan** with stakeholders
2. **Set up realtime/ package** directory
3. **Add public method aliases** to statechart.go (~20 lines)
4. **Create runtime.go stub** with struct definition
5. **Run initial compile check**

### 12.2 Week 1 Checklist

- [ ] Day 1-2: Package setup and core structs
- [ ] Day 3-4: Tick loop and event batching
- [ ] Day 5: Event processing integration
- [ ] End of Week 1: Basic tick-based runtime working

### 12.3 Week 2 Checklist

- [ ] Day 6-7: Unit tests
- [ ] Day 8-9: SCXML conformance tests
- [ ] Day 10: Documentation
- [ ] End of Week 2: Tested and documented

### 12.4 Week 3 Checklist

- [ ] Day 11-12: Parallel region sequential processing
- [ ] Day 13-14: Region event handling
- [ ] Day 15: Parallel state testing
- [ ] End of Week 3: Full feature parity

### 12.5 Week 4 Checklist

- [ ] Day 16-17: Error handling and edge cases
- [ ] Day 18-19: Examples and benchmarks
- [ ] Day 20: Final documentation
- [ ] End of Week 4: Production-ready release

---

## Appendix A: Code Reuse Breakdown

### Methods Reused from statechart.go (Zero Modifications)

| Method | Lines | Purpose |
|--------|-------|---------|
| `processEvent()` | 710-779 (70 lines) | External transition handling |
| `processMicrosteps()` | 784-861 (78 lines) | Eventless transitions |
| `computeLCA()` | 874-902 (29 lines) | Least Common Ancestor |
| `exitToLCA()` | 904-921 (18 lines) | Exit states to LCA |
| `enterFromLCA()` | 923-956 (34 lines) | Enter states from LCA |
| `pickTransitionHierarchical()` | 1063-1117 (55 lines) | Find matching transition |
| `recordHistory()` | 1202-1214 (13 lines) | Record history state |
| `restoreHistory()` | 1231-1236 (6 lines) | Restore history |
| `restoreShallowHistory()` | 1238-1263 (26 lines) | Shallow history |
| `restoreDeepHistory()` | 1265-1295 (31 lines) | Deep history |
| `checkFinalState()` | 958-979 (22 lines) | Check if final |
| `shouldEmitDoneEvent()` | 1023-1031 (9 lines) | Should emit done |
| `allRegionsInFinalState()` | 1033-1054 (22 lines) | All regions final |
| **TOTAL** | **413 lines** | **Reused with zero changes** |

### Methods Modified (Minor Adaptations)

| Method | Original Lines | Modification | New Lines |
|--------|----------------|--------------|-----------|
| `generateDoneEvent()` | 981-1021 (41 lines) | Change async send to batch append | ~5 lines changed |

### Total Reuse: 418 lines with 5 lines modified = 99% reuse of core logic

---

## Appendix B: Complete File Structure

```
statechartx_review/
├── statechart.go                        # EXISTING (1300 lines, +20 lines for public aliases)
├── realtime/
│   ├── runtime.go                       # NEW (150 lines)
│   ├── tick.go                          # NEW (50 lines)
│   ├── event.go                         # NEW (30 lines)
│   ├── doc.go                           # NEW (20 lines)
│   └── realtime_test.go                 # NEW (200 lines)
├── examples/
│   └── realtime/
│       ├── game_loop.go                 # NEW (100 lines)
│       ├── physics_sim.go               # NEW (100 lines)
│       └── replay.go                    # NEW (100 lines)
├── benchmarks/
│   └── realtime_bench_test.go           # NEW (150 lines)
├── testutil/
│   └── adapter.go                       # NEW (100 lines) - For running shared tests
└── README_REALTIME.md                   # NEW (100 lines)

TOTAL NEW CODE: ~1,000 lines
TOTAL REUSED CODE: ~418 lines
TOTAL CODE DUPLICATION: 0 lines
```

---

## Appendix C: Quick Reference

### Start Tick-Based Runtime

```go
rt := realtime.NewRuntime(machine, realtime.Config{
	TickRate: 16667 * time.Microsecond, // 60 FPS
	MaxEventsPerTick: 1000,
})
rt.Start(ctx)
defer rt.Stop()
```

### Send Events

```go
// Basic event
rt.SendEvent(statechartx.Event{ID: EVENT_ID, Data: data})

// Priority event
rt.SendEventWithPriority(event, priority)
```

### Query State

```go
// Current state (as of last tick)
state := rt.GetCurrentState()

// Is in state
if rt.IsInState(STATE_ID) {
	// ...
}

// Tick number
tick := rt.GetTickNumber()
```

### Key Methods Called from Embedded Runtime

```go
// These are called internally by RealtimeRuntime:
rt.Runtime.ProcessEvent(event)           // Process single event
rt.Runtime.ProcessMicrosteps(ctx)        // Process eventless transitions
rt.Runtime.ComputeLCA(from, to)          // Compute LCA
rt.Runtime.ExitToLCA(ctx, evt, f, t, l)  // Exit to LCA
rt.Runtime.EnterFromLCA(ctx, evt, f, t, l) // Enter from LCA
rt.Runtime.PickTransitionHierarchical(state, event) // Find transition
```

---

## Conclusion

This implementation plan provides a **concrete, actionable roadmap** for building the tick-based real-time runtime using the **embed-and-adapt approach**. By reusing ~418 lines of existing synchronous core logic, we achieve:

- **77% less new code** (230 lines vs 1000 lines in original plan)
- **Zero code duplication**
- **3-4 weeks to production** (vs 12 weeks)
- **Battle-tested correctness** (reuses proven transition logic)
- **Easy maintenance** (single source of truth for core logic)

**Next Action:** Begin Phase 1, Day 1 - Set up package and create runtime.go stub.

---

**Document Version:** 1.0  
**Date:** January 2, 2026  
**Status:** Ready for Implementation  
**Estimated Completion:** End of Week 4
