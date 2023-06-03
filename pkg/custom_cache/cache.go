package custom_cache

import (
	"context"
	"github.com/allegro/bigcache"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	bigcache_store "github.com/eko/gocache/store/bigcache/v4"
	redis_store "github.com/eko/gocache/store/redis/v4"
	"github.com/piraces/rsslay/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"log"
	"time"
)

type RedisConfig struct {
	Address  string
	Username string
	Password string
}

var MainCache *cache.Cache[[]byte]
var MainCacheRedis *cache.Cache[string]
var Initialized = false
var RedisConfiguration *RedisConfig

func InitializeCache() {
	if RedisConfiguration != nil {
		initializeRedisCache()
		log.Printf("[INFO] Using Redis cache ADDRESS %s:\n\n", RedisConfiguration.Address)
		Initialized = true
		return
	}
	log.Printf("[INFO] Using default memory cache\n\n")
	initializeBigCache()
	Initialized = true
}

func Get(key string) (string, error) {
	if !Initialized {
		InitializeCache()
	}
	if MainCacheRedis != nil {
		return getFromRedis(key)
	}
	return getFromBigCache(key)
}

func Set(key string, value string) error {
	if !Initialized {
		InitializeCache()
	}
	if MainCacheRedis != nil {
		return setToRedis(key, value)
	}

	return setToBigCache(key, value)
}

func initializeBigCache() {
	bigcacheClient, _ := bigcache.NewBigCache(bigcache.DefaultConfig(30 * time.Minute))
	bigcacheStore := bigcache_store.NewBigcache(bigcacheClient)

	MainCache = cache.New[[]byte](bigcacheStore)
}

func initializeRedisCache() {
	redisStore := redis_store.NewRedis(redis.NewClient(&redis.Options{
		Addr:     RedisConfiguration.Address,
		Username: RedisConfiguration.Username,
		Password: RedisConfiguration.Password,
	}))

	MainCacheRedis = cache.New[string](redisStore)
}

func getFromRedis(key string) (string, error) {
	value, err := MainCacheRedis.Get(context.Background(), key)
	switch err {
	case nil:
		log.Printf("[DEBUG] Get the key '%s' from the redis cache.", key)
	case redis.Nil:
		log.Printf("[DEBUG] Failed to find the key '%s' from the redis cache.", key)
	default:
		log.Printf("[ERROR] Failed to get the value from the redis cache with key '%s': %v", key, err)
		metrics.AppErrors.With(prometheus.Labels{"type": "CACHE"}).Inc()
	}
	return value, err
}

func setToRedis(key string, value string) error {
	return MainCacheRedis.Set(context.Background(), key, value, store.WithExpiration(30*time.Minute))
}

func getFromBigCache(key string) (string, error) {
	valueBytes, err := MainCache.Get(context.Background(), key)
	if err != nil {
		log.Printf("[DEBUG] Failed to find the key '%s' from the cache. Error: %v", key, err)
		return "", err
	}
	return string(valueBytes), nil
}

func setToBigCache(key string, value string) error {
	return MainCache.Set(context.Background(), key, []byte(value))
}
