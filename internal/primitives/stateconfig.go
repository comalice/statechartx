// Package primitives defines the foundational data structures for the statechart engine.
// All implementations use only the Go standard library (stdlib-only).
// No external dependencies.
//
// StateConfig represents a state in the statechart, supporting atomic, compound, parallel,
// and history state types with transitions, actions, and hierarchical nesting.
package primitives

import (
	"errors"
	"fmt"
	"strings"
)

// StateType defines the possible types of states in the statechart.
type StateType string

const (
	Atomic         StateType = "atomic"
	Compound       StateType = "compound"
	Parallel       StateType = "parallel"
	ShallowHistory StateType = "shallowHistory"
	DeepHistory    StateType = "deepHistory"
)

// StateConfig defines a state configuration, supporting hierarchical nesting.
type StateConfig struct {
	ID       string                        `json:"id" yaml:"id"`
	Type     StateType                     `json:"type" yaml:"type"`
	Initial  string                        `json:"initial,omitempty" yaml:"initial,omitempty"` // Initial child for compound/parallel
	On       map[string][]TransitionConfig `json:"on,omitempty" yaml:"on,omitempty"`
	Entry    []ActionRef                   `json:"entry,omitempty" yaml:"entry,omitempty"`
	Exit     []ActionRef                   `json:"exit,omitempty" yaml:"exit,omitempty"`
	Children []*StateConfig                `json:"children,omitempty" yaml:"children,omitempty"`
}

// NewStateConfig creates a new StateConfig with ID and Type.
func NewStateConfig(id string, typ StateType) *StateConfig {
	return &StateConfig{
		ID:   id,
		Type: typ,
	}
}

// WithInitial sets the initial child state ID (for compound/parallel).
func (s *StateConfig) WithInitial(initial string) *StateConfig {
	s.Initial = initial
	return s
}

// WithOn sets the event-to-transition map.
func (s *StateConfig) WithOn(on map[string][]TransitionConfig) *StateConfig {
	s.On = make(map[string][]TransitionConfig)
	for k, v := range on {
		s.On[k] = v
	}
	return s
}

// AddTransition adds a transition for an event.
func (s *StateConfig) AddTransition(event string, trans TransitionConfig) *StateConfig {
	if s.On == nil {
		s.On = make(map[string][]TransitionConfig)
	}
	s.On[event] = append(s.On[event], trans)
	return s
}

// WithEntry sets entry actions.
func (s *StateConfig) WithEntry(entry []ActionRef) *StateConfig {
	s.Entry = entry
	return s
}

// AddEntry adds an entry action.
func (s *StateConfig) AddEntry(action ActionRef) *StateConfig {
	s.Entry = append(s.Entry, action)
	return s
}

// WithExit sets exit actions.
func (s *StateConfig) WithExit(exit []ActionRef) *StateConfig {
	s.Exit = exit
	return s
}

// AddExit adds an exit action.
func (s *StateConfig) AddExit(action ActionRef) *StateConfig {
	s.Exit = append(s.Exit, action)
	return s
}

// WithChildren sets child states.
func (s *StateConfig) WithChildren(children []*StateConfig) *StateConfig {
	s.Children = children
	return s
}

// AddChild adds a child state.
func (s *StateConfig) AddChild(child *StateConfig) *StateConfig {
	s.Children = append(s.Children, child)
	return s
}

// State creates and adds a child state (atomic by default, or specified type).
// Returns the child for fluent chaining: parent.State("child").Transition("evt", "target").
func (s *StateConfig) State(id string, typ ...StateType) *StateConfig {
	t := Atomic
	if len(typ) > 0 {
		t = typ[0]
	}
	child := NewStateConfig(id, t)
	s.AddChild(child)
	return child
}

// Transition adds a simple transition from event to target.
// Optionally override with full TransitionConfig via first arg.
// Usage: .Transition("evt", "target") or .Transition("evt", "target", TransitionConfig{Guard: fn}).
func (s *StateConfig) Transition(event, target string, transOpts ...TransitionConfig) *StateConfig {
	trans := TransitionConfig{Target: target}
	if len(transOpts) > 0 {
		trans = transOpts[0]
	}
	return s.AddTransition(event, trans)
}

// Flatten returns a flat map[string]*StateConfig by recursing the entire hierarchy from this root.
func (s *StateConfig) Flatten() map[string]*StateConfig {
	m := make(map[string]*StateConfig)
	s.flattenHelper(m)
	return m
}

func (s *StateConfig) flattenHelper(m map[string]*StateConfig) {
	if _, ok := m[s.ID]; ok {
		return
	}
	m[s.ID] = s
	for _, child := range s.Children {
		child.flattenHelper(m)
	}
}

// Validate performs recursive validation of the StateConfig tree.
func (s *StateConfig) Validate() error {
	if s.ID == "" {
		return errors.New("state ID is required")
	}

	validTypes := map[StateType]struct{}{
		Atomic:         {},
		Compound:       {},
		Parallel:       {},
		ShallowHistory: {},
		DeepHistory:    {},
	}
	if _, ok := validTypes[s.Type]; !ok {
		return fmt.Errorf("invalid state type %q for state %s", s.Type, s.ID)
	}

	switch s.Type {
	case Atomic:
		if s.Initial != "" {
			return fmt.Errorf("atomic state %s cannot have Initial", s.ID)
		}
		if len(s.Children) > 0 {
			return fmt.Errorf("atomic state %s cannot have Children", s.ID)
		}
	case Compound, Parallel:
		if len(s.Children) == 0 {
			return fmt.Errorf("%s state %s requires Children", s.Type, s.ID)
		}
		if s.Initial == "" {
			return fmt.Errorf("%s state %s requires Initial child", s.Type, s.ID)
		}
		initialFound := false
		for _, child := range s.Children {
			if child.ID == s.Initial {
				initialFound = true
				break
			}
		}
		if !initialFound {
			return fmt.Errorf("initial child %q not found in children of %s", s.Initial, s.ID)
		}
	case ShallowHistory, DeepHistory:
		if len(s.Children) > 0 {
			return fmt.Errorf("history state %s cannot have Children (restored at runtime)", s.ID)
		}
	}

	if s.On != nil {
		for event := range s.On {
			if strings.TrimSpace(event) == "" {
				return fmt.Errorf("empty event name in On map for state %s", s.ID)
			}
		}
	}

	for i, child := range s.Children {
		if err := child.Validate(); err != nil {
			return fmt.Errorf("child %d (%s) of %s failed validation: %w", i, child.ID, s.ID, err)
		}
	}

	return nil
}
