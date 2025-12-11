package core

import (
	"fmt"
	"testing"
)

func TestHistoryManager_ShalllowDeep(t *testing.T) {
	h := NewHistoryManager()

	// Shallow
	h.RecordExit("shallow1", "childA", false)
	states, found := h.Restore("shallow1", false)
	if !found || len(states) != 1 || states[0] != "childA" {
		t.Errorf("shallow restore = %v, found=%v want childA true", states, found)
	}

	// Deep
	h.RecordExit("deep1", "leaf1", true)
	states, found = h.Restore("deep1", true)
	if !found || len(states) != 1 || states[0] != "leaf1" {
		t.Errorf("deep restore = %v, found=%v want leaf1 true", states, found)
	}

	// Clear
	h.Clear("shallow1")
	_, found = h.Restore("shallow1", false)
	if found {
		t.Error("shallow found after clear")
	}
}

func TestHistoryManager_Concurrent(t *testing.T) {
	h := NewHistoryManager()

	const N = 100
	done := make(chan bool)
	for i := 0; i < N; i++ {
		go func(i int) {
			id := fmt.Sprintf("test%d", i)
			h.RecordExit(id, "child", false)
			h.Restore(id, false)
			done <- true
		}(i)
	}
	for i := 0; i < N; i++ {
		<-done
	}
}