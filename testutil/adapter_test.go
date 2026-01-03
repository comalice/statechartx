package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/comalice/statechartx"
)

// TestAdapterInterface verifies that both adapters implement the interface correctly
func TestAdapterInterface(t *testing.T) {
	const (
		STATE_ROOT statechartx.StateID = 0
		STATE_A    statechartx.StateID = 1
		STATE_B    statechartx.StateID = 2
		EVENT_1    statechartx.EventID = 1
	)

	createTestMachine := func() *statechartx.Machine {
		stateA := &statechartx.State{
			ID: STATE_A,
			Transitions: []*statechartx.Transition{
				{
					Event:  EVENT_1,
					Target: STATE_B,
				},
			},
		}
		stateB := &statechartx.State{ID: STATE_B}

		root := &statechartx.State{
			ID:      STATE_ROOT,
			Initial: STATE_A,
			Children: map[statechartx.StateID]*statechartx.State{
				STATE_A: stateA,
				STATE_B: stateB,
			},
		}

		machine, _ := statechartx.NewMachine(root)
		return machine
	}

	tests := []struct {
		name    string
		adapter RuntimeAdapter
	}{
		{
			name:    "EventDriven",
			adapter: NewEventDrivenAdapter(createTestMachine()),
		},
		{
			name:    "TickBased",
			adapter: NewTickBasedAdapter(createTestMachine(), 10*time.Millisecond),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := tt.adapter

			ctx := context.Background()
			if err := adapter.Start(ctx); err != nil {
				t.Fatalf("Start failed: %v", err)
			}
			defer adapter.Stop()

			// Should start in state A
			if adapter.GetCurrentState() != STATE_A {
				t.Errorf("Expected initial state %d, got %d", STATE_A, adapter.GetCurrentState())
			}

			// Send event
			if err := adapter.SendEvent(statechartx.Event{ID: EVENT_1}); err != nil {
				t.Fatalf("SendEvent failed: %v", err)
			}

			// Wait for event to process
			if err := adapter.WaitForStability(1 * time.Second); err != nil {
				t.Fatalf("WaitForStability failed: %v", err)
			}

			// Should now be in state B
			if adapter.GetCurrentState() != STATE_B {
				t.Errorf("Expected state %d after transition, got %d", STATE_B, adapter.GetCurrentState())
			}
		})
	}
}

// RunCommonTests demonstrates how to run the same test logic on both runtimes
func RunCommonTests(t *testing.T, adapter RuntimeAdapter) {
	const (
		STATE_ROOT statechartx.StateID = 0
		STATE_A    statechartx.StateID = 1
		EVENT_1    statechartx.EventID = 1
	)

	ctx := context.Background()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer adapter.Stop()

	// Test 1: Initial state
	if adapter.GetCurrentState() != STATE_A {
		t.Errorf("Expected initial state %d, got %d", STATE_A, adapter.GetCurrentState())
	}

	// Test 2: IsInState
	if !adapter.IsInState(STATE_A) {
		t.Error("IsInState(STATE_A) should be true")
	}

	// Test 3: Event sending
	if err := adapter.SendEvent(statechartx.Event{ID: EVENT_1}); err != nil {
		t.Fatalf("SendEvent failed: %v", err)
	}

	adapter.WaitForStability(1 * time.Second)
}
