package handlers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nbd-wtf/go-nostr/nip05"
	"github.com/piraces/rsslay/pkg/feed"
	"github.com/piraces/rsslay/pkg/metrics"
	"github.com/piraces/rsslay/pkg/new/app"
	domainfeed "github.com/piraces/rsslay/pkg/new/domain/feed"
	"github.com/piraces/rsslay/web/templates"
)

var t = template.Must(template.ParseFS(templates.Templates, "*.tmpl"))

type Entry struct {
	PubKey       string
	NPubKey      string
	Url          string
	Error        bool
	ErrorMessage string
	ErrorCode    int
}

type PageData struct {
	Count          uint64
	FilteredCount  uint64
	Entries        []Entry
	MainDomainName string
}

type FeedDefnitionStorage interface {
	ListRandom(n int) ([]domainfeed.FeedDefinition, error)
	Search()
}

type Handler struct {
	app app.App
}

func NewHandler(
	app app.App,
) *Handler {
	return &Handler{
		app: app,
	}
}

func (f *Handler) HandleWebpage(w http.ResponseWriter, r *http.Request, mainDomainName *string) {
	mustRedirect := handleOtherRegion(w, r)
	if mustRedirect {
		return
	}

	metrics.IndexRequests.Inc()

	totalCount, err := f.app.GetTotalFeedCount.Handle()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// todo app handler
	randomFeedDefinitions, err := f.app.GetRandomFeeds.Handle(50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Count:          uint64(totalCount),
		Entries:        toEntries(randomFeedDefinitions),
		MainDomainName: *mainDomainName,
	}

	_ = t.ExecuteTemplate(w, "index.html.tmpl", data)
}

func (f *Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	mustRedirect := handleOtherRegion(w, r)
	if mustRedirect {
		return
	}

	metrics.SearchRequests.Inc()
	query := r.URL.Query().Get("query")
	if query == "" || len(query) <= 4 {
		http.Error(w, "Please enter more than 5 characters to search", 400)
		return
	}

	totalCount, err := f.app.GetTotalFeedCount.Handle()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	feedDefinitions, err := f.app.SearchFeeds.Handle(query, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Count:         uint64(totalCount),
		FilteredCount: uint64(len(feedDefinitions)),
		Entries:       toEntries(feedDefinitions),
	}

	_ = t.ExecuteTemplate(w, "search.html.tmpl", data)
}

func (f *Handler) HandleCreateFeed(w http.ResponseWriter, r *http.Request, dsn *string) {
	mustRedirect := handleRedirectToPrimaryNode(w, dsn)
	if mustRedirect {
		return
	}

	metrics.CreateRequests.Inc()

	entry := f.createFeed(r)
	_ = t.ExecuteTemplate(w, "created.html.tmpl", entry)
}

func (f *Handler) HandleApiFeed(w http.ResponseWriter, r *http.Request, dsn *string) {
	if r.Method == http.MethodGet || r.Method == http.MethodPost {
		f.handleCreateFeedEntry(w, r, dsn)
	} else {
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
	}
}

func HandleNip05(w http.ResponseWriter, r *http.Request, db *sql.DB, ownerPubKey *string, enableAutoRegistration *bool) {
	metrics.WellKnownRequests.Inc()
	name := r.URL.Query().Get("name")
	name, _ = url.QueryUnescape(name)
	w.Header().Set("Content-Type", "application/json")
	nip05WellKnownResponse := nip05.WellKnownResponse{
		Names: map[string]string{
			"_": *ownerPubKey,
		},
		Relays: nil,
	}

	var response []byte
	if name != "" && name != "_" && *enableAutoRegistration {
		row := db.QueryRow("SELECT publickey FROM feeds WHERE url like '%' || $1 || '%'", name)

		var entity feed.Entity
		err := row.Scan(&entity.PublicKey)
		if err == nil {
			nip05WellKnownResponse = nip05.WellKnownResponse{
				Names: map[string]string{
					name: entity.PublicKey,
				},
				Relays: nil,
			}
		}
	}

	response, _ = json.Marshal(nip05WellKnownResponse)
	_, _ = w.Write(response)
}

func (f *Handler) handleCreateFeedEntry(w http.ResponseWriter, r *http.Request, dsn *string) {
	mustRedirect := handleRedirectToPrimaryNode(w, dsn)
	if mustRedirect {
		return
	}

	metrics.CreateRequestsAPI.Inc()

	entry := f.createFeed(r)

	w.Header().Set("Content-Type", "application/json")

	if entry.ErrorCode >= 400 {
		w.WriteHeader(entry.ErrorCode)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	response, _ := json.Marshal(entry)
	_, _ = w.Write(response)
}

func (f *Handler) createFeed(r *http.Request) Entry {
	urlParam := r.URL.Query().Get("url")

	address, err := domainfeed.NewAddress(urlParam)
	if err != nil {
		return Entry{
			Error:        true,
			ErrorMessage: err.Error(),
			ErrorCode:    http.StatusBadRequest,
		}
	}

	feedDefinition, err := f.app.CreateFeedDefinition.Handle(address)
	if err != nil {
		return Entry{
			Error:        true,
			ErrorMessage: err.Error(),
			ErrorCode:    http.StatusInternalServerError,
		}
	}

	return toEntry(*feedDefinition)
}

func handleOtherRegion(w http.ResponseWriter, r *http.Request) bool {
	// If a different region is specified, redirect to that region.
	if region := r.URL.Query().Get("region"); region != "" && region != os.Getenv("FLY_REGION") {
		log.Printf("[DEBUG] redirecting from %q to %q", os.Getenv("FLY_REGION"), region)
		w.Header().Set("fly-replay", "region="+region)
		return true
	}
	return false
}

func handleRedirectToPrimaryNode(w http.ResponseWriter, dsn *string) bool {
	// If this node is not primary, look up and redirect to the current primary.
	primaryFilename := filepath.Join(filepath.Dir(*dsn), ".primary")
	primary, err := os.ReadFile(primaryFilename)
	if err != nil && !os.IsNotExist(err) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return true
	}
	if string(primary) != "" {
		log.Printf("[DEBUG] redirecting to primary instance: %q", string(primary))
		w.Header().Set("fly-replay", "instance="+string(primary))
		return true
	}

	return false
}

func toEntries(definitions []*domainfeed.FeedDefinition) []Entry {
	var entries []Entry
	for _, definition := range definitions {
		entries = append(entries, toEntry(*definition))
	}
	return entries
}

func toEntry(definition domainfeed.FeedDefinition) Entry {
	return Entry{
		PubKey:  definition.PublicKey().Hex(),
		NPubKey: definition.PublicKey().Nip19(),
		Url:     definition.Address().String(),
	}
}
