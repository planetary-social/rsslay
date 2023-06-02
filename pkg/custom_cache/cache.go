package custom_cache

import (
	"github.com/allegro/bigcache"
	"github.com/eko/gocache/lib/v4/cache"
	bigcache_store "github.com/eko/gocache/store/bigcache/v4"
	"time"
)

var MainCache *cache.Cache[[]byte]
var Initialized = false

func InitializeCache() {
	bigcacheClient, _ := bigcache.NewBigCache(bigcache.DefaultConfig(60 * time.Minute))
	bigcacheStore := bigcache_store.NewBigcache(bigcacheClient)

	MainCache = cache.New[[]byte](bigcacheStore)
	Initialized = true
}
