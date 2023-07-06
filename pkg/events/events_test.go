package events

import (
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/piraces/rsslay/pkg/feed"
	"github.com/stretchr/testify/assert"
)

const samplePubKey = "73e247ee8c4ff09a50525bed7b0869c371864c0bf2b4d6a2639acaed07613958"
const samplePrivateKey = "4d0888c07093941c9db16fcffb96fdf8af49a6839e865ea6110c7ab7cbd2d3d3"
const sampleValidNitterFeedUrl = "https://nitter.moomoo.me/Twitter/rss"
const sampleInvalidNitterFeedUrl = "https://example.com/Twitter/rss"
const sampleValidUrl = "https://mastodon.social/"

var nitterInstances = []string{"birdsite.xanny.family", "notabird.site", "nitter.moomoo.me", "nitter.fly.dev"}
var sqlRows = []string{"privatekey", "url", "nitter"}

func TestGetParsedFeedForNitterPubKey(t *testing.T) {
	t.Skip()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	rows := sqlmock.NewRows(sqlRows)
	rows.AddRow(samplePrivateKey, sampleValidNitterFeedUrl, true)
	mock.ExpectQuery("SELECT privatekey, url, nitter FROM feeds").WillReturnRows(rows)
	mock.ExpectClose()

	parsedFeed, entity := GetParsedFeedForPubKey(samplePubKey, db, true, nitterInstances)
	assert.NotNil(t, parsedFeed)
	assert.Equal(t, feed.Entity{
		PublicKey:  "",
		PrivateKey: samplePrivateKey,
		URL:        sampleValidNitterFeedUrl,
		Nitter:     true,
	}, entity)
	_ = db.Close()
}

func TestGetParsedFeedForExistingOutdatedNitterPubKey(t *testing.T) {
	t.Skip()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	rows := sqlmock.NewRows(sqlRows)
	rows.AddRow(samplePrivateKey, sampleValidNitterFeedUrl, false)
	mock.ExpectQuery("SELECT privatekey, url, nitter FROM feeds").WillReturnRows(rows)
	mock.ExpectExec("UPDATE feeds").WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectClose()

	parsedFeed, entity := GetParsedFeedForPubKey(samplePubKey, db, true, nitterInstances)
	assert.NotNil(t, parsedFeed)
	assert.Equal(t, feed.Entity{
		PublicKey:  "",
		PrivateKey: samplePrivateKey,
		URL:        sampleValidNitterFeedUrl,
		Nitter:     true,
	}, entity)
	_ = db.Close()
}

func TestGetParsedFeedForErrorExistingOutdatedNitterPubKey(t *testing.T) {
	t.Skip()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	rows := sqlmock.NewRows(sqlRows)
	rows.AddRow(samplePrivateKey, sampleValidNitterFeedUrl, false)
	mock.ExpectQuery("SELECT privatekey, url, nitter FROM feeds").WillReturnRows(rows)
	mock.ExpectExec("UPDATE feeds").WillReturnError(errors.New("error"))
	mock.ExpectClose()

	parsedFeed, entity := GetParsedFeedForPubKey(samplePubKey, db, true, nitterInstances)
	assert.NotNil(t, parsedFeed)
	assert.Equal(t, feed.Entity{
		PublicKey:  "",
		PrivateKey: samplePrivateKey,
		URL:        sampleValidNitterFeedUrl,
		Nitter:     true,
	}, entity)
	_ = db.Close()
}

func TestGetParsedFeedSQLErrorForNitterPubKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	rows := sqlmock.NewRows(sqlRows)
	rows.AddRow(samplePrivateKey, sampleValidNitterFeedUrl, false)
	mock.ExpectQuery("SELECT privatekey, url, nitter FROM feeds").WillReturnError(errors.New("error"))
	mock.ExpectClose()

	parsedFeed, entity := GetParsedFeedForPubKey(samplePubKey, db, true, nitterInstances)
	assert.Nil(t, parsedFeed)
	assert.Empty(t, entity)
	_ = db.Close()
}

func TestGetParsedFeedInvalidFeedUrlForNitterPubKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	rows := sqlmock.NewRows(sqlRows)
	rows.AddRow(samplePrivateKey, sampleInvalidNitterFeedUrl, false)
	mock.ExpectQuery("SELECT privatekey, url, nitter FROM feeds").WillReturnRows(rows)
	mock.ExpectClose()

	parsedFeed, entity := GetParsedFeedForPubKey(samplePubKey, db, true, nitterInstances)
	assert.Nil(t, parsedFeed)
	assert.Equal(t, feed.Entity{
		PublicKey:  "",
		PrivateKey: samplePrivateKey,
		URL:        sampleInvalidNitterFeedUrl,
		Nitter:     false,
	}, entity)
	_ = db.Close()
}

func TestGetParsedFeedInvalidFeedUrlForStandardPubKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	rows := sqlmock.NewRows(sqlRows)
	rows.AddRow(samplePrivateKey, sampleValidUrl, false)
	mock.ExpectQuery("SELECT privatekey, url, nitter FROM feeds").WillReturnRows(rows)
	expectedDeleteQuery := fmt.Sprintf("DELETE FROM feeds WHERE url=%s", sampleValidUrl)
	mock.ExpectQuery(expectedDeleteQuery)
	mock.ExpectClose()

	parsedFeed, entity := GetParsedFeedForPubKey(samplePubKey, db, true, nitterInstances)
	assert.Nil(t, parsedFeed)
	assert.Equal(t, feed.Entity{
		PublicKey:  "",
		PrivateKey: samplePrivateKey,
		URL:        sampleValidUrl,
		Nitter:     false,
	}, entity)
	_ = db.Close()
}

func TestGetParsedFeedInvalidUrlForStandardPubKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	rows := sqlmock.NewRows(sqlRows)
	rows.AddRow(samplePrivateKey, "not a url", false)
	mock.ExpectQuery("SELECT privatekey, url, nitter FROM feeds").WillReturnRows(rows)
	expectedDeleteQuery := fmt.Sprintf("DELETE FROM feeds WHERE url=%s", sampleValidUrl)
	mock.ExpectQuery(expectedDeleteQuery)
	mock.ExpectClose()

	parsedFeed, entity := GetParsedFeedForPubKey(samplePubKey, db, true, nitterInstances)
	assert.Nil(t, parsedFeed)
	assert.Equal(t, feed.Entity{
		PublicKey:  "",
		PrivateKey: samplePrivateKey,
		URL:        "not a url",
		Nitter:     false,
	}, entity)
	_ = db.Close()
}
