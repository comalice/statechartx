# StatechartX Limitations & Implementation Plan

**Document Version**: 1.0  
**Date**: January 2, 2026  
**Status**: Planning Phase  
**Current State**: 32/32 tests passing, Phase 5 complete

---

## Executive Summary

This document addresses all known limitations from PHASE5_SUMMARY.md (excluding datamodel, which is left to users). Each limitation is analyzed for complexity, test requirements, and implementation approach, then prioritized for development.

**Limitations to Address**: 3 core features
- Nested Parallel States (testing/validation)
- Done Events (done.state.id)
- History States (shallow/deep)

---

## Limitation 1: Nested Parallel States

### Current Status
- **Listed as**: "Not extensively tested (basic support exists)"
- **Priority**: **HIGH**
- **Complexity**: Medium (testing focus, not new implementation)

### Problem Analysis
The current implementation supports nested parallel states architecturally (parallel states can have parallel children), but lacks comprehensive test coverage. This creates uncertainty about edge cases:
- Parallel state containing parallel children
- Event routing through nested parallel hierarchies
- Cleanup order when exiting nested parallel states
- Context cancellation propagation through multiple levels
- Goroutine lifecycle management with deep nesting

### Implementation Approach

#### Phase 1: Test Design (No Code Changes)
1. **Create test scenarios** for nested parallel states:
   - 2-level nesting: Parallel state with parallel children
   - 3-level nesting: Parallel → Parallel → Parallel
   - Mixed nesting: Parallel → Sequential → Parallel
   - Event routing through nested hierarchies
   - Targeted events to deeply nested regions
   - Broadcast events across all nesting levels

2. **Edge case scenarios**:
   - Exit from deeply nested parallel state
   - Context cancellation at various nesting levels
   - Panic recovery in nested regions
   - Goroutine leak detection with deep nesting
   - Race condition testing with `-race` flag

#### Phase 2: Implementation (If Issues Found)
- Fix any bugs discovered during testing
- Add timeout protection for nested operations
- Enhance cleanup logic if needed
- Document nesting depth recommendations

### Test Plan

#### Test Suite: `statechart_nested_parallel_test.go`

**Test 1: TestTwoLevelNestedParallel**
- Setup: Parent parallel state with 2 regions, each region is parallel with 2 sub-regions
- Actions:
  - Enter parent parallel state
  - Verify 4 total goroutines spawned (2 parent + 2 per child)
  - Send broadcast event, verify all 4 regions receive it
  - Exit parent state
  - Verify all goroutines cleaned up
- Success Criteria: No goroutine leaks, all events delivered, clean shutdown

**Test 2: TestThreeLevelNestedParallel**
- Setup: 3 levels of parallel nesting
- Actions:
  - Enter top-level parallel state
  - Verify goroutine count matches expected (2^3 = 8 regions)
  - Send targeted event to deepest region
  - Verify only target receives event
  - Cancel context
  - Verify all goroutines exit within timeout
- Success Criteria: Correct goroutine count, targeted delivery works, clean cancellation

**Test 3: TestMixedNestedParallel**
- Setup: Parallel → Sequential → Parallel hierarchy
- Actions:
  - Enter top parallel state (2 regions)
  - Each region has sequential states
  - Sequential states have parallel children
  - Transition through sequential states
  - Verify parallel children spawn/cleanup correctly
- Success Criteria: Correct state transitions, proper goroutine lifecycle

**Test 4: TestNestedParallelEventRouting**
- Setup: 2-level nested parallel (4 leaf regions)
- Actions:
  - Send broadcast (Address=0) from parent
  - Verify all 4 leaf regions receive event
  - Send targeted event to specific leaf region
  - Verify only that region receives event
  - Send event to intermediate parallel state
  - Verify only its children receive event
- Success Criteria: Correct event routing at all levels

**Test 5: TestNestedParallelPanicRecovery**
- Setup: 2-level nested parallel
- Actions:
  - Trigger panic in deeply nested region
  - Verify panic is recovered
  - Verify other regions continue running
  - Verify parent runtime remains stable
- Success Criteria: Isolated panic recovery, no system crash

**Test 6: TestNestedParallelRaceConditions**
- Setup: 3-level nested parallel with shared state
- Actions:
  - All regions access shared counter concurrently
  - Increment counter 1000 times per region
  - Run with `-race` flag
- Success Criteria: No race conditions detected, correct final count

**Test 7: TestNestedParallelCleanupOrder**
- Setup: 2-level nested parallel with exit actions
- Actions:
  - Track cleanup order in slice
  - Exit parent parallel state
  - Verify children clean up before parent
  - Verify all goroutines exit
- Success Criteria: Correct cleanup order, no goroutine leaks

**Test 8: TestDeepNestingLimits**
- Setup: 5-level nested parallel (stress test)
- Actions:
  - Enter deeply nested structure
  - Verify goroutine count (2^5 = 32 regions)
  - Send events at various levels
  - Exit and verify cleanup
- Success Criteria: System handles deep nesting, no performance degradation

### Implementation Plan

#### Step 1: Create Test File (1-2 hours)
```bash
cd /home/ubuntu/github_repos/statechartx
touch statechart_nested_parallel_test.go
```
- Implement all 8 tests above
- Use existing test patterns from `statechart_parallel_test.go`
- Add helper functions for nested state creation

#### Step 2: Run Tests & Identify Issues (30 minutes)
```bash
go test ./... -race -timeout 30s -run TestNested -v
```
- Document any failures
- Use `-race` flag to detect race conditions
- Check for goroutine leaks

#### Step 3: Fix Issues (2-4 hours, if needed)
Potential fixes based on common issues:
- **Event routing**: Ensure Address field works correctly through nesting
- **Cleanup order**: Verify children exit before parents
- **Context propagation**: Ensure cancellation reaches all levels
- **Timeout handling**: Adjust timeouts for nested operations

#### Step 4: Documentation (30 minutes)
- Add nested parallel examples to README
- Document recommended nesting depth limits
- Add performance characteristics for nested states

### Success Metrics
- ✅ All 8 nested parallel tests pass
- ✅ No race conditions with `-race` flag
- ✅ No goroutine leaks detected
- ✅ Performance acceptable up to 3-4 nesting levels
- ✅ Documentation updated with examples

### Estimated Effort
- **Testing**: 2-3 hours
- **Bug fixes**: 2-4 hours (if issues found)
- **Documentation**: 30 minutes
- **Total**: 4.5-7.5 hours

---

## Limitation 2: Done Events (done.state.id)

### Current Status
- **Listed as**: "done.state.id events not yet implemented"
- **Priority**: **CRITICAL**
- **Complexity**: Medium-High

### Problem Analysis
SCXML specification requires automatic "done.state.id" events when:
1. A **final state** is entered in a sequential region
2. **All regions** in a parallel state reach their final states

These events enable:
- Automatic transitions when sub-machines complete
- Coordination between parallel regions
- Hierarchical state machine composition

Without done events, users must manually signal completion, breaking SCXML compatibility.

### SCXML Specification Reference

**Done Event Format**:
```
done.state.<stateID>
```

**Triggering Conditions**:
1. **Sequential State**: When entering a final state child
2. **Parallel State**: When ALL child regions reach final states

**Event Data**:
- Event ID: "done.state." + parent state ID
- Event Data: Optional data from final state

### Implementation Approach

#### Architecture Changes

**1. Add Done Event Generation**
```go
// In statechart.go

// Check if state is final
func (s *State) IsFinal() bool {
    return s.IsFinalState  // Existing field from Phase 4
}

// Generate done event for parent
func (rt *Runtime) generateDoneEvent(finalStateID StateID) {
    // Find parent of final state
    parent := rt.findParent(finalStateID)
    if parent == nil {
        return
    }
    
    // Check if parent should emit done event
    if rt.shouldEmitDoneEvent(parent) {
        doneEvent := Event{
            ID:      EventID("done.state." + string(parent.ID)),
            Data:    nil,  // TODO: Support final state data
            Address: 0,    // Broadcast
        }
        rt.SendEvent(rt.ctx, doneEvent)
    }
}

// Check if parent should emit done event
func (rt *Runtime) shouldEmitDoneEvent(parent *State) bool {
    if parent.IsParallel {
        // All regions must be in final state
        return rt.allRegionsInFinalState(parent)
    } else {
        // Sequential state: emit immediately when child is final
        return true
    }
}

// Check if all parallel regions are in final state
func (rt *Runtime) allRegionsInFinalState(parallelState *State) bool {
    rt.mu.RLock()
    defer rt.mu.RUnlock()
    
    for regionID := range parallelState.Children {
        region := rt.parallelRegions[regionID]
        if region == nil {
            return false
        }
        
        region.mu.RLock()
        currentState := region.currentState
        region.mu.RUnlock()
        
        state := rt.machine.GetState(currentState)
        if state == nil || !state.IsFinal() {
            return false
        }
    }
    return true
}
```

**2. Integrate into State Transitions**
```go
// In enterState function
func (rt *Runtime) enterState(ctx context.Context, stateID StateID) error {
    state := rt.machine.GetState(stateID)
    if state == nil {
        return fmt.Errorf("state not found: %v", stateID)
    }
    
    // Execute entry actions
    if state.OnEntry != nil {
        state.OnEntry(ctx, Event{})
    }
    
    // Check if this is a final state
    if state.IsFinal() {
        rt.generateDoneEvent(stateID)
    }
    
    // ... rest of entry logic
    return nil
}
```

**3. Add Done Event Tracking**
```go
// Add to Runtime struct
type Runtime struct {
    // ... existing fields
    doneEventsPending map[StateID]bool  // Track pending done events
    doneEventsMu      sync.RWMutex
}

// Track done event generation to prevent duplicates
func (rt *Runtime) markDoneEventSent(stateID StateID) {
    rt.doneEventsMu.Lock()
    defer rt.doneEventsMu.Unlock()
    rt.doneEventsPending[stateID] = true
}

func (rt *Runtime) clearDoneEventSent(stateID StateID) {
    rt.doneEventsMu.Lock()
    defer rt.doneEventsMu.Unlock()
    delete(rt.doneEventsPending, stateID)
}
```

### Test Plan

#### Test Suite: `statechart_done_events_test.go`

**Test 1: TestDoneEventSequentialState**
- Setup: Parent state with final child state
- Actions:
  - Enter parent state
  - Transition to final child state
  - Listen for "done.state.parent" event
  - Verify event is generated
- Success Criteria: Done event received with correct ID

**Test 2: TestDoneEventParallelStateAllRegions**
- Setup: Parallel state with 3 regions, each with final state
- Actions:
  - Enter parallel state
  - Transition region 1 to final (no done event yet)
  - Transition region 2 to final (no done event yet)
  - Transition region 3 to final (done event generated)
  - Verify "done.state.parallel" event received
- Success Criteria: Done event only after ALL regions final

**Test 3: TestDoneEventParallelStatePartialCompletion**
- Setup: Parallel state with 2 regions
- Actions:
  - Enter parallel state
  - Transition region 1 to final
  - Wait 500ms
  - Verify NO done event generated
  - Transition region 2 to final
  - Verify done event NOW generated
- Success Criteria: Done event only when all regions complete

**Test 4: TestDoneEventTriggersTransition**
- Setup: State machine with automatic transition on done event
- States:
  - State A (parent) → State B (final child)
  - State C (target of done transition)
- Transitions:
  - A → B (on EVENT_START)
  - A → C (on done.state.A)
- Actions:
  - Start in A
  - Send EVENT_START (enters B, which is final)
  - Verify done.state.A generated
  - Verify automatic transition to C
- Success Criteria: Automatic transition triggered by done event

**Test 5: TestDoneEventWithData**
- Setup: Final state with data payload
- Actions:
  - Enter final state with data: {"result": 42}
  - Verify done event contains data
  - Access data in transition action
- Success Criteria: Done event carries final state data

**Test 6: TestDoneEventNoDuplicates**
- Setup: State that could trigger multiple done events
- Actions:
  - Enter final state
  - Verify done event generated once
  - Re-enter same final state
  - Verify done event generated again (new entry)
  - Stay in final state
  - Verify no duplicate events
- Success Criteria: One done event per final state entry

**Test 7: TestDoneEventNestedParallel**
- Setup: Nested parallel states with final states
- Structure:
  - Parallel Parent (2 regions)
    - Region 1: Parallel Child (2 sub-regions)
    - Region 2: Sequential state
- Actions:
  - Complete both sub-regions in Region 1
  - Verify done.state.region1 generated
  - Complete Region 2
  - Verify done.state.parent generated
- Success Criteria: Done events at correct nesting levels

**Test 8: TestDoneEventConcurrentCompletion**
- Setup: Parallel state with 10 regions
- Actions:
  - Enter parallel state
  - Send events to all regions concurrently
  - All regions transition to final simultaneously
  - Verify exactly ONE done event generated
  - Run with `-race` flag
- Success Criteria: No race conditions, single done event

**Test 9: TestDoneEventWithHistory**
- Setup: State with history, transitions to final
- Actions:
  - Enter state, transition to final
  - Verify done event
  - Exit and re-enter via history
  - Verify done event generated again
- Success Criteria: Done events work with history states
- Note: Requires history state implementation (Limitation 3)

### Implementation Plan

#### Step 1: Core Done Event Logic (2-3 hours)
1. Add `generateDoneEvent()` function
2. Add `shouldEmitDoneEvent()` function
3. Add `allRegionsInFinalState()` function
4. Integrate into `enterState()` function
5. Add done event tracking to prevent duplicates

#### Step 2: Parallel State Done Events (2-3 hours)
1. Implement parallel region completion tracking
2. Add thread-safe checks for all regions final
3. Handle concurrent completion scenarios
4. Add timeout protection for done event generation

#### Step 3: Done Event Data Support (1-2 hours)
1. Add data field to final states
2. Propagate data to done events
3. Document data access in transitions

#### Step 4: Testing (3-4 hours)
1. Implement all 9 tests (8 initially, test 9 after history)
2. Run with `-race` flag
3. Test with existing 32 tests to ensure no regressions
4. Add stress test: 100 parallel regions completing

#### Step 5: Documentation (1 hour)
1. Add done event examples to README
2. Document SCXML compatibility
3. Add API documentation for done event format
4. Create migration guide for users

### Success Metrics
- ✅ All 8 done event tests pass (9 after history)
- ✅ No race conditions with `-race` flag
- ✅ No regressions in existing 32 tests
- ✅ Done events work with sequential states
- ✅ Done events work with parallel states
- ✅ Done events trigger automatic transitions
- ✅ SCXML compatibility documented

### Estimated Effort
- **Core implementation**: 2-3 hours
- **Parallel state support**: 2-3 hours
- **Data support**: 1-2 hours
- **Testing**: 3-4 hours
- **Documentation**: 1 hour
- **Total**: 9-13 hours

### Dependencies
- None (can be implemented immediately)

### Breaking Changes
- None (additive feature)

---

## Limitation 3: History States

### Current Status
- **Listed as**: "Not implemented (future work)"
- **Priority**: **MEDIUM**
- **Complexity**: High

### Problem Analysis
SCXML specification defines history states for remembering and restoring previous state configurations:

**Shallow History**: Restores direct child state only
**Deep History**: Restores entire state hierarchy

History states enable:
- Pause/resume functionality
- State restoration after interruptions
- Complex navigation patterns
- User experience continuity

Without history states, applications must manually track and restore state, increasing complexity.

### SCXML Specification Reference

**History State Types**:
1. **Shallow History** (`<history type="shallow">`): Restores only the immediate child state
2. **Deep History** (`<history type="deep">`): Restores the entire state configuration

**Behavior**:
- History states are pseudo-states (not real states)
- Transitioning to history state restores last active configuration
- If no history exists, transition to default state
- Each compound state can have one shallow and one deep history

### Implementation Approach

#### Architecture Changes

**1. Add History State Types**
```go
// In statechart.go

type HistoryType int

const (
    HistoryNone    HistoryType = iota
    HistoryShallow              // Restore direct child only
    HistoryDeep                 // Restore entire hierarchy
)

type State struct {
    ID              StateID
    IsParallel      bool
    IsFinalState    bool
    IsHistoryState  bool         // NEW: Mark as history pseudo-state
    HistoryType     HistoryType  // NEW: Shallow or deep
    HistoryDefault  StateID      // NEW: Default state if no history
    Children        map[StateID]*State
    Transitions     []*Transition
    OnEntry         Action
    OnExit          Action
    Parent          *State       // NEW: Parent reference for traversal
}
```

**2. Add History Tracking to Runtime**
```go
type Runtime struct {
    // ... existing fields
    
    // History tracking
    history   map[StateID]StateID      // stateID → last active child
    historyMu sync.RWMutex
    
    // Deep history tracking (full configuration)
    deepHistory   map[StateID][]StateID  // stateID → full state path
    deepHistoryMu sync.RWMutex
}

// Record history when exiting state
func (rt *Runtime) recordHistory(parentID StateID, childID StateID, deep bool) {
    if deep {
        rt.deepHistoryMu.Lock()
        defer rt.deepHistoryMu.Unlock()
        
        // Store full active state configuration
        config := rt.getActiveConfiguration()
        rt.deepHistory[parentID] = config
    } else {
        rt.historyMu.Lock()
        defer rt.historyMu.Unlock()
        
        // Store only direct child
        rt.history[parentID] = childID
    }
}

// Restore history when entering history state
func (rt *Runtime) restoreHistory(historyState *State) (StateID, error) {
    if historyState.HistoryType == HistoryDeep {
        return rt.restoreDeepHistory(historyState)
    }
    return rt.restoreShallowHistory(historyState)
}

func (rt *Runtime) restoreShallowHistory(historyState *State) (StateID, error) {
    rt.historyMu.RLock()
    defer rt.historyMu.RUnlock()
    
    parentID := historyState.Parent.ID
    lastChild, exists := rt.history[parentID]
    
    if !exists {
        // No history, use default
        return historyState.HistoryDefault, nil
    }
    
    return lastChild, nil
}

func (rt *Runtime) restoreDeepHistory(historyState *State) (StateID, error) {
    rt.deepHistoryMu.RLock()
    defer rt.deepHistoryMu.RUnlock()
    
    parentID := historyState.Parent.ID
    config, exists := rt.deepHistory[parentID]
    
    if !exists {
        // No history, use default
        return historyState.HistoryDefault, nil
    }
    
    // Restore full configuration
    return rt.restoreConfiguration(config)
}
```

**3. Integrate into Transition Logic**
```go
// In executeTransition function
func (rt *Runtime) executeTransition(ctx context.Context, t *Transition, event Event) error {
    // ... existing exit logic
    
    // Check if target is history state
    targetState := rt.machine.GetState(t.Target)
    if targetState.IsHistoryState {
        // Restore history instead of entering target directly
        restoredState, err := rt.restoreHistory(targetState)
        if err != nil {
            return err
        }
        t.Target = restoredState  // Redirect to restored state
    }
    
    // ... existing entry logic
    return nil
}

// Record history when exiting states
func (rt *Runtime) exitState(ctx context.Context, stateID StateID) error {
    state := rt.machine.GetState(stateID)
    if state == nil {
        return fmt.Errorf("state not found: %v", stateID)
    }
    
    // Record history before exiting
    if state.Parent != nil {
        rt.recordHistory(state.Parent.ID, stateID, false)  // Shallow
        rt.recordHistory(state.Parent.ID, stateID, true)   // Deep
    }
    
    // Execute exit actions
    if state.OnExit != nil {
        state.OnExit(ctx, Event{})
    }
    
    // ... rest of exit logic
    return nil
}
```

**4. Add Parent References**
```go
// In NewMachine function
func NewMachine(root *State) (*Machine, error) {
    m := &Machine{
        states: make(map[StateID]*State),
        root:   root,
    }
    
    // Build state tree and set parent references
    m.buildStateTree(root, nil)
    
    return m, nil
}

func (m *Machine) buildStateTree(state *State, parent *State) {
    state.Parent = parent  // Set parent reference
    m.states[state.ID] = state
    
    for _, child := range state.Children {
        m.buildStateTree(child, state)
    }
}
```

### Test Plan

#### Test Suite: `statechart_history_test.go`

**Test 1: TestShallowHistoryBasic**
- Setup:
  - Parent state P with children A, B, C
  - History state H (shallow) with default A
  - External state X
- Actions:
  - Enter P → A
  - Transition P → X (exit P, record history)
  - Transition X → H (restore history)
  - Verify current state is A (restored)
- Success Criteria: Shallow history restores last child

**Test 2: TestShallowHistoryAfterTransition**
- Setup: Same as Test 1
- Actions:
  - Enter P → A
  - Transition A → B (within P)
  - Transition P → X (exit P, record B as history)
  - Transition X → H (restore history)
  - Verify current state is B (not A)
- Success Criteria: History reflects last active child

**Test 3: TestShallowHistoryDefault**
- Setup: Same as Test 1
- Actions:
  - Never enter P (no history recorded)
  - Transition X → H (no history exists)
  - Verify current state is A (default)
- Success Criteria: Default state used when no history

**Test 4: TestDeepHistoryBasic**
- Setup:
  - Parent P with child A
  - A has children A1, A2
  - History state H (deep) with default A→A1
  - External state X
- Actions:
  - Enter P → A → A2
  - Transition P → X (exit P, record deep history)
  - Transition X → H (restore deep history)
  - Verify current state is A2 (full path restored)
- Success Criteria: Deep history restores full hierarchy

**Test 5: TestDeepHistoryVsShallowHistory**
- Setup:
  - Parent P with child A (has children A1, A2)
  - Shallow history HS, deep history HD
  - External state X
- Actions:
  - Enter P → A → A2
  - Transition P → X
  - Transition X → HS (shallow restore)
  - Verify current state is A → A1 (default child of A)
  - Transition A → X
  - Transition X → HD (deep restore)
  - Verify current state is A → A2 (full path)
- Success Criteria: Shallow vs deep behavior correct

**Test 6: TestHistoryWithParallelStates**
- Setup:
  - Parallel state P with regions R1, R2
  - Each region has children
  - Deep history state H
  - External state X
- Actions:
  - Enter P (R1→A, R2→B)
  - Transition R1: A → C
  - Transition P → X (exit parallel, record history)
  - Transition X → H (restore parallel history)
  - Verify R1 in C, R2 in B
- Success Criteria: History restores all parallel regions

**Test 7: TestHistoryMultipleEntries**
- Setup: Parent P with children A, B, C and history H
- Actions:
  - Enter P → A, exit to X (history = A)
  - Enter P → B, exit to X (history = B, overwrites A)
  - Enter P → C, exit to X (history = C, overwrites B)
  - Transition X → H
  - Verify current state is C (most recent)
- Success Criteria: History updates on each exit

**Test 8: TestHistoryConcurrentAccess**
- Setup: Parallel state with history in multiple regions
- Actions:
  - Multiple goroutines record/restore history concurrently
  - Run with `-race` flag
  - Verify no race conditions
  - Verify history integrity
- Success Criteria: Thread-safe history operations

**Test 9: TestHistoryWithDoneEvents**
- Setup: State with history that transitions to final state
- Actions:
  - Enter P → A (final state)
  - Verify done event generated
  - Exit P, re-enter via history
  - Verify done event generated again
- Success Criteria: History works with done events
- Note: Requires done event implementation (Limitation 2)

**Test 10: TestHistoryClearOnReset**
- Setup: State machine with history
- Actions:
  - Enter states, record history
  - Call runtime.Reset() or runtime.Stop()
  - Verify history cleared
  - Re-enter via history
  - Verify default state used (no history)
- Success Criteria: History cleared on reset

### Implementation Plan

#### Step 1: Data Structures (1-2 hours)
1. Add `HistoryType` enum
2. Add history fields to `State` struct
3. Add history tracking to `Runtime` struct
4. Add parent references to states

#### Step 2: History Recording (2-3 hours)
1. Implement `recordHistory()` function
2. Integrate into `exitState()` function
3. Add shallow history recording
4. Add deep history recording (full configuration)
5. Add thread-safety (mutexes)

#### Step 3: History Restoration (3-4 hours)
1. Implement `restoreHistory()` function
2. Implement `restoreShallowHistory()` function
3. Implement `restoreDeepHistory()` function
4. Implement `restoreConfiguration()` for deep history
5. Handle default states when no history exists

#### Step 4: Integration (2-3 hours)
1. Modify `executeTransition()` to detect history states
2. Redirect transitions to restored states
3. Update `NewMachine()` to build parent references
4. Add history state validation

#### Step 5: Parallel State Support (2-3 hours)
1. Extend history to track parallel region configurations
2. Restore all parallel regions from history
3. Handle concurrent history recording
4. Add timeout protection

#### Step 6: Testing (4-5 hours)
1. Implement all 10 tests
2. Run with `-race` flag
3. Test with existing 32 tests (no regressions)
4. Add stress test: 1000 history save/restore cycles

#### Step 7: Documentation (1-2 hours)
1. Add history state examples to README
2. Document shallow vs deep history
3. Add API documentation
4. Create migration guide
5. Document SCXML compatibility

### Success Metrics
- ✅ All 10 history tests pass
- ✅ No race conditions with `-race` flag
- ✅ No regressions in existing tests
- ✅ Shallow history works correctly
- ✅ Deep history works correctly
- ✅ History works with parallel states
- ✅ Thread-safe history operations
- ✅ SCXML compatibility documented

### Estimated Effort
- **Data structures**: 1-2 hours
- **History recording**: 2-3 hours
- **History restoration**: 3-4 hours
- **Integration**: 2-3 hours
- **Parallel support**: 2-3 hours
- **Testing**: 4-5 hours
- **Documentation**: 1-2 hours
- **Total**: 15-22 hours

### Dependencies
- None (can be implemented immediately)
- Recommended after done events for complete testing

### Breaking Changes
- None (additive feature)
- Requires adding `Parent` field to `State` struct (backward compatible)

---

## Priority Summary

### Critical Priority
1. **Done Events** (9-13 hours)
   - Essential for SCXML compatibility
   - Enables automatic transitions
   - Required for parallel state coordination
   - No dependencies

### High Priority
2. **Nested Parallel States** (4.5-7.5 hours)
   - Testing/validation focus
   - Ensures robustness of existing implementation
   - Low risk (mostly testing)
   - No dependencies

### Medium Priority
3. **History States** (15-22 hours)
   - Complex feature
   - Enables advanced use cases
   - Can be deferred if needed
   - Benefits from done events being implemented first

---

## Implementation Sequence Recommendation

### Phase A: Validation & Critical Features (2-3 weeks)
1. **Week 1**: Nested Parallel States testing (4.5-7.5 hours)
2. **Week 2**: Done Events implementation (9-13 hours)
3. **Week 3**: Integration testing and documentation

### Phase B: Advanced Features (2-3 weeks)
4. **Week 4-5**: History States implementation (15-22 hours)
5. **Week 6**: Comprehensive integration testing

### Phase C: Performance & Optimization (see performance testing plan)
6. **Week 7+**: Extreme performance testing and optimization

---

## Risk Assessment

### Low Risk
- **Nested Parallel States**: Mostly testing, existing code likely works

### Medium Risk
- **Done Events**: New event generation logic, potential for race conditions
- Mitigation: Extensive testing with `-race` flag, timeout protection

### High Risk
- **History States**: Complex state restoration, deep hierarchy tracking
- Mitigation: Incremental implementation, extensive testing, clear documentation

---

## Testing Strategy

All implementations must:
1. ✅ Pass with `-race` flag (no race conditions)
2. ✅ Pass with `-timeout 30s` (no hangs)
3. ✅ No goroutine leaks
4. ✅ No regressions in existing 32 tests
5. ✅ Comprehensive test coverage (>80%)
6. ✅ Stress tests for concurrent scenarios
7. ✅ Documentation with examples

---

## Success Criteria

### Completion Criteria
- All limitations addressed (except datamodel)
- All new tests passing
- No regressions in existing tests
- Documentation updated
- SCXML compatibility improved

### Quality Criteria
- No race conditions
- No goroutine leaks
- No performance degradation
- Clean, maintainable code
- Comprehensive test coverage

---

## Appendix: Datamodel Limitation

**Status**: Explicitly excluded from this plan (left to users)

**Rationale**:
- Datamodel is application-specific
- Users can implement using context or external state
- Library provides event data passing mechanism
- Adding built-in datamodel would impose opinions on users

**User Implementation Options**:
1. Use `Event.Data` field for passing data
2. Use Go context for shared state
3. Use external state management (e.g., struct with mutex)
4. Use channels for data flow

**Documentation Needed**:
- Add examples of datamodel patterns
- Document best practices for state management
- Show integration with popular state management libraries

---

**End of Limitations Plan**
