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
	e := primitives.NewEvent("tick", nil)
	numWorkers := 8
	eventsPerWorker := b.N / numWorkers
	if eventsPerWorker == 0 {
		eventsPerWorker = 1
	}
	var wg sync.WaitGroup
	var successfulSends int64
	var failedSends int64
	b.ResetTimer()
	b.ReportAllocs()
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < eventsPerWorker; i++ {
				if err := m.Send(e); err != nil {
					atomic.AddInt64(&failedSends, 1)
					return // Stop this worker on backpressure
				}
				atomic.AddInt64(&successfulSends, 1)
			}
		}()
	}
	wg.Wait()
	// Check if we hit backpressure
	totalFailed := atomic.LoadInt64(&failedSends)
	totalSuccessful := atomic.LoadInt64(&successfulSends)
	if totalFailed > 0 {
		b.StopTimer()
		b.Logf("Hit backpressure: %d successful, %d failed (%.1f%% of b.N)",
			totalSuccessful, totalFailed, float64(totalSuccessful)/float64(b.N)*100)
	}
	// Wait for processing of successful events only
	if totalSuccessful > 0 {
		timeout := time.After(30 * time.Second)
		for {
			if atomic.LoadInt64(&processed) >= totalSuccessful {
				break
			}
			select {
			case <-timeout:
				b.Fatalf("timeout waiting for processing, processed: %d / %d successful sends",
					atomic.LoadInt64(&processed), totalSuccessful)
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
		b.ReportMetric(float64(totalSuccessful)/b.Elapsed().Seconds(), "events/sec")
	}
}
