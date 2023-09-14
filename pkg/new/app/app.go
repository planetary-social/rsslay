package app

import (
	"github.com/mmcdole/gofeed"
	"github.com/piraces/rsslay/pkg/feed"
	feeddomain "github.com/piraces/rsslay/pkg/new/domain/feed"
	domain "github.com/piraces/rsslay/pkg/new/domain/nostr"
)

type App struct {
	CreateFeedDefinition *HandlerCreateFeedDefinition
	UpdateFeeds          *HandlerUpdateFeeds
	GetEvents            *HandlerGetEvents
}

type FeedDefinitionStorage interface {
	Put(definition *feeddomain.FeedDefinition) error
	List() ([]*feeddomain.FeedDefinition, error)
}

type EventStorage interface {
	GetEvents(filter domain.Filter) ([]domain.Event, error)
	PutEvents(author domain.PublicKey, events []domain.Event) error
}

type ConverterSelector interface {
	Select(feed *gofeed.Feed) feed.ItemToEventConverter
}
