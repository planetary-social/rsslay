package events

import (
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/piraces/rsslay/pkg/feed"
	"github.com/stretchr/testify/assert"
	"testing"
)

const samplePubKey = "73e247ee8c4ff09a50525bed7b0869c371864c0bf2b4d6a2639acaed07613958"
const samplePrivateKey = "4d0888c07093941c9db16fcffb96fdf8af49a6839e865ea6110c7ab7cbd2d3d3"
const sampleValidDirectFeedUrl = "https://mastodon.social/@Gargron.rss"
const sampleValidUrl = "https://mastodon.social/"

func TestGetParsedFeedForPubKey(t *testing.T) {
	testCases := []struct {
		pubKey              string
		expectedReturnUrl   string
		expectedSqlError    bool
		expectedSqlRow      bool
		expectedInvalidUrl  bool
		expectedInvalidFeed bool
	}{
		{
			pubKey:              samplePubKey,
			expectedReturnUrl:   sampleValidDirectFeedUrl,
			expectedSqlError:    false,
			expectedSqlRow:      true,
			expectedInvalidUrl:  false,
			expectedInvalidFeed: false,
		},
		{
			pubKey:              samplePubKey,
			expectedReturnUrl:   "",
			expectedSqlError:    false,
			expectedSqlRow:      false,
			expectedInvalidUrl:  false,
			expectedInvalidFeed: false,
		},
		{
			pubKey:              samplePubKey,
			expectedReturnUrl:   "",
			expectedSqlError:    true,
			expectedSqlRow:      false,
			expectedInvalidUrl:  false,
			expectedInvalidFeed: false,
		},
		{
			pubKey:              samplePubKey,
			expectedReturnUrl:   "invalid",
			expectedSqlError:    false,
			expectedSqlRow:      true,
			expectedInvalidUrl:  true,
			expectedInvalidFeed: false,
		},
		{
			pubKey:              samplePubKey,
			expectedReturnUrl:   sampleValidUrl,
			expectedSqlError:    false,
			expectedSqlRow:      true,
			expectedInvalidUrl:  false,
			expectedInvalidFeed: true,
		},
	}
	for _, tc := range testCases {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		rows := sqlmock.NewRows([]string{"privatekey", "url"})
		if tc.expectedSqlRow {
			rows.AddRow(samplePrivateKey, tc.expectedReturnUrl)
		}

		if tc.expectedSqlError {
			mock.ExpectQuery("SELECT privatekey, url FROM feeds").WillReturnError(errors.New("error"))
		} else {
			mock.ExpectQuery("SELECT privatekey, url FROM feeds").WillReturnRows(rows)
		}
		mock.ExpectClose()

		parsedFeed, entity := GetParsedFeedForPubKey(tc.pubKey, db)
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
			}, entity)
		} else if tc.expectedInvalidFeed {
			assert.Nil(t, parsedFeed)
			assert.Equal(t, feed.Entity{
				PublicKey:  "",
				PrivateKey: samplePrivateKey,
				URL:        tc.expectedReturnUrl,
			}, entity)
		} else {
			assert.NotNil(t, parsedFeed)
			assert.Equal(t, feed.Entity{
				PublicKey:  "",
				PrivateKey: samplePrivateKey,
				URL:        tc.expectedReturnUrl,
			}, entity)
		}
		_ = db.Close()
	}
}
