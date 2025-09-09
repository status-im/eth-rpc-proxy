package interfaces

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

//go:generate mockgen -source=keydb_client.go -destination=mock/keydb_client.go -package=mock

// KeyDbClient defines the interface for KeyDB/Redis client operations
type KeyDbClient interface {
	// Get retrieves a value by key
	Get(ctx context.Context, key string) *redis.StringCmd

	// Set stores a value with expiration
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd

	// Del deletes one or more keys
	Del(ctx context.Context, keys ...string) *redis.IntCmd

	// Ping tests connectivity
	Ping(ctx context.Context) *redis.StatusCmd

	// Close closes the client connection
	Close() error
}
