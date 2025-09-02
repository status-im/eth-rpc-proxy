package multi

import (
	"go.uber.org/zap"

	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/models"
)

// Ensure MultiCache implements interfaces.Cache and interfaces.LevelAwareCache
var _ interfaces.Cache = (*MultiCache)(nil)
var _ interfaces.LevelAwareCache = (*MultiCache)(nil)

// MultiCache implements a composite cache that tries multiple cache implementations
// It attempts to get/set values through an array of cache interfaces in order
type MultiCache struct {
	caches            []interfaces.Cache
	logger            *zap.Logger
	enablePropagation bool
}

// NewMultiCache creates a new MultiCache instance with provided cache implementations
func NewMultiCache(caches []interfaces.Cache, logger *zap.Logger, enablePropagation bool) interfaces.LevelAwareCache {
	return &MultiCache{
		caches:            caches,
		logger:            logger,
		enablePropagation: enablePropagation,
	}
}

// Get retrieves value from the first available cache that has the key
// It uses GetWithLevel internally to avoid code duplication
func (mc *MultiCache) Get(key string) (*models.CacheEntry, bool) {
	result := mc.GetWithLevel(key)
	return result.Entry, result.Found
}

// GetStale retrieves stale value from the first available cache that has the key
// It uses GetStaleWithLevel internally to avoid code duplication
func (mc *MultiCache) GetStale(key string) (*models.CacheEntry, bool) {
	result := mc.GetStaleWithLevel(key)
	return result.Entry, result.Found
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

// GetWithLevel retrieves value from cache with level information
func (mc *MultiCache) GetWithLevel(key string) *models.CacheResult {
	if len(mc.caches) == 0 {
		mc.logger.Warn("No caches available for get operation", zap.String("key", key))
		return &models.CacheResult{
			Entry: nil,
			Found: false,
			Level: models.CacheLevelMiss,
		}
	}

	for i, cache := range mc.caches {
		entry, found := cache.Get(key)
		if found {
			// If found in a later cache (i > 0) and propagation is enabled, propagate to earlier caches
			if i > 0 && mc.enablePropagation {
				mc.propagateToEarlierCaches(key, entry, i)
			}

			// Determine cache level based on index
			level := models.CacheLevelFromIndex(i)

			return &models.CacheResult{
				Entry: entry,
				Found: true,
				Level: level,
			}
		}
	}

	return &models.CacheResult{
		Entry: nil,
		Found: false,
		Level: models.CacheLevelMiss,
	}
}

// GetStaleWithLevel retrieves stale value from cache with level information
func (mc *MultiCache) GetStaleWithLevel(key string) *models.CacheResult {
	if len(mc.caches) == 0 {
		return &models.CacheResult{
			Entry: nil,
			Found: false,
			Level: models.CacheLevelMiss,
		}
	}

	for i, cache := range mc.caches {
		entry, found := cache.GetStale(key)
		if found {
			// If found in a later cache (i > 0) and propagation is enabled, propagate to earlier caches
			if i > 0 && mc.enablePropagation {
				mc.propagateToEarlierCaches(key, entry, i)
			}

			// Determine cache level based on index
			level := models.CacheLevelFromIndex(i)

			return &models.CacheResult{
				Entry: entry,
				Found: true,
				Level: level,
			}
		}
	}

	return &models.CacheResult{
		Entry: nil,
		Found: false,
		Level: models.CacheLevelMiss,
	}
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
