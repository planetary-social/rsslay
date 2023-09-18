package app

type HandlerGetTotalFeedCount struct {
	feedDefinitionStorage FeedDefinitionStorage
}

func NewHandlerGetTotalFeedCount(feedDefinitionStorage FeedDefinitionStorage) *HandlerGetTotalFeedCount {
	return &HandlerGetTotalFeedCount{
		feedDefinitionStorage: feedDefinitionStorage,
	}
}

func (h *HandlerGetTotalFeedCount) Handle() (int, error) {
	return h.feedDefinitionStorage.CountTotal()
}
