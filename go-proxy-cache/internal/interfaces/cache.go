package interfaces

import (
	"time"
)

//go:generate mockgen -package=mock -source=cache.go -destination=mock/cache.go

// Cache interface defines the contract for cache implementations
type Cache interface {
	Get(key string) (val []byte, fresh bool, found bool)
	GetStale(key string) (val []byte, found bool) // stale-if-error
	Set(key string, val []byte, ttl time.Duration, staleTTL time.Duration)
	Delete(key string)
}
