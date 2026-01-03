package realtime

import (
	"context"
)

// processTick processes one complete tick
func (rt *RealtimeRuntime) processTick() {
	// Phase 1: Collect events atomically
	events := rt.collectEvents()

	// Phase 2: Sort for deterministic order
	rt.sortEvents(events)

	// Phase 3: Process events using EXISTING core methods
	rt.processEvents(events)

	// Phase 4: Process microsteps using EXISTING core method
	rt.processMicrostepsIfNeeded()

	// Phase 5: Process parallel regions sequentially (if any)
	rt.processParallelRegionsSequentially()
}

// collectEvents atomically retrieves and clears the event batch
func (rt *RealtimeRuntime) collectEvents() []EventWithMeta {
	rt.batchMu.Lock()
	defer rt.batchMu.Unlock()

	events := rt.eventBatch
	rt.eventBatch = make([]EventWithMeta, 0, cap(rt.eventBatch))

	return events
}

// processEvents processes all events for this tick
func (rt *RealtimeRuntime) processEvents(events []EventWithMeta) {
	for _, eventMeta := range events {
		// CRITICAL: Call EXISTING processEvent method from embedded Runtime
		// This is where we reuse ~430 lines of battle-tested code!
		rt.Runtime.ProcessEvent(eventMeta.Event)
	}
}

// processMicrostepsIfNeeded processes eventless transitions
func (rt *RealtimeRuntime) processMicrostepsIfNeeded() {
	// CRITICAL: Call EXISTING processMicrosteps method
	// Reuses existing microstep logic (lines 784-861 of statechart.go)
	rt.Runtime.ProcessMicrosteps(context.Background())
}

// processParallelRegionsSequentially processes parallel regions in order
func (rt *RealtimeRuntime) processParallelRegionsSequentially() {
	// TODO: Implement in Phase 3 when parallel state support is added
	// Will reuse existing transition methods but process sequentially
}
