package statechartx

import (
	"testing"
)

// 200s SCXML tests: All require unsupported features (invoke, send delay/param/type/target, datamodel assign/var/expr,
// event data/origintype/invokeid, parallel, finalize, done.invoke, cancel, timing).
// All skipped per statechartx limitations.

func TestSCXML200(t *testing.T) {
	t.Skipf("SCXML200: requires send type='scxml' default behavior")
}

func TestSCXML201(t *testing.T) {
	t.Skipf("SCXML201: requires send type='http.request' (optional)")
}

func TestSCXML205(t *testing.T) {
	t.Skipf("SCXML205: requires send with <param>, event.data validation")
}

func TestSCXML207(t *testing.T) {
	t.Skipf("SCXML207: requires invoke delay/cancel, parent-child send")
}

func TestSCXML208(t *testing.T) {
	t.Skipf("SCXML208: requires send delay/cancel with sendid")
}

func TestSCXML210(t *testing.T) {
	t.Skipf("SCXML210: requires cancel with sendidexpr")
}

func TestSCXML215(t *testing.T) {
	t.Skipf("SCXML215: requires invoke with typeexpr")
}

func TestSCXML216(t *testing.T) {
	t.Skipf("SCXML216: requires invoke with srcexpr")
}

func TestSCXML220(t *testing.T) {
	t.Skipf("SCXML220: requires invoke type='scxml'")
}

func TestSCXML223(t *testing.T) {
	t.Skipf("SCXML223: requires invoke idlocation")
}

func TestSCXML224(t *testing.T) {
	t.Skipf("SCXML224: requires invoke auto-ID format validation")
}

func TestSCXML225(t *testing.T) {
	t.Skipf("SCXML225: requires unique invoke IDs")
}

func TestSCXML226(t *testing.T) {
	t.Skipf("SCXML226: requires invoke param passing")
}

func TestSCXML228(t *testing.T) {
	t.Skipf("SCXML228: requires invokeid in events")
}

func TestSCXML229(t *testing.T) {
	t.Skipf("SCXML229: requires invoke autoforward")
}

func TestSCXML230(t *testing.T) {
	t.Skipf("SCXML230: requires invoke autoforward, send #_parent, event fields (name/type/sendid/origin/origintype/invokeid/data)")
}

func TestSCXML232(t *testing.T) {
	t.Skipf("SCXML232: requires invoke, send #_parent from child, done.invoke, delays")
}

func TestSCXML233(t *testing.T) {
	t.Skipf("SCXML233: requires invoke finalize, datamodel assign from event.data, send params")
}

func TestSCXML234(t *testing.T) {
	t.Skipf("SCXML234: requires parallel, multiple invoke, finalize, datamodel assign event.data")
}

func TestSCXML235(t *testing.T) {
	t.Skipf("SCXML235: requires invoke id, done.invoke.{id}, delays")
}

func TestSCXML236(t *testing.T) {
	t.Skipf("SCXML236: requires invoke onexit send #_parent before done.invoke, delays")
}

func TestSCXML237(t *testing.T) {
	t.Skipf("SCXML237: requires invoke cancel on exit, no done.invoke")
}

func TestSCXML239(t *testing.T) {
	t.Skipf("SCXML239: requires invoke src vs content, delays")
}

func TestSCXML240(t *testing.T) {
	t.Skipf("SCXML240: requires invoke <param>, datamodel parent/child vars, send #_parent")
}

func TestSCXML241(t *testing.T) {
	t.Skipf("SCXML241: requires invoke namelist+param, datamodel, send #_parent")
}

func TestSCXML242(t *testing.T) {
	t.Skipf("SCXML242: requires invoke src/content consistency, delays")
}

func TestSCXML243(t *testing.T) {
	t.Skipf("SCXML243: requires invoke <param>, datamodel, send #_parent")
}

func TestSCXML244(t *testing.T) {
	t.Skipf("SCXML244: requires invoke namelist, datamodel parent/child, send #_parent")
}

func TestSCXML245(t *testing.T) {
	t.Skipf("SCXML245: requires invoke namelist, conf:isBound, send #_parent")
}

func TestSCXML247(t *testing.T) {
	t.Skipf("SCXML247: requires invoke, done.invoke, delays")
}

func TestSCXML250(t *testing.T) {
	t.Skipf("SCXML250: requires invoke cancel, child onexit")
}

func TestSCXML252(t *testing.T) {
	t.Skipf("SCXML252: requires invoke cancel on exit, child events ignored")
}

func TestSCXML253(t *testing.T) {
	t.Skipf("SCXML253: requires invoke type=scxml, send #_parent/#_foo, event.origintype, datamodel assign")
}

func TestSCXML276(t *testing.T) {
	t.Skipf("SCXML276: requires invoke param passing parent/child")
}

func TestSCXML277(t *testing.T) {
	t.Skipf("SCXML277: requires datamodel illegal expr, error.execution")
}

func TestSCXML278(t *testing.T) {
	t.Skipf("SCXML278: requires datamodel var scoping")
}

func TestSCXML279(t *testing.T) {
	t.Skipf("SCXML279: requires datamodel early binding")
}

func TestSCXML280(t *testing.T) {
	t.Skipf("SCXML280: requires datamodel late binding")
}

func TestSCXML286(t *testing.T) {
	t.Skipf("SCXML286: requires datamodel assign invalid loc, error.execution")
}

func TestSCXML287(t *testing.T) {
	t.Skipf("SCXML287: requires datamodel assign")
}

func TestSCXML294(t *testing.T) {
	t.Skipf("SCXML294: requires donedata param/content, event.data")
}

func TestSCXML298(t *testing.T) {
	t.Skipf("SCXML298: requires donedata invalid loc, error.execution, delays")
}
