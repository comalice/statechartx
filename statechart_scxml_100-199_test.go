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

func TestSCXML148(t *testing.T) {
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
		{Event: "baz", Target: "pass"},
		{Event: "*", Target: "fail"},
	}

	rt := NewRuntime(root, nil)
	s0.OnEntry = func(ctx context.Context, event Event, from, to StateID, ext any) {
		rt.SendEvent(ctx, "baz")
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

func TestSCXML149(t *testing.T) {
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
		{Event: "bat", Target: "pass"},
		{Event: "*", Target: "fail"},
	}

	rt := NewRuntime(root, nil)
	s0.OnEntry = func(ctx context.Context, event Event, from, to StateID, ext any) {
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

func TestSCXML150(t *testing.T) {
	t.Skipf("SCXML150: requires datamodel (foreach, variables)")
}

func TestSCXML151(t *testing.T) {
	t.Skipf("SCXML151: requires datamodel (foreach, variables)")
}

func TestSCXML152(t *testing.T) {
	t.Skipf("SCXML152: requires datamodel (foreach, error.execution, variables)")
}

func TestSCXML153(t *testing.T) {
	t.Skipf("SCXML153: requires datamodel (foreach, if/else, assign, variables)")
}

func TestSCXML155(t *testing.T) {
	t.Skipf("SCXML155: requires datamodel (foreach, variables, arithmetic)")
}

func TestSCXML156(t *testing.T) {
	t.Skipf("SCXML156: requires datamodel (foreach, assign, variables)")
}

func TestSCXML158(t *testing.T) {
	root := &State{ID: "root"}
	s0 := &State{ID: "s0", Parent: root}
	s1 := &State{ID: "s1", Parent: root}
	pass := &State{ID: "pass", Parent: root}
	fail := &State{ID: "fail", Parent: root}

	root.Children = map[StateID]*State{
		"s0":   s0,
		"s1":   s1,
		"pass": pass,
		"fail": fail,
	}
	root.Initial = s0

	s0.Transitions = []*Transition{
		{Event: "event1", Target: "s1"},
		{Event: "*", Target: "fail"},
	}
	s1.Transitions = []*Transition{
		{Event: "event2", Target: "pass"},
		{Event: "*", Target: "fail"},
	}

	rt := NewRuntime(root, nil)
	s0.OnEntry = func(ctx context.Context, event Event, from, to StateID, ext any) {
		rt.SendEvent(ctx, "event1")
		rt.SendEvent(ctx, "event2")
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

func TestSCXML159(t *testing.T) {
	t.Skipf("SCXML159: requires datamodel (send error handling, variables)")
}

func TestSCXML172(t *testing.T) {
	t.Skipf("SCXML172: requires datamodel (assign, send eventexpr, variables)")
}

func TestSCXML173(t *testing.T) {
	t.Skipf("SCXML173: requires datamodel (assign, send targetexpr, variables)")
}

func TestSCXML174(t *testing.T) {
	t.Skipf("SCXML174: requires datamodel (assign, send typeexpr, variables)")
}

func TestSCXML175(t *testing.T) {
	t.Skipf("SCXML175: requires datamodel (assign, send delayexpr, timed delays, variables)")
}

func TestSCXML176(t *testing.T) {
	t.Skipf("SCXML176: requires datamodel (send with param, event data, variables)")
}

func TestSCXML177(t *testing.T) {
	t.Skipf("SCXML177: test does not exist in W3C SCXML test suite")
}

func TestSCXML178(t *testing.T) {
	t.Skipf("SCXML178: requires send with param and manual log inspection")
}

func TestSCXML179(t *testing.T) {
	t.Skipf("SCXML179: requires send with content and event data validation")
}

func TestSCXML183(t *testing.T) {
	t.Skipf("SCXML183: requires <send> with idlocation (datamodel variable assignment for send ID)")
}

func TestSCXML185(t *testing.T) {
	t.Skipf("SCXML185: requires <send> with delay timing support")
}

func TestSCXML186(t *testing.T) {
	t.Skipf("SCXML186: requires <send> with delay, params, datamodel, assign, and event data validation")
}

func TestSCXML187(t *testing.T) {
	t.Skipf("SCXML187: requires <invoke>, delayed <send>, parent/child session communication, and timing")
}

func TestSCXML189(t *testing.T) {
	t.Skipf("SCXML189: requires internal vs external queue priority distinction (target='#_internal')")
}

func TestSCXML190(t *testing.T) {
	t.Skipf("SCXML190: requires send with #_scxml_sessionid, internal/external queue distinction")
}

func TestSCXML191(t *testing.T) {
	t.Skipf("SCXML191: requires invoke and send with #_parent target")
}

func TestSCXML192(t *testing.T) {
	t.Skipf("SCXML192: requires invoke with id, send to #_invokeid, done.invoke events")
}

func TestSCXML193(t *testing.T) {
	t.Skipf("SCXML193: requires send with type specification, internal/external queue distinction")
}

func TestSCXML194(t *testing.T) {
	t.Skipf("SCXML194: requires send with illegal target detection, error.execution events")
}

func TestSCXML198(t *testing.T) {
	t.Skipf("SCXML198: requires send default type detection, event origintype metadata")
}

func TestSCXML199(t *testing.T) {
	t.Skipf("SCXML199: requires send with invalid type detection, error.execution events")
}
