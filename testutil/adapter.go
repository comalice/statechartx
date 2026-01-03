package testutil

import (
	"context"
	"time"

	"github.com/comalice/statechartx"
	"github.com/comalice/statechartx/realtime"
)

// RuntimeAdapter provides a common interface for both event-driven and tick-based runtimes
// This allows running the same test suite on both runtimes
type RuntimeAdapter interface {
	Start(ctx context.Context) error
	Stop() error
	SendEvent(event statechartx.Event) error
	IsInState(stateID statechartx.StateID) bool
	GetCurrentState() statechartx.StateID
	WaitForStability(timeout time.Duration) error
}

// EventDrivenAdapter wraps the event-driven runtime
type EventDrivenAdapter struct {
	rt *statechartx.Runtime
}

// NewEventDrivenAdapter creates a new adapter for the event-driven runtime
func NewEventDrivenAdapter(machine *statechartx.Machine) *EventDrivenAdapter {
	return &EventDrivenAdapter{
		rt: statechartx.NewRuntime(machine, nil),
	}
}

func (a *EventDrivenAdapter) Start(ctx context.Context) error {
	return a.rt.Start(ctx)
}

func (a *EventDrivenAdapter) Stop() error {
	return a.rt.Stop()
}

func (a *EventDrivenAdapter) SendEvent(event statechartx.Event) error {
	return a.rt.SendEvent(context.Background(), event)
}

func (a *EventDrivenAdapter) IsInState(stateID statechartx.StateID) bool {
	return a.rt.IsInState(stateID)
}

func (a *EventDrivenAdapter) GetCurrentState() statechartx.StateID {
	return a.rt.GetCurrentState()
}

func (a *EventDrivenAdapter) WaitForStability(timeout time.Duration) error {
	// Event-driven processes immediately, small delay for goroutine scheduling
	time.Sleep(5 * time.Millisecond)
	return nil
}

// TickBasedAdapter wraps the tick-based runtime
type TickBasedAdapter struct {
	rt       *realtime.RealtimeRuntime
	tickRate time.Duration
}

// NewTickBasedAdapter creates a new adapter for the tick-based runtime
func NewTickBasedAdapter(machine *statechartx.Machine, tickRate time.Duration) *TickBasedAdapter {
	return &TickBasedAdapter{
		rt: realtime.NewRuntime(machine, realtime.Config{
			TickRate: tickRate,
		}),
		tickRate: tickRate,
	}
}

func (a *TickBasedAdapter) Start(ctx context.Context) error {
	return a.rt.Start(ctx)
}

func (a *TickBasedAdapter) Stop() error {
	return a.rt.Stop()
}

func (a *TickBasedAdapter) SendEvent(event statechartx.Event) error {
	return a.rt.SendEvent(event)
}

func (a *TickBasedAdapter) IsInState(stateID statechartx.StateID) bool {
	return a.rt.IsInState(stateID)
}

func (a *TickBasedAdapter) GetCurrentState() statechartx.StateID {
	return a.rt.GetCurrentState()
}

func (a *TickBasedAdapter) WaitForStability(timeout time.Duration) error {
	// Wait for next tick to process event
	time.Sleep(a.tickRate + 5*time.Millisecond)
	return nil
}
