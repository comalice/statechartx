package statechartx

import (
	"testing"
)

func TestSCXML500(t *testing.T) {
	t.Skipf("SCXML500: requires datamodel (scxmlEventIOLocation variable)")
}

func TestSCXML501(t *testing.T) {
	t.Skipf("SCXML501: requires send with targetVar, delays, external event queues")
}

func TestSCXML503(t *testing.T) {
	t.Skipf("SCXML503: requires internal transitions and eventless transitions (Phase 3)")
}

func TestSCXML504(t *testing.T) {
	t.Skipf("SCXML504: requires parallel states (not supported)")
}

func TestSCXML505(t *testing.T) {
	t.Skipf("SCXML505: requires self-transitions and eventless transitions (Phase 3)")
}

func TestSCXML506(t *testing.T) {
	t.Skipf("SCXML506: requires compound state self-transitions and eventless transitions (Phase 3)")
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
