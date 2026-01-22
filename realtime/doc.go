// Package realtime provides a tick-based deterministic runtime for StatechartX.
//
// The real-time runtime differs from the event-driven runtime in event dispatch:
//   - Events are batched and processed at fixed tick boundaries
//   - Deterministic event ordering via sequence numbers
//   - Parallel regions processed sequentially (no goroutines)
//   - Fixed time-step execution (e.g., 60 FPS)
//
// # Example Usage
//
//	machine, _ := statechartx.NewMachine(rootState)
//	rt := realtime.NewRuntime(machine, realtime.Config{
//		TickRate: 16667 * time.Microsecond, // 60 FPS
//	})
//	rt.Start(ctx)
//	rt.SendEvent(statechartx.Event{ID: 1})
//
// # Trade-offs vs Event-Driven
//
// Lower throughput (60K vs 2M events/sec at 60 FPS)
// Higher latency (16.67ms vs 217ns at 60 FPS)
// Guaranteed determinism and reproducibility
// Fixed time budget per tick
//
// # Use Cases
//
//   - Game engines (60 FPS game logic)
//   - Physics simulations (fixed time-step)
//   - Robotics (deterministic control loops)
//   - Testing/debugging (reproducible scenarios)
//
// # Architecture
//
// The RealtimeRuntime embeds the existing statechartx.Runtime and reuses all
// core state transition logic (~430 lines of battle-tested code). Only the
// event dispatch mechanism is replaced with tick-based batching (~230 lines
// of new code).
//
// This design ensures:
//   - Zero code duplication
//   - Consistent behavior with event-driven runtime
//   - Easy maintenance (fixes in core apply to both runtimes)
//   - Predictable performance characteristics
//
// # Event Ordering Guarantees
//
// Events are ordered deterministically using:
//  1. Priority (higher priority processed first)
//  2. Sequence number (FIFO for same priority)
//  3. Stable sorting (preserves relative order)
//
// This ensures that given the same sequence of SendEvent() calls,
// the state machine will always execute the same way, regardless of
// timing or concurrency.
//
// # Performance Characteristics
//
// At 60 FPS (16.67ms tick rate):
//   - Throughput: ~60,000 events/second
//   - Latency: 0-16.67ms (depends on when event arrives in tick)
//   - Memory: O(max_events_per_tick) for event batching
//   - CPU: Fixed time budget per tick
//
// At 1000 Hz (1ms tick rate):
//   - Throughput: ~1,000,000 events/second
//   - Latency: 0-1ms
//   - Memory: Same as above
//   - CPU: Tighter time budget
//
// # Comparison with Event-Driven Runtime
//
// Event-Driven:
//   - Best for: Low latency, high throughput, reactive systems
//   - Latency: ~217ns (nanoseconds)
//   - Throughput: ~2M events/sec
//   - Determinism: Best-effort (depends on goroutine scheduling)
//   - Use cases: Web servers, microservices, UI state management
//
// Tick-Based (this package):
//   - Best for: Determinism, fixed time-step, reproducibility
//   - Latency: ~16.67ms at 60 FPS (milliseconds)
//   - Throughput: ~60K events/sec at 60 FPS
//   - Determinism: Guaranteed
//   - Use cases: Games, physics sims, robotics, testing
package realtime
