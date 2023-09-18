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

	GetEvents         *HandlerGetEvents
	GetTotalFeedCount *HandlerGetTotalFeedCount
	GetRandomFeeds    *HandlerGetRandomFeeds
	SearchFeeds       *HandlerSearchFeeds
}

type FeedDefinitionStorage interface {
	Put(definition *feeddomain.FeedDefinition) error
	CountTotal() (int, error)
	List() ([]*feeddomain.FeedDefinition, error)
	ListRandom(limit int) ([]*feeddomain.FeedDefinition, error)
	Search(query string, limit int) ([]*feeddomain.FeedDefinition, error)
}

type EventStorage interface {
	GetEvents(filter domain.Filter) ([]domain.Event, error)
	PutEvents(author domain.PublicKey, events []domain.Event) error
}

type ConverterSelector interface {
	Select(feed *gofeed.Feed) feed.ItemToEventConverter
}

type EventPublisher interface {
	PublishNewEventCreated(evt domain.Event)
}
