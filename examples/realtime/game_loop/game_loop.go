package main

import (
	"context"
	"fmt"
	"time"

	"github.com/comalice/statechartx"
	"github.com/comalice/statechartx/realtime"
)

// Example: 60 FPS Game Loop with deterministic state management

const (
	// Game states
	STATE_ROOT     statechartx.StateID = 0
	STATE_MENU     statechartx.StateID = 1
	STATE_PLAYING  statechartx.StateID = 2
	STATE_PAUSED   statechartx.StateID = 3
	STATE_GAMEOVER statechartx.StateID = 4

	// Game events
	EVENT_START_GAME statechartx.EventID = 1
	EVENT_PAUSE      statechartx.EventID = 2
	EVENT_RESUME     statechartx.EventID = 3
	EVENT_GAME_OVER  statechartx.EventID = 4
	EVENT_BACK_MENU  statechartx.EventID = 5
)

func createGameStateMachine() *statechartx.Machine {
	// State: Menu
	stateMenu := &statechartx.State{
		ID: STATE_MENU,
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_START_GAME,
				Target: STATE_PLAYING,
			},
		},
	}
	stateMenu.OnEntry(func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
		fmt.Println("[MENU] Welcome to the game!")
		return nil
	})

	// State: Playing
	statePlaying := &statechartx.State{
		ID: STATE_PLAYING,
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_PAUSE,
				Target: STATE_PAUSED,
			},
			{
				Event:  EVENT_GAME_OVER,
				Target: STATE_GAMEOVER,
			},
		},
	}
	statePlaying.OnEntry(func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
		fmt.Println("[PLAYING] Game started!")
		return nil
	})

	// State: Paused
	statePaused := &statechartx.State{
		ID: STATE_PAUSED,
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_RESUME,
				Target: STATE_PLAYING,
			},
			{
				Event:  EVENT_BACK_MENU,
				Target: STATE_MENU,
			},
		},
	}
	statePaused.OnEntry(func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
		fmt.Println("[PAUSED] Game paused")
		return nil
	})

	// State: Game Over
	stateGameOver := &statechartx.State{
		ID: STATE_GAMEOVER,
		Transitions: []*statechartx.Transition{
			{
				Event:  EVENT_BACK_MENU,
				Target: STATE_MENU,
			},
		},
	}
	stateGameOver.OnEntry(func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
		fmt.Println("[GAME OVER] Thanks for playing!")
		return nil
	})

	// Root state
	root := &statechartx.State{
		ID:      STATE_ROOT,
		Initial: STATE_MENU,
		Children: map[statechartx.StateID]*statechartx.State{
			STATE_MENU:     stateMenu,
			STATE_PLAYING:  statePlaying,
			STATE_PAUSED:   statePaused,
			STATE_GAMEOVER: stateGameOver,
		},
	}

	machine, err := statechartx.NewMachine(root)
	if err != nil {
		panic(err)
	}

	return machine
}

func main() {
	fmt.Println("=== 60 FPS Game Loop Example ===")

	// Create tick-based runtime (60 FPS)
	machine := createGameStateMachine()
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate:         16667 * time.Microsecond, // 60 FPS
		MaxEventsPerTick: 100,
	})

	// Start runtime
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		panic(err)
	}
	defer rt.Stop()

	// Simulate game loop for 10 seconds
	fmt.Println("Simulating game events over 10 seconds @ 60 FPS...")

	// Event 1: Start game at 1 second (tick 60)
	time.Sleep(1 * time.Second)
	rt.SendEvent(statechartx.Event{ID: EVENT_START_GAME})
	fmt.Printf("Tick %d: Sent EVENT_START_GAME\n\n", rt.GetTickNumber())

	// Event 2: Pause at 3 seconds (tick 180)
	time.Sleep(2 * time.Second)
	rt.SendEvent(statechartx.Event{ID: EVENT_PAUSE})
	fmt.Printf("Tick %d: Sent EVENT_PAUSE\n\n", rt.GetTickNumber())

	// Event 3: Resume at 5 seconds (tick 300)
	time.Sleep(2 * time.Second)
	rt.SendEvent(statechartx.Event{ID: EVENT_RESUME})
	fmt.Printf("Tick %d: Sent EVENT_RESUME\n\n", rt.GetTickNumber())

	// Event 4: Game over at 7 seconds (tick 420)
	time.Sleep(2 * time.Second)
	rt.SendEvent(statechartx.Event{ID: EVENT_GAME_OVER})
	fmt.Printf("Tick %d: Sent EVENT_GAME_OVER\n\n", rt.GetTickNumber())

	// Event 5: Back to menu at 9 seconds (tick 540)
	time.Sleep(2 * time.Second)
	rt.SendEvent(statechartx.Event{ID: EVENT_BACK_MENU})
	fmt.Printf("Tick %d: Sent EVENT_BACK_MENU\n\n", rt.GetTickNumber())

	// Wait a bit more to see final state
	time.Sleep(1 * time.Second)

	fmt.Printf("\nFinal tick: %d\n", rt.GetTickNumber())
	fmt.Printf("Final state: %d (should be STATE_MENU = %d)\n", rt.GetCurrentState(), STATE_MENU)

	fmt.Println("\n=== Game Loop Complete ===")
}
