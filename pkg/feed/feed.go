package feed

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
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

const (
	causesLink       = "https://www.causes.com/api/v2/articles?feed_id=recency"
	causesNumWorkers = 10
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
	if url == causesLink {
		return causesLink
	}

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

	parser := getFeedParser(url)
	feed, err := parser.Parse()
	if err != nil {
		return nil, err
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

func getFeedParser(feedURL string) FeedParser {
	downloader := NewDownloader()

	switch feedURL {
	case causesLink:
		return NewCausesFeedParser(downloader, feedURL)
	default:
		return NewDefaultFeedParser(downloader, feedURL)
	}
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
		Tags:      nostr.Tags{[]string{"proxy", feed.FeedLink, "rss"}},
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

type FeedParser interface {
	Parse() (*gofeed.Feed, error)
}

type DefaultFeedParser struct {
	downloader *Downloader
	url        string
}

func NewDefaultFeedParser(downloader *Downloader, url string) *DefaultFeedParser {
	return &DefaultFeedParser{downloader: downloader, url: url}
}

func (d *DefaultFeedParser) Parse() (*gofeed.Feed, error) {
	body, err := d.downloader.Download(d.url)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	fp.RSSTranslator = NewCustomTranslator()
	return fp.Parse(body)
}

type causesResponseOrError struct {
	Response causesResponse
	Err      error
}

type CausesFeedParser struct {
	downloader *Downloader
	url        string
}

func NewCausesFeedParser(downloader *Downloader, url string) *CausesFeedParser {
	return &CausesFeedParser{downloader: downloader, url: url}
}

func (d *CausesFeedParser) Parse() (*gofeed.Feed, error) {
	resp, err := d.get(d.url)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chIn := make(chan int)
	chOut := make(chan causesResponseOrError)

	d.startWorkers(ctx, chIn, chOut, causesNumWorkers)

	go func() {
		defer close(chIn)

		for i := 1; i <= resp.Meta.Pagination.TotalPages; i++ {
			select {
			case chIn <- i:
				continue
			case <-ctx.Done():
				return
			}
		}
	}()

	feed := &gofeed.Feed{
		Title:       "causes.com",
		Description: "Causes - powered by Countable - makes it quick and easy to understand the laws Congress is considering.",
		Link:        "https://www.causes.com/",
		FeedLink:    causesLink,
		Links:       nil,
		Items:       nil,
	}

	for i := 1; i <= resp.Meta.Pagination.TotalPages; i++ {
		select {
		case result := <-chOut:
			if err := result.Err; err != nil {
				return nil, fmt.Errorf("worker error: %w", err)
			}

			for _, article := range result.Response.Articles {
				article := article
				item := d.itemFromArticle(article)
				feed.Items = append(feed.Items, item)
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return feed, nil
}

func (d *CausesFeedParser) startWorkers(ctx context.Context, chIn <-chan int, chOut chan<- causesResponseOrError, n int) {
	for i := 0; i < n; i++ {
		go d.startWorker(ctx, chIn, chOut)
	}
}

func (d *CausesFeedParser) startWorker(ctx context.Context, chIn <-chan int, chOut chan<- causesResponseOrError) {
	for {
		select {
		case in := <-chIn:
			result, err := d.work(in)
			if err != nil {
				select {
				case chOut <- causesResponseOrError{Err: err}:
					continue
				case <-ctx.Done():
					return
				}
			}

			select {
			case chOut <- causesResponseOrError{Response: result}:
				continue
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (d *CausesFeedParser) work(page int) (causesResponse, error) {
	return d.get(fmt.Sprintf("%s&page=%d", d.url, page))
}

func (d *CausesFeedParser) get(url string) (causesResponse, error) {
	var resp causesResponse

	body, err := d.downloader.Download(url)
	if err != nil {
		return resp, err
	}
	defer body.Close()

	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return resp, err
	}

	return resp, nil
}

func (d *CausesFeedParser) itemFromArticle(article causesResponseArticle) *gofeed.Item {
	return &gofeed.Item{
		Title:           article.Title,
		Content:         article.HtmlContent,
		Link:            article.Links.Self,
		Published:       article.CreatedAt.Format(time.RFC3339),
		PublishedParsed: &article.CreatedAt,
		GUID:            strconv.Itoa(article.Id),
	}

}

type causesResponse struct {
	Articles []causesResponseArticle `json:"articles"`
	Meta     causesResponseMeta      `json:"meta"`
}

type causesResponseArticle struct {
	Id          int                        `json:"id"`
	Title       string                     `json:"title"`
	CreatedAt   time.Time                  `json:"created_at"`
	HtmlContent string                     `json:"html_content"`
	Links       causesResponseArticleLinks `json:"links"`
}

type causesResponseArticleLinks struct {
	Self string `json:"self"`
}

type causesResponseMeta struct {
	Pagination causesResponseMetaPagination `json:"pagination"`
}

type causesResponseMetaPagination struct {
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
	TotalCount  int `json:"total_count"`
}
