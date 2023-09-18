package app

import (
	domainfeed "github.com/piraces/rsslay/pkg/new/domain/feed"
	"github.com/pkg/errors"
)

type HandlerGetRandomFeeds struct {
	feedDefinitionStorage FeedDefinitionStorage
}

func NewHandlerGetRandomFeeds(feedDefinitionStorage FeedDefinitionStorage) *HandlerGetRandomFeeds {
	return &HandlerGetRandomFeeds{
		feedDefinitionStorage: feedDefinitionStorage,
	}
}

func (h *HandlerGetRandomFeeds) Handle(limit int) ([]*domainfeed.FeedDefinition, error) {
	if limit <= 0 {
		return nil, errors.New("limit must be positive")
	}
	return h.feedDefinitionStorage.ListRandom(limit)
}
