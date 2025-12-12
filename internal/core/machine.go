// Package core provides the runtime core tier of the statechart engine.
// This includes the Machine runtime, event loop, state transitions, and history management.
// Dependencies: internal/primitives (Phase 1)
// Stdlib-only implementation.
// Pluggable components defined here as forward declarations for Phase 3 wiring.
//go:generate go test ./... -race

package core

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"time"

	"github.com/comalice/statechartx/internal/primitives"
)

// Pluggable component interfaces.
// Full implementations and options in Phase 3 (extensibility).

type ActionRunner interface {
	Run(ctx *primitives.Context, action primitives.ActionRef, event primitives.Event) error
}

type GuardEvaluator interface {
	Eval(ctx *primitives.Context, guard primitives.GuardRef, event primitives.Event) bool
}

type EventSource interface {
	Events() <-chan primitives.Event
}

type Persister interface {
	Save(ctx context.Context, snapshot MachineSnapshot) error
	Load(ctx context.Context, machineID string) (MachineSnapshot, error)
}

// MachineSnapshot is the serializable snapshot of machine runtime state.
type MachineSnapshot struct {
	MachineID    string                   `json:"machineID" yaml:"machineID"`
	Config       primitives.MachineConfig `json:"config" yaml:"config"`
	Current      []string                 `json:"current" yaml:"current"`
	ContextData  map[string]any           `json:"context" yaml:"context"`
	QueuedEvents []primitives.Event       `json:"queuedEvents,omitempty" yaml:"queuedEvents,omitempty"`
	Timestamp    time.Time                `json:"timestamp" yaml:"timestamp"`
}

type MachineMetadata struct {
	MachineID  string    `json:"machineID" yaml:"machineID"`
	Transition string    `json:"transition" yaml:"transition"`
	Timestamp  time.Time `json:"timestamp" yaml:"timestamp"`
}

type EventPublisher interface {
	Publish(ctx context.Context, event primitives.Event, metadata MachineMetadata) error
	Close() error
}

type Visualizer interface {
	ExportDOT(config primitives.MachineConfig, current []string) string
	ExportJSON(config primitives.MachineConfig) ([]byte, error)
}

// Option applies configuration to Machine via functional options pattern.
type Option func(*Machine)

// Machine is the core runtime instance of a statechart.
// Thread-safe for concurrent Send() from multiple goroutines.
// Event-driven actor model with buffered queue and graceful shutdown.
// Pluggable architecture for extensibility (Phase 3+).
// Stdlib-only core.
type Machine struct {
	config        primitives.MachineConfig
	current       []string // active leaf state paths
	ctx           *primitives.Context
	mu            sync.RWMutex
	eventQueue    chan primitives.Event
	done          chan struct{}
	stateCache    map[string]*primitives.StateConfig
	ancestorCache map[string][]string
	// Pluggable components (nil = defaults/stubs)
	actionRunner ActionRunner
	guardEval    GuardEvaluator
	eventSource  EventSource
	persister    Persister
	publisher    EventPublisher
	visualizer   Visualizer
	registry     Registry
}

 // Config returns the machine's configuration (thread-safe shallow copy).
func (m *Machine) Config() primitives.MachineConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// NewMachine creates and initializes a new Machine instance.
func NewMachine(config primitives.MachineConfig, opts ...Option) *Machine {
	m := &Machine{
		config:     config,
		ctx:        primitives.NewContext(),
		eventQueue: make(chan primitives.Event, 1000), // default buffered queue
		done:       make(chan struct{}),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Start initializes the machine, validates config, activates initial state,
// and launches the event processing goroutine.
// Idempotent: safe to call multiple times (no-op after first).
func (m *Machine) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate config
	if err := m.config.Validate(); err != nil {
		return err
	}

	// Idempotent check
	select {
	case <-m.done:
		return nil // already stopped
	default:
	}

	// Precompute caches for performance
	m.stateCache = make(map[string]*primitives.StateConfig)
	m.ancestorCache = make(map[string][]string)
	for _, state := range m.config.States {
		precomputePaths(state, "", m.stateCache, m.ancestorCache)
	}

	// Activate initial state
	if _, err := m.config.FindState(m.config.Initial); err != nil {
		return fmt.Errorf("invalid initial state %q: %w", m.config.Initial, err)
	}
	m.current = []string{resolveInitialLeaf(&m.config, m.config.Initial)}

	go m.interpret()

	// Wire EventSource if provided (Phase 3)
	if m.eventSource != nil {
		go func() {
			for event := range m.eventSource.Events() {
				if err := m.Send(event); err != nil {
					// Log backpressure? Silent drop for now
				}
			}
		}()
	}
	return nil
}

// interpret is the private event loop goroutine.
// Processes events from queue until shutdown signal.
func (m *Machine) interpret() {
	for {
		select {
		case event := <-m.eventQueue:
			m.processEvent(event)
		case <-m.done:
			// Graceful drain optional
			return
		}
	}
}

// processEvent handles a single event (stub for Phase 2.3 full interpreter).
func (m *Machine) processEvent(event primitives.Event) {
	// Phase 1: Read-only candidate search under RLock
	m.mu.RLock()
	candidates := []candidateTransition{}
	for _, leafPath := range m.current {
		ancestors, ok := m.ancestorCache[leafPath]
		if !ok {
			m.mu.RUnlock()
			return
		}
		for _, ancestorPath := range ancestors {
			state, ok := m.stateCache[ancestorPath]
			if !ok {
				continue
			}
			transList, ok := state.On[event.Type]
			if !ok {
				continue
			}
			for _, trans := range transList {
				guardOk := defaultGuardEval(m.ctx, trans.Guard, event)
				if m.guardEval != nil {
					guardOk = m.guardEval.Eval(m.ctx, trans.Guard, event)
				}
				if guardOk {
					candidates = append(candidates, candidateTransition{
						sourcePath: ancestorPath,
						trans:      trans,
						priority:   trans.Priority,
					})
				}
			}
		}
	}
	m.mu.RUnlock()

	if len(candidates) == 0 {
		return
	}

	// Phase 2: Select highest priority (lock-free)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].priority > candidates[j].priority
	})
	sourcePath := candidates[0].sourcePath
	trans := candidates[0].trans
	targetPath := trans.Target

	// Phase 3: Compute paths (lock-free, infrequent)
	lcca := computeLCCA(sourcePath, targetPath)
	exitStates := getExitStates(sourcePath, lcca)
	entryStates := getEntryStates(lcca, targetPath)
	targetLeaf := resolveInitialLeaf(&m.config, targetPath)

	// Phase 4: Exclusive update and actions under Lock
	m.mu.Lock()
	defer m.mu.Unlock()

	// Exit actions (innermost first)
	for i := len(exitStates) - 1; i >= 0; i-- {
		statePath := exitStates[i]
		state, ok := m.stateCache[statePath]
		if ok {
			for _, action := range state.Exit {
				if m.actionRunner != nil {
					m.actionRunner.Run(m.ctx, action, event)
				} else {
					defaultActionRun(m.ctx, action, event)
				}
			}
		}
	}

	// Transition actions
	for _, action := range trans.Actions {
		if m.actionRunner != nil {
			m.actionRunner.Run(m.ctx, action, event)
		} else {
			defaultActionRun(m.ctx, action, event)
		}
	}

	// Entry actions (outer first)
	for _, statePath := range entryStates {
		state, ok := m.stateCache[statePath]
		if ok {
			for _, action := range state.Entry {
				if m.actionRunner != nil {
					m.actionRunner.Run(m.ctx, action, event)
				} else {
					defaultActionRun(m.ctx, action, event)
				}
			}
		}
	}

	// Update current
	m.current = []string{targetLeaf}

	// Snapshot for persistence
	snapshot := MachineSnapshot{
		MachineID:    m.config.ID,
		Config:       m.config,
		Current:      append([]string(nil), m.current...),
		ContextData:  m.ctx.Snapshot(),
		QueuedEvents: nil,
		Timestamp:    time.Now(),
	}

	// Persist and publish after unlock (fire-and-forget for perf)
	go func() {
		if m.persister != nil {
			if err := m.persister.Save(context.Background(), snapshot); err != nil {
				// TODO log
			}
		}
		if m.publisher != nil {
			md := MachineMetadata{
				MachineID:  m.config.ID,
				Transition: fmt.Sprintf("%s -> %s", sourcePath, targetLeaf),
				Timestamp:  time.Now(),
			}
			if err := m.publisher.Publish(context.Background(), event, md); err != nil {
				// TODO log
			}
		}
		if m.registry != nil {
			if err := m.registry.Register(context.Background(), m.config.ID, snapshot); err != nil {
				// TODO log
			}
		}
	}()
}

// Send enqueues an event for asynchronous processing.
// Blocks briefly; returns error if queue backpressure (full).
// Thread-safe.
func (m *Machine) Send(event primitives.Event) error {
	select {
	case m.eventQueue <- event:
		return nil
	default:
		return errors.New("event queue full (backpressure)")
	}
}

// Current returns a copy of active state paths.
// Thread-safe snapshot.
func (m *Machine) Current() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	current := make([]string, len(m.current))
	copy(current, m.current)
	return current
}

// Ctx returns the machine's context (thread-safe read).
func (m *Machine) Ctx() *primitives.Context {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ctx
}

// Stop signals graceful shutdown.
// Closes done channel, goroutine exits after current event.
// Safe to call multiple times.
// Drains queue implicitly via select{}.
func (m *Machine) Stop() error {
	select {
	case <-m.done:
		return nil // already stopping/stopped
	default:
	}
	close(m.done)
	return nil
}

// Restore restores machine runtime state from a snapshot.
// Call before Start() or on stopped machine.
// Re-queues events if provided (may block if queue full).
func (m *Machine) Restore(snapshot MachineSnapshot) error {
	if m.config.ID != snapshot.MachineID {
		return fmt.Errorf("machine ID mismatch: have %q, snapshot %q", m.config.ID, snapshot.MachineID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = snapshot.Config
	m.current = append([]string(nil), snapshot.Current...)
	m.ctx.Restore(snapshot.ContextData)

	for _, event := range snapshot.QueuedEvents {
		if err := m.Send(event); err != nil {
			return fmt.Errorf("failed to restore queued event %q: %w", event.Type, err)
		}
	}

	return nil
}

// Visualize returns the Graphviz DOT visualization string of the current machine state.
func (m *Machine) Visualize() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.visualizer == nil {
		return "ERROR: No visualizer configured. Use WithVisualizer(&production.DefaultVisualizer{})"
	}
	return m.visualizer.ExportDOT(m.config, m.current)
}
