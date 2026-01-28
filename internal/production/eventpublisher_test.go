// Tests for ChannelPublisher delivery and Machine integration.
package production

import (
	"context"
	"testing"
	"time"

	"github.com/comalice/statechartx/internal/core"
	"github.com/comalice/statechartx/internal/primitives"
)

func TestChannelPublisher_Delivery(t *testing.T) {
	ch := make(chan PublishedEvent, 10)
	p := NewChannelPublisher(ch)

	event := primitives.NewEvent("test-event", "data")
	meta := core.MachineMetadata{
		MachineID:  "test-machine",
		Transition: "s1 -> s2",
		Timestamp:  time.Now(),
	}

	ctx := context.Background()
	err := p.Publish(ctx, event, meta)
	if err != nil {
		t.Errorf("Publish failed: %v", err)
	}

	select {
	case got := <-ch:
		if got.Event.Type != event.Type {
			t.Errorf("Event type mismatch: got %q, want %q", got.Event.Type, event.Type)
		}
		if got.Metadata.MachineID != meta.MachineID {
			t.Errorf("Metadata MachineID mismatch: got %q, want %q", got.Metadata.MachineID, meta.MachineID)
		}
		if got.Metadata.Transition != meta.Transition {
			t.Errorf("Metadata Transition mismatch: got %q, want %q", got.Metadata.Transition, meta.Transition)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("No event delivered")
	}
}

func TestChannelPublisher_BackpressureDrop(t *testing.T) {
	ch := make(chan PublishedEvent, 1)
	p := NewChannelPublisher(ch)
	ch <- PublishedEvent{} // Fill buffer

	event := primitives.NewEvent("drop-test", nil)
	meta := core.MachineMetadata{MachineID: "test"}

	ctx := context.Background()
	err := p.Publish(ctx, event, meta)
	if err != nil {
		t.Errorf("Publish on full channel failed: %v", err)
	}
	// Should drop silently
}

func TestChannelPublisher_Close(t *testing.T) {
	ch := make(chan PublishedEvent, 1)
	p := NewChannelPublisher(ch)

	if err := p.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
	// Channel closed successfully
}

func TestChannelPublisher_Integration_PublishMetadata(t *testing.T) {
	publishCh := make(chan PublishedEvent, 10)
	publisher := NewChannelPublisher(publishCh)

	event := primitives.NewEvent("TRANSITION", nil)
	meta := core.MachineMetadata{
		MachineID:  "integration-test",
		Transition: "green -> yellow",
		Timestamp:  time.Now(),
	}

	ctx := context.Background()
	err := publisher.Publish(ctx, event, meta)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-publishCh:
		if got.Metadata.Transition != "green -> yellow" {
			t.Errorf("Metadata transition mismatch: got %q, want %q", got.Metadata.Transition, "green -> yellow")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("No published event received")
	}
}
