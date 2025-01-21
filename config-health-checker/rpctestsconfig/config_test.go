package rpctestsconfig

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
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

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
