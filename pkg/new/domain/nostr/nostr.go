package nostr

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/pkg/errors"
)

const (
	publicKeyBytesLen  = 32
	privateKeyBytesLen = btcec.PrivKeyBytesLen
	idBytesLen         = sha256.Size
)

type Filter struct {
	filter *nostr.Filter
}

func NewFilter(filter *nostr.Filter) Filter {
	return Filter{
		filter: filter,
	}
}

func (f Filter) Matches(event Event) bool {
	return f.filter.Matches(&event.event)
}

type Event struct {
	id        ID
	publicKey PublicKey
	event     nostr.Event
}

func NewEvent(event nostr.Event) (Event, error) {
	publicKey, err := NewPublicKeyFromHex(event.PubKey)
	if err != nil {
		return Event{}, errors.Wrap(err, "error parsing the public key")
	}

	id, err := NewIDFromHex(event.ID)
	if err != nil {
		return Event{}, errors.Wrap(err, "error parsing the id")
	}

	return Event{
		id:        id,
		publicKey: publicKey,
		event:     event,
	}, nil
}

func (e Event) ID() ID {
	return e.id
}

func (e Event) PublicKey() PublicKey {
	return e.publicKey
}

func (e Event) CreatedAt() time.Time {
	return e.event.CreatedAt.Time()
}

func (e Event) Libevent() nostr.Event {
	return e.event
}

type ID struct {
	b []byte
}

func NewIDFromHex(s string) (ID, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return ID{}, errors.Wrap(err, "error decoding hex string")
	}

	if l := len(b); l != idBytesLen {
		return ID{}, fmt.Errorf("invalid event id length '%d'", l)
	}

	return ID{b: b}, nil
}

func (id ID) Hex() string {
	return hex.EncodeToString(id.b)
}

type PublicKey struct {
	b []byte
}

func NewPublicKeyFromHex(s string) (PublicKey, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return PublicKey{}, errors.Wrap(err, "error decoding hex string")
	}

	if l := len(b); l != publicKeyBytesLen {
		return PublicKey{}, fmt.Errorf("invalid public key length '%d'", l)
	}

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

func (p PublicKey) Matches(key PrivateKey) bool {
	_, publicKey := btcec.PrivKeyFromBytes(key.b)
	hexPublicKey := hex.EncodeToString(schnorr.SerializePubKey(publicKey))
	return p.Hex() == hexPublicKey
}

type PrivateKey struct {
	b []byte
}

func NewPrivateKeyFromHex(s string) (PrivateKey, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return PrivateKey{}, errors.Wrap(err, "error decoding hex string")
	}

	if l := len(b); l != privateKeyBytesLen {
		return PrivateKey{}, fmt.Errorf("invalid private key length '%d'", l)
	}

	return PrivateKey{b: b}, nil
}

func (k PrivateKey) Hex() any {
	return hex.EncodeToString(k.b)
}
