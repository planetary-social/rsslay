package nostr

import (
	"encoding/hex"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/pkg/errors"
)

type Filter struct {
	filter nostr.Filter
}

type Event struct {
	event nostr.Event
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
