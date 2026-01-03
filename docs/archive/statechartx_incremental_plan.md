# StatechartX Incremental Implementation Plan

## Overview

This plan provides a step-by-step approach to implementing the statechartx library, where each step is simple, focused, and naturally leads to the next architectural decision. The plan is based on analyzing 205 tests (17 non-skipped, 185 skipped) and the current 254-line implementation.

**Architecture Philosophy**: Synchronous FSM for normal/compound states + goroutines for parallel states (to avoid lock hell)

**Current State**: Basic `Machine` API exists with `State`, `Transition`, `Action`, `Guard` types using int-based `StateID` and `EventID`. Tests currently expect old string-based API but **will be updated** to use the new int-based API.

**Key Design Decisions**:
- **Int-based IDs**: `StateID int` and `EventID int` (no string conversion)
- **Event Addressing**: Events have an `address` field in their structure for routing
- **No Backward Compatibility**: Clean new API design, tests will be updated to match
- **Architecture**: Sync FSM for compound states, goroutines for parallel states

---

## Phase 1: Foundation - Runtime & Event Queue (Steps 1-3)

### Step 1: Implement Basic Runtime with Event Queue
**Goal**: Create the `Runtime` wrapper with internal event queue using int-based types

**What to implement**:
- `Runtime` struct wrapping `Machine` with event queue (channel-based)
- `NewRuntime(machine *Machine, ext any) *Runtime` constructor
- `Start(ctx context.Context)` - starts event loop goroutine
- `Stop()` - stops event loop gracefully
- `SendEvent(ctx context.Context, event Event)` - queues events (takes Event struct, not string)
- `IsInState(stateID StateID) bool` - checks current state
- Basic event loop that processes events sequentially

**What to update in tests**:
- Change `rt.SendEvent(ctx, "foo")` to `rt.SendEvent(ctx, Event{ID: FOO_EVENT})`
- Define event constants: `const (FOO_EVENT EventID = 1; BAR_EVENT EventID = 2; ...)`
- Define state constants: `const (STATE_A StateID = 1; STATE_B StateID = 2; ...)`

**Tests that will pass**: None yet (need more pieces)

**Architectural decision revealed**: 
- Event struct needs `address` field for routing (used in hierarchical/parallel states)
- Event queue processes `Event` structs with int IDs
- StateID remains `int` throughout (no type conversion needed)

**Complexity**: Medium

**Key insights**:
- No string-to-int conversion layer needed
- Event struct: `type Event struct { ID EventID; Data any; address StateID }`
- Runtime needs goroutine for event loop + channel for Event structs

---

### Step 2: Implement Event Addressing via event.address
**Goal**: Use event.address field for routing events to specific states

**What to implement**:
- Add `address StateID` field to Event struct
- When `event.address == 0`, event is broadcast (current behavior)
- When `event.address != 0`, event is routed only to that state (and its ancestors)
- Update `pickTransition()` to check if current state matches event.address
- Update `SendEvent()` to accept events with optional address

**What to update in tests**:
- Tests that need targeted events: `rt.SendEvent(ctx, Event{ID: FOO_EVENT, address: STATE_A})`
- Tests with broadcast events: `rt.SendEvent(ctx, Event{ID: FOO_EVENT})` (address defaults to 0)

**Tests that will pass**: 
- `TestSCXML144` - Basic sequential transitions with event queue (after test update)
- `TestSCXML148` - Event queue ordering (after test update)
- `TestSCXML149` - Simple event matching (after test update)

**Architectural decision revealed**:
- Event addressing enables targeted communication in parallel states
- Address 0 means "broadcast to all active states"
- Non-zero address means "only deliver to this state and its ancestors"
- Event queue must preserve FIFO order

**Complexity**: Simple

---

### Step 3: Implement Wildcard Event Matching
**Goal**: Support wildcard EventID (e.g., `ANY_EVENT`) in transitions

**What to implement**:
- Define special constant: `const ANY_EVENT EventID = -1` (or similar)
- Update `pickTransition()` to handle `ANY_EVENT` as wildcard
- Wildcard should match any event but have lower priority than specific matches
- Document order matters: first matching transition wins

**What to update in tests**:
- Change `Event: "*"` to `Event: ANY_EVENT` in transition definitions
- Tests: `TestSCXML147`, `TestSCXML158`

**Tests that will pass**:
- `TestSCXML147` - Wildcard event matching (after test update)
- `TestSCXML158` - Multiple events queued, processed in order (after test update)

**Architectural decision revealed**:
- Need to handle transition priority: specific events > wildcard
- Document order (slice order) determines precedence
- Guards can make transitions conditional

**Complexity**: Simple

---

## Phase 2: Hierarchical States (Steps 4-6)

### Step 4: Implement Parent-Child State Relationships
**Goal**: Support compound states with Parent/Children relationships using int StateIDs

**What to implement**:
- Add `Parent *State` field to State (already in code)
- Add `Children map[StateID]*State` field to State (already in code)
- Add `Initial StateID` field to State for default child (int, not pointer)
- Update `NewMachine()` to build state hierarchy from root
- Update `IsInState()` to check if current state or any ancestor matches

**What to update in tests**:
- Define state hierarchy with int IDs: `s0 := &State{ID: STATE_S0, Initial: STATE_S01, Children: map[StateID]*State{STATE_S01: s01, STATE_S02: s02}}`
- Update `IsInState()` calls to use int constants: `rt.IsInState(STATE_S01)`

**Tests that will pass**:
- `TestSCXML403a` - Nested states (s0 contains s01, s02) (after test update)

**Architectural decision revealed**:
- Need to track "configuration" (set of active states in hierarchy)
- Entering compound state must enter its initial child (by StateID)
- Exiting compound state must exit all children first
- Transition target resolution uses int StateIDs directly

**Complexity**: Medium

---

### Step 5: Implement Proper Entry/Exit for Hierarchical States
**Goal**: Handle entry/exit order for nested states

**What to implement**:
- Compute Least Common Ancestor (LCA) for transitions (using StateID comparisons)
- Exit states from current up to (but not including) LCA
- Enter states from LCA down to target
- Entry order: parent → child
- Exit order: child → parent
- Update `OnEntry` and `OnExit` signatures:
  - `func(ctx context.Context, event Event, from, to StateID, ext any)`

**What to update in tests**:
- Update OnEntry/OnExit function signatures to match new API
- Use int StateIDs in callbacks: `OnEntry: func(ctx, event, from, to, ext) { /* from and to are StateID (int) */ }`

**Tests that will pass**:
- `TestSCXML375` - OnEntry execution order (after test update)
- `TestSCXML377` - OnExit execution order (after test update)
- `TestSCXML407` - OnExit handlers with guards checking state (after test update)

**Architectural decision revealed**:
- Need to maintain "configuration" (active state set)
- Compound state entry must recursively enter initial children
- Need to handle "extended state" (the `ext any` parameter)
- Transition actions execute between exit and entry

**Complexity**: Complex

---

### Step 6: Implement Transition Selection with Hierarchy
**Goal**: Child transitions take precedence over parent transitions

**What to implement**:
- Update `pickTransition()` to search from innermost state outward
- Check current state's transitions first
- If no match, check parent's transitions, then grandparent, etc.
- Guards can disable transitions, causing search to continue
- Use int StateID and EventID for all comparisons

**What to update in tests**:
- Ensure transition definitions use int EventIDs
- Ensure guard functions use int StateIDs for comparisons

**Tests that will pass**:
- `TestSCXML403a` (fully) - Child transitions override parent transitions (after test update)

**Architectural decision revealed**:
- Need to handle guards properly in transition selection
- Failed guard means "try next transition in document order"
- If all transitions in current state fail, try parent state
- This is "optimal enablement" in SCXML terms

**Complexity**: Medium

---

## Phase 3: Advanced Transition Features (Steps 7-9)

### Step 7: Implement Internal Transitions
**Goal**: Support transitions with `Target: 0` (internal transitions)

**What to implement**:
- Detect when `Transition.Target == 0` (internal transition - no target state)
- Internal transitions execute action but don't exit/enter states
- Update transition execution to skip exit/entry for internal transitions
- Internal transitions still respect guards

**What to update in tests**:
- Change `Target: nil` or `Target: ""` to `Target: 0` in transition definitions
- Tests: `TestInternalTransitionDoesTransition`, `TestInternalTransitionExecsActionOnly`, `TestInternalPicksFirstEnabledTransition`

**Tests that will pass**:
- `TestInternalTransitionDoesTransition` - Basic internal transition (after test update)
- `TestInternalTransitionExecsActionOnly` - No entry/exit on internal (after test update)
- `TestInternalPicksFirstEnabledTransition` - Guard evaluation (after test update)

**Architectural decision revealed**:
- Internal transitions are useful for self-loops without state change
- Action still executes even though state doesn't change
- Guards work the same for internal and external transitions
- `Target: 0` is the sentinel value for "no target"

**Complexity**: Simple

---

### Step 8: Implement Eventless (Immediate) Transitions
**Goal**: Support transitions with EventID = 0 (immediate transitions)

**What to implement**:
- Define special constant: `const NO_EVENT EventID = 0` (or just use 0 directly)
- Eventless transitions fire immediately when state is entered
- After entering a state, check for eventless transitions before processing queue
- Eventless transitions take precedence over queued events
- Update event loop to check for eventless transitions after each state entry

**What to update in tests**:
- Change `Event: ""` to `Event: NO_EVENT` or `Event: 0` in transition definitions
- Tests: `TestSCXML355`, `TestSCXML419`, `TestSCXML407`

**Tests that will pass**:
- `TestSCXML355` - Immediate transition with empty event (after test update)
- `TestSCXML419` - Eventless transitions take precedence (after test update)
- `TestSCXML407` (if not already passing) - Immediate transitions with guards (after test update)

**Architectural decision revealed**:
- Need "microstep" vs "macrostep" concept
- Microstep: process eventless transitions until stable
- Macrostep: process one event from queue
- Prevents infinite loops: limit microsteps or detect cycles

**Complexity**: Medium

---

### Step 9: Implement Transition Actions and Guards
**Goal**: Ensure actions and guards work correctly in all contexts with int-based types

**What to implement**:
- Transition actions execute between exit and entry
- Guards are evaluated before any state changes
- Failed guard means transition doesn't fire
- Guard signature: `func(ctx context.Context, event Event, from, to StateID, ext any) bool`
- Action signature: `func(ctx context.Context, event Event, from, to StateID, ext any)`

**What to update in tests**:
- Update guard/action function signatures to use `Event` struct and int `StateID`
- Guards check int StateIDs: `Guard: func(ctx, event, from, to, ext) bool { return from == STATE_A }`
- Actions use int StateIDs: `Action: func(ctx, event, from, to, ext) { /* to is StateID (int) */ }`

**Tests that will pass**:
- `TestSCXML503` - Internal transition actions, guards checking counters (after test update)
- `TestSCXML505` - Self-transitions with actions (after test update)
- `TestSCXML506` - Compound state self-transitions (after test update)

**Architectural decision revealed**:
- Actions can modify extended state
- Guards can read extended state
- Self-transitions (target == source) still exit and re-enter
- Need to pass extended state through all callbacks

**Complexity**: Simple

---

## Phase 4: Advanced Features (Steps 10-12)

### Step 10: Implement Initial Transition Actions
**Goal**: Support executable content in initial transitions

**What to implement**:
- Add `InitialTransition *Transition` field to State
- Initial transition can have actions (but no event/guard)
- Execute initial transition action after parent entry, before child entry
- Order: parent OnEntry → initial transition action → child OnEntry

**What to update in tests**:
- Define initial transitions with actions: `InitialTransition: &Transition{Target: STATE_CHILD, Action: func(...) {...}}`
- Test: `TestSCXML412`

**Tests that will pass**:
- `TestSCXML412` - Initial transition execution order (after test update)

**Architectural decision revealed**:
- Initial transitions are special: no event, no guard, just action
- Affects entry order for compound states
- Useful for initialization logic

**Complexity**: Medium

---

### Step 11: Implement Event Matching Priority
**Goal**: Ensure correct event matching order with int-based EventIDs

**What to implement**:
- Document order: first transition in list wins
- Specific EventIDs beat `ANY_EVENT` wildcard
- Eventless transitions (`Event: 0`) beat event-driven ones
- Failed guards cause fallthrough to next transition

**What to update in tests**:
- Ensure transition order in slice matches expected priority
- Tests: `TestSCXML396`, `TestSCXML421`

**Tests that will pass**:
- `TestSCXML396` - First matching transition wins (after test update)
- `TestSCXML421` - Event queue ordering (internal vs external) (after test update)

**Architectural decision revealed**:
- May need separate internal and external event queues
- Internal events (raised) have priority over external (sent)
- For now, single queue is sufficient (tests don't distinguish)

**Complexity**: Simple

---

### Step 12: Implement Final States (Optional)
**Goal**: Support final states that stop the machine

**What to implement**:
- Add `Final bool` field to State (already exists)
- When entering final state, machine stops processing
- Generate "done.state.id" events (for parent compound states)
- Update Runtime to detect final state and stop

**What to update in tests**: None in current non-skipped set

**Tests that will pass**: None in current non-skipped set

**Architectural decision revealed**:
- Final states enable hierarchical completion
- Parent state can react to child completion
- Needed for more complex SCXML tests (currently skipped)
- Done events would use special EventID constants (e.g., `DONE_STATE_X`)

**Complexity**: Medium

---

## Phase 5: Parallel States (Steps 13-15)

### Step 13: Design Parallel State Architecture
**Goal**: Plan how to implement parallel states with goroutines using int-based types

**What to implement**:
- Add `Type` field to State: `const (ATOMIC, COMPOUND, PARALLEL StateType = iota)`
- Parallel states have multiple active children simultaneously
- Each parallel region runs in its own goroutine
- Need synchronization for transitions that cross regions
- Event addressing (event.address) enables targeted communication with specific regions

**What to update in tests**: None yet (all parallel tests are skipped)

**Tests that will pass**: None yet (all parallel tests are skipped)

**Architectural decision revealed**:
- Parallel states need separate event queues per region
- Broadcast events (address == 0) go to all regions
- Targeted events (address != 0) go to specific region
- Transitions can target multiple states (one per region)
- Need to handle race conditions without locks (channel-based sync)

**Complexity**: Complex

---

### Step 14: Implement Parallel State Entry/Exit
**Goal**: Enter/exit all parallel regions simultaneously

**What to implement**:
- When entering parallel state, spawn goroutine for each child region
- Each region has its own event loop
- Exit waits for all regions to complete
- Entry order: parent → all children (concurrently)
- Exit order: all children (wait for all) → parent
- Use event.address to route events to specific regions

**What to update in tests**: 
- Basic parallel tests (once un-skipped) will need int StateID/EventID updates

**Tests that will pass**: 
- Basic parallel tests (once un-skipped and updated)

**Architectural decision revealed**:
- Need WaitGroup or similar for synchronization
- Errors from any region should propagate
- Context cancellation should stop all regions
- Event addressing simplifies parallel communication

**Complexity**: Complex

---

### Step 15: Implement Parallel Event Broadcasting
**Goal**: Events sent to parallel state reach all regions (or specific region via address)

**What to implement**:
- Broadcast events (address == 0) to all active parallel regions
- Targeted events (address != 0) to specific region only
- Each region processes event independently
- Collect results from all regions
- Handle case where multiple regions transition

**What to update in tests**:
- Advanced parallel tests (once un-skipped) will need int StateID/EventID updates
- Tests will use event.address for targeted communication

**Tests that will pass**:
- Advanced parallel tests (once un-skipped and updated)

**Architectural decision revealed**:
- Need to handle conflicting transitions from different regions
- May need to serialize certain operations
- Configuration becomes a set of states (one per region)
- Event addressing provides clean parallel communication model

**Complexity**: Complex

---

## Summary of Test Coverage by Phase

### Phase 1 (Steps 1-3): 5 tests (after updating to int-based API)
- TestSCXML144, 147, 148, 149, 158

### Phase 2 (Steps 4-6): 4 tests (after updating to int-based API)
- TestSCXML375, 377, 403a, 407

### Phase 3 (Steps 7-9): 6 tests (after updating to int-based API)
- TestInternalTransitionDoesTransition
- TestInternalTransitionExecsActionOnly
- TestInternalPicksFirstEnabledTransition
- TestSCXML355, 419, 503, 505, 506

### Phase 4 (Steps 10-12): 3 tests (after updating to int-based API)
- TestSCXML396, 412, 421

### Phase 5 (Steps 13-15): 0 tests (all parallel tests skipped, will need updates when un-skipped)
- Enables future work on 15+ parallel tests

---

## Implementation Notes

### Key Architectural Decisions

1. **Int-based Types**: `StateID int`, `EventID int` - no string conversion
2. **Event Addressing**: `Event.address StateID` field for routing (0 = broadcast, non-zero = targeted)
3. **Event Queue**: Channel-based, FIFO, processes `Event` structs
4. **State Hierarchy**: Parent pointers + Children map (keyed by int StateID)
5. **Configuration**: Track active state(s) - single StateID for compound, set for parallel
6. **Transition Selection**: Innermost state first, document order, guards can disable
7. **Entry/Exit Order**: Compute LCA, exit child→parent, enter parent→child
8. **Eventless Transitions**: `Event: 0` - microstep loop after each state entry
9. **Internal Transitions**: `Target: 0` - skip exit/entry, still execute action
10. **Wildcard Events**: `ANY_EVENT` constant (e.g., -1) - matches any event
11. **Parallel States**: Goroutine per region, channel-based sync, event addressing for communication

### Type Signatures (New API)

```go
type StateID int
type EventID int
type StateType int

const (
    ATOMIC StateType = iota
    COMPOUND
    PARALLEL
)

const (
    NO_EVENT EventID = 0  // Eventless transition
    ANY_EVENT EventID = -1  // Wildcard event
)

type Event struct {
    ID      EventID
    Data    any
    address StateID  // 0 = broadcast, non-zero = targeted
}

type State struct {
    ID                StateID
    Type              StateType
    Parent            *State
    Children          map[StateID]*State
    Initial           StateID  // int, not pointer
    InitialTransition *Transition
    Transitions       []*Transition
    OnEntry           func(ctx context.Context, event Event, from, to StateID, ext any)
    OnExit            func(ctx context.Context, event Event, from, to StateID, ext any)
    Final             bool
}

type Transition struct {
    Event  EventID  // 0 for eventless, ANY_EVENT for wildcard
    Target StateID  // 0 for internal transition
    Guard  func(ctx context.Context, event Event, from, to StateID, ext any) bool
    Action func(ctx context.Context, event Event, from, to StateID, ext any)
}

type Runtime struct {
    machine *Machine
    ext     any  // extended state
    // ... internal fields
}

func NewRuntime(machine *Machine, ext any) *Runtime
func (rt *Runtime) Start(ctx context.Context)
func (rt *Runtime) Stop()
func (rt *Runtime) SendEvent(ctx context.Context, event Event)  // Takes Event struct, not string
func (rt *Runtime) IsInState(stateID StateID) bool
```

### Testing Strategy

1. **Update tests incrementally** as you implement each step
2. Define constants for StateIDs and EventIDs at the top of each test file
3. Replace string-based API calls with int-based equivalents
4. Run tests after each step to verify progress: `go test -v -run TestSCXML144`
5. Keep existing tests passing as you add features
6. Un-skip tests incrementally as features are implemented

### Test Update Pattern

**Before (old string-based API)**:
```go
rt.SendEvent(ctx, "foo")
rt.IsInState("stateA")
Transition{Event: "*", Target: "stateB"}
```

**After (new int-based API)**:
```go
const (
    FOO_EVENT EventID = 1
    STATE_A StateID = 1
    STATE_B StateID = 2
)

rt.SendEvent(ctx, Event{ID: FOO_EVENT})
rt.IsInState(STATE_A)
Transition{Event: ANY_EVENT, Target: STATE_B}
```

### Future Work (Skipped Tests)

- **Datamodel** (74 tests): Variables, expressions, assignments - will use int-based IDs
- **Invoke/Send** (61 tests): External service invocation, delayed events - will use Event struct
- **History** (1 test): History states (shallow/deep) - will use int StateIDs
- **Error Events** (8 tests): error.execution, error.communication - will use special EventID constants
- **Done Events** (2 tests): done.state.id, done.invoke.id - will use special EventID constants
- **Other** (24 tests): Various advanced features - all will use int-based API

---

## Recommended Implementation Order

1. **Start with Phase 1** (Steps 1-3) to get basic event-driven FSM working
   - Implement Runtime with Event struct and int-based types
   - Update 5 tests to use new API
   
2. **Move to Phase 2** (Steps 4-6) to add hierarchy - this is the biggest architectural change
   - Implement parent-child relationships with int StateIDs
   - Update 4 tests to use new API
   
3. **Complete Phase 3** (Steps 7-9) to handle edge cases and advanced transitions
   - Implement internal/eventless transitions with int-based types
   - Update 6 tests to use new API
   
4. **Add Phase 4** (Steps 10-12) for remaining non-parallel features
   - Implement initial transitions and final states
   - Update 3 tests to use new API
   
5. **Defer Phase 5** (Steps 13-15) until basic functionality is solid
   - Implement parallel states with goroutines and event addressing
   - Update parallel tests when un-skipped

Each step should take 30-60 minutes of focused implementation. Total estimated time: 12-20 hours for Phases 1-4.

---

## Success Criteria

- All 17 non-skipped tests pass (after updating to new API)
- Code remains under 500 lines (currently 254)
- No locks used (channel-based synchronization only)
- Clear separation between sync FSM (compound) and async (parallel)
- Clean int-based API with no string conversion overhead
- Event addressing enables clean parallel state communication
- Easy to extend for future features (datamodel, invoke, history)

---

## Key Differences from Old Plan

1. **No String Conversion**: Removed all steps involving string-to-int conversion
2. **Event Addressing**: Added event.address field for routing (especially important for parallel states)
3. **Test Updates**: Each step now includes "What to update in tests" section
4. **Clean API**: Focus on designing the new API cleanly, not maintaining backward compatibility
5. **Int-based Throughout**: StateID and EventID are int everywhere, no type conversions
6. **Special Constants**: Use constants like `NO_EVENT`, `ANY_EVENT` instead of special strings
7. **Sentinel Values**: Use `0` for "no target" (internal transitions) and "no event" (eventless transitions)
