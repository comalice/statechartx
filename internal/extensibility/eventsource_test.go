package extensibility

import (
	"testing"
	"time"

	"github.com/comalice/statechartx/internal/primitives"
)

func TestChannelEventSource(t *testing.T) {
	ch := make(chan primitives.Event, 1)
	s := NewChannelEventSource(ch)
	if s.Events() != ch {
		t.Error("Events() should return ch")
	}
}

func TestTimerEventSource(t *testing.T) {
	s := NewTimerEventSource("tick", "data", 50*time.Millisecond)
	defer s.Stop()

	// Should receive at least one event
	select {
	case ev := <-s.Events():
		if ev.Type != "tick" || ev.Data != "data" {
			t.Errorf("wrong event: %v %v", ev.Type, ev.Data)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("no event received")
	}

	// Second event
	select {
	case ev := <-s.Events():
		if ev.Type != "tick" || ev.Data != "data" {
			t.Errorf("second wrong event: %v %v", ev.Type, ev.Data)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("no second event")
	}
}

func TestTimerEventSource_Stop(t *testing.T) {
	s := NewTimerEventSource("tick", nil, 10*time.Millisecond)
	time.Sleep(30 * time.Millisecond) // let some events
	s.Stop()
	select {
	case <-s.Events():
		// ok if drained
	default:
		// channel closed
	}
}
