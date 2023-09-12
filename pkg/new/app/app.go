package app

import feeddomain "github.com/piraces/rsslay/pkg/new/domain/feed"

type App struct {
	CreateFeedDefinition *HandlerCreateFeedDefinition
	UpdateFeeds          *HandlerUpdateFeeds
	GetEvents            *HandlerGetEvents
}

type FeedDefinitionStorage interface {
	Put(definition *feeddomain.FeedDefinition) error
	List() ([]*feeddomain.FeedDefinition, error)
}
