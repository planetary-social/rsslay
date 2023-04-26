package feed

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/assert"
)

const samplePubKey = "1870bcd5f6081ef7ea4b17204ffa4e92de51670142be0c8140e0635b355ca85f"
const sampleUrlForPublicKey = "https://nitter.moomoo.me/Bitcoin/rss"
const samplePrivateKeyForPubKey = "27660ab89e69f59bb8d9f0bd60da4a8515cdd3e2ca4f91d72a242b086d6aaaa7"
const testSecret = "test"

const sampleInvalidUrl = "https:// nostr.example/"
const sampleInvalidUrlContentType = "https://accounts.google.com/.well-known/openid-configuration"
const sampleRedirectingUrl = "https://httpstat.us/301"
const sampleValidDirectFeedUrl = "https://mastodon.social/@Gargron.rss"
const sampleValidIndirectFeedUrl = "https://www.rssboard.org/"
const sampleValidIndirectFeedUrlExpected = "http://feeds.rssboard.org/rssboard"
const sampleValidWithoutFeedUrl = "https://go.dev/"
const sampleValidWithRelativeFeedUrl = "https://golangweekly.com/"
const sampleValidWithRelativeFeedUrlExpected = "https://golangweekly.com/rss"

var actualTime = time.Unix(time.Now().Unix(), 0)

func TestGetFeedURLWithInvalidURLReturnsEmptyString(t *testing.T) {
	feed := GetFeedURL(sampleInvalidUrl)
	assert.Empty(t, feed)
}

func TestGetFeedURLWithInvalidContentTypeReturnsEmptyString(t *testing.T) {
	feed := GetFeedURL(sampleInvalidUrlContentType)
	assert.Empty(t, feed)
}

func TestGetFeedURLWithRedirectingURLReturnsEmptyString(t *testing.T) {
	feed := GetFeedURL(sampleRedirectingUrl)
	assert.Empty(t, feed)
}

func TestGetFeedURLWithValidUrlOfValidTypesReturnsSameUrl(t *testing.T) {
	feed := GetFeedURL(sampleValidDirectFeedUrl)
	assert.Equal(t, sampleValidDirectFeedUrl, feed)
}

func TestGetFeedURLWithValidUrlOfHtmlTypeWithFeedReturnsFoundFeed(t *testing.T) {
	feed := GetFeedURL(sampleValidIndirectFeedUrl)
	assert.Equal(t, sampleValidIndirectFeedUrlExpected, feed)
}

func TestGetFeedURLWithValidUrlOfHtmlTypeWithRelativeFeedReturnsFoundFeed(t *testing.T) {
	feed := GetFeedURL(sampleValidWithRelativeFeedUrl)
	assert.Equal(t, sampleValidWithRelativeFeedUrlExpected, feed)
}

func TestGetFeedURLWithValidUrlOfHtmlTypeWithoutFeedReturnsEmpty(t *testing.T) {
	feed := GetFeedURL(sampleValidWithoutFeedUrl)
	assert.Empty(t, feed)
}

func TestParseFeedWithValidUrlReturnsParsedFeed(t *testing.T) {
	feed, err := ParseFeed(sampleValidWithRelativeFeedUrlExpected)
	assert.NotNil(t, feed)
	assert.NoError(t, err)
}

func TestParseFeedWithValidUrlWithoutFeedReturnsError(t *testing.T) {
	feed, err := ParseFeed(sampleValidWithoutFeedUrl)
	assert.Nil(t, feed)
	assert.Error(t, err)
}

func TestParseFeedWithCachedUrlReturnsCachedParsedFeed(t *testing.T) {
	_, _ = ParseFeed(sampleValidWithRelativeFeedUrlExpected)
	feed, err := ParseFeed(sampleValidWithRelativeFeedUrlExpected)
	assert.NotNil(t, feed)
	assert.NoError(t, err)
}

func TestEntryFeedToSetMetadata(t *testing.T) {
	testCases := []struct {
		pubKey                   string
		feed                     *gofeed.Feed
		originalUrl              string
		enableAutoRegistration   bool
		defaultProfilePictureUrl string
	}{
		{
			pubKey:                   samplePubKey,
			feed:                     &sampleNitterFeed,
			originalUrl:              sampleNitterFeed.FeedLink,
			enableAutoRegistration:   true,
			defaultProfilePictureUrl: "https://image.example",
		},
		{
			pubKey:                   samplePubKey,
			feed:                     &sampleDefaultFeed,
			originalUrl:              sampleDefaultFeed.FeedLink,
			enableAutoRegistration:   true,
			defaultProfilePictureUrl: "https://image.example",
		},
	}
	for _, tc := range testCases {
		metadata := EntryFeedToSetMetadata(tc.pubKey, tc.feed, tc.originalUrl, tc.enableAutoRegistration, tc.defaultProfilePictureUrl)
		assert.NotEmpty(t, metadata)
		assert.Equal(t, samplePubKey, metadata.PubKey)
		assert.Equal(t, 0, metadata.Kind)
		assert.Empty(t, metadata.Sig)
	}
}

func TestPrivateKeyFromFeed(t *testing.T) {
	sk := PrivateKeyFromFeed(sampleUrlForPublicKey, testSecret)
	assert.Equal(t, samplePrivateKeyForPubKey, sk)
}

func TestDeleteExistingInvalidFeed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when closing a stub database connection", err)
		}
	}(db)

	mock.ExpectExec("DELETE FROM feeds").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectClose()
	DeleteInvalidFeed(sampleUrlForPublicKey, db)
}

func TestDeleteNonExistingInvalidFeed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when closing a stub database connection", err)
		}
	}(db)

	mock.ExpectExec("DELETE FROM feeds").WillReturnError(errors.New(""))
	mock.ExpectClose()
	DeleteInvalidFeed(sampleUrlForPublicKey, db)
}
