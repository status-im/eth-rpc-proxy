package service

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"go-proxy-cache/internal/cache"
	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/models"
	"go-proxy-cache/internal/utils"
)

// CacheService handles cache operations with business logic
type CacheService struct {
	l1Cache         interfaces.Cache
	l2Cache         interfaces.Cache
	keyBuilder      interfaces.KeyBuilder
	cacheClassifier interfaces.CacheRulesClassifier
	logger          *zap.Logger
}

// NewCacheService creates a new cache service instance
func NewCacheService(l1Cache, l2Cache interfaces.Cache, cacheClassifier interfaces.CacheRulesClassifier, logger *zap.Logger) *CacheService {
	return &CacheService{
		l1Cache:         l1Cache,
		l2Cache:         l2Cache,
		keyBuilder:      cache.NewKeyBuilder(),
		cacheClassifier: cacheClassifier,
		logger:          logger,
	}
}

// GetResponse represents the result of a cache get operation
type GetResponse struct {
	Found bool   `json:"found"`
	Fresh bool   `json:"fresh"`
	Data  string `json:"data,omitempty"`
	Key   string `json:"key"`
}

// Get retrieves data from cache with L1/L2 fallback and ID fixing
func (s *CacheService) Get(chain, network, rawBody string) (*GetResponse, error) {
	// Parse JSON-RPC request from raw body
	request, err := utils.ParseJSONRPCRequest(rawBody)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON-RPC request: %w", err)
	}

	// Build cache key
	key, err := s.keyBuilder.Build(chain, network, request)
	if err != nil {
		return nil, fmt.Errorf("failed to build cache key: %w", err)
	}

	// Try L1 cache first
	entry, found := s.l1Cache.Get(key)
	if found {
		// Fix response ID to match current request
		fixedData := utils.FixResponseID(string(entry.Data), request.ID)

		s.logger.Debug("L1 cache hit",
			zap.String("key", key),
			zap.Bool("fresh", entry.IsFresh()))

		return &GetResponse{
			Found: true,
			Fresh: entry.IsFresh(),
			Data:  fixedData,
			Key:   key,
		}, nil
	}

	// Try L2 cache
	entry, found = s.l2Cache.Get(key)
	if found {
		// Store in L1 cache for future requests with remaining TTL
		remainingTTL := entry.RemainingTTL()
		s.l1Cache.Set(key, entry.Data, remainingTTL)

		// Fix response ID to match current request
		fixedData := utils.FixResponseID(string(entry.Data), request.ID)

		s.logger.Debug("L2 cache hit, stored in L1",
			zap.String("key", key),
			zap.Bool("fresh", entry.IsFresh()))

		return &GetResponse{
			Found: true,
			Fresh: entry.IsFresh(),
			Data:  fixedData,
			Key:   key,
		}, nil
	}

	// Cache miss
	s.logger.Debug("Cache miss",
		zap.String("key", key))

	return &GetResponse{
		Found: false,
		Fresh: false,
		Data:  "",
		Key:   key,
	}, nil
}

// Set stores data in both L1 and L2 caches
func (s *CacheService) Set(chain, network, rawBody, responseData string, customTTL, customStaleTTL *int) error {
	// Parse JSON-RPC request from raw body
	request, err := utils.ParseJSONRPCRequest(rawBody)
	if err != nil {
		return fmt.Errorf("invalid JSON-RPC request: %w", err)
	}

	// Build cache key
	key, err := s.keyBuilder.Build(chain, network, request)
	if err != nil {
		return fmt.Errorf("failed to build cache key: %w", err)
	}

	// Determine TTL
	var ttl, staleTTL time.Duration
	if customTTL != nil && customStaleTTL != nil {
		// Use provided TTL values
		ttl = time.Duration(*customTTL) * time.Second
		staleTTL = time.Duration(*customStaleTTL) * time.Second
	} else {
		// Get TTL from cache classifier
		cacheInfo := s.cacheClassifier.GetTtl(chain, network, request)
		ttl = cacheInfo.TTL
		staleTTL = cacheInfo.TTL / 10 // stale TTL is 10% of fresh TTL
	}

	// Don't cache if TTL is 0
	if ttl == 0 {
		s.logger.Debug("TTL is 0, not caching",
			zap.String("key", key),
			zap.String("method", request.Method))
		return nil
	}

	// Store in both L1 and L2 caches
	ttlStruct := models.TTL{Fresh: ttl, Stale: staleTTL}
	s.l1Cache.Set(key, []byte(responseData), ttlStruct)
	s.l2Cache.Set(key, []byte(responseData), ttlStruct)

	s.logger.Debug("Stored in cache",
		zap.String("key", key),
		zap.String("method", request.Method),
		zap.Duration("fresh_ttl", ttl),
		zap.Duration("stale_ttl", staleTTL))

	return nil
}

// GetCacheInfo returns cache type and TTL for a request
func (s *CacheService) GetCacheInfo(chain, network, rawBody string) (string, int, error) {
	// Parse JSON-RPC request from raw body
	request, err := utils.ParseJSONRPCRequest(rawBody)
	if err != nil {
		return "", 0, fmt.Errorf("invalid JSON-RPC request: %w", err)
	}

	// Get cache info using cache classifier
	cacheInfo := s.cacheClassifier.GetTtl(chain, network, request)

	return string(cacheInfo.CacheType), int(cacheInfo.TTL.Seconds()), nil
}
