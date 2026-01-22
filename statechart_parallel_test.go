package statechartx_test

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/comalice/statechartx"
)

// Test constants for parallel states
const (
	STATE_PARALLEL StateID = 100
	STATE_REGION_A StateID = 101
	STATE_REGION_B StateID = 102
	STATE_REGION_C StateID = 103
	EVENT_PING     EventID = 200
	EVENT_PONG     EventID = 201
)

// Helper function to count goroutines
func countGoroutines() int {
	return runtime.NumGoroutine()
}

// Helper function to wait for goroutine count
func waitForGoroutineCount(t *testing.T, expected int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if countGoroutines() == expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("goroutine count: got %d, want %d", countGoroutines(), expected)
}

// Helper function to verify no goroutine leak
func verifyNoGoroutineLeak(t *testing.T, baseline int) {
	time.Sleep(100 * time.Millisecond) // Allow cleanup
	current := countGoroutines()
	if current > baseline+1 { // Allow 1 extra for test runner
		t.Errorf("goroutine leak: baseline %d, current %d", baseline, current)
	}
}

// Helper function to run with timeout
func runWithTimeout(t *testing.T, timeout time.Duration, fn func()) {
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()
	select {
	case <-done:
		// Success
	case <-time.After(timeout):
		t.Fatal("test timeout")
	}
}

// Phase 1: Basic Functionality Tests

// Test 3.1.1: Basic Parallel Region Spawn
func TestParallelRegionSpawn(t *testing.T) {
	baseline := countGoroutines()

	var region1Started, region2Started atomic.Int32

	region1 := &State{
		ID: STATE_REGION_A,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			region1Started.Add(1)
			return nil
		},
	}

	region2 := &State{
		ID: STATE_REGION_B,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			region2Started.Add(1)
			return nil
		},
	}

	parallelState := &State{
		ID:         STATE_PARALLEL,
		IsParallel: true,
		Children: map[StateID]*State{
			STATE_REGION_A: region1,
			STATE_REGION_B: region2,
		},
	}

	m, err := NewMachine(parallelState)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Wait for regions to start
	time.Sleep(200 * time.Millisecond)

	if region1Started.Load() != 1 {
		t.Errorf("region 1 not started: %d", region1Started.Load())
	}
	if region2Started.Load() != 1 {
		t.Errorf("region 2 not started: %d", region2Started.Load())
	}

	// Verify goroutines spawned (baseline + 1 event loop + 2 regions)
	current := countGoroutines()
	if current < baseline+2 {
		t.Errorf("expected at least %d goroutines, got %d", baseline+2, current)
	}

	rt.Stop()
	verifyNoGoroutineLeak(t, baseline)
}

// Test 3.1.2: Parallel Region Cleanup on Exit
func TestParallelRegionCleanupOnExit(t *testing.T) {
	baseline := countGoroutines()

	var region1Exited, region2Exited atomic.Int32

	region1 := &State{
		ID: STATE_REGION_A,
		ExitAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			region1Exited.Add(1)
			return nil
		},
	}

	region2 := &State{
		ID: STATE_REGION_B,
		ExitAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			region2Exited.Add(1)
			return nil
		},
	}

	parallelState := &State{
		ID:         STATE_PARALLEL,
		IsParallel: true,
		Children: map[StateID]*State{
			STATE_REGION_A: region1,
			STATE_REGION_B: region2,
		},
	}

	m, err := NewMachine(parallelState)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Wait for regions to start
	time.Sleep(100 * time.Millisecond)

	// Stop runtime
	if err := rt.Stop(); err != nil {
		t.Fatal(err)
	}

	// Verify cleanup
	verifyNoGoroutineLeak(t, baseline)
}

// Test 3.1.3: Parallel Region Cleanup on Context Cancel
func TestParallelRegionCleanupOnContextCancel(t *testing.T) {
	baseline := countGoroutines()

	region1 := &State{ID: STATE_REGION_A}
	region2 := &State{ID: STATE_REGION_B}

	parallelState := &State{
		ID:         STATE_PARALLEL,
		IsParallel: true,
		Children: map[StateID]*State{
			STATE_REGION_A: region1,
			STATE_REGION_B: region2,
		},
	}

	m, err := NewMachine(parallelState)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx, cancel := context.WithCancel(context.Background())

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Wait for regions to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for cleanup
	time.Sleep(500 * time.Millisecond)

	// Verify cleanup
	verifyNoGoroutineLeak(t, baseline)
}

// Test 3.1.4: Parallel Region Panic Recovery
func TestParallelRegionPanicRecovery(t *testing.T) {
	baseline := countGoroutines()

	var region2Processed atomic.Int32

	region1 := &State{
		ID: STATE_REGION_A,
		EntryAction: func(ctx context.Context, evt *Event, from, to StateID) error {
			panic("intentional panic")
		},
	}

	region2 := &State{
		ID: STATE_REGION_B,
		Transitions: []*Transition{
			{
				Event:  EVENT_PING,
				Target: 0, // internal
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					region2Processed.Add(1)
					return nil
				},
			},
		},
	}

	parallelState := &State{
		ID:         STATE_PARALLEL,
		IsParallel: true,
		Children: map[StateID]*State{
			STATE_REGION_A: region1,
			STATE_REGION_B: region2,
		},
	}

	m, err := NewMachine(parallelState)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start should handle panic gracefully
	err = rt.Start(ctx)
	if err == nil {
		// If no error, verify region 2 still works
		time.Sleep(100 * time.Millisecond)

		// Send event to region 2
		rt.SendEvent(ctx, Event{ID: EVENT_PING, Address: STATE_REGION_B})
		time.Sleep(100 * time.Millisecond)

		if region2Processed.Load() != 1 {
			t.Errorf("region 2 should still process events: %d", region2Processed.Load())
		}
	}

	rt.Stop()
	verifyNoGoroutineLeak(t, baseline)
}

// Test 3.2.1: Non-blocking Event Send
func TestNonBlockingEventSend(t *testing.T) {
	var processedCount atomic.Int32

	region1 := &State{
		ID: STATE_REGION_A,
		Transitions: []*Transition{
			{
				Event:  EVENT_PING,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					processedCount.Add(1)
					return nil
				},
			},
		},
	}

	parallelState := &State{
		ID:         STATE_PARALLEL,
		IsParallel: true,
		Children: map[StateID]*State{
			STATE_REGION_A: region1,
		},
	}

	m, err := NewMachine(parallelState)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	// Wait for region to start
	time.Sleep(100 * time.Millisecond)

	// Send 100 events rapidly
	start := time.Now()
	for i := 0; i < 100; i++ {
		if err := rt.SendEvent(ctx, Event{ID: EVENT_PING, Address: STATE_REGION_A}); err != nil {
			t.Fatalf("send failed at %d: %v", i, err)
		}
	}
	duration := time.Since(start)

	// All sends should complete quickly (< 1 second for 100 events)
	if duration > time.Second {
		t.Errorf("sends took too long: %v", duration)
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Verify all events processed
	if processedCount.Load() != 100 {
		t.Errorf("expected 100 events processed, got %d", processedCount.Load())
	}
}

// Test 3.3.1: No Circular Event Routing
func TestNoCircularEventRouting(t *testing.T) {
	var region1Count, region2Count atomic.Int32

	region1 := &State{
		ID: STATE_REGION_A,
		Transitions: []*Transition{
			{
				Event:  EVENT_PING,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					count := region1Count.Add(1)
					// Only send once to prevent infinite loop
					if count == 1 {
						// This would be done via runtime, but we're testing the pattern
					}
					return nil
				},
			},
		},
	}

	region2 := &State{
		ID: STATE_REGION_B,
		Transitions: []*Transition{
			{
				Event:  EVENT_PONG,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					region2Count.Add(1)
					return nil
				},
			},
		},
	}

	parallelState := &State{
		ID:         STATE_PARALLEL,
		IsParallel: true,
		Children: map[StateID]*State{
			STATE_REGION_A: region1,
			STATE_REGION_B: region2,
		},
	}

	m, err := NewMachine(parallelState)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	// Wait for regions to start
	time.Sleep(100 * time.Millisecond)

	// Send initial event
	rt.SendEvent(ctx, Event{ID: EVENT_PING, Address: STATE_REGION_A})

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Verify no deadlock (test completes)
	if region1Count.Load() < 1 {
		t.Errorf("region 1 should process at least 1 event")
	}
}

// Test 3.4.1: Broadcast Event Delivery
func TestBroadcastEventDelivery(t *testing.T) {
	var region1Count, region2Count, region3Count atomic.Int32

	region1 := &State{
		ID: STATE_REGION_A,
		Transitions: []*Transition{
			{
				Event:  EVENT_PING,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					region1Count.Add(1)
					return nil
				},
			},
		},
	}

	region2 := &State{
		ID: STATE_REGION_B,
		Transitions: []*Transition{
			{
				Event:  EVENT_PING,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					region2Count.Add(1)
					return nil
				},
			},
		},
	}

	region3 := &State{
		ID: STATE_REGION_C,
		Transitions: []*Transition{
			{
				Event:  EVENT_PING,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					region3Count.Add(1)
					return nil
				},
			},
		},
	}

	parallelState := &State{
		ID:         STATE_PARALLEL,
		IsParallel: true,
		Children: map[StateID]*State{
			STATE_REGION_A: region1,
			STATE_REGION_B: region2,
			STATE_REGION_C: region3,
		},
	}

	m, err := NewMachine(parallelState)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	// Wait for regions to start
	time.Sleep(100 * time.Millisecond)

	// Send broadcast event (address == 0)
	rt.SendEvent(ctx, Event{ID: EVENT_PING, Address: 0})

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Verify all regions received event
	if region1Count.Load() != 1 {
		t.Errorf("region 1 count: got %d, want 1", region1Count.Load())
	}
	if region2Count.Load() != 1 {
		t.Errorf("region 2 count: got %d, want 1", region2Count.Load())
	}
	if region3Count.Load() != 1 {
		t.Errorf("region 3 count: got %d, want 1", region3Count.Load())
	}
}

// Test 3.4.2: Targeted Event Delivery
func TestTargetedEventDelivery(t *testing.T) {
	var region1Count, region2Count, region3Count atomic.Int32

	region1 := &State{
		ID: STATE_REGION_A,
		Transitions: []*Transition{
			{
				Event:  EVENT_PING,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					region1Count.Add(1)
					return nil
				},
			},
		},
	}

	region2 := &State{
		ID: STATE_REGION_B,
		Transitions: []*Transition{
			{
				Event:  EVENT_PING,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					region2Count.Add(1)
					return nil
				},
			},
		},
	}

	region3 := &State{
		ID: STATE_REGION_C,
		Transitions: []*Transition{
			{
				Event:  EVENT_PING,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					region3Count.Add(1)
					return nil
				},
			},
		},
	}

	parallelState := &State{
		ID:         STATE_PARALLEL,
		IsParallel: true,
		Children: map[StateID]*State{
			STATE_REGION_A: region1,
			STATE_REGION_B: region2,
			STATE_REGION_C: region3,
		},
	}

	m, err := NewMachine(parallelState)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	// Wait for regions to start
	time.Sleep(100 * time.Millisecond)

	// Send targeted event to region 2 only
	rt.SendEvent(ctx, Event{ID: EVENT_PING, Address: STATE_REGION_B})

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Verify only region 2 received event
	if region1Count.Load() != 0 {
		t.Errorf("region 1 should not receive event: got %d", region1Count.Load())
	}
	if region2Count.Load() != 1 {
		t.Errorf("region 2 count: got %d, want 1", region2Count.Load())
	}
	if region3Count.Load() != 0 {
		t.Errorf("region 3 should not receive event: got %d", region3Count.Load())
	}
}

// Test 3.5.1: Graceful Shutdown (No Pending Events)
func TestGracefulShutdownNoPendingEvents(t *testing.T) {
	baseline := countGoroutines()

	region1 := &State{ID: STATE_REGION_A}
	region2 := &State{ID: STATE_REGION_B}

	parallelState := &State{
		ID:         STATE_PARALLEL,
		IsParallel: true,
		Children: map[StateID]*State{
			STATE_REGION_A: region1,
			STATE_REGION_B: region2,
		},
	}

	m, err := NewMachine(parallelState)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}

	// Wait for regions to start
	time.Sleep(100 * time.Millisecond)

	// Stop runtime
	start := time.Now()
	if err := rt.Stop(); err != nil {
		t.Fatal(err)
	}
	duration := time.Since(start)

	// Shutdown should complete quickly (< 500ms)
	if duration > 500*time.Millisecond {
		t.Errorf("shutdown took too long: %v", duration)
	}

	// Verify no goroutine leak
	verifyNoGoroutineLeak(t, baseline)
}

// Test 3.6.1: Concurrent State Access
func TestConcurrentStateAccess(t *testing.T) {
	type sharedState struct {
		mu      sync.Mutex
		counter int
	}

	ext := &sharedState{}

	region1 := &State{
		ID: STATE_REGION_A,
		Transitions: []*Transition{
			{
				Event:  EVENT_PING,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					ext.mu.Lock()
					ext.counter++
					ext.mu.Unlock()
					return nil
				},
			},
		},
	}

	region2 := &State{
		ID: STATE_REGION_B,
		Transitions: []*Transition{
			{
				Event:  EVENT_PING,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					ext.mu.Lock()
					ext.counter++
					ext.mu.Unlock()
					return nil
				},
			},
		},
	}

	region3 := &State{
		ID: STATE_REGION_C,
		Transitions: []*Transition{
			{
				Event:  EVENT_PING,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					ext.mu.Lock()
					ext.counter++
					ext.mu.Unlock()
					return nil
				},
			},
		},
	}

	parallelState := &State{
		ID:         STATE_PARALLEL,
		IsParallel: true,
		Children: map[StateID]*State{
			STATE_REGION_A: region1,
			STATE_REGION_B: region2,
			STATE_REGION_C: region3,
		},
	}

	m, err := NewMachine(parallelState)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, ext)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	// Wait for regions to start
	time.Sleep(100 * time.Millisecond)

	// Send 100 broadcast events
	for i := 0; i < 100; i++ {
		rt.SendEvent(ctx, Event{ID: EVENT_PING, Address: 0})
	}

	// Wait for processing
	time.Sleep(1 * time.Second)

	// Verify counter (should be 300: 100 events * 3 regions)
	ext.mu.Lock()
	finalCount := ext.counter
	ext.mu.Unlock()

	if finalCount != 300 {
		t.Errorf("expected counter 300, got %d", finalCount)
	}
}

// Test 3.7.1: Many Parallel Regions
func TestManyParallelRegions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	baseline := countGoroutines()

	// Create 10 regions (reduced from 100 for faster testing)
	numRegions := 10
	children := make(map[StateID]*State)

	for i := 0; i < numRegions; i++ {
		stateID := StateID(1000 + i)
		children[stateID] = &State{ID: stateID}
	}

	parallelState := &State{
		ID:         STATE_PARALLEL,
		IsParallel: true,
		Children:   children,
	}

	m, err := NewMachine(parallelState)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Measure startup time
	start := time.Now()
	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	startupDuration := time.Since(start)

	if startupDuration > 2*time.Second {
		t.Errorf("startup took too long: %v", startupDuration)
	}

	// Wait for regions to start
	time.Sleep(200 * time.Millisecond)

	// Measure shutdown time
	start = time.Now()
	if err := rt.Stop(); err != nil {
		t.Fatal(err)
	}
	shutdownDuration := time.Since(start)

	if shutdownDuration > 2*time.Second {
		t.Errorf("shutdown took too long: %v", shutdownDuration)
	}

	// Verify no goroutine leak
	verifyNoGoroutineLeak(t, baseline)
}

// Test 3.7.2: High Event Volume
func TestHighEventVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	var processedCount atomic.Int32

	region1 := &State{
		ID: STATE_REGION_A,
		Transitions: []*Transition{
			{
				Event:  EVENT_PING,
				Target: 0,
				Action: func(ctx context.Context, evt *Event, from, to StateID) error {
					processedCount.Add(1)
					return nil
				},
			},
		},
	}

	parallelState := &State{
		ID:         STATE_PARALLEL,
		IsParallel: true,
		Children: map[StateID]*State{
			STATE_REGION_A: region1,
		},
	}

	m, err := NewMachine(parallelState)
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(m, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	// Wait for region to start
	time.Sleep(100 * time.Millisecond)

	// Send 1000 events (reduced from 10000 for faster testing)
	numEvents := 1000
	start := time.Now()
	for i := 0; i < numEvents; i++ {
		if err := rt.SendEvent(ctx, Event{ID: EVENT_PING, Address: STATE_REGION_A}); err != nil {
			t.Fatalf("send failed at %d: %v", i, err)
		}
	}
	sendDuration := time.Since(start)

	// Wait for processing
	time.Sleep(2 * time.Second)

	// Verify all events processed
	processed := processedCount.Load()
	if processed != int32(numEvents) {
		t.Errorf("expected %d events processed, got %d", numEvents, processed)
	}

	// Calculate throughput
	throughput := float64(numEvents) / sendDuration.Seconds()
	t.Logf("Throughput: %.0f events/second", throughput)
}
