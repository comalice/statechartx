package statechartx

import (
	"context"
	"testing"
)

func TestSCXML500(t *testing.T) {
	t.Skipf("SCXML500: requires datamodel (scxmlEventIOLocation variable)")
}

func TestSCXML501(t *testing.T) {
	t.Skipf("SCXML501: requires send with targetVar, delays, external event queues")
}

func TestSCXML503(t *testing.T) {
	root := &State{ID: "root"}
	s1 := &State{ID: "s1", Parent: root}
	s2 := &State{ID: "s2", Parent: root}
	s3 := &State{ID: "s3", Parent: root}
	pass := &State{ID: "pass", Parent: root}
	fail := &State{ID: "fail", Parent: root}

	root.Children = map[StateID]*State{
		"s1":   s1,
		"s2":   s2,
		"s3":   s3,
		"pass": pass,
		"fail": fail,
	}
	root.Initial = s1

	exitCount := 0

	s1.Transitions = []*Transition{
		{Target: "s2"},
	}

	s2.OnExit = func(ctx context.Context, event Event, from, to StateID, ext any) {
		exitCount++
	}

	transitionCount := 0
	s2.Transitions = []*Transition{
		{
			Event: "foo",
			Action: func(ctx context.Context, event Event, from, to StateID, ext any) {
				transitionCount++
			},
		},
		{Event: "bar", Target: "s3",
			Guard: func(ctx context.Context, event Event, from, to StateID, ext any) bool {
				return transitionCount == 1
			},
		},
		{Event: "bar", Target: "fail"},
	}

	s3.Transitions = []*Transition{
		{
			Target: "pass",
			Guard: func(ctx context.Context, event Event, from, to StateID, ext any) bool {
				return exitCount == 1
			},
		},
		{Target: "fail"},
	}

	machine, _ := NewMachine(root)

	rt := NewRuntime(machine, nil)
	s1.OnEntry = func(ctx context.Context, event Event, from, to StateID, ext any) {
		rt.SendEvent(ctx, "foo")
		rt.SendEvent(ctx, "bar")
	}
	ctx := context.Background()

	rt.Start(ctx)
	defer rt.Stop()

	if !rt.IsInState("pass") {
		t.Error("should reach pass state")
	}
}

func TestSCXML504(t *testing.T) {
	t.Skipf("SCXML504: requires parallel states (not supported)")
}

func TestSCXML505(t *testing.T) {
	root := &State{ID: "root"}
	s1 := &State{ID: "s1", Parent: root}
	s11 := &State{ID: "s11", Parent: s1}
	s2 := &State{ID: "s2", Parent: root}
	s3 := &State{ID: "s3", Parent: root}
	pass := &State{ID: "pass", Parent: root}
	fail := &State{ID: "fail", Parent: root}

	root.Children = map[StateID]*State{
		"s1":   s1,
		"s2":   s2,
		"s3":   s3,
		"pass": pass,
		"fail": fail,
	}
	s1.Children = map[StateID]*State{
		"s11": s11,
	}
	root.Initial = s1
	s1.Initial = s11

	s1ExitCount := 0
	s11ExitCount := 0
	transitionCount := 0

	s1.OnExit = func(ctx context.Context, event Event, from, to StateID, ext any) {
		s1ExitCount++
	}

	s11.OnExit = func(ctx context.Context, event Event, from, to StateID, ext any) {
		s11ExitCount++
	}

	s1.Transitions = []*Transition{
		{
			Event:  "foo",
			Target: "s11",
			Action: func(ctx context.Context, event Event, from, to StateID, ext any) {
				transitionCount++
			},
		},
		{Event: "bar", Target: "s2",
			Guard: func(ctx context.Context, event Event, from, to StateID, ext any) bool {
				return transitionCount == 1
			},
		},
		{Event: "bar", Target: "fail"},
	}

	s2.Transitions = []*Transition{
		{
			Target: "s3",
			Guard: func(ctx context.Context, event Event, from, to StateID, ext any) bool {
				return s1ExitCount == 1
			},
		},
		{Target: "fail"},
	}

	s3.Transitions = []*Transition{
		{
			Target: "pass",
			Guard: func(ctx context.Context, event Event, from, to StateID, ext any) bool {
				return s11ExitCount == 2
			},
		},
		{Target: "fail"},
	}

	machine, _ := NewMachine(root)

	rt := NewRuntime(machine, nil)
	s1.OnEntry = func(ctx context.Context, event Event, from, to StateID, ext any) {
		rt.SendEvent(ctx, "foo")
		rt.SendEvent(ctx, "bar")
	}
	ctx := context.Background()

	rt.Start(ctx)
	defer rt.Stop()

	if !rt.IsInState("pass") {
		t.Error("should reach pass state")
	}
}

func TestSCXML506(t *testing.T) {
	root := &State{ID: "root"}
	s1 := &State{ID: "s1", Parent: root}
	s2 := &State{ID: "s2", Parent: root}
	s21 := &State{ID: "s21", Parent: s2}
	s3 := &State{ID: "s3", Parent: root}
	s4 := &State{ID: "s4", Parent: root}
	pass := &State{ID: "pass", Parent: root}
	fail := &State{ID: "fail", Parent: root}

	root.Children = map[StateID]*State{
		"s1":   s1,
		"s2":   s2,
		"s3":   s3,
		"s4":   s4,
		"pass": pass,
		"fail": fail,
	}
	s2.Children = map[StateID]*State{
		"s21": s21,
	}
	root.Initial = s1
	s2.Initial = s21

	s2ExitCount := 0
	s21ExitCount := 0
	transitionCount := 0

	s1.Transitions = []*Transition{
		{Target: "s2"},
	}

	s2.OnExit = func(ctx context.Context, event Event, from, to StateID, ext any) {
		s2ExitCount++
	}

	s21.OnExit = func(ctx context.Context, event Event, from, to StateID, ext any) {
		s21ExitCount++
	}

	s2.Transitions = []*Transition{
		{
			Event:  "foo",
			Target: "s2",
			Action: func(ctx context.Context, event Event, from, to StateID, ext any) {
				transitionCount++
			},
		},
		{Event: "bar", Target: "s3",
			Guard: func(ctx context.Context, event Event, from, to StateID, ext any) bool {
				return transitionCount == 1
			},
		},
		{Event: "bar", Target: "fail"},
	}

	s3.Transitions = []*Transition{
		{
			Target: "s4",
			Guard: func(ctx context.Context, event Event, from, to StateID, ext any) bool {
				return s2ExitCount == 2
			},
		},
		{Target: "fail"},
	}

	s4.Transitions = []*Transition{
		{
			Target: "pass",
			Guard: func(ctx context.Context, event Event, from, to StateID, ext any) bool {
				return s21ExitCount == 2
			},
		},
		{Target: "fail"},
	}

	machine, _ := NewMachine(root)

	rt := NewRuntime(machine, nil)
	s1.OnEntry = func(ctx context.Context, event Event, from, to StateID, ext any) {
		rt.SendEvent(ctx, "foo")
		rt.SendEvent(ctx, "bar")
	}
	ctx := context.Background()

	rt.Start(ctx)
	defer rt.Stop()

	if !rt.IsInState("pass") {
		t.Error("should reach pass state")
	}
}

func TestSCXML509(t *testing.T) {
	t.Skipf("SCXML509: requires datamodel (variables, expr), <donedata> with content")
}

func TestSCXML510(t *testing.T) {
	t.Skipf("SCXML510: requires datamodel (variables, expr), <donedata> with param")
}

func TestSCXML518(t *testing.T) {
	t.Skipf("SCXML518: requires datamodel (variables, assign)")
}

func TestSCXML519(t *testing.T) {
	t.Skipf("SCXML519: requires datamodel (assign), error.execution event for invalid location")
}

func TestSCXML520(t *testing.T) {
	t.Skipf("SCXML520: requires datamodel (assign with expr)")
}

func TestSCXML521(t *testing.T) {
	t.Skipf("SCXML521: requires datamodel (assign), illegal assignment detection, error.execution events")
}

func TestSCXML522(t *testing.T) {
	t.Skipf("SCXML522: requires datamodel (variables, assign), invalid expr detection, error.execution events")
}

func TestSCXML525(t *testing.T) {
	t.Skipf("SCXML525: requires datamodel (variables, _event.name, assign)")
}

func TestSCXML527(t *testing.T) {
	t.Skipf("SCXML527: requires datamodel (variables, _event.data access)")
}

func TestSCXML528(t *testing.T) {
	t.Skipf("SCXML528: requires datamodel (variables, _event.sendid access), send with idlocation")
}

func TestSCXML529(t *testing.T) {
	t.Skipf("SCXML529: requires datamodel (variables, _event.origin access), send mechanics")
}

func TestSCXML530(t *testing.T) {
	t.Skipf("SCXML530: requires datamodel (variables, _event.origintype access), send type detection")
}

func TestSCXML531(t *testing.T) {
	t.Skipf("SCXML531: requires datamodel (variables, _event.invokeid access), invoke mechanics")
}

func TestSCXML532(t *testing.T) {
	t.Skipf("SCXML532: requires datamodel (variables, _event.type access), internal/external/platform event type detection")
}

func TestSCXML533(t *testing.T) {
	t.Skipf("SCXML533: requires datamodel (variables, assign), _event variable overwrite protection")
}

func TestSCXML534(t *testing.T) {
	t.Skipf("SCXML534: requires datamodel (variables, assign), _sessionid variable, overwrite protection")
}

func TestSCXML550(t *testing.T) {
	t.Skipf("SCXML550: requires datamodel (variables, expr, idVal guards), early binding")
}

func TestSCXML551(t *testing.T) {
	t.Skipf("SCXML551: requires datamodel (variables, expr, idVal guards), late binding")
}

func TestSCXML552(t *testing.T) {
	t.Skipf("SCXML552: requires datamodel (variables, assign, expr)")
}

func TestSCXML553(t *testing.T) {
	t.Skipf("SCXML553: requires datamodel (variables, cond guards with expressions)")
}

func TestSCXML554(t *testing.T) {
	t.Skipf("SCXML554: requires datamodel (variables, In() predicate function)")
}

func TestSCXML557(t *testing.T) {
	t.Skipf("SCXML557: requires datamodel (variables, _name system variable)")
}

func TestSCXML558(t *testing.T) {
	t.Skipf("SCXML558: requires invoke with type, src, idlocation, autoforward")
}

func TestSCXML560(t *testing.T) {
	t.Skipf("SCXML560: requires invoke, done.invoke events")
}

func TestSCXML561(t *testing.T) {
	t.Skipf("SCXML561: requires invoke with srcexpr, datamodel (variables)")
}

func TestSCXML562(t *testing.T) {
	t.Skipf("SCXML562: requires invoke with src attribute")
}

func TestSCXML567(t *testing.T) {
	t.Skipf("SCXML567: requires invoke, error.execution event for invalid src")
}

func TestSCXML569(t *testing.T) {
	t.Skipf("SCXML569: requires invoke, done.invoke events with event data")
}

func TestSCXML570(t *testing.T) {
	t.Skipf("SCXML570: requires invoke, <finalize> block for processing done.invoke data")
}

func TestSCXML576(t *testing.T) {
	t.Skipf("SCXML576: requires parallel states, initial attribute with multiple states")
}

func TestSCXML577(t *testing.T) {
	t.Skipf("SCXML577: requires parallel states with multiple child states, entry order")
}

func TestSCXML578(t *testing.T) {
	t.Skipf("SCXML578: requires parallel states, exit order verification")
}

func TestSCXML579(t *testing.T) {
	t.Skipf("SCXML579: requires parallel states, initial attribute, simultaneous child activation")
}

func TestSCXML580(t *testing.T) {
	t.Skipf("SCXML580: requires parallel states, transitions from parallel regions")
}
