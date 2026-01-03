# StatechartX Real-Time Runtime: Architectural Assessment

**Date:** January 2, 2026  
**Purpose:** Evaluate real-time extension plan and maximize core reuse  
**Priority:** Concise, Readable, Performant

---

## Executive Summary

**Critical Finding: The current core is ALREADY largely synchronous and highly reusable.**

Your intuition is correct—the proposed plan unnecessarily re-implements substantial core functionality that can and should be reused directly. The synchronous FSM core (lines 710-1143 of `statechart.go`) contains ~430 lines of well-tested, deterministic logic that applies equally to tick-based execution.

**Key Insights:**

1. **Goroutines are minimal** - Only ONE goroutine for the event loop (line 696-707), and one per parallel region
2. **Core transition logic is pure** - No goroutine dependencies in the critical path
3. **The event loop is the ONLY fundamental difference** - Replace async channel reads with tick-based batch processing
4. **~60% of proposed code is duplication** - Most transition/state logic already exists

**Recommendation:** Build a thin wrapper around the existing core, replacing only the event dispatch mechanism.

---

## Part 1: Current Architecture Analysis

### 1.1 Synchronous Core Components (REUSABLE)

The following components are **pure synchronous logic** with no goroutine coupling:

#### **State Transition Engine** (~300 lines)
```go
// Lines 710-779: processEvent - External transition handling
// Lines 784-861: processMicrosteps - Eventless transitions
// Lines 874-902: computeLCA - Least Common Ancestor
// Lines 904-921: exitToLCA - Exit states to LCA
// Lines 923-956: enterFromLCA - Enter states from LCA
// Lines 1063-1117: pickTransition + pickTransitionHierarchical
```

**Characteristics:**
- ✅ Zero goroutines
- ✅ Zero channels
- ✅ Zero mutexes
- ✅ Pure state machine semantics
- ✅ Thoroughly tested (SCXML conformance)

**Reusability:** **100%** - Can be called directly from tick-based runtime

#### **History State Management** (~100 lines)
```go
// Lines 1200-1306: History recording and restoration
// - recordHistory (lines 1202-1214)
// - restoreHistory (lines 1231-1236)
// - restoreShallowHistory (lines 1238-1263)
// - restoreDeepHistory (lines 1265-1295)
```

**Reusability:** **100%** - Pure logic, no concurrency primitives

#### **Done Event Management** (~100 lines)
```go
// Lines 958-1061: Done event generation and tracking
// - checkFinalState (lines 958-979)
// - generateDoneEvent (lines 981-1021)
// - shouldEmitDoneEvent (lines 1023-1031)
// - allRegionsInFinalState (lines 1033-1054)
```

**Reusability:** **95%** - Only lines 1013-1020 need modification (async event queuing → tick queuing)

#### **Hierarchy Traversal** (~30 lines)
```go
// Lines 203-229: findDeepestInitial
// Lines 863-872: getAncestors
```

**Reusability:** **100%** - Pure hierarchy navigation

### 1.2 Async Components (NEED ADAPTATION)

#### **Event Loop** (12 lines)
```go
// Lines 695-707: eventLoop goroutine
func (rt *Runtime) eventLoop() {
    defer rt.wg.Done()
    for {
        select {
        case <-rt.ctx.Done():
            return
        case event := <-rt.eventQueue:
            rt.processEvent(event)  // ← This calls the REUSABLE core
        }
    }
}
```

**Analysis:** 
- This is the ONLY place that introduces async behavior for sequential states
- The `processEvent` call is to reusable synchronous code
- Tick-based just needs to replace the channel loop with batch processing

**Replacement Strategy:**
```go
// Tick-based equivalent (pseudocode)
func (rt *RealtimeRuntime) processTick() {
    // Collect all events queued this tick
    events := rt.collectAndSortEvents()
    
    // Process each event using EXISTING core logic
    for _, event := range events {
        rt.processEvent(event)  // ← REUSE existing method
    }
}
```

#### **Parallel Region Goroutines** (~200 lines)
```go
// Lines 322-399: enterParallelState - Spawns goroutines
// Lines 401-453: exitParallelState - Stops goroutines
// Lines 477-575: parallelRegion.run - Region event loop
// Lines 609-646: sendEventToRegions - Event routing
```

**Analysis:**
- Goroutines enable true parallelism for parallel states
- **However**: The actual transition logic (lines 506-575) calls reusable core methods
- Only the event routing and coordination need change for tick-based

**Reusability:** **70%** - Reuse transition logic, replace coordination

### 1.3 Goroutine Usage Summary

**Total Goroutines in Current Implementation:**
1. **ONE** goroutine for main event loop (line 263)
2. **ONE** goroutine per parallel region (line 364)
3. **ONE** goroutine for done event sending (line 1013) - negligible

**Synchronous Core Size:**
- Pure state transition logic: ~430 lines
- Goroutine coordination: ~70 lines
- Ratio: **86% synchronous, 14% async**

**Conclusion:** The core is ALREADY predominantly synchronous. The event-driven nature comes almost entirely from the 12-line event loop.

---

## Part 2: Proposed Plan Analysis

### 2.1 What the Plan Proposes to Re-Implement

Examining the proposed `realtime/` package structure:

#### **runtime.go (~300 lines proposed)**
- Runtime struct and lifecycle
- Event sending API
- State queries
- Hook management

**Analysis:** 
- Runtime struct: Some new fields needed (tick buffers), but Machine reference reusable
- Lifecycle: Different enough (ticker vs goroutine)
- Event API: Different (no context, tick-queued)
- State queries: **CAN REUSE** existing methods

**Verdict:** ~40% new, ~60% reusable patterns

#### **tick.go (~200 lines proposed)**
- `processTick()` - Orchestrates tick execution
- `collectEvents()` - Gathers queued events
- `sortEvents()` - Orders events deterministically
- `processEvents()` - Batch processes events
- `commitTick()` - Double buffer swap

**Analysis:**
- `processEvents()` can call existing `processEvent()`
- `collectEvents()` and `sortEvents()` are new (event ordering)
- `commitTick()` is new (double buffering)
- **CRITICAL**: The plan shows `processEvent()` being re-implemented (~100 lines)

**Verdict:** ~60% new (batching/buffering), ~40% should reuse existing core

#### **state.go (~150 lines proposed)**
- TickState struct
- Double buffering logic
- State cloning

**Analysis:**
- All new (double buffering is tick-based specific)
- But should READ from existing Machine state structures

**Verdict:** ~100% new (justifiable)

#### **transition.go (~200 lines proposed)**
- Transition evaluation
- LCA computation
- Exit/entry logic

**Analysis:**
- **THIS IS THE PROBLEM** - These are already implemented (lines 710-956)!
- LCA computation: Already exists (lines 874-902)
- Exit logic: Already exists (lines 904-921)
- Entry logic: Already exists (lines 923-956)
- Transition evaluation: Already exists (lines 1063-1117)

**Verdict:** ~5% new, ~95% should reuse

#### **parallel.go (~200 lines proposed)**
- Sequential parallel region processing
- Region state tracking
- Deterministic region ordering

**Analysis:**
- Region tracking: New for tick-based (no goroutines)
- But transition logic within regions: **SHOULD REUSE** existing
- Current `processRegionEvent` (lines 505-575): Reusable core with different coordination

**Verdict:** ~50% new, ~50% should reuse

### 2.2 Duplication Analysis

**Estimated Code Duplication:**

| Category | Lines in Plan | Reusable Lines | Duplication % |
|----|----:|----:|----:|
| State transition core | 200 | 180 | 90% |
| Hierarchy traversal | 50 | 45 | 90% |
| History states | 100 | 95 | 95% |
| Done events | 80 | 60 | 75% |
| Guard/action execution | 40 | 40 | 100% |
| Microsteps | 80 | 75 | 94% |
| **Total** | **550** | **495** | **90%** |

**Conclusion:** The plan would re-implement approximately **~500 lines of already-tested, production-ready code**.

### 2.3 What's Actually Different for Tick-Based

**True Differences (Justifiable New Code):**

1. **Tick scheduler** (~50 lines)
   - Fixed-rate ticker
   - Tick counter
   - Tick loop management

2. **Event batching** (~100 lines)
   - Collect events between ticks
   - Deterministic sorting
   - Priority handling

3. **Double buffering** (~150 lines)
   - TickState struct
   - Buffer swapping
   - State cloning/copying

4. **Sequential parallel coordination** (~100 lines)
   - Replace goroutines with sequential iteration
   - Deterministic region ordering
   - Region state tracking

**Total Justifiable New Code:** ~400 lines

**Total Proposed Code:** ~1,000 lines

**Duplication:** ~600 lines (60%)

---

## Part 3: Event Ordering Guarantees

### 3.1 Current Event Ordering Characteristics

**Event-Driven Runtime Ordering:**

```go
// Line 596-606: SendEvent
select {
case rt.eventQueue <- event:  // ← Buffered channel (cap 100)
    return nil
case <-ctx.Done():
    return ctx.Err()
}

// Line 700-705: eventLoop
select {
case event := <-rt.eventQueue:  // ← FIFO from channel
    rt.processEvent(event)
}
```

**Guarantees:**
- ✅ **Single source FIFO**: Events from one goroutine maintain order
- ✅ **Microstep determinism**: Eventless transitions processed deterministically (lines 784-861)
- ✅ **Transition selection determinism**: Document order (lines 1063-1117)
- ❌ **Multi-source ordering**: No guarantee when multiple goroutines send concurrently
- ❌ **Parallel region sync**: Regions process independently

**Why Non-Deterministic:**
1. Go channel `select` with multiple senders has undefined order
2. Parallel regions run on separate goroutines with independent event queues
3. No synchronization points between regions

### 3.2 Tick-Based Ordering Solution

**Tick-Based Guarantees:**

```go
// Tick N-1 to Tick N transition
func (rt *RealtimeRuntime) processTick() {
    // Phase 1: Collect all events (thread-safe append to slice)
    events := rt.collectEvents()  // Atomic operation, preserves append order
    
    // Phase 2: Sort deterministically
    sort.SliceStable(events, func(i, j int) bool {
        // Priority → Sequence Number → Source ID
        if events[i].Priority != events[j].Priority {
            return events[i].Priority > events[j].Priority
        }
        if events[i].SequenceNum != events[j].SequenceNum {
            return events[i].SequenceNum < events[j].SequenceNum
        }
        return events[i].SourceID < events[j].SourceID
    })
    
    // Phase 3: Process in sorted order (REUSE existing core)
    for _, event := range events {
        rt.processEvent(event)  // ← Calls existing synchronous core
    }
    
    // Phase 4: Process parallel regions sequentially
    for _, regionID := range rt.getSortedRegionIDs() {
        rt.processRegion(regionID)  // ← Calls existing core per region
    }
}
```

**New Guarantees:**
- ✅ **Multi-source ordering**: Deterministic sort with stable tie-breaking
- ✅ **Parallel region sync**: Sequential processing, deterministic order
- ✅ **Tick boundaries**: All state changes visible simultaneously
- ✅ **Replay precision**: Exact reproduction with same event sequence

**What Doesn't Change:**
- Microstep logic (still uses existing `processMicrosteps`)
- Transition selection (still uses existing `pickTransitionHierarchical`)
- State entry/exit (still uses existing `exitToLCA` / `enterFromLCA`)

---

## Part 4: What CAN Be Reused

### 4.1 Reuse Strategy: Core Extraction

**Option 1: Direct Reuse (Recommended)**

Extract core methods into interface that both runtimes implement:

```go
// core/transition.go (NEW internal package)
type TransitionEngine interface {
    ProcessEvent(event Event) error
    ProcessMicrosteps() error
    ExitToLCA(event *Event, from, to, lca StateID)
    EnterFromLCA(event *Event, from, to, lca StateID)
    ComputeLCA(from, to StateID) StateID
    PickTransitionHierarchical(state *State, event Event) *Transition
}

// Implemented by:
type transitionEngine struct {
    machine *Machine
    getCurrentState func() StateID
    setCurrentState func(StateID)
    recordHistory func(StateID, StateID)
    checkFinalState func(context.Context)
}

// Both runtimes compose this engine:

// Event-driven runtime
type Runtime struct {
    engine *transitionEngine  // ← Shared core
    eventQueue chan Event
    // ... async-specific fields
}

// Tick-based runtime
type RealtimeRuntime struct {
    engine *transitionEngine  // ← Same shared core
    tickN_1 *TickState
    tickN   *TickState
    // ... tick-specific fields
}
```

**Lines Reused:** ~430 lines of battle-tested code

**Option 2: Embedding (Alternative)**

```go
// Embed Runtime and override only what's needed
type RealtimeRuntime struct {
    *Runtime  // Embed event-driven runtime
    
    // Tick-specific additions
    tickRate time.Duration
    eventBatch []Event
    // ...
}

// Override only event dispatch
func (rt *RealtimeRuntime) Start(ctx context.Context) error {
    // Different: tick loop instead of eventLoop goroutine
    ticker := time.NewTicker(rt.tickRate)
    for {
        select {
        case <-ticker.C:
            rt.processTick()  // Custom tick logic
        case <-ctx.Done():
            return nil
        }
    }
}

func (rt *RealtimeRuntime) processTick() {
    events := rt.collectAndSortEvents()
    for _, event := range events {
        rt.processEvent(event)  // ← REUSE embedded Runtime method
    }
}
```

**Lines Reused:** ~500 lines (most of Runtime)

### 4.2 Concrete Reusable Methods

**From statechart.go, directly callable:**

```go
// State transition core
rt.processEvent(event)           // Line 710 - REUSE AS-IS
rt.processMicrosteps(ctx)        // Line 784 - REUSE AS-IS
rt.computeLCA(from, to)          // Line 875 - REUSE AS-IS
rt.exitToLCA(ctx, evt, f, t, l)  // Line 905 - REUSE AS-IS
rt.enterFromLCA(ctx, evt, f, t, l) // Line 924 - REUSE AS-IS
rt.pickTransitionHierarchical(s, e) // Line 1105 - REUSE AS-IS

// History states
rt.recordHistory(parentID, childID) // Line 1203 - REUSE AS-IS
rt.restoreHistory(ctx, state, evt, from) // Line 1231 - REUSE AS-IS
rt.restoreShallowHistory(...)    // Line 1238 - REUSE AS-IS
rt.restoreDeepHistory(...)       // Line 1265 - REUSE AS-IS

// Hierarchy
rt.getAncestors(stateID)         // Line 864 - REUSE AS-IS
machine.findDeepestInitial(id)   // Line 204 - REUSE AS-IS

// Done events (with minor modification)
rt.checkFinalState(ctx)          // Line 959 - Modify line 1013-1020
rt.shouldEmitDoneEvent(parent)   // Line 1024 - REUSE AS-IS
rt.allRegionsInFinalState(state) // Line 1034 - REUSE AS-IS
```

**Modification Needed: Done Event Generation**

Only lines 1013-1020 need change:

```go
// CURRENT (async):
go func() {
    select {
    case rt.eventQueue <- doneEvent:  // ← Async channel send
    case <-ctx.Done():
    case <-time.After(100 * time.Millisecond):
    }
}()

// TICK-BASED (sync):
rt.tickEventQueue = append(rt.tickEventQueue, doneEvent)  // ← Append to batch
```

**Lines Requiring Change:** 8 lines out of ~1,300

### 4.3 Parallel State Reuse

**Current Parallel Processing:**

```go
// Lines 505-575: processRegionEvent
func (r *parallelRegion) processEvent(event Event, state *State) {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    // Find matching transition
    transition := r.runtime.pickTransitionHierarchical(...)  // ← REUSABLE
    
    // Compute LCA
    lca := r.runtime.computeLCA(from, to)  // ← REUSABLE
    
    // Exit states
    r.runtime.exitToLCA(...)  // ← REUSABLE
    
    // Execute transition action
    if transition.Action != nil {
        transition.Action(...)  // ← REUSABLE
    }
    
    // Enter states
    r.runtime.enterFromLCA(...)  // ← REUSABLE
    
    // Update current state
    r.currentState = to
}
```

**Tick-Based Parallel Processing:**

```go
// Sequential processing, same core logic
func (rt *RealtimeRuntime) processRegion(regionID StateID) {
    // Get region events (from broadcast or targeted)
    events := rt.getRegionEventsForTick(regionID)
    
    // Process each event using SAME core methods
    for _, event := range events {
        currentState := rt.tickN_1.regionStates[regionID]
        state := rt.machine.states[currentState]
        
        // REUSE existing methods
        transition := rt.pickTransitionHierarchical(state, event)
        if transition == nil { continue }
        
        lca := rt.computeLCA(from, to)
        rt.exitToLCA(&event, from, to, lca)
        
        if transition.Action != nil {
            transition.Action(rt.ctx, &event, from, to)
        }
        
        rt.enterFromLCA(&event, from, to, lca)
        
        // Write to tick N buffer
        rt.tickN.regionStates[regionID] = to
    }
}
```

**Reuse:** ~70% of logic is identical calls to existing methods

---

## Part 5: Critique and Recommendations

### 5.1 Critical Issues with Current Plan

#### **Issue 1: Massive Code Duplication (60%)**

**Problem:** Plan proposes ~600 lines of duplicated logic already in core

**Impact:**
- Maintenance burden (fix bugs in two places)
- Test duplication
- Divergent implementations over time
- Increased surface area for bugs

**Solution:** Extract shared core into internal package, both runtimes compose it

#### **Issue 2: Re-Implementing Battle-Tested Logic**

**Problem:** Current core has:
- 14+ test files
- W3C SCXML conformance tests
- Stress tests (1M states, events, transitions)
- Production hardening

**Impact:**
- Lose confidence in correctness
- Repeat debugging of edge cases
- Risk introducing new bugs
- Delay to production readiness

**Solution:** Reuse existing methods directly

#### **Issue 3: Double Buffering Complexity May Be Unnecessary**

**Problem:** Plan proposes read-from-tick-N-1 / write-to-tick-N double buffering

**Analysis:**
- Current core ALREADY processes events atomically (single-threaded event loop)
- Tick-based can process events sequentially within tick (no concurrency)
- Double buffering adds ~150 lines and 2x memory

**Question:** Is double buffering actually needed?

**Alternative:** Process events in-place within tick, no buffering needed

```go
func (rt *RealtimeRuntime) processTick() {
    // Collect and sort events
    events := rt.collectAndSortEvents()
    
    // Process sequentially using existing core (NO BUFFERING)
    rt.mu.Lock()
    for _, event := range events {
        rt.processEvent(event)  // ← Existing method, atomic state updates
    }
    rt.mu.Unlock()
    
    // Process parallel regions sequentially (NO BUFFERING)
    for _, regionID := range rt.getSortedRegionIDs() {
        rt.processRegionSequentially(regionID)
    }
}
```

**Benefits:**
- No double buffering overhead
- Simpler implementation
- Less memory usage
- Still deterministic (sequential processing)

**When double buffering IS useful:**
- Rollback/replay (but that's a submodule, not core)
- Reading state during tick (but you shouldn't - wait for tick completion)

**Recommendation:** Start without double buffering, add later if needed

#### **Issue 4: Submodule Boundary Confusion**

**Problem:** Core scope includes replay, distributed, interpolation hooks

**Impact:**
- Core bloat
- Premature abstraction
- YAGNI violations

**Recommendation:** Core should have NO hooks initially. Add hooks only when first submodule needs them (Phase 7+)

### 5.2 Recommended Architecture

#### **Minimal Tick-Based Runtime (Recommended)**

```
realtime/
├── runtime.go       # ~150 lines - Runtime struct, Start/Stop, SendEvent
├── tick.go          # ~100 lines - processTick, collectEvents, sortEvents
└── parallel.go      # ~50 lines - Sequential parallel processing

Total: ~300 lines (vs 1000 in plan)
```

**Key Insight:** Most logic is REUSED from existing Runtime methods

**runtime.go (simplified):**

```go
package realtime

import (
    "context"
    "sort"
    "time"
    "github.com/yourorg/statechartx"
)

type RealtimeRuntime struct {
    // Embed or compose existing runtime core
    *statechartx.Runtime  // ← REUSE existing core
    
    // Tick-specific fields
    tickRate      time.Duration
    ticker        *time.Ticker
    eventBatch    []eventWithMetadata
    mu            sync.Mutex
    sequenceNum   uint64
    
    // Parallel state tracking (sequential)
    regionOrder   []statechartx.StateID
}

type eventWithMetadata struct {
    Event       statechartx.Event
    Priority    int
    SequenceNum uint64
}

func NewRuntime(machine *statechartx.Machine, tickRate time.Duration) *RealtimeRuntime {
    return &RealtimeRuntime{
        Runtime:   statechartx.NewRuntime(machine, nil),  // ← REUSE
        tickRate:  tickRate,
        eventBatch: make([]eventWithMetadata, 0, 100),
    }
}

func (rt *RealtimeRuntime) Start(ctx context.Context) error {
    // Enter initial state using existing method
    if err := rt.Runtime.Start(ctx); err != nil {  // ← REUSE
        return err
    }
    
    // Start tick loop (only difference from event-driven)
    rt.ticker = time.NewTicker(rt.tickRate)
    go rt.tickLoop(ctx)
    return nil
}

func (rt *RealtimeRuntime) SendEvent(event statechartx.Event) error {
    rt.mu.Lock()
    defer rt.mu.Unlock()
    
    rt.eventBatch = append(rt.eventBatch, eventWithMetadata{
        Event:       event,
        SequenceNum: rt.sequenceNum,
    })
    rt.sequenceNum++
    return nil
}

func (rt *RealtimeRuntime) tickLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case <-rt.ticker.C:
            rt.processTick()
        }
    }
}
```

**tick.go (simplified):**

```go
func (rt *RealtimeRuntime) processTick() {
    // Phase 1: Collect events
    events := rt.collectEvents()
    
    // Phase 2: Sort deterministically
    rt.sortEvents(events)
    
    // Phase 3: Process using EXISTING core methods
    for _, eventMeta := range events {
        rt.Runtime.processEvent(eventMeta.Event)  // ← REUSE existing method
    }
    
    // Phase 4: Process microsteps using EXISTING method
    rt.Runtime.processMicrosteps(context.Background())  // ← REUSE
    
    // Phase 5: Process parallel regions sequentially
    if len(rt.regionOrder) > 0 {
        rt.processParallelRegionsSequentially()
    }
}

func (rt *RealtimeRuntime) collectEvents() []eventWithMetadata {
    rt.mu.Lock()
    defer rt.mu.Unlock()
    
    events := rt.eventBatch
    rt.eventBatch = make([]eventWithMetadata, 0, 100)
    return events
}

func (rt *RealtimeRuntime) sortEvents(events []eventWithMetadata) {
    sort.SliceStable(events, func(i, j int) bool {
        // FIFO by sequence number
        return events[i].SequenceNum < events[j].SequenceNum
    })
}
```

**parallel.go (simplified):**

```go
func (rt *RealtimeRuntime) processParallelRegionsSequentially() {
    // Process each region in deterministic order
    for _, regionID := range rt.regionOrder {
        rt.processRegion(regionID)
    }
}

func (rt *RealtimeRuntime) processRegion(regionID statechartx.StateID) {
    // Get region's events for this tick
    events := rt.getRegionEvents(regionID)
    
    // Process each event using EXISTING core
    for _, event := range events {
        // This calls existing pickTransitionHierarchical, computeLCA,
        // exitToLCA, enterFromLCA - ALL REUSED
        rt.processRegionEvent(regionID, event)
    }
}
```

**Total New Code: ~300 lines**
**Reused Code: ~500 lines (existing Runtime methods)**
**Duplication: ~0 lines**

### 5.3 Specific Recommendations

#### **Recommendation 1: Start with Embedding**

**Phase 1 (Week 1):**
```go
type RealtimeRuntime struct {
    *statechartx.Runtime  // Embed existing runtime
    tickRate time.Duration
    ticker   *time.Ticker
    // ... tick-specific fields
}
```

**Benefits:**
- Immediate access to all existing methods
- Override only what's different (Start, SendEvent)
- Minimal code (~200 lines)
- Proves out reuse strategy

**Later (Phase 3-4):** If needed, extract common core into internal package for both to use

#### **Recommendation 2: Skip Double Buffering (Initially)**

**Why:**
- Current core is already atomic (mutex-protected processEvent)
- Sequential tick processing has no concurrency
- Adds complexity and memory overhead
- Not needed for determinism

**When to add:**
- Phase 7+ (replay submodule needs snapshots)
- Performance testing shows read contention (unlikely)

**Estimated savings:** ~150 lines, 2x memory reduction

#### **Recommendation 3: Parallel State Strategy**

**Current parallel implementation:**
- Goroutines per region (lines 322-575)
- Independent event queues
- Concurrent processing

**Tick-based parallel implementation:**
- NO goroutines
- Sequential processing of regions
- Deterministic region order (sort by StateID)
- REUSE existing transition methods per region

**Code:**
```go
func (rt *RealtimeRuntime) processParallelRegionsSequentially() {
    regionIDs := rt.getSortedRegionIDs()  // Deterministic order
    
    for _, regionID := range regionIDs {
        // Process region using EXISTING core methods
        region := rt.machine.states[regionID]
        events := rt.getRegionEventsForTick(regionID)
        
        for _, event := range events {
            // REUSE: pickTransitionHierarchical, computeLCA,
            // exitToLCA, enterFromLCA
            rt.processRegionTransition(regionID, event)
        }
    }
}
```

**Lines:** ~50 lines new, ~200 lines reused

#### **Recommendation 4: Testing Strategy**

**Reuse existing tests:**
- W3C SCXML conformance tests (same state machine semantics)
- Stress tests (1M states, events, transitions)
- Unit tests for transitions, guards, actions

**New tests:**
- Tick timing accuracy
- Event ordering determinism
- Replay accuracy (later, submodule)

**Test adapter pattern:**
```go
type RuntimeAdapter interface {
    Start(ctx) error
    SendEvent(event) error
    GetCurrentState() StateID
}

// Runs SAME test on both runtimes
func TestSCXMLConformance(t *testing.T) {
    testFiles := loadSCXMLTests()
    
    for _, file := range testFiles {
        t.Run("EventDriven/"+file, func(t *testing.T) {
            rt := statechartx.NewRuntime(...)
            runSCXMLTest(t, rt, file)  // ← Same test
        })
        
        t.Run("TickBased/"+file, func(t *testing.T) {
            rt := realtime.NewRuntime(...)
            runSCXMLTest(t, rt, file)  // ← Same test
        })
    }
}
```

**Estimated test reuse:** ~80%

#### **Recommendation 5: Phased Approach**

**Phase 1: Minimal Viable (Week 1)**
- Embed existing Runtime
- Replace event loop with tick loop
- Event batching and ordering
- **Deliverable:** 60 FPS deterministic execution for simple states

**Phase 2: Hierarchical (Week 2)**
- Verify existing hierarchy methods work
- Test nested states
- **Deliverable:** Hierarchical state machines work

**Phase 3: Parallel (Week 3)**
- Sequential parallel region processing
- **Deliverable:** Parallel states work deterministically

**Phase 4: Polish (Week 4)**
- Error handling
- Edge cases
- Performance optimization
- **Deliverable:** Production ready

**Submodules (Phase 5+, Weeks 5+)**
- Replay (Week 5-6)
- Debug (Week 7-8)
- Others as needed

**Total to production:** 4 weeks (vs 12 in plan)

---

## Part 6: Event Ordering Deep Dive

### 6.1 Why Current Event-Driven is Non-Deterministic

**Scenario: Two goroutines send events concurrently**

```go
// Goroutine 1
go func() {
    rt.SendEvent(ctx, Event{ID: 1, Data: "A"})
}()

// Goroutine 2
go func() {
    rt.SendEvent(ctx, Event{ID: 2, Data: "B"})
}()

// Current implementation (lines 596-606):
func (rt *Runtime) SendEvent(ctx context.Context, event Event) error {
    select {
    case rt.eventQueue <- event:  // ← Unbuffered channel select
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

**Problem:** Go's channel select with multiple senders has undefined order
- Event 1 might arrive first, or Event 2 might
- Depends on goroutine scheduler, system load, cosmic rays
- **Result:** Non-deterministic state transitions

### 6.2 Tick-Based Solution

**Same scenario with tick-based:**

```go
// Goroutine 1
go func() {
    rt.SendEvent(Event{ID: 1, Data: "A"})  // No context
}()

// Goroutine 2
go func() {
    rt.SendEvent(Event{ID: 2, Data: "B"})
}()

// Tick-based implementation:
func (rt *RealtimeRuntime) SendEvent(event Event) error {
    rt.mu.Lock()
    defer rt.mu.Unlock()
    
    rt.eventBatch = append(rt.eventBatch, eventWithMetadata{
        Event:       event,
        SequenceNum: rt.sequenceNum,  // ← Atomic counter
    })
    rt.sequenceNum++
    return nil
}
```

**Guarantee:** Even with concurrent sends, sequence number determines order
- Event 1 gets SequenceNum = N
- Event 2 gets SequenceNum = N+1 (or vice versa)
- Deterministic tie-breaking: lower sequence number first
- **Result:** Reproducible order (whoever acquired lock first)

**Note:** Order between two concurrent sends is still technically undefined (race to acquire lock), but:
1. It's deterministic GIVEN the lock acquisition order
2. For replay, you record sequence numbers and replay in that order
3. Most importantly, it's **stable** - same binary, same conditions = same order

### 6.3 Additional Determinism Sources

**Tick-based provides:**

1. **Tick boundaries** - All events processed before state queries
2. **Sequential parallel regions** - No race between regions
3. **Stable sorting** - Tie-breaking by sequence number
4. **Microstep determinism** - Already present in current core (reusable!)

**What doesn't change (still deterministic in both):**
- Transition selection (document order) - lines 1068-1100
- Microstep processing - lines 784-861
- LCA computation - lines 874-902

---

## Part 7: Concise Implementation Example

### 7.1 Complete Minimal Implementation

**File 1: realtime/runtime.go (~150 lines)**

```go
package realtime

import (
    "context"
    "sort"
    "sync"
    "time"
    
    "github.com/yourorg/statechartx"
)

// RealtimeRuntime wraps the event-driven runtime with tick-based execution
type RealtimeRuntime struct {
    // Embed existing runtime to reuse ALL core methods
    *statechartx.Runtime
    
    // Tick-specific fields
    config      Config
    ticker      *time.Ticker
    eventBatch  []EventWithMeta
    batchMu     sync.Mutex
    sequenceNum uint64
    tickNum     uint64
    
    // Parallel state tracking (for sequential processing)
    regionOrder []statechartx.StateID
}

type Config struct {
    TickRate time.Duration  // e.g., 16.67ms for 60 FPS
    MaxEventsPerTick int    // Queue capacity
}

type EventWithMeta struct {
    Event       statechartx.Event
    SequenceNum uint64
    Priority    int  // Optional, for future priority ordering
}

func NewRuntime(machine *statechartx.Machine, cfg Config) *RealtimeRuntime {
    return &RealtimeRuntime{
        Runtime:    statechartx.NewRuntime(machine, nil),
        config:     cfg,
        eventBatch: make([]EventWithMeta, 0, cfg.MaxEventsPerTick),
    }
}

func (rt *RealtimeRuntime) Start(ctx context.Context) error {
    // Enter initial state using existing method
    if err := rt.Runtime.Start(ctx); err != nil {
        return err
    }
    
    // Start tick loop
    rt.ticker = time.NewTicker(rt.config.TickRate)
    go rt.tickLoop(ctx)
    return nil
}

func (rt *RealtimeRuntime) Stop() error {
    if rt.ticker != nil {
        rt.ticker.Stop()
    }
    return rt.Runtime.Stop()
}

// SendEvent queues event for next tick (thread-safe)
func (rt *RealtimeRuntime) SendEvent(event statechartx.Event) error {
    rt.batchMu.Lock()
    defer rt.batchMu.Unlock()
    
    if len(rt.eventBatch) >= rt.config.MaxEventsPerTick {
        return errors.New("event queue full")
    }
    
    rt.eventBatch = append(rt.eventBatch, EventWithMeta{
        Event:       event,
        SequenceNum: rt.sequenceNum,
    })
    rt.sequenceNum++
    return nil
}

func (rt *RealtimeRuntime) GetTickNumber() uint64 {
    rt.batchMu.Lock()
    defer rt.batchMu.Unlock()
    return rt.tickNum
}

func (rt *RealtimeRuntime) tickLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case <-rt.ticker.C:
            rt.processTick()
        }
    }
}

func (rt *RealtimeRuntime) processTick() {
    // Phase 1: Collect events atomically
    events := rt.collectEvents()
    
    // Phase 2: Sort for deterministic order
    rt.sortEvents(events)
    
    // Phase 3: Process events using EXISTING core
    rt.processEvents(events)
    
    // Phase 4: Process microsteps using EXISTING core
    rt.processMicrostepsIfNeeded()
    
    // Phase 5: Process parallel regions if any
    if len(rt.regionOrder) > 0 {
        rt.processParallelRegionsSequentially()
    }
    
    // Increment tick counter
    rt.batchMu.Lock()
    rt.tickNum++
    rt.batchMu.Unlock()
}

func (rt *RealtimeRuntime) collectEvents() []EventWithMeta {
    rt.batchMu.Lock()
    defer rt.batchMu.Unlock()
    
    events := rt.eventBatch
    rt.eventBatch = make([]EventWithMeta, 0, rt.config.MaxEventsPerTick)
    return events
}

func (rt *RealtimeRuntime) sortEvents(events []EventWithMeta) {
    // Stable sort by sequence number (FIFO)
    sort.SliceStable(events, func(i, j int) bool {
        if events[i].Priority != events[j].Priority {
            return events[i].Priority > events[j].Priority  // Higher priority first
        }
        return events[i].SequenceNum < events[j].SequenceNum  // FIFO within priority
    })
}

func (rt *RealtimeRuntime) processEvents(events []EventWithMeta) {
    for _, eventMeta := range events {
        // REUSE existing Runtime.processEvent method
        rt.Runtime.ProcessEventDirectly(eventMeta.Event)
    }
}

func (rt *RealtimeRuntime) processMicrostepsIfNeeded() {
    // REUSE existing microstep processing
    rt.Runtime.ProcessMicrostepsDirectly(context.Background())
}

func (rt *RealtimeRuntime) processParallelRegionsSequentially() {
    // Sequential processing of parallel regions
    // REUSES existing transition methods per region
    for _, regionID := range rt.regionOrder {
        rt.processRegion(regionID)
    }
}

func (rt *RealtimeRuntime) processRegion(regionID statechartx.StateID) {
    // Get events for this region
    events := rt.getRegionEventsForTick(regionID)
    
    for _, event := range events {
        // Process using EXISTING transition core
        // This calls pickTransitionHierarchical, computeLCA,
        // exitToLCA, enterFromLCA - ALL REUSED
        rt.Runtime.ProcessRegionEventDirectly(regionID, event)
    }
}

func (rt *RealtimeRuntime) getRegionEventsForTick(regionID statechartx.StateID) []statechartx.Event {
    // Implementation depends on how events are routed to regions
    // Could be broadcast or targeted
    return nil  // Placeholder
}
```

**Note:** This assumes we add these methods to existing Runtime:
- `ProcessEventDirectly(event Event)` - Expose existing processEvent
- `ProcessMicrostepsDirectly(ctx)` - Expose existing processMicrosteps
- `ProcessRegionEventDirectly(regionID, event)` - Extract from parallelRegion.processEvent

**Alternative:** Make these methods public in statechartx (no refactor needed if already accessible)

### 7.2 Code Metrics

**New Code:**
- RealtimeRuntime struct and methods: ~150 lines
- Tick processing logic: ~50 lines
- Event batching and sorting: ~30 lines
- **Total new: ~230 lines**

**Reused Code:**
- All of Runtime.processEvent: ~70 lines
- All of Runtime.processMicrosteps: ~80 lines
- All of Runtime.computeLCA: ~30 lines
- All of Runtime.exitToLCA: ~20 lines
- All of Runtime.enterFromLCA: ~30 lines
- All of Runtime.pickTransitionHierarchical: ~50 lines
- All history state methods: ~100 lines
- **Total reused: ~380 lines**

**Duplication: 0 lines**

**Comparison to Plan:**
- Plan: 1000 lines total, ~600 duplicated
- Recommendation: 230 lines new, ~380 reused, 0 duplicated
- **Reduction: 77% less code, 100% less duplication**

---

## Conclusion

### Summary of Findings

1. **Current core is 86% synchronous** - Only 14% is goroutine coordination
2. **~500 lines are directly reusable** - Battle-tested state transition logic
3. **Proposed plan duplicates 60% of code** - Unnecessary re-implementation
4. **Embedding strategy reduces new code by 77%** - From 1000 to 230 lines
5. **Double buffering likely unnecessary** - Sequential processing is already atomic
6. **Event ordering solved with simple batching** - No complex buffering needed

### Recommended Path Forward

**Week 1: Minimal Viable Tick-Based Runtime**
- Embed existing Runtime
- Add tick loop and event batching
- Test with simple state machines
- **Deliverable:** 230 lines, deterministic 60 FPS execution

**Week 2: Validation**
- Run existing SCXML conformance tests
- Run existing stress tests
- Verify determinism with replay tests
- **Deliverable:** Confidence in correctness

**Week 3: Parallel States**
- Implement sequential parallel processing
- Reuse existing region transition methods
- Test parallel state machines
- **Deliverable:** Full feature parity with event-driven

**Week 4: Polish & Documentation**
- Error handling, edge cases
- Performance benchmarks
- API documentation
- Migration guide
- **Deliverable:** Production ready

**Future (Weeks 5+): Submodules**
- Replay (when needed)
- Debug tools (when needed)
- Distribution (if needed)

### Final Recommendation

**DO:**
- ✅ Embed or compose existing Runtime
- ✅ Reuse all core transition methods
- ✅ Start minimal (230 lines)
- ✅ Test with existing test suite
- ✅ Skip double buffering initially
- ✅ Add submodules only when needed

**DON'T:**
- ❌ Re-implement transition logic
- ❌ Re-implement history states
- ❌ Re-implement microsteps
- ❌ Add hooks before they're needed
- ❌ Implement replay in core

**Result:** Concise, readable, performant tick-based runtime in ~230 lines that reuses ~380 lines of battle-tested core logic, with zero duplication.

---

**Assessment Date:** January 2, 2026  
**Prepared By:** Architectural Review  
**Status:** Ready for Implementation
