package app

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/mmcdole/gofeed"
	"github.com/nbd-wtf/go-nostr"
	"github.com/piraces/rsslay/pkg/events"
	"github.com/piraces/rsslay/pkg/feed"
	domainfeed "github.com/piraces/rsslay/pkg/new/domain/feed"
	domain "github.com/piraces/rsslay/pkg/new/domain/nostr"
	"github.com/pkg/errors"
)

const numWorkers = 10

type HandlerUpdateFeeds struct {
	deleteFailingFeeds          bool
	nitterInstances             []string
	enableAutoNIP05Registration bool
	defaultProfilePictureUrl    string
	mainDomainName              string

	db                    *sql.DB // todo remove!
	feedDefinitionStorage FeedDefinitionStorage
	converterSelector     ConverterSelector
	eventStorage          EventStorage
	eventPublisher        EventPublisher
}

func NewHandlerUpdateFeeds(
	deleteFailingFeeds bool,
	nitterInstances []string,
	enableAutoNIP05Registration bool,
	defaultProfilePictureUrl string,
	mainDomainName string,
	db *sql.DB,
	feedDefinitionStorage FeedDefinitionStorage,
	converterSelector ConverterSelector,
	eventStorage EventStorage,
	eventPublisher EventPublisher,
) *HandlerUpdateFeeds {
	return &HandlerUpdateFeeds{
		deleteFailingFeeds:          deleteFailingFeeds,
		nitterInstances:             nitterInstances,
		enableAutoNIP05Registration: enableAutoNIP05Registration,
		defaultProfilePictureUrl:    defaultProfilePictureUrl,
		mainDomainName:              mainDomainName,
		db:                          db,
		feedDefinitionStorage:       feedDefinitionStorage,
		converterSelector:           converterSelector,
		eventStorage:                eventStorage,
		eventPublisher:              eventPublisher,
	}
}

func (h *HandlerUpdateFeeds) Handle(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	definitions, err := h.feedDefinitionStorage.List()
	if err != nil {
		return errors.Wrap(err, "error getting feed definitions")
	}

	chIn := make(chan *domainfeed.FeedDefinition)
	chOut := make(chan definitionWithError)

	go func() {
		for _, definition := range definitions {
			definition := definition
			select {
			case chIn <- definition:
				continue
			case <-ctx.Done():
				return
			}
		}
	}()

	h.startWorkers(ctx, chIn, chOut)

	counterSuccess := 0
	counterError := 0

	var resultErr error
	for i := 0; i < len(definitions); i++ {
		select {
		case definitionWithError := <-chOut:
			if err := definitionWithError.Err; err != nil {
				resultErr = multierror.Append(resultErr, err)
				counterError++
			} else {
				counterSuccess++
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	log.Printf("updating feeds result success=%d error=%d", counterSuccess, counterError)

	return resultErr
}

func (h *HandlerUpdateFeeds) startWorkers(ctx context.Context, chIn chan *domainfeed.FeedDefinition, chOut chan definitionWithError) {
	for i := 0; i < numWorkers; i++ {
		go h.startWorker(ctx, chIn, chOut)
	}
}

func (h *HandlerUpdateFeeds) startWorker(ctx context.Context, chIn chan *domainfeed.FeedDefinition, chOut chan definitionWithError) {
	for {
		select {
		case definition := <-chIn:
			err := h.updateFeed(ctx, definition)
			select {
			case chOut <- definitionWithError{
				Definition: definition,
				Err:        err,
			}:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// todo restore the capability to replay events?
func (h *HandlerUpdateFeeds) updateFeed(ctx context.Context, definition *domainfeed.FeedDefinition) error {
	log.Printf("updating feed %s", definition.PublicKey().Hex())

	events, err := h.getFeedEvents(definition)
	if err != nil {
		return errors.Wrapf(err, "error getting events for feed '%s'", definition.PublicKey().Hex())
	}

	if err := h.eventStorage.PutEvents(definition.PublicKey(), events); err != nil {
		return errors.Wrap(err, "error saving events")
	}

	for _, event := range events {
		h.eventPublisher.PublishNewEventCreated(event)
	}

	return nil
}

func (h *HandlerUpdateFeeds) getFeedEvents(definition *domainfeed.FeedDefinition) ([]domain.Event, error) {
	parsedFeed, entity := events.GetParsedFeedForPubKey(
		definition.PublicKey().Hex(),
		h.db,
		h.deleteFailingFeeds,
		h.nitterInstances,
	)
	if parsedFeed == nil {
		return nil, nil // todo why is this not an error?
	}

	var events []domain.Event

	metadataEvent, err := h.makeMetadataEvent(definition, parsedFeed, entity)
	if err != nil {
		return nil, errors.Wrap(err, "error creating the metadata event")
	}
	events = append(events, metadataEvent)

	converter := h.converterSelector.Select(parsedFeed)

	for _, item := range parsedFeed.Items {
		defaultCreatedAt := time.Unix(time.Now().Unix(), 0)
		evt := converter.Convert(definition.PublicKey().Hex(), item, parsedFeed, defaultCreatedAt, entity.URL)

		// Feed need to have a date for each entry...
		if evt.CreatedAt == nostr.Timestamp(defaultCreatedAt.Unix()) {
			continue
		}

		if err = evt.Sign(entity.PrivateKey); err != nil {
			return nil, errors.Wrap(err, "error signing the event")
		}

		domainEvent, err := domain.NewEvent(evt)
		if err != nil {
			return nil, errors.Wrap(err, "error creating a domain event")
		}

		events = append(events, domainEvent)
	}

	return events, nil
}

func (h *HandlerUpdateFeeds) makeMetadataEvent(definition *domainfeed.FeedDefinition, parsedFeed *gofeed.Feed, entity feed.Entity) (domain.Event, error) {
	evt := feed.EntryFeedToSetMetadata(definition.PublicKey().Hex(), parsedFeed, entity.URL, h.enableAutoNIP05Registration, h.defaultProfilePictureUrl, h.mainDomainName)
	if err := evt.Sign(entity.PrivateKey); err != nil {
		return domain.Event{}, errors.Wrap(err, "error signing the event")
	}
	domainMetadataEvent, err := domain.NewEvent(evt)
	if err != nil {
		return domain.Event{}, errors.Wrap(err, "error creating a domain event")
	}
	return domainMetadataEvent, nil
}

type definitionWithError struct {
	Definition *domainfeed.FeedDefinition
	Err        error
}
