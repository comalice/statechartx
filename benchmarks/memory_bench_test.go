// Package benchmarks provides memory footprint benchmarks.
package benchmarks

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/comalice/statechartx/internal/core"
	"github.com/comalice/statechartx/internal/primitives"
)

func memorySimpleConfig() primitives.MachineConfig {
	idle := primitives.NewStateConfig("idle", primitives.Atomic)
	return primitives.MachineConfig{
		ID:      "simple",
		Initial: "idle",
		States: map[string]*primitives.StateConfig{
			"idle": idle,
		},
	}
}

func BenchmarkMemoryFootprint(b *testing.B) {
	config := memorySimpleConfig()
	if err := config.Validate(); err != nil {
		b.Fatal(err)
	}
	numMachines := 1000
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	machines := make([]*core.Machine, numMachines)
	for i := 0; i < numMachines; i++ {
		machines[i] = core.NewMachine(config)
	}
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	bytesPerMachine := (after.TotalAlloc - before.TotalAlloc) / uint64(numMachines)
	b.ReportMetric(float64(bytesPerMachine)/1024/1024, "MB/machine")
}

func BenchmarkMemoryFlat(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("states=%d", n), func(b *testing.B) {
			config := GenFlatConfig(n)
			if err := config.Validate(); err != nil {
				b.Fatal(err)
			}
			numMachines := 100
			var before runtime.MemStats
			runtime.ReadMemStats(&before)
			machines := make([]*core.Machine, numMachines)
			for i := 0; i < numMachines; i++ {
				machines[i] = core.NewMachine(config)
			}
			runtime.GC()
			var after runtime.MemStats
			runtime.ReadMemStats(&after)
			bytesPerMachine := (after.TotalAlloc - before.TotalAlloc) / uint64(numMachines)
			bytesPerState := bytesPerMachine / uint64(n)
			b.ReportMetric(float64(bytesPerMachine)/1024/1024, "MB/machine")
			b.ReportMetric(float64(bytesPerState)/1024, "KB/state")
		})
	}
}

func BenchmarkMemoryDeep(b *testing.B) {
	for _, depth := range []int{1, 3, 5} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			config := GenDeepConfig(depth)
			if err := config.Validate(); err != nil {
				b.Fatal(err)
			}
			// Approximate num_states = 2*depth + 1 (leaves + compounds)
			numStates := 2*depth + 1
			numMachines := 100
			var before runtime.MemStats
			runtime.ReadMemStats(&before)
			machines := make([]*core.Machine, numMachines)
			for i := 0; i < numMachines; i++ {
				machines[i] = core.NewMachine(config)
			}
			runtime.GC()
			var after runtime.MemStats
			runtime.ReadMemStats(&after)
			bytesPerMachine := (after.TotalAlloc - before.TotalAlloc) / uint64(numMachines)
			bytesPerState := bytesPerMachine / uint64(numStates)
			b.ReportMetric(float64(bytesPerMachine)/1024/1024, "MB/machine")
			b.ReportMetric(float64(bytesPerState)/1024, "KB/state")
		})
	}
}
