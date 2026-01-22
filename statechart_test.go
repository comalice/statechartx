package statechartx_test

import (
	"context"
	"testing"
	"time"

	. "github.com/comalice/statechartx"
)

const (
	STATE_1 StateID = 1
	EVENT_1 EventID = 1
)

// Test internal transition does transition: basic internal transition executes action, no state change.
func TestInternalTransitionDoesTransition(t *testing.T) {
	var actionCalled int

	action := Action(func(ctx context.Context, evt *Event, from, to StateID) error {
		actionCalled++
		return nil
	})

	s := &State{ID: STATE_1}
	s.On(EVENT_1, 0 /* internal transition */, nil, &action)

	m, err := NewMachine(s)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	rt.SendEvent(ctx, Event{ID: EVENT_1})

	// Give event loop time to process
	time.Sleep(50 * time.Millisecond)
	rt.Stop()

	if actionCalled != 1 {
		t.Errorf("expected action called 1 time, got %d", actionCalled)
	}
	if !rt.IsInState(STATE_1) {
		t.Errorf("expected state unchanged (ID=1)")
	}
}

// Test internal transition only execs action: verifies no entry/exit called.
func TestInternalTransitionExecsActionOnly(t *testing.T) {
	var actionCalled, entryCalled, exitCalled int

	action := Action(func(ctx context.Context, evt *Event, from, to StateID) error {
		actionCalled++
		return nil
	})
	entryAction := Action(func(ctx context.Context, evt *Event, from, to StateID) error {
		entryCalled++
		return nil
	})
	exitAction := Action(func(ctx context.Context, evt *Event, from, to StateID) error {
		exitCalled++
		return nil
	})

	s := &State{
		ID:          STATE_1,
		EntryAction: entryAction,
		ExitAction:  exitAction,
	}
	s.On(EVENT_1, 0 /* internal transition */, nil, &action)

	m, err := NewMachine(s)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}

	if entryCalled != 1 {
		t.Errorf("expected entry called 1 time after Start, got %d", entryCalled)
	}
	if exitCalled != 0 {
		t.Errorf("expected exit called 0 times after Start, got %d", exitCalled)
	}
	if !rt.IsInState(STATE_1) {
		t.Errorf("expected current state ID 1 after Start")
	}

	rt.SendEvent(ctx, Event{ID: EVENT_1})

	// Give event loop time to process
	time.Sleep(50 * time.Millisecond)
	rt.Stop()

	if actionCalled != 1 {
		t.Errorf("expected action called 1 time, got %d", actionCalled)
	}
	if entryCalled != 1 {
		t.Errorf("expected entry called 1 time total (no re-entry), got %d", entryCalled)
	}
	if exitCalled != 0 {
		t.Errorf("expected exit called 0 times (no exit), got %d", exitCalled)
	}
	if !rt.IsInState(STATE_1) {
		t.Errorf("expected still in state 1")
	}
}

func TestInternalPicksFirstEnabledTransition(t *testing.T) {
	var action1Called, action2Called int

	guardFalse := Guard(func(ctx context.Context, evt *Event, from, to StateID) (bool, error) {
		return false, nil
	})
	action1 := Action(func(ctx context.Context, evt *Event, from, to StateID) error {
		action1Called++
		return nil
	})
	action2 := Action(func(ctx context.Context, evt *Event, from, to StateID) error {
		action2Called++
		return nil
	})

	s := &State{ID: STATE_1}
	s.On(EVENT_1, 0 /* internal */, &guardFalse, &action1)
	s.On(EVENT_1, 0 /* internal */, nil /* guard true */, &action2)

	m, err := NewMachine(s)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}

	rt.SendEvent(ctx, Event{ID: EVENT_1})

	// Give event loop time to process
	time.Sleep(50 * time.Millisecond)
	rt.Stop()

	if action1Called != 0 {
		t.Error("first action (guard false) should not be called")
	}
	if action2Called != 1 {
		t.Error("second action (guard true) should be called")
	}
	if !rt.IsInState(STATE_1) {
		t.Errorf("expected still in state 1")
	}
}

// Phase 4 Additional Tests

func TestEventMatchingPrioritySpecificBeatsWildcard(t *testing.T) {
	// Test that specific event IDs beat ANY_EVENT wildcard
	const (
		STATE_ROOT StateID = 1
		STATE_S0   StateID = 2
		STATE_PASS StateID = 3
		STATE_FAIL StateID = 4
		EVENT_FOO  EventID = 1
	)

	root := &State{ID: STATE_ROOT}
	s0 := &State{ID: STATE_S0}
	pass := &State{ID: STATE_PASS, IsFinal: true}
	fail := &State{ID: STATE_FAIL, IsFinal: true}

	root.Children = map[StateID]*State{
		STATE_S0:   s0,
		STATE_PASS: pass,
		STATE_FAIL: fail,
	}
	root.Initial = STATE_S0

	machine, _ := NewMachine(root)
	rt := NewRuntime(machine, nil)
	ctx := context.Background()

	// Wildcard comes first in document order, but specific should still win
	s0.Transitions = []*Transition{
		{Event: ANY_EVENT, Target: STATE_FAIL}, // Wildcard
		{Event: EVENT_FOO, Target: STATE_PASS}, // Specific (should win)
	}

	s0.EntryAction = func(ctx context.Context, event *Event, from, to StateID) error {
		rt.SendEvent(ctx, Event{ID: EVENT_FOO})
		return nil
	}

	rt.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	defer rt.Stop()

	if !rt.IsInState(STATE_PASS) {
		t.Error("Specific event ID should beat ANY_EVENT wildcard")
	}
}

func TestEventlessTransitionBeatsEventDriven(t *testing.T) {
	// Test that eventless (NO_EVENT) transitions beat event-driven ones
	const (
		STATE_ROOT StateID = 1
		STATE_S0   StateID = 2
		STATE_PASS StateID = 3
		STATE_FAIL StateID = 4
		EVENT_FOO  EventID = 1
	)

	root := &State{ID: STATE_ROOT}
	s0 := &State{ID: STATE_S0}
	pass := &State{ID: STATE_PASS, IsFinal: true}
	fail := &State{ID: STATE_FAIL, IsFinal: true}

	root.Children = map[StateID]*State{
		STATE_S0:   s0,
		STATE_PASS: pass,
		STATE_FAIL: fail,
	}
	root.Initial = STATE_S0

	machine, _ := NewMachine(root)
	rt := NewRuntime(machine, nil)
	ctx := context.Background()

	s0.Transitions = []*Transition{
		{Event: NO_EVENT, Target: STATE_PASS},  // Eventless (should fire immediately)
		{Event: EVENT_FOO, Target: STATE_FAIL}, // Event-driven (should not fire)
	}

	s0.EntryAction = func(ctx context.Context, event *Event, from, to StateID) error {
		rt.SendEvent(ctx, Event{ID: EVENT_FOO})
		return nil
	}

	rt.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	defer rt.Stop()

	if !rt.IsInState(STATE_PASS) {
		t.Error("Eventless transition should fire before event-driven transitions")
	}
}

func TestFinalStateDetection(t *testing.T) {
	// Test that IsFinal flag is properly detected
	const (
		STATE_ROOT  StateID = 1
		STATE_S0    StateID = 2
		STATE_FINAL StateID = 3
		EVENT_GO    EventID = 1
	)

	root := &State{ID: STATE_ROOT}
	s0 := &State{ID: STATE_S0}
	final := &State{ID: STATE_FINAL, IsFinal: true}

	root.Children = map[StateID]*State{
		STATE_S0:    s0,
		STATE_FINAL: final,
	}
	root.Initial = STATE_S0

	machine, _ := NewMachine(root)
	rt := NewRuntime(machine, nil)
	ctx := context.Background()

	s0.Transitions = []*Transition{
		{Event: EVENT_GO, Target: STATE_FINAL},
	}

	rt.Start(ctx)
	rt.SendEvent(ctx, Event{ID: EVENT_GO})
	time.Sleep(50 * time.Millisecond)
	defer rt.Stop()

	if !rt.IsInState(STATE_FINAL) {
		t.Error("Should be in final state")
	}
}

func TestInitialActionWithMultipleLevels(t *testing.T) {
	// Test InitialAction with multiple levels of nesting
	const (
		STATE_ROOT StateID = 1
		STATE_S1   StateID = 2
		STATE_S11  StateID = 3
		STATE_S111 StateID = 4
	)

	root := &State{ID: STATE_ROOT}
	s1 := &State{ID: STATE_S1}
	s11 := &State{ID: STATE_S11}
	s111 := &State{ID: STATE_S111}

	root.Children = map[StateID]*State{STATE_S1: s1}
	root.Initial = STATE_S1

	s1.Children = map[StateID]*State{STATE_S11: s11}
	s1.Initial = STATE_S11

	s11.Children = map[StateID]*State{STATE_S111: s111}
	s11.Initial = STATE_S111

	machine, _ := NewMachine(root)
	rt := NewRuntime(machine, nil)
	ctx := context.Background()

	var executionOrder []string

	s1.EntryAction = func(ctx context.Context, event *Event, from, to StateID) error {
		executionOrder = append(executionOrder, "s1_entry")
		return nil
	}

	s1.InitialAction = func(ctx context.Context, event *Event, from, to StateID) error {
		executionOrder = append(executionOrder, "s1_initial")
		return nil
	}

	s11.EntryAction = func(ctx context.Context, event *Event, from, to StateID) error {
		executionOrder = append(executionOrder, "s11_entry")
		return nil
	}

	s11.InitialAction = func(ctx context.Context, event *Event, from, to StateID) error {
		executionOrder = append(executionOrder, "s11_initial")
		return nil
	}

	s111.EntryAction = func(ctx context.Context, event *Event, from, to StateID) error {
		executionOrder = append(executionOrder, "s111_entry")
		return nil
	}

	rt.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	defer rt.Stop()

	expected := []string{"s1_entry", "s1_initial", "s11_entry", "s11_initial", "s111_entry"}
	if len(executionOrder) != len(expected) {
		t.Errorf("Expected %d actions, got %d: %v", len(expected), len(executionOrder), executionOrder)
	}

	for i, v := range expected {
		if i >= len(executionOrder) || executionOrder[i] != v {
			t.Errorf("Expected %s at position %d, got %v", v, i, executionOrder)
			break
		}
	}
}
