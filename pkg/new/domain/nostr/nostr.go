package nostr

import (
	"bytes"
	"encoding/hex"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/pkg/errors"
)

type Filter struct {
	filter *nostr.Filter
}

func NewFilter(filter *nostr.Filter) Filter {
	return Filter{
		filter: filter,
	}
}
func (f Filter) Libfilter() *nostr.Filter {

	return f.filter
}

func (f Filter) Matches(event Event) bool {
	libevent := event.Libevent()
	return f.Libfilter().Matches(&libevent)
}

type Event struct {
	publicKey PublicKey
	event     nostr.Event
}

func NewEvent(event nostr.Event) (Event, error) {
	publicKey, err := NewPublicKeyFromHex(event.PubKey)
	if err != nil {
		return Event{}, errors.Wrap(err, "error parsing the public key")
	}
	return Event{
		publicKey: publicKey,
		event:     event,
	}, nil
}

func (e Event) Libevent() nostr.Event {
	return e.event
}

func (e Event) PublicKey() PublicKey {
	return e.publicKey
}

type PublicKey struct {
	b []byte
}

func NewPublicKeyFromHex(s string) (PublicKey, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return PublicKey{}, errors.Wrap(err, "error decoding hex string")
	}

	// todo len check

	return PublicKey{b: b}, nil
}

func (p PublicKey) Hex() string {
	return hex.EncodeToString(p.b)
}

func (p PublicKey) Nip19() string {
	nip19, err := nip19.EncodePublicKey(p.Hex())
	if err != nil {
		panic(err) // will either always fail or never fail so tests are enough
	}
	return nip19
}

func (p PublicKey) Equal(o PublicKey) bool {
	return bytes.Equal(p.b, o.b)
}

type PrivateKey struct {
	b []byte
}

func NewPrivateKeyFromHex(s string) (PrivateKey, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return PrivateKey{}, errors.Wrap(err, "error decoding hex string")
	}

	// todo len check

	return PrivateKey{b: b}, nil
}

func (k PrivateKey) Hex() any {
	return hex.EncodeToString(k.b)
}
