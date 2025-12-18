package statechartx

import (
	"context"
	"testing"
)

// 300s SCXML tests: Mostly datamodel (assign/expr/var), guards expr, deep history, parallel, invoke, send.
// Few simple (355 initial order, 375/377 entry/exit order, 396 event match). All skipped or simple impl per limitations.

func TestSCXML301(t *testing.T) {
	t.Skipf("SCXML301: requires <script src>, load-time eval")
}

func TestSCXML302(t *testing.T) {
	t.Skipf("SCXML302: requires <script> executable content")
}

func TestSCXML303(t *testing.T) {
	t.Skipf("SCXML303: requires <script> var declaration")
}

func TestSCXML304(t *testing.T) {
	t.Skipf("SCXML304: requires <script> multiple scripts")
}

func TestSCXML307(t *testing.T) {
	t.Skipf("SCXML307: requires datamodel late binding, manual inspect")
}

func TestSCXML309(t *testing.T) {
	t.Skipf("SCXML309: requires non-boolean expr in conditions")
}

func TestSCXML310(t *testing.T) {
	t.Skipf("SCXML310: requires parallel regions")
}

func TestSCXML311(t *testing.T) {
	t.Skipf("SCXML311: requires datamodel assign")
}

func TestSCXML312(t *testing.T) {
	t.Skipf("SCXML312: requires datamodel assign illegal expr")
}

func TestSCXML313(t *testing.T) {
	t.Skipf("SCXML313: requires datamodel assign location")
}

func TestSCXML314(t *testing.T) {
	t.Skipf("SCXML314: requires datamodel assign expr")
}

func TestSCXML318(t *testing.T) {
	t.Skipf("SCXML318: requires _event system var")
}

func TestSCXML319(t *testing.T) {
	t.Skipf("SCXML319: requires _event binding")
}

func TestSCXML321(t *testing.T) {
	t.Skipf("SCXML321: requires _sessionid system var")
}

func TestSCXML322(t *testing.T) {
	t.Skipf("SCXML322: requires _sessionid immutability")
}

func TestSCXML323(t *testing.T) {
	t.Skipf("SCXML323: requires _name system var")
}

func TestSCXML324(t *testing.T) {
	t.Skipf("SCXML324: requires _name immutability")
}

func TestSCXML325(t *testing.T) {
	t.Skipf("SCXML325: requires _ioprocessors system var")
}

func TestSCXML326(t *testing.T) {
	t.Skipf("SCXML326: requires _ioprocessors immutability")
}

func TestSCXML329(t *testing.T) {
	t.Skipf("SCXML329: requires system vars immutability")
}

func TestSCXML330(t *testing.T) {
	t.Skipf("SCXML330: requires send event.type validation")
}

func TestSCXML331(t *testing.T) {
	t.Skipf("SCXML331: requires send event.sendid")
}

func TestSCXML332(t *testing.T) {
	t.Skipf("SCXML332: requires send event.origin")
}

func TestSCXML333(t *testing.T) {
	t.Skipf("SCXML333: requires send event.origintype")
}

func TestSCXML335(t *testing.T) {
	t.Skipf("SCXML335: requires send event.invokeid")
}

func TestSCXML336(t *testing.T) {
	t.Skipf("SCXML336: requires send event.data")
}

func TestSCXML337(t *testing.T) {
	t.Skipf("SCXML337: requires event fields in guards")
}

func TestSCXML338(t *testing.T) {
	t.Skipf("SCXML338: requires invokeid in events")
}

func TestSCXML339(t *testing.T) {
	t.Skipf("SCXML339: requires invoke done.invoke")
}

func TestSCXML342(t *testing.T) {
	t.Skipf("SCXML342: requires datamodel send eventexpr, assign event.name")
}

func TestSCXML343(t *testing.T) {
	t.Skipf("SCXML343: requires donedata params, error.execution, final donedata")
}

func TestSCXML344(t *testing.T) {
	t.Skipf("SCXML344: requires non-boolean condition, error.execution")
}

func TestSCXML346(t *testing.T) {
	t.Skipf("SCXML346: requires system vars (_sessionid/_event/_ioprocessors/_name), error.execution")
}

func TestSCXML347(t *testing.T) {
	t.Skipf("SCXML347: requires invoke parent/child send type/target")
}

func TestSCXML348(t *testing.T) {
	t.Skipf("SCXML348: requires send type='http://www.w3.org/TR/scxml/#SCXMLEventProcessor'")
}

func TestSCXML349(t *testing.T) {
	t.Skipf("SCXML349: requires send origin, datamodel")
}

func TestSCXML350(t *testing.T) {
	t.Skipf("SCXML350: requires send target #_scxml_sessionid, datamodel")
}

func TestSCXML351(t *testing.T) {
	t.Skipf("SCXML351: requires send id/sendid, datamodel")
}

func TestSCXML352(t *testing.T) {
	t.Skipf("SCXML352: requires send type, event.origintype, datamodel")
}

func TestSCXML354(t *testing.T) {
	t.Skipf("SCXML354: requires send namelist/param/content, event.data")
}

func TestSCXML355(t *testing.T) {
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
	root.Initial = s0 // First in document order, could use builder.NewComposite

	s0.Transitions = []*Transition{
		{Event: "", Target: "pass"}, // Immediate
	}
	s1.Transitions = []*Transition{
		{Event: "", Target: "fail"},
	}

	rt := NewRuntime(root, nil)
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop(ctx)

	rt.SendEvent(ctx, "")

	if !rt.IsInState("pass") {
		t.Error("should transition to pass via immediate empty event")
	}
}

func TestSCXML364(t *testing.T) {
	t.Skipf("SCXML364: requires parallel initial multiple children")
}

func TestSCXML372(t *testing.T) {
	t.Skipf("SCXML372: requires final donedata, onentry/onexit final, datamodel")
}

func TestSCXML375(t *testing.T) {
	root := &State{ID: "root"}
	s0 := &State{ID: "s0", Parent: root}
	pass := &State{ID: "pass", Parent: root}

	root.Children = map[StateID]*State{
		"s0":   s0,
		"pass": pass,
	}
	root.Initial = s0

	rt := NewRuntime(root, nil)
	s0.OnEntry = func(ctx context.Context, event Event, from, to StateID, ext any) {
		rt.SendEvent(ctx, "e1")
	}
	s0.Transitions = []*Transition{
		{Event: "e1", Target: "pass"},
	}
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop(ctx)

	if !rt.IsInState("pass") {
		t.Error("onentry order")
	}
}

func TestSCXML376(t *testing.T) {
	t.Skipf("SCXML376: requires onentry error isolation illegal send")
}

func TestSCXML377(t *testing.T) {
	root := &State{ID: "root"}
	s0 := &State{ID: "s0", Parent: root}
	pass := &State{ID: "pass", Parent: root}

	root.Children = map[StateID]*State{
		"s0":   s0,
		"pass": pass,
	}
	root.Initial = s0

	rt := NewRuntime(root, nil)
	s0.OnExit = func(ctx context.Context, event Event, from, to StateID, ext any) {
		rt.SendEvent(ctx, "e1")
	}
	s0.Transitions = []*Transition{
		{Event: "e1", Target: "pass"},
	}
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop(ctx)

	if !rt.IsInState("pass") {
		t.Error("onexit order")
	}
}

func TestSCXML378(t *testing.T) {
	t.Skipf("SCXML378: requires onexit error isolation illegal send")
}

func TestSCXML387(t *testing.T) {
	t.Skipf("SCXML387: requires deep history")
}

func TestSCXML388(t *testing.T) {
	t.Skipf("SCXML388: requires deep history + datamodel counters")
}

func TestSCXML396(t *testing.T) {
	root := &State{ID: "root"}
	s0 := &State{ID: "s0", Parent: root}
	pass := &State{ID: "pass", Parent: root}

	root.Children = map[StateID]*State{
		"s0":   s0,
		"pass": pass,
	}
	root.Initial = s0

	s0.Transitions = []*Transition{
		{Event: "foo", Target: "pass"},
		{Event: "foo", Target: "fail"}, // Duplicate, first wins?
	}

	rt := NewRuntime(root, nil)
	s0.OnEntry = func(ctx context.Context, event Event, from, to StateID, ext any) {
		rt.SendEvent(ctx, "foo")
	}
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop(ctx)

	if !rt.IsInState("pass") {
		t.Error("first transition matches")
	}
}

func TestSCXML399(t *testing.T) {
	t.Skipf("SCXML399: requires wildcard prefix foo.*, multiple event desc")
}
