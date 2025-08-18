package interfaces

import (
	"go-proxy-cache/internal/models"
)

//go:generate mockgen -package=mock -source=cache.go -destination=mock/cache.go

// Cache interface defines the contract for cache implementations
type Cache interface {
	Get(key string) (val []byte, fresh bool, found bool)
	GetStale(key string) (val []byte, found bool) // stale-if-error
	Set(key string, val []byte, ttl models.TTL)
	Delete(key string)
}
