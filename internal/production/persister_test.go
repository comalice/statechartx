// Package production provides production integrations: persistence, event publishing, visualization.
// Tests for JSONPersister round-trip and integration with Machine.
package production

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/comalice/statechartx/internal/core"
	"github.com/comalice/statechartx/internal/primitives"
)

func TestJSONPersister_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	p, err := NewJSONPersister(dir)
	if err != nil {
		t.Fatalf("NewJSONPersister failed: %v", err)
	}

	config := primitives.MachineConfig{
		ID:      "test-machine",
		Initial: "s1",
		States: map[string]*primitives.StateConfig{
			"s1": {ID: "s1", Type: primitives.Atomic},
		},
	}
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}

	ctx := primitives.NewContext()
	ctx.Set("key", "value")
	ctx.Set("counter", 42)

	snapshot := core.MachineSnapshot{
		MachineID:   "test-machine",
		Config:      config,
		Current:     []string{"s1"},
		ContextData: ctx.Snapshot(),
		Timestamp:   time.Now(),
	}

	if err := p.Save(context.Background(), snapshot); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := p.Load(context.Background(), "test-machine")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Compare ignoring Timestamp (set during save)
	// Compare JSON instead of pointers
	snapJSON, _ := json.Marshal(snapshot)
	loadedJSON, _ := json.Marshal(loaded)
	if !bytes.Equal(snapJSON, loadedJSON) {
		t.Errorf("Snapshot JSON mismatch")
	}
}

func TestJSONPersister_LoadNonExistent(t *testing.T) {
	dir := t.TempDir()
	p, err := NewJSONPersister(dir)
	if err != nil {
		t.Fatalf("NewJSONPersister failed: %v", err)
	}

	_, err = p.Load(context.Background(), "nonexistent")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected os.ErrNotExist wrapped error, got %v", err)
	}
}

func TestJSONPersister_Integration_RestoreMachine(t *testing.T) {
	dir := t.TempDir()
	p, err := NewJSONPersister(dir)
	if err != nil {
		t.Fatal(err)
	}

	config := primitives.MachineConfig{
		ID:      "restore-test",
		Initial: "green",
		States: map[string]*primitives.StateConfig{
			"green": {
				ID:   "green",
				Type: primitives.Atomic,
				On: map[string][]primitives.TransitionConfig{
					"TIMER": {{Target: "yellow"}},
				},
			},
			"yellow": {ID: "yellow", Type: primitives.Atomic},
		},
	}

	// Simulate post-transition snapshot
	snapshot := core.MachineSnapshot{
		MachineID:   "restore-test",
		Config:      config,
		Current:     []string{"yellow"},
		ContextData: map[string]any{"restored": true},
		Timestamp:   time.Now(),
	}
	if err := p.Save(context.Background(), snapshot); err != nil {
		t.Fatal(err)
	}

	// New machine, restore
	m2 := core.NewMachine(config)
	loaded, err := p.Load(context.Background(), "restore-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := m2.Restore(loaded); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(m2.Current(), []string{"yellow"}) {
		t.Errorf("Restored current states mismatch: got %v, want %v", m2.Current(), []string{"yellow"})
	}
}
