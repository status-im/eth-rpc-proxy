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
	caches            []interfaces.Cache
	logger            *zap.Logger
	enablePropagation bool
}

// NewMultiCache creates a new MultiCache instance with provided cache implementations
func NewMultiCache(caches []interfaces.Cache, logger *zap.Logger, enablePropagation bool) interfaces.Cache {
	return &MultiCache{
		caches:            caches,
		logger:            logger,
		enablePropagation: enablePropagation,
	}
}

// Get retrieves value from the first available cache that has the key
// It tries each cache in order until it finds a value or exhausts all caches
// When found in a later cache, it propagates the entry to earlier caches with adjusted TTL
func (mc *MultiCache) Get(key string) (*models.CacheEntry, bool) {
	if len(mc.caches) == 0 {
		mc.logger.Warn("No caches available for get operation", zap.String("key", key))
		return nil, false
	}

	for i, cache := range mc.caches {
		entry, found := cache.Get(key)
		if found {
			// If found in a later cache (i > 0) and propagation is enabled, propagate to earlier caches
			if i > 0 && mc.enablePropagation {
				mc.propagateToEarlierCaches(key, entry, i)
			}
			return entry, true
		}
	}
	return nil, false
}

// GetStale retrieves stale value from the first available cache that has the key
// When found in a later cache, it propagates the entry to earlier caches with adjusted TTL
func (mc *MultiCache) GetStale(key string) (*models.CacheEntry, bool) {
	if len(mc.caches) == 0 {
		return nil, false
	}

	for i, cache := range mc.caches {
		entry, found := cache.GetStale(key)
		if found {
			// If found in a later cache (i > 0) and propagation is enabled, propagate to earlier caches
			if i > 0 && mc.enablePropagation {
				mc.propagateToEarlierCaches(key, entry, i)
			}
			return entry, true
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

// propagateToEarlierCaches propagates a cache entry to earlier caches with adjusted TTL
func (mc *MultiCache) propagateToEarlierCaches(key string, entry *models.CacheEntry, foundAtIndex int) {
	if entry == nil || entry.IsExpired() {
		return
	}
	remainingTTL := entry.RemainingTTL()

	// Only propagate if there's meaningful time left
	if remainingTTL.Fresh <= 0 && remainingTTL.Stale <= 0 {
		return
	}

	for i := 0; i < foundAtIndex; i++ {
		mc.caches[i].Set(key, entry.Data, remainingTTL)
	}
}
