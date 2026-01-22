package benchmarks

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/comalice/statechartx"
	"github.com/comalice/statechartx/realtime"
)

// Honest Realtime Runtime Benchmarks
//
// These benchmarks measure actual system performance and behavior:
// - Throughput: Events actually processed per second (verified via action counters)
// - Latency: Real end-to-end time from SendEvent to state transition
// - Queue Capacity: Actual queue limits before backpressure
// - Tick Processing: Time to process event batches within a tick

const (
	STATE_ROOT statechartx.StateID = 0
	STATE_A    statechartx.StateID = 1
	STATE_B    statechartx.StateID = 2
	EVENT_1    statechartx.EventID = 1
)

func createBenchmarkMachine() *statechartx.Machine {
	stateA := &statechartx.State{
		ID: STATE_A,
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_1,
				Target: STATE_B,
			},
		},
	}
	stateB := &statechartx.State{
		ID: STATE_B,
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_1,
				Target: STATE_A,
			},
		},
	}

	root := &statechartx.State{
		ID:      STATE_ROOT,
		Initial: STATE_A,
		Children: map[statechartx.StateID]*statechartx.State{
			STATE_A: stateA,
			STATE_B: stateB,
		},
	}

	machine, _ := statechartx.NewMachine(root)
	return machine
}

// BenchmarkRealtimeThroughput measures actual events processed per second
// with verification that events were actually executed by the state machine
func BenchmarkRealtimeThroughput(b *testing.B) {
	var processed int64

	// Create machine with action counter
	stateA := &statechartx.State{
		ID: STATE_A,
		EntryAction: func(ctx context.Context, evt *statechartx.Event, from statechartx.StateID, to statechartx.StateID) error {
			atomic.AddInt64(&processed, 1)
			return nil
		},
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_1,
				Target: STATE_B,
			},
		},
	}
	stateB := &statechartx.State{
		ID: STATE_B,
		EntryAction: func(ctx context.Context, evt *statechartx.Event, from statechartx.StateID, to statechartx.StateID) error {
			atomic.AddInt64(&processed, 1)
			return nil
		},
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_1,
				Target: STATE_A,
			},
		},
	}

	root := &statechartx.State{
		ID:      STATE_ROOT,
		Initial: STATE_A,
		Children: map[statechartx.StateID]*statechartx.State{
			STATE_A: stateA,
			STATE_B: stateB,
		},
	}

	machine, err := statechartx.NewMachine(root)
	if err != nil {
		b.Fatal(err)
	}

	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate:         1 * time.Millisecond, // 1000 Hz
		MaxEventsPerTick: 10000,
	})

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		b.Fatal(err)
	}
	defer rt.Stop()

	b.ResetTimer()
	b.ReportAllocs()

	successfulSends := 0
	for i := 0; i < b.N; i++ {
		if err := rt.SendEvent(statechartx.Event{ID: EVENT_1}); err != nil {
			// Hit backpressure - stop benchmark
			b.StopTimer()
			b.Logf("Stopped at backpressure after %d events (%.1f%% of b.N)",
				successfulSends, float64(successfulSends)/float64(b.N)*100)
			break
		}
		successfulSends++
	}

	// Wait for processing to complete
	if successfulSends > 0 {
		timeout := time.After(30 * time.Second)
		for {
			if atomic.LoadInt64(&processed) >= int64(successfulSends) {
				break
			}
			select {
			case <-timeout:
				b.Fatalf("timeout waiting for processing, processed: %d / %d successful sends",
					atomic.LoadInt64(&processed), successfulSends)
			default:
				time.Sleep(1 * time.Millisecond)
			}
		}
		b.ReportMetric(float64(successfulSends)/b.Elapsed().Seconds(), "events/sec")
	}
}

// BenchmarkRealtimeLatency measures time from SendEvent to actual state transition
// This measures the real latency including tick scheduling overhead
func BenchmarkRealtimeLatency(b *testing.B) {
	// Channel to signal when transition happens (buffered for all transitions)
	transitioned := make(chan time.Time, 100)
	var sendTimes []time.Time
	var sendMu sync.Mutex

	stateA := &statechartx.State{
		ID: STATE_A,
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_1,
				Target: STATE_B,
			},
		},
	}
	stateB := &statechartx.State{
		ID: STATE_B,
		EntryAction: func(ctx context.Context, evt *statechartx.Event, from statechartx.StateID, to statechartx.StateID) error {
			// Signal that transition completed
			transitioned <- time.Now()
			return nil
		},
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_1,
				Target: STATE_A,
			},
		},
	}

	root := &statechartx.State{
		ID:      STATE_ROOT,
		Initial: STATE_A,
		Children: map[statechartx.StateID]*statechartx.State{
			STATE_A: stateA,
			STATE_B: stateB,
		},
	}

	machine, err := statechartx.NewMachine(root)
	if err != nil {
		b.Fatal(err)
	}

	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate:         1 * time.Millisecond, // 1000 Hz
		MaxEventsPerTick: 1000,
	})

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		b.Fatal(err)
	}
	defer rt.Stop()

	b.ResetTimer()

	// Send all events first, recording send times
	for i := 0; i < b.N && i < 50; i++ { // Limit to 50 iterations for latency test
		sendMu.Lock()
		sendTimes = append(sendTimes, time.Now())
		sendMu.Unlock()

		if err := rt.SendEvent(statechartx.Event{ID: EVENT_1}); err != nil {
			// Hit backpressure - stop sending
			b.Logf("Stopped at backpressure after %d sends", len(sendTimes))
			break
		}
	}

	// Wait for all transitions and measure latencies
	var totalLatency time.Duration
	successfulMeasurements := 0
	timeout := time.After(5 * time.Second)

	for i := 0; i < len(sendTimes); i++ {
		select {
		case completeTime := <-transitioned:
			if i < len(sendTimes) {
				latency := completeTime.Sub(sendTimes[i])
				totalLatency += latency
				successfulMeasurements++
			}
		case <-timeout:
			b.Logf("timeout after %d/%d measurements", successfulMeasurements, len(sendTimes))
			goto done
		}
	}

done:
	if successfulMeasurements > 0 {
		avgLatency := totalLatency / time.Duration(successfulMeasurements)
		b.ReportMetric(float64(avgLatency.Nanoseconds()), "ns/latency")
		b.ReportMetric(float64(avgLatency.Microseconds()), "µs/latency")
		b.ReportMetric(float64(avgLatency.Milliseconds()), "ms/latency")
	}
}

// BenchmarkRealtimeQueueCapacity measures how many events can be queued
// before hitting backpressure, showing the practical queue limit
func BenchmarkRealtimeQueueCapacity(b *testing.B) {
	machine := createBenchmarkMachine()

	// Test different tick rates to see how capacity varies
	configs := []struct {
		name         string
		tickRate     time.Duration
		maxPerTick   int
	}{
		{"60FPS", 16667 * time.Microsecond, 10000},
		{"1000Hz", 1 * time.Millisecond, 10000},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			rt := realtime.NewRuntime(machine, realtime.Config{
				TickRate:         cfg.tickRate,
				MaxEventsPerTick: cfg.maxPerTick,
			})

			ctx := context.Background()
			if err := rt.Start(ctx); err != nil {
				b.Fatal(err)
			}
			defer rt.Stop()

			b.ResetTimer()

			successfulSends := 0
			for i := 0; i < b.N; i++ {
				if err := rt.SendEvent(statechartx.Event{ID: EVENT_1}); err != nil {
					// Hit backpressure - this is what we're measuring
					b.StopTimer()
					b.Logf("Queue capacity reached: %d events before backpressure", successfulSends)
					b.ReportMetric(float64(successfulSends), "events")
					return
				}
				successfulSends++
			}

			// If we got through all b.N without backpressure, report that
			b.ReportMetric(float64(successfulSends), "events")
			b.Logf("Sent all %d events without backpressure", successfulSends)
		})
	}
}

// BenchmarkRealtimeTickProcessing measures how long it takes to process
// a batch of events in a single tick
func BenchmarkRealtimeTickProcessing(b *testing.B) {
	var tickStartTime int64 // Unix nano, 0 = not set
	var tickEndTime int64   // Unix nano
	var tickDurations []time.Duration
	var tickMu sync.Mutex

	stateA := &statechartx.State{
		ID: STATE_A,
		EntryAction: func(ctx context.Context, evt *statechartx.Event, from statechartx.StateID, to statechartx.StateID) error {
			// Record when first event in tick starts processing
			if atomic.LoadInt64(&tickStartTime) == 0 {
				atomic.StoreInt64(&tickStartTime, time.Now().UnixNano())
			}
			return nil
		},
		ExitAction: func(ctx context.Context, evt *statechartx.Event, from statechartx.StateID, to statechartx.StateID) error {
			// Record when last event in tick finishes processing
			atomic.StoreInt64(&tickEndTime, time.Now().UnixNano())
			return nil
		},
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_1,
				Target: STATE_B,
			},
		},
	}
	stateB := &statechartx.State{
		ID: STATE_B,
		ExitAction: func(ctx context.Context, evt *statechartx.Event, from statechartx.StateID, to statechartx.StateID) error {
			atomic.StoreInt64(&tickEndTime, time.Now().UnixNano())
			return nil
		},
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_1,
				Target: STATE_A,
			},
		},
	}

	root := &statechartx.State{
		ID:      STATE_ROOT,
		Initial: STATE_A,
		Children: map[statechartx.StateID]*statechartx.State{
			STATE_A: stateA,
			STATE_B: stateB,
		},
	}

	machine, err := statechartx.NewMachine(root)
	if err != nil {
		b.Fatal(err)
	}

	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate:         10 * time.Millisecond, // Slower tick to allow batching
		MaxEventsPerTick: 1000,
	})

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		b.Fatal(err)
	}
	defer rt.Stop()

	b.ResetTimer()

	// Send events in bursts to fill up batches
	batchSize := 100
	for i := 0; i < b.N; i++ {
		atomic.StoreInt64(&tickStartTime, 0)
		atomic.StoreInt64(&tickEndTime, 0)

		// Send a batch of events
		for j := 0; j < batchSize; j++ {
			if err := rt.SendEvent(statechartx.Event{ID: EVENT_1}); err != nil {
				b.Logf("Backpressure at iteration %d, event %d", i, j)
				goto done
			}
		}

		// Wait for tick to process
		time.Sleep(15 * time.Millisecond)

		// Measure tick duration
		startNano := atomic.LoadInt64(&tickStartTime)
		endNano := atomic.LoadInt64(&tickEndTime)
		if startNano > 0 && endNano > 0 {
			duration := time.Duration(endNano - startNano)
			tickMu.Lock()
			tickDurations = append(tickDurations, duration)
			tickMu.Unlock()
		}
	}

done:
	if len(tickDurations) > 0 {
		var total time.Duration
		for _, d := range tickDurations {
			total += d
		}
		avgDuration := total / time.Duration(len(tickDurations))
		b.ReportMetric(float64(avgDuration.Nanoseconds()), "ns/tick")
		b.ReportMetric(float64(avgDuration.Microseconds()), "µs/tick")
		b.ReportMetric(float64(batchSize), "events/tick")
	}
}
