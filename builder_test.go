package statechartx_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/comalice/statechartx"
)

func TestBuilderTrafficLight(t *testing.T) {
	b := NewMachineBuilder("traffic", "green")

	b.State("green").Atomic().On("timer", "yellow", nil, nil)
	b.State("yellow").Atomic().On("timer", "red", nil, nil)
	b.State("red").Atomic().On("timer", "green", nil, nil)

	machine, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	// Test initial state
	greenID := b.GetID("green")
	if !rt.IsInState(greenID) {
		t.Error("should start in green")
	}

	// Test transitions
	timerEventID := EventID(b.GetID("event:timer"))

	rt.SendEvent(ctx, Event{ID: timerEventID})
	time.Sleep(50 * time.Millisecond)

	yellowID := b.GetID("yellow")
	if !rt.IsInState(yellowID) {
		t.Error("should transition to yellow")
	}

	rt.SendEvent(ctx, Event{ID: timerEventID})
	time.Sleep(50 * time.Millisecond)

	redID := b.GetID("red")
	if !rt.IsInState(redID) {
		t.Error("should transition to red")
	}

	rt.SendEvent(ctx, Event{ID: timerEventID})
	time.Sleep(50 * time.Millisecond)

	if !rt.IsInState(greenID) {
		t.Error("should cycle back to green")
	}
}

func TestBuilderCompoundStates(t *testing.T) {
	b := NewMachineBuilder("app", "off")

	b.State("off").Atomic().On("power", "on.idle", nil, nil)
	b.State("on").Compound("on.idle")
	b.State("on.idle").Atomic().On("work", "on.working", nil, nil)
	b.State("on.working").Atomic().On("done", "on.idle", nil, nil)
	b.State("on").On("power", "off", nil, nil)

	machine, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	// Start in off
	offID := b.GetID("off")
	if !rt.IsInState(offID) {
		t.Error("should start in off")
	}

	// Power on -> should enter on.idle
	powerEventID := EventID(b.GetID("event:power"))
	rt.SendEvent(ctx, Event{ID: powerEventID})
	time.Sleep(50 * time.Millisecond)

	onIdleID := b.GetID("on.idle")
	if !rt.IsInState(onIdleID) {
		t.Error("should be in on.idle after power on")
	}

	// Start working
	workEventID := EventID(b.GetID("event:work"))
	rt.SendEvent(ctx, Event{ID: workEventID})
	time.Sleep(50 * time.Millisecond)

	onWorkingID := b.GetID("on.working")
	if !rt.IsInState(onWorkingID) {
		t.Error("should be in on.working")
	}

	// Power off from nested state
	rt.SendEvent(ctx, Event{ID: powerEventID})
	time.Sleep(50 * time.Millisecond)

	if !rt.IsInState(offID) {
		t.Error("should be off after power event from nested state")
	}
}

func TestBuilderParallelStates(t *testing.T) {
	b := NewMachineBuilder("app", "running")

	b.State("running").Parallel()
	b.State("running.audio").Compound("running.audio.playing")
	b.State("running.audio.playing").Atomic().On("pause_audio", "running.audio.paused", nil, nil)
	b.State("running.audio.paused").Atomic().On("play_audio", "running.audio.playing", nil, nil)

	b.State("running.video").Compound("running.video.playing")
	b.State("running.video.playing").Atomic().On("pause_video", "running.video.paused", nil, nil)
	b.State("running.video.paused").Atomic().On("play_video", "running.video.playing", nil, nil)

	machine, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	time.Sleep(100 * time.Millisecond) // Let parallel regions initialize

	// Both regions should be active
	audioPlayingID := b.GetID("running.audio.playing")
	videoPlayingID := b.GetID("running.video.playing")

	if !rt.IsInState(audioPlayingID) {
		t.Error("audio should be playing")
	}
	if !rt.IsInState(videoPlayingID) {
		t.Error("video should be playing")
	}

	// Pause audio only
	pauseAudioID := EventID(b.GetID("event:pause_audio"))
	rt.SendEvent(ctx, Event{ID: pauseAudioID})
	time.Sleep(50 * time.Millisecond)

	audioPausedID := b.GetID("running.audio.paused")
	if !rt.IsInState(audioPausedID) {
		t.Error("audio should be paused")
	}
	if !rt.IsInState(videoPlayingID) {
		t.Error("video should still be playing")
	}
}

func TestBuilderFinalState(t *testing.T) {
	b := NewMachineBuilder("workflow", "start")

	b.State("start").Atomic().On("begin", "processing", nil, nil)
	b.State("processing").Atomic().On("complete", "done", nil, nil)
	b.State("done").Final("success")

	machine, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	// Transition to final state
	beginID := EventID(b.GetID("event:begin"))
	rt.SendEvent(ctx, Event{ID: beginID})
	time.Sleep(50 * time.Millisecond)

	completeID := EventID(b.GetID("event:complete"))
	rt.SendEvent(ctx, Event{ID: completeID})
	time.Sleep(50 * time.Millisecond)

	doneID := b.GetID("done")
	if !rt.IsInState(doneID) {
		t.Error("should be in done (final) state")
	}
}

func TestBuilderActions(t *testing.T) {
	b := NewMachineBuilder("counter", "idle")

	var entryCount, exitCount, transitionCount int32

	entryAction := func(ctx context.Context, evt *Event, from StateID, to StateID) error {
		atomic.AddInt32(&entryCount, 1)
		return nil
	}

	exitAction := func(ctx context.Context, evt *Event, from StateID, to StateID) error {
		atomic.AddInt32(&exitCount, 1)
		return nil
	}

	transitionAction := func(ctx context.Context, evt *Event, from StateID, to StateID) error {
		atomic.AddInt32(&transitionCount, 1)
		return nil
	}

	b.State("idle").Atomic().
		Entry(entryAction).
		Exit(exitAction).
		On("start", "active", nil, transitionAction)

	b.State("active").Atomic().
		Entry(entryAction).
		On("stop", "idle", nil, nil)

	machine, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	time.Sleep(50 * time.Millisecond)

	// Entry action should have fired for idle
	if atomic.LoadInt32(&entryCount) != 1 {
		t.Errorf("expected 1 entry, got %d", entryCount)
	}

	// Transition to active
	startID := EventID(b.GetID("event:start"))
	rt.SendEvent(ctx, Event{ID: startID})
	time.Sleep(50 * time.Millisecond)

	// Should have exit(idle) + transition + entry(active)
	if atomic.LoadInt32(&exitCount) != 1 {
		t.Errorf("expected 1 exit, got %d", exitCount)
	}
	if atomic.LoadInt32(&transitionCount) != 1 {
		t.Errorf("expected 1 transition action, got %d", transitionCount)
	}
	if atomic.LoadInt32(&entryCount) != 2 {
		t.Errorf("expected 2 entries, got %d", entryCount)
	}
}

func TestBuilderGuards(t *testing.T) {
	b := NewMachineBuilder("guarded", "start")

	allowTransition := true
	guard := func(ctx context.Context, evt *Event, from StateID, to StateID) (bool, error) {
		return allowTransition, nil
	}

	b.State("start").Atomic().On("next", "end", guard, nil)
	b.State("end").Atomic()

	machine, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	startID := b.GetID("start")
	endID := b.GetID("end")
	nextEventID := EventID(b.GetID("event:next"))

	// Guard blocks transition
	allowTransition = false
	rt.SendEvent(ctx, Event{ID: nextEventID})
	time.Sleep(50 * time.Millisecond)

	if !rt.IsInState(startID) {
		t.Error("should still be in start (guard blocked)")
	}

	// Guard allows transition
	allowTransition = true
	rt.SendEvent(ctx, Event{ID: nextEventID})
	time.Sleep(50 * time.Millisecond)

	if !rt.IsInState(endID) {
		t.Error("should be in end (guard allowed)")
	}
}

func TestBuilderInternalTransition(t *testing.T) {
	b := NewMachineBuilder("app", "running")

	var actionCount int32
	internalAction := func(ctx context.Context, evt *Event, from StateID, to StateID) error {
		atomic.AddInt32(&actionCount, 1)
		return nil
	}

	var entryCount, exitCount int32
	entryAction := func(ctx context.Context, evt *Event, from StateID, to StateID) error {
		atomic.AddInt32(&entryCount, 1)
		return nil
	}
	exitAction := func(ctx context.Context, evt *Event, from StateID, to StateID) error {
		atomic.AddInt32(&exitCount, 1)
		return nil
	}

	b.State("running").Atomic().
		Entry(entryAction).
		Exit(exitAction).
		OnInternal("update", nil, internalAction)

	machine, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}

	rt := NewRuntime(machine, nil)
	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer rt.Stop()

	time.Sleep(50 * time.Millisecond)

	// Entry should fire once
	if atomic.LoadInt32(&entryCount) != 1 {
		t.Errorf("expected 1 entry, got %d", entryCount)
	}

	// Send internal transition event
	updateEventID := EventID(b.GetID("event:update"))
	rt.SendEvent(ctx, Event{ID: updateEventID})
	time.Sleep(50 * time.Millisecond)

	// Internal action should fire, but not entry/exit
	if atomic.LoadInt32(&actionCount) != 1 {
		t.Errorf("expected 1 internal action, got %d", actionCount)
	}
	if atomic.LoadInt32(&exitCount) != 0 {
		t.Errorf("expected 0 exits (internal transition), got %d", exitCount)
	}
	if atomic.LoadInt32(&entryCount) != 1 {
		t.Errorf("expected still 1 entry (internal transition), got %d", entryCount)
	}

	runningID := b.GetID("running")
	if !rt.IsInState(runningID) {
		t.Error("should still be in running after internal transition")
	}
}

func TestBuilderDeterministicIDs(t *testing.T) {
	// Build same machine twice
	b1 := NewMachineBuilder("app", "idle")
	b1.State("idle").Atomic().On("start", "active", nil, nil)
	b1.State("active").Atomic().On("stop", "idle", nil, nil)

	b2 := NewMachineBuilder("app", "idle")
	b2.State("idle").Atomic().On("start", "active", nil, nil)
	b2.State("active").Atomic().On("stop", "idle", nil, nil)

	// IDs should match
	if b1.GetID("idle") != b2.GetID("idle") {
		t.Error("idle IDs should match across builds")
	}
	if b1.GetID("active") != b2.GetID("active") {
		t.Error("active IDs should match across builds")
	}
	if b1.GetID("event:start") != b2.GetID("event:start") {
		t.Error("start event IDs should match across builds")
	}
}

func TestBuilderGetNameReverseLookup(t *testing.T) {
	b := NewMachineBuilder("app", "state1")
	b.State("state1").Atomic()
	b.State("state2").Atomic()

	id1 := b.GetID("state1")
	id2 := b.GetID("state2")

	if b.GetName(id1) != "state1" {
		t.Error("GetName should return state1")
	}
	if b.GetName(id2) != "state2" {
		t.Error("GetName should return state2")
	}
	if b.GetName(999) != "" {
		t.Error("GetName for unknown ID should return empty string")
	}
}

func TestBuilderValidationMissingInitial(t *testing.T) {
	b := NewMachineBuilder("app", "parent")
	b.State("parent").Compound("child") // Declares initial but doesn't create child

	_, err := b.Build()
	if err == nil {
		t.Error("should error on missing initial state")
	}
}

func TestBuilderContextIntegration(t *testing.T) {
	b := NewMachineBuilder("app", "idle")

	action := func(ctx context.Context, evt *Event, from StateID, to StateID) error {
		// Access runtime context (would need to pass runtime in real usage)
		return nil
	}

	b.State("idle").Atomic().On("start", "active", nil, action)
	b.State("active").Atomic()

	machine, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}

	// Auto-created Context
	rt := NewRuntime(machine, nil)
	ctx := rt.Ctx()
	if ctx == nil {
		t.Fatal("expected auto-created Context")
	}

	// Should be able to use context
	ctx.Set("test_key", "test_value")
	if ctx.Get("test_key") != "test_value" {
		t.Error("Context should work with builder-created machine")
	}
}
