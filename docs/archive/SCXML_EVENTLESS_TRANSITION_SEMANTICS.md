# SCXML Eventless Transition Semantics - Research Findings

## Date: 2026-01-03

## Executive Summary

After researching the W3C SCXML specification and statecharts.dev, I now understand the correct semantics for eventless transitions. **Our current approach is correct** - we DO want to process eventless transitions to completion. The infinite loop bug is a real problem that needs fixing, not an expected behavior.

## Key Findings from W3C SCXML Spec

### Microsteps vs Macrosteps

**Microstep**: Execution of one transition set
- Exit states in exit order
- Execute transition actions in document order
- Enter states in entry order

**Macrostep**: Series of microsteps continuing until:
- Internal event queue is empty, AND
- No transitions enabled by NULL (eventless transitions)

### Processing Algorithm

1. Enter initial configuration
2. **Check for eventless transitions** (enabled by NULL)
3. If found, execute as microstep, GOTO 2
4. If none, process internal events
5. If internal event causes transition, GOTO 2
6. When internal queue empty and no NULL transitions, macrostep complete
7. Process external events, GOTO 2

**Critical**: After EACH microstep, the processor checks for eventless transitions again and continues processing them until none are enabled.

### Parallel State Semantics

Despite being called "parallel," regions are processed **sequentially** in **document order**:

> "the parallel children process the event in a defined, serial order, so no conflicts or race conditions can occur"

**Document order matters everywhere** in SCXML:
- Exit order: children before parents, **reverse document order** for siblings
- Entry order: parents before children, **document order** for siblings
- Transition selection: earlier states have priority

### Loop Protection

**SCXML does NOT specify hard limits on eventless transition loops.**

The spec says:
- Executable content "MUST execute swiftly"
- Authors must ensure conditions eventually become false
- No timeout or iteration count specified

However, practical implementations typically limit microsteps to prevent infinite loops (our MAX_MICROSTEPS = 100 is reasonable).

## Key Findings from statecharts.dev

### Automatic Transitions (Eventless)

Automatic transitions are:
- "checked immediately after the state is entered"
- Checked "every time the statechart handles an event"
- Checked "after other automatic transitions have fired"

Guards are rechecked after each transition, allowing chains of eventless transitions.

### Stability / Rest

Systems can have "condition states" that never rest - they immediately transition out upon entry via guarded eventless transitions.

## Analysis of Our Tests

### Test 404 Structure

```xml
<parallel id="s01p">
  <onexit>
    <raise event="event3"/>  <!-- 3rd event -->
  </onexit>

  <transition target="s02">  <!-- EVENTLESS TRANSITION -->
    <raise event="event4"/>   <!-- 4th event -->
  </transition>

  <state id="s01p1">
    <onexit><raise event="event2"/></onexit>  <!-- 2nd event -->
  </state>

  <state id="s01p2">
    <onexit><raise event="event1"/></onexit>  <!-- 1st event -->
  </state>
</parallel>
```

**Expected Flow**:
1. Enter parallel state s01p
2. Enter regions s01p1 and s01p2 (in document order)
3. **Check for eventless transitions** - find transition on s01p with no event
4. Execute that transition:
   - Exit s01p2 (raises event1)
   - Exit s01p1 (raises event2)
   - Exit s01p (raises event3)
   - Execute transition action (raises event4)
   - Enter s02
5. Process raised events from internal queue
6. Event1 transitions s02→s03
7. Event2 transitions s03→s04
8. Event3 transitions s04→s05
9. Event4 transitions s05→pass

## Why Our Implementation Has Infinite Loops

### The Problem

We're calling `processRegionMicrosteps()` from within `enterRegionHierarchy()`, creating recursion:

```
enterParallelState()
  → enterRegionHierarchy(region1)
    → processRegionMicrosteps()  ← PROBLEM: called during entry
      → finds NO_EVENT transition
      → executes transition
      → enterRegionHierarchy(new state)
        → processRegionMicrosteps()  ← RECURSION
          → ...infinite
```

### The Real Issue

**Eventless transitions should be checked AFTER all entry actions complete, not DURING entry.**

Per SCXML spec:
> "the SCXML processor will not process any events or take any transitions until all <onentry> handlers in S have finished"

This means:
1. Complete ALL entry actions for ALL states first
2. THEN check for eventless transitions
3. If found, execute them (which may trigger more entries)
4. Repeat step 2-3 until no eventless transitions found

### What Test 404 Actually Tests

Test 404 has an eventless transition **on the parallel state itself**, not within the regions. This transition fires after the parallel state is entered, causing all regions to exit in proper order.

## Correct Implementation Strategy

### Phase 1: Complete All Entries

```go
func (rt *RealtimeRuntime) enterParallelState(ctx, state) error {
    // 1. Execute parent entry action
    if state.EntryAction != nil {
        state.EntryAction(...)
    }

    // 2. Enter ALL regions (entry actions only, NO transition checking)
    for _, regionID := range sortedRegionIDs {
        region := createRegion(regionID)
        enterRegionHierarchyNoTransitions(ctx, region, child)
        // DON'T check for transitions yet
    }

    // All entries complete - NOW check for eventless transitions
    return nil
}
```

### Phase 2: Check for Eventless Transitions

After enterParallelState returns, the caller should:

```go
// In enterInitialStateSequential():
rt.enterParallelState(ctx, state)

// Now that all entry actions complete, check for eventless transitions
rt.processMicrostepsToCompletion(ctx)
```

### Phase 3: Process Microsteps to Completion

```go
func (rt *RealtimeRuntime) processMicrostepsToCompletion(ctx) {
    for i := 0; i < MAX_MICROSTEPS; i++ {
        // Check ALL possible sources of eventless transitions:

        // 1. Check parallel regions
        hasTransition := rt.checkParallelRegionsForEventlessTransitions(ctx)

        // 2. Check current non-parallel state
        if !hasTransition {
            hasTransition = rt.checkCurrentStateForEventlessTransitions(ctx)
        }

        // 3. Process internal event queue
        if !hasTransition {
            hasTransition = rt.processInternalEvents(ctx)
        }

        if !hasTransition {
            break // Macrostep complete - stable configuration
        }

        // A transition fired - loop to check again
    }
}
```

## The Correct Semantics

### YES to Processing Until Stable

We SHOULD process eventless transitions repeatedly until:
- No eventless transitions are enabled, AND
- Internal event queue is empty

This is not "infinite recursion" - it's **iterative processing to completion**.

### NO to Recursive Calls

We should NOT:
- Call microstep processing from within entry/exit actions
- Allow unbounded recursion
- Process transitions during state entry

### YES to Hard Limits

We SHOULD:
- Limit microstep iterations (MAX_MICROSTEPS = 100)
- Detect when we're not making progress
- Provide clear error messages when limit exceeded

## Implementation Plan

### Step 1: Remove Microstep Calls from Entry/Exit

Remove `processRegionMicrosteps()` calls from:
- `enterRegionHierarchy()`
- `processRegionEvent()` (except at the very end)
- Any other entry/exit methods

### Step 2: Add Top-Level Microstep Processing

After `enterParallelState()` completes in `enterInitialStateSequential()`:
```go
rt.enterParallelState(ctx, state)
rt.processMacrostepToCompletion(ctx)  // NEW
```

After processing an event in `processRegionEvent()`:
```go
rt.executeTransition(...)
rt.processMacrostepToCompletion(ctx)  // NEW
```

### Step 3: Implement processMacrostepToCompletion

```go
func (rt *RealtimeRuntime) processMacrostepToCompletion(ctx context.Context) {
    for i := 0; i < MAX_MICROSTEPS; i++ {
        madeProgress := false

        // Check each parallel region for eventless transitions
        for _, parallelStateID := range rt.getActiveParallelStates() {
            regions := rt.parallelRegionStates[parallelStateID]
            for _, regionID := range sortedRegionIDs(regions) {
                region := regions[regionID]
                if rt.processRegionEventlessTransition(ctx, region) {
                    madeProgress = true
                    break // Start over from first region
                }
            }
            if madeProgress {
                break
            }
        }

        // Process internal events raised by transitions
        if !madeProgress {
            madeProgress = rt.processInternalEventQueue(ctx)
        }

        if !madeProgress {
            break // Stable configuration reached
        }
    }
}
```

## Why This Fixes Our Bug

**Before**: Recursive calls during entry → stack overflow
**After**: Iterative processing after entry completes → bounded iteration

**Before**: Transitions checked during entry actions
**After**: Transitions checked after all entry actions complete

**Before**: No clear stable configuration
**After**: Clear stable state when loop exits

## Verification

Tests should pass because:
1. Test 404: Eventless transition on parallel state fires after regions enter
2. Test 405: Eventless transitions in regions processed after entry
3. Test 406: Entry order correct, eventless transitions processed after

## Sources

- [W3C SCXML Specification - Interpretation Algorithm](https://www.w3.org/TR/scxml/#AlgorithmforSCXMLInterpretation)
- [W3C SCXML Specification - Parallel States](https://www.w3.org/TR/scxml/#parallel)
- [statecharts.dev - Automatic Transitions](https://statecharts.dev/glossary/automatic-transition.html)
- [statecharts.dev - Local Transitions](https://statecharts.dev/glossary/local-transition.html)

## Conclusion

**We do NOT want infinite recursion.** We want **iterative microstep-to-completion processing** with proper limits. The SCXML spec is clear: process eventless transitions and internal events until reaching a stable configuration, but do it AFTER entry actions complete, not during them.

Our MAX_MICROSTEPS limit is appropriate and necessary for practical implementations.
