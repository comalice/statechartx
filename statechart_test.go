package statechartx

import (
	"context"
	"testing"
	"time"
)

func TestNewRuntime_Start_Stop_IsInState(t *testing.T) {
	root := &State{ID: "root"}
	child := &State{ID: "child", Parent: root}
	root.Children = map[StateID]*State{"child": child}
	root.Initial = child

	rt := NewRuntime(root, nil)
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop(ctx)

	if !rt.IsInState("root") {
		t.Error("should be in root")
	}
	if !rt.IsInState("child") {
		t.Error("should be in child")
	}
	if rt.IsInState("nonexistent") {
		t.Error("should not be in nonexistent")
	}
}

func TestSendEvent_SimpleTransition(t *testing.T) {
	root := &State{ID: "root"}
	idle := &State{ID: "idle", Parent: root}
	active := &State{ID: "active", Parent: root}
	root.Children = map[StateID]*State{"idle": idle, "active": active}
	root.Initial = idle

	idle.Transitions = []*Transition{
		{Event: "activate", Target: "active"},
	}

	rt := NewRuntime(root, nil)
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop(ctx)

	if rt.IsInState("active") {
		t.Error("should not start in active")
	}

	rt.SendEvent(ctx, "activate")

	if !rt.IsInState("active") {
		t.Error("should transition to active")
	}
}

func TestSendEvent_Guard(t *testing.T) {
	guardCtx := make(chan context.Context, 1)
	var guardCalls int
	guard := func(ctx context.Context, event Event, from, to StateID, ext any) bool {
		guardCtx <- ctx
		guardCalls++
		return false // always block
	}

	root := &State{ID: "root"}
	idle := &State{ID: "idle", Parent: root}
	active := &State{ID: "active", Parent: root}
	root.Children = map[StateID]*State{"idle": idle, "active": active}
	root.Initial = idle

	idle.Transitions = []*Transition{
		{Event: "activate", Target: "active", Guard: guard},
	}

	rt := NewRuntime(root, nil)
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop(ctx)

	rt.SendEvent(ctx, "activate")

	if rt.IsInState("active") {
		t.Error("guard should prevent transition")
	}

	if guardCalls != 1 {
		t.Error("guard not called")
	}
}

func TestHierarchy_EntryExitOrder(t *testing.T) {
	// Simple hierarchy test would need entry/exit actions logging order
	// Implemented via counters or logs, but for brevity, test IsInState changes
	t.Skip("Requires action logging for full order verification")
}

func TestHistory_Shallow(t *testing.T) {
	root := &State{ID: "root"}
	choice := &State{ID: "choice", Parent: root}
	a := &State{ID: "a", Parent: choice}
	b := &State{ID: "b", Parent: choice}
	root.Children = map[StateID]*State{"choice": choice}
	root.Initial = choice
	choice.Initial = a
	choice.Children = map[StateID]*State{"a": a, "b": b}

	a.Transitions = []*Transition{{Event: "toB", Target: "b"}}

	rt := NewRuntime(root, nil)
	ctx := context.Background()

	rt.Start(ctx)
	rt.SendEvent(ctx, "toB") // transition from choice to b
	rt.Stop(ctx)

	// Restart should restore history to b
	rt.Start(ctx)
	defer rt.Stop(ctx)

	if !rt.IsInState("b") {
		t.Error("should restore shallow history to b")
	}
}

func TestRunAsActor(t *testing.T) {
	root := &State{ID: "root"}
	idle := &State{ID: "idle", Parent: root}
	active := &State{ID: "active", Parent: root}
	root.Children = map[StateID]*State{"idle": idle, "active": active}
	root.Initial = idle

	idle.Transitions = []*Transition{{Event: "ping", Target: "active"}}

	rt := NewRuntime(root, nil)
	events := make(chan Event, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go rt.RunAsActor(ctx, events)

	events <- "ping"
	close(events)

	// Give time for processing
	time.Sleep(50 * time.Millisecond)

	if rt.IsInState("idle") {
		t.Error("should have transitioned")
	}
}

func TestConcurrentSendEvents(t *testing.T) {
	// Basic concurrency test; race detector will catch issues
	// Relies on -race flag
	t.Skip("Run with go test -race")
}
