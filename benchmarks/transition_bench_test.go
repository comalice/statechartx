// Package benchmarks provides performance benchmarks for the statechart engine core transitions.
package benchmarks

import (
	"testing"

	"github.com/comalice/statechartx/internal/core"
	"github.com/comalice/statechartx/internal/primitives"
)

func simpleConfig() primitives.MachineConfig {
	idle := primitives.NewStateConfig("idle", primitives.Atomic)
	idle.AddTransition("tick", primitives.TransitionConfig{
		Target: "idle", // self-loop for consistent simple transition
	})
	return primitives.MachineConfig{
		ID:      "simple",
		Initial: "idle",
		States: map[string]*primitives.StateConfig{
			"idle": idle,
		},
	}
}

func BenchmarkSimpleTransition(b *testing.B) {
	config := simpleConfig()
	if err := config.Validate(); err != nil {
		b.Fatal(err)
	}
	m := core.NewMachine(config, core.WithQueueSize(100000))
	if err := m.Start(); err != nil {
		b.Fatal(err)
	}
	e := primitives.NewEvent("tick", nil)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := m.Send(e); err != nil {
			b.Fatal(err)
		}
	}
}

func hierarchicalConfig() primitives.MachineConfig {
	leaf1 := primitives.NewStateConfig("leaf1", primitives.Atomic)
	leaf1.AddTransition("tick", primitives.TransitionConfig{
		Target: "leaf2",
	})
	leaf2 := primitives.NewStateConfig("leaf2", primitives.Atomic)
	leaf2.AddTransition("tick", primitives.TransitionConfig{
		Target: "leaf1",
	})
	parent := primitives.NewStateConfig("parent", primitives.Compound)
	parent.Initial = "leaf1"
	parent.Children = []*primitives.StateConfig{leaf1, leaf2}
	return primitives.MachineConfig{
		ID:      "hier",
		Initial: "parent",
		States: map[string]*primitives.StateConfig{
			"parent": parent,
			"leaf1":  leaf1,
			"leaf2":  leaf2,
		},
	}
}

func BenchmarkHierarchicalTransition(b *testing.B) {
	config := hierarchicalConfig()
	if err := config.Validate(); err != nil {
		b.Fatal(err)
	}
	m := core.NewMachine(config, core.WithQueueSize(100000))
	if err := m.Start(); err != nil {
		b.Fatal(err)
	}
	e := primitives.NewEvent("tick", nil)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := m.Send(e); err != nil {
			b.Fatal(err)
		}
	}
}

func parallelConfig() primitives.MachineConfig {
	region1 := primitives.NewStateConfig("region1", primitives.Atomic)
	region1.AddTransition("tick", primitives.TransitionConfig{
		Target: "region2",
	})
	region2 := primitives.NewStateConfig("region2", primitives.Atomic)
	region2.AddTransition("tick", primitives.TransitionConfig{
		Target: "region1",
	})
	parallel := primitives.NewStateConfig("parallel", primitives.Parallel)
	parallel.Initial = "region1"
	parallel.Children = []*primitives.StateConfig{region1, region2}
	return primitives.MachineConfig{
		ID:      "parallel",
		Initial: "parallel",
		States: map[string]*primitives.StateConfig{
			"parallel": parallel,
			"region1":  region1,
			"region2":  region2,
		},
	}
}

func BenchmarkParallelTransition(b *testing.B) {
	config := parallelConfig()
	if err := config.Validate(); err != nil {
		b.Fatal(err)
	}
	m := core.NewMachine(config, core.WithQueueSize(100000))
	if err := m.Start(); err != nil {
		b.Fatal(err)
	}
	e := primitives.NewEvent("tick", nil)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := m.Send(e); err != nil {
			b.Fatal(err)
		}
	}
}

func guardedConfig() primitives.MachineConfig {
	idle := primitives.NewStateConfig("idle", primitives.Atomic)
	guard := func(ctx *primitives.Context, e primitives.Event) bool {
		return true
	}
	idle.AddTransition("tick", primitives.TransitionConfig{
		Target: "idle",
		Guard:  guard,
	})
	return primitives.MachineConfig{
		ID:      "guarded",
		Initial: "idle",
		States: map[string]*primitives.StateConfig{
			"idle": idle,
		},
	}
}

func BenchmarkGuardedTransition(b *testing.B) {
	config := guardedConfig()
	if err := config.Validate(); err != nil {
		b.Fatal(err)
	}
	m := core.NewMachine(config, core.WithQueueSize(100000))
	if err := m.Start(); err != nil {
		b.Fatal(err)
	}
	e := primitives.NewEvent("tick", nil)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := m.Send(e); err != nil {
			b.Fatal(err)
		}
	}
}
