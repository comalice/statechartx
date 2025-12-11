// Package benchmarks provides memory footprint benchmarks.
package benchmarks

import (
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
