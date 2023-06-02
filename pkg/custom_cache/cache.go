package custom_cache

import (
	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	store "github.com/eko/gocache/store/ristretto/v4"
	"log"
)

var MainCache *cache.Cache[string]

func InitializeCache() {
	ristrettoCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     400000000,
		BufferItems: 64,
		Metrics:     true,
	})
	if err != nil {
		log.Fatalf("[FATAL] failed to initialize internal custom_cache: %v", err)
	}
	ristrettoStore := store.NewRistretto(ristrettoCache)

	MainCache = cache.New[string](ristrettoStore)
}
