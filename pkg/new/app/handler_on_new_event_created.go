package app

import (
	"context"
	"time"

	"github.com/nbd-wtf/go-nostr"
	domain "github.com/piraces/rsslay/pkg/new/domain/nostr"
)

type HandlerOnNewEventCreated struct {
	updatesCh     chan<- nostr.Event
	lastEventTime map[string]time.Time
}

func NewHandlerOnNewEventCreated(updatesCh chan<- nostr.Event) *HandlerOnNewEventCreated {
	return &HandlerOnNewEventCreated{
		updatesCh:     updatesCh,
		lastEventTime: make(map[string]time.Time),
	}
}

func (h *HandlerOnNewEventCreated) Handle(ctx context.Context, event domain.Event) error {
	key := event.PublicKey().Hex()
	if last, ok := h.lastEventTime[key]; !ok || last.Before(event.CreatedAt()) {
		h.lastEventTime[key] = event.CreatedAt()
		h.updatesCh <- event.Libevent()
	}
	return nil
}
