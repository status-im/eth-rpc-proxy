package config

import (
	"os"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func createTestConfigFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "cache_config_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpFile.Name()
}

func TestLoadConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)

	validConfig := `
bigcache:
  enabled: true
  size: 200

keydb:
  enabled: true
  connection:
    connect_timeout: 2s
    send_timeout: 2s
    read_timeout: 2s
  keepalive:
    pool_size: 20
    max_idle_timeout: 20s
  cache:
    default_ttl: 2h
    max_ttl: 48h

multi_cache:
  enable_propagation: true
`

	configFile := createTestConfigFile(t, validConfig)
	defer os.Remove(configFile)

	config, err := LoadConfig(configFile, logger)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Test BigCache config
	if !config.BigCache.Enabled {
		t.Errorf("LoadConfig() BigCache.Enabled = false, want true")
	}
	if config.BigCache.Size != 200 {
		t.Errorf("LoadConfig() BigCache.Size = %v, want 200", config.BigCache.Size)
	}

	// Test KeyDB config
	if !config.KeyDB.Enabled {
		t.Errorf("LoadConfig() KeyDB.Enabled = false, want true")
	}
	if config.KeyDB.Connection.ConnectTimeout != 2*time.Second {
		t.Errorf("LoadConfig() KeyDB.Connection.ConnectTimeout = %v, want 2s", config.KeyDB.Connection.ConnectTimeout)
	}
	if config.KeyDB.Keepalive.PoolSize != 20 {
		t.Errorf("LoadConfig() KeyDB.Keepalive.PoolSize = %v, want 20", config.KeyDB.Keepalive.PoolSize)
	}
	if config.KeyDB.Cache.DefaultTTL != 2*time.Hour {
		t.Errorf("LoadConfig() KeyDB.Cache.DefaultTTL = %v, want 2h", config.KeyDB.Cache.DefaultTTL)
	}

	// Test MultiCache config
	if !config.MultiCache.EnablePropagation {
		t.Errorf("LoadConfig() MultiCache.EnablePropagation = false, want true")
	}
}

func TestLoadConfig_WithDefaults(t *testing.T) {
	logger := zaptest.NewLogger(t)

	minimalConfig := `
bigcache:
  enabled: true

keydb:
  enabled: true
`

	configFile := createTestConfigFile(t, minimalConfig)
	defer os.Remove(configFile)

	config, err := LoadConfig(configFile, logger)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Test that defaults are applied
	if config.BigCache.Size != 100 {
		t.Errorf("LoadConfig() BigCache.Size = %v, want 100 (default)", config.BigCache.Size)
	}
	if config.KeyDB.Connection.ConnectTimeout != 1000*time.Millisecond {
		t.Errorf("LoadConfig() KeyDB.Connection.ConnectTimeout = %v, want 1000ms (default)", config.KeyDB.Connection.ConnectTimeout)
	}
	if config.KeyDB.Keepalive.PoolSize != 10 {
		t.Errorf("LoadConfig() KeyDB.Keepalive.PoolSize = %v, want 10 (default)", config.KeyDB.Keepalive.PoolSize)
	}
	if config.KeyDB.Cache.DefaultTTL != 3600*time.Second {
		t.Errorf("LoadConfig() KeyDB.Cache.DefaultTTL = %v, want 3600s (default)", config.KeyDB.Cache.DefaultTTL)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	logger := zaptest.NewLogger(t)

	_, err := LoadConfig("/nonexistent/file.yaml", logger)
	if err == nil {
		t.Fatal("LoadConfig() should return error for nonexistent file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	logger := zaptest.NewLogger(t)

	invalidConfig := `
bigcache:
  enabled: true
  invalid yaml syntax [
`

	configFile := createTestConfigFile(t, invalidConfig)
	defer os.Remove(configFile)

	_, err := LoadConfig(configFile, logger)
	if err == nil {
		t.Fatal("LoadConfig() should return error for invalid YAML")
	}
}

func TestConfig_ApplyDefaults(t *testing.T) {
	config := &Config{}
	config.applyDefaults()

	// Test BigCache defaults
	if config.BigCache.Size != 100 {
		t.Errorf("applyDefaults() BigCache.Size = %v, want 100", config.BigCache.Size)
	}

	// Test KeyDB connection defaults
	if config.KeyDB.Connection.ConnectTimeout != 1000*time.Millisecond {
		t.Errorf("applyDefaults() KeyDB.Connection.ConnectTimeout = %v, want 1000ms", config.KeyDB.Connection.ConnectTimeout)
	}
	if config.KeyDB.Connection.SendTimeout != 1000*time.Millisecond {
		t.Errorf("applyDefaults() KeyDB.Connection.SendTimeout = %v, want 1000ms", config.KeyDB.Connection.SendTimeout)
	}
	if config.KeyDB.Connection.ReadTimeout != 1000*time.Millisecond {
		t.Errorf("applyDefaults() KeyDB.Connection.ReadTimeout = %v, want 1000ms", config.KeyDB.Connection.ReadTimeout)
	}

	// Test KeyDB keepalive defaults
	if config.KeyDB.Keepalive.PoolSize != 10 {
		t.Errorf("applyDefaults() KeyDB.Keepalive.PoolSize = %v, want 10", config.KeyDB.Keepalive.PoolSize)
	}
	if config.KeyDB.Keepalive.MaxIdleTimeout != 10000*time.Millisecond {
		t.Errorf("applyDefaults() KeyDB.Keepalive.MaxIdleTimeout = %v, want 10000ms", config.KeyDB.Keepalive.MaxIdleTimeout)
	}

	// Test KeyDB cache defaults
	if config.KeyDB.Cache.DefaultTTL != 3600*time.Second {
		t.Errorf("applyDefaults() KeyDB.Cache.DefaultTTL = %v, want 3600s", config.KeyDB.Cache.DefaultTTL)
	}
	if config.KeyDB.Cache.MaxTTL != 86400*time.Second {
		t.Errorf("applyDefaults() KeyDB.Cache.MaxTTL = %v, want 86400s", config.KeyDB.Cache.MaxTTL)
	}
}

func TestConfig_PartialDefaults(t *testing.T) {
	config := &Config{
		BigCache: BigCacheConfig{
			Size: 250, // Custom value
		},
		KeyDB: KeyDBConfig{
			Connection: ConnectionConfig{
				ConnectTimeout: 2000 * time.Millisecond, // Custom value
				// SendTimeout and ReadTimeout should get defaults
			},
		},
	}

	config.applyDefaults()

	// Custom values should be preserved
	if config.BigCache.Size != 250 {
		t.Errorf("applyDefaults() should preserve custom BigCache.Size = %v", config.BigCache.Size)
	}
	if config.KeyDB.Connection.ConnectTimeout != 2000*time.Millisecond {
		t.Errorf("applyDefaults() should preserve custom KeyDB.Connection.ConnectTimeout = %v", config.KeyDB.Connection.ConnectTimeout)
	}

	// Missing values should get defaults
	if config.KeyDB.Connection.SendTimeout != 1000*time.Millisecond {
		t.Errorf("applyDefaults() KeyDB.Connection.SendTimeout = %v, want 1000ms (default)", config.KeyDB.Connection.SendTimeout)
	}
	if config.KeyDB.Connection.ReadTimeout != 1000*time.Millisecond {
		t.Errorf("applyDefaults() KeyDB.Connection.ReadTimeout = %v, want 1000ms (default)", config.KeyDB.Connection.ReadTimeout)
	}
}
