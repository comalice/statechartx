package main

import (
	"context"
	"fmt"
	"time"

	"github.com/comalice/statechartx"
	"github.com/comalice/statechartx/realtime"
)

// Example: 1000 Hz Physics Simulation with deterministic updates

const (
	// Physics states
	STATE_ROOT     statechartx.StateID = 0
	STATE_IDLE     statechartx.StateID = 1
	STATE_RUNNING  statechartx.StateID = 2
	STATE_COLLIDED statechartx.StateID = 3

	// Physics events
	EVENT_START     statechartx.EventID = 1
	EVENT_STOP      statechartx.EventID = 2
	EVENT_COLLISION statechartx.EventID = 3
	EVENT_RESET     statechartx.EventID = 4
)

type PhysicsState struct {
	Position float64
	Velocity float64
	Time     float64
}

func createPhysicsStateMachine() (*statechartx.Machine, *PhysicsState) {
	physicsState := &PhysicsState{
		Position: 0.0,
		Velocity: 0.0,
		Time:     0.0,
	}

	// State: Idle
	stateIdle := &statechartx.State{
		ID: STATE_IDLE,
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_START,
				Target: STATE_RUNNING,
			},
		},
	}
	stateIdle.OnEntry(func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
		fmt.Println("[IDLE] Physics simulation idle")
		return nil
	})

	// State: Running (physics updates happen here)
	stateRunning := &statechartx.State{
		ID: STATE_RUNNING,
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_STOP,
				Target: STATE_IDLE,
			},
			{
				Event:  EVENT_COLLISION,
				Target: STATE_COLLIDED,
			},
		},
	}
	stateRunning.OnEntry(func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
		fmt.Println("[RUNNING] Physics simulation started")
		physicsState.Velocity = 10.0 // Initial velocity
		return nil
	})

	// State: Collided
	stateCollided := &statechartx.State{
		ID: STATE_COLLIDED,
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_RESET,
				Target: STATE_IDLE,
			},
		},
	}
	stateCollided.OnEntry(func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
		fmt.Printf("[COLLIDED] Collision detected at position %.2f after %.3f seconds\n",
			physicsState.Position, physicsState.Time)
		return nil
	})

	// Root state
	root := &statechartx.State{
		ID:      STATE_ROOT,
		Initial: STATE_IDLE,
		Children: map[statechartx.StateID]*statechartx.State{
			STATE_IDLE:     stateIdle,
			STATE_RUNNING:  stateRunning,
			STATE_COLLIDED: stateCollided,
		},
	}

	machine, err := statechartx.NewMachine(root)
	if err != nil {
		panic(err)
	}

	return machine, physicsState
}

func main() {
	fmt.Println("=== 1000 Hz Physics Simulation Example ===")
	fmt.Println("Simulating object movement with collision detection")

	// Create tick-based runtime (1000 Hz = 1ms per tick)
	machine, physicsState := createPhysicsStateMachine()
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate:         1 * time.Millisecond, // 1000 Hz
		MaxEventsPerTick: 10,
	})

	// Start runtime
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		panic(err)
	}
	defer rt.Stop()

	// Start physics simulation
	time.Sleep(100 * time.Millisecond)
	rt.SendEvent(statechartx.Event{ID: EVENT_START})
	fmt.Printf("Tick %d: Started simulation\n\n", rt.GetTickNumber())

	// Simulate physics updates every tick
	const dt = 0.001 // 1ms = 0.001 seconds
	const wallPosition = 50.0

	// Run simulation for 100ms
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for i := 0; i < 10; i++ {
		<-ticker.C

		if rt.GetCurrentState() == STATE_RUNNING {
			// Update physics (simplified Euler integration)
			physicsState.Time += dt * 10 // Approximate for 10ms
			physicsState.Position += physicsState.Velocity * dt * 10

			fmt.Printf("Tick %d: Position = %.2f, Velocity = %.2f\n",
				rt.GetTickNumber(), physicsState.Position, physicsState.Velocity)

			// Check for collision
			if physicsState.Position >= wallPosition {
				rt.SendEvent(statechartx.Event{ID: EVENT_COLLISION})
			}
		}
	}

	time.Sleep(100 * time.Millisecond)

	fmt.Printf("\nFinal tick: %d\n", rt.GetTickNumber())
	fmt.Printf("Final position: %.2f\n", physicsState.Position)
	fmt.Printf("Simulation time: %.3f seconds\n", physicsState.Time)

	fmt.Println("\n=== Physics Simulation Complete ===")
}
