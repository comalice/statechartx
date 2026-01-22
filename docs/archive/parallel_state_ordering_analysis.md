# Parallel State Ordering Analysis

## Date: 2026-01-03

## Summary

After reviewing the realtime and event-driven runtime implementations for parallel state handling, I've identified that **neither runtime currently provides deterministic event ordering guarantees for parallel states**. This affects SCXML conformance tests 404, 405, and 406.

## Implementation Details

### Event-Driven Runtime (statechart.go)

The event-driven runtime uses a **goroutine-per-region** architecture:

1. **Entry** (`enterParallelState`, lines 323-402):
   - Spawns one goroutine per child region
   - Each goroutine runs independently with its own event channel
   - Entry actions execute **concurrently** across regions

2. **Exit** (`exitParallelState`, lines 404-456):
   - Signals all regions to stop via `close(region.done)`
   - Waits for goroutines to finish with timeout
   - Exit actions execute **concurrently** in each region goroutine

3. **Event Routing** (`sendEventToRegions`, lines 744-782):
   - Broadcast (address=0): sends to **all** region channels concurrently
   - Targeted (address=stateID): sends to specific region channel
   - No ordering guarantee across regions

### Realtime Runtime (realtime/)

The realtime runtime **embeds** the event-driven Runtime and adds tick-based event batching:

1. **Tick Processing** (`processTick`, realtime/tick.go:8-23):
   - Collects events from batch queue
   - Sorts events for deterministic order
   - Calls `rt.Runtime.ProcessEvent()` - **delegates to event-driven runtime**

2. **Parallel State Processing** (`processParallelRegionsSequentially`, line 52-56):
   ```go
   // TODO: Implement in Phase 3 when parallel state support is added
   // Will reuse existing transition methods but process sequentially
   ```
   **This is a stub!** No sequential processing is implemented.

3. **Current Behavior**:
   - Realtime runtime **inherits goroutine-based parallel state handling**
   - No deterministic ordering for parallel region entry/exit/events

## SCXML Test Requirements

### Test 404: Parallel Exit Order
- **Expected**: Children exit in reverse document order before parents
- **Sequence**: s01p2 exit → s01p1 exit → s01p exit → transition action
- **Events**: event1 (s01p2) → event2 (s01p1) → event3 (s01p) → event4 (transition)

### Test 405: Parallel Transition Execution Order
- **Expected**: Transitions execute in document order after exits complete
- **Sequence**: s01p21 exit → s01p11 exit → s01p11→s01p12 transition → s01p21→s01p22 transition
- **Events**: event1 (s01p21 exit) → event2 (s01p11 exit) → event3 (transition 1) → event4 (transition 2)

### Test 406: Parallel Entry Order
- **Expected**: States enter in entry order (parents before children, document order for siblings)
- **Sequence**: s01→s0p2 transition action → s0p2 entry → s01p21 entry → s01p22 entry
- **Events**: event1 (transition) → event2 (s0p2) → event3 (s01p21) → event4 (s01p22)

## Why Tests Fail

The goroutine-based implementation **cannot guarantee** these orderings because:

1. **Concurrent Execution**: Goroutines are scheduled by Go runtime, not in document order
2. **Race Conditions**: Entry/exit actions in different regions execute simultaneously
3. **Channel Broadcast**: Events sent to region channels arrive in undefined order
4. **No Synchronization**: No barriers or ordering constraints between regions

## Options for Resolution

### Option 1: Implement Sequential Processing in Realtime Runtime (RECOMMENDED)

**Pros**:
- Enables SCXML conformance for realtime use cases
- Deterministic behavior for game engines, physics sims, robotics
- Maintains existing goroutine-based implementation for event-driven use cases

**Cons**:
- Requires implementing `processParallelRegionsSequentially()`
- Higher latency for parallel states in realtime runtime
- Two different execution models to maintain

**Implementation**:
- Process parallel region entry/exit in document order
- Execute entry/exit actions sequentially
- Route events to regions one at a time in document order
- Keep goroutine-based implementation for event-driven runtime

### Option 2: Relax SCXML Test Requirements for Both Runtimes

**Pros**:
- Acknowledges fundamental architectural constraint
- No code changes needed
- Honest about capabilities

**Cons**:
- Reduces SCXML conformance
- May surprise users expecting deterministic parallel state behavior
- Limits use cases (games, simulations requiring reproducibility)

**Implementation**:
- Mark tests 404, 405, 406 as "skip" with explanation
- Document in README that parallel states have no ordering guarantees
- Add note that this is by design (goroutine-based parallelism)

### Option 3: Implement Sequential Mode for Event-Driven Runtime

**Pros**:
- Single implementation
- Users can choose concurrent vs sequential via configuration

**Cons**:
- Complicates event-driven runtime
- May confuse users about performance characteristics
- Requires runtime mode switching logic

## Recommendation

**Implement Option 1**: Add sequential parallel state processing to the realtime runtime only.

### Rationale:

1. **Use Case Alignment**:
   - Realtime runtime targets **deterministic** scenarios (games, physics, robotics)
   - Event-driven runtime targets **throughput** scenarios (servers, async workflows)

2. **Performance Trade-offs**:
   - Realtime already has ~16.67ms latency per tick @ 60 FPS
   - Sequential processing adds minimal overhead compared to tick latency
   - Event-driven needs maximum throughput, benefits from concurrent regions

3. **SCXML Conformance**:
   - Tests 404, 405, 406 can pass in realtime runtime
   - Document event-driven runtime's relaxed ordering guarantees
   - Users choose appropriate runtime for their needs

## Implementation Plan

### Phase 1: Realtime Sequential Processing

1. **Implement `processParallelRegionsSequentially()`**:
   ```go
   func (rt *RealtimeRuntime) processParallelRegionsSequentially() {
       // Get current parallel state (if any)
       current := rt.Runtime.GetCurrentState()
       state := rt.Runtime.machine.states[current]
       if state == nil || !state.IsParallel {
           return
       }

       // Process regions in document order (sorted by StateID)
       regionIDs := make([]StateID, 0, len(state.Children))
       for id := range state.Children {
           regionIDs = append(regionIDs, id)
       }
       sort.Slice(regionIDs, func(i, j int) bool {
           return regionIDs[i] < regionIDs[j]
       })

       // Process each region sequentially
       for _, regionID := range regionIDs {
           rt.processRegion(regionID)
       }
   }
   ```

2. **Override parallel state entry/exit** in RealtimeRuntime:
   - Detect parallel state transitions during tick processing
   - Execute entry/exit actions in document order
   - Don't spawn goroutines - maintain region state in RealtimeRuntime

3. **Update test expectations**:
   - Tests 404, 405, 406 should pass in realtime runtime
   - Add documentation about sequential processing

### Phase 2: Documentation

1. **Update README.md**:
   - Document ordering guarantees per runtime type
   - Add decision matrix: "Use realtime for determinism, event-driven for throughput"

2. **Update realtime/README.md**:
   - Document sequential parallel state processing
   - Note performance characteristics

3. **Update CLAUDE.md**:
   - Add guidance on which runtime to use for different scenarios
   - Document parallel state behavior differences

## Testing Strategy

### Realtime Runtime:
- Tests 404, 405, 406: **Full SCXML conformance** (deterministic ordering)
- Test with `-race` flag to ensure no data races in sequential processing

### Event-Driven Runtime:
- Tests 404, 405, 406: **Relaxed tests** that verify:
  - All expected states are entered/exited
  - All expected events are raised
  - Final state is correct
  - **Order is NOT verified**

- Example relaxed test:
  ```go
  func TestSCXML404_EventDriven_Relaxed(t *testing.T) {
      // Test that parallel states exit correctly (order not guaranteed)
      // Verify: all events raised, final state correct, no deadlocks

      // ... setup ...

      time.Sleep(200 * time.Millisecond)

      // Relaxed assertion: we should end up in PASS state
      // even if event order varies across runs
      if currentState != STATE_PASS {
          t.Error("Should eventually reach pass state (event order may vary)")
      }
  }
  ```

## References

- Archive docs: `docs/archive/statechartx_parallel_testing_addendum.md`
- Phase 5 summary: `docs/archive/PHASE5_SUMMARY.md`
- SCXML test sources: `test/scxml/w3c_test_suite/404/`, `405/`, `406/`
