package interfaces

import (
	"time"
)

//go:generate mockgen -package=mock -source=cacherules.go -destination=mock/cacherules.go

// CacheRulesLoader defines the interface for loading cache rules from configuration
type CacheRulesLoader interface {
	LoadCacheRules(rulesPath string) (CacheRulesService, error)
}

// CacheRulesService defines the interface for cache rules operations
type CacheRulesService interface {
	// CachePolicy interface methods
	Resolve(method string, params interface{}) TTL

	// Additional cache rules specific methods
	GetCacheInfo(chain, network string, request *JSONRPCRequest) (cacheType string, ttl time.Duration)
}

// CacheRulesConfig represents the structure for cache rules configuration
type CacheRulesConfig interface {
	// Validate checks if the configuration is valid
	Validate() error

	// GetTTLDefaults returns TTL defaults for a given key (chain, network, or "default")
	GetTTLDefaults(key string) (map[string]interface{}, bool)

	// GetCacheRule returns cache rule for a given method
	GetCacheRule(method string) (interface{}, bool)
}
