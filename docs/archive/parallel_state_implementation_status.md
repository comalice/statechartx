# Parallel State Sequential Processing Implementation Status

## Date: 2026-01-03

## Summary

I've begun implementing sequential parallel state processing for the realtime runtime to enable deterministic SCXML conformance tests 404, 405, and 406. The implementation is partially complete but has some architectural challenges that need to be resolved.

## What's Been Implemented

### Core Infrastructure (Complete)

1. **Region State Tracking** (`realtime/runtime.go`):
   - Added `parallelRegionStates` map to track regions sequentially
   - Added `realtimeRegion` struct to store region state and event queues
   - Added `regionMu` mutex for thread safety

2. **Helper Methods** (`statechart.go`):
   - `GetMachine()` - exposes machine for realtime runtime
   - `GetState()` - gets state by ID
   - `FindDeepestInitial()` - public wrapper for finding initial states
   - `SetCurrentState()` - sets current state
   - `SetContext()` - initializes runtime context
   - `ErrEventQueueFull` - error constant

3. **Parallel State Methods** (`realtime/parallel.go`):
   - `Start()` - overrides embedded Runtime's Start
   - `enterInitialStateSequential()` - initializes without goroutines
   - `enterParallelState()` - enters parallel state with sequential regions
   - `exitParallelState()` - exits parallel state in reverse document order
   - `enterRegionHierarchy()` - executes entry actions in order
   - `exitRegionHierarchy()` - executes exit actions in reverse order
   - `processRegionEvent()` - processes events within region context
   - `SendEvent()` - routes events to parallel regions or batch queue

4. **Tick Processing** (`realtime/tick.go`):
   - `processParallelRegionsSequentially()` - processes regions in document order
   - `processParallelStateRegions()` - handles individual parallel state's regions
   - Added sort import and statechartx import

## Current Status: BLOCKED

### The Problem

The implementation is **architecturally blocked** by the deep integration between the realtime runtime and the embedded event-driven Runtime. Here's the issue:

1. **Initial Entry Works**: Our `enterInitialStateSequential()` successfully enters parallel states and initializes regions without spawning goroutines.

2. **Microsteps Trigger Event-Driven Code**: When we call `rt.Runtime.ProcessMicrosteps()`, it calls the embedded Runtime's internal methods (`enterFromLCA`, `exitToLCA`, etc.).

3. **Event-Driven Runtime Doesn't Know About Sequential Regions**: The embedded Runtime still tries to use its goroutine-based parallel state handling when it encounters parallel states during transitions.

4. **Event Routing Confusion**: Events sent via `rt.SendEvent()` go to our sequential region queues, but the embedded Runtime's transition processing doesn't know about these queues.

### Test Results

All three tests fail with state = 11 (FAIL):
- **TestSCXML404_Realtime**: Exit order test
- **TestSCXML405_Realtime**: Transition execution order test
- **TestSCXML406_Realtime**: Entry order test

The tests don't panic (good!) but end in FAIL state because:
- Events are being queued in our sequential regions
- But the embedded Runtime's microstep processing doesn't know to check those queues
- Eventless transitions from parallel states don't fire correctly

## Root Cause Analysis

The fundamental issue is **dual runtime modes**:

```
RealtimeRuntime {
    *statechartx.Runtime  // Event-driven, goroutine-based
    parallelRegionStates   // Sequential, tick-based
}
```

When we override `Start()` and `SendEvent()`, we initialize our sequential structures. But when `ProcessMicrosteps()` is called (which we MUST call to process eventless transitions), it invokes the embedded Runtime's methods which:
- Don't know about `parallelRegionStates`
- Try to spawn goroutines when entering parallel states
- Look for events in goroutine channels, not our sequential queues

## Possible Solutions

### Option 1: Complete Override (High Effort, Clean Architecture)

**Approach**: Don't embed Runtime at all. Reimplement all state machine logic in RealtimeRuntime with sequential parallel state support baked in.

**Pros**:
- Clean separation of concerns
- Full control over parallel state behavior
- No dual-mode confusion

**Cons**:
- Must reimplement ~800 lines of core statechart logic
- Duplicate maintenance burden
- Risk of behavior divergence between runtimes

**Effort**: 2-3 days of focused work

### Option 2: Hook Points (Medium Effort, Pragmatic)

**Approach**: Add hook/callback system to the embedded Runtime for parallel state operations:

```go
type ParallelStateHooks struct {
    EnterParallel func(ctx context.Context, state *State) error
    ExitParallel  func(ctx context.Context, state *State) error
    SendToRegion  func(regionID StateID, event Event) error
}
```

Realtime runtime registers hooks that use sequential processing. Event-driven runtime uses nil hooks (default goroutine behavior).

**Pros**:
- Minimal code duplication
- Both runtimes share core logic
- Clear extension point

**Cons**:
- Adds complexity to core Runtime
- Hook indirection may confuse readers
- Need to identify all hook points correctly

**Effort**: 1 day of work

### Option 3: Wrapper/Decorator Pattern (Low Effort, Hacky)

**Approach**: Wrap every embedded Runtime method to intercept parallel state operations:

```go
func (rt *RealtimeRuntime) ProcessMicrosteps(ctx context.Context) {
    // Check for parallel states in current configuration
    if rt.isInParallelState() {
        rt.processMicrostepsSequential(ctx)
    } else {
        rt.Runtime.ProcessMicrosteps(ctx)
    }
}
```

**Pros**:
- Minimal changes to core Runtime
- Quick to implement

**Cons**:
- Fragile (easy to miss a method)
- Lots of boilerplate wrapper methods
- Hard to maintain

**Effort**: 1-2 days, but technical debt

### Option 4: Flag-Based Mode Switch (Simplest, Some Limitations)

**Approach**: Add a `SequentialParallelMode bool` flag to Runtime. When true, use sequential processing for parallel states.

```go
type Runtime struct {
    // ... existing fields ...
    SequentialParallelMode bool
    // ... rest ...
}
```

Modify `enterParallelState()`, `exitParallelState()`, etc. to check this flag.

**Pros**:
- Single runtime implementation
- Minimal code changes
- Easy to test both modes

**Cons**:
- If/else branches throughout core code
- Slight performance overhead for all runtimes
- Less clean separation

**Effort**: 4-6 hours

## Recommendation

**I recommend Option 2: Hook Points**

This provides the best balance of:
- Code reuse (both runtimes share core logic)
- Clean architecture (hooks are explicit extension points)
- Maintainability (changes to core logic benefit both runtimes)
- Performance (no overhead when hooks are nil)

### Implementation Plan for Option 2

1. **Add ParallelStateHooks to Runtime** (30min):
   ```go
   type ParallelStateHooks struct {
       OnEnterParallel func(ctx context.Context, state *State) error
       OnExitParallel  func(ctx context.Context, state *State) error
       OnSendToRegion  func(regionID StateID, event Event) error
   }
   ```

2. **Modify Core Runtime Methods** (2 hours):
   - `enterParallelState()`: Check if hook exists, call it, else use goroutines
   - `exitParallelState()`: Check if hook exists, call it, else use goroutines
   - `sendEventToRegions()`: Check if hook exists, call it, else use channels

3. **Register Hooks in RealtimeRuntime** (1 hour):
   - In `NewRuntime()`, set `rt.Runtime.ParallelHooks = &ParallelStateHooks{...}`
   - Hook implementations use our sequential logic from `parallel.go`

4. **Test and Debug** (3 hours):
   - Run SCXML tests 404, 405, 406
   - Debug event routing issues
   - Verify deterministic ordering

**Total Effort**: ~6-7 hours of focused work

## Alternative: Defer to User Decision

Given the architectural complexity discovered during implementation, we could:

1. **Document Current Limitations**:
   - Update README to explain event-driven runtime has no ordering guarantees
   - Mark tests 404, 405, 406 as "requires sequential processing (not yet implemented)"
   - Create GitHub issue for future implementation

2. **Create Relaxed Tests for Event-Driven Runtime**:
   - Tests verify parallel states work (entry/exit/events)
   - Tests DON'T verify specific ordering
   - Document that ordering is non-deterministic

3. **Plan Sequential Processing for Future Release**:
   - Get user input on which option (1-4) to pursue
   - Schedule for next development cycle

## Current Code State

### What Works
- Realtime runtime compiles without errors
- Parallel state initialization (entry actions execute)
- Event queuing to sequential regions
- No goroutines spawned (verified - no panic about nil context)

### What Doesn't Work
- Event processing from sequential region queues
- Eventless transitions from parallel states
- Transitions between states in parallel regions

### Tests Status
- TestSCXML404_Realtime: **FAIL** (state 11 instead of PASS)
- TestSCXML405_Realtime: **FAIL** (state 3 instead of PASS)
- TestSCXML406_Realtime: **FAIL** (state 11 instead of PASS)

## Files Modified

1. **realtime/runtime.go**:
   - Added `parallelRegionStates`, `regionMu`
   - Added `realtimeRegion` struct
   - Removed duplicate `SendEvent()` and `Start()`

2. **realtime/parallel.go** (NEW):
   - Sequential parallel state entry/exit
   - Event routing to regions
   - Region event processing

3. **realtime/tick.go**:
   - Implemented `processParallelRegionsSequentially()`
   - Added imports for sort and statechartx

4. **statechart.go**:
   - Added helper methods for realtime runtime
   - Added `ErrEventQueueFull` constant

5. **realtime/scxml_parallel_test.go** (existing):
   - Tests 404, 405, 406 already written
   - Currently failing but not crashing

## Next Steps

**User Decision Needed**: Which approach should I pursue?

A. **Option 2 (Hook Points)** - ~6-7 hours, clean architecture
B. **Option 4 (Flag-Based)** - ~4-6 hours, simpler but less clean
C. **Defer Implementation** - Document limitations, create relaxed tests, plan for future
D. **Other** - User has different idea/priority

I recommend **Option 2** for production quality, or **Option 4** for quick prototype to validate the approach works.
