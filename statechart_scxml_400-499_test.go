package statechartx

import (
        "context"
        "testing"
        "time"
)

func TestSCXML401(t *testing.T) {
        t.Skipf("SCXML401: requires error events, internal queue priority, assign to invalid location")
}

func TestSCXML402(t *testing.T) {
        t.Skipf("SCXML402: requires error events, internal queue ordering, assign to invalid location")
}

func TestSCXML403a(t *testing.T) {
        // Test optimal enablement: child transitions take precedence over parent,
        // document order breaks ties, and parent transitions fire if no child match
        const (
                STATE_ROOT StateID = 1
                STATE_S0   StateID = 2
                STATE_S01  StateID = 3
                STATE_S02  StateID = 4
                STATE_PASS StateID = 5
                STATE_FAIL StateID = 6
                EVENT_1    EventID = 1
                EVENT_2    EventID = 2
        )

        root := &State{ID: STATE_ROOT}
        s0 := &State{ID: STATE_S0}
        s01 := &State{ID: STATE_S01}
        s02 := &State{ID: STATE_S02}
        pass := &State{ID: STATE_PASS}
        fail := &State{ID: STATE_FAIL}

        root.Children = map[StateID]*State{
                STATE_S0:   s0,
                STATE_PASS: pass,
                STATE_FAIL: fail,
        }
        root.Initial = STATE_S0

        s0.Children = map[StateID]*State{
                STATE_S01: s01,
                STATE_S02: s02,
        }
        s0.Initial = STATE_S01

        // Parent s0 transitions (should not fire if child matches)
        s0.Transitions = []*Transition{
                {Event: EVENT_1, Target: STATE_FAIL},
                {Event: EVENT_2, Target: STATE_PASS},
        }

        // s01 transitions: first one should fire (document order)
        s01.Transitions = []*Transition{
                {Event: EVENT_1, Target: STATE_S02},
                {Event: ANY_EVENT, Target: STATE_FAIL},
        }

        // s02 has local transition with false guard, so parent should fire
        s02.Transitions = []*Transition{
                {Event: EVENT_1, Target: STATE_FAIL},
                {Event: EVENT_2, Target: STATE_FAIL, Guard: func(ctx context.Context, event *Event, from, to StateID) (bool, error) {
                        return false, nil // This guard fails, so transition shouldn't fire
                }},
        }

        machine, _ := NewMachine(root)

        rt := NewRuntime(machine, nil)
        s01.EntryAction = func(ctx context.Context, event *Event, from, to StateID) error {
                rt.SendEvent(ctx, Event{ID: EVENT_1})
                return nil
        }
        s02.EntryAction = func(ctx context.Context, event *Event, from, to StateID) error {
                rt.SendEvent(ctx, Event{ID: EVENT_2})
                return nil
        }
        ctx := context.Background()

        rt.Start(ctx)
        time.Sleep(100 * time.Millisecond) // Give event loop time to process
        defer rt.Stop()

        if !rt.IsInState(STATE_PASS) {
                t.Error("should reach pass state")
        }
}

func TestSCXML403b(t *testing.T) {
        t.Skipf("SCXML403b: requires parallel states and datamodel (optimally enabled set is a set)")
}

func TestSCXML403c(t *testing.T) {
        t.Skipf("SCXML403c: requires parallel states, preemption, and datamodel")
}

func TestSCXML404(t *testing.T) {
	t.Skipf("SCXML404: requires parallel state exit ordering - see realtime/scxml_parallel_test.go:TestSCXML404_Realtime for deterministic test")
}

func TestSCXML405(t *testing.T) {
	t.Skipf("SCXML405: requires parallel transition execution ordering - see realtime/scxml_parallel_test.go:TestSCXML405_Realtime for deterministic test")
}

func TestSCXML406(t *testing.T) {
	t.Skipf("SCXML406: requires parallel state entry ordering - see realtime/scxml_parallel_test.go:TestSCXML406_Realtime for deterministic test")
}

func TestSCXML407(t *testing.T) {
        t.Skipf("SCXML407: requires eventless transitions (Phase 3)")
}

func TestSCXML409(t *testing.T) {
        t.Skipf("SCXML409: requires In() predicate in onexit handlers to check active states")
}

func TestSCXML411(t *testing.T) {
        t.Skipf("SCXML411: requires In() predicate in onentry handlers to check active states")
}

func TestSCXML412(t *testing.T) {
        // Test that initial transition actions execute in correct order:
        // parent OnEntry → initial transition action → child OnEntry
        const (
                STATE_ROOT  StateID = 1
                STATE_S1    StateID = 2
                STATE_S11   StateID = 3
                STATE_PASS  StateID = 4
                STATE_FAIL  StateID = 5
                EVENT_CHECK EventID = 1
        )

        root := &State{ID: STATE_ROOT}
        s1 := &State{ID: STATE_S1}
        s11 := &State{ID: STATE_S11}
        pass := &State{ID: STATE_PASS, IsFinal: true}
        fail := &State{ID: STATE_FAIL, IsFinal: true}

        root.Children = map[StateID]*State{
                STATE_S1:   s1,
                STATE_PASS: pass,
                STATE_FAIL: fail,
        }
        root.Initial = STATE_S1

        s1.Children = map[StateID]*State{
                STATE_S11: s11,
        }
        s1.Initial = STATE_S11

        machine, err := NewMachine(root)
        if err != nil {
                t.Fatalf("Failed to create machine: %v", err)
        }

        // Track execution order
        var executionOrder []string

        rt := NewRuntime(machine, nil)
        ctx := context.Background()

        // s1 entry action
        s1.EntryAction = func(ctx context.Context, event *Event, from, to StateID) error {
                executionOrder = append(executionOrder, "s1_entry")
                return nil
        }

        // s1 initial transition action (Step 10)
        s1.InitialAction = func(ctx context.Context, event *Event, from, to StateID) error {
                executionOrder = append(executionOrder, "s1_initial_action")
                return nil
        }

        // s11 entry action
        s11.EntryAction = func(ctx context.Context, event *Event, from, to StateID) error {
                executionOrder = append(executionOrder, "s11_entry")
                return nil
        }

        // s11 has eventless transition to check execution order
        s11.Transitions = []*Transition{
                {
                        Event:  NO_EVENT,
                        Target: STATE_PASS,
                        Guard: func(ctx context.Context, event *Event, from, to StateID) (bool, error) {
                                // Verify execution order
                                expected := []string{"s1_entry", "s1_initial_action", "s11_entry"}
                                if len(executionOrder) != len(expected) {
                                        return false, nil
                                }
                                for i, v := range expected {
                                        if executionOrder[i] != v {
                                                return false, nil
                                        }
                                }
                                return true, nil
                        },
                },
                {Event: NO_EVENT, Target: STATE_FAIL}, // Fallback if order is wrong
        }

        if err := rt.Start(ctx); err != nil {
                t.Fatalf("Failed to start runtime: %v", err)
        }
        time.Sleep(100 * time.Millisecond)
        defer rt.Stop()

        if !rt.IsInState(STATE_PASS) {
                t.Errorf("Initial transition action should execute after parent entry but before child entry. Order: %v", executionOrder)
        }
}

func TestSCXML413(t *testing.T) {
	// Test that parallel states with multiple initial state specification
	// This test requires support for specifying initial states in parallel regions
	// which targets specific states within each region
	t.Skipf("SCXML413: requires support for multiple initial state specification in parallel regions (advanced feature)")
}

func TestSCXML415(t *testing.T) {
        t.Skipf("SCXML415: manual test - requires final state halting behavior verification")
}

func TestSCXML416(t *testing.T) {
        t.Skipf("SCXML416: requires final states and done.state.id event generation")
}

func TestSCXML417(t *testing.T) {
        t.Skipf("SCXML417: requires parallel states, final states, and done.state.id events")
}

func TestSCXML419(t *testing.T) {
        // Test that eventless transitions take precedence over event-driven ones
        const (
                STATE_ROOT           StateID = 1
                STATE_S1             StateID = 2
                STATE_PASS           StateID = 3
                STATE_FAIL           StateID = 4
                EVENT_INTERNAL_EVENT EventID = 1
                EVENT_EXTERNAL_EVENT EventID = 2
        )

        root := &State{ID: STATE_ROOT}
        s1 := &State{ID: STATE_S1}
        pass := &State{ID: STATE_PASS, Final: true}
        fail := &State{ID: STATE_FAIL, Final: true}

        root.Children = map[StateID]*State{
                STATE_S1:   s1,
                STATE_PASS: pass,
                STATE_FAIL: fail,
        }
        root.Initial = STATE_S1

        machine, err := NewMachine(root)
        if err != nil {
                t.Fatalf("Failed to create machine: %v", err)
        }

        rt := NewRuntime(machine, nil)
        ctx := context.Background()

        // s1 onentry raises internal and external events
        s1.EntryAction = func(ctx context.Context, event *Event, from, to StateID) error {
                rt.SendEvent(ctx, Event{ID: EVENT_INTERNAL_EVENT})
                rt.SendEvent(ctx, Event{ID: EVENT_EXTERNAL_EVENT})
                return nil
        }

        // s1 has two transitions:
        // 1. Event-driven transition (should NOT fire because eventless takes precedence)
        // 2. Eventless transition (should fire first)
        s1.Transitions = []*Transition{
                {Event: ANY_EVENT, Target: STATE_FAIL}, // Event-driven (wildcard)
                {Event: NO_EVENT, Target: STATE_PASS},  // Eventless (should take precedence)
        }

        if err := rt.Start(ctx); err != nil {
                t.Fatalf("Failed to start runtime: %v", err)
        }
        time.Sleep(100 * time.Millisecond) // Give time for transitions
        defer rt.Stop()

        if !rt.IsInState(STATE_PASS) {
                t.Error("Eventless transition should take precedence over event-driven transitions")
        }
}

func TestSCXML421(t *testing.T) {
        // Test event matching priority: document order matters
        // First matching transition (with passing guard) should win
        const (
                STATE_ROOT StateID = 1
                STATE_S0   StateID = 2
                STATE_S1   StateID = 3
                STATE_S2   StateID = 4
                STATE_PASS StateID = 5
                STATE_FAIL StateID = 6
                EVENT_E1   EventID = 1
        )

        root := &State{ID: STATE_ROOT}
        s0 := &State{ID: STATE_S0}
        s1 := &State{ID: STATE_S1}
        s2 := &State{ID: STATE_S2}
        pass := &State{ID: STATE_PASS, IsFinal: true}
        fail := &State{ID: STATE_FAIL, IsFinal: true}

        root.Children = map[StateID]*State{
                STATE_S0:   s0,
                STATE_S1:   s1,
                STATE_S2:   s2,
                STATE_PASS: pass,
                STATE_FAIL: fail,
        }
        root.Initial = STATE_S0

        machine, err := NewMachine(root)
        if err != nil {
                t.Fatalf("Failed to create machine: %v", err)
        }

        rt := NewRuntime(machine, nil)
        ctx := context.Background()

        // s0 has multiple transitions for same event - document order should win
        s0.Transitions = []*Transition{
                {
                        Event:  EVENT_E1,
                        Target: STATE_FAIL,
                        Guard: func(ctx context.Context, event *Event, from, to StateID) (bool, error) {
                                return false, nil // Guard fails, should try next
                        },
                },
                {
                        Event:  EVENT_E1,
                        Target: STATE_S1, // This should fire (first with passing guard)
                },
                {
                        Event:  EVENT_E1,
                        Target: STATE_FAIL, // Should not fire (comes after)
                },
        }

        s1.Transitions = []*Transition{
                {Event: NO_EVENT, Target: STATE_PASS}, // Immediate transition to pass
        }

        s0.EntryAction = func(ctx context.Context, event *Event, from, to StateID) error {
                rt.SendEvent(ctx, Event{ID: EVENT_E1})
                return nil
        }

        if err := rt.Start(ctx); err != nil {
                t.Fatalf("Failed to start runtime: %v", err)
        }
        time.Sleep(100 * time.Millisecond)
        defer rt.Stop()

        if !rt.IsInState(STATE_PASS) {
                t.Error("First transition with passing guard should win (document order)")
        }
}

func TestSCXML422(t *testing.T) {
        t.Skipf("SCXML422: requires invoke functionality and parent-child session communication")
}

func TestSCXML423(t *testing.T) {
        t.Skipf("SCXML423: requires separate internal/external event queues (Phase 4)")
}

func TestSCXML436(t *testing.T) {
        t.Skipf("SCXML436: requires parallel in() predicate")
}

func TestSCXML444(t *testing.T) {
        t.Skipf("SCXML444: requires datamodel <data> expr")
}

func TestSCXML445(t *testing.T) {
        t.Skipf("SCXML445: requires datamodel undefined")
}

func TestSCXML446(t *testing.T) {
        t.Skipf("SCXML446: requires datamodel JSON/src")
}

func TestSCXML448(t *testing.T) {
        t.Skipf("SCXML448: requires datamodel global parallel")
}

func TestSCXML449(t *testing.T) {
        t.Skipf("SCXML449: requires datamodel truthy cond")
}

func TestSCXML451(t *testing.T) {
        t.Skipf("SCXML451: requires parallel in()")
}

func TestSCXML452(t *testing.T) {
        t.Skipf("SCXML452: requires datamodel assign substruct")
}

func TestSCXML453(t *testing.T) {
        t.Skipf("SCXML453: requires datamodel function expr")
}

func TestSCXML456(t *testing.T) {
        t.Skipf("SCXML456: requires <script> datamodel")
}

func TestSCXML457(t *testing.T) {
        t.Skipf("SCXML457: requires foreach arrays error")
}

func TestSCXML459(t *testing.T) {
        t.Skipf("SCXML459: requires foreach index order")
}

func TestSCXML460(t *testing.T) {
        t.Skipf("SCXML460: requires foreach shallow copy")
}

func TestSCXML487(t *testing.T) {
        t.Skipf("SCXML487: requires datamodel assign illegal expr error")
}

func TestSCXML488(t *testing.T) {
        t.Skipf("SCXML488: requires donedata param error, done.state data")
}

func TestSCXML495(t *testing.T) {
        t.Skipf("SCXML495: requires send type internal/external queue")
}

func TestSCXML496(t *testing.T) {
        t.Skipf("SCXML496: requires send unreachable target error")
}
