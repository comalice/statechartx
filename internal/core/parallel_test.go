package core

import "testing"

func TestParallelStub(t *testing.T) {
	// Stub test for parallel sync - verifies no panic on stub
	if err := syncParallelRegions(nil); err != nil {
		t.Error("stub should return nil")
	}
}