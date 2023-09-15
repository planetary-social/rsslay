package pubsub

import (
	"context"

	"github.com/piraces/rsslay/pkg/new/domain/nostr"
)

type EventCreatedPubSub struct {
	pubsub *GoChannelPubSub[nostr.Event]
}

func NewReceivedEventPubSub() *EventCreatedPubSub {
	return &EventCreatedPubSub{
		pubsub: NewGoChannelPubSub[nostr.Event](),
	}
}

func (m *EventCreatedPubSub) PublishNewEventCreated(evt nostr.Event) {
	m.pubsub.Publish(evt)
}

func (m *EventCreatedPubSub) Subscribe(ctx context.Context) <-chan nostr.Event {
	return m.pubsub.Subscribe(ctx)
}
