package multi

import (
	"go.uber.org/zap"

	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/models"
)

// Ensure MultiCache implements interfaces.Cache
var _ interfaces.Cache = (*MultiCache)(nil)

// MultiCache implements a composite cache that tries multiple cache implementations
// It attempts to get/set values through an array of cache interfaces in order
type MultiCache struct {
	caches []interfaces.Cache
	logger *zap.Logger
}

// NewMultiCache creates a new MultiCache instance with provided cache implementations
func NewMultiCache(caches []interfaces.Cache, logger *zap.Logger) interfaces.Cache {
	return &MultiCache{
		caches: caches,
		logger: logger,
	}
}

// Get retrieves value from the first available cache that has the key
// It tries each cache in order until it finds a value or exhausts all caches
func (mc *MultiCache) Get(key string) (val []byte, fresh bool, found bool) {
	if len(mc.caches) == 0 {
		mc.logger.Warn("No caches available for get operation", zap.String("key", key))
		return nil, false, false
	}

	for _, cache := range mc.caches {
		val, fresh, found := cache.Get(key)
		if found {
			return val, fresh, true
		}
	}
	return nil, false, false
}

// GetStale retrieves stale value from the first available cache that has the key
func (mc *MultiCache) GetStale(key string) (val []byte, found bool) {
	if len(mc.caches) == 0 {
		return nil, false
	}

	for _, cache := range mc.caches {
		val, found := cache.GetStale(key)
		if found {
			return val, true
		}
	}
	return nil, false
}

// Set stores value in all available caches
func (mc *MultiCache) Set(key string, val []byte, ttl models.TTL) {
	if len(mc.caches) == 0 {
		mc.logger.Warn("No caches available for set operation", zap.String("key", key))
		return
	}

	for _, cache := range mc.caches {
		cache.Set(key, val, ttl)
	}
}

// Delete removes entry from all available caches
func (mc *MultiCache) Delete(key string) {
	if len(mc.caches) == 0 {
		mc.logger.Warn("No caches available for delete operation", zap.String("key", key))
		return
	}

	for _, cache := range mc.caches {
		cache.Delete(key)
	}
}

// GetCacheCount returns the number of caches in the multi-cache
func (mc *MultiCache) GetCacheCount() int {
	return len(mc.caches)
}
