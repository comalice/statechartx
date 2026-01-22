package statechartx

import (
	"context"
	"testing"
)

// BenchmarkStateTransition measures the time for a single state transition
// Target: < 1μs per transition
func BenchmarkStateTransition(b *testing.B) {
	const (
		STATE1   StateID = 1
		STATE2   StateID = 2
		EVENT_GO EventID = 1
	)

	state1 := &State{ID: STATE1, Transitions: []*Transition{}}
	state2 := &State{ID: STATE2, Transitions: []*Transition{}}
	state1.Transitions = append(state1.Transitions, &Transition{Event: EVENT_GO, Source: state1, Target: STATE2})
	state2.Transitions = append(state2.Transitions, &Transition{Event: EVENT_GO, Source: state2, Target: STATE1})

	root := &State{
		ID:       100,
		Initial:  STATE1,
		Children: map[StateID]*State{STATE1: state1, STATE2: state2},
	}
	state1.Parent = root
	state2.Parent = root

	machine, err := NewMachine(root)
	if err != nil {
		b.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		b.Fatalf("Failed to start: %v", err)
	}
	defer rt.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.SendEvent(ctx, Event{ID: EVENT_GO})
	}
}

// BenchmarkEventSending measures the time to send an event
// Target: < 500ns per event
func BenchmarkEventSending(b *testing.B) {
	const (
		STATE1 StateID = 1
		STATE2 StateID = 2
		EVENT1 EventID = 1
	)

	state1 := &State{ID: STATE1, Transitions: []*Transition{}}
	state2 := &State{ID: STATE2, Transitions: []*Transition{}}
	state1.Transitions = append(state1.Transitions, &Transition{Event: EVENT1, Source: state1, Target: STATE2})

	root := &State{
		ID:       100,
		Initial:  STATE1,
		Children: map[StateID]*State{STATE1: state1, STATE2: state2},
	}
	state1.Parent = root
	state2.Parent = root

	machine, err := NewMachine(root)
	if err != nil {
		b.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		b.Fatalf("Failed to start: %v", err)
	}
	defer rt.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.SendEvent(ctx, Event{ID: EVENT1})
	}
}

// BenchmarkLCAComputation measures LCA computation time
// Target: < 100ns for shallow hierarchies
func BenchmarkLCAComputation(b *testing.B) {
	root := &State{ID: 1, Children: make(map[StateID]*State)}
	parent := &State{ID: 2, Parent: root, Children: make(map[StateID]*State)}
	child1 := &State{ID: 3, Parent: parent}
	child2 := &State{ID: 4, Parent: parent}

	root.Children[parent.ID] = parent
	parent.Children[child1.ID] = child1
	parent.Children[child2.ID] = child2

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = computeLCA(child1, child2)
	}
}

// BenchmarkLCAComputationDeep measures LCA computation for deep hierarchies
func BenchmarkLCAComputationDeep(b *testing.B) {
	root := &State{ID: 1, Children: make(map[StateID]*State)}
	current := root

	// Create 100-level deep hierarchy
	for i := 0; i < 100; i++ {
		child := &State{
			ID:       StateID(i + 2),
			Parent:   current,
			Children: make(map[StateID]*State),
		}
		current.Children[child.ID] = child
		current = child
	}

	deepState := current

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = computeLCA(deepState, root)
	}
}

// BenchmarkParallelRegionSpawn measures time to spawn parallel regions
// Target: < 1ms for 10 regions
func BenchmarkParallelRegionSpawn(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		root := &State{
			ID:         1,
			IsParallel: true,
			Children:   make(map[StateID]*State),
		}

		// Create 10 parallel regions
		for j := 0; j < 10; j++ {
			region := &State{
				ID:       StateID(j + 2),
				Parent:   root,
				Children: make(map[StateID]*State),
			}
			state := &State{
				ID:     StateID(j + 100),
				Parent: region,
			}
			region.Children[state.ID] = state
			region.Initial = state.ID
			root.Children[region.ID] = region
		}

		b.StartTimer()
		machine, err := NewMachine(root)
		if err != nil {
			b.Fatalf("Failed to create machine: %v", err)
		}
		rt := NewRuntime(machine, nil)
		ctx := context.Background()
		if err := rt.Start(ctx); err != nil {
			b.Fatalf("Failed to start: %v", err)
		}
		rt.Stop()
	}
}

// BenchmarkParallelRegionSpawn100 measures time for 100 parallel regions
func BenchmarkParallelRegionSpawn100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		root := &State{
			ID:         1,
			IsParallel: true,
			Children:   make(map[StateID]*State),
		}

		for j := 0; j < 100; j++ {
			region := &State{
				ID:       StateID(j + 2),
				Parent:   root,
				Children: make(map[StateID]*State),
			}
			state := &State{
				ID:     StateID(j + 1000),
				Parent: region,
			}
			region.Children[state.ID] = state
			region.Initial = state.ID
			root.Children[region.ID] = region
		}

		b.StartTimer()
		machine, err := NewMachine(root)
		if err != nil {
			b.Fatalf("Failed to create machine: %v", err)
		}
		rt := NewRuntime(machine, nil)
		ctx := context.Background()
		if err := rt.Start(ctx); err != nil {
			b.Fatalf("Failed to start: %v", err)
		}
		rt.Stop()
	}
}

// BenchmarkEventRouting measures event routing time across parallel regions
func BenchmarkEventRouting(b *testing.B) {
	const EVENT1 EventID = 1

	root := &State{
		ID:         1,
		IsParallel: true,
		Children:   make(map[StateID]*State),
	}

	// Create 10 parallel regions with transitions
	for i := 0; i < 10; i++ {
		region := &State{
			ID:       StateID(i + 2),
			Parent:   root,
			Children: make(map[StateID]*State),
		}

		state1 := &State{ID: StateID(i*2 + 100), Parent: region, Transitions: []*Transition{}}
		state2 := &State{ID: StateID(i*2 + 101), Parent: region, Transitions: []*Transition{}}

		state1.Transitions = append(state1.Transitions, &Transition{Event: EVENT1, Source: state1, Target: state2.ID})

		region.Children[state1.ID] = state1
		region.Children[state2.ID] = state2
		region.Initial = state1.ID
		root.Children[region.ID] = region
	}

	machine, err := NewMachine(root)
	if err != nil {
		b.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		b.Fatalf("Failed to start: %v", err)
	}
	defer rt.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.SendEvent(ctx, Event{ID: EVENT1})
	}
}

// BenchmarkHistoryRestoration measures history state restoration time
func BenchmarkHistoryRestoration(b *testing.B) {
	const (
		PARENT       StateID = 1
		CHILD1       StateID = 2
		CHILD2       StateID = 3
		OUTSIDE      StateID = 4
		EVENT_OUT    EventID = 1
		EVENT_BACK   EventID = 2
		EVENT_TOGGLE EventID = 3
	)

	parent := &State{
		ID:             PARENT,
		Children:       make(map[StateID]*State),
		IsHistoryState: true,
		HistoryType:    HistoryShallow,
		Initial:        CHILD1,
	}

	child1 := &State{ID: CHILD1, Parent: parent, Transitions: []*Transition{}}
	child2 := &State{ID: CHILD2, Parent: parent, Transitions: []*Transition{}}
	outside := &State{ID: OUTSIDE, Transitions: []*Transition{}}

	child1.Transitions = append(child1.Transitions, &Transition{Event: EVENT_OUT, Source: child1, Target: OUTSIDE})
	child1.Transitions = append(child1.Transitions, &Transition{Event: EVENT_TOGGLE, Source: child1, Target: CHILD2})
	outside.Transitions = append(outside.Transitions, &Transition{Event: EVENT_BACK, Source: outside, Target: PARENT})

	parent.Children[CHILD1] = child1
	parent.Children[CHILD2] = child2

	root := &State{
		ID:       100,
		Initial:  PARENT,
		Children: map[StateID]*State{PARENT: parent, OUTSIDE: outside},
	}
	parent.Parent = root
	outside.Parent = root

	machine, err := NewMachine(root)
	if err != nil {
		b.Fatalf("Failed to create machine: %v", err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		b.Fatalf("Failed to start: %v", err)
	}
	defer rt.Stop()

	// Transition to child2, then out
	rt.SendEvent(ctx, Event{ID: EVENT_TOGGLE})
	rt.SendEvent(ctx, Event{ID: EVENT_OUT})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.SendEvent(ctx, Event{ID: EVENT_BACK})
		rt.SendEvent(ctx, Event{ID: EVENT_OUT})
	}
}

// BenchmarkStateCreation measures state creation time
func BenchmarkStateCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		root := &State{
			ID:       1,
			Children: make(map[StateID]*State),
		}

		for j := 0; j < 100; j++ {
			state := &State{
				ID:     StateID(j + 2),
				Parent: root,
			}
			root.Children[state.ID] = state
		}
	}
}

// BenchmarkTransitionCreation measures transition creation time
func BenchmarkTransitionCreation(b *testing.B) {
	root := &State{
		ID:       1,
		Children: make(map[StateID]*State),
	}

	states := make([]*State, 100)
	for i := 0; i < 100; i++ {
		states[i] = &State{
			ID:          StateID(i + 2),
			Parent:      root,
			Transitions: []*Transition{},
		}
		root.Children[states[i].ID] = states[i]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 99; j++ {
			trans := &Transition{
				Event:  EventID(j + 1),
				Source: states[j],
				Target: states[j+1].ID,
			}
			states[j].Transitions = append(states[j].Transitions, trans)
		}
	}
}

// BenchmarkComplexStatechart measures performance of a realistic complex statechart
func BenchmarkComplexStatechart(b *testing.B) {
	const EVENT_NEXT EventID = 1

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		root := &State{
			ID:         1,
			IsParallel: true,
			Children:   make(map[StateID]*State),
		}

		stateID := StateID(2)

		// Create a complex structure: 5 parallel regions, each with 10 states
		for r := 0; r < 5; r++ {
			region := &State{
				ID:       stateID,
				Parent:   root,
				Children: make(map[StateID]*State),
			}
			root.Children[stateID] = region
			stateID++

			states := make([]*State, 10)
			for s := 0; s < 10; s++ {
				states[s] = &State{
					ID:          stateID,
					Parent:      region,
					Transitions: []*Transition{},
				}
				region.Children[stateID] = states[s]
				stateID++
			}

			// Add transitions
			for s := 0; s < 9; s++ {
				states[s].Transitions = append(states[s].Transitions,
					&Transition{Event: EVENT_NEXT, Source: states[s], Target: states[s+1].ID})
			}

			region.Initial = states[0].ID
		}

		machine, err := NewMachine(root)
		if err != nil {
			b.Fatalf("Failed to create machine: %v", err)
		}

		rt := NewRuntime(machine, nil)
		ctx := context.Background()

		b.StartTimer()
		if err := rt.Start(ctx); err != nil {
			b.Fatalf("Failed to start: %v", err)
		}

		// Process some events
		for j := 0; j < 10; j++ {
			rt.SendEvent(ctx, Event{ID: EVENT_NEXT})
		}

		rt.Stop()
	}
}

// BenchmarkMemoryAllocation measures memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ReportAllocs()

	const (
		STATE1 StateID = 1
		STATE2 StateID = 2
		EVENT1 EventID = 1
	)

	for i := 0; i < b.N; i++ {
		state1 := &State{ID: STATE1, Transitions: []*Transition{}}
		state2 := &State{ID: STATE2, Transitions: []*Transition{}}
		state1.Transitions = append(state1.Transitions, &Transition{Event: EVENT1, Source: state1, Target: STATE2})

		root := &State{
			ID:       100,
			Initial:  STATE1,
			Children: map[StateID]*State{STATE1: state1, STATE2: state2},
		}
		state1.Parent = root
		state2.Parent = root

		machine, _ := NewMachine(root)
		rt := NewRuntime(machine, nil)
		ctx := context.Background()
		rt.Start(ctx)
		rt.SendEvent(ctx, Event{ID: EVENT1})
		rt.Stop()
	}
}
