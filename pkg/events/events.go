package events

import (
	"database/sql"
	"github.com/mmcdole/gofeed"
	"github.com/piraces/rsslay/pkg/feed"
	"github.com/piraces/rsslay/pkg/helpers"
	"github.com/piraces/rsslay/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"net/url"
	"strings"
)

func GetParsedFeedForPubKey(pubKey string, db *sql.DB, deleteFailingFeeds bool, nitterInstances []string) (*gofeed.Feed, feed.Entity) {
	pubKey = strings.TrimSpace(pubKey)
	row := db.QueryRow("SELECT privatekey, url, nitter FROM feeds WHERE publickey=$1", pubKey)

	var entity feed.Entity
	err := row.Scan(&entity.PrivateKey, &entity.URL, &entity.Nitter)
	if err != nil && err == sql.ErrNoRows {
		return nil, entity
	} else if err != nil {
		log.Printf("[ERROR] failed when trying to retrieve row with pubkey '%s': %v", pubKey, err)
		metrics.AppErrors.With(prometheus.Labels{"type": "SQL_SCAN"}).Inc()
		return nil, entity
	}

	if !helpers.IsValidHttpUrl(entity.URL) {
		log.Printf("[INFO] retrieved invalid url from database %q", entity.URL)
		if deleteFailingFeeds {
			feed.DeleteInvalidFeed(entity.URL, db)
		}
		return nil, entity
	}

	parsedFeed, err := feed.ParseFeed(entity.URL)
	if err != nil && entity.Nitter {
		log.Printf("[DEBUG] failed to parse feed at url %q: %v. Now iterating through other Nitter instances", entity.URL, err)
		for i, instance := range nitterInstances {
			newUrl, _ := setHostname(entity.URL, instance)
			log.Printf("[DEBUG] attempt %d: use %q instead of %q", i, newUrl, entity.URL)
			parsedFeed, err = feed.ParseFeed(newUrl)
			if err == nil {
				log.Printf("[DEBUG] attempt %d: success with %q", i, newUrl)
				break
			}
		}
	}

	if err != nil {
		log.Printf("[DEBUG] failed to parse feed at url %q: %v", entity.URL, err)
		if deleteFailingFeeds {
			feed.DeleteInvalidFeed(entity.URL, db)
		}
		return nil, entity
	}

	if strings.Contains(parsedFeed.Description, "Twitter feed") && !entity.Nitter {
		updateDatabaseEntry(&entity, db)
		entity.Nitter = true
	}

	return parsedFeed, entity
}

func updateDatabaseEntry(entity *feed.Entity, db *sql.DB) {
	log.Printf("[DEBUG] attempting to set feed at url %q with publicKey %s as nitter instance", entity.URL, entity.PublicKey)
	if _, err := db.Exec(`UPDATE feeds SET nitter = ? WHERE publickey = ?`, 1, entity.PublicKey); err != nil {
		log.Printf("[ERROR] failure while updating record on db to set as nitter feed: %v", err)
		metrics.AppErrors.With(prometheus.Labels{"type": "SQL_WRITE"}).Inc()
	} else {
		log.Printf("[DEBUG] set feed at url %q with publicKey %s as nitter instance", entity.URL, entity.PublicKey)
	}
}

func setHostname(addr, hostname string) (string, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return "", err
	}
	u.Host = hostname
	return u.String(), nil
}
