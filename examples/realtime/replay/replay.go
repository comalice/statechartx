package main

import (
	"context"
	"fmt"
	"time"

	"github.com/comalice/statechartx"
	"github.com/comalice/statechartx/realtime"
)

// Example: Deterministic Replay
// Record events with sequence numbers, then replay them to get identical results

const (
	// States
	STATE_ROOT statechartx.StateID = 0
	STATE_A    statechartx.StateID = 1
	STATE_B    statechartx.StateID = 2
	STATE_C    statechartx.StateID = 3

	// Events
	EVENT_A_TO_B statechartx.EventID = 1
	EVENT_B_TO_C statechartx.EventID = 2
	EVENT_C_TO_A statechartx.EventID = 3
)

// EventRecord records events for replay
type EventRecord struct {
	TickNumber uint64
	Event      statechartx.Event
}

func createReplayStateMachine() *statechartx.Machine {
	stateA := &statechartx.State{
		ID: STATE_A,
		Transitions: []*statechartx.Transition{
			{Event: EVENT_A_TO_B, Target: STATE_B},
		},
	}
	stateA.OnEntry(func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
		fmt.Println("  [STATE_A] Entered")
		return nil
	})

	stateB := &statechartx.State{
		ID: STATE_B,
		Transitions: []*statechartx.Transition{
			{Event: EVENT_B_TO_C, Target: STATE_C},
		},
	}
	stateB.OnEntry(func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
		fmt.Println("  [STATE_B] Entered")
		return nil
	})

	stateC := &statechartx.State{
		ID: STATE_C,
		Transitions: []*statechartx.Transition{
			{Event: EVENT_C_TO_A, Target: STATE_A},
		},
	}
	stateC.OnEntry(func(ctx context.Context, evt *statechartx.Event, from, to statechartx.StateID) error {
		fmt.Println("  [STATE_C] Entered")
		return nil
	})

	root := &statechartx.State{
		ID:      STATE_ROOT,
		Initial: STATE_A,
		Children: map[statechartx.StateID]*statechartx.State{
			STATE_A: stateA,
			STATE_B: stateB,
			STATE_C: stateC,
		},
	}

	machine, err := statechartx.NewMachine(root)
	if err != nil {
		panic(err)
	}

	return machine
}

func recordRun() []EventRecord {
	fmt.Println("\n=== Recording Session ===")

	machine := createReplayStateMachine()
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate:         10 * time.Millisecond,
		MaxEventsPerTick: 100,
	})

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		panic(err)
	}
	defer rt.Stop()

	var recording []EventRecord

	// Record event 1
	time.Sleep(20 * time.Millisecond)
	event1 := statechartx.Event{ID: EVENT_A_TO_B}
	rt.SendEvent(event1)
	ticksnapshot1 := rt.GetTickNumber()
	recording = append(recording, EventRecord{TickNumber: ticksnapshot1, Event: event1})
	fmt.Printf("Recorded: Tick %d, Event %d\n", ticksnapshot1, event1.ID)

	// Record event 2
	time.Sleep(30 * time.Millisecond)
	event2 := statechartx.Event{ID: EVENT_B_TO_C}
	rt.SendEvent(event2)
	ticksnapshot2 := rt.GetTickNumber()
	recording = append(recording, EventRecord{TickNumber: ticksnapshot2, Event: event2})
	fmt.Printf("Recorded: Tick %d, Event %d\n", ticksnapshot2, event2.ID)

	// Record event 3
	time.Sleep(25 * time.Millisecond)
	event3 := statechartx.Event{ID: EVENT_C_TO_A}
	rt.SendEvent(event3)
	ticksnapshot3 := rt.GetTickNumber()
	recording = append(recording, EventRecord{TickNumber: ticksnapshot3, Event: event3})
	fmt.Printf("Recorded: Tick %d, Event %d\n", ticksnapshot3, event3.ID)

	time.Sleep(20 * time.Millisecond)

	finalState := rt.GetCurrentState()
	fmt.Printf("Final state: %d\n", finalState)

	return recording
}

func replayRun(recording []EventRecord) {
	fmt.Println("\n=== Replay Session ===")

	machine := createReplayStateMachine()
	rt := realtime.NewRuntime(machine, realtime.Config{
		TickRate:         10 * time.Millisecond,
		MaxEventsPerTick: 100,
	})

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		panic(err)
	}
	defer rt.Stop()

	// Replay events at the same tick numbers
	for _, record := range recording {
		// Wait until we reach the recorded tick
		for rt.GetTickNumber() < record.TickNumber {
			time.Sleep(1 * time.Millisecond)
		}

		// Send event
		rt.SendEvent(record.Event)
		fmt.Printf("Replayed: Tick %d, Event %d\n", rt.GetTickNumber(), record.Event.ID)
	}

	// Wait for final event to process
	time.Sleep(20 * time.Millisecond)

	finalState := rt.GetCurrentState()
	fmt.Printf("Final state: %d\n", finalState)
}

func main() {
	fmt.Println("=== Deterministic Replay Example ===")
	fmt.Println("This example demonstrates how tick-based execution")
	fmt.Println("allows for perfect replay of recorded event sequences.")

	// Record a session
	recording := recordRun()

	// Replay the session
	replayRun(recording)

	fmt.Println("\n=== Replay Complete ===")
	fmt.Println("Both runs should have identical state transitions!")
}
