package main

import (
	"context"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/fiatjaf/relayer"
	_ "github.com/fiatjaf/relayer"
	"github.com/hashicorp/logutils"
	"github.com/hellofresh/health-go/v5"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"
	"github.com/piraces/rsslay/internal/handlers"
	"github.com/piraces/rsslay/pkg/custom_cache"
	"github.com/piraces/rsslay/pkg/feed"
	"github.com/piraces/rsslay/pkg/metrics"
	"github.com/piraces/rsslay/pkg/new/adapters"
	pubsubadapters "github.com/piraces/rsslay/pkg/new/adapters/pubsub"
	"github.com/piraces/rsslay/pkg/new/app"
	"github.com/piraces/rsslay/pkg/new/domain"
	"github.com/piraces/rsslay/pkg/new/ports"
	pubsub2 "github.com/piraces/rsslay/pkg/new/ports/pubsub"
	"github.com/piraces/rsslay/pkg/replayer"
	"github.com/piraces/rsslay/scripts"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Command line flags.
var (
	dsn = flag.String("dsn", "", "datasource name")
)

const assetsDir = "/assets/"

type Relay struct {
	Secret                          string   `envconfig:"SECRET" required:"true"`
	DatabaseDirectory               string   `envconfig:"DB_DIR" default:"db/rsslay.sqlite"`
	DefaultProfilePictureUrl        string   `envconfig:"DEFAULT_PROFILE_PICTURE_URL" default:"https://i.imgur.com/MaceU96.png"`
	Version                         string   `envconfig:"VERSION" default:"unknown"`
	ReplayToRelays                  bool     `envconfig:"REPLAY_TO_RELAYS" default:"false"`
	RelaysToPublish                 []string `envconfig:"RELAYS_TO_PUBLISH_TO" default:""`
	NitterInstances                 []string `envconfig:"NITTER_INSTANCES" default:""`
	DefaultWaitTimeBetweenBatches   int64    `envconfig:"DEFAULT_WAIT_TIME_BETWEEN_BATCHES" default:"60000"`
	DefaultWaitTimeForRelayResponse int64    `envconfig:"DEFAULT_WAIT_TIME_FOR_RELAY_RESPONSE" default:"3000"`
	MaxEventsToReplay               int      `envconfig:"MAX_EVENTS_TO_REPLAY" default:"20"`
	EnableAutoNIP05Registration     bool     `envconfig:"ENABLE_AUTO_NIP05_REGISTRATION" default:"false"`
	MainDomainName                  string   `envconfig:"MAIN_DOMAIN_NAME" default:""`
	OwnerPublicKey                  string   `envconfig:"OWNER_PUBLIC_KEY" default:""`
	MaxSubroutines                  int      `envconfig:"MAX_SUBROUTINES" default:"20"`
	RelayName                       string   `envconfig:"INFO_RELAY_NAME" default:"rsslay"`
	Contact                         string   `envconfig:"INFO_CONTACT" default:"~"`
	MaxContentLength                int      `envconfig:"MAX_CONTENT_LENGTH" default:"250"`
	DeleteFailingFeeds              bool     `envconfig:"DELETE_FAILING_FEEDS" default:"false"`
	RedisConnectionString           string   `envconfig:"REDIS_CONNECTION_STRING" default:""`

	updates            chan nostr.Event
	lastEmitted        sync.Map
	db                 *sql.DB
	healthCheck        *health.Health
	mutex              sync.Mutex
	routineQueueLength int
	converterSelector  *feed.ConverterSelector
	cache              *cache.Cache[string]
	handler            *handlers.Handler
	store              *store
}

var relayInstance = &Relay{
	updates: make(chan nostr.Event),
}

func CreateHealthCheck() {
	h, _ := health.New(health.WithComponent(health.Component{
		Name:    "rsslay",
		Version: os.Getenv("VERSION"),
	}), health.WithChecks(health.Config{
		Name:      "self",
		Timeout:   time.Second * 5,
		SkipOnErr: false,
		Check: func(ctx context.Context) error {
			return nil
		},
	},
	))
	relayInstance.healthCheck = h
}

func ConfigureLogging() {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"},
		MinLevel: logutils.LogLevel(os.Getenv("LOG_LEVEL")),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)
}

func ConfigureCache() {
	if relayInstance.RedisConnectionString != "" {
		custom_cache.RedisConnectionString = &relayInstance.RedisConnectionString
	}
	custom_cache.InitializeCache()
}

func (r *Relay) Name() string {
	return r.RelayName
}

func (r *Relay) OnInitialized(s *relayer.Server) {
	s.Router().Path("/").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		r.handler.HandleWebpage(writer, request, &r.MainDomainName)
	})
	s.Router().Path("/create").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		r.handler.HandleCreateFeed(writer, request, dsn)
	})
	s.Router().Path("/search").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		r.handler.HandleSearch(writer, request)
	})
	s.Router().
		PathPrefix(assetsDir).
		Handler(http.StripPrefix(assetsDir, http.FileServer(http.Dir("./web/"+assetsDir))))
	s.Router().Path("/healthz").HandlerFunc(relayInstance.healthCheck.HandlerFunc)
	s.Router().Path("/api/feed").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		r.handler.HandleApiFeed(writer, request, dsn)
	})
	s.Router().Path("/.well-known/nostr.json").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		handlers.HandleNip05(writer, request, r.db, &r.OwnerPublicKey, &r.EnableAutoNIP05Registration)
	})
	s.Router().Path("/metrics").Handler(promhttp.Handler())
}

func (r *Relay) Init() error {
	ctx := context.TODO()

	flag.Parse()
	err := envconfig.Process("", r)
	if err != nil {
		return fmt.Errorf("couldn't process envconfig: %w", err)
	} else {
		log.Printf("[INFO] Running VERSION %s:\n - DSN=%s\n - DB_DIR=%s\n\n", r.Version, *dsn, r.DatabaseDirectory)
	}

	ConfigureCache()

	db := InitDatabase(r)
	feedDefinitionStorage := adapters.NewFeedDefinitionStorage(db)
	eventStorage := adapters.NewEventStorage()
	receivedEventPubSub := pubsubadapters.NewReceivedEventPubSub()

	secret, err := domain.NewSecret(r.Secret)
	if err != nil {
		return errors.Wrap(err, "error creating a secret")
	}

	r.converterSelector = feed.NewConverterSelector(feed.NewLongFormConverter())

	handlerCreateFeedDefinition := app.NewHandlerCreateFeedDefinition(secret, feedDefinitionStorage)
	handlerUpdateFeeds := app.NewHandlerUpdateFeeds(
		r.DeleteFailingFeeds,
		r.NitterInstances,
		r.EnableAutoNIP05Registration,
		r.DefaultProfilePictureUrl,
		r.MainDomainName,
		db,
		feedDefinitionStorage,
		r.converterSelector,
		eventStorage,
		receivedEventPubSub,
	)
	handlerGetEvents := app.NewHandlerGetEvents(eventStorage)
	handlerOnNewEventCreated := app.NewHandlerOnNewEventCreated(r.updates)

	updateFeedsTimer := ports.NewUpdateFeedsTimer(handlerUpdateFeeds)
	receivedEventSubscriber := pubsub2.NewReceivedEventSubscriber(receivedEventPubSub, handlerOnNewEventCreated)

	app := app.App{
		CreateFeedDefinition: handlerCreateFeedDefinition,
		UpdateFeeds:          handlerUpdateFeeds,
		GetEvents:            handlerGetEvents,
	}

	r.db = db
	r.handler = handlers.NewHandler(db, feedDefinitionStorage, app)
	r.store = newStore(app)

	go updateFeedsTimer.Run(ctx)
	go receivedEventSubscriber.Run(ctx)

	return nil
}

func (r *Relay) AttemptReplayEvents(events []replayer.EventWithPrivateKey) {
	if relayInstance.ReplayToRelays && relayInstance.routineQueueLength < relayInstance.MaxSubroutines && len(events) > 0 {
		r.routineQueueLength++
		metrics.ReplayRoutineQueueLength.Set(float64(r.routineQueueLength))
		replayer.ReplayEventsToRelays(&replayer.ReplayParameters{
			MaxEventsToReplay:        relayInstance.MaxEventsToReplay,
			RelaysToPublish:          relayInstance.RelaysToPublish,
			Mutex:                    &relayInstance.mutex,
			Queue:                    &relayInstance.routineQueueLength,
			WaitTime:                 relayInstance.DefaultWaitTimeBetweenBatches,
			WaitTimeForRelayResponse: relayInstance.DefaultWaitTimeForRelayResponse,
			Events:                   events,
		})
	}
}

func (r *Relay) AcceptEvent(_ *nostr.Event) bool {
	metrics.InvalidEventsRequests.Inc()
	return false
}

func (r *Relay) Storage() relayer.Storage {
	return r.store
}

func (r *Relay) InjectEvents() chan nostr.Event {
	return r.updates
}

func (r *Relay) GetNIP11InformationDocument() nip11.RelayInformationDocument {
	metrics.RelayInfoRequests.Inc()
	infoDocument := nip11.RelayInformationDocument{
		Name:          relayInstance.Name(),
		Description:   "Relay that creates virtual nostr profiles for each RSS feed submitted, powered by the relayer framework",
		PubKey:        relayInstance.OwnerPublicKey,
		Contact:       relayInstance.Contact,
		SupportedNIPs: []int{5, 9, 11, 12, 15, 16, 19, 20},
		Software:      "git+https://github.com/piraces/rsslay.git",
		Version:       relayInstance.Version,
	}

	if relayInstance.OwnerPublicKey == "" {
		infoDocument.PubKey = "~"
	}

	return infoDocument
}

func main() {
	CreateHealthCheck()
	ConfigureLogging()
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatalf("[FATAL] failed to close the database connection: %v", err)
		}
	}(relayInstance.db)

	if err := relayer.Start(relayInstance); err != nil {
		log.Fatalf("[FATAL] server terminated: %v", err)
	}
}

func InitDatabase(r *Relay) *sql.DB {
	finalConnection := dsn
	if *dsn == "" {
		log.Print("[INFO] dsn required is not present... defaulting to DB_DIR")
		finalConnection = &r.DatabaseDirectory
	}

	// Create empty dir if not exists
	dbPath := path.Dir(*finalConnection)
	err := os.MkdirAll(dbPath, 0660)
	if err != nil {
		log.Printf("[INFO] unable to initialize DB_DIR at: %s. Error: %v", dbPath, err)
	}

	// Connect to SQLite database.
	sqlDb, err := sql.Open("sqlite3", *finalConnection)
	if err != nil {
		log.Fatalf("[FATAL] open db: %v", err)
	}

	log.Printf("[INFO] database opened at %s", *finalConnection)

	// Run migrations
	if _, err := sqlDb.Exec(scripts.SchemaSQL); err != nil {
		log.Fatalf("[FATAL] cannot migrate schema: %v", err)
	}

	if _, err := sqlDb.Exec(scripts.CheckNitterColumnSQL); err != nil {
		_, err := sqlDb.Exec(scripts.CreateNitterColumnSQL)
		if err != nil {
			log.Fatalf("[FATAL] cannot migrate schema from previous versions: %v", err)
		}
	}

	return sqlDb
}
