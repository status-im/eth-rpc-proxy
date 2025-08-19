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
l1:
  enabled: true
  size: 200

l2:
  enabled: true
  connection:
    connect_timeout: 2000
    send_timeout: 2000
    read_timeout: 2000
  keepalive:
    pool_size: 20
    max_idle_timeout: 20000
  cache:
    default_ttl: 7200
    max_ttl: 172800
`

	configFile := createTestConfigFile(t, validConfig)
	defer os.Remove(configFile)

	config, err := LoadConfig(configFile, logger)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Test L1 config
	if !config.L1.Enabled {
		t.Errorf("LoadConfig() L1.Enabled = false, want true")
	}
	if config.L1.Size != 200 {
		t.Errorf("LoadConfig() L1.Size = %v, want 200", config.L1.Size)
	}

	// Test L2 config
	if !config.L2.Enabled {
		t.Errorf("LoadConfig() L2.Enabled = false, want true")
	}
	if config.L2.Connection.ConnectTimeout != 2000 {
		t.Errorf("LoadConfig() L2.Connection.ConnectTimeout = %v, want 2000", config.L2.Connection.ConnectTimeout)
	}
	if config.L2.Keepalive.PoolSize != 20 {
		t.Errorf("LoadConfig() L2.Keepalive.PoolSize = %v, want 20", config.L2.Keepalive.PoolSize)
	}
	if config.L2.Cache.DefaultTTL != 7200 {
		t.Errorf("LoadConfig() L2.Cache.DefaultTTL = %v, want 7200", config.L2.Cache.DefaultTTL)
	}
}

func TestLoadConfig_WithDefaults(t *testing.T) {
	logger := zaptest.NewLogger(t)

	minimalConfig := `
l1:
  enabled: true

l2:
  enabled: true
`

	configFile := createTestConfigFile(t, minimalConfig)
	defer os.Remove(configFile)

	config, err := LoadConfig(configFile, logger)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Test that defaults are applied
	if config.L1.Size != 100 {
		t.Errorf("LoadConfig() L1.Size = %v, want 100 (default)", config.L1.Size)
	}
	if config.L2.Connection.ConnectTimeout != 1000 {
		t.Errorf("LoadConfig() L2.Connection.ConnectTimeout = %v, want 1000 (default)", config.L2.Connection.ConnectTimeout)
	}
	if config.L2.Keepalive.PoolSize != 10 {
		t.Errorf("LoadConfig() L2.Keepalive.PoolSize = %v, want 10 (default)", config.L2.Keepalive.PoolSize)
	}
	if config.L2.Cache.DefaultTTL != 3600 {
		t.Errorf("LoadConfig() L2.Cache.DefaultTTL = %v, want 3600 (default)", config.L2.Cache.DefaultTTL)
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
l1:
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

func TestConfig_TimeoutMethods(t *testing.T) {
	config := &Config{
		L2: L2Config{
			Connection: ConnectionConfig{
				ConnectTimeout: 1500,
				SendTimeout:    2500,
				ReadTimeout:    3500,
			},
			Keepalive: KeepaliveConfig{
				MaxIdleTimeout: 15000,
			},
			Cache: CacheConfig{
				DefaultTTL: 7200,
				MaxTTL:     86400,
			},
		},
	}

	tests := []struct {
		name     string
		method   func() time.Duration
		expected time.Duration
	}{
		{
			name:     "GetConnectTimeout",
			method:   config.GetConnectTimeout,
			expected: 1500 * time.Millisecond,
		},
		{
			name:     "GetSendTimeout",
			method:   config.GetSendTimeout,
			expected: 2500 * time.Millisecond,
		},
		{
			name:     "GetReadTimeout",
			method:   config.GetReadTimeout,
			expected: 3500 * time.Millisecond,
		},
		{
			name:     "GetMaxIdleTimeout",
			method:   config.GetMaxIdleTimeout,
			expected: 15000 * time.Millisecond,
		},
		{
			name:     "GetDefaultTTL",
			method:   config.GetDefaultTTL,
			expected: 7200 * time.Second,
		},
		{
			name:     "GetMaxTTL",
			method:   config.GetMaxTTL,
			expected: 86400 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.method()
			if result != tt.expected {
				t.Errorf("%s() = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestConfig_ApplyDefaults(t *testing.T) {
	config := &Config{}
	config.applyDefaults()

	// Test L1 defaults
	if config.L1.Size != 100 {
		t.Errorf("applyDefaults() L1.Size = %v, want 100", config.L1.Size)
	}

	// Test L2 connection defaults
	if config.L2.Connection.ConnectTimeout != 1000 {
		t.Errorf("applyDefaults() L2.Connection.ConnectTimeout = %v, want 1000", config.L2.Connection.ConnectTimeout)
	}
	if config.L2.Connection.SendTimeout != 1000 {
		t.Errorf("applyDefaults() L2.Connection.SendTimeout = %v, want 1000", config.L2.Connection.SendTimeout)
	}
	if config.L2.Connection.ReadTimeout != 1000 {
		t.Errorf("applyDefaults() L2.Connection.ReadTimeout = %v, want 1000", config.L2.Connection.ReadTimeout)
	}

	// Test L2 keepalive defaults
	if config.L2.Keepalive.PoolSize != 10 {
		t.Errorf("applyDefaults() L2.Keepalive.PoolSize = %v, want 10", config.L2.Keepalive.PoolSize)
	}
	if config.L2.Keepalive.MaxIdleTimeout != 10000 {
		t.Errorf("applyDefaults() L2.Keepalive.MaxIdleTimeout = %v, want 10000", config.L2.Keepalive.MaxIdleTimeout)
	}

	// Test L2 cache defaults
	if config.L2.Cache.DefaultTTL != 3600 {
		t.Errorf("applyDefaults() L2.Cache.DefaultTTL = %v, want 3600", config.L2.Cache.DefaultTTL)
	}
	if config.L2.Cache.MaxTTL != 86400 {
		t.Errorf("applyDefaults() L2.Cache.MaxTTL = %v, want 86400", config.L2.Cache.MaxTTL)
	}
}

func TestConfig_PartialDefaults(t *testing.T) {
	config := &Config{
		L1: L1Config{
			Size: 250, // Custom value
		},
		L2: L2Config{
			Connection: ConnectionConfig{
				ConnectTimeout: 2000, // Custom value
				// SendTimeout and ReadTimeout should get defaults
			},
		},
	}

	config.applyDefaults()

	// Custom values should be preserved
	if config.L1.Size != 250 {
		t.Errorf("applyDefaults() should preserve custom L1.Size = %v", config.L1.Size)
	}
	if config.L2.Connection.ConnectTimeout != 2000 {
		t.Errorf("applyDefaults() should preserve custom L2.Connection.ConnectTimeout = %v", config.L2.Connection.ConnectTimeout)
	}

	// Missing values should get defaults
	if config.L2.Connection.SendTimeout != 1000 {
		t.Errorf("applyDefaults() L2.Connection.SendTimeout = %v, want 1000 (default)", config.L2.Connection.SendTimeout)
	}
	if config.L2.Connection.ReadTimeout != 1000 {
		t.Errorf("applyDefaults() L2.Connection.ReadTimeout = %v, want 1000 (default)", config.L2.Connection.ReadTimeout)
	}
}
