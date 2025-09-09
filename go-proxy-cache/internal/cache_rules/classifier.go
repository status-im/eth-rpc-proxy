package cache_rules

import (
	"go.uber.org/zap"

	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/models"
)

// Classifier implements the CacheRulesClassifier interface
type Classifier struct {
	logger    *zap.Logger
	configTTL interfaces.CacheRulesConfig
}

// Ensure Classifier implements the CacheRulesClassifier interface
var _ interfaces.CacheRulesClassifier = (*Classifier)(nil)

// NewClassifier creates a new Classifier instance
func NewClassifier(logger *zap.Logger, configTTL interfaces.CacheRulesConfig) *Classifier {
	return &Classifier{
		logger:    logger,
		configTTL: configTTL,
	}
}

// GetTtl implements CacheRulesClassifier interface
func (c *Classifier) GetTtl(chain, network string, request *models.JSONRPCRequest) models.CacheInfo {
	if request == nil || request.Method == "" {
		return models.CacheInfo{TTL: 0, CacheType: "none"}
	}

	cacheType := c.configTTL.GetCacheTypeForMethod(request.Method)
	if cacheType == models.CacheTypeNone {
		return models.CacheInfo{TTL: 0, CacheType: models.CacheTypeNone}
	}

	ttl := c.configTTL.GetTtlForCacheType(chain, network, cacheType)
	if ttl == 0 {
		return models.CacheInfo{TTL: 0, CacheType: models.CacheTypeNone}
	}

	return models.CacheInfo{TTL: ttl, CacheType: cacheType}
}
