package service

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"go-proxy-cache/internal/cache"
	"go-proxy-cache/internal/cache/multi"
	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/models"
	"go-proxy-cache/internal/utils"
)

// CacheService handles cache operations with business logic
type CacheService struct {
	multiCache      interfaces.LevelAwareCache
	keyBuilder      interfaces.KeyBuilder
	cacheClassifier interfaces.CacheRulesClassifier
	logger          *zap.Logger
}

// NewCacheService creates a new cache service instance with MultiCache
func NewCacheService(l1Cache, l2Cache interfaces.Cache, cacheClassifier interfaces.CacheRulesClassifier, enablePropagation bool, logger *zap.Logger) *CacheService {
	// Create MultiCache with L1 and L2 caches
	caches := []interfaces.Cache{l1Cache, l2Cache}
	multiCache := multi.NewMultiCache(caches, logger, enablePropagation)

	return &CacheService{
		multiCache:      multiCache,
		keyBuilder:      cache.NewKeyBuilder(),
		cacheClassifier: cacheClassifier,
		logger:          logger,
	}
}

// GetResponse represents the result of a cache get operation
type GetResponse struct {
	Found      bool              `json:"found"`
	Fresh      bool              `json:"fresh"`
	Data       string            `json:"data,omitempty"`
	Key        string            `json:"key"`
	Bypass     bool              `json:"bypass"`
	CacheType  string            `json:"cache_type,omitempty"`
	TTL        int               `json:"ttl,omitempty"`
	CacheLevel models.CacheLevel `json:"cache_level,omitempty"`
}

// Get retrieves data from cache using MultiCache and ID fixing
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

	// Check if caching should be bypassed (TTL = 0)
	cacheInfo := s.cacheClassifier.GetTtl(chain, network, request)
	if cacheInfo.TTL == 0 {
		return &GetResponse{
			Found:      false,
			Fresh:      false,
			Data:       "",
			Key:        key,
			Bypass:     true,
			CacheType:  string(cacheInfo.CacheType),
			TTL:        int(cacheInfo.TTL.Seconds()),
			CacheLevel: models.CacheLevelMiss,
		}, nil
	}

	// Try MultiCache with level information
	result := s.multiCache.GetWithLevel(key)
	if result.Found && result.Entry != nil {
		// Fix response ID to match current request
		fixedData := utils.FixResponseID(string(result.Entry.Data), request.ID)

		return &GetResponse{
			Found:      true,
			Fresh:      result.Entry.IsFresh(),
			Data:       fixedData,
			Key:        key,
			Bypass:     false,
			CacheType:  string(cacheInfo.CacheType),
			TTL:        int(cacheInfo.TTL.Seconds()),
			CacheLevel: result.Level,
		}, nil
	}

	return &GetResponse{
		Found:      false,
		Fresh:      false,
		Data:       "",
		Key:        key,
		Bypass:     false,
		CacheType:  string(cacheInfo.CacheType),
		TTL:        int(cacheInfo.TTL.Seconds()),
		CacheLevel: models.CacheLevelMiss,
	}, nil
}

// Set stores data using MultiCache
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
		return nil
	}

	// Store using MultiCache (will store in all configured caches)
	ttlStruct := models.TTL{Fresh: ttl, Stale: staleTTL}
	s.multiCache.Set(key, []byte(responseData), ttlStruct)

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
