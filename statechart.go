package statechartx

import (
        "context"
        "errors"
        "fmt"
        "sync"
        "time"
)

type StateID int
type EventID int

const (
        NO_EVENT  EventID = 0  // Eventless/immediate transition
        ANY_EVENT EventID = -1 // Wildcard event
)

const (
        MAX_MICROSTEPS = 100 // Maximum microstep iterations to prevent infinite loops
)

// Timeout constants for parallel state operations
const (
        DefaultEntryTimeout  = 5 * time.Second
        DefaultExitTimeout   = 5 * time.Second
        DefaultSendTimeout   = 100 * time.Millisecond
        DefaultActionTimeout = 5 * time.Second
)

type Event struct {
        ID      EventID
        Data    any
        Address StateID // 0 = broadcast, non-zero = targeted (for parallel states)
}

type Action func(ctx context.Context, evt *Event, from StateID, to StateID) error
type Guard func(ctx context.Context, evt *Event, from StateID, to StateID) (bool, error)

// ---

// HistoryType defines the type of history state
type HistoryType int

const (
        HistoryNone    HistoryType = iota // Not a history state
        HistoryShallow                    // Restore direct child only
        HistoryDeep                       // Restore entire hierarchy
)

type State struct {
        ID            StateID
        Transitions   []*Transition
        EntryAction   Action
        ExitAction    Action
        InitialAction Action  // Action to execute when entering initial child (Step 10)
        IsFinal       bool    // True if this is a final state (Step 12)
        Final         bool    // Deprecated: use IsFinal instead
        Parent        *State
        Children      map[StateID]*State
        Initial       StateID // Initial child state for compound states
        IsParallel    bool    // True if this is a parallel state (Step 13)
        
        // History state support
        IsHistoryState  bool        // True if this is a history pseudo-state
        HistoryType     HistoryType // Type of history (shallow or deep)
        HistoryDefault  StateID     // Default state if no history exists
        FinalStateData  any         // Data to include in done event
}

type CompoundState struct {
        State
        Initial  StateID
        Children []*State
}

type Transition struct {
        Event  EventID
        Source *State
        Target StateID // 0 --> internal transition
        Guard  Guard   // nil --> always true
        Action Action  // nil --> do nothing
}

// Machine is a CompoundState with helper functions for chart evaluation.
type Machine struct {
        CompoundState
        states  map[StateID]*State
        current *State
}

// Runtime wraps a Machine and provides event queue processing
type Runtime struct {
        machine    *Machine
        ext        any // extended state
        eventQueue chan Event
        ctx        context.Context
        cancel     context.CancelFunc
        wg         sync.WaitGroup
        mu         sync.RWMutex
        current    StateID
        
        // Parallel state support
        parallelRegions map[StateID]*parallelRegion
        regionMu        sync.RWMutex
        
        // History state support
        history       map[StateID]StateID   // stateID → last active child (shallow)
        historyMu     sync.RWMutex
        deepHistory   map[StateID][]StateID // stateID → full state path (deep)
        deepHistoryMu sync.RWMutex
        
        // Done event support
        doneEventsPending map[StateID]bool // Track pending done events
        doneEventsMu      sync.RWMutex
}

// parallelRegion represents a single region in a parallel state
type parallelRegion struct {
        stateID      StateID
        events       chan Event
        done         chan struct{} // signal to exit
        finished     chan struct{} // signal that exit is complete
        ctx          context.Context
        cancel       context.CancelFunc
        runtime      *Runtime // reference to parent runtime
        currentState StateID
        mu           sync.RWMutex
}

//
// Public API
//

func (s *State) OnEntry(action Action) {
        s.EntryAction = action
}

func (s *State) OnExit(action Action) {
        s.ExitAction = action
}

func NewMachine(root *State) (*Machine, error) {
        if root == nil {
                return nil, errors.New("no root state provided")
        }
        
        m := &Machine{
                states: map[StateID]*State{},
        }

        // Build state lookup table recursively and establish parent-child relationships
        var buildLUT func(*State, *State) error
        buildLUT = func(s *State, parent *State) error {
                if s == nil {
                        return nil
                }
                
                // Set parent relationship
                s.Parent = parent
                
                // Check for duplicate state IDs
                if _, exists := m.states[s.ID]; exists {
                        return errors.New("duplicate state ID")
                }
                m.states[s.ID] = s
                
                // Process children
                for _, child := range s.Children {
                        if err := buildLUT(child, s); err != nil {
                                return err
                        }
                }
                
                // Validate transitions have a source set
                for _, t := range s.Transitions {
                        if t == nil {
                                continue
                        }
                        if t.Source == nil {
                                t.Source = s
                        }
                }
                
                return nil
        }
        
        if err := buildLUT(root, nil); err != nil {
                return nil, err
        }
        
        // Set initial state - recursively find the deepest initial state
        initialStateID := m.findDeepestInitial(root.ID)
        
        m.Initial = initialStateID
        m.current = m.states[initialStateID]
        if m.current == nil {
                return nil, errors.New("initial state not found")
        }

        return m, nil
}

// findDeepestInitial recursively finds the deepest initial state in the hierarchy
func (m *Machine) findDeepestInitial(stateID StateID) StateID {
        state := m.states[stateID]
        if state == nil {
                return stateID
        }
        
        // Parallel states don't have a single initial state
        if state.IsParallel {
                return stateID
        }
        
        // If state has an initial child, recurse into it
        if state.Initial != 0 {
                return m.findDeepestInitial(state.Initial)
        }
        
        // If state has children but no explicit initial, use first child
        if len(state.Children) > 0 {
                for childID := range state.Children {
                        return m.findDeepestInitial(childID)
                }
        }
        
        // Atomic state - return it
        return stateID
}

// NewRuntime creates a new Runtime for the given machine
func NewRuntime(machine *Machine, ext any) *Runtime {
        return &Runtime{
                machine:           machine,
                ext:               ext,
                eventQueue:        make(chan Event, 100), // buffered channel for event queue
                current:           machine.Initial,
                parallelRegions:   make(map[StateID]*parallelRegion),
                history:           make(map[StateID]StateID),
                deepHistory:       make(map[StateID][]StateID),
                doneEventsPending: make(map[StateID]bool),
        }
}

// Start begins the event processing loop
func (rt *Runtime) Start(ctx context.Context) error {
        if rt.ctx != nil {
                return errors.New("runtime already started")
        }

        rt.ctx, rt.cancel = context.WithCancel(ctx)

        // Enter initial state hierarchy (from root to initial state)
        rt.mu.Lock()
        if err := rt.enterInitialState(rt.ctx); err != nil {
                rt.mu.Unlock()
                return err
        }
        rt.mu.Unlock()

        // Start event processing loop
        rt.wg.Add(1)
        go rt.eventLoop()

        return nil
}

// enterInitialState enters the initial state hierarchy from root to deepest initial state
func (rt *Runtime) enterInitialState(ctx context.Context) error {
        // Build path from root to initial state
        var path []StateID
        current := rt.machine.states[rt.current]
        for current != nil {
                path = append([]StateID{current.ID}, path...) // prepend to reverse order
                current = current.Parent
        }

        // Enter states in order (parent to child)
        // Execute InitialAction after parent entry but before child entry (Step 10)
        for i, stateID := range path {
                state := rt.machine.states[stateID]
                if state == nil {
                        continue
                }
                
                // Check if this is a parallel state
                if state.IsParallel {
                        // Enter parallel state - spawn goroutines for each region
                        if err := rt.enterParallelState(ctx, state); err != nil {
                                return err
                        }
                        // Don't continue down the path - parallel regions handle their own entry
                        return nil
                }
                
                // Execute entry action
                if state.EntryAction != nil {
                        if err := state.EntryAction(ctx, nil, 0, rt.current); err != nil {
                                return err
                        }
                }
                
                // Execute initial action if this state has children and we're entering them
                // InitialAction runs after parent entry but before child entry
                if i < len(path)-1 && state.InitialAction != nil {
                        nextStateID := path[i+1]
                        if err := state.InitialAction(ctx, nil, stateID, nextStateID); err != nil {
                                return err
                        }
                }
        }
        
        // Check if we entered a final state (Step 12)
        rt.checkFinalState(ctx)
        
        // Process eventless transitions after entering initial state (Step 8)
        rt.processMicrosteps(ctx)
        
        return nil
}

// enterParallelState enters a parallel state by spawning goroutines for each region
func (rt *Runtime) enterParallelState(ctx context.Context, state *State) error {
        if !state.IsParallel {
                return errors.New("not a parallel state")
        }
        
        if len(state.Children) == 0 {
                return errors.New("parallel state has no children")
        }
        
        // Execute parent entry action
        if state.EntryAction != nil {
                if err := state.EntryAction(ctx, nil, 0, state.ID); err != nil {
                        return err
                }
        }
        
        // Create context with timeout for entry
        entryCtx, entryCancel := context.WithTimeout(ctx, DefaultEntryTimeout)
        defer entryCancel()
        
        // Spawn goroutine for each child region
        errChan := make(chan error, len(state.Children))
        startedChan := make(chan StateID, len(state.Children))
        
        rt.regionMu.Lock()
        for childID, child := range state.Children {
                // Create region context
                regionCtx, regionCancel := context.WithCancel(rt.ctx)
                
                region := &parallelRegion{
                        stateID:      childID,
                        events:       make(chan Event, 10), // buffered channel
                        done:         make(chan struct{}),
                        finished:     make(chan struct{}),
                        ctx:          regionCtx,
                        cancel:       regionCancel,
                        runtime:      rt,
                        currentState: rt.machine.findDeepestInitial(childID),
                }
                
                rt.parallelRegions[childID] = region
                
                go func(r *parallelRegion, s *State) {
                        defer func() {
                                close(r.finished) // Signal that goroutine has exited
                                if rec := recover(); rec != nil {
                                        errChan <- fmt.Errorf("panic in region %d: %v", r.stateID, rec)
                                }
                        }()

                        // Signal that region has started
                        startedChan <- r.stateID

                        if err := r.run(s); err != nil {
                                errChan <- err
                        }
                }(region, child)
        }
        rt.regionMu.Unlock()
        
        // Wait for all regions to start with timeout
        numStarted := 0
        for numStarted < len(state.Children) {
                select {
                case <-startedChan:
                        numStarted++
                case <-entryCtx.Done():
                        // Timeout - cleanup and return error
                        rt.cleanupParallelRegions(state.ID)
                        return errors.New("parallel state entry timeout")
                case err := <-errChan:
                        // Error during entry - cleanup and return
                        rt.cleanupParallelRegions(state.ID)
                        return err
                }
        }
        
        return nil
}

// exitParallelState exits a parallel state by stopping all region goroutines
func (rt *Runtime) exitParallelState(ctx context.Context, state *State) error {
        if !state.IsParallel {
                return errors.New("not a parallel state")
        }
        
        // Create context with timeout for exit
        exitCtx, exitCancel := context.WithTimeout(ctx, DefaultExitTimeout)
        defer exitCancel()
        
        // Signal all regions to stop
        rt.regionMu.Lock()
        for childID := range state.Children {
                if region, exists := rt.parallelRegions[childID]; exists {
                        close(region.done)
                }
        }
        rt.regionMu.Unlock()
        
        // Wait for all regions to exit with timeout
        allDone := make(chan struct{})
        go func() {
                rt.regionMu.RLock()
                for childID := range state.Children {
                        if region, exists := rt.parallelRegions[childID]; exists {
                                <-region.finished // Wait for goroutine to finish
                        }
                }
                rt.regionMu.RUnlock()
                close(allDone)
        }()
        
        select {
        case <-allDone:
                // All regions exited successfully
        case <-exitCtx.Done():
                // Timeout - force cleanup
                rt.cleanupParallelRegions(state.ID)
                return errors.New("parallel state exit timeout")
        }
        
        // Cleanup regions
        rt.cleanupParallelRegions(state.ID)
        
        // Execute parent exit action
        if state.ExitAction != nil {
                if err := state.ExitAction(ctx, nil, state.ID, 0); err != nil {
                        return err
                }
        }
        
        return nil
}

// cleanupParallelRegions cleans up all regions for a parallel state
func (rt *Runtime) cleanupParallelRegions(stateID StateID) {
        rt.regionMu.Lock()
        defer rt.regionMu.Unlock()
        
        state := rt.machine.states[stateID]
        if state == nil || !state.IsParallel {
                return
        }
        
        for childID := range state.Children {
                if region, exists := rt.parallelRegions[childID]; exists {
                        // Cancel context
                        region.cancel()
                        // Close event channel
                        close(region.events)
                        // Remove from map
                        delete(rt.parallelRegions, childID)
                }
        }
}

// run is the main event loop for a parallel region
func (r *parallelRegion) run(state *State) error {
        // If this region is a parallel state, spawn child regions first
        if state.IsParallel {
                if err := r.runtime.enterParallelState(r.ctx, state); err != nil {
                        return err
                }
                // KEY FIX: Don't return - continue to event loop to monitor exit signals
        }

        // Enter the initial state hierarchy (from region root to deepest initial)
        // This ensures entry actions are executed and done events are generated
        if !state.IsParallel {
                r.enterInitialHierarchy(r.ctx, state)
        }

        // Always run event loop to monitor done channel
        for {
                select {
                case <-r.done:
                        // Exit current state before returning
                        r.exitCurrentState(r.ctx)
                        return nil
                case <-r.ctx.Done():
                        // Exit current state before returning
                        r.exitCurrentState(r.ctx)
                        return r.ctx.Err()
                case event := <-r.events:
                        // Parallel states delegate event processing to children
                        if !state.IsParallel {
                                r.processEvent(event, state)
                        }
                }
        }
}

// processEvent processes a single event in a parallel region
func (r *parallelRegion) processEvent(event Event, state *State) {
        r.mu.Lock()

        // Find matching transition
        currentState := r.runtime.machine.states[r.currentState]
        if currentState == nil {
                r.mu.Unlock()
                return
        }

        transition := r.runtime.pickTransitionHierarchical(currentState, event)
        if transition == nil {
                r.mu.Unlock()
                return
        }

        // Internal transition
        if transition.Target == 0 {
                if transition.Action != nil {
                        transition.Action(r.ctx, &event, r.currentState, r.currentState)
                }
                r.mu.Unlock()
                return
        }
        
        // External transition
        from := r.currentState
        to := transition.Target
        
        // Check if target is a history state
        targetState := r.runtime.machine.states[to]
        if targetState != nil && targetState.IsHistoryState {
                // Restore history instead of entering target directly
                restoredState, err := r.runtime.restoreHistory(r.ctx, targetState, &event, from)
                if err != nil {
                        // History restoration failed, use default or skip
                        if targetState.HistoryDefault != 0 {
                                to = targetState.HistoryDefault
                        } else {
                                r.mu.Unlock()
                                return // Cannot restore history and no default
                        }
                } else {
                        to = restoredState
                }
        }
        
        // Compute LCA
        lca := r.runtime.computeLCA(from, to)
        
        // Exit states
        r.runtime.exitToLCA(r.ctx, &event, from, to, lca)
        
        // Execute transition action
        if transition.Action != nil {
                transition.Action(r.ctx, &event, from, to)
        }
        
        // Enter states
        r.runtime.enterFromLCA(r.ctx, &event, from, to, lca)
        
        // Update current state
        r.currentState = to
        
        // Check if we entered a final state in this region
        newState := r.runtime.machine.states[r.currentState]
        var doneParent *State
        var doneParallelParent *State
        if newState != nil && (newState.IsFinal || newState.Final) {
                // Save the states to generate done events for after releasing lock
                doneParent = newState.Parent
                regionState := r.runtime.machine.states[r.stateID]
                if regionState != nil && regionState.Parent != nil && regionState.Parent.IsParallel {
                        doneParallelParent = regionState.Parent
                }
        }

        r.mu.Unlock()

        // Generate done events after releasing the lock to avoid deadlock
        // Get the parallel state ID (parent of region state)
        regionState := r.runtime.machine.states[r.stateID]
        var parallelStateID StateID
        if regionState != nil && regionState.Parent != nil {
                parallelStateID = regionState.Parent.ID
        }

        if doneParent != nil {
                r.runtime.generateDoneEvent(r.ctx, doneParent, newState, parallelStateID)
        }
        if doneParallelParent != nil {
                r.runtime.generateDoneEvent(r.ctx, doneParallelParent, regionState, parallelStateID)
        }
}

// enterInitialHierarchy enters the initial state hierarchy for a parallel region
func (r *parallelRegion) enterInitialHierarchy(ctx context.Context, regionRoot *State) {
        // Build path from region root to current state (deepest initial)
        var path []StateID
        current := r.runtime.machine.states[r.currentState]
        for current != nil {
                path = append([]StateID{current.ID}, path...) // prepend
                if current == regionRoot {
                        break // Include region root in path
                }
                current = current.Parent
        }

        // Enter states in order (parent to child)
        for i, stateID := range path {
                state := r.runtime.machine.states[stateID]
                if state == nil {
                        continue
                }

                // Execute entry action
                if state.EntryAction != nil {
                        state.EntryAction(ctx, nil, 0, r.currentState)
                }

                // Execute initial action if moving to next state
                if i < len(path)-1 && state.InitialAction != nil {
                        nextStateID := path[i+1]
                        state.InitialAction(ctx, nil, stateID, nextStateID)
                }

                // Check if this state is final and should generate done event
                if state.IsFinal || state.Final {
                        // Get the parallel state ID (parent of region state)
                        regionState := r.runtime.machine.states[r.stateID]
                        var parallelStateID StateID
                        if regionState != nil && regionState.Parent != nil {
                                parallelStateID = regionState.Parent.ID
                        }

                        // First generate done event for the region compound state (if it has one)
                        if state.Parent != nil {
                                r.runtime.generateDoneEvent(ctx, state.Parent, state, parallelStateID)
                        }

                        // Then check if parent parallel state should emit done event
                        if regionState != nil && regionState.Parent != nil && regionState.Parent.IsParallel {
                                r.runtime.generateDoneEvent(ctx, regionState.Parent, regionState, parallelStateID)
                        }
                }
        }
}

// exitCurrentState executes exit actions for the current state in the region
func (r *parallelRegion) exitCurrentState(ctx context.Context) {
        regionState := r.runtime.machine.states[r.stateID]

        // If this region is a parallel state, exit child regions first
        if regionState != nil && regionState.IsParallel {
                r.runtime.exitParallelState(ctx, regionState)

                // Execute region state's own exit action after children exit
                if regionState.ExitAction != nil {
                        regionState.ExitAction(ctx, nil, regionState.ID, 0)
                }
                return
        }

        r.mu.Lock()
        defer r.mu.Unlock()

        // Exit from current state up to and including region root
        current := r.runtime.machine.states[r.currentState]

        for current != nil {
                if current.ExitAction != nil {
                        current.ExitAction(ctx, nil, current.ID, 0)
                }

                // Stop after executing region state's exit action
                if current == regionState {
                        break
                }

                current = current.Parent
        }
}

// Stop stops the event processing loop
func (rt *Runtime) Stop() error {
        // Cancel context to signal all goroutines to exit
        if rt.cancel != nil {
                rt.cancel()
        }
        rt.wg.Wait()

        // After all goroutines have exited, execute top-level state's exit action if it's parallel
        rt.mu.Lock()
        defer rt.mu.Unlock()

        currentState := rt.machine.states[rt.current]
        if currentState != nil && currentState.IsParallel && currentState.ExitAction != nil {
                // Execute the parallel state's exit action (child regions already exited)
                ctx := context.Background()
                currentState.ExitAction(ctx, nil, currentState.ID, 0)
        }

        return nil
}

// SendEvent queues an event for processing
func (rt *Runtime) SendEvent(ctx context.Context, event Event) error {
        // Check if we're in a parallel state and need to route the event
        rt.regionMu.RLock()
        hasRegions := len(rt.parallelRegions) > 0
        rt.regionMu.RUnlock()
        
        if hasRegions {
                return rt.sendEventToRegions(ctx, event)
        }
        
        // Normal event queue
        select {
        case rt.eventQueue <- event:
                return nil
        case <-ctx.Done():
                return ctx.Err()
        case <-rt.ctx.Done():
                return rt.ctx.Err()
        }
}

// sendEventToRegions routes events to parallel regions based on address
func (rt *Runtime) sendEventToRegions(ctx context.Context, event Event) error {
        sendCtx, cancel := context.WithTimeout(ctx, DefaultSendTimeout)
        defer cancel()
        
        rt.regionMu.RLock()
        defer rt.regionMu.RUnlock()
        
        if event.Address == 0 {
                // Broadcast to all regions
                for _, region := range rt.parallelRegions {
                        select {
                        case region.events <- event:
                                // Event sent successfully
                        case <-sendCtx.Done():
                                return errors.New("broadcast timeout")
                        case <-region.ctx.Done():
                                // Region is shutting down, skip
                                continue
                        }
                }
                return nil
        }
        
        // Targeted delivery
        region, exists := rt.parallelRegions[event.Address]
        if !exists {
                return fmt.Errorf("region %d not found", event.Address)
        }
        
        select {
        case region.events <- event:
                return nil
        case <-sendCtx.Done():
                return errors.New("send timeout")
        case <-region.ctx.Done():
                return errors.New("region shutting down")
        }
}

// IsInState checks if the runtime is currently in the given state or any of its ancestors
func (rt *Runtime) IsInState(stateID StateID) bool {
        rt.mu.RLock()
        defer rt.mu.RUnlock()
        
        // Check if current state or any ancestor matches
        current := rt.machine.states[rt.current]
        for current != nil {
                if current.ID == stateID {
                        return true
                }
                current = current.Parent
        }
        
        // Check parallel regions
        rt.regionMu.RLock()
        defer rt.regionMu.RUnlock()
        for _, region := range rt.parallelRegions {
                region.mu.RLock()
                regionCurrent := rt.machine.states[region.currentState]
                region.mu.RUnlock()
                
                for regionCurrent != nil {
                        if regionCurrent.ID == stateID {
                                return true
                        }
                        regionCurrent = regionCurrent.Parent
                }
        }
        
        return false
}

// getActiveStates returns all active states in the hierarchy (from root to current)
func (rt *Runtime) getActiveStates() []StateID {
        rt.mu.RLock()
        defer rt.mu.RUnlock()
        
        var states []StateID
        current := rt.machine.states[rt.current]
        for current != nil {
                states = append([]StateID{current.ID}, states...) // prepend to get root-to-leaf order
                current = current.Parent
        }
        return states
}

// eventLoop processes events from the queue
func (rt *Runtime) eventLoop() {
        defer rt.wg.Done()

        for {
                select {
                case <-rt.ctx.Done():
                        return
                case event := <-rt.eventQueue:
                        rt.processEvent(event)
                }
        }
}

// processEvent handles a single event (macrostep)
func (rt *Runtime) processEvent(event Event) {
        rt.mu.Lock()
        defer rt.mu.Unlock()

        currentState := rt.machine.states[rt.current]
        if currentState == nil {
                return
        }

        // Find matching transition (guards are checked in pickTransition)
        // Search from innermost state outward (Step 6)
        transition := rt.pickTransitionHierarchical(currentState, event)
        if transition == nil {
                return // No matching transition, ignore event
        }

        // Internal transition (Target == 0)
        if transition.Target == 0 {
                // Execute action only, no state change
                if transition.Action != nil {
                        transition.Action(rt.ctx, &event, rt.current, rt.current)
                }
                // Check if current state should generate done event
                // (e.g., compound state whose child is now done)
                rt.checkFinalState(rt.ctx)
                // Process any eventless transitions
                rt.processMicrosteps(rt.ctx)
                return
        }

        // External transition - use LCA algorithm for proper entry/exit order (Step 5)
        from := rt.current
        to := transition.Target
        
        // Check if target is a history state
        targetState := rt.machine.states[to]
        if targetState != nil && targetState.IsHistoryState {
                // Restore history instead of entering target directly
                restoredState, err := rt.restoreHistory(rt.ctx, targetState, &event, from)
                if err != nil {
                        // History restoration failed, use default or skip
                        if targetState.HistoryDefault != 0 {
                                to = targetState.HistoryDefault
                        } else {
                                return // Cannot restore history and no default
                        }
                } else {
                        to = restoredState
                }
        }

        // Compute Least Common Ancestor
        lca := rt.computeLCA(from, to)

        // Exit states from current up to (but not including) LCA
        rt.exitToLCA(rt.ctx, &event, from, to, lca)

        // Execute transition action
        if transition.Action != nil {
                transition.Action(rt.ctx, &event, from, to)
        }

        // Enter states from LCA down to target
        rt.enterFromLCA(rt.ctx, &event, from, to, lca)

        // Update current state - enterFromLCA updates rt.current to deepest entered state
        // (it's already been set by enterInitialChildren within enterFromLCA)

        // Check if we entered a final state (Step 12)
        rt.checkFinalState(rt.ctx)

        // Process eventless transitions after state change (Step 8 - microsteps)
        rt.processMicrosteps(rt.ctx)
}

// processMicrosteps processes eventless (NO_EVENT) transitions until stable (Step 8)
// This implements the "microstep" concept - keep processing eventless transitions
// until no more are enabled. Includes loop protection to prevent infinite loops.
func (rt *Runtime) processMicrosteps(ctx context.Context) {
        // Create a NO_EVENT event for eventless transitions
        noEvent := Event{ID: NO_EVENT}
        
        for i := 0; i < MAX_MICROSTEPS; i++ {
                currentState := rt.machine.states[rt.current]
                if currentState == nil {
                        return
                }
                
                // Look for eventless transition (Event == NO_EVENT)
                // Search from innermost state outward, same as normal transitions
                transition := rt.pickTransitionHierarchical(currentState, noEvent)
                if transition == nil {
                        // No eventless transition found, stable state reached
                        return
                }
                
                // Found an eventless transition - execute it
                
                // Internal transition (Target == 0)
                if transition.Target == 0 {
                        // Execute action only, no state change
                        if transition.Action != nil {
                                transition.Action(ctx, &noEvent, rt.current, rt.current)
                        }
                        // Internal transition doesn't change state, but we continue
                        // the microstep loop in case there are more eventless transitions
                        continue
                }
                
                // External eventless transition
                from := rt.current
                to := transition.Target
                
                // Check if target is a history state
                targetState := rt.machine.states[to]
                if targetState != nil && targetState.IsHistoryState {
                        // Restore history instead of entering target directly
                        restoredState, err := rt.restoreHistory(ctx, targetState, &noEvent, from)
                        if err != nil {
                                // History restoration failed, use default or skip
                                if targetState.HistoryDefault != 0 {
                                        to = targetState.HistoryDefault
                                } else {
                                        continue // Cannot restore history and no default, skip this transition
                                }
                        } else {
                                to = restoredState
                        }
                }
                
                // Compute Least Common Ancestor
                lca := rt.computeLCA(from, to)
                
                // Exit states from current up to (but not including) LCA
                rt.exitToLCA(ctx, &noEvent, from, to, lca)
                
                // Execute transition action
                if transition.Action != nil {
                        transition.Action(ctx, &noEvent, from, to)
                }
                
                // Enter states from LCA down to target
                rt.enterFromLCA(ctx, &noEvent, from, to, lca)
                
                // Update current state
                rt.current = to
                
                // Check if we entered a final state (Step 12)
                rt.checkFinalState(ctx)
                
                // Continue loop to check for more eventless transitions in new state
        }
        
        // If we reach here, we've hit MAX_MICROSTEPS - potential infinite loop
        // In production, this might warrant logging or error handling
}

// getAncestors returns the path from the given state to the root (inclusive)
func (rt *Runtime) getAncestors(stateID StateID) []StateID {
        var ancestors []StateID
        current := rt.machine.states[stateID]
        for current != nil {
                ancestors = append(ancestors, current.ID)
                current = current.Parent
        }
        return ancestors
}

// computeLCA finds the Least Common Ancestor of two states
func (rt *Runtime) computeLCA(from, to StateID) StateID {
        if from == to {
                // Self-transition: LCA is the parent
                state := rt.machine.states[from]
                if state != nil && state.Parent != nil {
                        return state.Parent.ID
                }
                return 0 // No parent (root state)
        }

        fromAncestors := rt.getAncestors(from)
        toAncestors := rt.getAncestors(to)

        // Convert to map for O(1) lookup
        fromSet := make(map[StateID]bool)
        for _, id := range fromAncestors {
                fromSet[id] = true
        }

        // Find first common ancestor in 'to' path
        for _, id := range toAncestors {
                if fromSet[id] {
                        return id
                }
        }

        return 0 // No common ancestor (shouldn't happen in valid state machine)
}

// exitToLCA exits states from current up to (but not including) LCA
func (rt *Runtime) exitToLCA(ctx context.Context, event *Event, from, to, lca StateID) {
        current := rt.machine.states[from]
        for current != nil && current.ID != lca {
                // Record history before exiting
                if current.Parent != nil {
                        rt.recordHistory(current.Parent.ID, current.ID)
                }
                
                // Clear done event flag when exiting
                rt.clearDoneEvent(current.ID)
                
                if current.ExitAction != nil {
                        current.ExitAction(ctx, event, from, to)
                }
                current = current.Parent
        }
}

// enterFromLCA enters states from LCA down to target and its initial children
func (rt *Runtime) enterFromLCA(ctx context.Context, event *Event, from, to, lca StateID) {
        // Build path from LCA to target
        var path []StateID
        current := rt.machine.states[to]
        for current != nil && current.ID != lca {
                path = append([]StateID{current.ID}, path...) // prepend to reverse order
                current = current.Parent
        }

        // Enter states in order (parent to child)
        // Execute InitialAction after parent entry but before child entry (Step 10)
        for i, stateID := range path {
                state := rt.machine.states[stateID]
                if state == nil {
                        continue
                }

                // Check if this is a parallel state
                if state.IsParallel {
                        // Enter parallel state - spawn goroutines for each region
                        rt.enterParallelState(ctx, state)
                        // Don't continue - parallel regions handle their own entry
                        return
                }

                // Execute entry action
                if state.EntryAction != nil {
                        state.EntryAction(ctx, event, from, to)
                } else if stateID == 112 {
                        // DEBUG: Why is state 112's entry action not being called?
                        _ = stateID // prevent unused warning
                }

                // Execute initial action if this state has children and we're entering them
                // InitialAction runs after parent entry but before child entry
                if i < len(path)-1 && state.InitialAction != nil {
                        nextStateID := path[i+1]
                        state.InitialAction(ctx, event, stateID, nextStateID)
                }
        }

        // Continue entering initial children until we reach an atomic state
        // Only do this if the target state has children (i.e., it's a compound state)
        lastState := rt.machine.states[to]
        if lastState != nil && !lastState.IsParallel && lastState.Initial != 0 && len(lastState.Children) > 0 {
                rt.enterInitialChildren(ctx, event, from, to, lastState)
        } else if lastState != nil {
                // Target is a leaf state or has no initial - set it as current
                rt.current = to
        }
        // If lastState is nil, rt.current remains unchanged
}

// enterInitialChildren recursively enters initial children until reaching an atomic state
func (rt *Runtime) enterInitialChildren(ctx context.Context, event *Event, from, to StateID, state *State) {
        for state.Initial != 0 && len(state.Children) > 0 {
                initialChild := rt.machine.states[state.Initial]
                if initialChild == nil {
                        break
                }

                // Execute InitialAction before entering child
                if state.InitialAction != nil {
                        state.InitialAction(ctx, event, state.ID, initialChild.ID)
                }

                // Check if initial child is parallel
                if initialChild.IsParallel {
                        rt.enterParallelState(ctx, initialChild)
                        return
                }

                // Execute entry action for initial child
                if initialChild.EntryAction != nil {
                        initialChild.EntryAction(ctx, event, from, to)
                }

                // Update rt.current to the child we just entered
                rt.current = initialChild.ID

                // Check if we entered a final state
                rt.checkFinalState(ctx)

                // Continue with this child's children
                state = initialChild
        }
}

// checkFinalState checks if current state is final and generates done.state.id events (Step 12)
func (rt *Runtime) checkFinalState(ctx context.Context) {
        currentState := rt.machine.states[rt.current]
        if currentState == nil {
                return
        }

        // Check if current state is final (support both IsFinal and deprecated Final)
        if !currentState.IsFinal && !currentState.Final {
                return
        }

        // Current state is final - walk up the ancestor chain and generate done events
        // for any compound states that are now complete
        parent := currentState.Parent
        for parent != nil {
                // Generate done event for this parent (use rt.current as parallel state ID)
                rt.generateDoneEvent(ctx, parent, currentState, rt.current)

                // Continue up the chain to check if grandparent is also complete
                // (Only continue if parent is a compound state with single child)
                if parent.IsParallel || len(parent.Children) != 1 {
                        break // Parallel or multi-child states don't cascade
                }
                parent = parent.Parent
        }
}

// generateDoneEvent generates a done.state.id event when a state completes
// parallelStateID is the ID of the current parallel state (used to determine event routing)
func (rt *Runtime) generateDoneEvent(ctx context.Context, parent *State, finalState *State, parallelStateID StateID) {
        if parent == nil {
                return
        }
        
        // Check if we already sent done event for this parent (check first to avoid race)
        rt.doneEventsMu.Lock()
        if rt.doneEventsPending[parent.ID] {
                rt.doneEventsMu.Unlock()
                return
        }

        // Check if we should emit done event (while holding lock to prevent race)
        if !rt.shouldEmitDoneEvent(parent) {
                rt.doneEventsMu.Unlock()
                return
        }

        // Mark as pending before releasing lock
        rt.doneEventsPending[parent.ID] = true
        rt.doneEventsMu.Unlock()
        
        // Create done event: done.state.<parentID>
        // We use negative EventIDs for done events to avoid conflicts
        // done.state.X = -(1000000 + X)
        doneEventID := EventID(-(1000000 + int(parent.ID)))
        
        doneEvent := Event{
                ID:      doneEventID,
                Data:    finalState.FinalStateData,
                Address: 0, // broadcast
        }
        
        // Determine where to send the done event
        // If parent is the current parallel state, send to root event queue
        // Otherwise, use SendEvent to route to appropriate region
        parallelState := rt.machine.states[parallelStateID]
        shouldSendToRoot := (parallelState != nil && parallelState.IsParallel && parent.ID == parallelState.ID)

        go func() {
                sendCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
                defer cancel()

                if shouldSendToRoot {
                        // Send to root event queue for parallel state done events
                        select {
                        case rt.eventQueue <- doneEvent:
                                // Event sent successfully
                        case <-sendCtx.Done():
                                // Timeout - event not delivered
                        case <-rt.ctx.Done():
                                // Runtime shutting down
                        }
                } else {
                        // Use SendEvent for region-level done events
                        rt.SendEvent(sendCtx, doneEvent)
                }
        }()
}

// shouldEmitDoneEvent checks if a parent state should emit a done event
func (rt *Runtime) shouldEmitDoneEvent(parent *State) bool {
        if parent.IsParallel {
                // All regions must be in final state
                return rt.allRegionsInFinalState(parent)
        }
        // Sequential state: emit immediately when child is final
        return true
}

// allRegionsInFinalState checks if all parallel regions are in final state
func (rt *Runtime) allRegionsInFinalState(parallelState *State) bool {
        rt.regionMu.RLock()
        defer rt.regionMu.RUnlock()

        for childID := range parallelState.Children {
                region, exists := rt.parallelRegions[childID]
                if !exists {
                        return false
                }

                region.mu.RLock()
                currentStateID := region.currentState
                region.mu.RUnlock()

                state := rt.machine.states[currentStateID]
                if state == nil || (!state.IsFinal && !state.Final) {
                        return false
                }
        }
        return true
}

// clearDoneEvent clears the done event flag when exiting a state
func (rt *Runtime) clearDoneEvent(stateID StateID) {
        rt.doneEventsMu.Lock()
        defer rt.doneEventsMu.Unlock()
        delete(rt.doneEventsPending, stateID)
}

// pickTransition finds the first matching transition for the given event in a single state
// Guards are checked here to allow fallthrough to next transition if guard fails
func (rt *Runtime) pickTransition(state *State, event Event) *Transition {
        var wildcardTransition *Transition

        for _, t := range state.Transitions {
                if t == nil {
                        continue
                }

                // Check for exact event match
                if t.Event == event.ID {
                        // Check guard if present
                        if t.Guard != nil {
                                pass, err := t.Guard(rt.ctx, &event, rt.current, t.Target)
                                if err != nil || !pass {
                                        continue // Guard failed, try next transition
                                }
                        }
                        return t
                }

                // Check for wildcard match (ANY_EVENT)
                // Note: ANY_EVENT should NOT match NO_EVENT (eventless transitions)
                if t.Event == ANY_EVENT && event.ID != NO_EVENT && wildcardTransition == nil {
                        // Check guard if present
                        if t.Guard != nil {
                                pass, err := t.Guard(rt.ctx, &event, rt.current, t.Target)
                                if err != nil || !pass {
                                        continue // Guard failed, try next transition
                                }
                        }
                        wildcardTransition = t
                }
        }

        // Return wildcard transition if no exact match found
        return wildcardTransition
}

// pickTransitionHierarchical searches for matching transition from innermost state outward (Step 6)
// Child transitions take precedence over parent transitions
func (rt *Runtime) pickTransitionHierarchical(state *State, event Event) *Transition {
        current := state
        for current != nil {
                // Try to find a matching transition in current state
                transition := rt.pickTransition(current, event)
                if transition != nil {
                        return transition
                }
                // No match in current state, try parent
                current = current.Parent
        }
        return nil // No matching transition found in entire hierarchy
}

// enterState executes entry actions for a state
func (rt *Runtime) enterState(ctx context.Context, event *Event, from StateID, to StateID) error {
        state := rt.machine.states[to]
        if state == nil {
                return errors.New("target state not found")
        }

        if state.EntryAction != nil {
                return state.EntryAction(ctx, event, from, to)
        }
        return nil
}

// exitState executes exit actions for a state
func (rt *Runtime) exitState(ctx context.Context, event *Event, from StateID, to StateID) error {
        state := rt.machine.states[from]
        if state == nil {
                return nil
        }

        if state.ExitAction != nil {
                return state.ExitAction(ctx, event, from, to)
        }
        return nil
}

// Legacy Machine API (kept for backward compatibility with old tests)

// Start enters machine initial state.
func (m *Machine) Start(ctx context.Context) error {
        if m.current == nil {
                return errors.New("machine has no current state")
        }
        return m.current.enterState(ctx, nil, m.current.ID, m.current.ID)
}

// SendEvent sends an event to the machine.
func (m *Machine) SendEvent(ctx context.Context, event Event) error {
        return errors.New("use Runtime.SendEvent instead")
}

// IsInState checks if the machine is in the given state.
func (m *Machine) IsInState(stateID StateID) bool {
        if m.current == nil {
                return false
        }
        return m.current.ID == stateID
}

// enterState enters a state.
func (s *State) enterState(ctx context.Context, event *Event, from StateID, to StateID) error {
        if s.EntryAction != nil {
                return s.EntryAction(ctx, event, from, to)
        }
        return nil
}

// exitState exits a state.
func (s *State) exitState(ctx context.Context, event *Event, from StateID, to StateID) error {
        if s.ExitAction != nil {
                return s.ExitAction(ctx, event, from, to)
        }
        return nil
}

// On adds a transition to the state.
func (s *State) On(event EventID, target StateID, guard *Guard, action *Action) {
        t := &Transition{
                Event:  event,
                Source: s,
                Target: target,
        }
        if guard != nil {
                t.Guard = *guard
        }
        if action != nil {
                t.Action = *action
        }
        s.Transitions = append(s.Transitions, t)
}

// History State Support Functions

// recordHistory records the last active child state for history restoration
func (rt *Runtime) recordHistory(parentID StateID, childID StateID) {
        // Record shallow history
        rt.historyMu.Lock()
        rt.history[parentID] = childID
        rt.historyMu.Unlock()
        
        // Record deep history (full active configuration)
        rt.deepHistoryMu.Lock()
        config := rt.getActiveConfiguration()
        rt.deepHistory[parentID] = config
        rt.deepHistoryMu.Unlock()
}

// getActiveConfiguration returns the current active state configuration
func (rt *Runtime) getActiveConfiguration() []StateID {
        var config []StateID
        
        // Add current state and all ancestors
        current := rt.machine.states[rt.current]
        for current != nil {
                config = append([]StateID{current.ID}, config...) // prepend
                current = current.Parent
        }
        
        return config
}

// restoreHistory restores a previously saved state configuration
func (rt *Runtime) restoreHistory(ctx context.Context, historyState *State, event *Event, from StateID) (StateID, error) {
        if historyState.HistoryType == HistoryDeep {
                return rt.restoreDeepHistory(ctx, historyState, event, from)
        }
        return rt.restoreShallowHistory(ctx, historyState, event, from)
}

// restoreShallowHistory restores only the direct child state
func (rt *Runtime) restoreShallowHistory(ctx context.Context, historyState *State, event *Event, from StateID) (StateID, error) {
        if historyState.Parent == nil {
                return historyState.HistoryDefault, nil
        }
        
        parentID := historyState.Parent.ID
        
        rt.historyMu.RLock()
        lastChild, exists := rt.history[parentID]
        rt.historyMu.RUnlock()
        
        if !exists || lastChild == 0 {
                // No history, use default
                if historyState.HistoryDefault != 0 {
                        return historyState.HistoryDefault, nil
                }
                // No default specified, use parent's initial state
                if historyState.Parent.Initial != 0 {
                        return historyState.Parent.Initial, nil
                }
                return 0, fmt.Errorf("no history and no default for history state %d", historyState.ID)
        }
        
        return lastChild, nil
}

// restoreDeepHistory restores the entire state hierarchy
func (rt *Runtime) restoreDeepHistory(ctx context.Context, historyState *State, event *Event, from StateID) (StateID, error) {
        if historyState.Parent == nil {
                return historyState.HistoryDefault, nil
        }
        
        parentID := historyState.Parent.ID
        
        rt.deepHistoryMu.RLock()
        config, exists := rt.deepHistory[parentID]
        rt.deepHistoryMu.RUnlock()
        
        if !exists || len(config) == 0 {
                // No history, use default
                if historyState.HistoryDefault != 0 {
                        return historyState.HistoryDefault, nil
                }
                // No default specified, use parent's initial state
                if historyState.Parent.Initial != 0 {
                        return rt.machine.findDeepestInitial(historyState.Parent.Initial), nil
                }
                return 0, fmt.Errorf("no history and no default for history state %d", historyState.ID)
        }
        
        // Return the deepest state in the configuration
        if len(config) > 0 {
                return config[len(config)-1], nil
        }
        
        return historyState.HistoryDefault, nil
}

// clearHistory clears all history for a given state
func (rt *Runtime) clearHistory(stateID StateID) {
        rt.historyMu.Lock()
        delete(rt.history, stateID)
        rt.historyMu.Unlock()
        
        rt.deepHistoryMu.Lock()
        delete(rt.deepHistory, stateID)
        rt.deepHistoryMu.Unlock()
}

// DoneEventID returns the EventID for a done.state.id event
// This is a helper function for users to create transitions on done events
func DoneEventID(stateID StateID) EventID {
        return EventID(-(1000000 + int(stateID)))
}


// Public aliases for tick-based runtime (realtime package)
// These expose internal methods for use by the RealtimeRuntime

// ProcessEvent exposes processEvent for tick-based runtime
func (rt *Runtime) ProcessEvent(event Event) {
        rt.processEvent(event)
}

// ProcessMicrosteps exposes processMicrosteps for tick-based runtime
func (rt *Runtime) ProcessMicrosteps(ctx context.Context) {
        rt.processMicrosteps(ctx)
}

// GetCurrentState exposes current state for tick-based runtime
func (rt *Runtime) GetCurrentState() StateID {
        rt.mu.RLock()
        defer rt.mu.RUnlock()
        return rt.current
}
