package replayer

import (
	"context"
	"github.com/nbd-wtf/go-nostr"
	"github.com/piraces/rsslay/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"sort"
	"sync"
	"time"
)

type ReplayParameters struct {
	MaxEventsToReplay        int
	RelaysToPublish          []string
	Mutex                    *sync.Mutex
	Queue                    *int
	WaitTime                 int64
	WaitTimeForRelayResponse int64
	Events                   []EventWithPrivateKey
}

type EventWithPrivateKey struct {
	Event      *nostr.Event
	PrivateKey string
}

func ReplayEventsToRelays(parameters *ReplayParameters) {
	eventCount := len(parameters.Events)
	if eventCount == 0 {
		return
	}

	if eventCount > parameters.MaxEventsToReplay {
		sort.Slice(parameters.Events, func(i, j int) bool {
			return parameters.Events[i].Event.CreatedAt > parameters.Events[j].Event.CreatedAt
		})
		parameters.Events = parameters.Events[:parameters.MaxEventsToReplay]
	}

	go func() {
		parameters.Mutex.Lock()
		// publish the event to predefined relays
		for _, url := range parameters.RelaysToPublish {
			statusSummary := 0
			for _, ev := range parameters.Events {
				relay := connectToRelay(url, ev.PrivateKey)
				if relay == nil {
					continue
				}
				_ = relay.Close()

				publishStatus := publishEvent(relay, *ev.Event, url)
				statusSummary = statusSummary | int(publishStatus)
			}
			if statusSummary < 0 {
				log.Printf("[WARN] Replayed %d events to %s with failed status summary %d\n", len(parameters.Events), url, statusSummary)
			} else {
				log.Printf("[DEBUG] Replayed %d events to %s with status summary %d\n", len(parameters.Events), url, statusSummary)
			}
		}
		time.Sleep(time.Duration(parameters.WaitTime) * time.Millisecond)
		*parameters.Queue--
		metrics.ReplayRoutineQueueLength.Set(float64(*parameters.Queue))
		parameters.Mutex.Unlock()
	}()
}

func publishEvent(relay *nostr.Relay, ev nostr.Event, url string) nostr.Status {
	publishStatus, err := relay.Publish(context.Background(), ev)
	switch publishStatus {
	case nostr.PublishStatusSent:
		metrics.ReplayEvents.With(prometheus.Labels{"relay": url}).Inc()
		break
	default:
		metrics.ReplayErrorEvents.With(prometheus.Labels{"relay": url}).Inc()
		break
	}
	_ = relay.Close()
	if err != nil {
		log.Printf("[INFO] Failed to replay event to %s with error: %v", url, err)
	}
	return publishStatus
}

func connectToRelay(url string, privateKey string) *nostr.Relay {
	relay, e := nostr.RelayConnect(context.Background(), url, nostr.WithAuthHandler(func(ctx context.Context, authEvent *nostr.Event) (ok bool) {
		err := authEvent.Sign(privateKey)
		if err != nil {
			log.Printf("[ERROR] Error while trying to authenticate with relay '%s': %v", url, err)
			return false
		}
		return true
	}),
	)
	if e != nil {
		log.Printf("[ERROR] Error while trying to connect with relay '%s': %v", url, e)
		metrics.AppErrors.With(prometheus.Labels{"type": "REPLAY_CONNECT"}).Inc()
		return nil
	}

	return relay
}
