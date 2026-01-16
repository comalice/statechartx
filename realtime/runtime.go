package realtime

import (
        "context"
        "errors"
        "sync"
        "time"

        "github.com/comalice/statechartx"
)

// RealtimeRuntime provides tick-based deterministic execution by embedding
// the existing event-driven Runtime and adapting only the event dispatch.
type RealtimeRuntime struct {
        // Embed existing runtime to reuse ALL core methods:
        // - processEvent()
        // - processMicrosteps()
        // - computeLCA()
        // - exitToLCA() / enterFromLCA()
        // - pickTransitionHierarchical()
        // - History state methods
        // - Done event methods
        *statechartx.Runtime

        // Tick-specific fields
        tickRate    time.Duration      // e.g., 16.67ms for 60 FPS
        ticker      *time.Ticker
        tickNum     uint64

        // Event batching (replaces async channel)
        eventBatch  []EventWithMeta
        batchMu     sync.Mutex
        sequenceNum uint64

        // Parallel state support (sequential processing for determinism)
        // Map of parallel state ID -> region states
        parallelRegionStates map[statechartx.StateID]map[statechartx.StateID]*realtimeRegion
        regionMu             sync.RWMutex

        // Internal event queue for SCXML run-to-completion semantics
        // Events raised during macrostep processing (e.g., in entry/exit/transition actions)
        // are queued here and processed within the same macrostep
        internalEventQueue []statechartx.Event
        internalQueueMu    sync.Mutex
        inMacrostep        bool // Flag indicating we're in macrostep processing

        // Control
        tickCtx     context.Context
        tickCancel  context.CancelFunc
        stopped     chan struct{}
}

// realtimeRegion represents a single region in a parallel state (sequential processing)
type realtimeRegion struct {
        regionID     statechartx.StateID   // ID of the region (child of parallel state)
        currentState statechartx.StateID   // Current state within this region
        eventQueue   []statechartx.Event   // Events queued for this region
}

// Config configures the real-time runtime
type Config struct {
        TickRate         time.Duration // Fixed tick rate (e.g., 16.67ms for 60 FPS)
        MaxEventsPerTick int           // Event queue capacity (default: 1000)
}

// NewRuntime creates a new tick-based runtime by embedding the event-driven runtime
func NewRuntime(machine *statechartx.Machine, cfg Config) *RealtimeRuntime {
        if cfg.MaxEventsPerTick == 0 {
                cfg.MaxEventsPerTick = 1000
        }
        if cfg.TickRate == 0 {
                cfg.TickRate = 16667 * time.Microsecond // Default 60 FPS
        }

        rt := &RealtimeRuntime{
                // Embed existing runtime (THIS IS THE KEY - REUSE EVERYTHING)
                Runtime:              statechartx.NewRuntime(machine, nil),
                tickRate:             cfg.TickRate,
                eventBatch:           make([]EventWithMeta, 0, cfg.MaxEventsPerTick),
                stopped:              make(chan struct{}),
                parallelRegionStates: make(map[statechartx.StateID]map[statechartx.StateID]*realtimeRegion),
        }

        // Register parallel state hooks for sequential processing
        rt.Runtime.ParallelHooks = rt.createParallelHooks()

        return rt
}

// Start is implemented in parallel.go to handle sequential parallel state processing

// Stop gracefully stops the runtime
func (rt *RealtimeRuntime) Stop() error {
        if rt.tickCancel != nil {
                rt.tickCancel()
        }
        if rt.ticker != nil {
                rt.ticker.Stop()
        }

        // Wait for tick loop to exit
        <-rt.stopped

        // Stop embedded runtime
        return rt.Runtime.Stop()
}

// tickLoop is the main tick execution loop
func (rt *RealtimeRuntime) tickLoop() {
        defer close(rt.stopped)
        defer func() {
                if r := recover(); r != nil {
                        // Log panic but don't crash the runtime
                        // In production, this should be logged properly
                        _ = r // TODO: Add proper logging
                }
        }()

        for {
                select {
                case <-rt.tickCtx.Done():
                        return
                case <-rt.ticker.C:
                        // Process tick with panic recovery
                        func() {
                                defer func() {
                                        if r := recover(); r != nil {
                                                // Recover from panic in tick processing
                                                // In production, this should be logged
                                                _ = r // TODO: Add proper logging
                                        }
                                }()
                                rt.processTick()
                        }()

                        rt.batchMu.Lock()
                        rt.tickNum++
                        rt.batchMu.Unlock()
                }
        }
}

// SendEvent is implemented in parallel.go to handle both normal and parallel state routing

// SendEventWithPriority queues an event with priority
func (rt *RealtimeRuntime) SendEventWithPriority(event statechartx.Event, priority int) error {
        rt.batchMu.Lock()
        defer rt.batchMu.Unlock()

        if len(rt.eventBatch) >= cap(rt.eventBatch) {
                return errors.New("event queue full")
        }

        rt.eventBatch = append(rt.eventBatch, EventWithMeta{
                Event:       event,
                SequenceNum: rt.sequenceNum,
                Priority:    priority,
        })
        rt.sequenceNum++

        return nil
}

// GetTickNumber returns the current tick count
func (rt *RealtimeRuntime) GetTickNumber() uint64 {
        rt.batchMu.Lock()
        defer rt.batchMu.Unlock()
        return rt.tickNum
}

// GetCurrentState returns the current state ID
func (rt *RealtimeRuntime) GetCurrentState() statechartx.StateID {
        return rt.Runtime.GetCurrentState()
}
