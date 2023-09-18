package adapters

import (
	"errors"
	"log"
	"sync"

	domain "github.com/piraces/rsslay/pkg/new/domain/nostr"
)

type EventStorage struct {
	events     map[string][]domain.Event
	eventsLock sync.RWMutex
}

func NewEventStorage() *EventStorage {
	return &EventStorage{
		events: make(map[string][]domain.Event),
	}
}

func (e *EventStorage) PutEvents(author domain.PublicKey, events []domain.Event) error {
	e.eventsLock.Lock()
	defer e.eventsLock.Unlock()

	log.Printf("saving %d events for feed %s", len(events), author.Hex())

	for _, event := range events {
		if !author.Equal(event.PublicKey()) {
			return errors.New("one or more events weren't created by this author")
		}
	}

	e.events[author.Hex()] = events
	return nil
}

func (e *EventStorage) GetEvents(filter domain.Filter) ([]domain.Event, error) {
	e.eventsLock.RLock()
	defer e.eventsLock.RUnlock()

	// todo optimize
	var results []domain.Event

	// todo optimize
	for _, events := range e.events {
		for _, event := range events {
			if filter.Matches(event) {
				results = append(results, event)
			}
		}
	}
	return results, nil
}
