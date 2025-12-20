package statechartx_test

import (
	"context"
	"testing"

	. "github.com/comalice/statechartx"
)

// Test internal transition does transition: basic internal transition executes action, no state change.
func TestInternalTransitionDoesTransition(t *testing.T) {
	var actionCalled int

	action := Action(func(ctx context.Context, evt *Event, from, to StateID) error {
		actionCalled++
		return nil
	})

	s := &State{ID: 1}
	e := Event{ID: 1}
	s.On(e, nil, nil, &action)

	m, err := NewMachine(s)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}

	m.Send(ctx, e)

	if actionCalled != 1 {
		t.Errorf("expected action called 1 time, got %d", actionCalled)
	}
	if m.current.ID != 1 {
		t.Errorf("expected state unchanged (ID=1), got %d", m.current.ID)
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
		ID:          1,
		EntryAction: entryAction,
		ExitAction:  exitAction,
	}
	e := Event{ID: 1}
	s.On(e, nil /* target nil for internal */, nil, &action)

	m, err := NewMachine(s)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}

	if entryCalled != 1 {
		t.Errorf("expected entry called 1 time after Start, got %d", entryCalled)
	}
	if exitCalled != 0 {
		t.Errorf("expected exit called 0 times after Start, got %d", exitCalled)
	}
	if m.current.ID != 1 {
		t.Errorf("expected current state ID 1 after Start")
	}

	m.Send(ctx, e)

	if actionCalled != 1 {
		t.Errorf("expected action called 1 time, got %d", actionCalled)
	}
	if entryCalled != 1 {
		t.Errorf("expected entry called 1 time total (no re-entry), got %d", entryCalled)
	}
	if exitCalled != 0 {
		t.Errorf("expected exit called 0 times (no exit), got %d", exitCalled)
	}
	if m.current.ID != 1 {
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

	s := &State{ID: 1}
	e := Event{ID: 1}
	s.On(e, nil, &guardFalse, &action1)
	s.On(e, nil, nil /* guard true */, &action2)

	m, err := NewMachine(s)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	m.Start(ctx)
	m.Send(ctx, e)

	if action1Called != 0 {
		t.Error("first action (guard false) should not be called")
	}
	if action2Called != 1 {
		t.Error("second action (guard true) should be called")
	}
	if m.current.ID != 1 {
		t.Errorf("expected still in state 1, got %d", m.current.ID)
	}
}
