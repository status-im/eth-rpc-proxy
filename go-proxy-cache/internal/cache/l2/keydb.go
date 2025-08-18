package l2

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"go-proxy-cache/internal/config"
	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/models"
)

// Ensure KeyDBCache implements interfaces.Cache
var _ interfaces.Cache = (*KeyDBCache)(nil)

// CacheEntry represents an entry in the L2 cache with TTL information
type CacheEntry struct {
	Data      []byte `json:"data"`
	ExpiresAt int64  `json:"expires_at"`
	StaleAt   int64  `json:"stale_at"`
	CreatedAt int64  `json:"created_at"`
}

// KeyDBCache implements L2 cache using Redis/KeyDB
type KeyDBCache struct {
	client interfaces.KeyDbClient
	config *config.Config
	logger *zap.Logger
}

// NewKeyDBCache creates a new KeyDBCache instance with provided client
func NewKeyDBCache(cfg *config.Config, client interfaces.KeyDbClient, logger *zap.Logger) interfaces.Cache {
	return &KeyDBCache{
		client: client,
		config: cfg,
		logger: logger,
	}
}

// Get retrieves value from KeyDB cache with freshness information
func (kc *KeyDBCache) Get(key string) (val []byte, fresh bool, found bool) {
	ctx, cancel := context.WithTimeout(context.Background(), kc.config.GetReadTimeout())
	defer cancel()

	data, err := kc.client.Get(ctx, key).Result()
	if err != nil {
		kc.logger.Error("L2 cache get error", zap.String("key", key), zap.Error(err))
		return nil, false, false
	}

	var entry CacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		kc.logger.Error("Failed to unmarshal L2 cache entry", zap.String("key", key), zap.Error(err))
		kc.client.Del(context.Background(), key)
		return nil, false, false
	}

	now := time.Now().Unix()

	// Check if entry is expired
	if now > entry.ExpiresAt {
		kc.client.Del(context.Background(), key)
		return nil, false, false
	}

	// Check if entry is stale but still valid
	fresh = now <= entry.StaleAt
	return entry.Data, fresh, true
}

// GetStale retrieves value from KeyDB cache regardless of freshness
func (kc *KeyDBCache) GetStale(key string) (val []byte, found bool) {
	ctx, cancel := context.WithTimeout(context.Background(), kc.config.GetReadTimeout())
	defer cancel()

	data, err := kc.client.Get(ctx, key).Result()
	if err != nil {
		kc.logger.Error("L2 cache stale get error", zap.String("key", key), zap.Error(err))
		return nil, false
	}

	var entry CacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		kc.logger.Error("Failed to unmarshal L2 cache entry for stale get", zap.String("key", key), zap.Error(err))
		kc.client.Del(context.Background(), key)
		return nil, false
	}

	now := time.Now().Unix()

	// Check if entry is completely expired
	if now > entry.ExpiresAt {
		kc.client.Del(context.Background(), key)
		return nil, false
	}

	return entry.Data, true
}

// Set stores value in KeyDB cache with TTL
func (kc *KeyDBCache) Set(key string, val []byte, ttl models.TTL) {
	ctx, cancel := context.WithTimeout(context.Background(), kc.config.GetSendTimeout())
	defer cancel()

	now := time.Now().Unix()

	entry := CacheEntry{
		Data:      val,
		CreatedAt: now,
		StaleAt:   now + int64(ttl.Fresh.Seconds()),
		ExpiresAt: now + int64(ttl.Fresh.Seconds()) + int64(ttl.Stale.Seconds()),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		kc.logger.Error("Failed to marshal L2 cache entry", zap.String("key", key), zap.Error(err))
		return
	}

	// Set with total expiration time (Fresh TTL + Stale TTL)
	totalTTL := ttl.Fresh + ttl.Stale
	err = kc.client.Set(ctx, key, data, totalTTL).Err()
	if err != nil {
		kc.logger.Error("Failed to set L2 cache entry", zap.String("key", key), zap.Error(err))
		return
	}
}

// Delete removes entry from KeyDB cache
func (kc *KeyDBCache) Delete(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), kc.config.GetSendTimeout())
	defer cancel()

	err := kc.client.Del(ctx, key).Err()
	if err != nil {
		kc.logger.Error("Failed to delete L2 cache entry", zap.String("key", key), zap.Error(err))
		return
	}
}

// Close closes the KeyDB connection
func (kc *KeyDBCache) Close() error {
	return kc.client.Close()
}
