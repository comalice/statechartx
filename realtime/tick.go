package realtime

import (
	"context"
	"sort"

	"github.com/comalice/statechartx"
)

// processTick processes one complete tick
func (rt *RealtimeRuntime) processTick() {
	// Phase 1: Collect events atomically
	events := rt.collectEvents()

	// Phase 2: Sort for deterministic order
	rt.sortEvents(events)

	// Phase 3: Process events using EXISTING core methods
	rt.processEvents(events)

	// Phase 4: Process macrostep to completion (eventless transitions + internal events)
	// This implements SCXML run-to-completion semantics
	rt.processMacrostepToCompletion(context.Background())

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
	// CRITICAL: Set macrostep mode so events raised during transitions go to internal queue
	rt.internalQueueMu.Lock()
	rt.inMacrostep = true
	rt.internalQueueMu.Unlock()

	defer func() {
		rt.internalQueueMu.Lock()
		rt.inMacrostep = false
		rt.internalQueueMu.Unlock()
	}()

	// CRITICAL: Call EXISTING processMicrosteps method
	// Reuses existing microstep logic (lines 784-861 of statechart.go)
	rt.Runtime.ProcessMicrosteps(context.Background())
}

// processParallelRegionsSequentially processes parallel regions in document order
func (rt *RealtimeRuntime) processParallelRegionsSequentially() {
	rt.regionMu.RLock()
	defer rt.regionMu.RUnlock()

	// Process each parallel state's regions
	for parallelStateID, regions := range rt.parallelRegionStates {
		rt.processParallelStateRegions(parallelStateID, regions)
	}
}

// processParallelStateRegions processes all regions of a parallel state in document order
func (rt *RealtimeRuntime) processParallelStateRegions(parallelStateID statechartx.StateID, regions map[statechartx.StateID]*realtimeRegion) {
	// Get sorted list of region IDs for deterministic ordering
	regionIDs := make([]statechartx.StateID, 0, len(regions))
	for regionID := range regions {
		regionIDs = append(regionIDs, regionID)
	}

	// Sort by StateID for document order
	sort.Slice(regionIDs, func(i, j int) bool {
		return regionIDs[i] < regionIDs[j]
	})

	// Process each region's event queue sequentially
	ctx := context.Background()
	for _, regionID := range regionIDs {
		region := regions[regionID]
		if region == nil {
			continue
		}

		// Process all queued events for this region
		for len(region.eventQueue) > 0 {
			event := region.eventQueue[0]
			region.eventQueue = region.eventQueue[1:]

			// Process event in this region's context
			rt.processRegionEvent(ctx, region, event)
		}
	}

	// Process eventless transitions to completion after all events processed
	rt.processMacrostepToCompletion(ctx)
}
