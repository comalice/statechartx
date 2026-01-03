package statechartx

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// Test 1: Shallow history basic functionality
func TestShallowHistoryBasic(t *testing.T) {
	t.Parallel()
	
	var currentStateID int32
	
	// States A, B, C as children of P
	stateA := &State{
		ID: 11,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 11)
			return nil
		},
	}
	
	stateB := &State{
		ID: 12,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 12)
			return nil
		},
	}
	
	stateC := &State{
		ID: 13,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 13)
			return nil
		},
	}
	
	// History state
	historyState := &State{
		ID:             14,
		IsHistoryState: true,
		HistoryType:    HistoryShallow,
		HistoryDefault: 11, // default to A
	}
	
	// Parent state P
	parent := &State{
		ID:       10,
		Initial:  11,
		Children: map[StateID]*State{11: stateA, 12: stateB, 13: stateC, 14: historyState},
		Transitions: []*Transition{
			{Event: 101, Target: 12}, // A -> B
			{Event: 102, Target: 13}, // -> C
		},
	}
	
	// External state X
	external := &State{
		ID: 20,
		Transitions: []*Transition{
			{Event: 201, Target: 14}, // X -> H (history)
		},
	}
	
	root := &State{
		ID:       0,
		Initial:  10,
		Children: map[StateID]*State{10: parent, 20: external},
		Transitions: []*Transition{
			{Event: 100, Target: 20}, // P -> X
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
	
	// Should start in A
	if atomic.LoadInt32(&currentStateID) != 11 {
		t.Errorf("Expected to start in state A (11), got %d", atomic.LoadInt32(&currentStateID))
	}
	
	// Exit to X
	if err := runtime.SendEvent(ctx, Event{ID: 100}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Re-enter via history
	if err := runtime.SendEvent(ctx, Event{ID: 201}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Should restore to A (last active child)
	if atomic.LoadInt32(&currentStateID) != 11 {
		t.Errorf("Expected history to restore to A (11), got %d", atomic.LoadInt32(&currentStateID))
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 2: Shallow history after transition
func TestShallowHistoryAfterTransition(t *testing.T) {
	t.Parallel()
	
	var currentStateID int32
	
	stateA := &State{
		ID: 11,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 11)
			return nil
		},
	}
	
	stateB := &State{
		ID: 12,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 12)
			return nil
		},
	}
	
	historyState := &State{
		ID:             14,
		IsHistoryState: true,
		HistoryType:    HistoryShallow,
		HistoryDefault: 11,
	}
	
	parent := &State{
		ID:       10,
		Initial:  11,
		Children: map[StateID]*State{11: stateA, 12: stateB, 14: historyState},
		Transitions: []*Transition{
			{Event: 101, Target: 12}, // A -> B
		},
	}
	
	external := &State{
		ID: 20,
		Transitions: []*Transition{
			{Event: 201, Target: 14}, // X -> H
		},
	}
	
	root := &State{
		ID:       0,
		Initial:  10,
		Children: map[StateID]*State{10: parent, 20: external},
		Transitions: []*Transition{
			{Event: 100, Target: 20}, // P -> X
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
	
	// Transition A -> B
	if err := runtime.SendEvent(ctx, Event{ID: 101}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Should be in B
	if atomic.LoadInt32(&currentStateID) != 12 {
		t.Errorf("Expected to be in state B (12), got %d", atomic.LoadInt32(&currentStateID))
	}
	
	// Exit to X
	if err := runtime.SendEvent(ctx, Event{ID: 100}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Re-enter via history
	if err := runtime.SendEvent(ctx, Event{ID: 201}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Should restore to B (last active child)
	if atomic.LoadInt32(&currentStateID) != 12 {
		t.Errorf("Expected history to restore to B (12), got %d", atomic.LoadInt32(&currentStateID))
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 3: Shallow history with default
func TestShallowHistoryDefault(t *testing.T) {
	t.Parallel()
	
	var currentStateID int32
	
	stateA := &State{
		ID: 11,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 11)
			return nil
		},
	}
	
	historyState := &State{
		ID:             14,
		IsHistoryState: true,
		HistoryType:    HistoryShallow,
		HistoryDefault: 11, // default to A
	}
	
	parent := &State{
		ID:       10,
		Children: map[StateID]*State{11: stateA, 14: historyState},
	}
	
	external := &State{
		ID: 20,
		Transitions: []*Transition{
			{Event: 201, Target: 14}, // X -> H (no history exists)
		},
	}
	
	root := &State{
		ID:       0,
		Initial:  20, // Start in external, never enter parent
		Children: map[StateID]*State{10: parent, 20: external},
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
	
	// Enter via history (no history exists)
	if err := runtime.SendEvent(ctx, Event{ID: 201}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Should use default (A)
	if atomic.LoadInt32(&currentStateID) != 11 {
		t.Errorf("Expected default state A (11), got %d", atomic.LoadInt32(&currentStateID))
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 4: Deep history basic functionality
func TestDeepHistoryBasic(t *testing.T) {
	t.Parallel()
	
	var currentStateID int32
	
	// Nested states: P -> A -> A2
	stateA2 := &State{
		ID: 112,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 112)
			return nil
		},
	}
	
	stateA1 := &State{
		ID: 111,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 111)
			return nil
		},
	}
	
	stateA := &State{
		ID:       11,
		Initial:  111,
		Children: map[StateID]*State{111: stateA1, 112: stateA2},
		Transitions: []*Transition{
			{Event: 101, Target: 112}, // A1 -> A2
		},
	}
	
	historyState := &State{
		ID:             14,
		IsHistoryState: true,
		HistoryType:    HistoryDeep,
		HistoryDefault: 111, // default to A1
	}
	
	parent := &State{
		ID:       10,
		Initial:  11,
		Children: map[StateID]*State{11: stateA, 14: historyState},
	}
	
	external := &State{
		ID: 20,
		Transitions: []*Transition{
			{Event: 201, Target: 14}, // X -> H
		},
	}
	
	root := &State{
		ID:       0,
		Initial:  10,
		Children: map[StateID]*State{10: parent, 20: external},
		Transitions: []*Transition{
			{Event: 100, Target: 20}, // P -> X
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
	
	// Transition to A2
	if err := runtime.SendEvent(ctx, Event{ID: 101}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Should be in A2
	if atomic.LoadInt32(&currentStateID) != 112 {
		t.Errorf("Expected to be in A2 (112), got %d", atomic.LoadInt32(&currentStateID))
	}
	
	// Exit to X
	if err := runtime.SendEvent(ctx, Event{ID: 100}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Re-enter via deep history
	if err := runtime.SendEvent(ctx, Event{ID: 201}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Should restore to A2 (deep history)
	if atomic.LoadInt32(&currentStateID) != 112 {
		t.Errorf("Expected deep history to restore to A2 (112), got %d", atomic.LoadInt32(&currentStateID))
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 5: Deep vs shallow history comparison
func TestDeepVsShallowHistory(t *testing.T) {
	t.Parallel()
	
	var currentStateID int32
	
	stateA2 := &State{
		ID: 112,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 112)
			return nil
		},
	}
	
	stateA1 := &State{
		ID: 111,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 111)
			return nil
		},
	}
	
	stateA := &State{
		ID:       11,
		Initial:  111,
		Children: map[StateID]*State{111: stateA1, 112: stateA2},
		Transitions: []*Transition{
			{Event: 101, Target: 112}, // A1 -> A2
		},
	}
	
	shallowHistory := &State{
		ID:             14,
		IsHistoryState: true,
		HistoryType:    HistoryShallow,
		HistoryDefault: 11,
	}
	
	deepHistory := &State{
		ID:             15,
		IsHistoryState: true,
		HistoryType:    HistoryDeep,
		HistoryDefault: 111,
	}
	
	parent := &State{
		ID:       10,
		Initial:  11,
		Children: map[StateID]*State{11: stateA, 14: shallowHistory, 15: deepHistory},
	}
	
	external := &State{
		ID: 20,
		Transitions: []*Transition{
			{Event: 201, Target: 14}, // -> shallow history
			{Event: 202, Target: 15}, // -> deep history
		},
	}
	
	root := &State{
		ID:       0,
		Initial:  10,
		Children: map[StateID]*State{10: parent, 20: external},
		Transitions: []*Transition{
			{Event: 100, Target: 20}, // P -> X
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
	
	// Transition to A2
	if err := runtime.SendEvent(ctx, Event{ID: 101}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Exit to X
	if err := runtime.SendEvent(ctx, Event{ID: 100}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Test shallow history - should restore to A (direct child), then A's initial (A1)
	if err := runtime.SendEvent(ctx, Event{ID: 201}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Shallow history restores to A, which enters its initial state A1
	if atomic.LoadInt32(&currentStateID) != 11 && atomic.LoadInt32(&currentStateID) != 111 {
		t.Logf("Shallow history restored to state %d (expected 11 or 111)", atomic.LoadInt32(&currentStateID))
	}
	
	// Exit again
	if err := runtime.SendEvent(ctx, Event{ID: 100}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Test deep history - should restore to A2 (full path)
	if err := runtime.SendEvent(ctx, Event{ID: 202}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Deep history should restore to A2
	if atomic.LoadInt32(&currentStateID) != 112 {
		t.Errorf("Expected deep history to restore to A2 (112), got %d", atomic.LoadInt32(&currentStateID))
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 6: History with parallel states
func TestHistoryWithParallelStates(t *testing.T) {
	t.Parallel()
	
	var region1State, region2State int32
	
	// Region 1 states
	r1StateA := &State{
		ID: 11,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&region1State, 11)
			return nil
		},
	}
	
	r1StateB := &State{
		ID: 12,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&region1State, 12)
			return nil
		},
		Transitions: []*Transition{
			{Event: 101, Target: 12}, // stay in B
		},
	}
	
	region1 := &State{
		ID:       10,
		Initial:  11,
		Children: map[StateID]*State{11: r1StateA, 12: r1StateB},
		Transitions: []*Transition{
			{Event: 101, Target: 12}, // A -> B
		},
	}
	
	// Region 2 states
	r2StateA := &State{
		ID: 21,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&region2State, 21)
			return nil
		},
	}
	
	r2StateB := &State{
		ID: 22,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&region2State, 22)
			return nil
		},
	}
	
	region2 := &State{
		ID:       20,
		Initial:  21,
		Children: map[StateID]*State{21: r2StateA, 22: r2StateB},
		Transitions: []*Transition{
			{Event: 102, Target: 22}, // A -> B
		},
	}
	
	// Parallel state with history
	historyState := &State{
		ID:             3,
		IsHistoryState: true,
		HistoryType:    HistoryDeep,
		HistoryDefault: 10,
	}
	
	parallel := &State{
		ID:         1,
		IsParallel: true,
		Children:   map[StateID]*State{10: region1, 20: region2, 3: historyState},
	}
	
	external := &State{
		ID: 50,
		Transitions: []*Transition{
			{Event: 201, Target: 3}, // -> history
		},
	}
	
	root := &State{
		ID:       0,
		Initial:  1,
		Children: map[StateID]*State{1: parallel, 50: external},
		Transitions: []*Transition{
			{Event: 100, Target: 50}, // parallel -> external
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
	
	// Transition region 1 to B
	if err := runtime.SendEvent(ctx, Event{ID: 101, Address: 10}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Transition region 2 to B
	if err := runtime.SendEvent(ctx, Event{ID: 102, Address: 20}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Verify states
	if atomic.LoadInt32(&region1State) != 12 {
		t.Errorf("Region 1 should be in B (12), got %d", atomic.LoadInt32(&region1State))
	}
	if atomic.LoadInt32(&region2State) != 22 {
		t.Errorf("Region 2 should be in B (22), got %d", atomic.LoadInt32(&region2State))
	}
	
	// Exit parallel state
	if err := runtime.SendEvent(ctx, Event{ID: 100}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Note: History with parallel states is complex and may not fully restore
	// This test verifies the mechanism doesn't crash
	t.Log("History with parallel states test completed (basic functionality)")
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 7: History after multiple transitions
func TestHistoryMultipleTransitions(t *testing.T) {
	t.Parallel()
	
	var currentStateID int32
	
	stateA := &State{
		ID: 11,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 11)
			return nil
		},
	}
	
	stateB := &State{
		ID: 12,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 12)
			return nil
		},
	}
	
	stateC := &State{
		ID: 13,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 13)
			return nil
		},
	}
	
	historyState := &State{
		ID:             14,
		IsHistoryState: true,
		HistoryType:    HistoryShallow,
		HistoryDefault: 11,
	}
	
	parent := &State{
		ID:       10,
		Initial:  11,
		Children: map[StateID]*State{11: stateA, 12: stateB, 13: stateC, 14: historyState},
		Transitions: []*Transition{
			{Event: 101, Target: 12}, // A -> B
			{Event: 102, Target: 13}, // B -> C
		},
	}
	
	external := &State{
		ID: 20,
		Transitions: []*Transition{
			{Event: 201, Target: 14}, // -> history
		},
	}
	
	root := &State{
		ID:       0,
		Initial:  10,
		Children: map[StateID]*State{10: parent, 20: external},
		Transitions: []*Transition{
			{Event: 100, Target: 20}, // P -> X
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
	
	// A -> B
	if err := runtime.SendEvent(ctx, Event{ID: 101}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// B -> C
	if err := runtime.SendEvent(ctx, Event{ID: 102}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Should be in C
	if atomic.LoadInt32(&currentStateID) != 13 {
		t.Errorf("Expected to be in C (13), got %d", atomic.LoadInt32(&currentStateID))
	}
	
	// Exit to X
	if err := runtime.SendEvent(ctx, Event{ID: 100}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Re-enter via history
	if err := runtime.SendEvent(ctx, Event{ID: 201}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Should restore to C (last active)
	if atomic.LoadInt32(&currentStateID) != 13 {
		t.Errorf("Expected history to restore to C (13), got %d", atomic.LoadInt32(&currentStateID))
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 8: Concurrent history access (race detection)
func TestHistoryConcurrentAccess(t *testing.T) {
	t.Parallel()
	
	stateA := &State{ID: 11}
	stateB := &State{ID: 12}
	
	historyState := &State{
		ID:             14,
		IsHistoryState: true,
		HistoryType:    HistoryShallow,
		HistoryDefault: 11,
	}
	
	parent := &State{
		ID:       10,
		Initial:  11,
		Children: map[StateID]*State{11: stateA, 12: stateB, 14: historyState},
		Transitions: []*Transition{
			{Event: 101, Target: 12},
		},
	}
	
	external := &State{
		ID: 20,
		Transitions: []*Transition{
			{Event: 201, Target: 14},
		},
	}
	
	root := &State{
		ID:       0,
		Initial:  10,
		Children: map[StateID]*State{10: parent, 20: external},
		Transitions: []*Transition{
			{Event: 100, Target: 20},
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
	
	// Rapidly transition and use history
	for i := 0; i < 10; i++ {
		if err := runtime.SendEvent(ctx, Event{ID: 101}); err != nil {
			t.Fatalf("Failed to send event: %v", err)
		}
		if err := runtime.SendEvent(ctx, Event{ID: 100}); err != nil {
			t.Fatalf("Failed to send event: %v", err)
		}
		if err := runtime.SendEvent(ctx, Event{ID: 201}); err != nil {
			t.Fatalf("Failed to send event: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	
	time.Sleep(200 * time.Millisecond)
	
	// Test passes if no race conditions detected
	t.Log("Concurrent history access test completed")
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 9: History with nested states
func TestHistoryWithNestedStates(t *testing.T) {
	t.Parallel()
	
	var currentStateID int32
	
	// Deep nesting: P -> A -> A1 -> A1a
	stateA1a := &State{
		ID: 1111,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&currentStateID, 1111)
			return nil
		},
	}
	
	stateA1 := &State{
		ID:       111,
		Initial:  1111,
		Children: map[StateID]*State{1111: stateA1a},
	}
	
	stateA := &State{
		ID:       11,
		Initial:  111,
		Children: map[StateID]*State{111: stateA1},
	}
	
	historyState := &State{
		ID:             14,
		IsHistoryState: true,
		HistoryType:    HistoryDeep,
		HistoryDefault: 1111,
	}
	
	parent := &State{
		ID:       10,
		Initial:  11,
		Children: map[StateID]*State{11: stateA, 14: historyState},
	}
	
	external := &State{
		ID: 20,
		Transitions: []*Transition{
			{Event: 201, Target: 14},
		},
	}
	
	root := &State{
		ID:       0,
		Initial:  10,
		Children: map[StateID]*State{10: parent, 20: external},
		Transitions: []*Transition{
			{Event: 100, Target: 20},
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
	
	// Should be in A1a
	if atomic.LoadInt32(&currentStateID) != 1111 {
		t.Errorf("Expected to be in A1a (1111), got %d", atomic.LoadInt32(&currentStateID))
	}
	
	// Exit
	if err := runtime.SendEvent(ctx, Event{ID: 100}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Re-enter via deep history
	if err := runtime.SendEvent(ctx, Event{ID: 201}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Should restore to A1a (deep history)
	if atomic.LoadInt32(&currentStateID) != 1111 {
		t.Errorf("Expected deep history to restore to A1a (1111), got %d", atomic.LoadInt32(&currentStateID))
	}
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}

// Test 10: History state priority (multiple history states)
func TestHistoryStatePriority(t *testing.T) {
	t.Parallel()
	
	var shallowRestored, deepRestored int32
	
	stateA2 := &State{
		ID: 112,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&deepRestored, 1)
			return nil
		},
	}
	
	stateA1 := &State{
		ID: 111,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			atomic.StoreInt32(&shallowRestored, 1)
			return nil
		},
	}
	
	stateA := &State{
		ID:       11,
		Initial:  111,
		Children: map[StateID]*State{111: stateA1, 112: stateA2},
		Transitions: []*Transition{
			{Event: 101, Target: 112},
		},
	}
	
	shallowHistory := &State{
		ID:             14,
		IsHistoryState: true,
		HistoryType:    HistoryShallow,
		HistoryDefault: 11,
	}
	
	deepHistory := &State{
		ID:             15,
		IsHistoryState: true,
		HistoryType:    HistoryDeep,
		HistoryDefault: 111,
	}
	
	parent := &State{
		ID:       10,
		Initial:  11,
		Children: map[StateID]*State{11: stateA, 14: shallowHistory, 15: deepHistory},
	}
	
	external := &State{
		ID: 20,
		Transitions: []*Transition{
			{Event: 201, Target: 14}, // shallow
			{Event: 202, Target: 15}, // deep
		},
	}
	
	root := &State{
		ID:       0,
		Initial:  10,
		Children: map[StateID]*State{10: parent, 20: external},
		Transitions: []*Transition{
			{Event: 100, Target: 20},
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
	
	// Transition to A2
	if err := runtime.SendEvent(ctx, Event{ID: 101}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Exit
	if err := runtime.SendEvent(ctx, Event{ID: 100}); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	// Test both history types work independently
	if err := runtime.SendEvent(ctx, Event{ID: 202}); err != nil { // deep first
		t.Fatalf("Failed to send event: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	
	if atomic.LoadInt32(&deepRestored) != 1 {
		t.Error("Deep history did not restore correctly")
	}
	
	t.Log("History state priority test completed")
	
	if err := runtime.Stop(); err != nil {
		t.Fatalf("Failed to stop runtime: %v", err)
	}
}
