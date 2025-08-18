package cache_rules

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"go-proxy-cache/internal/interfaces"
)

// LoadCacheRulesConfig loads cache rules from a YAML file and returns a config reader
func LoadCacheRulesConfig(rulesPath string, logger *zap.Logger) (interfaces.CacheRulesConfig, error) {
	logger.Info("Loading cache rules config", zap.String("path", rulesPath))

	file, err := os.Open(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache rules file: %w", err)
	}
	defer file.Close()

	var config CacheRulesConfig
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode YAML cache rules: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("cache rules validation failed: %w", err)
	}

	logger.Info("Cache rules config loaded successfully")

	return NewCacheConfig(&config, logger), nil
}

// validateConfig validates the cache rules configuration structure
func validateConfig(config *CacheRulesConfig) error {
	if len(config.ChainsTTLDefaults) == 0 {
		return fmt.Errorf("missing ttl_defaults section")
	}

	if len(config.CacheRules) == 0 {
		return fmt.Errorf("missing cache_rules section")
	}

	// Check for default TTL section
	_, ok := config.ChainsTTLDefaults["default"]
	if !ok {
		return fmt.Errorf("missing ttl_defaults.default section")
	}

	return nil
}
