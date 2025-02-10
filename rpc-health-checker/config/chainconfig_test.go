package config

import (
	"os"
	"testing"

	"github.com/status-im/eth-rpc-proxy/provider"
	"github.com/stretchr/testify/assert"
)

func TestLoadChains(t *testing.T) {
	// Create temporary test file
	content := `{
		"chains": [
			{
				"name": "ethereum",
				"network": "mainnet",
				"chainId": 1,
				"providers": [
					{
						"name": "infura",
						"url": "https://mainnet.infura.io/v3",
						"authType": "token-auth",
						"authToken": "test",
						"chainId": 1
					}
				]
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "test-config-*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	t.Run("successful load", func(t *testing.T) {
		chains, err := LoadChains(tmpFile.Name())
		assert.NoError(t, err)
		assert.Len(t, chains.Chains, 1)
		assert.Equal(t, "ethereum", chains.Chains[0].Name)
		assert.Equal(t, "mainnet", chains.Chains[0].Network)
		assert.Equal(t, 1, chains.Chains[0].ChainID)
		assert.Len(t, chains.Chains[0].Providers, 1)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadChains("nonexistent.json")
		assert.Error(t, err)
	})

	t.Run("invalid json", func(t *testing.T) {
		invalidFile, err := os.CreateTemp("", "invalid-*.json")
		assert.NoError(t, err)
		defer os.Remove(invalidFile.Name())

		_, err = invalidFile.WriteString("{invalid}")
		assert.NoError(t, err)
		invalidFile.Close()

		_, err = LoadChains(invalidFile.Name())
		assert.Error(t, err)
	})
}

func TestLoadReferenceChains(t *testing.T) {
	// Create temporary test file
	content := `{
		"chains": [
			{
				"name": "ethereum",
				"network": "mainnet",
				"chainId": 1,
				"provider": {
					"name": "infura",
					"url": "https://mainnet.infura.io/v3",
					"authType": "token-auth",
					"authToken": "test",
					"chainId": 1
				}
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "test-ref-config-*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()

	t.Run("successful load", func(t *testing.T) {
		chains, err := LoadReferenceChains(tmpFile.Name())
		assert.NoError(t, err)
		assert.Len(t, chains.Chains, 1)
		assert.Equal(t, "ethereum", chains.Chains[0].Name)
		assert.Equal(t, "mainnet", chains.Chains[0].Network)
		assert.Equal(t, 1, chains.Chains[0].ChainId)
		assert.Equal(t, "infura", chains.Chains[0].Provider.Name)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadReferenceChains("nonexistent.json")
		assert.Error(t, err)
	})

	t.Run("invalid json", func(t *testing.T) {
		invalidFile, err := os.CreateTemp("", "invalid-ref-*.json")
		assert.NoError(t, err)
		defer os.Remove(invalidFile.Name())

		_, err = invalidFile.WriteString("{invalid}")
		assert.NoError(t, err)
		invalidFile.Close()

		_, err = LoadReferenceChains(invalidFile.Name())
		assert.Error(t, err)
	})

	t.Run("missing required fields", func(t *testing.T) {
		invalidFile, err := os.CreateTemp("", "missing-fields-*.json")
		assert.NoError(t, err)
		defer os.Remove(invalidFile.Name())

		_, err = invalidFile.WriteString(`{"chains": [{"name": "ethereum"}]}`)
		assert.NoError(t, err)
		invalidFile.Close()

		_, err = LoadReferenceChains(invalidFile.Name())
		assert.Error(t, err)
	})

	t.Run("normalization to lowercase", func(t *testing.T) {
		content := `{
			"chains": [
				{
					"name": "ETHEREUM",
					"network": "MAINNET",
					"chainId": 1,
					"provider": {
						"name": "infura",
						"url": "https://mainnet.infura.io/v3",
						"authType": "token-auth",
						"authToken": "test",
						"chainId": 1
					}
				}
			]
		}`

		tmpFile, err := os.CreateTemp("", "test-ref-upper-*.json")
		assert.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(content)
		assert.NoError(t, err)
		tmpFile.Close()

		chains, err := LoadReferenceChains(tmpFile.Name())
		assert.NoError(t, err)
		assert.Equal(t, "ethereum", chains.Chains[0].Name)
		assert.Equal(t, "mainnet", chains.Chains[0].Network)
		assert.Equal(t, 1, chains.Chains[0].ChainId)
	})
}

func TestGetChainByNameAndNetwork(t *testing.T) {
	chains := []ChainConfig{
		{
			Name:    "ethereum",
			Network: "mainnet",
			ChainID: 1,
		},
		{
			Name:    "ethereum",
			Network: "sepolia",
			ChainID: 11155111,
		},
	}

	t.Run("found", func(t *testing.T) {
		chain, err := GetChainByNameAndNetwork(chains, "ethereum", "mainnet")
		assert.NoError(t, err)
		assert.Equal(t, "mainnet", chain.Network)
		assert.Equal(t, 1, chain.ChainID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := GetChainByNameAndNetwork(chains, "unknown", "testnet")
		assert.Error(t, err)
	})
}

func TestGetReferenceProvider(t *testing.T) {
	chains := []ReferenceChainConfig{
		{
			Name:    "ethereum",
			Network: "mainnet",
			ChainId: 1,
			Provider: provider.RPCProvider{
				Name:    "infura",
				ChainID: 1,
			},
		},
		{
			Name:    "ethereum",
			Network: "sepolia",
			ChainId: 11155111,
			Provider: provider.RPCProvider{
				Name:    "alchemy",
				ChainID: 11155111,
			},
		},
	}

	t.Run("found", func(t *testing.T) {
		provider, err := GetReferenceProvider(chains, "ethereum", "mainnet")
		assert.NoError(t, err)
		assert.Equal(t, "infura", provider.Name)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := GetReferenceProvider(chains, "unknown", "testnet")
		assert.Error(t, err)
	})
}

func TestValidateChainConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  ChainConfig
		wantErr bool
	}{
		{
			name: "valid",
			config: ChainConfig{
				Name:    "ethereum",
				Network: "mainnet",
				ChainID: 1,
				Providers: []provider.RPCProvider{
					{
						Name:     "provider1",
						URL:      "https://provider1.example.com",
						AuthType: "no-auth",
						ChainID:  1,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: ChainConfig{
				Network: "mainnet",
				ChainID: 1,
				Providers: []provider.RPCProvider{
					{Name: "provider1", ChainID: 1},
				},
			},
			wantErr: true,
		},
		{
			name: "missing network",
			config: ChainConfig{
				Name:    "ethereum",
				ChainID: 1,
				Providers: []provider.RPCProvider{
					{Name: "provider1", ChainID: 1},
				},
			},
			wantErr: true,
		},
		{
			name: "no providers",
			config: ChainConfig{
				Name:    "ethereum",
				Network: "mainnet",
				ChainID: 1,
			},
			wantErr: true,
		},
		{
			name: "missing chainId",
			config: ChainConfig{
				Name:    "ethereum",
				Network: "mainnet",
				Providers: []provider.RPCProvider{
					{Name: "provider1", ChainID: 1},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChainConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
