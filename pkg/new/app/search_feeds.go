package app

import (
	domainfeed "github.com/piraces/rsslay/pkg/new/domain/feed"
	"github.com/pkg/errors"
)

type HandlerSearchFeeds struct {
	feedDefinitionStorage FeedDefinitionStorage
}

func NewHandlerSearchFeeds(feedDefinitionStorage FeedDefinitionStorage) *HandlerSearchFeeds {
	return &HandlerSearchFeeds{
		feedDefinitionStorage: feedDefinitionStorage,
	}
}

func (h *HandlerSearchFeeds) Handle(query string, limit int) ([]*domainfeed.FeedDefinition, error) {
	if limit <= 0 {
		return nil, errors.New("limit must be positive")
	}
	return h.feedDefinitionStorage.Search(query, limit)
}
