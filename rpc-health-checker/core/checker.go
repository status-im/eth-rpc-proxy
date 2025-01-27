package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/status-im/eth-rpc-proxy/config"

	"github.com/status-im/eth-rpc-proxy/provider"
	requestsrunner "github.com/status-im/eth-rpc-proxy/requests_runner"
)

// MultiMethodTestResult contains results for multiple method tests
type MultiMethodTestResult struct {
	Results map[string]CheckResult `json:"results"` // method -> result
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// TestEVMMethodWithCaller tests a single EVM method against multiple providers
// Returns a map of provider names to their validation results
func TestEVMMethodWithCaller(
	ctx context.Context,
	config config.EVMMethodTestConfig,
	caller requestsrunner.MethodCaller,
	providers []provider.RPCProvider,
	referenceProvider provider.RPCProvider,
	timeout time.Duration,
) map[string]CheckResult {
	// Validate inputs
	if caller == nil || referenceProvider.Name == "" {
		return map[string]CheckResult{}
	}

	// Combine reference provider with test providers
	allProviders := append([]provider.RPCProvider{referenceProvider}, providers...)

	// Execute the EVM method in parallel using ParallelCallEVMMethods
	results := requestsrunner.ParallelCallEVMMethods(ctx, allProviders, config.Method, config.Params, timeout, caller)

	// Extract reference result
	refResult, refExists := results[referenceProvider.Name]
	if !refExists || !refResult.Success {
		return handleReferenceError(results,
			referenceProvider.Name,
			fmt.Sprintf("validation failed: reference provider %s failed", referenceProvider.Name))
	}

	// Parse reference value
	refValue, err := parseJSONRPCResult(refResult.Response)
	if err != nil {
		return handleReferenceError(results,
			referenceProvider.Name,
			fmt.Sprintf("failed to parse reference provider %s response: %v", referenceProvider.Name, err))
	}

	// Compare each provider's result to reference
	checkResults := make(map[string]CheckResult)
	for _, provider := range providers {
		result, exists := results[provider.Name]
		if !exists {
			checkResults[provider.Name] = CheckResult{
				Valid:           false,
				Error:           "provider result not found",
				ReferenceResult: refResult,
			}
			continue
		}

		// Handle failed requests
		if !result.Success {
			checkResults[provider.Name] = CheckResult{
				Valid:           false,
				Result:          result,
				Error:           result.Error.Error(),
				ReferenceResult: refResult,
			}
			continue
		}

		// Parse provider's result
		providerValue, err := parseJSONRPCResult(result.Response)
		if err != nil {
			checkResults[provider.Name] = CheckResult{
				Valid:           false,
				Result:          result,
				Error:           fmt.Sprintf("failed to parse provider response: %v", err),
				ReferenceResult: refResult,
			}
			continue
		}

		// Use provided comparison function
		valid := true
		if refResult.Success {
			valid = config.CompareFunc(refValue, providerValue)
		}

		checkResults[provider.Name] = CheckResult{
			Valid:           valid,
			Result:          result,
			ReferenceResult: refResult,
		}
	}

	return checkResults
}

// CheckResult contains the validation result for a provider
type CheckResult struct {
	Valid           bool
	Result          requestsrunner.ProviderResult
	ReferenceResult requestsrunner.ProviderResult
	Error           string
}

// TestMultipleEVMMethods runs multiple EVM method tests and returns results per provider per method
func TestMultipleEVMMethods(
	ctx context.Context,
	methodConfigs []config.EVMMethodTestConfig, // list of method configs
	caller requestsrunner.MethodCaller,
	providers []provider.RPCProvider,
	referenceProvider provider.RPCProvider,
	timeout time.Duration,
) map[string]map[string]CheckResult { // provider -> method -> result
	results := make(map[string]map[string]CheckResult)

	// Initialize result structure
	for _, provider := range providers {
		results[provider.Name] = make(map[string]CheckResult)
	}

	// Run tests for each method
	for _, config := range methodConfigs {
		methodResults := TestEVMMethodWithCaller(ctx, config, caller, providers, referenceProvider, timeout)

		// Store results per provider using method name from config
		for providerName, result := range methodResults {
			if _, exists := results[providerName]; exists {
				results[providerName][config.Method] = result
			} else {
				fmt.Printf("warning: provider %s not found in results map\n", providerName)
			}
		}
	}

	return results
}

// handleReferenceError creates check results for all providers with a given error message
func handleReferenceError(results map[string]requestsrunner.ProviderResult, referenceName, errMsg string) map[string]CheckResult {
	checkResults := make(map[string]CheckResult)
	for name, result := range results {
		if name == referenceName {
			continue
		}
		checkResults[name] = CheckResult{
			Valid:  result.Success,
			Result: result,
			Error:  errMsg,
		}
	}
	return checkResults
}

// ValidateMultipleEVMMethods runs multiple EVM method tests and returns validation summary
func ValidateMultipleEVMMethods(
	ctx context.Context,
	methodConfigs []config.EVMMethodTestConfig,
	caller requestsrunner.MethodCaller,
	providers []provider.RPCProvider,
	referenceProvider provider.RPCProvider,
	timeout time.Duration,
) map[string]ProviderValidationResult {
	// Run all method tests
	methodResults := TestMultipleEVMMethods(ctx, methodConfigs, caller, providers, referenceProvider, timeout)

	// Prepare validation results
	validationResults := make(map[string]ProviderValidationResult)

	for providerName, results := range methodResults {
		// Track failed and successful methods
		failedMethods := make(map[string]FailedMethodResult)
		succeedMethods := make(map[string]SucceedMethodResult)
		allValid := true

		for method, result := range results {
			if !result.Valid {
				allValid = false
				failedMethods[method] = FailedMethodResult{
					Result:          result.Result,
					ReferenceResult: result.ReferenceResult,
				}
			} else {
				succeedMethods[method] = SucceedMethodResult{
					Result:          result.Result,
					ReferenceResult: result.ReferenceResult,
				}
			}
		}

		validationResults[providerName] = ProviderValidationResult{
			Valid:          allValid,
			FailedMethods:  failedMethods,
			SucceedMethods: succeedMethods,
		}
	}

	return validationResults
}

// FailedMethodResult contains details about a failed method test
type FailedMethodResult struct {
	Result          requestsrunner.ProviderResult // Raw result from the provider
	ReferenceResult requestsrunner.ProviderResult // Raw result from the reference provider
}

// SucceedMethodResult contains details about a successful method test
type SucceedMethodResult struct {
	Result          requestsrunner.ProviderResult // Raw result from the provider
	ReferenceResult requestsrunner.ProviderResult // Raw result from the reference provider
}

// ProviderValidationResult contains aggregated validation results for a provider
type ProviderValidationResult struct {
	Valid          bool                           // Overall validation status
	FailedMethods  map[string]FailedMethodResult  // Map of failed test methods to their results
	SucceedMethods map[string]SucceedMethodResult // Map of successful test methods to their results
}

// parseJSONRPCResult extracts the numeric result from a JSON-RPC response
// Returns the parsed big.Int value or an error if parsing fails
func parseJSONRPCResult(response []byte) (*big.Int, error) {
	if len(response) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	var jsonResponse struct {
		Result string `json:"result"`
		Error  struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(response, &jsonResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON-RPC response: %w", err)
	}

	// Check for JSON-RPC error
	if jsonResponse.Error.Code != 0 {
		return nil, fmt.Errorf("JSON-RPC error: %s (code: %d)",
			jsonResponse.Error.Message,
			jsonResponse.Error.Code)
	}

	if jsonResponse.Result == "" {
		return nil, errors.New("empty result in JSON-RPC response")
	}

	// Remove 0x prefix if present
	resultStr := jsonResponse.Result
	if len(resultStr) > 2 && resultStr[0:2] == "0x" {
		resultStr = resultStr[2:]
	}

	value, ok := new(big.Int).SetString(resultStr, 16)
	if !ok {
		return nil, fmt.Errorf("failed to parse result as hex number: %s", resultStr)
	}

	return value, nil
}
