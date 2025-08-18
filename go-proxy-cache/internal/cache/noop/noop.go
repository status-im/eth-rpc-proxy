package noop

import (
	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/models"
)

// Ensure NoOpCache implements interfaces.Cache
var _ interfaces.Cache = (*NoOpCache)(nil)

// NoOpCache is a no-operation cache implementation for disabled caches
type NoOpCache struct{}

// NewNoOpCache creates a new no-operation cache instance
func NewNoOpCache() interfaces.Cache {
	return &NoOpCache{}
}

// Get always returns cache miss
func (n *NoOpCache) Get(key string) (val []byte, fresh bool, found bool) {
	return nil, false, false
}

// GetStale always returns cache miss
func (n *NoOpCache) GetStale(key string) (val []byte, found bool) {
	return nil, false
}

// Set does nothing
func (n *NoOpCache) Set(key string, val []byte, ttl models.TTL) {
	// No-op
}

// Delete does nothing
func (n *NoOpCache) Delete(key string) {
	// No-op
}
