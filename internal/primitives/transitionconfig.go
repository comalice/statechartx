// Package primitives defines the foundational data structures for the statechart engine.
// TransitionConfig defines transitions between states with guards, actions, and priority.
// Supports SCXML semantics: Event-triggered, guarded, prioritized transitions.
// All implementations use only the Go standard library (stdlib-only).
// No external dependencies.
//
// TransitionConfig supports hierarchical targets via dot-separated paths (e.g., "parent.child").
// Guards and Actions are pluggable references (function or string ID).
// Higher Priority values are evaluated first during transition selection.
package primitives

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ActionRef references an action: either a string ID or func(*Context, Event).
type ActionRef any

// GuardRef references a guard condition: either a string ID or func(*Context, Event) bool.
type GuardRef any

// TransitionConfig defines a single transition triggered by an Event.
type TransitionConfig struct {
	Event    string      `json:"event"`
	Guard    GuardRef    `json:"guard,omitempty"`
	Target   string      `json:"target"`
	Actions  []ActionRef `json:"actions,omitempty"`
	Priority int         `json:"priority,omitempty"` // higher = evaluated first (default 0)
}

// Validate checks TransitionConfig fields and target path syntax.
func (t *TransitionConfig) Validate() error {
	if t.Event == "" {
		return errors.New("event is required")
	}
	if t.Target == "" {
		return errors.New("target is required")
	}
	// Target path syntax: dot-separated non-empty alphanumeric segments (basic validation)
	segments := strings.Split(t.Target, ".")
	for i, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return fmt.Errorf("invalid target path %q: empty segment at index %d", t.Target, i)
		}
		// Basic ID validation: alphanumeric + underscores/hyphens
		for _, r := range seg {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
				return fmt.Errorf("invalid target path %q: invalid character '%c' at index %d", t.Target, r, i)
			}
		}
	}
	if t.Priority < 0 {
		return errors.New("priority must be non-negative")
	}
	return nil
}

// SortTransitions sorts the slice in place by Priority descending (highest first).
func SortTransitions(transitions []TransitionConfig) {
	sort.Slice(transitions, func(i, j int) bool {
		return transitions[i].Priority > transitions[j].Priority
	})
}
