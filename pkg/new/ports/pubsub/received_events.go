package pubsub

import (
	"context"
	"log"

	"github.com/piraces/rsslay/pkg/new/adapters/pubsub"
	domain "github.com/piraces/rsslay/pkg/new/domain/nostr"
)

type EventCreatedHandler interface {
	Handle(ctx context.Context, event domain.Event) error
}

type ReceivedEventSubscriber struct {
	pubsub  *pubsub.EventCreatedPubSub
	handler EventCreatedHandler
}

func NewReceivedEventSubscriber(
	pubsub *pubsub.EventCreatedPubSub,
	handler EventCreatedHandler,
) *ReceivedEventSubscriber {
	return &ReceivedEventSubscriber{
		pubsub:  pubsub,
		handler: handler,
	}
}

func (p *ReceivedEventSubscriber) Run(ctx context.Context) {
	for event := range p.pubsub.Subscribe(ctx) {
		if err := p.handler.Handle(ctx, event); err != nil {
			log.Printf("error passing event '%s' to event created handler: %s", event.Libevent().ID, err)
		}
	}
}
