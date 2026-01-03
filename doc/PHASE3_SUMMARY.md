# Phase 3 Implementation Summary: Advanced Transitions

## Overview
Phase 3 focused on implementing eventless/immediate transitions (Step 8), as Steps 7 (Internal Transitions) and 9 (Transition Actions and Guards) were already completed in Phase 1.

## What Was Implemented

### Step 8: Eventless/Immediate Transitions
Eventless transitions are transitions with `Event: NO_EVENT` (0) that fire automatically when a state is entered, without requiring an external event.

#### Key Features:
1. **Microstep Processing**: After entering any state, the runtime checks for eventless transitions and processes them in a loop until no more are enabled
2. **Loop Protection**: `MAX_MICROSTEPS` constant (100) prevents infinite loops
3. **Priority**: Eventless transitions are processed BEFORE queued events
4. **Wildcard Exclusion**: `ANY_EVENT` wildcard does NOT match `NO_EVENT`

#### Implementation Details:

**New Function: `processMicrosteps(ctx)`**
- Creates a `NO_EVENT` event
- Loops up to `MAX_MICROSTEPS` times
- Uses `pickTransitionHierarchical()` to find eventless transitions
- Executes transitions (both internal and external)
- Continues until no more eventless transitions are found

**Integration Points:**
1. `enterInitialState()` - calls `processMicrosteps()` after entering initial state hierarchy
2. `processEvent()` - calls `processMicrosteps()` after each external transition (state change)
3. Internal transitions do NOT trigger microstep processing (no state change)

**Bug Fix:**
- Modified `pickTransition()` to exclude `NO_EVENT` from `ANY_EVENT` wildcard matching
- Added condition: `event.ID != NO_EVENT` when checking wildcard transitions

## Tests Implemented

### TestSCXML355
Tests that the default initial state (first in document order) is entered and that an eventless transition fires immediately.

**Scenario:**
- Root has two children: s0 and s1
- s0 is the initial state (first in document order)
- s0 has eventless transition to PASS
- s1 has eventless transition to FAIL

**Expected:** Machine enters s0, then immediately transitions to PASS via eventless transition

### TestSCXML377
Tests chaining of eventless transitions (simplified from original SCXML test).

**Scenario:**
- Chain of eventless transitions: s0 → s1 → PASS
- All transitions are eventless (NO_EVENT)

**Expected:** Machine follows the entire chain in one microstep sequence

**Note:** Original SCXML377 tests onexit handler order with internal event queues, which requires Phase 4 (separate internal/external event queues).

### TestSCXML419
Tests that eventless transitions take precedence over event-driven transitions.

**Scenario:**
- s1 onentry sends two events to the queue
- s1 has two transitions:
  1. `ANY_EVENT` → FAIL (event-driven)
  2. `NO_EVENT` → PASS (eventless)

**Expected:** Eventless transition fires first, transitioning to PASS before queued events are processed

## Test Results

All Phase 1-3 tests passing (14/14):

**Phase 1: Runtime & Event Queue (5 tests)**
- TestSCXML144 ✓
- TestSCXML147 ✓
- TestSCXML148 ✓
- TestSCXML149 ✓
- TestSCXML158 ✓

**Phase 2: Hierarchical States (3 tests)**
- TestSCXML375 ✓
- TestSCXML396 ✓
- TestSCXML403a ✓

**Phase 3: Advanced Transitions (6 tests)**
- TestSCXML355 ✓ (eventless transitions)
- TestSCXML377 ✓ (eventless chaining)
- TestSCXML419 ✓ (eventless precedence)
- TestInternalTransitionDoesTransition ✓ (Step 7 - already implemented)
- TestInternalTransitionExecsActionOnly ✓ (Step 7 - already implemented)
- TestInternalPicksFirstEnabledTransition ✓ (Step 9 - already implemented)

## Code Changes

### statechart.go
1. Added `MAX_MICROSTEPS` constant (100)
2. Implemented `processMicrosteps(ctx)` function
3. Modified `enterInitialState()` to call `processMicrosteps()`
4. Modified `processEvent()` to call `processMicrosteps()` after external transitions
5. Fixed `pickTransition()` to exclude NO_EVENT from ANY_EVENT wildcard

### Test Files
1. `statechart_scxml_300-399_test.go`: Implemented TestSCXML355 and TestSCXML377
2. `statechart_scxml_400-499_test.go`: Implemented TestSCXML419

### go.mod
- Fixed Go version from 1.25.4 to 1.19 (compatibility fix)

## SCXML Compliance

The implementation follows SCXML semantics for eventless transitions:
- ✓ Eventless transitions checked on state entry
- ✓ Eventless transitions take precedence over event-driven transitions
- ✓ Microstep processing continues until stable
- ✓ Loop protection to prevent infinite loops
- ✓ Document order determines priority when multiple transitions match

## Next Steps (Phase 4)

Phase 4 will implement:
- **Step 10**: Initial transition actions
- **Step 11**: Separate internal/external event queues
- **Step 12**: Event queue priority (internal events processed before external)

This will enable full SCXML377 compliance (onexit handlers with internal events).

## Architecture Notes

**Microstep vs Macrostep:**
- **Macrostep**: Processing one event from the queue (external event)
- **Microstep**: Processing eventless transitions until stable (no state change visible to external observers)

**Event Processing Order:**
1. Enter state (execute entry actions)
2. Process all eventless transitions (microsteps)
3. Stable state reached
4. Wait for next event from queue (macrostep)
5. Repeat

This ensures that eventless transitions always complete before the next external event is processed, matching SCXML semantics.
