# StatechartX Phase 4 - Quick Reference Card

## New Features

### 1. Initial Transition Actions (Step 10)

**Purpose**: Execute initialization logic when entering a compound state's initial child.

**Usage**:
```go
parentState := &State{
    ID:      STATE_PARENT,
    Initial: STATE_CHILD,
    
    // Entry action runs first
    EntryAction: func(ctx context.Context, event *Event, from, to StateID) error {
        fmt.Println("Parent entered")
        return nil
    },
    
    // InitialAction runs after parent entry, before child entry
    InitialAction: func(ctx context.Context, event *Event, from, to StateID) error {
        fmt.Println("Initializing child transition")
        return nil
    },
}

childState := &State{
    ID: STATE_CHILD,
    
    // Entry action runs last
    EntryAction: func(ctx context.Context, event *Event, from, to StateID) error {
        fmt.Println("Child entered")
        return nil
    },
}
```

**Execution Order**:
1. Parent `EntryAction`
2. Parent `InitialAction` ← NEW!
3. Child `EntryAction`

### 2. Event Matching Priority (Step 11)

**Priority Rules** (highest to lowest):

1. **Specific EventID** beats `ANY_EVENT` wildcard
   ```go
   state.Transitions = []*Transition{
       {Event: ANY_EVENT, Target: STATE_FAIL},  // Lower priority
       {Event: EVENT_FOO, Target: STATE_PASS},  // Higher priority (wins)
   }
   ```

2. **Eventless (NO_EVENT)** beats event-driven
   ```go
   state.Transitions = []*Transition{
       {Event: NO_EVENT, Target: STATE_PASS},   // Fires immediately
       {Event: EVENT_FOO, Target: STATE_FAIL},  // Never fires
   }
   ```

3. **Document order** within same priority level
   ```go
   state.Transitions = []*Transition{
       {Event: EVENT_FOO, Target: STATE_FIRST},  // Fires first
       {Event: EVENT_FOO, Target: STATE_SECOND}, // Never fires
   }
   ```

4. **Guards** enable fallthrough
   ```go
   state.Transitions = []*Transition{
       {
           Event: EVENT_FOO,
           Target: STATE_SKIP,
           Guard: func(...) (bool, error) { return false, nil }, // Fails
       },
       {Event: EVENT_FOO, Target: STATE_PASS}, // Fires (fallthrough)
   }
   ```

### 3. Final States (Step 12)

**Purpose**: Mark states as final/terminal states.

**Usage**:
```go
finalState := &State{
    ID:      STATE_FINAL,
    IsFinal: true,  // NEW! Marks this as a final state
}

// Or use deprecated field for backward compatibility
finalState := &State{
    ID:    STATE_FINAL,
    Final: true,  // Deprecated, use IsFinal instead
}
```

**Behavior**:
- Final state detection is automatic
- `checkFinalState()` called after every state entry
- Future: Will generate `done.state.id` events (Phase 5+)

## API Changes

### State Struct
```go
type State struct {
    ID            StateID
    Transitions   []*Transition
    EntryAction   Action
    ExitAction    Action
    InitialAction Action  // NEW: Runs after parent entry, before child entry
    IsFinal       bool    // NEW: True if this is a final state
    Final         bool    // Deprecated: use IsFinal instead
    Parent        *State
    Children      map[StateID]*State
    Initial       StateID
}
```

### No Breaking Changes
- All existing code continues to work
- New fields are optional (default to nil/false)
- Backward compatible with Phase 1-3 code

## Testing

### Run Phase 4 Tests Only
```bash
go test -v -run "TestSCXML412|TestSCXML421|TestEventMatchingPriority|TestEventlessTransition|TestFinalState|TestInitialAction"
```

### Run All Tests
```bash
go test -v
```

### Expected Results
- 20 tests passing
- 0 failures
- Many tests skipped (future phases)

## Common Patterns

### Pattern 1: Initialization with InitialAction
```go
// Use InitialAction for setup that needs to happen
// after parent is entered but before child becomes active
parent.InitialAction = func(ctx context.Context, event *Event, from, to StateID) error {
    // Initialize resources
    // Set up child state context
    // Log transition
    return nil
}
```

### Pattern 2: Priority-Based Transitions
```go
// Order matters! Put specific conditions first
state.Transitions = []*Transition{
    // 1. Most specific (with guard)
    {Event: EVENT_SPECIAL, Target: STATE_SPECIAL, Guard: specialGuard},
    
    // 2. Specific event
    {Event: EVENT_NORMAL, Target: STATE_NORMAL},
    
    // 3. Wildcard (catches everything else)
    {Event: ANY_EVENT, Target: STATE_DEFAULT},
}
```

### Pattern 3: Final State with Cleanup
```go
finalState := &State{
    ID:      STATE_FINAL,
    IsFinal: true,
    
    EntryAction: func(ctx context.Context, event *Event, from, to StateID) error {
        // Cleanup resources
        // Log completion
        // Notify observers
        return nil
    },
}
```

## Migration Guide

### From Phase 3 to Phase 4

**No changes required!** Phase 4 is fully backward compatible.

**Optional enhancements**:

1. Add InitialAction for compound states:
   ```go
   // Before (Phase 3)
   parent.EntryAction = func(...) error {
       // Setup code here
       return nil
   }
   
   // After (Phase 4) - more precise control
   parent.EntryAction = func(...) error {
       // Parent-specific setup
       return nil
   }
   parent.InitialAction = func(...) error {
       // Child initialization
       return nil
   }
   ```

2. Use IsFinal instead of Final:
   ```go
   // Before (Phase 3)
   state.Final = true
   
   // After (Phase 4) - preferred
   state.IsFinal = true
   ```

## Performance Notes

- InitialAction adds minimal overhead (only on state entry)
- Event matching priority is O(n) where n = number of transitions
- Final state checking is O(1) (simple boolean check)
- No performance regressions from Phase 3

## Debugging Tips

### Debug InitialAction Execution
```go
parent.InitialAction = func(ctx context.Context, event *Event, from, to StateID) error {
    log.Printf("InitialAction: from=%d to=%d", from, to)
    return nil
}
```

### Debug Event Matching Priority
```go
// Add guards with logging to see which transitions are evaluated
transition.Guard = func(ctx context.Context, event *Event, from, to StateID) (bool, error) {
    result := /* your condition */
    log.Printf("Guard evaluated: event=%d from=%d to=%d result=%v", event.ID, from, to, result)
    return result, nil
}
```

### Debug Final State Detection
```go
finalState.EntryAction = func(ctx context.Context, event *Event, from, to StateID) error {
    log.Printf("Entered final state: %d", to)
    return nil
}
```

## Known Limitations

1. **Done Events**: `done.state.id` events not yet implemented (Phase 5+)
2. **Parallel States**: No parallel region support yet (Phase 5+)
3. **History States**: Not implemented (Phase 6+)
4. **Dynamic EventIDs**: EventID must be compile-time constants

## Next Steps

Ready to implement Phase 5? See `/home/ubuntu/statechartx_incremental_plan.md` for:
- Parallel states (orthogonal regions)
- Done event generation
- History states
- Advanced SCXML features

---

**Phase 4 Status**: ✅ Complete (20/20 tests passing)
**Branch**: `phase1-runtime`
**Commit**: `2fa6e89` - "Phase 4: Implement Advanced Features (Steps 10-12)"
