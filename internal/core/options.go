// Package core provides the runtime core tier of the statechart engine.
// Options for configuring Machine instances.
package core

import "github.com/comalice/statechartx/internal/primitives"

// WithActionRunner configures the Machine with a custom ActionRunner.
func WithActionRunner(r ActionRunner) Option {
	return func(m *Machine) {
		m.actionRunner = r
	}
}

// WithGuardEvaluator configures the Machine with a custom GuardEvaluator.
func WithGuardEvaluator(e GuardEvaluator) Option {
	return func(m *Machine) {
		m.guardEval = e
	}
}

// WithEventSource configures the Machine with a custom EventSource.
func WithEventSource(s EventSource) Option {
	return func(m *Machine) {
		m.eventSource = s
	}
}

// WithPersister configures the Machine with a custom Persister.
func WithPersister(p Persister) Option {
	return func(m *Machine) {
		m.persister = p
	}
}

// WithPublisher configures the Machine with a custom EventPublisher.
func WithPublisher(pb EventPublisher) Option {
	return func(m *Machine) {
		m.publisher = pb
	}
}

// WithVisualizer configures the Machine with a custom Visualizer.
func WithVisualizer(v Visualizer) Option {
	return func(m *Machine) {
		m.visualizer = v
	}
}

// WithQueueSize configures the event queue buffer size.
// Note: Overwrites the default channel; for stub use in Phase 2.
func WithQueueSize(size int) Option {
	return func(m *Machine) {
		m.eventQueue = make(chan primitives.Event, size)
	}
}

// WithRegistry configures the Machine with a custom Registry for versioning snapshots.
func WithRegistry(r Registry) Option {
	return func(m *Machine) {
		m.registry = r
	}
}
