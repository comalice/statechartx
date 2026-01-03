package statechartx

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// TestMaxStates finds the maximum number of states before failure
func TestMaxStates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping breaking point test in short mode")
	}

	t.Log("Starting TestMaxStates - finding breaking point...")

	// Start with 100K states and double until failure
	startSize := 100000
	maxAttempts := 10

	for attempt := 0; attempt < maxAttempts; attempt++ {
		numStates := startSize * (1 << attempt) // 100K, 200K, 400K, 800K, ...

		t.Logf("Attempting %d states...", numStates)

		// Record memory before
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		root := &State{
			ID:       1,
			Children: make(map[StateID]*State),
		}

		start := time.Now()
		success := true

		// Create states in batches to avoid timeout
		const batchSize = 10000
		stateID := StateID(2)
		for i := 0; i < numStates; i += batchSize {
			for j := 0; j < batchSize && i+j < numStates; j++ {
				state := &State{
					ID:     stateID,
					Parent: root,
				}
				root.Children[stateID] = state
				stateID++
			}

			// Check if we're running out of memory
			if i%100000 == 0 && i > 0 {
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				if m.Alloc > 8*1024*1024*1024 { // 8GB limit
					t.Logf("Memory limit reached at %d states", i)
					success = false
					break
				}
			}
		}

		if !success {
			t.Logf("Failed at %d states due to memory constraints", numStates)
			break
		}

		duration := time.Since(start)

		// Try to start it
		root.Initial = StateID(2) // First child
		machine, err := NewMachine(root)
		if err != nil {
			t.Logf("Failed to create machine with %d states: %v", numStates, err)
			break
		}

		rt := NewRuntime(machine, nil)
		ctx := context.Background()
		err = rt.Start(ctx)
		if err != nil {
			t.Logf("Failed to start with %d states: %v", numStates, err)
			rt.Stop()
			break
		}
		rt.Stop()

		// Measure memory
		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		memoryUsed := float64(m2.Alloc-m1.Alloc) / (1024 * 1024 * 1024)

		t.Logf("✓ Successfully created and started %d states in %v (%.2f GB)", numStates, duration, memoryUsed)

		// If this took too long, stop
		if duration > 30*time.Second {
			t.Logf("Stopping test - creation time exceeds 30s")
			break
		}
	}
}

// TestMaxEventThroughput finds maximum sustainable event throughput
func TestMaxEventThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping breaking point test in short mode")
	}

	t.Log("Starting TestMaxEventThroughput...")

	const (
		STATE1 StateID = 1
		STATE2 StateID = 2
		EVENT_TOGGLE EventID = 1
	)

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
		t.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer rt.Stop()

	// Test increasing event loads
	eventCounts := []int{10000, 100000, 1000000, 5000000, 10000000}

	for _, numEvents := range eventCounts {
		t.Logf("Testing %d events...", numEvents)

		start := time.Now()
		for i := 0; i < numEvents; i++ {
			rt.SendEvent(ctx, Event{ID: EVENT_TOGGLE})
		}
		duration := time.Since(start)

		throughput := float64(numEvents) / duration.Seconds()
		avgTime := duration / time.Duration(numEvents)

		t.Logf("  Throughput: %.0f events/sec, Avg time: %v", throughput, avgTime)

		// If throughput drops significantly, we've found the limit
		if avgTime > 10*time.Microsecond {
			t.Logf("Throughput degradation detected at %d events", numEvents)
			break
		}
	}
}

// TestMaxParallelRegions finds maximum number of parallel regions
func TestMaxParallelRegions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping breaking point test in short mode")
	}

	t.Log("Starting TestMaxParallelRegions...")

	// Test increasing numbers of parallel regions
	regionCounts := []int{100, 500, 1000, 2000, 5000, 10000}

	for _, numRegions := range regionCounts {
		t.Logf("Testing %d parallel regions...", numRegions)

		root := &State{
			ID:         1,
			IsParallel: true,
			Children:   make(map[StateID]*State),
		}

		// Create parallel regions
		createStart := time.Now()
		stateID := StateID(2)
		for i := 0; i < numRegions; i++ {
			region := &State{
				ID:       stateID,
				Parent:   root,
				Children: make(map[StateID]*State),
			}
			stateID++
			
			state := &State{
				ID:     stateID,
				Parent: region,
			}
			stateID++
			
			region.Children[state.ID] = state
			region.Initial = state.ID
			root.Children[region.ID] = region
		}
		createTime := time.Since(createStart)

		// Try to start
		startTime := time.Now()
		machine, err := NewMachine(root)
		if err != nil {
			t.Logf("Failed to create machine with %d regions: %v", numRegions, err)
			break
		}

		rt := NewRuntime(machine, nil)
		ctx := context.Background()
		err = rt.Start(ctx)
		if err != nil {
			t.Logf("Failed to start with %d regions: %v", numRegions, err)
			rt.Stop()
			break
		}
		startDuration := time.Since(startTime)

		t.Logf("  Created in %v, Started in %v", createTime, startDuration)

		// Test event processing
		eventStart := time.Now()
		rt.SendEvent(ctx, Event{ID: 1})
		eventTime := time.Since(eventStart)

		t.Logf("  Event processing: %v", eventTime)

		rt.Stop()

		// If startup takes too long, we've found the practical limit
		if startDuration > 10*time.Second {
			t.Logf("Startup time exceeds 10s at %d regions", numRegions)
			break
		}
	}
}

// TestMaxHierarchyDepth finds maximum hierarchy depth
func TestMaxHierarchyDepth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping breaking point test in short mode")
	}

	t.Log("Starting TestMaxHierarchyDepth...")

	// Test increasing depths
	depths := []int{100, 500, 1000, 2000, 5000, 10000}

	for _, depth := range depths {
		t.Logf("Testing depth %d...", depth)

		root := &State{
			ID:       1,
			Children: make(map[StateID]*State),
		}

		// Create deep hierarchy
		createStart := time.Now()
		current := root
		stateID := StateID(2)
		
		for i := 0; i < depth; i++ {
			child := &State{
				ID:       stateID,
				Parent:   current,
				Children: make(map[StateID]*State),
			}
			current.Children[stateID] = child
			current.Initial = stateID
			current = child
			stateID++
		}
		createTime := time.Since(createStart)

		// Try to start (enters all nested states)
		startTime := time.Now()
		machine, err := NewMachine(root)
		if err != nil {
			t.Logf("Failed to create machine with depth %d: %v", depth, err)
			break
		}

		rt := NewRuntime(machine, nil)
		ctx := context.Background()
		err = rt.Start(ctx)
		if err != nil {
			t.Logf("Failed to start with depth %d: %v", depth, err)
			rt.Stop()
			break
		}
		startDuration := time.Since(startTime)

		t.Logf("  Created in %v, Started in %v", createTime, startDuration)

		// Test LCA computation
		lcaStart := time.Now()
		lca := computeLCA(current, root)
		lcaTime := time.Since(lcaStart)

		if lca == nil {
			t.Logf("LCA computation failed at depth %d", depth)
			rt.Stop()
			break
		}

		t.Logf("  LCA computation: %v", lcaTime)

		rt.Stop()

		// If operations take too long, we've found the practical limit
		if startDuration > 5*time.Second || lcaTime > 1*time.Second {
			t.Logf("Performance degradation at depth %d", depth)
			break
		}
	}
}

// TestMemoryPressure tests behavior under memory pressure
func TestMemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping breaking point test in short mode")
	}

	t.Log("Starting TestMemoryPressure...")

	// Create many statecharts to apply memory pressure
	var runtimes []*Runtime
	var m runtime.MemStats

	const (
		STATE1 StateID = 1
		STATE2 StateID = 2
	)

	for i := 0; ; i++ {
		root := &State{
			ID:       100,
			Initial:  STATE1,
			Children: make(map[StateID]*State),
		}

		// Create a moderately complex statechart
		for j := 0; j < 100; j++ {
			state := &State{
				ID:     StateID(j + 1),
				Parent: root,
			}
			root.Children[state.ID] = state
		}

		machine, err := NewMachine(root)
		if err != nil {
			t.Logf("Failed to create machine %d: %v", i, err)
			break
		}

		rt := NewRuntime(machine, nil)
		ctx := context.Background()
		if err := rt.Start(ctx); err != nil {
			t.Logf("Failed to start machine %d: %v", i, err)
			break
		}

		runtimes = append(runtimes, rt)

		// Check memory every 100 machines
		if i%100 == 0 {
			runtime.ReadMemStats(&m)
			memoryGB := float64(m.Alloc) / (1024 * 1024 * 1024)
			t.Logf("Created %d machines, Memory: %.2f GB", i, memoryGB)

			// Stop at 4GB to be safe
			if m.Alloc > 4*1024*1024*1024 {
				t.Logf("Stopping at 4GB memory usage with %d machines", i)
				break
			}
		}

		// Stop after 10,000 machines regardless
		if i >= 10000 {
			t.Logf("Reached 10,000 machines limit")
			break
		}
	}

	t.Logf("Total machines created: %d", len(runtimes))

	// Cleanup
	for _, rt := range runtimes {
		rt.Stop()
	}
}
