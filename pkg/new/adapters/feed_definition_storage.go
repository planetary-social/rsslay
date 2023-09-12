package adapters

import (
	"database/sql"
	"github.com/piraces/rsslay/pkg/feed"
	"github.com/piraces/rsslay/pkg/metrics"
	domainfeed "github.com/piraces/rsslay/pkg/new/domain/feed"
	"github.com/piraces/rsslay/pkg/new/domain/nostr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"log"
)

type FeedDefinitionStorage struct {
	db *sql.DB
}

func (f *FeedDefinitionStorage) List() ([]*domainfeed.FeedDefinition, error) {
	//TODO implement me
	panic("implement me")
}

func NewFeedDefinitionStorage(db *sql.DB) *FeedDefinitionStorage {
	return &FeedDefinitionStorage{db: db}
}

func (f *FeedDefinitionStorage) CountTotal() (int, error) {
	var count int
	row := f.db.QueryRow(`SELECT count(*) FROM feeds`)
	err := row.Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "error counting feeds")
	}

	return count, nil
}

func (f *FeedDefinitionStorage) ListRandom(limit int) ([]*domainfeed.FeedDefinition, error) {
	rows, err := f.db.Query(`
		SELECT publickey, privatekey, url, nitter
		FROM feeds
		ORDER BY RANDOM()
		LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error getting feed definitions")
	}
	defer rows.Close() // not much we can do here

	return f.scan(rows)
}

func (f *FeedDefinitionStorage) Search(query string, limit int) ([]*domainfeed.FeedDefinition, error) {
	rows, err := f.db.Query(`
		SELECT publickey, privatekey, url, nitter
		FROM feeds
		WHERE url
		LIKE '%' || $1 || '%' LIMIT $2`,
		query,
		limit,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error getting feed definitions")
	}
	defer rows.Close() // not much we can do here

	return f.scan(rows)
}

func (f *FeedDefinitionStorage) Put(definition *domainfeed.FeedDefinition) error {
	row := f.db.QueryRow("SELECT privatekey, url FROM feeds WHERE publickey=$1", definition.PublicKey().Hex())

	var entity feed.Entity
	err := row.Scan(&entity.PrivateKey, &entity.URL)
	if err != nil && err == sql.ErrNoRows {
		log.Printf("[DEBUG] not found feed at url %q as publicKey %s", definition.Address().String(), definition.PublicKey().Hex())
		if _, err := f.db.Exec(`INSERT INTO feeds (publickey, privatekey, url, nitter) VALUES (?, ?, ?, ?)`, definition.PublicKey().Hex(), definition.PrivateKey().Hex(), definition.Address().String(), definition.Nitter()); err != nil {
			log.Printf("[ERROR] failure: %v", err)
			metrics.AppErrors.With(prometheus.Labels{"type": "SQL_WRITE"}).Inc()
			return errors.Wrap(err, "error inserting the new feed")
		} else {
			log.Printf("[DEBUG] saved feed at url %q as publicKey %s", definition.Address().String(), definition.PublicKey().Hex())
			return nil
		}
	} else if err != nil {
		metrics.AppErrors.With(prometheus.Labels{"type": "SQL_SCAN"}).Inc()
		return errors.Wrap(err, "error checking if feed exists")
	}

	log.Printf("[DEBUG] found feed at url %q as publicKey %s", definition.Address().String(), definition.PublicKey().Hex())
	return nil
}

func (f *FeedDefinitionStorage) scan(rows *sql.Rows) ([]*domainfeed.FeedDefinition, error) {
	var items []*domainfeed.FeedDefinition
	for rows.Next() {
		var (
			tmppublickey  string
			tmpprivatekey string
			tmpurl        string
			tmpnitter     bool
		)

		if err := rows.Scan(&tmppublickey, &tmpprivatekey, &tmpurl, &tmpnitter); err != nil {
			metrics.AppErrors.With(prometheus.Labels{"type": "SQL_SCAN"}).Inc()
			return nil, errors.Wrap(err, "error scanning the retrieved rows")
		}

		publicKey, err := nostr.NewPublicKeyFromHex(tmppublickey)
		if err != nil {
			return nil, errors.Wrap(err, "error creating public key")
		}

		privateKey, err := nostr.NewPrivateKeyFromHex(tmpprivatekey)
		if err != nil {
			return nil, errors.Wrap(err, "error creating private key")
		}

		address, err := domainfeed.NewAddress(tmpurl)
		if err != nil {
			return nil, errors.Wrap(err, "error creating address")
		}

		feedDefinition := domainfeed.NewFeedDefinition(
			publicKey,
			privateKey,
			address,
			tmpnitter,
		)

		items = append(items, feedDefinition)
	}
	return items, nil
}
