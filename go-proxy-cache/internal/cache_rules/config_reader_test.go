package cache_rules

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/status-im/proxy-common/models"
)

func TestLoadCacheRulesConfig_Success(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create a temporary YAML file with valid configuration
	validYAML := `
ttl_defaults:
  default:
    permanent: 86400s
    short: 5s
    minimal: 0s
  ethereum:mainnet:
    permanent: 43200s
    short: 10s
    minimal: 0s

cache_rules:
  eth_getBlockByHash: "permanent"
  eth_blockNumber: "short"
  eth_sendRawTransaction: "minimal"
`

	tmpFile := createTempYAMLFile(t, validYAML)
	defer func() { _ = os.Remove(tmpFile) }()

	// Test loading the configuration
	config, err := LoadCacheRulesConfig(tmpFile, logger)

	require.NoError(t, err)
	require.NotNil(t, config)

	// Verify the configuration was loaded correctly
	cacheConfig, ok := config.(*CacheConfig)
	require.True(t, ok)

	// Check TTL defaults
	assert.Equal(t, 2, len(cacheConfig.config.ChainsTTLDefaults))

	defaultTTLs := cacheConfig.config.ChainsTTLDefaults["default"]
	assert.Equal(t, 86400*time.Second, defaultTTLs[models.CacheTypePermanent])
	assert.Equal(t, 5*time.Second, defaultTTLs[models.CacheTypeShort])
	assert.Equal(t, 0*time.Second, defaultTTLs[models.CacheTypeMinimal])

	ethTTLs := cacheConfig.config.ChainsTTLDefaults["ethereum:mainnet"]
	assert.Equal(t, 43200*time.Second, ethTTLs[models.CacheTypePermanent])
	assert.Equal(t, 10*time.Second, ethTTLs[models.CacheTypeShort])

	// Check cache rules
	assert.Equal(t, models.CacheTypePermanent, cacheConfig.config.CacheRules["eth_getBlockByHash"])
	assert.Equal(t, models.CacheTypeShort, cacheConfig.config.CacheRules["eth_blockNumber"])
	assert.Equal(t, models.CacheTypeMinimal, cacheConfig.config.CacheRules["eth_sendRawTransaction"])
}

func TestLoadCacheRulesConfig_FileNotFound(t *testing.T) {
	logger := zaptest.NewLogger(t)

	config, err := LoadCacheRulesConfig("/nonexistent/file.yaml", logger)

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to open cache rules file")
}

func TestLoadCacheRulesConfig_InvalidYAML(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create a temporary file with invalid YAML
	invalidYAML := `
ttl_defaults:
  default:
    permanent: 86400
    short: 5
    minimal: 0
cache_rules:
  eth_getBlockByHash: "permanent"
  - invalid_yaml_structure
`

	tmpFile := createTempYAMLFile(t, invalidYAML)
	defer func() { _ = os.Remove(tmpFile) }()

	config, err := LoadCacheRulesConfig(tmpFile, logger)

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to decode YAML cache rules")
}

func TestLoadCacheRulesConfig_ValidationFailure(t *testing.T) {
	logger := zaptest.NewLogger(t)

	testCases := []struct {
		name     string
		yaml     string
		errorMsg string
	}{
		{
			name: "missing ttl_defaults",
			yaml: `
cache_rules:
  eth_getBlockByHash: "permanent"
`,
			errorMsg: "missing ttl_defaults section",
		},
		{
			name: "missing cache_rules",
			yaml: `
ttl_defaults:
  default:
    permanent: 86400s
`,
			errorMsg: "missing cache_rules section",
		},
		{
			name: "missing default ttl_defaults",
			yaml: `
ttl_defaults:
  ethereum:mainnet:
    permanent: 86400s
    short: 5s
    minimal: 0s
cache_rules:
  eth_getBlockByHash: "permanent"
`,
			errorMsg: "missing ttl_defaults.default section",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile := createTempYAMLFile(t, tc.yaml)
			defer func() { _ = os.Remove(tmpFile) }()

			config, err := LoadCacheRulesConfig(tmpFile, logger)

			assert.Error(t, err)
			assert.Nil(t, config)
			assert.Contains(t, err.Error(), "cache rules validation failed")
			assert.Contains(t, err.Error(), tc.errorMsg)
		})
	}
}

func TestValidateConfig_Success(t *testing.T) {
	validConfig := &CacheRulesConfig{
		ChainsTTLDefaults: map[string]TTLDefaults{
			"default": {
				models.CacheTypePermanent: 86400 * time.Second,
				models.CacheTypeShort:     5 * time.Second,
				models.CacheTypeMinimal:   0,
			},
		},
		CacheRules: map[string]models.CacheType{
			"eth_getBlockByHash": models.CacheTypePermanent,
		},
	}

	err := validateConfig(validConfig)
	assert.NoError(t, err)
}

func TestValidateConfig_EmptyTTLDefaults(t *testing.T) {
	config := &CacheRulesConfig{
		ChainsTTLDefaults: map[string]TTLDefaults{},
		CacheRules: map[string]models.CacheType{
			"eth_getBlockByHash": models.CacheTypePermanent,
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing ttl_defaults section")
}

func TestValidateConfig_EmptyCacheRules(t *testing.T) {
	config := &CacheRulesConfig{
		ChainsTTLDefaults: map[string]TTLDefaults{
			"default": {
				models.CacheTypePermanent: 86400 * time.Second,
			},
		},
		CacheRules: map[string]models.CacheType{},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing cache_rules section")
}

func TestValidateConfig_MissingDefaultTTL(t *testing.T) {
	config := &CacheRulesConfig{
		ChainsTTLDefaults: map[string]TTLDefaults{
			"ethereum:mainnet": {
				models.CacheTypePermanent: 86400 * time.Second,
			},
		},
		CacheRules: map[string]models.CacheType{
			"eth_getBlockByHash": models.CacheTypePermanent,
		},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing ttl_defaults.default section")
}

func TestLoadCacheRulesConfig_ComplexConfiguration(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Test with a more complex configuration similar to the actual cache_rules.yaml
	complexYAML := `
ttl_defaults:
  default:
    permanent: 86400s
    short: 5s
    minimal: 0s
  ethereum:mainnet:
    permanent: 86400s
    short: 10s
    minimal: 0s
  polygon:mainnet:
    permanent: 43200s
    short: 5s
    minimal: 0s

cache_rules:
  # Permanent cache
  eth_getBlockByHash: "permanent"
  eth_getBlockByNumber: "permanent"
  eth_getTransactionByHash: "permanent"
  eth_getTransactionReceipt: "permanent"
  
  # Short cache
  eth_blockNumber: "short"
  eth_gasPrice: "short"
  eth_getBalance: "short"
  eth_call: "short"
  
  # Minimal cache
  eth_sendRawTransaction: "minimal"
  eth_sendTransaction: "minimal"
  
  # Web3 methods
  web3_clientVersion: "permanent"
  net_version: "permanent"
`

	tmpFile := createTempYAMLFile(t, complexYAML)
	defer func() { _ = os.Remove(tmpFile) }()

	config, err := LoadCacheRulesConfig(tmpFile, logger)

	require.NoError(t, err)
	require.NotNil(t, config)

	cacheConfig, ok := config.(*CacheConfig)
	require.True(t, ok)

	// Verify multiple chain configurations
	assert.Equal(t, 3, len(cacheConfig.config.ChainsTTLDefaults))

	// Test specific TTL lookups
	assert.Equal(t, 86400*time.Second, cacheConfig.GetTtlForCacheType("ethereum", "mainnet", models.CacheTypePermanent))
	assert.Equal(t, 10*time.Second, cacheConfig.GetTtlForCacheType("ethereum", "mainnet", models.CacheTypeShort))
	assert.Equal(t, 43200*time.Second, cacheConfig.GetTtlForCacheType("polygon", "mainnet", models.CacheTypePermanent))

	// Test cache type lookups
	assert.Equal(t, models.CacheTypePermanent, cacheConfig.GetCacheTypeForMethod("eth_getBlockByHash"))
	assert.Equal(t, models.CacheTypeShort, cacheConfig.GetCacheTypeForMethod("eth_blockNumber"))
}

func TestLoadCacheRulesConfig_EmptyFile(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tmpFile := createTempYAMLFile(t, "")
	defer func() { _ = os.Remove(tmpFile) }()

	config, err := LoadCacheRulesConfig(tmpFile, logger)

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to decode YAML cache rules")
}

func TestLoadCacheRulesConfig_InvalidCacheType(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// YAML with invalid cache type
	invalidCacheTypeYAML := `
ttl_defaults:
  default:
    permanent: 86400s
    short: 5s
    minimal: 0s

cache_rules:
  eth_getBlockByHash: "invalid_cache_type"
`

	tmpFile := createTempYAMLFile(t, invalidCacheTypeYAML)
	defer func() { _ = os.Remove(tmpFile) }()

	config, err := LoadCacheRulesConfig(tmpFile, logger)

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to decode YAML cache rules")
}

func TestLoadCacheRulesConfig_WithLogger(t *testing.T) {
	// Test that the logger is properly used
	logger := zaptest.NewLogger(t)

	validYAML := `
ttl_defaults:
  default:
    permanent: 86400s
    short: 5s
    minimal: 0s

cache_rules:
  eth_getBlockByHash: "permanent"
`

	tmpFile := createTempYAMLFile(t, validYAML)
	defer func() { _ = os.Remove(tmpFile) }()

	config, err := LoadCacheRulesConfig(tmpFile, logger)

	require.NoError(t, err)
	require.NotNil(t, config)

	// Verify that the returned config has the logger
	cacheConfig, ok := config.(*CacheConfig)
	require.True(t, ok)
	assert.NotNil(t, cacheConfig.logger)
}

// Helper function to create temporary YAML files for testing
func createTempYAMLFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_cache_rules.yaml")

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	return tmpFile
}

// Benchmark tests
func BenchmarkLoadCacheRulesConfig(b *testing.B) {
	logger := zap.NewNop()

	validYAML := `
ttl_defaults:
  default:
    permanent: 86400s
    short: 5s
    minimal: 0s
  ethereum:mainnet:
    permanent: 43200s
    short: 10s
    minimal: 0s

cache_rules:
  eth_getBlockByHash: "permanent"
  eth_blockNumber: "short"
  eth_sendRawTransaction: "minimal"
`

	tmpFile := filepath.Join(b.TempDir(), "bench_cache_rules.yaml")
	err := os.WriteFile(tmpFile, []byte(validYAML), 0644)
	require.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		config, err := LoadCacheRulesConfig(tmpFile, logger)
		if err != nil {
			b.Fatal(err)
		}
		if config == nil {
			b.Fatal("config is nil")
		}
	}
}

func BenchmarkValidateConfig(b *testing.B) {
	config := &CacheRulesConfig{
		ChainsTTLDefaults: map[string]TTLDefaults{
			"default": {
				models.CacheTypePermanent: 86400 * time.Second,
				models.CacheTypeShort:     5 * time.Second,
				models.CacheTypeMinimal:   0,
			},
			"ethereum:mainnet": {
				models.CacheTypePermanent: 43200 * time.Second,
				models.CacheTypeShort:     10 * time.Second,
				models.CacheTypeMinimal:   0,
			},
		},
		CacheRules: map[string]models.CacheType{
			"eth_getBlockByHash":     models.CacheTypePermanent,
			"eth_blockNumber":        models.CacheTypeShort,
			"eth_sendRawTransaction": models.CacheTypeMinimal,
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := validateConfig(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}
