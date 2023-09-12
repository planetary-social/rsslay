package adapters

import (
	"errors"
	domain "github.com/piraces/rsslay/pkg/new/domain/nostr"
)

type EventStorage struct {
}

func (e EventStorage) GetEvents(filter domain.Filter) ([]domain.Event, error) {
	return nil, errors.New("not implemented")
}
