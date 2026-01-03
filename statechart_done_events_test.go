package statechartx

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// Test 1: Done event from sequential state with final child
func TestDoneEventSequentialState(t *testing.T) {
	t.Parallel()
	
	var doneEventReceived int32
	
	// Create states: Parent -> Child (final)
	finalChild := &State{
		ID:      2,
		IsFinal: true,
	}
	
	parent := &State{
		ID:       1,
		Initial:  2,
		Children: map[StateID]*State{2: finalChild},
	}

	// External state to transition back
	external := &State{
		ID: 3,
	}

	root := &State{
		ID:       0,
		Initial:  3,
		Children: map[StateID]*State{1: parent, 3: external},
		Transitions: []*Transition{
			{
				Event:  100,
				Target: 1, // enter parent
			},
			{
				Event:  DoneEventID(1), // done.state.1
				Target: 0,              // internal
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					atomic.StoreInt32(&doneEventReceived, 1)
					return nil
				},
			},
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
	
	time.Sleep(100 * time.Millisecond)
	
	// Send event to enter parent state (which has final child)
	if err := runtime.SendEvent(ctx, Event{ID: 100}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	
	// Wait for done event to be processed
	time.Sleep(300 * time.Millisecond)
	
	// Verify done event was received
	if atomic.LoadInt32(&doneEventReceived) != 1 {
		t.Error("Done event was not received for sequential state with final child")
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 2: Done event from parallel state when all regions complete
func TestDoneEventParallelStateAllRegions(t *testing.T) {
	t.Parallel()
	
	var doneEventReceived int32
	
	// Create parallel state with 3 regions, each with final state
	finalA := &State{ID: 11, IsFinal: true}
	finalB := &State{ID: 21, IsFinal: true}
	finalC := &State{ID: 31, IsFinal: true}
	
	initialA := &State{
		ID: 10,
		Transitions: []*Transition{
			{Event: 101, Target: 11},
		},
	}
	
	initialB := &State{
		ID: 20,
		Transitions: []*Transition{
			{Event: 102, Target: 21},
		},
	}
	
	initialC := &State{
		ID: 30,
		Transitions: []*Transition{
			{Event: 103, Target: 31},
		},
	}
	
	regionA := &State{
		ID:       1,
		Initial:  10,
		Children: map[StateID]*State{10: initialA, 11: finalA},
	}
	
	regionB := &State{
		ID:       2,
		Initial:  20,
		Children: map[StateID]*State{20: initialB, 21: finalB},
	}
	
	regionC := &State{
		ID:       3,
		Initial:  30,
		Children: map[StateID]*State{30: initialC, 31: finalC},
	}
	
	parallel := &State{
		ID:         100,
		IsParallel: true,
		Children:   map[StateID]*State{1: regionA, 2: regionB, 3: regionC},
	}
	
	root := &State{
		ID:       0,
		Initial:  100,
		Children: map[StateID]*State{100: parallel},
		Transitions: []*Transition{
			{
				Event:  DoneEventID(100), // done.state.100
				Target: 0,                // internal
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					atomic.StoreInt32(&doneEventReceived, 1)
					return nil
				},
			},
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
	
	time.Sleep(200 * time.Millisecond)
	
	// Transition region A to final
	if err := runtime.SendEvent(ctx, Event{ID: 101, Address: 1}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Should not have done event yet
	if atomic.LoadInt32(&doneEventReceived) != 0 {
		t.Error("Done event received too early (only 1/3 regions complete)")
	}
	
	// Transition region B to final
	if err := runtime.SendEvent(ctx, Event{ID: 102, Address: 2}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Still should not have done event
	if atomic.LoadInt32(&doneEventReceived) != 0 {
		t.Error("Done event received too early (only 2/3 regions complete)")
	}
	
	// Transition region C to final
	if err := runtime.SendEvent(ctx, Event{ID: 103, Address: 3}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	
	// Now done event should be received
	if atomic.LoadInt32(&doneEventReceived) != 1 {
		t.Error("Done event not received after all 3 regions completed")
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 3: Automatic transition triggered by done event
func TestDoneEventTriggersTransition(t *testing.T) {
	t.Parallel()
	
	var targetReached int32
	
	// State B (final child of A)
	stateB := &State{
		ID:      2,
		IsFinal: true,
	}
	
	// State A (parent with final child)
	stateA := &State{
		ID:       1,
		Initial:  2,
		Children: map[StateID]*State{2: stateB},
	}
	
	// State C (target of done transition)
	stateC := &State{
		ID: 3,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&targetReached, 1)
			return nil
		},
	}
	
	// Root with transition on done event
	root := &State{
		ID:       0,
		Initial:  1,
		Children: map[StateID]*State{1: stateA, 3: stateC},
		Transitions: []*Transition{
			{
				Event:  DoneEventID(1), // done.state.1
				Target: 3,              // transition to C
			},
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
	
	// Wait for automatic transition via done event
	time.Sleep(500 * time.Millisecond)
	
	// Verify we transitioned to state C
	if atomic.LoadInt32(&targetReached) != 1 {
		t.Error("Automatic transition on done event did not occur")
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 4: Done event with data payload
func TestDoneEventWithData(t *testing.T) {
	t.Parallel()
	
	var receivedData any
	
	// Final state with data
	finalState := &State{
		ID:             2,
		IsFinal:        true,
		FinalStateData: map[string]int{"result": 42},
	}
	
	parent := &State{
		ID:       1,
		Initial:  2,
		Children: map[StateID]*State{2: finalState},
	}
	
	root := &State{
		ID:       0,
		Initial:  1,
		Children: map[StateID]*State{1: parent},
		Transitions: []*Transition{
			{
				Event:  DoneEventID(1),
				Target: 0, // internal
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					receivedData = evt.Data
					return nil
				},
			},
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
	
	time.Sleep(300 * time.Millisecond)
	
	// Verify data was received
	if receivedData == nil {
		t.Error("Done event data was not received")
	} else if dataMap, ok := receivedData.(map[string]int); !ok {
		t.Error("Done event data has wrong type")
	} else if dataMap["result"] != 42 {
		t.Errorf("Done event data incorrect: expected 42, got %d", dataMap["result"])
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 5: Nested compound states with done events
func TestDoneEventNestedCompound(t *testing.T) {
	t.Parallel()
	
	var innerDone, outerDone int32
	
	// Innermost final state
	innerFinal := &State{
		ID:      3,
		IsFinal: true,
	}
	
	// Inner compound state
	innerCompound := &State{
		ID:       2,
		Initial:  3,
		Children: map[StateID]*State{3: innerFinal},
	}
	
	// Outer compound state
	outerCompound := &State{
		ID:       1,
		Initial:  2,
		Children: map[StateID]*State{2: innerCompound},
		Transitions: []*Transition{
			{
				Event:  DoneEventID(2), // done.state.2 (inner)
				Target: 0,              // internal
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					atomic.StoreInt32(&innerDone, 1)
					return nil
				},
			},
		},
	}
	
	root := &State{
		ID:       0,
		Initial:  1,
		Children: map[StateID]*State{1: outerCompound},
		Transitions: []*Transition{
			{
				Event:  DoneEventID(1), // done.state.1 (outer)
				Target: 0,              // internal
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					atomic.StoreInt32(&outerDone, 1)
					return nil
				},
			},
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
	
	time.Sleep(500 * time.Millisecond)
	
	// Both done events should be received
	if atomic.LoadInt32(&innerDone) != 1 {
		t.Error("Inner done event not received")
	}
	if atomic.LoadInt32(&outerDone) != 1 {
		t.Error("Outer done event not received")
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 6: Multiple done events in sequence
func TestDoneEventMultipleSequence(t *testing.T) {
	t.Parallel()
	
	var doneCount int32
	
	// Create multiple compound states with final children
	final1 := &State{ID: 11, IsFinal: true}
	final2 := &State{ID: 21, IsFinal: true}
	final3 := &State{ID: 31, IsFinal: true}
	
	compound1 := &State{
		ID:       10,
		Initial:  11,
		Children: map[StateID]*State{11: final1},
	}
	
	compound2 := &State{
		ID:       20,
		Initial:  21,
		Children: map[StateID]*State{21: final2},
	}
	
	compound3 := &State{
		ID:       30,
		Initial:  31,
		Children: map[StateID]*State{31: final3},
	}
	
	root := &State{
		ID:       0,
		Initial:  10,
		Children: map[StateID]*State{10: compound1, 20: compound2, 30: compound3},
		Transitions: []*Transition{
			{
				Event:  DoneEventID(10),
				Target: 20,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					atomic.AddInt32(&doneCount, 1)
					return nil
				},
			},
			{
				Event:  DoneEventID(20),
				Target: 30,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					atomic.AddInt32(&doneCount, 1)
					return nil
				},
			},
			{
				Event:  DoneEventID(30),
				Target: 0, // internal
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					atomic.AddInt32(&doneCount, 1)
					return nil
				},
			},
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
	
	// Wait for all done events to cascade
	time.Sleep(1 * time.Second)
	
	// All 3 done events should have been processed
	count := atomic.LoadInt32(&doneCount)
	if count != 3 {
		t.Errorf("Expected 3 done events, got %d", count)
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 7: Done event with guard
func TestDoneEventWithGuard(t *testing.T) {
	t.Parallel()
	
	var guardChecked, transitionTaken int32
	
	finalState := &State{
		ID:      2,
		IsFinal: true,
	}
	
	parent := &State{
		ID:       1,
		Initial:  2,
		Children: map[StateID]*State{2: finalState},
	}
	
	target := &State{
		ID: 3,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&transitionTaken, 1)
			return nil
		},
	}
	
	root := &State{
		ID:       0,
		Initial:  1,
		Children: map[StateID]*State{1: parent, 3: target},
		Transitions: []*Transition{
			{
				Event:  DoneEventID(1),
				Target: 3,
				Guard: func(ctx context.Context, evt *Event, from, to StateID) (bool, error) {
					atomic.StoreInt32(&guardChecked, 1)
					return true, nil // Allow transition
				},
			},
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
	
	time.Sleep(500 * time.Millisecond)
	
	// Verify guard was checked
	if atomic.LoadInt32(&guardChecked) != 1 {
		t.Error("Guard was not checked for done event transition")
	}
	
	// Verify transition was taken
	if atomic.LoadInt32(&transitionTaken) != 1 {
		t.Error("Transition was not taken after guard passed")
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 8: Done event in parallel regions
func TestDoneEventInParallelRegions(t *testing.T) {
	t.Parallel()
	
	var region1Done, region2Done int32
	
	// Region 1: compound with final child
	final1 := &State{ID: 12, IsFinal: true}
	compound1 := &State{
		ID:       11,
		Initial:  12,
		Children: map[StateID]*State{12: final1},
	}
	
	region1 := &State{
		ID:       10,
		Initial:  11,
		Children: map[StateID]*State{11: compound1},
		Transitions: []*Transition{
			{
				Event:  DoneEventID(11),
				Target: 0, // internal
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					atomic.StoreInt32(&region1Done, 1)
					return nil
				},
			},
		},
	}
	
	// Region 2: compound with final child
	final2 := &State{ID: 22, IsFinal: true}
	compound2 := &State{
		ID:       21,
		Initial:  22,
		Children: map[StateID]*State{22: final2},
	}
	
	region2 := &State{
		ID:       20,
		Initial:  21,
		Children: map[StateID]*State{21: compound2},
		Transitions: []*Transition{
			{
				Event:  DoneEventID(21),
				Target: 0, // internal
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					atomic.StoreInt32(&region2Done, 1)
					return nil
				},
			},
		},
	}
	
	parallel := &State{
		ID:         1,
		IsParallel: true,
		Children:   map[StateID]*State{10: region1, 20: region2},
	}
	
	root := &State{
		ID:       0,
		Initial:  1,
		Children: map[StateID]*State{1: parallel},
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
	
	time.Sleep(500 * time.Millisecond)
	
	// Both regions should have received their done events
	if atomic.LoadInt32(&region1Done) != 1 {
		t.Error("Region 1 done event not received")
	}
	if atomic.LoadInt32(&region2Done) != 1 {
		t.Error("Region 2 done event not received")
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 9: Concurrent done event generation (stress test)
func TestDoneEventConcurrentGeneration(t *testing.T) {
	t.Parallel()
	
	var doneCount int32
	
	// Create 5 parallel regions, each with final state
	regions := make(map[StateID]*State)
	for i := 0; i < 5; i++ {
		finalState := &State{
			ID:      StateID(100 + i),
			IsFinal: true,
		}
		
		region := &State{
			ID:       StateID(10 + i),
			Initial:  StateID(100 + i),
			Children: map[StateID]*State{StateID(100 + i): finalState},
		}
		
		regions[StateID(10+i)] = region
	}
	
	parallel := &State{
		ID:         1,
		IsParallel: true,
		Children:   regions,
	}
	
	root := &State{
		ID:       0,
		Initial:  1,
		Children: map[StateID]*State{1: parallel},
		Transitions: []*Transition{
			{
				Event:  DoneEventID(1),
				Target: 0, // internal
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					atomic.AddInt32(&doneCount, 1)
					return nil
				},
			},
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
	
	// Wait for all regions to complete and done event to be generated
	time.Sleep(1 * time.Second)
	
	// Exactly one done event should be generated
	count := atomic.LoadInt32(&doneCount)
	if count != 1 {
		t.Errorf("Expected exactly 1 done event, got %d", count)
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}
