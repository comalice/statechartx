// Package core provides the runtime core tier of the statechart engine.
// HistoryManager handles shallow and deep history state restoration per SCXML semantics.
// Stdlib-only implementation.
// Thread-safe for concurrent access.
package core

import (
	"sync"
)

// HistoryManager tracks history configurations for shallow and deep history states.
// Shallow: remembers the most recent direct child state.
// Deep: remembers the full active leaf configuration under the history region.
// Used during state transitions to restore previous configurations.
type HistoryManager struct {
	mu              sync.RWMutex
	shallowHistory  map[string]string       // historyStateID -> last direct child ID
	deepHistory     map[string][]string     // historyStateID -> active leaf paths under region
}

// NewHistoryManager creates a new HistoryManager.
func NewHistoryManager() *HistoryManager {
	return &HistoryManager{
		shallowHistory: make(map[string]string),
		deepHistory:    make(map[string][]string),
	}
}

// RecordExit records the active configuration when exiting a history region.
// For shallow history: records the direct active child ID.
// For deep history: records the full list of active leaf paths under the region.
// Called by processEvent during exit sequence for compound/parallel states containing history children.
func (h *HistoryManager) RecordExit(historyStateID string, activeChild string, isDeep bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if isDeep {
		// Note: In full integration, activeChild would be ignored; pass []string of leaves matching prefix.
		// Simplified: treat activeChild as single leaf for stub.
		h.deepHistory[historyStateID] = []string{activeChild}
	} else {
		h.shallowHistory[historyStateID] = activeChild
	}
}

// Restore returns the recorded configuration for a history state, if available.
// Returns active state paths to enter, and whether history was found.
// For shallow: returns resolved child path (single string slice).
// For deep: returns full recorded leaf paths.
// Caller resolves initials recursively using resolveInitialLeaf if needed.
func (h *HistoryManager) Restore(historyStateID string, isDeep bool) ([]string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if isDeep {
		if states, ok := h.deepHistory[historyStateID]; ok && len(states) > 0 {
			return states, true
		}
	} else {
		if child, ok := h.shallowHistory[historyStateID]; ok && child != "" {
			return []string{child}, true
		}
	}
	return nil, false
}

// Clear removes recorded history for the given history state ID.
func (h *HistoryManager) Clear(historyStateID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.shallowHistory, historyStateID)
	delete(h.deepHistory, historyStateID)
}
