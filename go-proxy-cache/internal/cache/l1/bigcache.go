package l1

import (
	"context"
	"encoding/json"
	"time"

	"github.com/allegro/bigcache/v3"
	"go.uber.org/zap"

	"go-proxy-cache/internal/config"
	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/models"
)

// Ensure BigCache implements interfaces.Cache
var _ interfaces.Cache = (*BigCache)(nil)

// BigCache implements L1 cache using BigCache
type BigCache struct {
	cache  *bigcache.BigCache
	logger *zap.Logger
}

// NewBigCache creates a new BigCache instance
func NewBigCache(bigcacheCfg *config.BigCacheConfig, logger *zap.Logger) (interfaces.Cache, error) {
	config := bigcache.DefaultConfig(10 * time.Minute) // Default eviction time
	config.HardMaxCacheSize = bigcacheCfg.Size         // Size in MB
	config.Verbose = false
	config.MaxEntrySize = 1024 * 1024 // 1MB max entry size

	cache, err := bigcache.New(context.Background(), config)
	if err != nil {
		return nil, err
	}

	return &BigCache{
		cache:  cache,
		logger: logger,
	}, nil
}

// Get retrieves value from cache with freshness information
func (bc *BigCache) Get(key string) (*models.CacheEntry, bool) {
	data, err := bc.cache.Get(key)
	if err != nil {
		return nil, false
	}

	var entry models.CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		bc.cache.Delete(key) // Remove corrupted entry
		return nil, false
	}

	// Check if entry is expired
	if entry.IsExpired() {
		bc.cache.Delete(key)
		return nil, false
	}

	return &entry, true
}

// GetStale retrieves value from cache regardless of freshness (for stale-if-error)
func (bc *BigCache) GetStale(key string) (*models.CacheEntry, bool) {
	data, err := bc.cache.Get(key)
	if err != nil {
		return nil, false
	}

	var entry models.CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		bc.cache.Delete(key) // Remove corrupted entry
		return nil, false
	}

	// Check if entry is completely expired (beyond stale time)
	if entry.IsExpired() {
		bc.cache.Delete(key)
		return nil, false
	}

	return &entry, true
}

// Set stores value in cache with TTL
func (bc *BigCache) Set(key string, val []byte, ttl models.TTL) {
	now := time.Now().Unix()

	entry := models.CacheEntry{
		Data:      val,
		CreatedAt: now,
		StaleAt:   now + int64(ttl.Fresh.Seconds()),
		ExpiresAt: now + int64(ttl.Fresh.Seconds()) + int64(ttl.Stale.Seconds()),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		bc.logger.Error("Failed to marshal cache entry", zap.String("key", key), zap.Error(err))
		return
	}

	err = bc.cache.Set(key, data)
	if err != nil {
		bc.logger.Error("Failed to set cache entry", zap.String("key", key), zap.Error(err))
		return
	}
}

// Delete removes entry from cache
func (bc *BigCache) Delete(key string) {
	err := bc.cache.Delete(key)
	if err != nil {
		return
	}
}

// Close closes the cache
func (bc *BigCache) Close() error {
	return bc.cache.Close()
}
