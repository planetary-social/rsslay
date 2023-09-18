package feed

import (
	"errors"

	"github.com/piraces/rsslay/pkg/helpers"
	"github.com/piraces/rsslay/pkg/new/domain/nostr"
)

type FeedDefinition struct {
	publicKey  nostr.PublicKey
	privateKey nostr.PrivateKey
	address    Address
	nitter     bool
}

func NewFeedDefinition(publicKey nostr.PublicKey, privateKey nostr.PrivateKey, address Address, nitter bool) (*FeedDefinition, error) {
	if !publicKey.Matches(privateKey) {
		return nil, errors.New("public/private key mismatch")
	}

	return &FeedDefinition{publicKey: publicKey, privateKey: privateKey, address: address, nitter: nitter}, nil
}

func (f FeedDefinition) PublicKey() nostr.PublicKey {
	return f.publicKey
}

func (f FeedDefinition) PrivateKey() nostr.PrivateKey {
	return f.privateKey
}

func (f FeedDefinition) Address() Address {
	return f.address
}

func (f FeedDefinition) Nitter() bool {
	return f.nitter
}

type Address struct {
	s string
}

func NewAddress(s string) (Address, error) {
	if s == "" {
		return Address{}, errors.New("address can't me an empty string")
	}

	if !helpers.IsValidHttpUrl(s) {
		return Address{}, errors.New("invalid URL provided (must be in absolute format and with https or https scheme)")
	}

	return Address{s: s}, nil
}

func (a Address) String() string {
	return a.s
}
