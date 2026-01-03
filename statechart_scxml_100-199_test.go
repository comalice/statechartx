package statechartx

import (
	"context"
	"testing"
	"time"
)

// Event and State ID constants for tests
const (
	EVENT_FOO EventID = 1
	EVENT_BAR EventID = 2
	EVENT_BAZ EventID = 3
	EVENT_BAT EventID = 4

	STATE_ROOT StateID = 100
	STATE_S0   StateID = 101
	STATE_S1   StateID = 102
	STATE_PASS StateID = 200
	STATE_FAIL StateID = 201
)

func TestSCXML144(t *testing.T) {
	root := &State{ID: STATE_ROOT}
	s0 := &State{ID: STATE_S0, Parent: root}
	s1 := &State{ID: STATE_S1, Parent: root}
	pass := &State{ID: STATE_PASS, Parent: root}

	root.Children = map[StateID]*State{
		STATE_S0:   s0,
		STATE_S1:   s1,
		STATE_PASS: pass,
	}
	root.Initial = STATE_S0

	s0.Transitions = []*Transition{
		{Event: EVENT_FOO, Target: STATE_S1},
	}
	s1.Transitions = []*Transition{
		{Event: EVENT_BAR, Target: STATE_PASS},
	}

	machine, _ := NewMachine(root)

	rt := NewRuntime(machine, nil)
	s0.OnEntry(func(ctx context.Context, event *Event, from, to StateID) error {
		rt.SendEvent(ctx, Event{ID: EVENT_FOO})
		return nil
	})
	s1.OnEntry(func(ctx context.Context, event *Event, from, to StateID) error {
		rt.SendEvent(ctx, Event{ID: EVENT_BAR})
		return nil
	})
	ctx := context.Background()

	rt.Start(ctx)
	time.Sleep(50 * time.Millisecond) // Give event loop time to process
	defer rt.Stop()

	if !rt.IsInState(STATE_PASS) {
		t.Error("should reach pass state")
	}
}

func TestSCXML147(t *testing.T) {
	root := &State{ID: STATE_ROOT}
	s0 := &State{ID: STATE_S0, Parent: root}
	pass := &State{ID: STATE_PASS, Parent: root}
	fail := &State{ID: STATE_FAIL, Parent: root}

	root.Children = map[StateID]*State{
		STATE_S0:   s0,
		STATE_PASS: pass,
		STATE_FAIL: fail,
	}
	root.Initial = STATE_S0

	s0.Transitions = []*Transition{
		{Event: EVENT_BAR, Target: STATE_PASS},
		{Event: ANY_EVENT, Target: STATE_FAIL},
	}

	machine, _ := NewMachine(root)

	rt := NewRuntime(machine, nil)
	s0.OnEntry(func(ctx context.Context, event *Event, from, to StateID) error {
		rt.SendEvent(ctx, Event{ID: EVENT_BAR})
		rt.SendEvent(ctx, Event{ID: EVENT_BAT})
		return nil
	})
	ctx := context.Background()

	rt.Start(ctx)
	time.Sleep(50 * time.Millisecond) // Give event loop time to process
	defer rt.Stop()

	if !rt.IsInState(STATE_PASS) {
		t.Error("should reach pass state")
	}
}

func TestSCXML148(t *testing.T) {
	root := &State{ID: STATE_ROOT}
	s0 := &State{ID: STATE_S0, Parent: root}
	pass := &State{ID: STATE_PASS, Parent: root}
	fail := &State{ID: STATE_FAIL, Parent: root}

	root.Children = map[StateID]*State{
		STATE_S0:   s0,
		STATE_PASS: pass,
		STATE_FAIL: fail,
	}
	root.Initial = STATE_S0

	s0.Transitions = []*Transition{
		{Event: EVENT_BAZ, Target: STATE_PASS},
		{Event: ANY_EVENT, Target: STATE_FAIL},
	}

	machine, _ := NewMachine(root)

	rt := NewRuntime(machine, nil)
	s0.OnEntry(func(ctx context.Context, event *Event, from, to StateID) error {
		rt.SendEvent(ctx, Event{ID: EVENT_FOO})
		rt.SendEvent(ctx, Event{ID: EVENT_BAR})
		rt.SendEvent(ctx, Event{ID: EVENT_BAZ})
		return nil
	})
	ctx := context.Background()

	rt.Start(ctx)
	time.Sleep(50 * time.Millisecond) // Give event loop time to process
	defer rt.Stop()

	if !rt.IsInState(STATE_FAIL) {
		t.Error("should reach fail state (first event foo should match wildcard)")
	}
}

func TestSCXML149(t *testing.T) {
	root := &State{ID: STATE_ROOT}
	s0 := &State{ID: STATE_S0, Parent: root}
	pass := &State{ID: STATE_PASS, Parent: root}
	fail := &State{ID: STATE_FAIL, Parent: root}

	root.Children = map[StateID]*State{
		STATE_S0:   s0,
		STATE_PASS: pass,
		STATE_FAIL: fail,
	}
	root.Initial = STATE_S0

	s0.Transitions = []*Transition{
		{Event: EVENT_FOO, Target: STATE_PASS},
		{Event: ANY_EVENT, Target: STATE_FAIL},
	}

	machine, _ := NewMachine(root)

	rt := NewRuntime(machine, nil)
	s0.OnEntry(func(ctx context.Context, event *Event, from, to StateID) error {
		rt.SendEvent(ctx, Event{ID: EVENT_FOO})
		return nil
	})
	ctx := context.Background()

	rt.Start(ctx)
	time.Sleep(50 * time.Millisecond) // Give event loop time to process
	defer rt.Stop()

	if !rt.IsInState(STATE_PASS) {
		t.Error("should reach pass state (specific event should match before wildcard)")
	}
}

func TestSCXML158(t *testing.T) {
	root := &State{ID: STATE_ROOT}
	s0 := &State{ID: STATE_S0, Parent: root}
	s1 := &State{ID: STATE_S1, Parent: root}
	pass := &State{ID: STATE_PASS, Parent: root}

	root.Children = map[StateID]*State{
		STATE_S0:   s0,
		STATE_S1:   s1,
		STATE_PASS: pass,
	}
	root.Initial = STATE_S0

	s0.Transitions = []*Transition{
		{Event: EVENT_FOO, Target: STATE_S1},
	}
	s1.Transitions = []*Transition{
		{Event: EVENT_BAR, Target: STATE_PASS},
	}

	machine, _ := NewMachine(root)

	rt := NewRuntime(machine, nil)
	s0.OnEntry(func(ctx context.Context, event *Event, from, to StateID) error {
		// Queue multiple events
		rt.SendEvent(ctx, Event{ID: EVENT_FOO})
		rt.SendEvent(ctx, Event{ID: EVENT_BAR})
		return nil
	})
	ctx := context.Background()

	rt.Start(ctx)
	time.Sleep(50 * time.Millisecond) // Give event loop time to process
	defer rt.Stop()

	if !rt.IsInState(STATE_PASS) {
		t.Error("should reach pass state (events processed in order)")
	}
}
