package statechartx

import (
	"context"
	"testing"
)

func TestSCXML144(t *testing.T) {
	root := &State{ID: "root"}
	s0 := &State{ID: "s0", Parent: root}
	s1 := &State{ID: "s1", Parent: root}
	pass := &State{ID: "pass", Parent: root}

	root.Children = map[StateID]*State{
		"s0":   s0,
		"s1":   s1,
		"pass": pass,
	}
	root.Initial = s0

	s0.Transitions = []*Transition{
		{Event: "foo", Target: "s1"},
	}
	s1.Transitions = []*Transition{
		{Event: "bar", Target: "pass"},
	}

	rt := NewRuntime(root, nil)
	s0.OnEntry = func(ctx context.Context, event Event, from, to StateID, ext any) {
		rt.SendEvent(ctx, "foo")
	}
	s1.OnEntry = func(ctx context.Context, event Event, from, to StateID, ext any) {
		rt.SendEvent(ctx, "bar")
	}
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop(ctx)

	if !rt.IsInState("pass") {
		t.Error("should reach pass state")
	}
}

func TestSCXML147(t *testing.T) {
	root := &State{ID: "root"}
	s0 := &State{ID: "s0", Parent: root}
	pass := &State{ID: "pass", Parent: root}
	fail := &State{ID: "fail", Parent: root}

	root.Children = map[StateID]*State{
		"s0":   s0,
		"pass": pass,
		"fail": fail,
	}
	root.Initial = s0

	s0.Transitions = []*Transition{
		{Event: "bar", Target: "pass"},
		{Event: "*", Target: "fail"},
	}

	rt := NewRuntime(root, nil)
	s0.OnEntry = func(ctx context.Context, event Event, from, to StateID, ext any) {
		rt.SendEvent(ctx, "bar")
		rt.SendEvent(ctx, "bat")
	}
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop(ctx)

	if !rt.IsInState("pass") {
		t.Error("should reach pass state")
	}
}