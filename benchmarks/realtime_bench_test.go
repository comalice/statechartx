package benchmarks

import (
	"context"
	"testing"
	"time"

	"github.com/comalice/statechartx"
	"github.com/comalice/statechartx/realtime"
)

// Benchmark: Event-driven vs Tick-based runtime comparison

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

// BenchmarkEventDrivenRuntime benchmarks the event-driven runtime
func BenchmarkEventDrivenRuntime(b *testing.B) {
	machine := createBenchmarkMachine()
	rt := statechartx.NewRuntime(machine, nil)

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		b.Fatal(err)
	}
	defer rt.Stop()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rt.SendEvent(ctx, statechartx.Event{ID: EVENT_1})
	}
}

// BenchmarkTickBasedRuntime60FPS benchmarks tick-based runtime at 60 FPS
func BenchmarkTickBasedRuntime60FPS(b *testing.B) {
	machine := createBenchmarkMachine()
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate:         16667 * time.Microsecond, // 60 FPS
		MaxEventsPerTick: 10000,
	})

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		b.Fatal(err)
	}
	defer rt.Stop()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rt.SendEvent(statechartx.Event{ID: EVENT_1})
	}

	// Wait for all events to process
	time.Sleep(100 * time.Millisecond)
}

// BenchmarkTickBasedRuntime1000Hz benchmarks tick-based runtime at 1000 Hz
func BenchmarkTickBasedRuntime1000Hz(b *testing.B) {
	machine := createBenchmarkMachine()
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

	for i := 0; i < b.N; i++ {
		rt.SendEvent(statechartx.Event{ID: EVENT_1})
	}

	// Wait for all events to process
	time.Sleep(20 * time.Millisecond)
}

// BenchmarkEventDrivenLatency measures event-driven runtime latency
func BenchmarkEventDrivenLatency(b *testing.B) {
	machine := createBenchmarkMachine()
	rt := statechartx.NewRuntime(machine, nil)

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		b.Fatal(err)
	}
	defer rt.Stop()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		rt.SendEvent(ctx, statechartx.Event{ID: EVENT_1})
		// Small delay to allow event processing
		time.Sleep(10 * time.Microsecond)
		_ = time.Since(start)
	}
}

// BenchmarkTickBasedLatency60FPS measures tick-based latency at 60 FPS
func BenchmarkTickBasedLatency60FPS(b *testing.B) {
	machine := createBenchmarkMachine()
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate:         16667 * time.Microsecond, // 60 FPS
		MaxEventsPerTick: 1000,
	})

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		b.Fatal(err)
	}
	defer rt.Stop()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		rt.SendEvent(statechartx.Event{ID: EVENT_1})
		// Wait for next tick
		time.Sleep(17 * time.Millisecond)
		_ = time.Since(start)
	}
}

// BenchmarkEventBatching benchmarks event batching efficiency
func BenchmarkEventBatching(b *testing.B) {
	machine := createBenchmarkMachine()
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate:         10 * time.Millisecond,
		MaxEventsPerTick: 10000,
	})

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		b.Fatal(err)
	}
	defer rt.Stop()

	b.ResetTimer()

	// Send events in bursts
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			rt.SendEvent(statechartx.Event{ID: EVENT_1})
		}
		time.Sleep(15 * time.Millisecond) // Wait for tick to process
	}
}

// BenchmarkDeterminism tests determinism overhead
func BenchmarkDeterminism(b *testing.B) {
	machine := createBenchmarkMachine()
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate:         1 * time.Millisecond,
		MaxEventsPerTick: 1000,
	})

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		b.Fatal(err)
	}
	defer rt.Stop()

	b.ResetTimer()

	// Send events with priority
	for i := 0; i < b.N; i++ {
		priority := i % 10
		rt.SendEventWithPriority(statechartx.Event{ID: EVENT_1}, priority)
	}

	time.Sleep(20 * time.Millisecond)
}
