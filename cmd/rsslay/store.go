package main

import (
	"github.com/nbd-wtf/go-nostr"
	"github.com/piraces/rsslay/pkg/metrics"
	"github.com/piraces/rsslay/pkg/new/app"
	nostrdomain "github.com/piraces/rsslay/pkg/new/domain/nostr"
	"github.com/pkg/errors"
)

type store struct {
	app app.App
}

func newStore(app app.App) *store {
	return &store{app: app}
}

func (b store) Init() error {
	return nil
}

func (b store) SaveEvent(_ *nostr.Event) error {
	metrics.InvalidEventsRequests.Inc()
	return errors.New("blocked: we don't accept any events")
}

func (b store) DeleteEvent(_, _ string) error {
	metrics.InvalidEventsRequests.Inc()
	return errors.New("blocked: we can't delete any events")
}

func (b store) QueryEvents(libfilter *nostr.Filter) ([]nostr.Event, error) {
	metrics.QueryEventsRequests.Inc()

	filter := nostrdomain.NewFilter(libfilter)
	events, err := b.app.GetEvents.Handle(filter)
	if err != nil {
		return nil, errors.Wrap(err, "error getting events")
	}

	return b.toEvents(events), nil
}

func (b store) toEvents(events []nostrdomain.Event) []nostr.Event {
	var result []nostr.Event
	for _, event := range events {
		result = append(result, event.Libevent())
	}
	return result
}
