package ports

import (
	"context"
	"log"
	"time"
)

type HandlerUpdateFeeds interface {
	Handle() error
}

type UpdateFeedsTimer struct {
	handler HandlerUpdateFeeds
}

func NewUpdateFeedsTimer(handler HandlerUpdateFeeds) *UpdateFeedsTimer {
	return &UpdateFeedsTimer{handler: handler}
}

func (h *UpdateFeedsTimer) Run(ctx context.Context) {
	for {
		if err := h.handler.Handle(); err != nil {
			log.Printf("error updating feeds %s", err)
		}

		select {
		case <-time.After(30 * time.Minute):
			continue
		case <-ctx.Done():
			return
		}
	}
}
