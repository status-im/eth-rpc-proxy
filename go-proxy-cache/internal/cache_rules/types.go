package cache_rules

import (
	"time"

	"github.com/status-im/proxy-common/models"
)

// TTLDefaults represents TTL settings for different cache types
type TTLDefaults map[models.CacheType]time.Duration

// CacheRulesConfig represents the cache rules configuration
type CacheRulesConfig struct {
	ChainsTTLDefaults map[string]TTLDefaults      `yaml:"ttl_defaults"`
	SkipNullCache     []string                    `yaml:"skip_null_cache"`
	CacheRules        map[string]models.CacheType `yaml:"cache_rules"`
}
