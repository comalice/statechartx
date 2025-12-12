// Package benchmarks provides performance benchmarks for event throughput.
package benchmarks

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/comalice/statechartx/internal/core"
	"github.com/comalice/statechartx/internal/primitives"
)

func throughputConfig(action primitives.ActionRef) primitives.MachineConfig {
	idle := primitives.NewStateConfig("idle", primitives.Atomic)
	idle.AddTransition("tick", primitives.TransitionConfig{
		Target:  "idle",
		Actions: []primitives.ActionRef{action},
	})
	return primitives.MachineConfig{
		ID:      "throughput",
		Initial: "idle",
		States: map[string]*primitives.StateConfig{
			"idle": idle,
		},
	}
}

func BenchmarkEventThroughput(b *testing.B) {
	var processed int64
	action := func(ctx *primitives.Context, e primitives.Event) {
		atomic.AddInt64(&processed, 1)
	}
	config := throughputConfig(action)
	if err := config.Validate(); err != nil {
		b.Fatal(err)
	}
	m := core.NewMachine(config, core.WithQueueSize(10000))
	if err := m.Start(); err != nil {
		b.Fatal(err)
	}
	defer m.Stop()
	e := primitives.NewEvent("tick", nil)
	numWorkers := 8
	eventsPerWorker := b.N / numWorkers
	if eventsPerWorker == 0 {
		eventsPerWorker = 1
	}
	var wg sync.WaitGroup
	b.ResetTimer()
	b.ReportAllocs()
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < eventsPerWorker; i++ {
				m.Send(e)
			}
		}()
	}
	wg.Wait()
	// Wait for processing
	timeout := time.After(30 * time.Second)
	for {
		if atomic.LoadInt64(&processed) >= int64(b.N) {
			break
		}
		select {
		case <-timeout:
			b.Fatalf("timeout waiting for processing, processed: %d / %d", atomic.LoadInt64(&processed), b.N)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/second")
}

func BenchmarkEventThroughputGuarded(b *testing.B) {
	var processed int64
	guard := func(ctx *primitives.Context, e primitives.Event) bool {
		return true
	}
	action := func(ctx *primitives.Context, e primitives.Event) {
		atomic.AddInt64(&processed, 1)
	}
	idle := primitives.NewStateConfig("idle", primitives.Atomic)
	idle.AddTransition("tick", primitives.TransitionConfig{
		Target:  "idle",
		Guard:   guard,
		Actions: []primitives.ActionRef{action},
	})
	config := primitives.MachineConfig{
		ID:      "throughput_guarded",
		Initial: "idle",
		States: map[string]*primitives.StateConfig{
			"idle": idle,
		},
	}
	if err := config.Validate(); err != nil {
		b.Fatal(err)
	}
	m := core.NewMachine(config, core.WithQueueSize(10000))
	if err := m.Start(); err != nil {
		b.Fatal(err)
	}
	defer m.Stop()
	e := primitives.NewEvent("tick", nil)
	numWorkers := 8
	eventsPerWorker := b.N / numWorkers
	if eventsPerWorker == 0 {
		eventsPerWorker = 1
	}
	var wg sync.WaitGroup
	b.ResetTimer()
	b.ReportAllocs()
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < eventsPerWorker; i++ {
				m.Send(e)
			}
		}()
	}
	wg.Wait()
	timeout := time.After(30 * time.Second)
	for {
		if atomic.LoadInt64(&processed) >= int64(b.N) {
			break
		}
		select {
		case <-timeout:
			b.Fatalf("timeout waiting for processing, processed: %d / %d", atomic.LoadInt64(&processed), b.N)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/second")
}

func BenchmarkEventThroughputDeep(b *testing.B) {
	config := GenDeepConfig(5)
	if err := config.Validate(); err != nil {
		b.Fatal(err)
	}
	m := core.NewMachine(config, core.WithQueueSize(10000))
	if err := m.Start(); err != nil {
		b.Fatal(err)
	}
	defer m.Stop()
	e := primitives.NewEvent("tick", nil)
	numWorkers := 8
	eventsPerWorker := b.N / numWorkers
	if eventsPerWorker == 0 {
		eventsPerWorker = 1
	}
	var wg sync.WaitGroup
	b.ResetTimer()
	b.ReportAllocs()
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < eventsPerWorker; i++ {
				m.Send(e)
			}
		}()
	}
	wg.Wait()
	// Approximate drain time for processing
	time.Sleep(100 * time.Millisecond)
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/second")
}
