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
	"github.com/status-im/eth-rpc-proxy/requests_runner/mocks"
	"github.com/stretchr/testify/assert"
)

func TestValidateMultipleEVMMethods(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create mock providers
	referenceProvider := provider.RPCProvider{
		Name:     "reference",
		URL:      "http://reference.com",
		AuthType: provider.NoAuth,
	}

	providerA := provider.RPCProvider{
		Name:     "providerA",
		URL:      "http://providerA.com",
		AuthType: provider.NoAuth,
	}

	providerB := provider.RPCProvider{
		Name:     "providerB",
		URL:      "http://providerB.com",
		AuthType: provider.NoAuth,
	}

	// Create mock MethodCaller
	mockCaller := &mocks.EVMMethodCaller{
		Responses: map[string]requestsrunner.ProviderResult{
			"reference": {
				Success:     true,
				Response:    []byte(`{"result":"0x64"}`),
				Result:      "0x64",
				ElapsedTime: 100 * time.Millisecond,
			},
			"providerA": {
				Success:     true,
				Response:    []byte(`{"result":"0x64"}`),
				Result:      "0x64",
				ElapsedTime: 100 * time.Millisecond,
			},
			"providerB": {
				Success:     true,
				Response:    []byte(`{"result":"0x6e"}`),
				Result:      "0x6e",
				ElapsedTime: 100 * time.Millisecond,
			},
		},
	}

	// Create comparison function
	compareFunc := func(reference, result *big.Int) bool {
		diff := new(big.Int).Abs(new(big.Int).Sub(result, reference))
		return diff.Cmp(big.NewInt(2)) <= 0
	}

	// Define multiple method tests
	methodConfigs := []config.EVMMethodTestConfig{
		{
			Method:      "eth_blockNumber",
			Params:      nil,
			CompareFunc: compareFunc,
		},
		{
			Method:      "eth_chainId",
			Params:      nil,
			CompareFunc: compareFunc,
		},
	}

	t.Run("successful validation with some failures", func(t *testing.T) {
		results := ValidateMultipleEVMMethods(
			ctx,
			methodConfigs,
			mockCaller,
			[]provider.RPCProvider{providerA, providerB},
			referenceProvider,
			500*time.Millisecond,
		)

		// Verify results structure
		assert.Len(t, results, 2, "should have results for both providers")
		assert.Contains(t, results, "providerA", "should have results for providerA")
		assert.Contains(t, results, "providerB", "should have results for providerB")

		// Verify providerA results
		providerAResults := results["providerA"]
		assert.True(t, providerAResults.Valid, "providerA should be valid")
		assert.Len(t, providerAResults.FailedMethods, 0, "providerA should have no failed methods")

		// Verify providerB results
		providerBResults := results["providerB"]
		assert.False(t, providerBResults.Valid, "providerB should be invalid")
		assert.Len(t, providerBResults.FailedMethods, 2, "providerB should have 2 failed methods")
		assert.Contains(t, providerBResults.FailedMethods, "eth_blockNumber", "should have eth_blockNumber failure")
		assert.Contains(t, providerBResults.FailedMethods, "eth_chainId", "should have eth_chainId failure")
	})

	t.Run("reference provider failure", func(t *testing.T) {
		// Create failing reference mock
		failingMock := &mocks.EVMMethodCaller{
			Responses: map[string]requestsrunner.ProviderResult{
				"reference": {
					Success: false,
					Error:   errors.New("reference failed"),
				},
				"providerA": {
					Success:  true,
					Response: []byte(`{"result":"0x64"}`),
				},
			},
		}

		results := ValidateMultipleEVMMethods(
			ctx,
			methodConfigs,
			failingMock,
			[]provider.RPCProvider{providerA},
			referenceProvider,
			500*time.Millisecond,
		)

		// Verify results are valid if reference fails
		providerAResults := results["providerA"]
		assert.True(t, providerAResults.Valid)
		assert.Len(t, providerAResults.FailedMethods, 0)
	})

	t.Run("partial provider failures", func(t *testing.T) {
		partialMock := &mocks.EVMMethodCaller{
			Responses: map[string]requestsrunner.ProviderResult{
				"reference": {
					Success:  true,
					Response: []byte(`{"result":"0x64"}`),
				},
				"providerA": {
					Success:  true,
					Response: []byte(`{"result":"0x65"}`),
				},
			},
			MethodResponses: map[string]map[string]requestsrunner.ProviderResult{
				"providerA": {
					"eth_blockNumber": {
						Success:  true,
						Response: []byte(`{"result":"0x65"}`),
					},
					"eth_chainId": {
						Success: false,
						Error:   errors.New("method failed"),
					},
				},
			},
		}

		results := ValidateMultipleEVMMethods(
			ctx,
			methodConfigs,
			partialMock,
			[]provider.RPCProvider{providerA},
			referenceProvider,
			500*time.Millisecond,
		)

		// Verify partial failure results
		providerAResults := results["providerA"]
		assert.False(t, providerAResults.Valid)
		assert.Len(t, providerAResults.FailedMethods, 1)
		assert.Contains(t, providerAResults.FailedMethods, "eth_chainId")
	})
}

func TestFailedMethodResultResponse(t *testing.T) {
	// Create a mock provider
	provider := provider.RPCProvider{
		Name:     "testProvider",
		URL:      "http://test.com",
		AuthType: provider.NoAuth,
	}

	// Create mock MethodCaller with test response
	mockCaller := &mocks.EVMMethodCaller{
		Responses: map[string]requestsrunner.ProviderResult{
			"testProvider": {
				Success:     true,
				Response:    []byte(`{"result":"test response"}`),
				ElapsedTime: 100 * time.Millisecond,
			},
		},
	}

	// Call the method
	result := mockCaller.CallMethod(
		context.Background(),
		provider,
		"testMethod",
		nil,
		500*time.Millisecond,
	)

	// Create a FailedMethodResult with the response
	failedResult := FailedMethodResult{
		Result: result,
	}

	// Verify the response is accessible through the Result field
	assert.Equal(t, `{"result":"test response"}`, string(failedResult.Result.Response))
}
