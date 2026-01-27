package statechartx

import (
	"fmt"
	"strings"
)

// MachineBuilder provides a fluent API for constructing state machines using string-based state names
// instead of manual integer-based State struct creation.
type MachineBuilder struct {
	nextID   StateID
	nameToID map[string]StateID
	idToName map[StateID]string // For debugging/reverse lookup
	states   map[StateID]*State
	root     *State
	rootName string
}

// StateBuilder provides fluent methods for configuring individual states.
type StateBuilder struct {
	b     *MachineBuilder
	state *State
	name  string
}

// NewMachineBuilder creates a new builder for constructing a state machine.
// rootName is the name of the root compound state, and initialStateName is the initial state to enter.
func NewMachineBuilder(rootName, initialStateName string) *MachineBuilder {
	b := &MachineBuilder{
		nextID:   1, // Root gets ID 0
		nameToID: make(map[string]StateID),
		idToName: make(map[StateID]string),
		states:   make(map[StateID]*State),
		rootName: rootName,
	}

	// Create root state
	rootID := StateID(0)
	b.nameToID[rootName] = rootID
	b.idToName[rootID] = rootName
	b.root = &State{
		ID:       rootID,
		Initial:  b.assignID(initialStateName), // Forward ref ok
		Children: make(map[StateID]*State),
	}
	b.states[rootID] = b.root

	return b
}

// State creates or retrieves a state by name.
// Supports dot notation for hierarchical states (e.g., "parent.child").
// If the parent doesn't exist, it will be auto-created as a compound state.
func (b *MachineBuilder) State(name string) *StateBuilder {
	// Handle hierarchical names
	parentPath, _ := splitPath(name)

	// Get or create parent
	parent := b.root
	if parentPath != "" {
		parentID := b.assignID(parentPath)
		parent = b.states[parentID]
		if parent == nil {
			// Auto-create parent as compound
			parent = &State{
				ID:       parentID,
				Children: make(map[StateID]*State),
			}
			b.states[parentID] = parent

			// Link to grandparent
			grandparentPath, _ := splitPath(parentPath)
			if grandparentPath == "" {
				// Parent of root
				b.root.Children[parentID] = parent
				parent.Parent = b.root
			} else {
				grandparentID := b.assignID(grandparentPath)
				if grandparent := b.states[grandparentID]; grandparent != nil {
					if grandparent.Children == nil {
						grandparent.Children = make(map[StateID]*State)
					}
					grandparent.Children[parentID] = parent
					parent.Parent = grandparent
				}
			}
		}
	}

	// Get or create state
	id := b.assignID(name)
	state := b.states[id]
	if state == nil {
		state = &State{ID: id}
		b.states[id] = state

		// Add to parent
		if parent.Children == nil {
			parent.Children = make(map[StateID]*State)
		}
		parent.Children[id] = state
		state.Parent = parent
	}

	return &StateBuilder{b: b, state: state, name: name}
}

// Build validates the state machine configuration and constructs the Machine.
// Returns an error if the configuration is invalid.
func (b *MachineBuilder) Build() (*Machine, error) {
	// Validate: all referenced states exist
	if err := b.validate(); err != nil {
		return nil, err
	}

	// Use existing NewMachine (tested)
	return NewMachine(b.root)
}

// GetID returns the assigned StateID for a given state name.
// Returns 0 if the name hasn't been registered.
func (b *MachineBuilder) GetID(name string) StateID {
	return b.nameToID[name]
}

// GetName returns the name for a given StateID.
// Returns empty string if the ID doesn't exist.
func (b *MachineBuilder) GetName(id StateID) string {
	return b.idToName[id]
}

// assignID returns the existing ID for a name, or creates a new sequential ID.
// This ensures deterministic ID assignment.
func (b *MachineBuilder) assignID(name string) StateID {
	if id, exists := b.nameToID[name]; exists {
		return id
	}

	id := b.nextID
	b.nextID++
	b.nameToID[name] = id
	b.idToName[id] = name
	return id
}

// validate checks that the state machine configuration is valid.
func (b *MachineBuilder) validate() error {
	// Check that all transition targets exist
	for id, state := range b.states {
		for _, trans := range state.Transitions {
			if trans.Target != 0 { // 0 is internal transition
				if _, exists := b.states[trans.Target]; !exists {
					return fmt.Errorf("state %s has transition to unknown target state ID %d", b.idToName[id], trans.Target)
				}
			}
		}

		// Check compound states have valid Initial
		if len(state.Children) > 0 && state.Initial == 0 && !state.IsParallel {
			return fmt.Errorf("compound state %s must have an initial state", b.idToName[id])
		}

		if state.Initial != 0 {
			if _, exists := b.states[state.Initial]; !exists {
				return fmt.Errorf("state %s has invalid initial state ID %d", b.idToName[id], state.Initial)
			}
		}
	}

	return nil
}

// splitPath splits a hierarchical path into parent and name components.
// For example, "parent.child" returns ("parent", "child").
// For "child", returns ("", "child").
func splitPath(path string) (parent, name string) {
	idx := strings.LastIndex(path, ".")
	if idx == -1 {
		return "", path
	}
	return path[:idx], path[idx+1:]
}

// StateBuilder fluent methods

// Atomic marks this state as atomic (no children).
// This is the default for states without children.
func (sb *StateBuilder) Atomic() *StateBuilder {
	// State is already atomic by default
	return sb
}

// Compound marks this state as a compound state with the given initial child state.
// The initial state will be entered when this compound state is entered.
func (sb *StateBuilder) Compound(initialStateName string) *StateBuilder {
	initialID := sb.b.assignID(initialStateName)
	sb.state.Initial = initialID
	if sb.state.Children == nil {
		sb.state.Children = make(map[StateID]*State)
	}
	return sb
}

// Parallel marks this state as a parallel state.
// All child states will be active concurrently when this state is entered.
func (sb *StateBuilder) Parallel() *StateBuilder {
	sb.state.IsParallel = true
	if sb.state.Children == nil {
		sb.state.Children = make(map[StateID]*State)
	}
	return sb
}

// Final marks this state as a final state with optional data.
// When entered, this state will generate a done event.
func (sb *StateBuilder) Final(data any) *StateBuilder {
	sb.state.IsFinal = true
	sb.state.FinalStateData = data
	return sb
}

// History marks this state as a history pseudo-state.
// historyType should be HistoryShallow or HistoryDeep.
// defaultStateName is the state to enter if no history exists.
func (sb *StateBuilder) History(historyType HistoryType, defaultStateName string) *StateBuilder {
	sb.state.IsHistoryState = true
	sb.state.HistoryType = historyType
	sb.state.HistoryDefault = sb.b.assignID(defaultStateName)
	return sb
}

// Entry sets the entry action for this state.
// The action will be executed when entering this state.
func (sb *StateBuilder) Entry(action Action) *StateBuilder {
	sb.state.EntryAction = action
	return sb
}

// Exit sets the exit action for this state.
// The action will be executed when exiting this state.
func (sb *StateBuilder) Exit(action Action) *StateBuilder {
	sb.state.ExitAction = action
	return sb
}

// InitialAction sets the initial action for this state.
// The action will be executed when first entering this state.
func (sb *StateBuilder) InitialAction(action Action) *StateBuilder {
	sb.state.InitialAction = action
	return sb
}

// On adds a transition from this state to the target state when the given event occurs.
// eventName is the string name of the event (will be prefixed with "event:" internally).
// targetName is the name of the target state.
// guard is an optional guard condition (can be nil).
// action is an optional transition action (can be nil).
func (sb *StateBuilder) On(eventName string, targetName string, guard Guard, action Action) *StateBuilder {
	// Namespace events with "event:" prefix
	eventID := EventID(sb.b.assignID("event:" + eventName))
	targetID := sb.b.assignID(targetName)

	transition := &Transition{
		Event:  eventID,
		Source: sb.state,
		Target: targetID,
		Guard:  guard,
		Action: action,
	}

	sb.state.Transitions = append(sb.state.Transitions, transition)
	return sb
}

// OnInternal adds an internal transition that doesn't change state.
// The transition action executes but no exit/entry actions are triggered.
func (sb *StateBuilder) OnInternal(eventName string, guard Guard, action Action) *StateBuilder {
	eventID := EventID(sb.b.assignID("event:" + eventName))

	transition := &Transition{
		Event:  eventID,
		Source: sb.state,
		Target: 0, // 0 indicates internal transition
		Guard:  guard,
		Action: action,
	}

	sb.state.Transitions = append(sb.state.Transitions, transition)
	return sb
}
