package interfaces

import (
	"go-proxy-cache/internal/models"
)

//go:generate mockgen -package=mock -source=cache.go -destination=mock/cache.go

// Cache interface defines the contract for cache implementations
type Cache interface {
	Get(key string) (*models.CacheEntry, bool)      // returns entry and found flag
	GetStale(key string) (*models.CacheEntry, bool) // stale-if-error, returns entry and found flag
	Set(key string, val []byte, ttl models.TTL)
	Delete(key string)
}

// LevelAwareCache interface extends Cache with level-aware operations
type LevelAwareCache interface {
	Cache
	GetWithLevel(key string) *models.CacheResult      // returns result with level information
	GetStaleWithLevel(key string) *models.CacheResult // stale-if-error with level information
}
