package realtime

import (
	"sort"

	"github.com/comalice/statechartx"
)

// EventWithMeta adds sequencing metadata for deterministic ordering
type EventWithMeta struct {
	Event       statechartx.Event
	SequenceNum uint64
	Priority    int // For future priority ordering
}

// sortEvents orders events deterministically
// Stable sort preserves insertion order for equal priorities
func (rt *RealtimeRuntime) sortEvents(events []EventWithMeta) {
	// Stable sort preserves insertion order for equal priorities
	sort.SliceStable(events, func(i, j int) bool {
		// Primary: Higher priority first
		if events[i].Priority != events[j].Priority {
			return events[i].Priority > events[j].Priority
		}

		// Secondary: Earlier sequence number first (FIFO)
		return events[i].SequenceNum < events[j].SequenceNum
	})
}

// Event ordering guarantees:
// 1. Events from same source processed in submission order (sequence number)
// 2. Higher priority events processed first
// 3. Deterministic tie-breaking via sequence number
// 4. Stable sort preserves relative order
