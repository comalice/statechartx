package realtime

import (
	"context"

	"sort"
	"time"

	"github.com/comalice/statechartx"
)

// createParallelHooks creates hooks for sequential parallel state processing
func (rt *RealtimeRuntime) createParallelHooks() *statechartx.ParallelStateHooks {
	return &statechartx.ParallelStateHooks{
		OnEnterParallel: func(ctx context.Context, state *statechartx.State) error {
			return rt.enterParallelState(ctx, state)
		},
		OnExitParallel: func(ctx context.Context, state *statechartx.State) error {
			return rt.exitParallelState(ctx, state)
		},
		OnSendToRegions: func(ctx context.Context, event statechartx.Event) error {
			return rt.sendEventToRegionsSequential(ctx, event)
		},
	}
}

// Start overrides the embedded Runtime's Start to use sequential parallel state processing
func (rt *RealtimeRuntime) Start(ctx context.Context) error {

	// Initialize the embedded runtime's context (needed for internal operations)
	rt.Runtime.SetContext(ctx)

	// Initialize machine state without using goroutines for parallel states
	rt.enterInitialStateSequential(ctx)

	// Start tick loop
	rt.tickCtx, rt.tickCancel = context.WithCancel(ctx)
	rt.ticker = time.NewTicker(rt.tickRate)

	go rt.tickLoop()

	return nil
}

// enterInitialStateSequential enters the initial state without spawning goroutines
func (rt *RealtimeRuntime) enterInitialStateSequential(ctx context.Context) error {
	// Ensure we're in macrostep mode for entire initialization
	rt.internalQueueMu.Lock()
	rt.inMacrostep = true
	rt.internalQueueMu.Unlock()

	defer func() {
		rt.internalQueueMu.Lock()
		rt.inMacrostep = false
		rt.internalQueueMu.Unlock()
	}()

	machine := rt.Runtime.GetMachine()
	initialStateID := machine.Initial

	// Find the deepest initial state
	deepestInitial := machine.FindDeepestInitial(initialStateID)

	// Build the path from root to deepest initial
	var path []statechartx.StateID
	current := machine.GetState(deepestInitial)
	for current != nil {
		path = append([]statechartx.StateID{current.ID}, path...) // prepend
		if current.ID == initialStateID {
			break
		}
		current = current.Parent
	}

	// Enter each state in the path
	for i, stateID := range path {
		state := machine.GetState(stateID)
		if state == nil {
			continue
		}

		// If this is a parallel state, initialize regions instead of spawning goroutines
		if state.IsParallel {
			if err := rt.enterParallelState(ctx, state); err != nil {
				return err
			}
			// Don't continue down - parallel state handles its own children
			return nil
		}

		// Execute entry action
		if state.EntryAction != nil {
			if err := state.EntryAction(ctx, nil, 0, stateID); err != nil {
				return err
			}
		}

		// Execute initial action if not at end
		if i < len(path)-1 && state.InitialAction != nil {
			nextStateID := path[i+1]
			if err := state.InitialAction(ctx, nil, stateID, nextStateID); err != nil {
				return err
			}
		}
	}

	// Set the current state in the embedded runtime
	rt.Runtime.SetCurrentState(deepestInitial)

	// Process eventless transitions to completion (macrostep semantics)
	rt.processMacrostepToCompletion(ctx)

	return nil
}

// enterParallelState enters a parallel state by initializing regions sequentially
func (rt *RealtimeRuntime) enterParallelState(ctx context.Context, state *statechartx.State) error {
	if state == nil || !state.IsParallel {
		return nil
	}

	// Initialize region map and sort region IDs (needs lock)
	rt.regionMu.Lock()
	if rt.parallelRegionStates[state.ID] == nil {
		rt.parallelRegionStates[state.ID] = make(map[statechartx.StateID]*realtimeRegion)
	}

	// Get sorted list of region IDs for deterministic ordering
	regionIDs := make([]statechartx.StateID, 0, len(state.Children))
	for regionID := range state.Children {
		regionIDs = append(regionIDs, regionID)
	}
	rt.regionMu.Unlock()

	// Sort by StateID for document order
	sort.Slice(regionIDs, func(i, j int) bool {
		return regionIDs[i] < regionIDs[j]
	})

	// Execute parent entry action first
	if state.EntryAction != nil {
		if err := state.EntryAction(ctx, nil, 0, state.ID); err != nil {
			return err
		}
	}

	// CRITICAL: Wrap the parallel state's ExitAction to ensure region exits happen
	// The embedded Runtime's exitToLCA doesn't call exitParallelState - it just calls ExitAction
	// So we wrap the ExitAction to call our region exit logic
	originalExitAction := state.ExitAction
	state.ExitAction = func(ctx context.Context, event *statechartx.Event, from, to statechartx.StateID) error {
		// DEBUG

		// NOTE: Don't need to set inMacrostep here - it should already be set by the caller
		// (either processMacrostepToCompletion or the explicit setting before Runtime calls)

		// First, exit all regions in reverse document order
		rt.regionMu.Lock()
		regions, exists := rt.parallelRegionStates[state.ID]
		if exists {
			// Get sorted region IDs
			regionIDs := make([]statechartx.StateID, 0, len(regions))
			for regionID := range regions {
				regionIDs = append(regionIDs, regionID)
			}
			sort.Slice(regionIDs, func(i, j int) bool {
				return regionIDs[i] < regionIDs[j]
			})

			// Exit regions in REVERSE document order
			for i := len(regionIDs) - 1; i >= 0; i-- {
				regionID := regionIDs[i]
				region := regions[regionID]
				if region != nil {
					rt.exitRegionHierarchy(ctx, region)
				}
			}

			// Clean up region state
			delete(rt.parallelRegionStates, state.ID)
		}
		rt.regionMu.Unlock()

		// Then call the original exit action if it exists
		if originalExitAction != nil {
			return originalExitAction(ctx, event, from, to)
		}
		return nil
	}

	// Enter each region sequentially in document order
	for _, regionID := range regionIDs {
		child := state.Children[regionID]
		if child == nil {
			continue
		}

		// Find deepest initial state for this region
		initialState := rt.Runtime.GetMachine().FindDeepestInitial(regionID)

		// Create region state
		region := &realtimeRegion{
			regionID:     regionID,
			currentState: initialState,
			eventQueue:   make([]statechartx.Event, 0, 10),
		}

		rt.regionMu.Lock()
		rt.parallelRegionStates[state.ID][regionID] = region
		rt.regionMu.Unlock()

		// Execute entry actions for the region hierarchy
		rt.enterRegionHierarchyWithoutMicrosteps(ctx, region, child)
	}

	// NOTE: Do NOT call processMacrostepToCompletion here!
	// It must only be called at the top level (after enterInitialStateSequential
	// completes, or after tick processing) to avoid infinite recursion.
	// The embedded Runtime's ProcessMicrosteps will be called from
	// processMacrostepToCompletion at the top level.

	return nil
}

// exitParallelState exits a parallel state by exiting regions sequentially in reverse order
func (rt *RealtimeRuntime) exitParallelState(ctx context.Context, state *statechartx.State) error {
	if state == nil || !state.IsParallel {
		return nil
	}

	rt.regionMu.Lock()
	defer rt.regionMu.Unlock()

	regions, exists := rt.parallelRegionStates[state.ID]
	if !exists {
		return nil
	}

	// Get sorted list of region IDs
	regionIDs := make([]statechartx.StateID, 0, len(regions))
	for regionID := range regions {
		regionIDs = append(regionIDs, regionID)
	}

	// Sort by StateID
	sort.Slice(regionIDs, func(i, j int) bool {
		return regionIDs[i] < regionIDs[j]
	})

	// Exit regions in REVERSE document order (children before parents)
	for i := len(regionIDs) - 1; i >= 0; i-- {
		regionID := regionIDs[i]
		region := regions[regionID]
		if region == nil {
			continue
		}

		// Exit current state hierarchy in this region
		rt.exitRegionHierarchy(ctx, region)
	}

	// Execute parent exit action last
	if state.ExitAction != nil {
		if err := state.ExitAction(ctx, nil, state.ID, 0); err != nil {
			return err
		}
	}

	// Clean up region state
	delete(rt.parallelRegionStates, state.ID)

	return nil
}

// enterRegionHierarchyWithoutMicrosteps enters states without processing NO_EVENT transitions
// Used during microstep processing to avoid infinite recursion
func (rt *RealtimeRuntime) enterRegionHierarchyWithoutMicrosteps(ctx context.Context, region *realtimeRegion, regionRoot *statechartx.State) {
	// Build path from region root to current state (deepest initial)
	var path []statechartx.StateID
	current := rt.Runtime.GetMachine().GetState(region.currentState)
	for current != nil && current.ID != regionRoot.ID {
		path = append([]statechartx.StateID{current.ID}, path...) // prepend
		current = current.Parent
	}
	// Add region root itself
	if regionRoot != nil {
		path = append([]statechartx.StateID{regionRoot.ID}, path...)
	}

	// Execute entry actions in order (parent to child)
	for i, stateID := range path {
		state := rt.Runtime.GetMachine().GetState(stateID)
		if state == nil {
			continue
		}

		// Execute entry action
		if state.EntryAction != nil {
			state.EntryAction(ctx, nil, 0, stateID)
		}

		// Execute initial action if this is not the last state
		if i < len(path)-1 && state.InitialAction != nil {
			nextStateID := path[i+1]
			state.InitialAction(ctx, nil, stateID, nextStateID)
		}
	}
}

// enterRegionHierarchy is now just an alias for enterRegionHierarchyWithoutMicrosteps
// Microsteps are processed separately at the top level after all entries complete
func (rt *RealtimeRuntime) enterRegionHierarchy(ctx context.Context, region *realtimeRegion, regionRoot *statechartx.State) {
	rt.enterRegionHierarchyWithoutMicrosteps(ctx, region, regionRoot)
}

// exitRegionHierarchy exits the current state hierarchy for a region
func (rt *RealtimeRuntime) exitRegionHierarchy(ctx context.Context, region *realtimeRegion) {
	// Build path from current state to region root
	var path []statechartx.StateID
	current := rt.Runtime.GetMachine().GetState(region.currentState)
	for current != nil {
		path = append(path, current.ID)
		// Stop at region boundary (when parent would be the parallel state)
		if current.Parent != nil && current.Parent.IsParallel {
			break
		}
		current = current.Parent
	}

	// Execute exit actions in order (child to parent)
	// Note: path is already in child→parent order from the loop above
	for _, stateID := range path {
		state := rt.Runtime.GetMachine().GetState(stateID)
		if state == nil {
			continue
		}

		if state.ExitAction != nil {
			state.ExitAction(ctx, nil, state.ID, 0)
		}
	}
}

// processRegionMicrosteps processes eventless transitions for a region
func (rt *RealtimeRuntime) processRegionMicrosteps(ctx context.Context, region *realtimeRegion) {
	// Process NO_EVENT transitions up to MAX_MICROSTEPS times
	visitedStates := make(map[statechartx.StateID]bool)

	for i := 0; i < 100; i++ { // MAX_MICROSTEPS = 100
		currentState := rt.Runtime.GetMachine().GetState(region.currentState)
		if currentState == nil {
			return
		}

		// Check for infinite loop - if we've visited this state before in this microstep run
		if visitedStates[region.currentState] {
			// We're in a loop, stop processing
			return
		}
		visitedStates[region.currentState] = true

		// Look for eventless transition
		var selectedTransition *statechartx.Transition
		for _, transition := range currentState.Transitions {
			// Only check NO_EVENT transitions
			if transition.Event != statechartx.NO_EVENT {
				continue
			}

			// Check guard
			if transition.Guard != nil {
				ok, err := transition.Guard(ctx, nil, region.currentState, transition.Target)
				if err != nil || !ok {
					continue
				}
			}

			// Found matching eventless transition
			selectedTransition = transition
			break
		}

		if selectedTransition == nil {
			// No more eventless transitions
			return
		}

		// Execute the eventless transition
		if selectedTransition.Target != 0 {
			// External transition
			rt.exitRegionHierarchy(ctx, region)

			if selectedTransition.Action != nil {
				selectedTransition.Action(ctx, nil, region.currentState, selectedTransition.Target)
			}

			// Update region state to target
			region.currentState = rt.Runtime.GetMachine().FindDeepestInitial(selectedTransition.Target)

			// Enter the new state hierarchy
			targetState := rt.Runtime.GetMachine().GetState(selectedTransition.Target)
			if targetState != nil {
				rt.enterRegionHierarchyWithoutMicrosteps(ctx, region, targetState)
			}
			// Loop will continue to check for more NO_EVENT transitions in the new state
		} else {
			// Internal transition - just execute action, don't process further microsteps
			if selectedTransition.Action != nil {
				selectedTransition.Action(ctx, nil, region.currentState, 0)
			}
			// Don't continue for internal transitions - they don't change state
			return
		}
	}
}

// processSingleEventlessTransition processes ONE eventless transition on the current state
// Returns true if a transition was found and processed, false otherwise
// Delegates to the embedded Runtime's ProcessSingleMicrostep method
func (rt *RealtimeRuntime) processSingleEventlessTransition(ctx context.Context) bool {
	return rt.Runtime.ProcessSingleMicrostep(ctx)
}

// processMacrostepToCompletion processes eventless transitions and internal events until stable
// This implements SCXML macrostep semantics: continue processing until no eventless transitions
// are enabled and the internal event queue is empty
func (rt *RealtimeRuntime) processMacrostepToCompletion(ctx context.Context) {
	// Set macrostep flag so SendEvent routes to internal queue
	rt.internalQueueMu.Lock()
	rt.inMacrostep = true
	rt.internalQueueMu.Unlock()

	defer func() {
		rt.internalQueueMu.Lock()
		rt.inMacrostep = false
		rt.internalQueueMu.Unlock()
	}()

	_ = rt.Runtime.GetCurrentState()

	for i := 0; i < 100; i++ { // MAX_MICROSTEPS
		madeProgress := false

		// SCXML Macrostep Algorithm:
		// 1. Check internal queue - if not empty, process ONE event
		// 2. If internal queue empty, check for eventless transition
		// 3. If neither, check parallel regions
		// 4. If nothing to do, we're done

		// Step 1: Check internal event queue FIRST (higher priority than eventless transitions)
		rt.internalQueueMu.Lock()
		if len(rt.internalEventQueue) > 0 {
			// Dequeue first internal event
			event := rt.internalEventQueue[0]
			rt.internalEventQueue = rt.internalEventQueue[1:]
			rt.internalQueueMu.Unlock()

			// Process this internal event
			// Check if we have parallel regions
			rt.regionMu.RLock()
			hasParallelRegions := len(rt.parallelRegionStates) > 0
			rt.regionMu.RUnlock()

			if !hasParallelRegions {
				// No parallel regions - use embedded Runtime
				// IMPORTANT: Use ProcessEventWithoutMicrosteps to prevent automatic microstep processing
				// We want to maintain manual control over the macrostep loop
				rt.Runtime.ProcessEventWithoutMicrosteps(event)
			} else {
				// Have parallel regions - check BOTH the parallel state AND the regions
				// First, check if the event triggers a transition on the parallel state itself
				// (this allows transitions OUT of the parallel state)
				rt.Runtime.ProcessEventWithoutMicrosteps(event)

				// Then, route to all parallel regions (if we're still in the parallel state)
				rt.regionMu.RLock()
				hasRegionsStill := len(rt.parallelRegionStates) > 0
				if hasRegionsStill {
					for _, regions := range rt.parallelRegionStates {
						// Get sorted region IDs
						regionIDs := make([]statechartx.StateID, 0, len(regions))
						for regionID := range regions {
							regionIDs = append(regionIDs, regionID)
						}
						sort.Slice(regionIDs, func(i, j int) bool {
							return regionIDs[i] < regionIDs[j]
						})

						// Process event in each region sequentially
						for _, regionID := range regionIDs {
							region := regions[regionID]
							if region != nil {
								rt.processRegionEvent(ctx, region, event)
							}
						}
					}
				}
				rt.regionMu.RUnlock()
			}

			madeProgress = true
			_ = rt.Runtime.GetCurrentState()
			continue // Start over to check for more internal events or eventless transitions
		}
		rt.internalQueueMu.Unlock()

		// Step 2: If internal queue empty, check for eventless transition
		if rt.processSingleEventlessTransition(ctx) {
			newStateID := rt.Runtime.GetCurrentState()
			madeProgress = true
			_ = newStateID
			continue // Start over from beginning
		}

		// Step 3: Check each parallel region for eventless transitions
		rt.regionMu.RLock()
		unlocked := false
		for _, regions := range rt.parallelRegionStates {
			regionIDs := make([]statechartx.StateID, 0, len(regions))
			for regionID := range regions {
				regionIDs = append(regionIDs, regionID)
			}
			sort.Slice(regionIDs, func(i, j int) bool {
				return regionIDs[i] < regionIDs[j]
			})

			for _, regionID := range regionIDs {
				region := regions[regionID]
				if region == nil {
					continue
				}

				if rt.hasEventlessTransition(region) {
					rt.regionMu.RUnlock()
					unlocked = true
					rt.executeEventlessTransition(ctx, region)
					madeProgress = true
					break
				}
			}

			if madeProgress {
				break
			}
		}
		if !unlocked {
			rt.regionMu.RUnlock()
		}

		if madeProgress {
			continue // Start over from beginning
		}

		// No progress made - stable configuration reached
		break
	}
}

// hasEventlessTransition checks if the region's current state has an enabled eventless transition
func (rt *RealtimeRuntime) hasEventlessTransition(region *realtimeRegion) bool {
	currentState := rt.Runtime.GetMachine().GetState(region.currentState)
	if currentState == nil {
		return false
	}

	for _, transition := range currentState.Transitions {
		if transition.Event != statechartx.NO_EVENT {
			continue
		}

		// Check guard if present
		if transition.Guard != nil {
			ok, err := transition.Guard(context.Background(), nil, region.currentState, transition.Target)
			if err != nil || !ok {
				continue
			}
		}

		return true
	}

	return false
}

// executeEventlessTransition executes a single eventless transition for a region
func (rt *RealtimeRuntime) executeEventlessTransition(ctx context.Context, region *realtimeRegion) {
	currentState := rt.Runtime.GetMachine().GetState(region.currentState)
	if currentState == nil {
		return
	}

	// Find the eventless transition
	var selectedTransition *statechartx.Transition
	for _, transition := range currentState.Transitions {
		if transition.Event != statechartx.NO_EVENT {
			continue
		}

		// Check guard
		if transition.Guard != nil {
			ok, err := transition.Guard(ctx, nil, region.currentState, transition.Target)
			if err != nil || !ok {
				continue
			}
		}

		selectedTransition = transition
		break
	}

	if selectedTransition == nil {
		return
	}

	// Execute the transition
	if selectedTransition.Target != 0 {
		// External transition
		rt.exitRegionHierarchy(ctx, region)

		if selectedTransition.Action != nil {
			selectedTransition.Action(ctx, nil, region.currentState, selectedTransition.Target)
		}

		region.currentState = rt.Runtime.GetMachine().FindDeepestInitial(selectedTransition.Target)

		targetState := rt.Runtime.GetMachine().GetState(selectedTransition.Target)
		if targetState != nil {
			rt.enterRegionHierarchyWithoutMicrosteps(ctx, region, targetState)
		}
	} else {
		// Internal transition
		if selectedTransition.Action != nil {
			selectedTransition.Action(ctx, nil, region.currentState, 0)
		}
	}
}

// processRegionEvent processes a single event within a region's context
func (rt *RealtimeRuntime) processRegionEvent(ctx context.Context, region *realtimeRegion, event statechartx.Event) {
	// Find matching transition from current state
	currentState := rt.Runtime.GetMachine().GetState(region.currentState)
	if currentState == nil {
		return
	}

	// Look for matching transition
	var selectedTransition *statechartx.Transition
	for _, transition := range currentState.Transitions {
		// Check event match
		if transition.Event != event.ID && transition.Event != statechartx.ANY_EVENT {
			continue
		}

		// Check guard
		if transition.Guard != nil {
			ok, err := transition.Guard(ctx, &event, region.currentState, transition.Target)
			if err != nil || !ok {
				continue
			}
		}

		// Found matching transition
		selectedTransition = transition
		break
	}

	if selectedTransition == nil {
		return
	}

	// Execute transition
	if selectedTransition.Target != 0 {
		// External transition - exit current, execute action, enter target
		rt.exitRegionHierarchy(ctx, region)

		if selectedTransition.Action != nil {
			selectedTransition.Action(ctx, &event, region.currentState, selectedTransition.Target)
		}

		// Update region's current state
		region.currentState = rt.Runtime.GetMachine().FindDeepestInitial(selectedTransition.Target)

		// Enter new state hierarchy
		targetState := rt.Runtime.GetMachine().GetState(selectedTransition.Target)
		if targetState != nil {
			rt.enterRegionHierarchyWithoutMicrosteps(ctx, region, targetState)
		}
	} else {
		// Internal transition - just execute action
		if selectedTransition.Action != nil {
			selectedTransition.Action(ctx, &event, region.currentState, 0)
		}
	}
}

// SendEvent queues an event for processing
// If called during macrostep processing, routes to internal queue (SCXML run-to-completion)
// Otherwise batches for next tick
func (rt *RealtimeRuntime) SendEvent(event statechartx.Event) error {
	// Check if we're in macrostep processing
	rt.internalQueueMu.Lock()
	if rt.inMacrostep {
		// Route to internal queue for immediate processing within this macrostep
		rt.internalEventQueue = append(rt.internalEventQueue, event)
		rt.internalQueueMu.Unlock()
		return nil
	}
	rt.internalQueueMu.Unlock()

	// Not in macrostep - batch for next tick
	rt.batchMu.Lock()
	defer rt.batchMu.Unlock()

	if len(rt.eventBatch) >= cap(rt.eventBatch) {
		return statechartx.ErrEventQueueFull
	}

	rt.eventBatch = append(rt.eventBatch, EventWithMeta{
		Event:       event,
		SequenceNum: rt.sequenceNum,
		Priority:    0,
	})
	rt.sequenceNum++
	return nil
}

// sendEventToRegionsSequential routes events to parallel regions (hook implementation)
func (rt *RealtimeRuntime) sendEventToRegionsSequential(ctx context.Context, event statechartx.Event) error {
	rt.regionMu.Lock()
	defer rt.regionMu.Unlock()

	// Route to parallel regions
	if event.Address == 0 {
		// Broadcast to all regions
		for _, regions := range rt.parallelRegionStates {
			for _, region := range regions {
				region.eventQueue = append(region.eventQueue, event)
			}
		}
	} else {
		// Targeted delivery
		for _, regions := range rt.parallelRegionStates {
			if region, exists := regions[event.Address]; exists {
				region.eventQueue = append(region.eventQueue, event)
				return nil
			}
		}
	}

	return nil
}
