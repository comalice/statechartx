package statechartx

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestMillionStates creates 1 million states and validates performance
func TestMillionStates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Log("Starting TestMillionStates...")
	start := time.Now()

	// Record initial memory
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialAlloc := m1.Alloc

	// Create a statechart with 1 million states
	// We'll create a flat structure with many parallel regions to distribute states
	const numRegions = 100
	const statesPerRegion = 10000
	const totalStates = numRegions * statesPerRegion

	// Create root parallel state
	root := &State{
		ID:         1,
		IsParallel: true,
		Children:   make(map[StateID]*State),
	}

	stateID := StateID(2)
	
	// Create parallel regions
	for r := 0; r < numRegions; r++ {
		region := &State{
			ID:       stateID,
			Parent:   root,
			Children: make(map[StateID]*State),
		}
		root.Children[stateID] = region
		stateID++
		
		// Add states to each region
		var firstChild StateID
		for s := 0; s < statesPerRegion; s++ {
			state := &State{
				ID:     stateID,
				Parent: region,
			}
			region.Children[stateID] = state
			if s == 0 {
				firstChild = stateID
			}
			stateID++
		}
		region.Initial = firstChild
	}

	creationTime := time.Since(start)

	// Measure memory after creating states
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	finalAlloc := m2.Alloc

	// Handle case where GC reduced memory below initial allocation
	var memoryUsed float64
	if finalAlloc > initialAlloc {
		memoryUsed = float64(finalAlloc-initialAlloc) / (1024 * 1024) // MB
	} else {
		memoryUsed = -float64(initialAlloc-finalAlloc) / (1024 * 1024) // MB (negative indicates GC freed memory)
	}

	t.Logf("Created %d states in %v", totalStates, creationTime)
	t.Logf("Memory used: %.2f MB", memoryUsed)
	t.Logf("Average time per state: %v", creationTime/totalStates)

	// Validate performance targets
	if creationTime > 10*time.Second {
		t.Errorf("Creation time %v exceeds 10s target", creationTime)
	}

	if memoryUsed > 1024.0 {
		t.Errorf("Memory usage %.2f MB exceeds 1GB target", memoryUsed)
	}

	// Verify we can create a machine
	startTime := time.Now()
	machine, err := NewMachine(root)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}
	
	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	err = rt.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}
	startupTime := time.Since(startTime)
	t.Logf("Startup time: %v", startupTime)
	
	rt.Stop()
}

// TestMillionEvents processes 1 million events and validates throughput
func TestMillionEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Log("Starting TestMillionEvents...")

	// Create a simple statechart with a few states
	const (
		STATE1 StateID = 1
		STATE2 StateID = 2
		STATE3 StateID = 3
		EVENT1 EventID = 1
		EVENT2 EventID = 2
		EVENT3 EventID = 3
	)

	state1 := &State{ID: STATE1, Transitions: []*Transition{}}
	state2 := &State{ID: STATE2, Transitions: []*Transition{}}
	state3 := &State{ID: STATE3, Transitions: []*Transition{}}

	state1.Transitions = append(state1.Transitions, &Transition{Event: EVENT1, Source: state1, Target: STATE2})
	state2.Transitions = append(state2.Transitions, &Transition{Event: EVENT2, Source: state2, Target: STATE3})
	state3.Transitions = append(state3.Transitions, &Transition{Event: EVENT3, Source: state3, Target: STATE1})

	root := &State{
		ID:       100,
		Initial:  STATE1,
		Children: map[StateID]*State{STATE1: state1, STATE2: state2, STATE3: state3},
	}
	state1.Parent = root
	state2.Parent = root
	state3.Parent = root

	machine, err := NewMachine(root)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	err = rt.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}
	defer rt.Stop()

	// Process 1 million events
	const numEvents = 1000000
	events := []EventID{EVENT1, EVENT2, EVENT3}

	start := time.Now()
	for i := 0; i < numEvents; i++ {
		eventID := events[i%3]
		rt.SendEvent(ctx, Event{ID: eventID})
	}
	duration := time.Since(start)

	throughput := float64(numEvents) / duration.Seconds()
	avgTime := duration / numEvents

	t.Logf("Processed %d events in %v", numEvents, duration)
	t.Logf("Throughput: %.0f events/sec", throughput)
	t.Logf("Average time per event: %v", avgTime)

	// Validate performance target
	if throughput < 10000 {
		t.Errorf("Throughput %.0f events/sec is below 10K events/sec target", throughput)
	}
}

// TestMassiveParallelRegions creates 1,000 parallel regions
func TestMassiveParallelRegions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Log("Starting TestMassiveParallelRegions...")

	const numRegions = 1000

	root := &State{
		ID:         1,
		IsParallel: true,
		Children:   make(map[StateID]*State),
	}

	stateID := StateID(2)
	const EVENT_TOGGLE EventID = 1

	// Create 1000 parallel regions, each with a few states
	for i := 0; i < numRegions; i++ {
		region := &State{
			ID:       stateID,
			Parent:   root,
			Children: make(map[StateID]*State),
		}
		root.Children[stateID] = region
		regionID := stateID
		stateID++

		state1 := &State{ID: stateID, Parent: region, Transitions: []*Transition{}}
		stateID++
		state2 := &State{ID: stateID, Parent: region, Transitions: []*Transition{}}
		stateID++

		state1.Transitions = append(state1.Transitions, &Transition{Event: EVENT_TOGGLE, Source: state1, Target: state2.ID})
		state2.Transitions = append(state2.Transitions, &Transition{Event: EVENT_TOGGLE, Source: state2, Target: state1.ID})

		region.Children[state1.ID] = state1
		region.Children[state2.ID] = state2
		region.Initial = state1.ID
		
		root.Children[regionID] = region
	}

	// Measure startup time
	start := time.Now()
	machine, err := NewMachine(root)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	err = rt.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}
	defer rt.Stop()
	
	startupTime := time.Since(start)

	t.Logf("Started %d parallel regions in %v", numRegions, startupTime)
	t.Logf("Average time per region: %v", startupTime/numRegions)

	// Validate performance target
	if startupTime > 5*time.Second {
		t.Errorf("Startup time %v exceeds 5s target", startupTime)
	}

	// Test event processing across all regions
	eventStart := time.Now()
	rt.SendEvent(ctx, Event{ID: EVENT_TOGGLE})
	eventTime := time.Since(eventStart)

	t.Logf("Event processed across %d regions in %v", numRegions, eventTime)
}

// TestDeepHierarchy creates a 1,000-level deep state hierarchy
func TestDeepHierarchy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Log("Starting TestDeepHierarchy...")

	const depth = 1000

	// Create deep hierarchy
	start := time.Now()
	
	root := &State{
		ID:       1,
		Children: make(map[StateID]*State),
	}
	
	currentState := root
	stateID := StateID(2)
	
	for i := 0; i < depth; i++ {
		child := &State{
			ID:       stateID,
			Parent:   currentState,
			Children: make(map[StateID]*State),
		}
		currentState.Children[stateID] = child
		currentState.Initial = stateID
		currentState = child
		stateID++
	}
	
	creationTime := time.Since(start)

	t.Logf("Created %d-level deep hierarchy in %v", depth, creationTime)

	// Start the statechart (should enter all nested states)
	startTime := time.Now()
	machine, err := NewMachine(root)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	err = rt.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}
	defer rt.Stop()
	
	startupTime := time.Since(startTime)

	t.Logf("Started deep hierarchy in %v", startupTime)

	// Test LCA computation at maximum depth
	lcaStart := time.Now()
	deepestState := currentState
	lca := computeLCA(deepestState, root)
	lcaTime := time.Since(lcaStart)

	t.Logf("LCA computation at depth %d took %v", depth, lcaTime)

	if lca == nil {
		t.Error("LCA computation failed")
	}
}

// TestConcurrentStateMachines runs 10,000 state machines simultaneously
func TestConcurrentStateMachines(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Log("Starting TestConcurrentStateMachines...")

	const numMachines = 10000
	const eventsPerMachine = 100

	// Record initial memory
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	initialAlloc := m1.Alloc

	// Create state machines
	start := time.Now()
	runtimes := make([]*Runtime, numMachines)
	
	const (
		STATE1 StateID = 1
		STATE2 StateID = 2
		EVENT_TOGGLE EventID = 1
	)
	
	for i := 0; i < numMachines; i++ {
		state1 := &State{ID: STATE1, Transitions: []*Transition{}}
		state2 := &State{ID: STATE2, Transitions: []*Transition{}}
		
		state1.Transitions = append(state1.Transitions, &Transition{Event: EVENT_TOGGLE, Source: state1, Target: STATE2})
		state2.Transitions = append(state2.Transitions, &Transition{Event: EVENT_TOGGLE, Source: state2, Target: STATE1})
		
		root := &State{
			ID:       100,
			Initial:  STATE1,
			Children: map[StateID]*State{STATE1: state1, STATE2: state2},
		}
		state1.Parent = root
		state2.Parent = root
		
		machine, err := NewMachine(root)
		if err != nil {
			t.Fatalf("Failed to create machine %d: %v", i, err)
		}
		
		rt := NewRuntime(machine, nil)
		runtimes[i] = rt
	}
	creationTime := time.Since(start)

	t.Logf("Created %d state machines in %v", numMachines, creationTime)

	// Start all machines
	startTime := time.Now()
	ctx := context.Background()
	for _, rt := range runtimes {
		if err := rt.Start(ctx); err != nil {
			t.Fatalf("Failed to start machine: %v", err)
		}
	}
	startupTime := time.Since(startTime)

	t.Logf("Started %d machines in %v", numMachines, startupTime)

	// Process events concurrently
	var wg sync.WaitGroup
	var totalEvents atomic.Int64

	eventStart := time.Now()
	for _, rt := range runtimes {
		wg.Add(1)
		go func(runtime *Runtime) {
			defer wg.Done()
			for j := 0; j < eventsPerMachine; j++ {
				runtime.SendEvent(ctx, Event{ID: EVENT_TOGGLE})
				totalEvents.Add(1)
			}
		}(rt)
	}

	wg.Wait()
	eventTime := time.Since(eventStart)

	total := totalEvents.Load()
	throughput := float64(total) / eventTime.Seconds()

	t.Logf("Processed %d events across %d machines in %v", total, numMachines, eventTime)
	t.Logf("Throughput: %.0f events/sec", throughput)

	// Measure memory while machines are still running
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	finalAlloc := m2.Alloc

	// Stop all machines
	for _, rt := range runtimes {
		rt.Stop()
	}

	// Handle case where GC reduced memory below initial allocation
	var memoryUsed float64
	if finalAlloc > initialAlloc {
		memoryUsed = float64(finalAlloc-initialAlloc) / (1024 * 1024) // MB
	} else {
		memoryUsed = -float64(initialAlloc-finalAlloc) / (1024 * 1024) // MB (negative indicates GC freed memory)
	}

	t.Logf("Memory used: %.2f MB (%.2f KB per machine)", memoryUsed, memoryUsed*1024/float64(numMachines))
}

// Helper function to compute LCA (Least Common Ancestor)
func computeLCA(state1, state2 *State) *State {
	// Build ancestor chains
	ancestors1 := make(map[*State]bool)
	for s := state1; s != nil; s = s.Parent {
		ancestors1[s] = true
	}

	// Find first common ancestor
	for s := state2; s != nil; s = s.Parent {
		if ancestors1[s] {
			return s
		}
	}

	return nil
}
