package builder

import (
	"context"

	"github.com/comalice/statechartx" // the core package
)

// StateID shortcut
type ID = statechartx.StateID

// Convenience types for actions/guards
type Action func(ctx context.Context, event statechartx.Event, from, to ID, ext any)
type Guard func(ctx context.Context, event statechartx.Event, from, to ID, ext any) bool

// New creates a basic leaf state
func New(id ID, opts ...Option) *statechartx.State {
	s := &statechartx.State{
		ID:          id,
		Children:    make(map[ID]*statechartx.State),
		Transitions: []*statechartx.Transition{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func NewComposite(id statechartx.StateID, children ...*statechartx.State) *statechartx.State {
	s := &statechartx.State{
		ID:       id,
		Children: make(map[statechartx.StateID]*statechartx.State),
	}
	if len(children) > 0 {
		s.Initial = children[0]
		for _, ch := range children {
			ch.Parent = s
			s.Children[ch.ID] = ch
		}
	}
	return s
}

// Composite creates a composite state with children in order (first = initial)
func Composite(id ID, children ...*statechartx.State) *statechartx.State {
	s := New(id)
	for i, ch := range children {
		ch.Parent = s
		s.Children[ch.ID] = ch
		if i == 0 {
			s.Initial = ch
		}
	}
	return s
}

// Option pattern for configuring states
type Option func(*statechartx.State)

// OnEntry adds an action to a state that executes when the state is entered.
func OnEntry(act Action) Option {
	return func(s *statechartx.State) { s.OnEntry = statechartx.Action(act) }
}

// OnExit adds an action to a state that executes when the state is exited.
func OnExit(act Action) Option {
	return func(s *statechartx.State) { s.OnExit = statechartx.Action(act) }
}

// On adds an outbound transition to a target state.
func On(event statechartx.Event, target ID, opts ...TransOption) Option {
	return func(s *statechartx.State) {
		t := &statechartx.Transition{
			Event:  event,
			Target: target,
		}
		// optional guard and/or action
		for _, opt := range opts {
			opt(t)
		}
		s.Transitions = append(s.Transitions, t)
	}
}

type TransOption func(*statechartx.Transition)

func WithGuard(g Guard) TransOption {
	return func(t *statechartx.Transition) { t.Guard = statechartx.Guard(g) }
}

func WithAction(act Action) TransOption {
	return func(t *statechartx.Transition) { t.Action = statechartx.Action(act) }
}
