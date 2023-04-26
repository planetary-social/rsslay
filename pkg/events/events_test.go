package events

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/piraces/rsslay/pkg/feed"
	"github.com/stretchr/testify/assert"
)

const samplePubKey = "73e247ee8c4ff09a50525bed7b0869c371864c0bf2b4d6a2639acaed07613958"
const samplePrivateKey = "4d0888c07093941c9db16fcffb96fdf8af49a6839e865ea6110c7ab7cbd2d3d3"
const sampleValidDirectFeedUrl = "https://mastodon.social/@Gargron.rss"
const sampleValidNitterFeedUrl = "https://nitter.moomoo.me/Twitter/rss"
const sampleInvalidNitterFeedUrl = "https://example.com/Twitter/rss"
const sampleValidUrl = "https://mastodon.social/"

func TestGetParsedFeedForPubKey(t *testing.T) {
	testCases := []struct {
		pubKey                    string
		expectedReturnUrl         string
		expectedNitterFeed        bool
		expectedUpdatedNitterFeed bool
		expectedSqlError          bool
		expectedSqlRow            bool
		expectedInvalidUrl        bool
		expectedInvalidFeed       bool
	}{
		{
			pubKey:                    samplePubKey,
			expectedReturnUrl:         sampleValidDirectFeedUrl,
			expectedNitterFeed:        false,
			expectedUpdatedNitterFeed: false,
			expectedSqlError:          false,
			expectedSqlRow:            true,
			expectedInvalidUrl:        false,
			expectedInvalidFeed:       false,
		},
		{
			pubKey:                    samplePubKey,
			expectedReturnUrl:         sampleValidNitterFeedUrl,
			expectedNitterFeed:        true,
			expectedUpdatedNitterFeed: true,
			expectedSqlError:          false,
			expectedSqlRow:            true,
			expectedInvalidUrl:        false,
			expectedInvalidFeed:       false,
		},
		{
			pubKey:                    samplePubKey,
			expectedReturnUrl:         sampleValidNitterFeedUrl,
			expectedNitterFeed:        false,
			expectedUpdatedNitterFeed: true,
			expectedSqlError:          false,
			expectedSqlRow:            true,
			expectedInvalidUrl:        false,
			expectedInvalidFeed:       false,
		},
		{
			pubKey:                    samplePubKey,
			expectedReturnUrl:         sampleInvalidNitterFeedUrl,
			expectedNitterFeed:        true,
			expectedUpdatedNitterFeed: true,
			expectedSqlError:          false,
			expectedSqlRow:            true,
			expectedInvalidUrl:        false,
			expectedInvalidFeed:       false,
		},
		{
			pubKey:                    samplePubKey,
			expectedReturnUrl:         "",
			expectedNitterFeed:        false,
			expectedUpdatedNitterFeed: false,
			expectedSqlError:          false,
			expectedSqlRow:            false,
			expectedInvalidUrl:        false,
			expectedInvalidFeed:       false,
		},
		{
			pubKey:                    samplePubKey,
			expectedReturnUrl:         "",
			expectedNitterFeed:        false,
			expectedUpdatedNitterFeed: false,
			expectedSqlError:          true,
			expectedSqlRow:            false,
			expectedInvalidUrl:        false,
			expectedInvalidFeed:       false,
		},
		{
			pubKey:                    samplePubKey,
			expectedReturnUrl:         "invalid",
			expectedNitterFeed:        false,
			expectedUpdatedNitterFeed: false,
			expectedSqlError:          false,
			expectedSqlRow:            true,
			expectedInvalidUrl:        true,
			expectedInvalidFeed:       false,
		},
		{
			pubKey:                    samplePubKey,
			expectedReturnUrl:         sampleValidUrl,
			expectedNitterFeed:        false,
			expectedUpdatedNitterFeed: false,
			expectedSqlError:          false,
			expectedSqlRow:            true,
			expectedInvalidUrl:        false,
			expectedInvalidFeed:       true,
		},
	}
	for _, tc := range testCases {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		rows := sqlmock.NewRows([]string{"privatekey", "url", "nitter"})
		if tc.expectedSqlRow {
			rows.AddRow(samplePrivateKey, tc.expectedReturnUrl, tc.expectedNitterFeed)
		}

		if tc.expectedSqlError {
			mock.ExpectQuery("SELECT privatekey, url, nitter FROM feeds").WillReturnError(errors.New("error"))
		} else {
			mock.ExpectQuery("SELECT privatekey, url, nitter FROM feeds").WillReturnRows(rows)
		}
		mock.ExpectClose()

		parsedFeed, entity := GetParsedFeedForPubKey(tc.pubKey, db, true, []string{"birdsite.xanny.family", "notabird.site", "nitter.moomoo.me", "nitter.fly.dev"})
		if tc.expectedSqlError {
			assert.Nil(t, parsedFeed)
			assert.Empty(t, entity)
		} else if !tc.expectedSqlRow {
			assert.Nil(t, parsedFeed)
			assert.Empty(t, entity)
		} else if tc.expectedInvalidUrl {
			assert.Nil(t, parsedFeed)
			assert.Equal(t, feed.Entity{
				PublicKey:  "",
				PrivateKey: samplePrivateKey,
				URL:        tc.expectedReturnUrl,
				Nitter:     tc.expectedUpdatedNitterFeed,
			}, entity)
		} else if tc.expectedInvalidFeed {
			assert.Nil(t, parsedFeed)
			assert.Equal(t, feed.Entity{
				PublicKey:  "",
				PrivateKey: samplePrivateKey,
				URL:        tc.expectedReturnUrl,
				Nitter:     tc.expectedUpdatedNitterFeed,
			}, entity)
		} else {
			assert.NotNil(t, parsedFeed)
			assert.Equal(t, feed.Entity{
				PublicKey:  "",
				PrivateKey: samplePrivateKey,
				URL:        tc.expectedReturnUrl,
				Nitter:     tc.expectedUpdatedNitterFeed,
			}, entity)
		}
		_ = db.Close()
	}
}
