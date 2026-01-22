package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/comalice/statechartx"
)

// State IDs - using numeric constants for clarity
const (
	RootState statechartx.StateID = 1

	IdleState   statechartx.StateID = 2
	ActiveState statechartx.StateID = 3
	ErrorState  statechartx.StateID = 4
)

// Event IDs
const (
	ActivateEvent   statechartx.EventID = 10
	DeactivateEvent statechartx.EventID = 11
	ErrorEvent      statechartx.EventID = 12
	ResetEvent      statechartx.EventID = 13
)

func main() {
	fmt.Println("StatechartX Basic Example")
	fmt.Println("==========================")
	fmt.Println()

	// Create the state machine hierarchy
	machine := createStateMachine()

	// Create runtime
	rt := statechartx.NewRuntime(machine, nil)

	// Start the runtime
	ctx := context.Background()
	rt.Start(ctx)
	defer rt.Stop()

	fmt.Println("State machine started")
	fmt.Println()

	// Scenario: Normal operation cycle
	fmt.Println("--- Scenario 1: Normal Operation ---")
	fmt.Println("Sending ACTIVATE event...")
	rt.SendEvent(ctx, statechartx.Event{ID: ActivateEvent})
	time.Sleep(50 * time.Millisecond) // Allow processing

	fmt.Println("Sending DEACTIVATE event...")
	rt.SendEvent(ctx, statechartx.Event{ID: DeactivateEvent})
	time.Sleep(50 * time.Millisecond)

	// Scenario: Error handling
	fmt.Println("\n--- Scenario 2: Error Handling ---")
	fmt.Println("Sending ACTIVATE event...")
	rt.SendEvent(ctx, statechartx.Event{ID: ActivateEvent})
	time.Sleep(50 * time.Millisecond)

	fmt.Println("Sending ERROR event...")
	rt.SendEvent(ctx, statechartx.Event{ID: ErrorEvent})
	time.Sleep(50 * time.Millisecond)

	fmt.Println("Sending RESET event to recover...")
	rt.SendEvent(ctx, statechartx.Event{ID: ResetEvent})
	time.Sleep(50 * time.Millisecond)

	fmt.Println("\n--- Example Complete ---")
	fmt.Println("State machine demonstration finished successfully!")
}

func createStateMachine() *statechartx.Machine {
	// Create root state
	root := &statechartx.State{
		ID:      RootState,
		Initial: IdleState, // Start in idle state
	}

	// Create idle state with entry action
	idle := &statechartx.State{
		ID:     IdleState,
		Parent: root,
		EntryAction: func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
			fmt.Println("  → Entered IDLE state")
			return nil
		},
		ExitAction: func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
			fmt.Println("  ← Exiting IDLE state")
			return nil
		},
	}

	// Create active state with entry/exit actions
	active := &statechartx.State{
		ID:     ActiveState,
		Parent: root,
		EntryAction: func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
			fmt.Println("  → Entered ACTIVE state - system is now running")
			return nil
		},
		ExitAction: func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
			fmt.Println("  ← Exiting ACTIVE state - stopping operations")
			return nil
		},
	}

	// Create error state
	errorState := &statechartx.State{
		ID:     ErrorState,
		Parent: root,
		EntryAction: func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
			fmt.Println("  → Entered ERROR state - system needs attention!")
			return nil
		},
		ExitAction: func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
			fmt.Println("  ← Exiting ERROR state - attempting recovery")
			return nil
		},
	}

	// Build hierarchy
	root.Children = map[statechartx.StateID]*statechartx.State{
		IdleState:   idle,
		ActiveState: active,
		ErrorState:  errorState,
	}

	// Define transitions for idle state
	idle.Transitions = []*statechartx.Transition{
		{
			Event:  ActivateEvent,
			Target: ActiveState,
			Action: func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
				fmt.Println("  ⚡ Transition: IDLE → ACTIVE")
				return nil
			},
		},
	}

	// Define transitions for active state
	active.Transitions = []*statechartx.Transition{
		{
			Event:  DeactivateEvent,
			Target: IdleState,
			Action: func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
				fmt.Println("  ⚡ Transition: ACTIVE → IDLE")
				return nil
			},
		},
		{
			Event:  ErrorEvent,
			Target: ErrorState,
			Action: func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
				fmt.Println("  ⚡ Transition: ACTIVE → ERROR (fault detected)")
				return nil
			},
		},
	}

	// Define transitions for error state
	errorState.Transitions = []*statechartx.Transition{
		{
			Event:  ResetEvent,
			Target: IdleState,
			Action: func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
				fmt.Println("  ⚡ Transition: ERROR → IDLE (system reset)")
				return nil
			},
		},
	}

	// Create machine
	machine, err := statechartx.NewMachine(root)
	if err != nil {
		log.Fatalf("Failed to create machine: %v", err)
	}

	return machine
}
