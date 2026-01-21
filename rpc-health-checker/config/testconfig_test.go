package config

import (
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadConfig(t *testing.T) {
	// Create temporary test file
	content := `[
		{
			"method": "eth_blockNumber",
			"params": [],
			"maxDifference": "10"
		},
		{
			"method": "eth_getBalance",
			"params": ["0x123...", "latest"],
			"maxDifference": "1000000000000000000"
		}
	]`

	tmpFile, err := os.CreateTemp("", "test-config-*.json")
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test reading config
	configs, err := ReadConfig(tmpFile.Name())
	require.NoError(t, err)
	require.Len(t, configs, 2)

	// Test first config
	require.Equal(t, "eth_blockNumber", configs[0].Method)
	require.Len(t, configs[0].Params, 0)
	require.True(t, configs[0].CompareFunc(big.NewInt(100), big.NewInt(105)))
	require.False(t, configs[0].CompareFunc(big.NewInt(111), big.NewInt(100)))

	// Test second config
	require.Equal(t, "eth_getBalance", configs[1].Method)
	require.Len(t, configs[1].Params, 2)
	require.True(t, configs[1].CompareFunc(
		func() *big.Int {
			n, _ := new(big.Int).SetString("10000000000000000000", 10)
			return n
		}(),
		func() *big.Int {
			n, _ := new(big.Int).SetString("10000000000000000001", 10)
			return n
		}(),
	))
	require.False(t, configs[1].CompareFunc(
		func() *big.Int {
			n, _ := new(big.Int).SetString("20000000000000000000", 10)
			return n
		}(),
		func() *big.Int {
			n, _ := new(big.Int).SetString("10000000000000000000", 10)
			return n
		}(),
	))
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		configs []EVMMethodTestConfig
		wantErr bool
	}{
		{
			name:    "empty config",
			configs: []EVMMethodTestConfig{},
			wantErr: true,
		},
		{
			name: "valid config",
			configs: []EVMMethodTestConfig{
				{
					Method: "eth_blockNumber",
					CompareFunc: func(a, b *big.Int) bool {
						return true
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid config - empty method",
			configs: []EVMMethodTestConfig{
				{
					Method: "",
					CompareFunc: func(a, b *big.Int) bool {
						return true
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.configs)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReadConfigWithSkipChains(t *testing.T) {
	content := `[
		{
			"method": "eth_blockNumber",
			"params": [],
			"maxDifference": "10"
		},
		{
			"method": "eth_estimateGas",
			"params": [{"from": "0x123", "to": "0x456"}],
			"maxDifference": "100000",
			"skipChains": [59141, 11155111]
		}
	]`

	tmpFile, err := os.CreateTemp("", "test-config-skip-*.json")
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	configs, err := ReadConfig(tmpFile.Name())
	require.NoError(t, err)
	require.Len(t, configs, 2)

	// Test first config - no skipChains
	require.Equal(t, "eth_blockNumber", configs[0].Method)
	require.Empty(t, configs[0].SkipChains, "SkipChains should be empty when not specified")

	// Test second config - has skipChains
	require.Equal(t, "eth_estimateGas", configs[1].Method)
	require.NotNil(t, configs[1].SkipChains)
	require.True(t, configs[1].SkipChains[59141], "chain 59141 should be in skipChains")
	require.True(t, configs[1].SkipChains[11155111], "chain 11155111 should be in skipChains")
	require.False(t, configs[1].SkipChains[1], "chain 1 should not be in skipChains")
}

func TestReadConfigWithEmptySkipChains(t *testing.T) {
	// Create temporary test file with empty skipChains
	content := `[
		{
			"method": "eth_getBalance",
			"params": ["0x123...", "latest"],
			"maxDifference": "0",
			"skipChains": []
		}
	]`

	tmpFile, err := os.CreateTemp("", "test-config-empty-skip-*.json")
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test reading config
	configs, err := ReadConfig(tmpFile.Name())
	require.NoError(t, err)
	require.Len(t, configs, 1)

	// SkipChains map should be empty but not nil
	require.NotNil(t, configs[0].SkipChains)
	require.Len(t, configs[0].SkipChains, 0)
}
