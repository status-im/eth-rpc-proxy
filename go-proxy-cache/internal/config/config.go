package config

import (
	"fmt"
	"os"

	"github.com/status-im/proxy-common/cache"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	BigCache   cache.BigCacheConfig   `yaml:"bigcache"`
	KeyDB      cache.KeyDBConfig      `yaml:"keydb"`
	MultiCache cache.MultiCacheConfig `yaml:"multi_cache"`
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
	// Apply defaults from proxy-common
	c.BigCache.ApplyDefaults()
	c.KeyDB.ApplyDefaults()
}
