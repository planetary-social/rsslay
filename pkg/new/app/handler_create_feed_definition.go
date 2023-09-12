package app

import (
	"github.com/nbd-wtf/go-nostr"
	"github.com/piraces/rsslay/pkg/feed"
	"github.com/piraces/rsslay/pkg/new/domain"
	feeddomain "github.com/piraces/rsslay/pkg/new/domain/feed"
	nostrdomain "github.com/piraces/rsslay/pkg/new/domain/nostr"
	"github.com/pkg/errors"
	"strings"
)

type FeedDefinitionStorage interface {
	Put(definition *feeddomain.FeedDefinition) error
}

type HandlerCreateFeedDefinition struct {
	secret                domain.Secret
	feedDefinitionStorage FeedDefinitionStorage
}

func NewHandlerCreateFeedDefinition(secret domain.Secret, feedDefinitionStorage FeedDefinitionStorage) *HandlerCreateFeedDefinition {
	return &HandlerCreateFeedDefinition{secret: secret, feedDefinitionStorage: feedDefinitionStorage}
}

func (h *HandlerCreateFeedDefinition) Handle(address feeddomain.Address) (*feeddomain.FeedDefinition, error) {
	feedUrl := feed.GetFeedURL(address.String())
	if feedUrl == "" {
		return nil, errors.New("could not find a feed URL in there")
	}

	parsedFeed, err := feed.ParseFeed(feedUrl)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing feed")
	}

	sk := feed.PrivateKeyFromFeed(feedUrl, h.secret.String())
	publicKey, err := nostr.GetPublicKey(sk)
	if err != nil {
		return nil, errors.Wrap(err, "error creating a public key")
	}

	publicKey = strings.TrimSpace(publicKey)
	isNitterFeed := strings.Contains(parsedFeed.Description, "Twitter feed") // todo this decision should occur at domain level

	// this function still calls other functions which do not use appropriate
	// domain types therefore we need to convert the return values to domain
	// types here

	domainPublicKey, err := nostrdomain.NewPublicKeyFromHex(publicKey)
	if err != nil {
		return nil, errors.Wrap(err, "error creating address from feed url")
	}

	domainPrivateKey, err := nostrdomain.NewPrivateKeyFromHex(publicKey)
	if err != nil {
		return nil, errors.Wrap(err, "error creating address from feed url")
	}

	domainFeedUrl, err := feeddomain.NewAddress(feedUrl)
	if err != nil {
		return nil, errors.Wrap(err, "error creating address from feed url")
	}

	definition := feeddomain.NewFeedDefinition(
		domainPublicKey,
		domainPrivateKey,
		domainFeedUrl,
		isNitterFeed,
	)

	if err := h.feedDefinitionStorage.Put(definition); err != nil {
		return nil, errors.Wrap(err, "error saving the feed definition")
	}

	return definition, nil
}
