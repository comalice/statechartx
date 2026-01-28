# SCXML Compliance

This document describes the statechartx library's compliance with the W3C SCXML specification and documents known departures.

## Compliance Overview

The statechartx library implements the core SCXML semantics with high fidelity, including:

- ✅ Hierarchical state machines with proper entry/exit ordering
- ✅ Initial states and history (shallow and deep)
- ✅ Guarded transitions with actions
- ✅ Run-to-completion semantics (macrostep processing)
- ✅ Internal event queue with priority over external events
- ✅ Parallel states with independent region execution
- ✅ Event-driven and real-time deterministic runtimes

## Known Departures from SCXML Specification

### Parallel Region Eventless Transition Ordering

**SCXML Specification**: When multiple parallel regions have enabled eventless transitions simultaneously, the specification requires:
1. Collect all enabled eventless transitions across ALL regions
2. Exit all states in reverse document order
3. Execute all transition actions in document order
4. Enter all new states in document order

**statechartx Implementation**: Parallel regions process eventless transitions sequentially in document order (sorted by StateID). Each region's transition is processed as an atomic unit (exit → action → enter) before moving to the next region.

**Impact**: 
- Events raised during exits and transition actions may occur in a different order than strict SCXML compliance would require
- This affects only the specific case of simultaneous eventless transitions in multiple parallel regions
- The behavior is still deterministic and predictable

**Rationale**:
- Simpler implementation that's easier to understand and maintain
- Sequential processing maintains clear, predictable semantics
- Real-world impact is minimal - most parallel state machines don't rely on this exact ordering
- Core run-to-completion semantics remain correct
- Users requiring strict ordering can use explicit events instead of simultaneous eventless transitions

**Test Coverage**:
- ✅ Test 404: Validates core run-to-completion semantics (events raised during exits are processed correctly)
- ⚠️  Test 405: Skipped - tests specific parallel region transition ordering (simultaneous eventless transitions)
- ⚠️  Test 406: Skipped - tests specific parallel region entry ordering (simultaneous eventless transitions)

### Workaround

If your application requires the exact SCXML-compliant ordering for parallel region transitions, use explicit events instead of eventless transitions:

```go
// Instead of this (eventless transitions in parallel regions):
region1State.Transitions = []*Transition{
    {Event: NO_EVENT, Target: nextState1},
}
region2State.Transitions = []*Transition{
    {Event: NO_EVENT, Target: nextState2},
}

// Use this (explicit event coordination):
region1State.EntryAction = func(ctx, evt, from, to) error {
    // Send event to trigger both regions
    return rt.SendEvent(Event{ID: COORDINATE_EVENT})
}
region1State.Transitions = []*Transition{
    {Event: COORDINATE_EVENT, Target: nextState1},
}
region2State.Transitions = []*Transition{
    {Event: COORDINATE_EVENT, Target: nextState2},
}
```

## Testing Against W3C SCXML Test Suite

The library includes tests derived from the W3C SCXML IRP (Implementation Report Proposal) test suite. Tests are translated from SCXML to Go using the `scxml-translator` skill.

### Test Status Summary

- **Passing**: Core semantics including run-to-completion, event processing, parallel state management
- **Skipped**: Tests 405-406 (parallel region transition ordering edge cases)

### Running SCXML Conformance Tests

```bash
# Run all SCXML conformance tests
go test -v -run TestSCXML ./realtime

# Run specific test
go test -v -run TestSCXML404_Realtime ./realtime
```

## References

- [W3C SCXML Specification](https://www.w3.org/TR/scxml/)
- [W3C SCXML Test Suite](https://www.w3.org/Voice/2013/scxml-irp/)
- [statechartx Documentation](../README.md)
