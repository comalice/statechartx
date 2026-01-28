package main

import (
	"context"
	"fmt"
	"time"

	"github.com/comalice/statechartx"
)

func main() {
	// Example 1: Simple Traffic Light
	fmt.Println("=== Traffic Light Example ===")
	trafficLight()
	fmt.Println()

	// Example 2: Compound States (Nested)
	fmt.Println("=== Application States Example ===")
	applicationStates()
	fmt.Println()

	// Example 3: With Actions and Context
	fmt.Println("=== Actions and Context Example ===")
	actionsAndContext()
}

func trafficLight() {
	// Create a builder for a traffic light state machine
	b := statechartx.NewMachineBuilder("traffic", "green")

	// Define states and transitions using fluent API
	b.State("green").Atomic().
		On("timer", "yellow", nil, nil)

	b.State("yellow").Atomic().
		On("timer", "red", nil, nil)

	b.State("red").Atomic().
		On("timer", "green", nil, nil)

	// Build the machine
	machine, err := b.Build()
	if err != nil {
		panic(err)
	}

	// Create runtime with auto-created Context
	rt := statechartx.NewRuntime(machine, nil)
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		panic(err)
	}
	defer rt.Stop()

	// Helper to get current state name
	getCurrentState := func() string {
		if rt.IsInState(b.GetID("green")) {
			return "green"
		}
		if rt.IsInState(b.GetID("yellow")) {
			return "yellow"
		}
		if rt.IsInState(b.GetID("red")) {
			return "red"
		}
		return "unknown"
	}

	fmt.Printf("Initial state: %s\n", getCurrentState())

	// Cycle through states
	timerEventID := statechartx.EventID(b.GetID("event:timer"))
	for i := 0; i < 3; i++ {
		rt.SendEvent(ctx, statechartx.Event{ID: timerEventID})
		time.Sleep(50 * time.Millisecond)
		fmt.Printf("After timer event %d: %s\n", i+1, getCurrentState())
	}
}

func applicationStates() {
	// Create a builder with nested compound states
	b := statechartx.NewMachineBuilder("app", "off")

	// Define states
	b.State("off").Atomic().
		On("power_on", "on.idle", nil, nil)

	b.State("on").Compound("on.idle").
		On("power_off", "off", nil, nil)

	b.State("on.idle").Atomic().
		On("start_work", "on.working", nil, nil)

	b.State("on.working").Atomic().
		On("finish_work", "on.idle", nil, nil)

	machine, err := b.Build()
	if err != nil {
		panic(err)
	}

	rt := statechartx.NewRuntime(machine, nil)
	ctx := context.Background()

	if err := rt.Start(ctx); err != nil {
		panic(err)
	}
	defer rt.Stop()

	getCurrentState := func() string {
		if rt.IsInState(b.GetID("off")) {
			return "off"
		}
		if rt.IsInState(b.GetID("on.idle")) {
			return "on.idle"
		}
		if rt.IsInState(b.GetID("on.working")) {
			return "on.working"
		}
		return "unknown"
	}

	fmt.Printf("Initial state: %s\n", getCurrentState())

	// Power on
	rt.SendEvent(ctx, statechartx.Event{ID: statechartx.EventID(b.GetID("event:power_on"))})
	time.Sleep(50 * time.Millisecond)
	fmt.Printf("After power_on: %s\n", getCurrentState())

	// Start working
	rt.SendEvent(ctx, statechartx.Event{ID: statechartx.EventID(b.GetID("event:start_work"))})
	time.Sleep(50 * time.Millisecond)
	fmt.Printf("After start_work: %s\n", getCurrentState())

	// Power off (from nested state)
	rt.SendEvent(ctx, statechartx.Event{ID: statechartx.EventID(b.GetID("event:power_off"))})
	time.Sleep(50 * time.Millisecond)
	fmt.Printf("After power_off: %s\n", getCurrentState())
}

func actionsAndContext() {
	b := statechartx.NewMachineBuilder("counter", "idle")

	// Define actions that use context
	incrementAction := func(ctx context.Context, evt *statechartx.Event, from statechartx.StateID, to statechartx.StateID) error {
		fmt.Println("  [Action] Incrementing counter...")
		return nil
	}

	entryAction := func(ctx context.Context, evt *statechartx.Event, from statechartx.StateID, to statechartx.StateID) error {
		fmt.Printf("  [Entry] Entering state %d\n", to)
		return nil
	}

	exitAction := func(ctx context.Context, evt *statechartx.Event, from statechartx.StateID, to statechartx.StateID) error {
		fmt.Printf("  [Exit] Exiting state %d\n", from)
		return nil
	}

	// Build machine with actions
	b.State("idle").Atomic().
		Entry(entryAction).
		Exit(exitAction).
		On("start", "counting", nil, incrementAction)

	b.State("counting").Atomic().
		Entry(entryAction).
		On("stop", "idle", nil, nil)

	machine, err := b.Build()
	if err != nil {
		panic(err)
	}

	rt := statechartx.NewRuntime(machine, nil)
	ctx := rt.Ctx() // Get the auto-created Context

	// Store some data in context
	ctx.Set("counter", 0)
	ctx.Set("name", "Example Machine")

	bgCtx := context.Background()
	if err := rt.Start(bgCtx); err != nil {
		panic(err)
	}
	defer rt.Stop()

	time.Sleep(50 * time.Millisecond)

	fmt.Println("Sending start event...")
	rt.SendEvent(bgCtx, statechartx.Event{ID: statechartx.EventID(b.GetID("event:start"))})
	time.Sleep(50 * time.Millisecond)

	fmt.Println("Sending stop event...")
	rt.SendEvent(bgCtx, statechartx.Event{ID: statechartx.EventID(b.GetID("event:stop"))})
	time.Sleep(50 * time.Millisecond)

	// Access context data
	fmt.Printf("\nContext data: name=%v, counter=%v\n", ctx.Get("name"), ctx.Get("counter"))
}
