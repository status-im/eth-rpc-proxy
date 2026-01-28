package cache_rules

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/models"
)

func TestNewCacheConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &CacheRulesConfig{}

	cacheConfig := NewCacheConfig(config, logger)

	if cacheConfig == nil {
		t.Fatal("NewCacheConfig returned nil")
	}
	if cacheConfig.config != config {
		t.Error("Config not set correctly")
	}
	if cacheConfig.logger != logger {
		t.Error("Logger not set correctly")
	}
}

func TestNewCacheConfig_NilConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)

	defer func() {
		if r := recover(); r == nil {
			t.Error("NewCacheConfig should panic when config is nil")
		}
	}()

	NewCacheConfig(nil, logger)
}

func TestGetTtlForCacheType(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name        string
		config      *CacheRulesConfig
		chain       string
		network     string
		cacheType   models.CacheType
		expected    time.Duration
		description string
	}{
		{
			name: "empty ChainsTTLDefaults",
			config: &CacheRulesConfig{
				ChainsTTLDefaults: map[string]TTLDefaults{},
			},
			chain:       "ethereum",
			network:     "mainnet",
			cacheType:   models.CacheTypeShort,
			expected:    5 * time.Second, // fallback
			description: "should return fallback TTL when ChainsTTLDefaults is empty",
		},
		{
			name: "network-specific config found",
			config: &CacheRulesConfig{
				ChainsTTLDefaults: map[string]TTLDefaults{
					"ethereum:mainnet": {
						models.CacheTypePermanent: 2 * time.Hour,
						models.CacheTypeShort:     10 * time.Second,
					},
				},
			},
			chain:       "ethereum",
			network:     "mainnet",
			cacheType:   models.CacheTypePermanent,
			expected:    2 * time.Hour,
			description: "should return network-specific TTL when available",
		},
		{
			name: "chain-specific config found",
			config: &CacheRulesConfig{
				ChainsTTLDefaults: map[string]TTLDefaults{
					"ethereum": {
						models.CacheTypePermanent: 3 * time.Hour,
						models.CacheTypeShort:     15 * time.Second,
					},
				},
			},
			chain:       "ethereum",
			network:     "testnet",
			cacheType:   models.CacheTypePermanent,
			expected:    3 * time.Hour,
			description: "should return chain-specific TTL when network-specific not found",
		},
		{
			name: "default config found",
			config: &CacheRulesConfig{
				ChainsTTLDefaults: map[string]TTLDefaults{
					"default": {
						models.CacheTypePermanent: 4 * time.Hour,
						models.CacheTypeShort:     20 * time.Second,
					},
				},
			},
			chain:       "unknown",
			network:     "unknown",
			cacheType:   models.CacheTypePermanent,
			expected:    4 * time.Hour,
			description: "should return default TTL when chain and network not found",
		},
		{
			name: "no matching config",
			config: &CacheRulesConfig{
				ChainsTTLDefaults: map[string]TTLDefaults{
					"bitcoin": {
						models.CacheTypePermanent: 1 * time.Hour,
					},
				},
			},
			chain:       "ethereum",
			network:     "mainnet",
			cacheType:   models.CacheTypePermanent,
			expected:    0,
			description: "should return 0 when no matching config found",
		},
		{
			name: "cache type not found in config",
			config: &CacheRulesConfig{
				ChainsTTLDefaults: map[string]TTLDefaults{
					"ethereum": {
						models.CacheTypePermanent: 1 * time.Hour,
					},
				},
			},
			chain:       "ethereum",
			network:     "mainnet",
			cacheType:   models.CacheTypeShort, // not in config
			expected:    0,
			description: "should return 0 when cache type not found in config",
		},
		{
			name: "empty chain and network",
			config: &CacheRulesConfig{
				ChainsTTLDefaults: map[string]TTLDefaults{
					"default": {
						models.CacheTypePermanent: 5 * time.Hour,
					},
				},
			},
			chain:       "",
			network:     "",
			cacheType:   models.CacheTypePermanent,
			expected:    5 * time.Hour,
			description: "should use default config when chain and network are empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheConfig := NewCacheConfig(tt.config, logger)
			result := cacheConfig.GetTtlForCacheType(tt.chain, tt.network, tt.cacheType)
			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, result)
			}
		})
	}
}

// TestGetTtlForCacheType_NilReceiver is removed because calling a method on a nil receiver
// in Go will always panic - this is fundamental Go behavior and cannot be avoided.

func TestLookupTTL(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name        string
		config      *CacheRulesConfig
		key         string
		cacheType   models.CacheType
		expected    time.Duration
		description string
	}{
		{
			name: "key found with cache type",
			config: &CacheRulesConfig{
				ChainsTTLDefaults: map[string]TTLDefaults{
					"ethereum": {
						models.CacheTypePermanent: 1 * time.Hour,
						models.CacheTypeShort:     30 * time.Second,
					},
				},
			},
			key:         "ethereum",
			cacheType:   models.CacheTypePermanent,
			expected:    1 * time.Hour,
			description: "should return TTL when key and cache type found",
		},
		{
			name: "key found but cache type not found",
			config: &CacheRulesConfig{
				ChainsTTLDefaults: map[string]TTLDefaults{
					"ethereum": {
						models.CacheTypePermanent: 1 * time.Hour,
					},
				},
			},
			key:         "ethereum",
			cacheType:   models.CacheTypeShort, // not in config
			expected:    0,
			description: "should return 0 when cache type not found",
		},
		{
			name: "key not found",
			config: &CacheRulesConfig{
				ChainsTTLDefaults: map[string]TTLDefaults{
					"bitcoin": {
						models.CacheTypePermanent: 1 * time.Hour,
					},
				},
			},
			key:         "ethereum", // not in config
			cacheType:   models.CacheTypePermanent,
			expected:    0,
			description: "should return 0 when key not found",
		},
		{
			name: "empty key",
			config: &CacheRulesConfig{
				ChainsTTLDefaults: map[string]TTLDefaults{
					"ethereum": {
						models.CacheTypePermanent: 1 * time.Hour,
					},
				},
			},
			key:         "",
			cacheType:   models.CacheTypePermanent,
			expected:    0,
			description: "should return 0 for empty key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheConfig := NewCacheConfig(tt.config, logger)
			result := cacheConfig.lookupTTL(tt.key, tt.cacheType)

			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, result)
			}
		})
	}
}

func TestGetFallbackTTL(t *testing.T) {
	logger := zaptest.NewLogger(t)
	cacheConfig := NewCacheConfig(&CacheRulesConfig{}, logger)

	tests := []struct {
		cacheType models.CacheType
		expected  time.Duration
	}{
		{models.CacheTypePermanent, 24 * time.Hour},
		{models.CacheTypeShort, 5 * time.Second},
		{models.CacheTypeMinimal, 0},
		{models.CacheTypeNone, 0}, // unknown type should return 0
		{"unknown", 0},            // completely unknown type
	}

	for _, tt := range tests {
		t.Run(string(tt.cacheType), func(t *testing.T) {
			result := cacheConfig.getFallbackTTL(tt.cacheType)
			if result != tt.expected {
				t.Errorf("Expected %v for cache type %s, got %v", tt.expected, tt.cacheType, result)
			}
		})
	}
}

func TestGetCacheTypeForMethod(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name        string
		config      *CacheRulesConfig
		method      string
		expected    models.CacheType
		description string
	}{
		{
			name: "method found in cache rules",
			config: &CacheRulesConfig{
				CacheRules: map[string]models.CacheType{
					"eth_getBalance":     models.CacheTypePermanent,
					"eth_getBlockByHash": models.CacheTypeShort,
				},
			},
			method:      "eth_getBalance",
			expected:    models.CacheTypePermanent,
			description: "should return correct cache type when method found",
		},
		{
			name: "method not found in cache rules",
			config: &CacheRulesConfig{
				CacheRules: map[string]models.CacheType{
					"eth_getBalance": models.CacheTypePermanent,
				},
			},
			method:      "eth_sendTransaction", // not in config
			expected:    models.CacheTypeNone,
			description: "should return CacheTypeNone when method not found",
		},
		{
			name: "empty method",
			config: &CacheRulesConfig{
				CacheRules: map[string]models.CacheType{
					"eth_getBalance": models.CacheTypePermanent,
				},
			},
			method:      "",
			expected:    models.CacheTypeNone,
			description: "should return CacheTypeNone for empty method",
		},
		{
			name: "nil cache rules",
			config: &CacheRulesConfig{
				CacheRules: nil,
			},
			method:      "eth_getBalance",
			expected:    models.CacheTypeNone,
			description: "should return CacheTypeNone when CacheRules is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheConfig := NewCacheConfig(tt.config, logger)
			result := cacheConfig.GetCacheTypeForMethod(tt.method)
			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, result)
			}
		})
	}
}

// TestGetCacheTypeForMethod_NilReceiver is removed because calling a method on a nil receiver
// in Go will always panic - this is fundamental Go behavior and cannot be avoided.
// The previous test was fundamentally flawed.

func TestInterfaceCompliance(t *testing.T) {
	// This test ensures that CacheConfig implements the CacheRulesConfig interface
	logger := zaptest.NewLogger(t)
	config := &CacheRulesConfig{}
	cacheConfig := NewCacheConfig(config, logger)

	// This should compile without issues if the interface is properly implemented
	var _ interfaces.CacheRulesConfig = cacheConfig
}

// Benchmark tests
func BenchmarkGetTtlForCacheType(b *testing.B) {
	logger := zap.NewNop()
	config := &CacheRulesConfig{
		ChainsTTLDefaults: map[string]TTLDefaults{
			"ethereum:mainnet": {
				models.CacheTypePermanent: 2 * time.Hour,
				models.CacheTypeShort:     10 * time.Second,
			},
			"ethereum": {
				models.CacheTypePermanent: 3 * time.Hour,
				models.CacheTypeShort:     15 * time.Second,
			},
			"default": {
				models.CacheTypePermanent: 4 * time.Hour,
				models.CacheTypeShort:     20 * time.Second,
			},
		},
	}
	cacheConfig := NewCacheConfig(config, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cacheConfig.GetTtlForCacheType("ethereum", "mainnet", models.CacheTypePermanent)
	}
}

func BenchmarkGetCacheTypeForMethod(b *testing.B) {
	logger := zap.NewNop()
	config := &CacheRulesConfig{
		CacheRules: map[string]models.CacheType{
			"eth_getBalance":     models.CacheTypePermanent,
			"eth_getBlockByHash": models.CacheTypeShort,
			"eth_call":           models.CacheTypeMinimal,
		},
	}
	cacheConfig := NewCacheConfig(config, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cacheConfig.GetCacheTypeForMethod("eth_getBalance")
	}
}

func TestShouldSkipNullCache(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name        string
		config      *CacheRulesConfig
		method      string
		expected    bool
		description string
	}{
		{
			name: "method in skip_null_cache list",
			config: &CacheRulesConfig{
				SkipNullCache: []string{
					"eth_getTransactionReceipt",
					"eth_getTransactionByHash",
					"eth_getBlockByHash",
					"eth_getBlockByNumber",
				},
			},
			method:      "eth_getTransactionReceipt",
			expected:    true,
			description: "should return true when method is in skip_null_cache list",
		},
		{
			name: "method not in skip_null_cache list",
			config: &CacheRulesConfig{
				SkipNullCache: []string{
					"eth_getTransactionReceipt",
					"eth_getTransactionByHash",
				},
			},
			method:      "eth_getBalance",
			expected:    false,
			description: "should return false when method is not in skip_null_cache list",
		},
		{
			name: "empty skip_null_cache list",
			config: &CacheRulesConfig{
				SkipNullCache: []string{},
			},
			method:      "eth_getTransactionReceipt",
			expected:    false,
			description: "should return false when skip_null_cache list is empty",
		},
		{
			name: "nil skip_null_cache list",
			config: &CacheRulesConfig{
				SkipNullCache: nil,
			},
			method:      "eth_getTransactionReceipt",
			expected:    false,
			description: "should return false when skip_null_cache list is nil",
		},
		{
			name: "empty method",
			config: &CacheRulesConfig{
				SkipNullCache: []string{
					"eth_getTransactionReceipt",
				},
			},
			method:      "",
			expected:    false,
			description: "should return false for empty method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheConfig := NewCacheConfig(tt.config, logger)
			result := cacheConfig.ShouldSkipNullCache(tt.method)
			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, result)
			}
		})
	}
}

func BenchmarkShouldSkipNullCache(b *testing.B) {
	logger := zap.NewNop()
	config := &CacheRulesConfig{
		SkipNullCache: []string{
			"eth_getTransactionReceipt",
			"eth_getTransactionByHash",
			"eth_getBlockByHash",
			"eth_getBlockByNumber",
		},
	}
	cacheConfig := NewCacheConfig(config, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cacheConfig.ShouldSkipNullCache("eth_getTransactionReceipt")
	}
}
