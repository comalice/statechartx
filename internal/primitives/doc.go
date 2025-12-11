// Package primitives provides the foundational, zero-dependency data structures
// for the statechart engine.
//
// This package and all `internal/*` packages use ONLY the Go standard library.
// No external dependencies are permitted in the core engine to achieve:
// - Minimal binary size
// - Zero vendor bloat
// - Deterministic builds
// - Sub-microsecond performance
//
// Core invariants:
// - Immutability where possible (Event)
// - Thread-safe context (RWMutex)
// - Zero-allocation hot paths
//
// See ../../docs/ARCHITECTURE.md for complete design rationale.
package primitives
