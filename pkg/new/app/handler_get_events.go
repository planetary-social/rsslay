package app

import (
	domain "github.com/piraces/rsslay/pkg/new/domain/nostr"
)

type HandlerGetEvents struct {
	eventStorage EventStorage
}

func NewHandlerGetEvents(eventStorage EventStorage) *HandlerGetEvents {
	return &HandlerGetEvents{eventStorage: eventStorage}
}

func (h *HandlerGetEvents) Handle(domainfilter domain.Filter) ([]domain.Event, error) {
	return h.eventStorage.GetEvents(domainfilter)
}

//func (h *HandlerGetEvents) tmptoremove(domainfilter domain.Filter) ([]domain.Event, error) {
//	var parsedEvents []nostr.Event
//	var eventsToReplay []replayer.EventWithPrivateKey
//
//	metrics.QueryEventsRequests.Inc()
//
//	filter := domainfilter.Libfilter()
//
//	if filter.IDs != nil || len(filter.Tags) > 0 {
//		return parsedEvents, nil
//	}
//
//	for _, pubkey := range filter.Authors {
//		parsedFeed, entity := events.GetParsedFeedForPubKey(pubkey, relayInstance.db, relayInstance.DeleteFailingFeeds, relayInstance.NitterInstances)
//
//		if parsedFeed == nil {
//			continue
//		}
//
//		converter := relayInstance.converterSelector.Select(parsedFeed)
//
//		if filter.Kinds == nil || slices.Contains(filter.Kinds, nostr.KindSetMetadata) {
//			evt := feed.EntryFeedToSetMetadata(pubkey, parsedFeed, entity.URL, relayInstance.EnableAutoNIP05Registration, relayInstance.DefaultProfilePictureUrl, relayInstance.MainDomainName)
//
//			if filter.Since != nil && evt.CreatedAt < *filter.Since {
//				continue
//			}
//			if filter.Until != nil && evt.CreatedAt > *filter.Until {
//				continue
//			}
//
//			_ = evt.Sign(entity.PrivateKey)
//			parsedEvents = append(parsedEvents, evt)
//			if relayInstance.ReplayToRelays {
//				eventsToReplay = append(eventsToReplay, replayer.EventWithPrivateKey{Event: &evt, PrivateKey: entity.PrivateKey})
//			}
//		}
//
//		if filter.Kinds == nil || slices.Contains(filter.Kinds, nostr.KindTextNote) || slices.Contains(filter.Kinds, feed.KindLongFormTextContent) {
//			var last uint32 = 0
//			for _, item := range parsedFeed.Items {
//				defaultCreatedAt := time.Unix(time.Now().Unix(), 0)
//				evt := converter.Convert(pubkey, item, parsedFeed, defaultCreatedAt, entity.URL)
//
//				// Feed need to have a date for each entry...
//				if evt.CreatedAt == nostr.Timestamp(defaultCreatedAt.Unix()) {
//					continue
//				}
//
//				if filter.Since != nil && evt.CreatedAt < *filter.Since {
//					continue
//				}
//				if filter.Until != nil && evt.CreatedAt > *filter.Until {
//					continue
//				}
//
//				_ = evt.Sign(entity.PrivateKey)
//
//				if !filter.Matches(&evt) {
//					continue
//				}
//
//				if evt.CreatedAt > nostr.Timestamp(int64(last)) {
//					last = uint32(evt.CreatedAt)
//				}
//
//				parsedEvents = append(parsedEvents, evt)
//				if relayInstance.ReplayToRelays {
//					eventsToReplay = append(eventsToReplay, replayer.EventWithPrivateKey{Event: &evt, PrivateKey: entity.PrivateKey})
//				}
//			}
//
//			relayInstance.lastEmitted.Store(entity.URL, last)
//		}
//	}
//
//	relayInstance.AttemptReplayEvents(eventsToReplay)
//
//	return parsedEvents, nil
//}
