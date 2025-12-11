// Package primitives includes builder helpers for MachineConfig.
package primitives

// MachineBuilder builds hierarchical MachineConfig fluently.
type MachineBuilder struct {
	config *MachineConfig
	states map[string]*StateConfig
	stack  []*StateConfig // For nesting Up()
}

// NewMachineBuilder creates a new MachineBuilder.
func NewMachineBuilder(id, initial string) *MachineBuilder {
	return &MachineBuilder{
		config: &MachineConfig{ID: id, Initial: initial},
		states: make(map[string]*StateConfig),
	}
}

// Compound starts a compound state (push to stack).
func (b *MachineBuilder) Compound(id string) *StateBuilder {
	s := NewStateConfig(id, Compound)
	b.states[id] = s
	b.stack = append(b.stack, s)
	return &StateBuilder{state: s, mb: b}
}

// Parallel starts a parallel region.
func (b *MachineBuilder) Parallel(id string) *StateBuilder {
	s := NewStateConfig(id, Parallel)
	b.states[id] = s
	b.stack = append(b.stack, s)
	return &StateBuilder{state: s, mb: b}
}

// Atomic starts an atomic state.
func (b *MachineBuilder) Atomic(id string) *StateBuilder {
	s := NewStateConfig(id, Atomic)
	b.states[id] = s
	if len(b.stack) > 0 {
		b.stack[len(b.stack)-1].AddChild(s)
	}
	return &StateBuilder{state: s, mb: b}
}

// History starts a history state (shallow/deep).
func (b *MachineBuilder) History(id string, shallow bool) *StateBuilder {
	typ := ShallowHistory
	if !shallow {
		typ = DeepHistory
	}
	s := NewStateConfig(id, typ)
	b.states[id] = s
	if len(b.stack) > 0 {
		b.stack[len(b.stack)-1].AddChild(s)
	}
	return &StateBuilder{state: s, mb: b}
}

// State sugar for Atomic.
func (b *MachineBuilder) State(id string) *StateBuilder {
	return b.Atomic(id)
}

// StateBuilder for fluent transitions/nesting.
type StateBuilder struct {
	state *StateConfig
	mb    *MachineBuilder
}

// Transition adds transition.
func (sb *StateBuilder) Transition(event, target string, opts ...TransitionConfig) *StateBuilder {
	sb.state.Transition(event, target, opts...)
	return sb
}

// Compound nests compound child.
func (sb *StateBuilder) Compound(id string) *StateBuilder {
	child := sb.state.State(id, Compound)
	sb.mb.states[child.ID] = child
	sb.mb.stack = append(sb.mb.stack, child)
	return &StateBuilder{state: child, mb: sb.mb}
}

// Parallel nests parallel child.
func (sb *StateBuilder) Parallel(id string) *StateBuilder {
	child := sb.state.State(id, Parallel)
	sb.mb.states[child.ID] = child
	sb.mb.stack = append(sb.mb.stack, child)
	return &StateBuilder{state: child, mb: sb.mb}
}

// Atomic/State nests atomic child.
func (sb *StateBuilder) Atomic(id string) *StateBuilder {
	child := sb.state.State(id)
	sb.mb.states[child.ID] = child
	return &StateBuilder{state: child, mb: sb.mb}
}

// History nests history child.
func (sb *StateBuilder) History(id string, shallow bool) *StateBuilder {
	typ := ShallowHistory
	if !shallow {
		typ = DeepHistory
	}
	child := sb.state.State(id, typ)
	sb.mb.states[child.ID] = child
	return &StateBuilder{state: child, mb: sb.mb}
}

// Up pops stack to parent.
func (sb *StateBuilder) Up() *StateBuilder {
	if len(sb.mb.stack) > 1 {
		sb.mb.stack = sb.mb.stack[:len(sb.mb.stack)-1]
		parent := sb.mb.stack[len(sb.mb.stack)-1]
		return &StateBuilder{state: parent, mb: sb.mb}
	}
	return sb
}

// WithInitial sets initial for current (compound/parallel).
func (sb *StateBuilder) WithInitial(initial string) *StateBuilder {
	sb.state.WithInitial(initial)
	return sb
}

// Build finalizes config (flattens, validates).
func (b *MachineBuilder) Build() MachineConfig {
	if len(b.stack) > 0 {
		b.config.States = b.states
	} else {
		b.config.States = make(map[string]*StateConfig)
	}
	if err := b.config.Validate(); err != nil {
		panic(err) // Or return error
	}
	return *b.config
}
