package cache_rules

import (
	"fmt"
	"time"

	"go.uber.org/zap"

	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/models"
)

// CacheConfig implements the CacheRulesConfig interface
type CacheConfig struct {
	config *CacheRulesConfig
	logger *zap.Logger
}

// Ensure CacheConfig implements the CacheRulesConfig interface
var _ interfaces.CacheRulesConfig = (*CacheConfig)(nil)

// NewCacheConfig creates a new CacheConfig instance
func NewCacheConfig(config *CacheRulesConfig, logger *zap.Logger) *CacheConfig {
	if config == nil {
		panic("config cannot be nil")
	}
	return &CacheConfig{
		config: config,
		logger: logger,
	}
}

// GetTtlForCacheType implements CacheRulesConfig interface
func (cr *CacheConfig) GetTtlForCacheType(chain, network string, cacheType models.CacheType) time.Duration {
	if len(cr.config.ChainsTTLDefaults) == 0 {
		return cr.getFallbackTTL(cacheType)
	}

	// Try network-specific config first
	if chain != "" && network != "" {
		networkKey := fmt.Sprintf("%s:%s", chain, network)
		if ttl := cr.lookupTTL(networkKey, cacheType); ttl > 0 {
			return ttl
		}
	}

	// Try chain-specific config
	if chain != "" {
		if ttl := cr.lookupTTL(chain, cacheType); ttl > 0 {
			return ttl
		}
	}

	// Fall back to default config
	if ttl := cr.lookupTTL("default", cacheType); ttl > 0 {
		return ttl
	}

	return 0
}

// lookupTTL looks up TTL value from config
func (cr *CacheConfig) lookupTTL(key string, cacheType models.CacheType) time.Duration {
	ttlDefaults, ok := cr.config.ChainsTTLDefaults[key]
	if !ok {
		return 0
	}

	if duration, exists := ttlDefaults[cacheType]; exists {
		return duration
	}

	return 0
}

// GetAllMethods returns all configured RPC methods from cache rules
func (cr *CacheConfig) GetAllMethods() []string {
	methods := make([]string, 0, len(cr.config.CacheRules))
	for method := range cr.config.CacheRules {
		methods = append(methods, method)
	}
	return methods
}

// getFallbackTTL provides fallback TTL values when config is not available
func (cr *CacheConfig) getFallbackTTL(cacheType models.CacheType) time.Duration {
	fallbackTTLs := map[models.CacheType]time.Duration{
		models.CacheTypePermanent: 24 * time.Hour,
		models.CacheTypeShort:     5 * time.Second,
		models.CacheTypeMinimal:   0,
	}

	if ttl, ok := fallbackTTLs[cacheType]; ok {
		return ttl
	}

	return 0
}

func (cr *CacheConfig) GetCacheTypeForMethod(method string) models.CacheType {
	if method == "" {
		if cr.logger != nil {
			cr.logger.Warn("Empty method provided, returning minimal cache type")
		}
		return models.CacheTypeNone
	}

	if cr.config.CacheRules == nil {
		if cr.logger != nil {
			cr.logger.Warn("CacheRules is nil, returning minimal cache type")
		}
		return models.CacheTypeNone
	}

	// Look up cache type for the method
	if cacheType, exists := cr.config.CacheRules[method]; exists {
		return cacheType
	}

	// Default to minimal cache type if method not found
	if cr.logger != nil {
		cr.logger.Debug("Method not found in cache rules, returning minimal cache type",
			zap.String("method", method))
	}
	return models.CacheTypeNone
}
