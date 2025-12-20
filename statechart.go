package statechartx

import (
	"context"
	"errors"
)

type StateID int
type EventID int

type Event struct {
	ID      EventID
	Payload any
}

type Action func(ctx context.Context, evt *Event, from StateID, to StateID) error
type Guard func(ctx context.Context, evt *Event, from StateID, to StateID) (bool, error)

// ---

type State struct {
	ID          StateID
	Transitions []*Transition
	EntryAction Action
	ExitAction  Action
	Initial     bool
	Final       bool
}

type CompoundState struct {
	State
	Initial  StateID
	Children []*State
}

type Transition struct {
	Event  Event
	Source *State
	Target *State // nil --> internal transition
	Guard  Guard  // nil --> do nothing
	Action Action // nil --> do nothing
}

// Machine is a CompoundState with helper functions for chart evaluation.
type Machine struct {
	CompoundState
	states  map[StateID]*State
	current *State
}

//
// Public API
//

func (s *State) OnEntry(action Action) {
	s.EntryAction = action
}

func (s *State) OnExit(action Action) {
	s.ExitAction = action
}

func NewMachine(states ...*State) (*Machine, error) {
	if len(states) == 0 {
		return nil, errors.New("no states provided")
	}
	m := &Machine{
		CompoundState: CompoundState{
			Children: states,
		},
		states: map[StateID]*State{},
	}

	// Build LUT and find initial state.
	var initial *State
	for _, s := range states {
		if s == nil {
			return nil, errors.New("nil state")
		}
		// Check registered states against this new state.
		if _, exists := m.states[s.ID]; exists {
			return nil, errors.New("duplicate state ID")
		}
		m.states[s.ID] = s
		if s.Initial {
			if initial != nil {
				return nil, errors.New("more than one initial state")
			}
			initial = s
		}
	}

	if initial == nil {
		initial = states[0] // First state is assigned as initial.
	}
	m.Initial = initial.ID
	m.current = initial

	// Validate transitions have a source set.
	// NOTE I'm not sure this is necessary?
	for _, s := range states {
		for _, t := range s.Transitions {
			if t == nil {
				continue
			}
			if t.Source == nil {
				t.Source = s
			}
		}
	}

	return m, nil
}

// Start enters machine initial state.
func (m *Machine) Start(ctx context.Context) error {
	if m.current == nil {
		return errors.New("machine has no current state")
	}
	return m.current.enterState(ctx, nil, m.current.ID, m.current.ID)
}

func (m *Machine) Send(ctx context.Context, evt Event) error {
	if m.current == nil {
		return errors.New("machine not started") // Should we collapse 'start' functionality into this function?
	}

	t := m.pickTransition(m.current, &evt)
	if t == nil {
		// No transitions match this event, ignore.
		return nil
	}

	next, err := t.doTransition(ctx, &evt)
	if err != nil {
		return err
	}

	m.current = next
	return nil
}

func (s *State) On(e Event, target *State, guard *Guard, action *Action) {
	t := &Transition{
		Event:  e,
		Source: s,
		Target: target,
		Guard:  nil,
		Action: nil,
	}

	if guard != nil {
		t.Guard = *guard
	}
	if action != nil {
		t.Action = *action
	}

	s.Transitions = append(s.Transitions, t)
}

//
// Helper Functions (internal API)
//

func (t *State) evaluateEntryAction(ctx context.Context, evt *Event, sourceID StateID, targetID StateID) error {
	if t.EntryAction != nil {
		return t.EntryAction(ctx, evt, sourceID, targetID)
	}
	return nil
}

func (t *State) evaluateExitAction(ctx context.Context, evt *Event, sourceID StateID, targetID StateID) error {
	if t.ExitAction != nil {
		return t.ExitAction(ctx, evt, sourceID, targetID)
	}
	return nil
}

// enterState enters a target state.
func (s *State) enterState(ctx context.Context, evt *Event, from StateID, to StateID) error {
	if err := s.evaluateEntryAction(ctx, evt, from, to); err != nil {
		return err
	}
	return nil
}

// exitState exits a target state.
func (s *State) exitState(ctx context.Context, evt *Event, from StateID, to StateID) error {
	if err := s.evaluateExitAction(ctx, evt, from, to); err != nil {
		return err
	}
	return nil
}

// pickTransition grabs the _first_ matching transition (following SCXML document order reqs)
func (m *Machine) pickTransition(s *State, evt *Event) *Transition {
	for _, t := range s.Transitions {
		if t == nil {
			continue
		}
		if t.Event.ID != evt.ID {
			continue
		}
		// Found one.
		return t
	}
	return nil
}

func (t *Transition) evaluateGuard(ctx context.Context, evt *Event, sourceID StateID, targetID StateID) (bool, error) {
	if t.Guard != nil {
		return t.Guard(ctx, evt, sourceID, targetID)
	}
	return true, nil
}

func (t *Transition) evaluateAction(ctx context.Context, evt *Event, sourceID StateID, targetID StateID) error {
	if t.Action != nil {
		return t.Action(ctx, evt, sourceID, targetID)
	}
	return nil
}

// doTransition evaluates a transition and returns the final state's ID.
func (t *Transition) doTransition(ctx context.Context, evt *Event) (*State, error) {
	// Check guard.
	pass, err := t.evaluateGuard(ctx, evt, t.Source.ID, t.Target.ID)
	// If error OR guard returns False, state in source state.
	if err != nil || !pass {
		return t.Source, err
	}

	// Exit previous state.
	if err := t.Source.exitState(ctx, evt, t.Source.ID, t.Target.ID); err != nil {
		return t.Source, err
	}

	// Do transition action.
	if err := t.evaluateAction(ctx, evt, t.Source.ID, t.Target.ID); err != nil {
		// Rewind to previous state using internal transition == nil.
		if err := t.Source.enterState(ctx, nil, t.Source.ID, t.Target.ID); err != nil {
			return t.Source, err
		}
		return t.Source, nil
	}

	// Enter next state.
	if err := t.Target.enterState(ctx, evt, t.Source.ID, t.Target.ID); err != nil {
		return t.Source, err
	}

	return t.Target, nil
}
