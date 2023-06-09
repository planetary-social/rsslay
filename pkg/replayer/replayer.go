package replayer

import (
	"context"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip42"
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
	Event         *nostr.Event
	PrivateKey    string
	MetadataEvent *nostr.Event
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
			relay := connectToRelay(url)
			if relay == nil {
				continue
			}

			challenge, shouldPerformAuthRequest := needsAuth(relay, parameters.WaitTimeForRelayResponse)

			statusSummary := 0
			for _, ev := range parameters.Events {
				if shouldPerformAuthRequest && !tryAuth(relay, *challenge, url, parameters.WaitTimeForRelayResponse, &ev) {
					continue
				}

				err := relay.Connection.Ping()
				if err != nil {
					log.Printf("[DEBUG] ping to relay failed, reconnecting to %s because of error: %v\n", url, err)
					relay = connectToRelay(url)
				}

				if ev.MetadataEvent != nil {
					publishMetadataStatus := publishEvent(err, relay, *ev.MetadataEvent, url)
					statusSummary = statusSummary | int(publishMetadataStatus)
				}

				publishStatus := publishEvent(err, relay, *ev.Event, url)
				statusSummary = statusSummary | int(publishStatus)
			}
			if statusSummary < 0 {
				log.Printf("[WARN] Replayed %d events to %s with failed status summary %d\n", len(parameters.Events), url, statusSummary)
			} else {
				log.Printf("[DEBUG] Replayed %d events to %s with status summary %d\n", len(parameters.Events), url, statusSummary)
			}

			_ = relay.Close()
		}
		time.Sleep(time.Duration(parameters.WaitTime) * time.Millisecond)
		*parameters.Queue--
		metrics.ReplayRoutineQueueLength.Set(float64(*parameters.Queue))
		parameters.Mutex.Unlock()
	}()
}

func publishEvent(err error, relay *nostr.Relay, ev nostr.Event, url string) nostr.Status {
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

func needsAuth(relay *nostr.Relay, waitTime int64) (*string, bool) {
	afterCh := time.After(time.Duration(waitTime) * time.Millisecond)
	for {
		select {
		case d := <-relay.Challenges:
			log.Println("[DEBUG] Got challenge:", d)
			return &d, true
		case <-afterCh:
			log.Println("[DEBUG] No challenge received... Skipping auth")
			return nil, false
		}
	}
}

func tryAuth(relay *nostr.Relay, challenge string, url string, waitTime int64, ev *EventWithPrivateKey) bool {
	event := nip42.CreateUnsignedAuthEvent(challenge, ev.Event.PubKey, url)
	err := event.Sign(ev.PrivateKey)
	if err != nil {
		log.Printf("[ERROR] Failed to sign event while trying to authenticate. PubKey: %s\n", ev.Event.PubKey)
		metrics.AppErrors.With(prometheus.Labels{"type": "REPLAY_AUTH"}).Inc()
		return false
	}

	// Set-up context with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(waitTime)*time.Millisecond)
	defer cancel()

	// Send the event by calling relay.Auth.
	// Returned status is either success, fail, or sent (if no reply given in the 3-second timeout).
	authStatus, err := relay.Auth(ctx, event)
	if err != nil {
		log.Printf("[ERROR] Failed while trying to authenticate after sending AUTH event. Error: %v\n", err)
		metrics.AppErrors.With(prometheus.Labels{"type": "REPLAY_AUTH"}).Inc()
		return false
	}

	log.Printf("[DEBUG] authenticated as %s: %s\n", ev.Event.PubKey, authStatus)
	if authStatus == nostr.PublishStatusSucceeded || authStatus == nostr.PublishStatusSent {
		return true
	}
	return false
}

func connectToRelay(url string) *nostr.Relay {
	relay, e := nostr.RelayConnect(context.Background(), url)
	if e != nil {
		log.Printf("[ERROR] Error while trying to connect with relay '%s': %v", url, e)
		metrics.AppErrors.With(prometheus.Labels{"type": "REPLAY_CONNECT"}).Inc()
		return nil
	}

	return relay
}
