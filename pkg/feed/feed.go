package feed

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
	"github.com/nbd-wtf/go-nostr"
	"github.com/piraces/rsslay/pkg/custom_cache"
	"github.com/piraces/rsslay/pkg/helpers"
	"github.com/piraces/rsslay/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	fp     = gofeed.NewParser()
	client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 2 {
				return errors.New("stopped after 2 redirects")
			}
			return nil
		},
		Timeout: 5 * time.Second,
	}
)

type Entity struct {
	PublicKey  string
	PrivateKey string
	URL        string
	Nitter     bool
}

var types = []string{
	"rss+xml",
	"atom+xml",
	"feed+json",
	"text/xml",
	"application/xml",
}

func GetFeedURL(url string) string {
	resp, err := client.Get(url)
	if err != nil || resp.StatusCode >= 300 {
		return ""
	}

	ct := resp.Header.Get("Content-Type")
	for _, typ := range types {
		if strings.Contains(ct, typ) {
			return url
		}
	}

	if strings.Contains(ct, "text/html") {
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return ""
		}

		for _, typ := range types {
			href, _ := doc.Find(fmt.Sprintf("link[type*='%s']", typ)).Attr("href")
			if href == "" {
				continue
			}
			if !strings.HasPrefix(href, "http") && !strings.HasPrefix(href, "https") {
				href, _ = helpers.UrlJoin(url, href)
			}
			return href
		}
	}

	return ""
}

func ParseFeed(url string) (*gofeed.Feed, error) {
	feedString, err := custom_cache.Get(url)
	if err == nil {
		metrics.CacheHits.Inc()

		var feed gofeed.Feed
		err := json.Unmarshal([]byte(feedString), &feed)
		if err != nil {
			log.Printf("[ERROR] failure to parse cache stored feed: %v", err)
			metrics.AppErrors.With(prometheus.Labels{"type": "CACHE_PARSE"}).Inc()
		} else {
			return &feed, nil
		}
	} else {
		log.Printf("[DEBUG] entry not found in cache: %v", err)
	}

	metrics.CacheMiss.Inc()
	fp.RSSTranslator = NewCustomTranslator()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, err
	}

	// cleanup a little so we don't store too much junk
	for i := range feed.Items {
		feed.Items[i].Content = ""
	}

	marshal, err := json.Marshal(feed)
	if err == nil {
		err = custom_cache.Set(url, string(marshal))
	}

	if err != nil {
		log.Printf("[ERROR] failure to store into cache feed: %v", err)
		metrics.AppErrors.With(prometheus.Labels{"type": "CACHE_SET"}).Inc()
	}

	return feed, nil
}

func EntryFeedToSetMetadata(pubkey string, feed *gofeed.Feed, originalUrl string, enableAutoRegistration bool, defaultProfilePictureUrl string, mainDomainName string) nostr.Event {
	// Handle Nitter special cases (http schema)
	if strings.Contains(feed.Description, "Twitter feed") {
		if strings.HasPrefix(originalUrl, "https://") {
			feed.Description = strings.ReplaceAll(feed.Description, "http://", "https://")
			feed.Title = strings.ReplaceAll(feed.Title, "http://", "https://")
			if feed.Image != nil {
				feed.Image.URL = strings.ReplaceAll(feed.Image.URL, "http://", "https://")
			}

			feed.Link = strings.ReplaceAll(feed.Link, "http://", "https://")
		}
	}

	var theDescription = feed.Description
	var theFeedTitle = feed.Title
	if strings.Contains(feed.Link, "reddit.com") {
		var subredditParsePart1 = strings.Split(feed.Link, "/r/")
		var subredditParsePart2 = strings.Split(subredditParsePart1[1], "/")
		theDescription = feed.Description + fmt.Sprintf(" #%s", subredditParsePart2[0])

		theFeedTitle = "/r/" + subredditParsePart2[0]
	}
	metadata := map[string]string{
		"name":  theFeedTitle + " (RSS Feed)",
		"about": theDescription + "\n\n" + feed.Link,
	}

	if enableAutoRegistration {
		metadata["nip05"] = fmt.Sprintf("%s@%s", originalUrl, mainDomainName)
	}

	if feed.Image != nil {
		metadata["picture"] = feed.Image.URL
	} else if defaultProfilePictureUrl != "" {
		metadata["picture"] = defaultProfilePictureUrl
	}

	content, _ := json.Marshal(metadata)

	createdAt := time.Unix(time.Now().Unix(), 0)
	if feed.PublishedParsed != nil {
		createdAt = *feed.PublishedParsed
	}

	evt := nostr.Event{
		PubKey:    pubkey,
		CreatedAt: nostr.Timestamp(createdAt.Unix()),
		Kind:      nostr.KindSetMetadata,
		Tags:      nostr.Tags{},
		Content:   string(content),
	}
	evt.ID = string(evt.Serialize())

	return evt
}

func PrivateKeyFromFeed(url string, secret string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(url))
	r := m.Sum(nil)
	return hex.EncodeToString(r)
}

func DeleteInvalidFeed(url string, db *sql.DB) {
	if _, err := db.Exec(`DELETE FROM feeds WHERE url=?`, url); err != nil {
		log.Printf("[ERROR] failure to delete invalid feed: %v", err)
		metrics.AppErrors.With(prometheus.Labels{"type": "SQL_WRITE"}).Inc()
	} else {
		log.Printf("[DEBUG] deleted invalid feed with url %q", url)
	}
}
