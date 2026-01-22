package config

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// BigCacheConfig represents BigCache (L1) configuration
type BigCacheConfig struct {
	Enabled      bool `yaml:"enabled"`
	Size         int  `yaml:"size"`           // Size in MB
	MaxEntrySize int  `yaml:"max_entry_size"` // Max entry size in bytes
	Shards       int  `yaml:"shards"`         // Number of shards (must be power of 2)
}

// KeyDBConfig represents KeyDB (L2) cache configuration
type KeyDBConfig struct {
	Enabled    bool             `yaml:"enabled"`
	Connection ConnectionConfig `yaml:"connection"`
	Keepalive  KeepaliveConfig  `yaml:"keepalive"`
	Cache      CacheConfig      `yaml:"cache"`
}

// ConnectionConfig represents connection settings
type ConnectionConfig struct {
	ConnectTimeout time.Duration `yaml:"connect_timeout"`
	SendTimeout    time.Duration `yaml:"send_timeout"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
}

// KeepaliveConfig represents connection pool settings
type KeepaliveConfig struct {
	PoolSize       int           `yaml:"pool_size"` // max connections in pool
	MaxIdleTimeout time.Duration `yaml:"max_idle_timeout"`
}

// CacheConfig represents cache-specific settings
type CacheConfig struct {
	DefaultTTL time.Duration `yaml:"default_ttl"`
	MaxTTL     time.Duration `yaml:"max_ttl"`
}

// MultiCacheConfig represents multi-cache configuration
type MultiCacheConfig struct {
	EnablePropagation bool `yaml:"enable_propagation"`
}

// Config represents the main configuration structure
type Config struct {
	BigCache   BigCacheConfig   `yaml:"bigcache"`
	KeyDB      KeyDBConfig      `yaml:"keydb"`
	MultiCache MultiCacheConfig `yaml:"multi_cache"`
}

// LoadConfig loads configuration from file path
func LoadConfig(configPath string, logger *zap.Logger) (*Config, error) {
	logger.Info("Loading configuration", zap.String("path", configPath))

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() { _ = file.Close() }()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode YAML config: %w", err)
	}

	// Apply defaults
	config.applyDefaults()
	return &config, nil
}

// applyDefaults sets default values for missing configuration
func (c *Config) applyDefaults() {
	// BigCache defaults
	if c.BigCache.Size == 0 {
		c.BigCache.Size = 100 // 100MB default
	}
	if c.BigCache.MaxEntrySize == 0 {
		c.BigCache.MaxEntrySize = 1048576 // 1MB default
	}
	if c.BigCache.Shards == 0 {
		c.BigCache.Shards = 256 // Reduced from default 1024 to increase shard size
	}

	// KeyDB connection defaults
	if c.KeyDB.Connection.ConnectTimeout == 0 {
		c.KeyDB.Connection.ConnectTimeout = 1000 * time.Millisecond
	}
	if c.KeyDB.Connection.SendTimeout == 0 {
		c.KeyDB.Connection.SendTimeout = 1000 * time.Millisecond
	}
	if c.KeyDB.Connection.ReadTimeout == 0 {
		c.KeyDB.Connection.ReadTimeout = 1000 * time.Millisecond
	}

	// KeyDB keepalive defaults
	if c.KeyDB.Keepalive.PoolSize == 0 {
		c.KeyDB.Keepalive.PoolSize = 10
	}
	if c.KeyDB.Keepalive.MaxIdleTimeout == 0 {
		c.KeyDB.Keepalive.MaxIdleTimeout = 10000 * time.Millisecond
	}

	// KeyDB cache defaults
	if c.KeyDB.Cache.DefaultTTL == 0 {
		c.KeyDB.Cache.DefaultTTL = 3600 * time.Second
	}
	if c.KeyDB.Cache.MaxTTL == 0 {
		c.KeyDB.Cache.MaxTTL = 86400 * time.Second
	}
}
