package nostr_test

import (
	"github.com/piraces/rsslay/pkg/new/domain/nostr"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNip19DoesNotPanic(t *testing.T) {
	publicKey, err := nostr.NewPublicKeyFromHex("6ce3fe33ca1d1c4ab7de95ddf2dcceea7d328ce9c0ff14f5209e10f2db248a6d")
	require.NoError(t, err)

	nip19 := publicKey.Nip19()
	require.Equal(t, "npub1dn3luv72r5wy4d77jhwl9hxwaf7n9r8fcrl3fafqncg09key3fksk92ep4", nip19)
}
