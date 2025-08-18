package interfaces

import "time"

//go:generate mockgen -package=mock -source=cache_rules_config.go -destination=mock/cache_rules_config.go

// CacheRulesConfig defines the interface for getting TTL values for different cache types
type CacheRulesConfig interface {
	// GetTtlForKey returns TTL values for short, permanent, and minimal cache types
	// for a given chain and network combination
	GetTtlForKey(chain, network, key string) time.Duration
}
