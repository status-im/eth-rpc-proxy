package service

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"go-proxy-cache/internal/cache"
	"go-proxy-cache/internal/cache/multi"
	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/metrics"
	"go-proxy-cache/internal/models"
	"go-proxy-cache/internal/utils"
)

// CacheService handles cache operations with business logic
type CacheService struct {
	multiCache      interfaces.LevelAwareCache
	keyBuilder      interfaces.KeyBuilder
	cacheClassifier interfaces.CacheRulesClassifier
	logger          *zap.Logger
	l1Cache         interfaces.Cache // Keep reference to L1 cache for metrics
}

// NewCacheService creates a new cache service instance with MultiCache
func NewCacheService(l1Cache, l2Cache interfaces.Cache, cacheClassifier interfaces.CacheRulesClassifier, enablePropagation bool, logger *zap.Logger) *CacheService {
	// Create MultiCache with L1 and L2 caches
	caches := []interfaces.Cache{l1Cache, l2Cache}
	multiCache := multi.NewMultiCache(caches, logger, enablePropagation)

	service := &CacheService{
		multiCache:      multiCache,
		keyBuilder:      cache.NewKeyBuilder(),
		cacheClassifier: cacheClassifier,
		logger:          logger,
		l1Cache:         l1Cache,
	}

	return service
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
	cacheType := string(cacheInfo.CacheType)

	if cacheInfo.TTL == 0 {
		return &GetResponse{
			Found:      false,
			Fresh:      false,
			Data:       "",
			Key:        key,
			Bypass:     true,
			CacheType:  cacheType,
			TTL:        int(cacheInfo.TTL.Seconds()),
			CacheLevel: models.CacheLevelMiss,
		}, nil
	}

	// Start timing cache get operation
	timer := metrics.TimeCacheGetOperation("multi")
	defer timer()

	// Try MultiCache with level information
	result := s.multiCache.GetWithLevel(key)
	if result.Found && result.Entry != nil {
		// Record cache hit with level information
		var level string
		switch result.Level {
		case models.CacheLevelL1:
			level = "l1"
		case models.CacheLevelL2:
			level = "l2"
		default:
			level = "unknown"
		}
		// Calculate item age for TTL effectiveness analysis
		itemAge := time.Duration(time.Now().Unix()-result.Entry.CreatedAt) * time.Second
		metrics.RecordCacheHit(cacheType, level, chain, network, request.Method, itemAge)

		// Record bytes read
		metrics.RecordCacheBytesRead(level, cacheType, chain, network, len(result.Entry.Data))

		// Fix response ID to match current request
		fixedData := utils.FixResponseID(string(result.Entry.Data), request.ID)

		return &GetResponse{
			Found:      true,
			Fresh:      result.Entry.IsFresh(),
			Data:       fixedData,
			Key:        key,
			Bypass:     false,
			CacheType:  cacheType,
			TTL:        int(cacheInfo.TTL.Seconds()),
			CacheLevel: result.Level,
		}, nil
	}

	// Record cache miss
	metrics.RecordCacheMiss(cacheType, chain, network, request.Method)

	return &GetResponse{
		Found:      false,
		Fresh:      false,
		Data:       "",
		Key:        key,
		Bypass:     false,
		CacheType:  cacheType,
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

	// Skip caching null results for methods configured in skip_null_cache
	if s.cacheClassifier.ShouldSkipNullCache(request.Method) && utils.IsNullResult(responseData) {
		s.logger.Debug("Skipping cache for null result",
			zap.String("method", request.Method),
			zap.String("chain", chain),
			zap.String("network", network))
		return nil
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

	// Time the set operation
	timer := metrics.TimeCacheOperation("set", "multi")
	s.multiCache.Set(key, []byte(responseData), ttlStruct)
	timer()

	// Record cache set operation and bytes written for each level
	cacheInfo := s.cacheClassifier.GetTtl(chain, network, request)
	cacheType := string(cacheInfo.CacheType)
	dataSize := len(responseData)

	// Record for both L1 and L2 (since MultiCache writes to both)
	metrics.RecordCacheSet("l1", cacheType, chain, network, dataSize)
	metrics.RecordCacheSet("l2", cacheType, chain, network, dataSize)

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
