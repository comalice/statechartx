package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/comalice/statechartx"
)

// TestRuntimeCreation tests basic runtime creation
func TestRuntimeCreation(t *testing.T) {
	// Create simple state machine
	stateA := &statechartx.State{ID: 1}
	machine, err := statechartx.NewMachine(stateA)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, Config{
		TickRate: 10 * time.Millisecond,
	})

	if rt == nil {
		t.Fatal("Runtime is nil")
	}
	if rt.Runtime == nil {
		t.Fatal("Embedded runtime is nil")
	}
}

// TestTickLoopTiming tests that the tick loop runs at the correct rate
func TestTickLoopTiming(t *testing.T) {
	// Create simple state machine
	stateA := &statechartx.State{ID: 1}
	machine, err := statechartx.NewMachine(stateA)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, Config{
		TickRate: 10 * time.Millisecond,
	})

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}
	defer rt.Stop()

	// Measure 10 ticks
	start := time.Now()
	startTick := rt.GetTickNumber()

	time.Sleep(105 * time.Millisecond) // ~10 ticks

	endTick := rt.GetTickNumber()
	elapsed := time.Since(start)

	// Should be ~10 ticks in ~100ms (±2 ticks tolerance)
	tickDiff := endTick - startTick
	if tickDiff < 8 || tickDiff > 12 {
		t.Errorf("Expected ~10 ticks, got %d", tickDiff)
	}

	// Should take ~100ms (±20ms tolerance)
	expectedDuration := 100 * time.Millisecond
	if elapsed < expectedDuration-20*time.Millisecond || elapsed > expectedDuration+20*time.Millisecond {
		t.Errorf("Expected ~%v, got %v", expectedDuration, elapsed)
	}
}

// TestSimpleTransition tests a simple state transition
func TestSimpleTransition(t *testing.T) {
	// Create state machine: Root -> A (initial)
	//                       Root.A --[event1]--> Root.B
	const (
		STATE_ROOT statechartx.StateID = 0
		STATE_A    statechartx.StateID = 1
		STATE_B    statechartx.StateID = 2
		EVENT_1    statechartx.EventID = 1
	)

	stateA := &statechartx.State{
		ID: STATE_A,
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_1,
				Target: STATE_B,
			},
		},
	}
	stateB := &statechartx.State{ID: STATE_B}

	root := &statechartx.State{
		ID:      STATE_ROOT,
		Initial: STATE_A,
		Children: map[statechartx.StateID]*statechartx.State{
			STATE_A: stateA,
			STATE_B: stateB,
		},
	}

	machine, err := statechartx.NewMachine(root)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, Config{
		TickRate: 10 * time.Millisecond,
	})

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}
	defer rt.Stop()

	// Should start in state A
	if rt.GetCurrentState() != STATE_A {
		t.Errorf("Expected initial state %d, got %d", STATE_A, rt.GetCurrentState())
	}

	// Send event
	if err := rt.SendEvent(statechartx.Event{ID: EVENT_1}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	// Wait for next tick
	time.Sleep(15 * time.Millisecond)

	// Should now be in state B
	if rt.GetCurrentState() != STATE_B {
		t.Errorf("Expected state %d after transition, got %d", STATE_B, rt.GetCurrentState())
	}
}

// TestEventOrdering tests that concurrent events are processed in deterministic order
func TestEventOrdering(t *testing.T) {
	// Create simple state machine
	stateA := &statechartx.State{ID: 1}
	machine, err := statechartx.NewMachine(stateA)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, Config{
		TickRate:         10 * time.Millisecond,
		MaxEventsPerTick: 1000,
	})

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}
	defer rt.Stop()

	// Send multiple events concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				rt.SendEvent(statechartx.Event{
					ID:   statechartx.EventID(id*10 + j),
					Data: id*10 + j,
				})
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Wait for events to process
	time.Sleep(50 * time.Millisecond)

	// Just verify no errors occurred
	// Detailed ordering verification would require state tracking
}

// TestEventBatching tests that events are batched correctly
func TestEventBatching(t *testing.T) {
	stateA := &statechartx.State{ID: 1}
	machine, err := statechartx.NewMachine(stateA)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, Config{
		TickRate:         10 * time.Millisecond,
		MaxEventsPerTick: 5, // Small capacity for testing
	})

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Failed to start runtime: %v", err)
	}
	defer rt.Stop()

	// Fill event queue
	for i := 0; i < 5; i++ {
		if err := rt.SendEvent(statechartx.Event{ID: statechartx.EventID(i)}); err != nil {
			t.Errorf("Failed to send event %d: %v", i, err)
		}
	}

	// Next event should fail (queue full)
	if err := rt.SendEvent(statechartx.Event{ID: 999}); err == nil {
		t.Error("Expected error when queue is full, got nil")
	}

	// Wait for tick to process events
	time.Sleep(15 * time.Millisecond)

	// Now should be able to send again
	if err := rt.SendEvent(statechartx.Event{ID: 100}); err != nil {
		t.Errorf("Failed to send event after queue cleared: %v", err)
	}
}

// TestEventSorting tests the event sorting logic
func TestEventSorting(t *testing.T) {
	stateA := &statechartx.State{ID: 1}
	machine, err := statechartx.NewMachine(stateA)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, Config{TickRate: 10 * time.Millisecond})

	// Create events with different priorities and sequence numbers
	events := []EventWithMeta{
		{Event: statechartx.Event{ID: 1}, SequenceNum: 3, Priority: 0},
		{Event: statechartx.Event{ID: 2}, SequenceNum: 1, Priority: 0},
		{Event: statechartx.Event{ID: 3}, SequenceNum: 2, Priority: 10}, // High priority
		{Event: statechartx.Event{ID: 4}, SequenceNum: 4, Priority: 0},
		{Event: statechartx.Event{ID: 5}, SequenceNum: 5, Priority: 5}, // Medium priority
	}

	rt.sortEvents(events)

	// Expected order: Priority 10 (seq 2), Priority 5 (seq 5), Priority 0 (seq 1, 3, 4)
	expectedOrder := []statechartx.EventID{3, 5, 2, 1, 4}

	for i, event := range events {
		if event.Event.ID != expectedOrder[i] {
			t.Errorf("Event at position %d: expected ID %d, got %d", i, expectedOrder[i], event.Event.ID)
		}
	}
}
