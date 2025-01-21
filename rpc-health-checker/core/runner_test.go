package core

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/status-im/eth-rpc-proxy/config"
	"github.com/status-im/eth-rpc-proxy/provider"
	requestsrunner "github.com/status-im/eth-rpc-proxy/requests_runner"
	"github.com/stretchr/testify/assert"
)

// MockEVMMethodCaller implements MethodCaller for testing
type MockEVMMethodCaller struct {
	results map[string]requestsrunner.ProviderResult
}

func (m *MockEVMMethodCaller) CallMethod(
	ctx context.Context,
	provider provider.RPCProvider,
	method string,
	params []interface{},
	timeout time.Duration,
) requestsrunner.ProviderResult {
	return m.results[provider.Name]
}

func TestChainValidationRunner_Run(t *testing.T) {
	// Setup test data
	chainCfgs := map[int64]config.ChainConfig{
		1: {
			Providers: []provider.RPCProvider{
				{Name: "provider1"},
				{Name: "provider2"},
			},
		},
	}

	referenceCfgs := map[int64]config.ReferenceChainConfig{
		1: {
			Provider: provider.RPCProvider{Name: "reference"},
		},
	}

	methodConfigs := []config.EVMMethodTestConfig{
		{
			Method: "eth_blockNumber",
			CompareFunc: func(ref, res *big.Int) bool {
				return ref.Cmp(res) == 0
			},
		},
	}

	// Create mock caller with predefined results
	mockCaller := &MockEVMMethodCaller{
		results: map[string]requestsrunner.ProviderResult{
			"reference": {
				Success:     true,
				Response:    []byte(`{"jsonrpc":"2.0","id":1,"result":"0x1234"}`),
				ElapsedTime: 100 * time.Millisecond,
			},
			"provider1": {
				Success:     true,
				Response:    []byte(`{"jsonrpc":"2.0","id":1,"result":"0x1234"}`),
				ElapsedTime: 100 * time.Millisecond,
			},
			"provider2": {
				Success:     true,
				Response:    []byte(`{"jsonrpc":"2.0","id":1,"result":"0x5678"}`),
				ElapsedTime: 100 * time.Millisecond,
			},
		},
	}

	// Create runner
	runner := NewChainValidationRunner(
		chainCfgs,
		referenceCfgs,
		methodConfigs,
		mockCaller,
		10*time.Second,
		"", // Empty output path for tests
		"", // Empty log path for tests
	)

	// Run tests
	runner.Run(context.Background())
}

func TestChainValidationRunner_ReferenceProviderFailure(t *testing.T) {
	// Setup test data
	chainCfgs := map[int64]config.ChainConfig{
		1: {
			Providers: []provider.RPCProvider{
				{Name: "provider1"},
			},
		},
	}

	referenceCfgs := map[int64]config.ReferenceChainConfig{
		1: {
			Provider: provider.RPCProvider{Name: "reference"},
		},
	}

	methodConfigs := []config.EVMMethodTestConfig{
		{
			Method: "eth_blockNumber",
			CompareFunc: func(ref, res *big.Int) bool {
				return ref.Cmp(res) == 0
			},
		},
	}

	// Create mock caller with failing reference provider
	mockCaller := &MockEVMMethodCaller{
		results: map[string]requestsrunner.ProviderResult{
			"reference": {
				Success: false,
				Error:   errors.New("reference failed"),
			},
			"provider1": {
				Success:  true,
				Response: []byte(`{"result":"0x1234"}`),
			},
		},
	}

	// Create runner
	runner := NewChainValidationRunner(
		chainCfgs,
		referenceCfgs,
		methodConfigs,
		mockCaller,
		10*time.Second,
		"", // Empty output path for tests
		"", // Empty log path for tests
	)

	// Run tests
	runner.Run(context.Background())
}

func TestChainValidationRunner_ValidateChains(t *testing.T) {
	// Setup test data
	chainCfgs := map[int64]config.ChainConfig{
		1: {
			Providers: []provider.RPCProvider{
				{Name: "provider1"},
				{Name: "provider2"},
			},
		},
	}

	referenceCfgs := map[int64]config.ReferenceChainConfig{
		1: {
			Provider: provider.RPCProvider{Name: "reference"},
		},
	}

	methodConfigs := []config.EVMMethodTestConfig{
		{
			Method: "eth_blockNumber",
			CompareFunc: func(ref, res *big.Int) bool {
				return ref.Cmp(res) == 0
			},
		},
	}

	// Create mock caller with predefined results
	mockCaller := &MockEVMMethodCaller{
		results: map[string]requestsrunner.ProviderResult{
			"reference": {
				Success:  true,
				Response: []byte(`{"result":"0x1234"}`),
			},
			"provider1": {
				Success:  true,
				Response: []byte(`{"result":"0x1234"}`),
			},
			"provider2": {
				Success:  true,
				Response: []byte(`{"result":"0x5678"}`),
			},
		},
	}

	// Create runner
	runner := NewChainValidationRunner(
		chainCfgs,
		referenceCfgs,
		methodConfigs,
		mockCaller,
		10*time.Second,
		"", // Empty output path for tests
		"", // Empty log path for tests
	)

	// Test validateChains
	t.Run("valid chains", func(t *testing.T) {
		validChains, results := runner.validateChains(context.Background())

		assert.Contains(t, results, int64(1), "should have results for chain ID 1")
		assert.Len(t, validChains, 1, "should have one valid chain")

		chainResults := results[1]
		assert.Contains(t, chainResults, "provider1", "should have results for provider1")
		assert.True(t, chainResults["provider1"].Valid, "provider1 should be valid")
		assert.Contains(t, chainResults, "provider2", "should have results for provider2")
		assert.False(t, chainResults["provider2"].Valid, "provider2 should be invalid")
		assert.Equal(t, `{"result":"0x5678"}`, string(chainResults["provider2"].FailedMethods["eth_blockNumber"].Result.Response), "provider2 should have correct failed response")
	})

	t.Run("failed methods tracking", func(t *testing.T) {
		validChains, results := runner.validateChains(context.Background())

		assert.Contains(t, results, int64(1), "should have results for chain ID 1")
		assert.Len(t, validChains, 1, "should have one valid chain")

		chainResults := results[1]
		assert.Contains(t, chainResults, "provider1", "should have results for provider1")
		assert.True(t, chainResults["provider1"].Valid, "provider1 should be valid")
		assert.Contains(t, chainResults, "provider2", "should have results for provider2")
		assert.False(t, chainResults["provider2"].Valid, "provider2 should be invalid")
		assert.Contains(t, chainResults["provider2"].FailedMethods, "eth_blockNumber", "should track failed eth_blockNumber method")
	})
}
