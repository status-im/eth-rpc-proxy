package cache_rules

import (
	"time"

	"go-proxy-cache/internal/models"
)

// TTLDefaults represents TTL settings for different cache types
type TTLDefaults map[models.CacheType]time.Duration

// CacheRulesConfig represents the cache rules configuration
type CacheRulesConfig struct {
	ChainsTTLDefaults map[string]TTLDefaults      `yaml:"ttl_defaults"`
	CacheRules        map[string]models.CacheType `yaml:"cache_rules"`
}
