package extensibility

import (
	"fmt"
	"testing"
	"time"

	"github.com/comalice/statechartx/internal/core"
	"github.com/comalice/statechartx/internal/primitives"
)

func TestMachineWithCustomExtensibility(t *testing.T) {
	// Simple counter statechart: increments count on TICK until count >= 3, then STOPPED
	count := 0
	config := primitives.MachineConfig{
		ID:      "counter",
		Initial: "running",
		States: map[string]*primitives.StateConfig{
			"running": primitives.NewStateConfig("running", primitives.Atomic).
				WithOn(map[string][]primitives.TransitionConfig{
					"TICK": {{
						Target:   "running",
						Guard:    "count < 3",
						Actions:  []primitives.ActionRef{func(ctx *primitives.Context, e primitives.Event) { count++; ctx.Set("count", float64(count)) }},
						Priority: 1,
					}},
					"STOP": {{Target: "stopped"}},
				}),
			"stopped": primitives.NewStateConfig("stopped", primitives.Atomic).
				WithOn(map[string][]primitives.TransitionConfig{
					"RESET": {{Target: "running"}},
				}),
		},
	}
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}

	// Custom extensibility components
	actionRunner := NewLoggingActionRunner(&DefaultActionRunner{})
	guardEval := NewExpressionGuardEvaluator()
	timerSource := NewTimerEventSource("TICK", nil, 20*time.Millisecond)

	m := core.NewMachine(config,
		core.WithActionRunner(actionRunner),
		core.WithGuardEvaluator(guardEval),
		core.WithEventSource(timerSource),
	)

	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()

	// Wait for a few ticks, expect transitions
	time.Sleep(100 * time.Millisecond)

	current := m.Current()
	if len(current) != 1 || current[0] != "running" {
		t.Errorf("expected running, got %v", current)
	}

	// Set initial count for guard test
	ctx := m.Ctx()
	ctx.Set("count", float64(0))

	// Manual tick to test guard/action
	if err := m.Send(primitives.NewEvent("TICK", nil)); err != nil {
		t.Error(err)
	}
	time.Sleep(10 * time.Millisecond)
	if count != 1 {
		t.Errorf("count should be 1, got %d", count)
	}

	// Send more ticks until guard fails
	for count < 3 {
		if err := m.Send(primitives.NewEvent("TICK", nil)); err != nil {
			t.Error(err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// After 3 ticks, no more transitions (guard false)
	if count != 3 {
		t.Errorf("count should be 3, got %d", count)
	}

	// Verify guard blocks further
	if err := m.Send(primitives.NewEvent("TICK", nil)); err != nil {
		t.Error(err)
	}
	time.Sleep(10 * time.Millisecond)
	if count != 3 {
		t.Error("guard failed to block")
	}

	// Logs should show actions executed 3 times
	fmt.Println("Integration test complete - check logs for action executions")
}
