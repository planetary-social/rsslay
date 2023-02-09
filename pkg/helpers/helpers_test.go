package helpers

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

const sampleInvalidUrl = "https:// nostr.example/"
const sampleValidUrl = "https://nostr.example"

func TestJoinWithInvalidUrlReturnsNil(t *testing.T) {
	join, err := UrlJoin(sampleInvalidUrl)
	assert.Equal(t, join, "")
	assert.ErrorContains(t, err, "invalid character")
}

func TestJoinWithValidUrlAndNoExtraElementsReturnsBaseUrl(t *testing.T) {
	join, err := UrlJoin(sampleValidUrl)
	assert.Equal(t, sampleValidUrl, join)
	assert.NoError(t, err)
}

func TestJoinWithValidUrlAndExtraElementsReturnsValidUrl(t *testing.T) {
	join, err := UrlJoin(sampleValidUrl, "rss")
	expectedJoinResult := fmt.Sprintf("%s/%s", sampleValidUrl, "rss")
	assert.Equal(t, expectedJoinResult, join)
	assert.NoError(t, err)
}

func TestIsValidUrl(t *testing.T) {
	testCases := []struct {
		rawUrl        string
		expectedValid bool
	}{
		{
			rawUrl:        "hi/there?",
			expectedValid: false,
		},
		{
			rawUrl:        "http://golang.cafe/",
			expectedValid: true,
		},
		{
			rawUrl:        "http://golang.org/index.html?#page1",
			expectedValid: true,
		},
		{
			rawUrl:        "golang.org",
			expectedValid: false,
		},
		{
			rawUrl:        "https://golang.cafe/",
			expectedValid: true,
		},
		{
			rawUrl:        "wss://nostr.moe",
			expectedValid: false,
		},
		{
			rawUrl:        "ftp://nostr.moe",
			expectedValid: false,
		},
	}
	for _, tc := range testCases {
		isValid := IsValidHttpUrl(tc.rawUrl)
		if tc.expectedValid {
			assert.True(t, isValid)
		} else {
			assert.False(t, isValid)
		}
	}
}
