package app

import (
	domain "github.com/piraces/rsslay/pkg/new/domain/nostr"
)

type HandlerGetEvents struct {
	eventStorage EventStorage
}

func NewHandlerGetEvents(eventStorage EventStorage) *HandlerGetEvents {
	return &HandlerGetEvents{eventStorage: eventStorage}
}

func (h *HandlerGetEvents) Handle(domainfilter domain.Filter) ([]domain.Event, error) {
	return h.eventStorage.GetEvents(domainfilter)
}
