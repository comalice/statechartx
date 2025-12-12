// Package primitives defines the foundational data structures for the statechart engine.
// All implementations use only the Go standard library (stdlib-only).
// No external dependencies.
//
// MachineConfig represents the top-level configuration of a statechart machine,
// containing the machine ID, initial state, and flat map of all states by ID.
// States support hierarchical nesting via the Children field.
// Validation ensures ID/Initial presence, state validity, target existence, and no orphans.

package primitives

import (
	"errors"
	"fmt"
	"strings"
)

// MachineConfig defines the complete statechart configuration.
type MachineConfig struct {
	Version string `json:\"version,omitempty\" yaml:\"version,omitempty\"`
	ID      string                  `json:"id" yaml:"id"`
	Initial string                  `json:"initial" yaml:"initial"`
	States  map[string]*StateConfig `json:"states" yaml:"states"`
}

// Validate validates the entire machine configuration:
// - Non-empty ID and Initial
// - Initial exists in States
// - All individual states validate (recursive)
// - All transition targets exist in States
// - No orphaned states (all reachable from Initial via Children hierarchy)
func (m *MachineConfig) Validate() error {
	if m.ID == "" {
		return errors.New("machine ID is required")
	}
	if m.Initial == "" {
		return errors.New("initial state ID is required")
	}
	if m.States == nil || len(m.States) == 0 {
		return errors.New("states map is required and cannot be empty")
	}
	initialState, ok := m.States[m.Initial]
	if !ok {
		return fmt.Errorf("initial state %q not found in states", m.Initial)
	}

	// Validate all states recursively
	for sid, state := range m.States {
		if err := state.Validate(); err != nil {
			return fmt.Errorf("state %q validation failed: %w", sid, err)
		}
	}

	// Validate transition targets exist
	for sid, state := range m.States {
		if state.On != nil {
			for event, transitions := range state.On {
				for i, trans := range transitions {
					if trans.Target != "" {
						if _, exists := m.States[trans.Target]; !exists {
							return fmt.Errorf("invalid transition target %q (state %q, event %q, transition %d)", trans.Target, sid, event, i)
						}
					}
				}
			}
		}
	}

	// Check no orphaned states via reachability
	visited := make(map[string]bool)
	if err := m.markReachable(initialState, m.States, visited); err != nil {
		return fmt.Errorf("reachability check failed: %w", err)
	}
	for sid := range m.States {
		if !visited[sid] {
			return fmt.Errorf("orphaned state %q (not reachable from initial %q)", sid, m.Initial)
		}
	}

	return nil
}

// markReachable recursively marks reachable states via Children hierarchy and transition targets.
func (m *MachineConfig) markReachable(state *StateConfig, states map[string]*StateConfig, visited map[string]bool) error {
	if visited[state.ID] {
		return nil
	}
	visited[state.ID] = true

	// Traverse children (hierarchical)
	for _, child := range state.Children {
		if err := m.markReachable(child, states, visited); err != nil {
			return err
		}
	}

	// Traverse transition targets (sibling/cross-hierarchy)
	if state.On != nil {
		for _, transitions := range state.On {
			for _, trans := range transitions {
				if trans.Target != "" {
					// Handle both simple IDs and hierarchical paths
					targetID := strings.Split(trans.Target, ".")[0]
					if targetState, ok := states[targetID]; ok && !visited[targetID] {
						if err := m.markReachable(targetState, states, visited); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

// FindState resolves a state by hierarchical path (e.g. "parent.child.grandchild").
func (m *MachineConfig) FindState(path string) (*StateConfig, error) {
	if path == "" {
		return nil, errors.New("path cannot be empty")
	}
	segments := strings.Split(path, ".")
	if len(segments) == 0 {
		return nil, errors.New("invalid path")
	}
	current, ok := m.States[segments[0]]
	if !ok {
		return nil, fmt.Errorf("state %q not found", segments[0])
	}
	for i := 1; i < len(segments); i++ {
		seg := segments[i]
		found := false
		for _, child := range current.Children {
			if child.ID == seg {
				current = child
				found = true
				break
			}
		}
		if !found {
			prefix := strings.Join(segments[:i], ".")
			return nil, fmt.Errorf("child %q not found in %q", seg, prefix)
		}
	}
	return current, nil
}
