package events

import (
	"database/sql"
	"github.com/mmcdole/gofeed"
	"github.com/piraces/rsslay/pkg/feed"
	"github.com/piraces/rsslay/pkg/helpers"
	"log"
	"strings"
)

func GetParsedFeedForPubKey(pubKey string, db *sql.DB) (*gofeed.Feed, feed.Entity) {
	pubKey = strings.TrimSpace(pubKey)
	row := db.QueryRow("SELECT privatekey, url FROM feeds WHERE publickey=$1", pubKey)

	var entity feed.Entity
	err := row.Scan(&entity.PrivateKey, &entity.URL)
	if err != nil && err == sql.ErrNoRows {
		return nil, entity
	} else if err != nil {
		log.Printf("failed when trying to retrieve row with pubkey '%s': %v", pubKey, err)
		return nil, entity
	}

	if !helpers.IsValidHttpUrl(entity.URL) {
		log.Printf("retrieved invalid url from database %q, deleting...", entity.URL)
		feed.DeleteInvalidFeed(entity.URL, db)
		return nil, entity
	}

	parsedFeed, err := feed.ParseFeed(entity.URL)
	if err != nil {
		log.Printf("failed to parse feed at url %q: %v", entity.URL, err)
		feed.DeleteInvalidFeed(entity.URL, db)
		return nil, entity
	}

	return parsedFeed, entity
}
