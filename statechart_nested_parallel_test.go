package statechartx

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Test 1: Two-level nested parallel states
func TestTwoLevelNestedParallel(t *testing.T) {
	t.Parallel()

	var entryCount, exitCount int32

	// Create nested parallel structure:
	// ParentParallel
	//   ├─ Region1 (parallel)
	//   │   ├─ Region1A
	//   │   └─ Region1B
	//   └─ Region2 (parallel)
	//       ├─ Region2A
	//       └─ Region2B

	region1A := &State{
		ID: 11,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.AddInt32(&entryCount, 1)
			return nil
		},
		ExitAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.AddInt32(&exitCount, 1)
			return nil
		},
	}

	region1B := &State{
		ID: 12,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.AddInt32(&entryCount, 1)
			return nil
		},
		ExitAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.AddInt32(&exitCount, 1)
			return nil
		},
	}

	region1 := &State{
		ID:         10,
		IsParallel: true,
		Children: map[StateID]*State{
			11: region1A,
			12: region1B,
		},
	}

	region2A := &State{
		ID: 21,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.AddInt32(&entryCount, 1)
			return nil
		},
		ExitAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.AddInt32(&exitCount, 1)
			return nil
		},
	}

	region2B := &State{
		ID: 22,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.AddInt32(&entryCount, 1)
			return nil
		},
		ExitAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.AddInt32(&exitCount, 1)
			return nil
		},
	}

	region2 := &State{
		ID:         20,
		IsParallel: true,
		Children: map[StateID]*State{
			21: region2A,
			22: region2B,
		},
	}

	parentParallel := &State{
		ID:         1,
		IsParallel: true,
		Children: map[StateID]*State{
			10: region1,
			20: region2,
		},
	}

	root := &State{
		ID:      0,
		Initial: 1,
		Children: map[StateID]*State{
			1: parentParallel,
		},
	}

	machine, err := NewMachine(root)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	runtime := NewRuntime(machine, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runtime.Start(ctx); err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}

	// Wait for all regions to start
	time.Sleep(200 * time.Millisecond)

	// Verify all 4 leaf regions entered (2 in region1, 2 in region2)
	if count := atomic.LoadInt32(&entryCount); count != 4 {
		t.Errorf("Expected 4 entry actions, got %d", count)
	}

	// Send broadcast event to all regions
	var eventCount int32
	broadcastEvent := Event{
		ID:      100,
		Address: 0, // broadcast
	}

	// Add transitions to count events
	for _, state := range []*State{region1A, region1B, region2A, region2B} {
		state.Transitions = []*Transition{
			{
				Event:  100,
				Target: 0, // internal
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					atomic.AddInt32(&eventCount, 1)
					return nil
				},
			},
		}
	}

	if err := runtime.SendEvent(ctx, broadcastEvent); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Verify all 4 regions received the broadcast
	if count := atomic.LoadInt32(&eventCount); count != 4 {
		t.Errorf("Expected 4 regions to receive broadcast, got %d", count)
	}

	// Stop runtime and verify cleanup
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Verify all 4 leaf regions exited
	if count := atomic.LoadInt32(&exitCount); count != 4 {
		t.Errorf("Expected 4 exit actions, got %d", count)
	}
}

// Test 2: Three-level nested parallel states
func TestThreeLevelNestedParallel(t *testing.T) {
	t.Parallel()

	var entryCount int32

	// Create 3-level nested parallel structure
	// Level 1: 2 regions
	// Level 2: each has 2 regions (4 total)
	// Level 3: each has 2 regions (8 total leaf states)

	createLeafState := func(id StateID) *State {
		return &State{
			ID: id,
			EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
				atomic.AddInt32(&entryCount, 1)
				return nil
			},
		}
	}

	createParallelState := func(id StateID, children map[StateID]*State) *State {
		return &State{
			ID:         id,
			IsParallel: true,
			Children:   children,
		}
	}

	// Level 3 (leaf states)
	leaf111 := createLeafState(111)
	leaf112 := createLeafState(112)
	leaf121 := createLeafState(121)
	leaf122 := createLeafState(122)
	leaf211 := createLeafState(211)
	leaf212 := createLeafState(212)
	leaf221 := createLeafState(221)
	leaf222 := createLeafState(222)

	// Level 2 (parallel states with leaf children)
	region11 := createParallelState(11, map[StateID]*State{111: leaf111, 112: leaf112})
	region12 := createParallelState(12, map[StateID]*State{121: leaf121, 122: leaf122})
	region21 := createParallelState(21, map[StateID]*State{211: leaf211, 212: leaf212})
	region22 := createParallelState(22, map[StateID]*State{221: leaf221, 222: leaf222})

	// Level 1 (parallel states with parallel children)
	region1 := createParallelState(1, map[StateID]*State{11: region11, 12: region12})
	region2 := createParallelState(2, map[StateID]*State{21: region21, 22: region22})

	// Root (parallel state)
	root := createParallelState(0, map[StateID]*State{1: region1, 2: region2})

	machine, err := NewMachine(root)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	runtime := NewRuntime(machine, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := runtime.Start(ctx); err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}

	// Wait for all regions to start
	time.Sleep(500 * time.Millisecond)

	// Verify all 8 leaf regions entered
	if count := atomic.LoadInt32(&entryCount); count != 8 {
		t.Errorf("Expected 8 entry actions (2^3 leaf states), got %d", count)
	}

	// Send targeted event to deepest region
	var targetedEventCount int32
	leaf111.Transitions = []*Transition{
		{
			Event:  200,
			Target: 0, // internal
			Action: func(ctx context.Context, evt *Event, from, to StateID) error {
				atomic.AddInt32(&targetedEventCount, 1)
				return nil
			},
		},
	}

	targetedEvent := Event{
		ID:      200,
		Address: 111, // target specific leaf
	}

	if err := runtime.SendEvent(ctx, targetedEvent); err != nil {
		t.Fatalf("Failed to send targeted event: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Verify only target received event
	if count := atomic.LoadInt32(&targetedEventCount); count != 1 {
		t.Errorf("Expected 1 targeted event delivery, got %d", count)
	}

	// Cancel context and verify cleanup
	cancel()
	time.Sleep(500 * time.Millisecond)

	// Verify runtime stopped
	if err := runtime.Stop(); err != nil {
		t.Logf("Stop returned error (expected after cancel): %v", err)
	}
}

// Test 3: Event routing in nested parallel states
func TestNestedParallelEventRouting(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	eventLog := make([]string, 0)

	logEvent := func(regionID StateID, eventID EventID) {
		mu.Lock()
		defer mu.Unlock()
		eventLog = append(eventLog, fmt.Sprintf("%d:%d", regionID, eventID))
	}

	// Create 2-level nested parallel with 4 leaf regions
	leafA1 := &State{
		ID: 11,
		Transitions: []*Transition{
			{
				Event:  100,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					logEvent(11, evt.ID)
					return nil
				},
			},
			{
				Event:  101,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					logEvent(11, evt.ID)
					return nil
				},
			},
		},
	}

	leafA2 := &State{
		ID: 12,
		Transitions: []*Transition{
			{
				Event:  100,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					logEvent(12, evt.ID)
					return nil
				},
			},
			{
				Event:  101,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					logEvent(12, evt.ID)
					return nil
				},
			},
		},
	}

	leafB1 := &State{
		ID: 21,
		Transitions: []*Transition{
			{
				Event:  100,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					logEvent(21, evt.ID)
					return nil
				},
			},
			{
				Event:  101,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					logEvent(21, evt.ID)
					return nil
				},
			},
		},
	}

	leafB2 := &State{
		ID: 22,
		Transitions: []*Transition{
			{
				Event:  100,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					logEvent(22, evt.ID)
					return nil
				},
			},
			{
				Event:  101,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					logEvent(22, evt.ID)
					return nil
				},
			},
		},
	}

	regionA := &State{
		ID:         10,
		IsParallel: true,
		Children:   map[StateID]*State{11: leafA1, 12: leafA2},
	}

	regionB := &State{
		ID:         20,
		IsParallel: true,
		Children:   map[StateID]*State{21: leafB1, 22: leafB2},
	}

	root := &State{
		ID:         0,
		IsParallel: true,
		Children:   map[StateID]*State{10: regionA, 20: regionB},
	}

	machine, err := NewMachine(root)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	runtime := NewRuntime(machine, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runtime.Start(ctx); err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Test 1: Broadcast event (Address=0) - all 4 regions should receive
	eventLog = eventLog[:0] // clear log
	if err := runtime.SendEvent(ctx, Event{ID: 100, Address: 0}); err != nil {
		t.Fatalf("Failed to send broadcast event: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	if len(eventLog) != 4 {
		t.Errorf("Broadcast: expected 4 deliveries, got %d: %v", len(eventLog), eventLog)
	}
	mu.Unlock()

	// Test 2: Targeted event to specific leaf (Address=11)
	eventLog = eventLog[:0]
	if err := runtime.SendEvent(ctx, Event{ID: 101, Address: 11}); err != nil {
		t.Fatalf("Failed to send targeted event: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	if len(eventLog) != 1 {
		t.Errorf("Targeted: expected 1 delivery, got %d: %v", len(eventLog), eventLog)
	} else if eventLog[0] != "11:101" {
		t.Errorf("Targeted: expected '11:101', got '%s'", eventLog[0])
	}
	mu.Unlock()

	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 4: Cleanup on exit from nested parallel
func TestNestedParallelCleanupOrder(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	cleanupOrder := make([]StateID, 0)

	recordCleanup := func(id StateID) {
		mu.Lock()
		defer mu.Unlock()
		cleanupOrder = append(cleanupOrder, id)
	}

	// Create nested parallel structure
	leaf1 := &State{
		ID: 11,
		ExitAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			recordCleanup(11)
			return nil
		},
	}

	leaf2 := &State{
		ID: 12,
		ExitAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			recordCleanup(12)
			return nil
		},
	}

	childParallel := &State{
		ID:         10,
		IsParallel: true,
		Children:   map[StateID]*State{11: leaf1, 12: leaf2},
		ExitAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			recordCleanup(10)
			return nil
		},
	}

	leaf3 := &State{
		ID: 21,
		ExitAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			recordCleanup(21)
			return nil
		},
	}

	region2 := &State{
		ID:       20,
		Initial:  21,
		Children: map[StateID]*State{21: leaf3},
		ExitAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			recordCleanup(20)
			return nil
		},
	}

	parentParallel := &State{
		ID:         1,
		IsParallel: true,
		Children:   map[StateID]*State{10: childParallel, 20: region2},
		ExitAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			recordCleanup(1)
			return nil
		},
	}

	root := &State{
		ID:       0,
		Initial:  1,
		Children: map[StateID]*State{1: parentParallel},
	}

	machine, err := NewMachine(root)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	runtime := NewRuntime(machine, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runtime.Start(ctx); err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Stop runtime to trigger cleanup
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Verify cleanup order: children before parents
	mu.Lock()
	defer mu.Unlock()

	if len(cleanupOrder) < 3 {
		t.Errorf("Expected at least 3 cleanup calls, got %d: %v", len(cleanupOrder), cleanupOrder)
		return
	}

	// Verify leaf states cleaned up before parent parallel
	parentIdx := -1
	for i, id := range cleanupOrder {
		if id == 1 {
			parentIdx = i
			break
		}
	}

	if parentIdx == -1 {
		t.Errorf("Parent parallel state (1) not found in cleanup order")
		return
	}

	// Check that child states (10, 11, 12, 20, 21) appear before parent (1)
	for i := 0; i < parentIdx; i++ {
		id := cleanupOrder[i]
		if id != 10 && id != 11 && id != 12 && id != 20 && id != 21 {
			t.Errorf("Unexpected state %d in cleanup before parent", id)
		}
	}
}

// Test 5: Race conditions in nested parallel
func TestNestedParallelRaceConditions(t *testing.T) {
	t.Parallel()

	var counter int32

	// Create nested parallel with shared counter access
	createLeafWithCounter := func(id StateID) *State {
		return &State{
			ID: id,
			Transitions: []*Transition{
				{
					Event:  100,
					Target: 0,
					Action: func(ctx context.Context, evt *Event, from, to StateID) error {
						// Increment counter 100 times
						for i := 0; i < 100; i++ {
							atomic.AddInt32(&counter, 1)
						}
						return nil
					},
				},
			},
		}
	}

	// Create 2-level nested parallel (4 leaf states)
	leaf11 := createLeafWithCounter(11)
	leaf12 := createLeafWithCounter(12)
	leaf21 := createLeafWithCounter(21)
	leaf22 := createLeafWithCounter(22)

	region1 := &State{
		ID:         10,
		IsParallel: true,
		Children:   map[StateID]*State{11: leaf11, 12: leaf12},
	}

	region2 := &State{
		ID:         20,
		IsParallel: true,
		Children:   map[StateID]*State{21: leaf21, 22: leaf22},
	}

	root := &State{
		ID:         0,
		IsParallel: true,
		Children:   map[StateID]*State{10: region1, 20: region2},
	}

	machine, err := NewMachine(root)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	runtime := NewRuntime(machine, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runtime.Start(ctx); err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Send broadcast event to all regions
	if err := runtime.SendEvent(ctx, Event{ID: 100, Address: 0}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify counter: 4 regions * 100 increments = 400
	finalCount := atomic.LoadInt32(&counter)
	if finalCount != 400 {
		t.Errorf("Expected counter=400, got %d", finalCount)
	}

	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 6: Broadcast events in nested parallel
func TestNestedParallelBroadcastEvents(t *testing.T) {
	t.Parallel()

	var receivedCount int32

	// Create 3-level nested parallel (8 leaf states)
	createLeaf := func(id StateID) *State {
		return &State{
			ID: id,
			Transitions: []*Transition{
				{
					Event:  100,
					Target: 0,
					Action: func(ctx context.Context, evt *Event, from, to StateID) error {
						atomic.AddInt32(&receivedCount, 1)
						return nil
					},
				},
			},
		}
	}

	createParallel := func(id StateID, children map[StateID]*State) *State {
		return &State{
			ID:         id,
			IsParallel: true,
			Children:   children,
		}
	}

	// Build 3-level structure
	leaves := make(map[int]*State)
	for i := 0; i < 8; i++ {
		leaves[i] = createLeaf(StateID(100 + i))
	}

	level2_1 := createParallel(11, map[StateID]*State{100: leaves[0], 101: leaves[1]})
	level2_2 := createParallel(12, map[StateID]*State{102: leaves[2], 103: leaves[3]})
	level2_3 := createParallel(21, map[StateID]*State{104: leaves[4], 105: leaves[5]})
	level2_4 := createParallel(22, map[StateID]*State{106: leaves[6], 107: leaves[7]})

	level1_1 := createParallel(1, map[StateID]*State{11: level2_1, 12: level2_2})
	level1_2 := createParallel(2, map[StateID]*State{21: level2_3, 22: level2_4})

	root := createParallel(0, map[StateID]*State{1: level1_1, 2: level1_2})

	machine, err := NewMachine(root)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	runtime := NewRuntime(machine, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := runtime.Start(ctx); err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Send broadcast event
	if err := runtime.SendEvent(ctx, Event{ID: 100, Address: 0}); err != nil {
		t.Fatalf("Failed to send broadcast: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify all 8 leaf states received the event
	count := atomic.LoadInt32(&receivedCount)
	if count != 8 {
		t.Errorf("Expected 8 broadcast deliveries, got %d", count)
	}

	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 7: Targeted events in nested parallel
func TestNestedParallelTargetedEvents(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	receivedBy := make([]StateID, 0)

	createLeaf := func(id StateID) *State {
		return &State{
			ID: id,
			Transitions: []*Transition{
				{
					Event:  100,
					Target: 0,
					Action: func(ctx context.Context, evt *Event, from, to StateID) error {
						mu.Lock()
						receivedBy = append(receivedBy, from)
						mu.Unlock()
						return nil
					},
				},
			},
		}
	}

	// Create 2-level nested parallel
	leaf11 := createLeaf(11)
	leaf12 := createLeaf(12)
	leaf21 := createLeaf(21)
	leaf22 := createLeaf(22)

	region1 := &State{
		ID:         10,
		IsParallel: true,
		Children:   map[StateID]*State{11: leaf11, 12: leaf12},
	}

	region2 := &State{
		ID:         20,
		IsParallel: true,
		Children:   map[StateID]*State{21: leaf21, 22: leaf22},
	}

	root := &State{
		ID:         0,
		IsParallel: true,
		Children:   map[StateID]*State{10: region1, 20: region2},
	}

	machine, err := NewMachine(root)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	runtime := NewRuntime(machine, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runtime.Start(ctx); err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Test targeted delivery to each leaf
	targets := []StateID{11, 12, 21, 22}
	for _, target := range targets {
		receivedBy = receivedBy[:0] // clear

		if err := runtime.SendEvent(ctx, Event{ID: 100, Address: target}); err != nil {
			t.Fatalf("Failed to send targeted event to %d: %v", target, err)
		}

		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		if len(receivedBy) != 1 {
			t.Errorf("Target %d: expected 1 delivery, got %d: %v", target, len(receivedBy), receivedBy)
		} else if receivedBy[0] != target {
			t.Errorf("Target %d: expected delivery to %d, got %d", target, target, receivedBy[0])
		}
		mu.Unlock()
	}

	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 8: Panic recovery in nested parallel
func TestNestedParallelPanicRecovery(t *testing.T) {
	t.Parallel()

	var panicRecovered, otherRegionOk int32

	// Create nested parallel where one region panics
	panicLeaf := &State{
		ID: 11,
		Transitions: []*Transition{
			{
				Event:  100,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					panic("intentional panic for testing")
				},
			},
		},
	}

	normalLeaf := &State{
		ID: 12,
		Transitions: []*Transition{
			{
				Event:  100,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					atomic.StoreInt32(&otherRegionOk, 1)
					return nil
				},
			},
			{
				Event:  101,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					atomic.StoreInt32(&panicRecovered, 1)
					return nil
				},
			},
		},
	}

	region1 := &State{
		ID:         10,
		IsParallel: true,
		Children:   map[StateID]*State{11: panicLeaf, 12: normalLeaf},
	}

	root := &State{
		ID:       0,
		Initial:  10,
		Children: map[StateID]*State{10: region1},
	}

	machine, err := NewMachine(root)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	runtime := NewRuntime(machine, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runtime.Start(ctx); err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Send event that triggers panic in one region
	if err := runtime.SendEvent(ctx, Event{ID: 100, Address: 0}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	// Verify other region still works
	if atomic.LoadInt32(&otherRegionOk) != 1 {
		t.Error("Other region did not process event after panic in sibling")
	}

	// Send another event to verify runtime still functional
	if err := runtime.SendEvent(ctx, Event{ID: 101, Address: 0}); err != nil {
		t.Fatalf("Failed to send event after panic: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Verify runtime recovered
	if atomic.LoadInt32(&panicRecovered) != 1 {
		t.Error("Runtime did not recover after panic")
	}

	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}
