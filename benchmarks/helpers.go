// Package benchmarks provides shared helpers for benchmark tests.
package benchmarks

import (
	"fmt"
	"time"

	"github.com/comalice/statechartx/internal/core"
	"github.com/comalice/statechartx/internal/primitives"
	"gopkg.in/yaml.v3"
)

// GenFlatConfig creates a flat machine with n atomic states cycling via "tick" events.
func GenFlatConfig(n int) primitives.MachineConfig {
	if n < 1 {
		n = 1
	}
	config := primitives.MachineConfig{
		ID:      fmt.Sprintf("flat_%d", n),
		Initial: "s0",
		States:  make(map[string]*primitives.StateConfig, n),
	}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("s%d", i)
		sc := primitives.NewStateConfig(id, primitives.Atomic)
		targetIdx := (i + 1) % n
		target := fmt.Sprintf("s%d", targetIdx)
		sc.AddTransition("tick", primitives.TransitionConfig{Target: target})
		config.States[id] = sc
	}
	return config
}

// GenDeepConfig creates a deeply nested hierarchy flipping between leaves at the bottom.
func GenDeepConfig(depth int) primitives.MachineConfig {
	if depth < 1 {
		depth = 1
	}
	mb := primitives.NewMachineBuilder(fmt.Sprintf("deep_%d", depth), "c0")
	sb := mb.Compound("c0")
	sb.WithInitial("leaf1")
	leaf1 := sb.Atomic("leaf1")
	leaf1.Transition("tick", "leaf2")
	leaf2 := sb.Atomic("leaf2")
	leaf2.Transition("tick", "leaf1")
	sb = sb.Up()
	for i := 1; i < depth; i++ {
		sb = sb.Compound(fmt.Sprintf("c%d", i))
		sb.WithInitial("leaf1")
		leaf1 = sb.Atomic("leaf1")
		leaf1.Transition("tick", "leaf2")
		leaf2 = sb.Atomic("leaf2")
		leaf2.Transition("tick", "leaf1")
		sb = sb.Up()
	}
	config := mb.Build()
	return config
}

// GenWideTransitions creates one main state with many outgoing "tick" transitions (prioritized).
func GenWideTransitions(numTransitions int) primitives.MachineConfig {
	if numTransitions < 1 {
		numTransitions = 1
	}
	config := primitives.MachineConfig{
		ID:      fmt.Sprintf("wide_%d", numTransitions),
		Initial: "main",
		States:  make(map[string]*primitives.StateConfig, numTransitions+1),
	}
	main := primitives.NewStateConfig("main", primitives.Atomic)
	for i := 0; i < numTransitions; i++ {
		target := fmt.Sprintf("target%d", i)
		trans := primitives.TransitionConfig{
			Event:    "tick",
			Target:   target,
			Priority: numTransitions - i,
			Guard: func(ctx *primitives.Context, e primitives.Event) bool {
				return i == 0 // only highest priority always fires
			},
		}
		main.AddTransition("tick", trans)
		tsc := primitives.NewStateConfig(target, primitives.Atomic)
		tsc.AddTransition("tick", primitives.TransitionConfig{Target: "main"})
		config.States[target] = tsc
	}
		primitives.SortTransitions(main.On["tick"])
	config.States["main"] = main
	return config
}

// SnapshotFromMachine creates a MachineSnapshot from a running machine.
func SnapshotFromMachine(m *core.Machine) core.MachineSnapshot {
	return core.MachineSnapshot{
		MachineID:   m.Config().ID,
		Config:      m.Config(),
		Current:     m.Current(),
		ContextData: m.Ctx().Snapshot(),
		Timestamp:   time.Now(),
	}
}

// GenSnapshotYAML generates YAML bytes for a snapshot of given size.
func GenSnapshotYAML(numStates int, hierarchical bool) []byte {
	var config primitives.MachineConfig
	if hierarchical {
		config = GenDeepConfig(5)
	} else {
		config = GenFlatConfig(numStates)
	}
	m := core.NewMachine(config)
	if err := m.Start(); err != nil {
		panic(err)
	}
	defer m.Stop()
	// Send one event to mutate state
	e := primitives.NewEvent("tick", nil)
	m.Send(e)
	snap := SnapshotFromMachine(m)
	data, err := yaml.Marshal(snap)
	if err != nil {
		panic(err)
	}
	return data
}
