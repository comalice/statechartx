package production

import (
	"context"

	"github.com/comalice/statechartx/internal/core"
	"github.com/comalice/statechartx/internal/primitives"
)

// PublishedEvent bundles an event with its machine metadata for publishing.
type PublishedEvent struct {
	Event    primitives.Event
	Metadata core.MachineMetadata
}

// ChannelPublisher is a stdlib-only implementation that forwards events to a Go channel.
// Non-blocking publish with drop on backpressure.
type ChannelPublisher struct {
	ch chan<- PublishedEvent
}

// NewChannelPublisher creates a ChannelPublisher with the given output channel.
func NewChannelPublisher(ch chan<- PublishedEvent) *ChannelPublisher {
	return &ChannelPublisher{ch: ch}
}

func (p *ChannelPublisher) Publish(ctx context.Context, event primitives.Event, metadata core.MachineMetadata) error {
	select {
	case p.ch <- PublishedEvent{Event: event, Metadata: metadata}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil // Non-blocking drop
	}
}

func (p *ChannelPublisher) Close() error {
	close(p.ch)
	return nil
}
