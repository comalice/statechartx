package extensibility

import (
	"time"

	"github.com/comalice/statechartx/internal/primitives"
)

// ChannelEventSource is an EventSource implementation backed by a Go channel.
// Provides a simple way to feed external events into the Machine via Send().
type ChannelEventSource struct {
	ch chan primitives.Event
}

// Events returns the receive-only channel for events.
func (s *ChannelEventSource) Events() <-chan primitives.Event {
	return s.ch
}

// NewChannelEventSource creates a new ChannelEventSource with the given channel.
// The channel should be buffered if backpressure handling is needed.
func NewChannelEventSource(ch chan primitives.Event) *ChannelEventSource {
	return &ChannelEventSource{ch: ch}
}

// TimerEventSource generates periodic events using time.Ticker.
// Useful for timeout/heartbeat statecharts.
type TimerEventSource struct {
	ch        chan primitives.Event
	eventType string
	data      any
	ticker    *time.Ticker
	stop      chan struct{}
}

// NewTimerEventSource creates a TimerEventSource that emits events every d duration.
func NewTimerEventSource(eventType string, data any, d time.Duration) *TimerEventSource {
	ch := make(chan primitives.Event, 10)
	t := &TimerEventSource{
		ch:        ch,
		eventType: eventType,
		data:      data,
		ticker:    time.NewTicker(d),
		stop:      make(chan struct{}),
	}
	go t.run()
	return t
}

func (t *TimerEventSource) run() {
	for {
		select {
		case <-t.ticker.C:
			select {
			case t.ch <- primitives.NewEvent(t.eventType, t.data):
			default:
				// drop if full
			}
		case <-t.stop:
			t.ticker.Stop()
			close(t.ch)
			return
		}
	}
}

// Events returns the event channel.
func (t *TimerEventSource) Events() <-chan primitives.Event {
	return t.ch
}

// Stop stops the ticker and closes the channel.
func (t *TimerEventSource) Stop() {
	close(t.stop)
}
