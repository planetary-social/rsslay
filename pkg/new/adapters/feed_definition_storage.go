package adapters

import (
	"database/sql"
	"github.com/piraces/rsslay/pkg/metrics"
	domainfeed "github.com/piraces/rsslay/pkg/new/domain/feed"
	"github.com/piraces/rsslay/pkg/new/domain/nostr"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

type FeedDefinitionStorage struct {
	db *sql.DB
}

func NewFeedDefinitionStorage(db *sql.DB) *FeedDefinitionStorage {
	return &FeedDefinitionStorage{db: db}
}

func (f *FeedDefinitionStorage) ListRandom(n int) ([]*domainfeed.FeedDefinition, error) {
	var count uint64
	row := f.db.QueryRow(`SELECT count(*) FROM feeds`)
	err := row.Scan(&count)
	if err != nil {
		return nil, errors.Wrap(err, "error counting feeds")
	}

	rows, err := f.db.Query(`SELECT publickey, privatekey, url, nitter FROM feeds ORDER BY RANDOM() LIMIT 50`)
	if err != nil {
		return nil, errors.Wrap(err, "error getting random feed definitions")
	}
	defer rows.Close() // not much we can do here

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
