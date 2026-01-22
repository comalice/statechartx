# Parallel State Hooks Implementation - Status Update

## Date: 2026-01-03

## Summary

Successfully implemented the hook-based architecture for parallel state processing. The core infrastructure is complete and tests 406 passed before adding microstep processing. Currently debugging an infinite loop introduced by eventless transition handling within parallel regions.

## Completed Work

### 1. Core Hooks Infrastructure ✅

**File: `statechart.go`**
- Added `ParallelStateHooks` struct with three hooks:
  - `OnEnterParallel`: Custom parallel state entry
  - `OnExitParallel`: Custom parallel state exit
  - `OnSendToRegions`: Custom event routing to regions
- Added `ParallelHooks` field to Runtime
- Modified `enterParallelState()` to check hooks before default goroutine implementation
- Modified `exitParallelState()` to check hooks before default goroutine implementation
- Modified `sendEventToRegions()` to check hooks before default channel-based routing

### 2. Realtime Runtime Hooks ✅

**File: `realtime/parallel.go`**
- Created `createParallelHooks()` method that returns hook implementations
- Hooks delegate to realtime's sequential methods:
  - `enterParallelState()` - sequential region initialization in document order
  - `exitParallelState()` - sequential region exit in reverse document order
  - `sendEventToRegionsSequential()` - routes to region event queues

**File: `realtime/runtime.go`**
- Modified `NewRuntime()` to register hooks on embedded Runtime
- Hooks are registered at construction time

### 3. Sequential Parallel State Processing ✅

**File: `realtime/parallel.go`**
- `enterParallelState()`: Enters parallel state sequentially
  - Executes parent entry action
  - Sorts regions by StateID for document order
  - Initializes each region without spawning goroutines
  - Calls `enterRegionHierarchy()` for each region

- `exitParallelState()`: Exits parallel state in reverse order
  - Sorts regions by StateID
  - Exits in **reverse** order (last to first)
  - Calls `exitRegionHierarchy()` for each region
  - Executes parent exit action last

- `enterRegionHierarchy()`: Executes entry actions top-down
  - Builds path from region root to current state
  - Executes entry actions in parent→child order
  - Executes initial actions between states

- `exitRegionHierarchy()`: Executes exit actions bottom-up
  - Builds path from current state to region root
  - Executes exit actions in child→parent order

- `processRegionEvent()`: Processes events within region context
  - Finds matching transition
  - Executes exit/action/entry sequence
  - Calls `processRegionMicrosteps()` after transitions

### 4. Test Results

**Before microstep processing:**
- ✅ TestSCXML406_Realtime: **PASSED** (entry order test)

**After adding microstep processing:**
- ❌ All tests hang with infinite loop

## Current Blocker: Infinite Loop in Microstep Processing

### Problem

Added `processRegionMicrosteps()` to handle eventless (`NO_EVENT`) transitions within parallel regions. This caused infinite loops in all tests.

### Root Cause Analysis

The infinite loop is caused by recursive microstep processing:

1. `enterParallelState()` calls `enterRegionHierarchy()`
2. `enterRegionHierarchy()` calls `processRegionMicrosteps()`
3. `processRegionMicrosteps()` finds NO_EVENT transition
4. Executes transition, calls `enterRegionHierarchy()` for new state
5. GOTO 2 (infinite recursion if new state also has NO_EVENT transition)

### Attempted Fixes

1. **Loop detection**: Added check for `previousState == currentState`
   - Doesn't help if transitioning between different states in a cycle

2. **Internal transition short-circuit**: Return early for internal transitions
   - Helps, but doesn't prevent external transition loops

3. **Remove recursive call**: Don't call `processRegionMicrosteps()` from within itself
   - Still hangs because the outer loop keeps finding transitions

### The Real Problem

Looking at test 404, there's an eventless transition on the parallel state itself:
```go
s01p.Transitions = []*Transition{
    {
        Event:  sc.NO_EVENT,
        Target: STATE_S02,
        ...
    },
}
```

This transition should fire **after** the parallel regions are entered. But our current architecture doesn't handle eventless transitions **on** the parallel state - only **within** the regions.

## Next Steps to Fix

### Option A: Don't Process Microsteps in Regions (Simplest)

Remove `processRegionMicrosteps()` calls entirely. Eventless transitions would only be processed at the top level by the embedded Runtime's `ProcessMicrosteps()`.

**Pros:**
- Prevents infinite loops
- Simpler architecture
- Top-level eventless transitions still work

**Cons:**
- Eventless transitions **within** parallel regions won't work
- May fail some SCXML tests that rely on this

### Option B: Process Microsteps Once Per Tick (Recommended)

Move microstep processing to tick boundaries instead of inline:

1. Remove calls to `processRegionMicrosteps()` from `enterRegionHierarchy()` and `processRegionEvent()`
2. Add `processRegionMicrosteps()` call to `processParallelRegionsSequentially()` **after** all event processing
3. Process microsteps once per region per tick

**Pros:**
- Prevents infinite loops (limited to one iteration per tick)
- Eventless transitions still work
- Matches tick-based execution model

**Cons:**
- Eventless transitions have one tick latency
- May need multiple ticks for chains of eventless transitions

### Option C: Hierarchical Microstep Processing (Complex)

Process microsteps at two levels:
1. **Region-level**: Process eventless transitions within each region
2. **Parallel-level**: Process eventless transitions on the parallel state itself

Requires careful ordering and loop detection.

**Pros:**
- Full SCXML conformance
- Handles all eventless transition scenarios

**Cons:**
- Complex to implement correctly
- Easy to introduce bugs
- Performance overhead

## Recommendation

**Implement Option B**: Process microsteps once per tick at region level.

### Implementation

1. Remove these calls:
   - In `enterParallelState()`: Remove `rt.processRegionMicrosteps(ctx, region)` after `enterRegionHierarchy()`
   - In `processRegionEvent()`: Remove `rt.processRegionMicrosteps(ctx, region)` after transitions

2. Add to `processParallelRegionsSequentially()`:
   ```go
   // Process each region's event queue sequentially
   for _, regionID := range regionIDs {
       region := regions[regionID]
       // ... process events ...
   }

   // Process eventless transitions for each region (once per tick)
   for _, regionID := range regionIDs {
       region := regions[regionID]
       rt.processRegionMicrosteps(ctx, region)
   }
   ```

3. Simplify `processRegionMicrosteps()`: Remove the loop detection since it only runs once per tick

This matches the realtime runtime's philosophy: deterministic, tick-based execution with bounded processing per tick.

## Files Modified

1. **statechart.go**:
   - Added `ParallelStateHooks` struct
   - Added `ParallelHooks` field to Runtime
   - Modified `enterParallelState()`, `exitParallelState()`, `sendEventToRegions()`

2. **realtime/runtime.go**:
   - Modified `NewRuntime()` to register hooks

3. **realtime/parallel.go**:
   - Added `createParallelHooks()`
   - Implemented sequential parallel state entry/exit
   - Added region hierarchy traversal
   - Added event processing for regions
   - **IN PROGRESS**: Debugging microstep processing

## Test Status

- TestSCXML404_Realtime: ❌ Hangs (infinite loop)
- TestSCXML405_Realtime: ❌ Hangs (infinite loop)
- TestSCXML406_Realtime: ✅ Passed (before microsteps), ❌ Hangs (after microsteps)

## Code Quality

- ✅ Compiles without errors
- ✅ Clean hook-based architecture
- ✅ No code duplication between runtimes
- ✅ Good separation of concerns
- ❌ Infinite loop bug in microstep processing

## Estimated Time to Complete

**Option B (Recommended)**: 30-60 minutes
- Remove problematic microstep calls: 10 min
- Add microstep processing to tick loop: 15 min
- Test and debug: 15-30 min
