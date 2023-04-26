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
	"github.com/piraces/rsslay/pkg/helpers"
	"github.com/rif/cache2go"
)

var (
	fp        = gofeed.NewParser()
	feedCache = cache2go.New(512, time.Minute*19)
	client    = &http.Client{
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
	if feed, ok := feedCache.Get(url); ok {
		return feed.(*gofeed.Feed), nil
	}
	fp.RSSTranslator = NewCustomTranslator()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, err
	}

	feedCache.Set(url, feed)

	return feed, nil
}

func EntryFeedToSetMetadata(pubkey string, feed *gofeed.Feed, originalUrl string, enableAutoRegistration bool, defaultProfilePictureUrl string) nostr.Event {
	// Handle Nitter special cases (http schema)
	if strings.Contains(feed.Description, "Twitter feed") {
		if strings.HasPrefix(originalUrl, "https://") {
			feed.Description = strings.ReplaceAll(feed.Description, "http://", "https://")
			feed.Title = strings.ReplaceAll(feed.Title, "http://", "https://")
			feed.Image.URL = strings.ReplaceAll(feed.Image.URL, "http://", "https://")
			feed.Link = strings.ReplaceAll(feed.Link, "http://", "https://")
		}
	}

	metadata := map[string]string{
		"name":  feed.Title + " (RSS Feed)",
		"about": feed.Description + "\n\n" + feed.Link,
	}

	if enableAutoRegistration {
		metadata["nip05"] = fmt.Sprintf("%s@%s", originalUrl, "rsslay.nostr.moe")
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
	} else {
		log.Printf("[DEBUG] deleted invalid feed with url %q", url)
	}
}
