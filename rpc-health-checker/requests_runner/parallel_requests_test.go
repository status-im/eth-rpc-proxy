package requestsrunner_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/eth-rpc-proxy/provider"
	requestsrunner "github.com/status-im/eth-rpc-proxy/requests_runner"
)

// ParallelCheckProvidersTestSuite defines the test suite for ParallelCheckProviders.
type ParallelCheckProvidersTestSuite struct {
	suite.Suite
}

// getSampleProviders returns a predefined list of RPC providers.
func getSampleProviders() []provider.RPCProvider {
	return []provider.RPCProvider{
		{
			Name:      "Provider1",
			URL:       "https://provider1.example.com",
			AuthType:  provider.NoAuth,
			ChainName: "ethereum",
		},
		{
			Name:      "Provider2",
			URL:       "https://provider2.example.com",
			AuthType:  provider.TokenAuth,
			AuthToken: "dummy_token",
			ChainName: "ethereum",
		},
		{
			Name:         "Provider3",
			URL:          "https://provider3.example.com",
			AuthType:     provider.BasicAuth,
			AuthLogin:    "user",
			AuthPassword: "pass",
			ChainName:    "ethereum",
		},
	}
}

// createChecker is a factory function that returns a checker function based on the test case configuration.
// failProviders maps provider names to the errors they should return.
// delay specifies the simulated processing time for each provider.
func createChecker(failProviders map[string]error, delay time.Duration) requestsrunner.RequestFunc {
	return func(ctx context.Context, provider provider.RPCProvider) requestsrunner.ProviderResult {
		var result requestsrunner.ProviderResult

		// Determine the expected result based on whether the provider should fail.
		if err, shouldFail := failProviders[provider.Name]; shouldFail {
			result = requestsrunner.ProviderResult{
				Success:     false,
				Error:       err,
				Response:    nil,
				Result:      "",
				ElapsedTime: delay,
			}
		} else {
			result = requestsrunner.ProviderResult{
				Success:     true,
				Error:       nil,
				Response:    []byte("OK"),
				Result:      "OK",
				ElapsedTime: delay,
			}
		}

		select {
		case <-time.After(delay):
			return result
		case <-ctx.Done():
			return requestsrunner.ProviderResult{
				Success:     false,
				Error:       ctx.Err(),
				Response:    nil,
				ElapsedTime: 0,
			}
		}
	}
}

// runParallelChecks executes ParallelCheckProviders and returns the results.
func runParallelChecks(ctx context.Context, providers []provider.RPCProvider, timeout time.Duration, checker requestsrunner.RequestFunc) map[string]requestsrunner.ProviderResult {
	resultsChan := make(chan map[string]requestsrunner.ProviderResult)

	go func() {
		results := requestsrunner.ParallelCheckProviders(ctx, providers, timeout, checker)
		resultsChan <- results
	}()

	return <-resultsChan
}

// assertProviderResults verifies that the actual results match the expected outcomes.
func assertProviderResults(suite *ParallelCheckProvidersTestSuite, providers []provider.RPCProvider, results map[string]requestsrunner.ProviderResult, expectedResults map[string]requestsrunner.ProviderResult) {
	assert.Len(suite.T(), results, len(providers), "Expected results for all providers")

	for _, provider := range providers {
		result, exists := results[provider.Name]
		assert.True(suite.T(), exists, "Result for %s should exist", provider.Name)

		expected, ok := expectedResults[provider.Name]
		if ok {
			assert.Equal(suite.T(), expected.Success, result.Success, "Provider %s Success status mismatch", provider.Name)
			if expected.Error != nil {
				assert.NotNil(suite.T(), result.Error, "Provider %s should have an error", provider.Name)
				assert.Equal(suite.T(), expected.Error.Error(), result.Error.Error(), "Provider %s Error mismatch", provider.Name)
			} else {
				assert.Nil(suite.T(), result.Error, "Provider %s should have no error", provider.Name)
			}
			assert.Equal(suite.T(), expected.Response, result.Response, "Provider %s Response mismatch", provider.Name)
			assert.Equal(suite.T(), expected.ElapsedTime, result.ElapsedTime, "Provider %s ElapsedTime mismatch", provider.Name)
		}
	}
}

// TestParallelCheckProviders runs all table-driven test cases for ParallelCheckProviders.
func (suite *ParallelCheckProvidersTestSuite) TestParallelCheckProviders() {
	testCases := []struct {
		name            string
		providers       []provider.RPCProvider
		failProviders   map[string]error
		delay           time.Duration
		timeout         time.Duration
		expectedResults map[string]requestsrunner.ProviderResult
	}{
		{
			name:          "AllProvidersSuccess",
			providers:     getSampleProviders(),
			failProviders: map[string]error{
				// No failures
			},
			delay:   10 * time.Millisecond,
			timeout: 1 * time.Second,
			expectedResults: map[string]requestsrunner.ProviderResult{
				"Provider1": {Success: true, Error: nil, Response: []byte("OK"), Result: "OK", ElapsedTime: 10 * time.Millisecond},
				"Provider2": {Success: true, Error: nil, Response: []byte("OK"), Result: "OK", ElapsedTime: 10 * time.Millisecond},
				"Provider3": {Success: true, Error: nil, Response: []byte("OK"), Result: "OK", ElapsedTime: 10 * time.Millisecond},
			},
		},
		{
			name:      "SomeProvidersFail",
			providers: getSampleProviders(),
			failProviders: map[string]error{
				"Provider2": errors.New("connection timeout"),
			},
			delay:   10 * time.Millisecond,
			timeout: 1 * time.Second,
			expectedResults: map[string]requestsrunner.ProviderResult{
				"Provider1": {Success: true, Error: nil, Response: []byte("OK"), ElapsedTime: 10 * time.Millisecond},
				"Provider2": {Success: false, Error: errors.New("connection timeout"), Response: nil, ElapsedTime: 10 * time.Millisecond},
				"Provider3": {Success: true, Error: nil, Response: []byte("OK"), ElapsedTime: 10 * time.Millisecond},
			},
		},
		{
			name:          "OverallTimeout",
			providers:     getSampleProviders(),
			failProviders: map[string]error{
				// No failures, but delay causes timeout
			},
			delay:   2 * time.Second,
			timeout: 50 * time.Millisecond,
			expectedResults: map[string]requestsrunner.ProviderResult{
				"Provider1": {Success: false, Error: errors.New("context deadline exceeded"), Response: nil, ElapsedTime: 0},
				"Provider2": {Success: false, Error: errors.New("context deadline exceeded"), Response: nil, ElapsedTime: 0},
				"Provider3": {Success: false, Error: errors.New("context deadline exceeded"), Response: nil, ElapsedTime: 0},
			},
		},
		{
			name:      "PartialSuccess",
			providers: getSampleProviders(),
			failProviders: map[string]error{
				"Provider2": errors.New("authentication failed"),
			},
			delay:   20 * time.Millisecond,
			timeout: 2 * time.Second,
			expectedResults: map[string]requestsrunner.ProviderResult{
				"Provider1": {Success: true, Error: nil, Response: []byte("OK"), ElapsedTime: 20 * time.Millisecond},
				"Provider2": {Success: false, Error: errors.New("authentication failed"), Response: nil, ElapsedTime: 20 * time.Millisecond},
				"Provider3": {Success: true, Error: nil, Response: []byte("OK"), ElapsedTime: 20 * time.Millisecond},
			},
		},
		{
			name:            "NoProviders",
			providers:       []provider.RPCProvider{},
			failProviders:   map[string]error{},
			delay:           10 * time.Millisecond,
			timeout:         1 * time.Second,
			expectedResults: map[string]requestsrunner.ProviderResult{},
		},
		{
			name: "InvalidAuthType",
			providers: []provider.RPCProvider{
				{
					Name:      "Provider1",
					URL:       "https://provider1.example.com",
					AuthType:  provider.RPCProviderAuthType("invalid-auth"), // Assuming RPCProviderAuthType is a string alias
					AuthToken: "",
					ChainName: "ethereum",
				},
			},
			failProviders: map[string]error{
				"Provider1": errors.New("unknown authentication type"),
			},
			delay:   5 * time.Millisecond,
			timeout: 1 * time.Second,
			expectedResults: map[string]requestsrunner.ProviderResult{
				"Provider1": {Success: false, Error: errors.New("unknown authentication type"), Response: nil, ElapsedTime: 5 * time.Millisecond},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		suite.Run(tc.name, func() {
			// Create the checker function using the factory
			checker := createChecker(tc.failProviders, tc.delay)

			// Perform parallel checks with specified timeout
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			results := runParallelChecks(ctx, tc.providers, tc.timeout, checker)
			assertProviderResults(suite, tc.providers, results, tc.expectedResults)
		})
	}
}

func TestParallelCallEVMMethods(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","result":"0x1"}`))
	}))
	defer server.Close()

	// Create test providers
	providers := []provider.RPCProvider{
		{
			Name:      "Provider1",
			URL:       server.URL,
			AuthType:  provider.NoAuth,
			ChainName: "testchain",
		},
		{
			Name:      "Provider2",
			URL:       server.URL,
			AuthType:  provider.NoAuth,
			ChainName: "testchain",
		},
	}

	// Test successful parallel execution
	t.Run("SuccessfulExecution", func(t *testing.T) {
		ctx := context.Background()
		runner := requestsrunner.NewRequestsRunner()
		results := requestsrunner.ParallelCallEVMMethods(ctx, providers, "eth_blockNumber", nil, 1*time.Second, runner)

		assert.Len(t, results, len(providers))
		for _, provider := range providers {
			result, exists := results[provider.Name]
			assert.True(t, exists)
			assert.True(t, result.Success)
			assert.Equal(t, []byte(`{"jsonrpc":"2.0","result":"0x1"}`), result.Response)
			assert.Nil(t, result.Error)
		}
	})

	// Test timeout
	t.Run("Timeout", func(t *testing.T) {
		// Create a slow test server that responds after 100ms
		slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","result":"0x1"}`))
		}))
		defer slowServer.Close()

		// Create providers pointing to the slow server
		slowProviders := []provider.RPCProvider{
			{
				Name:      "SlowProvider1",
				URL:       slowServer.URL,
				AuthType:  provider.NoAuth,
				ChainName: "testchain",
			},
			{
				Name:      "SlowProvider2",
				URL:       slowServer.URL,
				AuthType:  provider.NoAuth,
				ChainName: "testchain",
			},
		}

		ctx := context.Background()
		runner := requestsrunner.NewRequestsRunner()
		results := requestsrunner.ParallelCallEVMMethods(ctx, slowProviders, "eth_blockNumber", nil, 10*time.Millisecond, runner)

		assert.Len(t, results, len(slowProviders))
		for _, provider := range slowProviders {
			result, exists := results[provider.Name]
			assert.True(t, exists)
			assert.False(t, result.Success, "Expected timeout failure for provider %s", provider.Name)
			assert.Contains(t, result.Error.Error(), "context deadline exceeded", "Expected timeout error for provider %s", provider.Name)
		}
	})

	// Test empty providers
	t.Run("EmptyProviders", func(t *testing.T) {
		ctx := context.Background()
		runner := requestsrunner.NewRequestsRunner()
		results := requestsrunner.ParallelCallEVMMethods(ctx, []provider.RPCProvider{}, "eth_blockNumber", nil, 1*time.Second, runner)
		assert.Empty(t, results)
	})
}

// TestParallelCheckProvidersContextCancellation tests handling of context cancellation.
func (suite *ParallelCheckProvidersTestSuite) TestParallelCheckProvidersContextCancellation() {
	// Use getSampleProviders to define providers
	providers := getSampleProviders()

	checker := createChecker(map[string]error{}, 1*time.Second)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timeout := 2 * time.Second // Set timeout longer than checker delay to allow manual cancellation

	// Perform parallel checks in a separate goroutine
	resultsChan := make(chan map[string]requestsrunner.ProviderResult)
	go func() {
		results := requestsrunner.ParallelCheckProviders(ctx, providers, timeout, checker)
		resultsChan <- results
	}()

	// Cancel the context before all checks complete
	time.Sleep(10 * time.Millisecond) // Sleep briefly to ensure ParallelCheckProviders has started
	cancel()

	// Receive the results
	results := <-resultsChan

	// Define expected results: All providers fail due to context cancellation
	expectedResults := make(map[string]requestsrunner.ProviderResult)
	for _, provider := range providers {
		expectedResults[provider.Name] = requestsrunner.ProviderResult{
			Success:     false,
			Error:       errors.New("context canceled"),
			Response:    nil,
			ElapsedTime: 0,
		}
	}

	// Assert that results are as expected using assertProviderResults
	assertProviderResults(suite, providers, results, expectedResults)
}

// Run the test suite
func TestParallelCheckProvidersTestSuite(t *testing.T) {
	suite.Run(t, new(ParallelCheckProvidersTestSuite))
}

func TestCallEVMMethod(t *testing.T) {
	tests := []struct {
		name        string
		provider    provider.RPCProvider
		method      string
		params      []interface{}
		handler     func(http.ResponseWriter, *http.Request)
		wantSuccess bool
		wantResult  string
		wantError   string
	}{
		{
			name: "Successful NoAuth request",
			provider: provider.RPCProvider{
				Name:      "test",
				URL:       "", // Will be set to test server URL
				AuthType:  provider.NoAuth,
				ChainName: "testchain",
			},
			method: "eth_blockNumber",
			params: []interface{}{},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var req map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&req)
				assert.NoError(t, err)
				assert.Equal(t, "2.0", req["jsonrpc"])
				assert.Equal(t, "eth_blockNumber", req["method"])
				assert.Equal(t, []interface{}{}, req["params"])
				assert.Equal(t, float64(1), req["id"])

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jsonrpc":"2.0","result":"0x1"}`))
			},
			wantSuccess: true,
			wantResult:  "0x1",
		},
		{
			name: "Successful BasicAuth request",
			provider: provider.RPCProvider{
				Name:         "test",
				URL:          "", // Will be set to test server URL
				AuthType:     provider.BasicAuth,
				AuthLogin:    "user",
				AuthPassword: "pass",
				ChainName:    "testchain",
			},
			method: "eth_getBalance",
			params: []interface{}{"0x123", "latest"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				assert.True(t, ok)
				assert.Equal(t, "user", user)
				assert.Equal(t, "pass", pass)

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jsonrpc":"2.0","result":"0x100"}`))
			},
			wantSuccess: true,
			wantResult:  "0x100",
		},
		{
			name: "Successful TokenAuth request",
			provider: provider.RPCProvider{
				Name:      "test",
				URL:       "", // Will be set to test server URL
				AuthType:  provider.TokenAuth,
				AuthToken: "test-token",
			},
			method: "eth_chainId",
			params: []interface{}{},
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Verify token is in URL path
				assert.Contains(t, r.URL.String(), "test-token")

				// Verify no Authorization header
				assert.Empty(t, r.Header.Get("Authorization"))

				// Verify JSON-RPC request
				var req map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&req)
				assert.NoError(t, err)
				assert.Equal(t, "2.0", req["jsonrpc"])
				assert.Equal(t, "eth_chainId", req["method"])
				assert.Equal(t, []interface{}{}, req["params"])
				assert.Equal(t, float64(1), req["id"])

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jsonrpc":"2.0","result":"0x1"}`))
			},
			wantSuccess: true,
			wantResult:  "0x1",
		},
		{
			name: "Server error response",
			provider: provider.RPCProvider{
				Name:     "test",
				URL:      "", // Will be set to test server URL
				AuthType: provider.NoAuth,
			},
			method: "eth_blockNumber",
			params: []interface{}{},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantSuccess: false,
			wantError:   "500",
		},
		{
			name: "Invalid JSON response",
			provider: provider.RPCProvider{
				Name:     "test",
				URL:      "", // Will be set to test server URL
				AuthType: provider.NoAuth,
			},
			method: "eth_blockNumber",
			params: []interface{}{},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`invalid json`))
			},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tt.handler))
			defer server.Close()

			// Update provider URL
			tt.provider.URL = server.URL

			// Call the method
			runner := requestsrunner.NewRequestsRunner()
			result := runner.CallMethod(context.Background(), tt.provider, tt.method, tt.params, 1*time.Second)

			// Verify results
			assert.Equal(t, tt.wantSuccess, result.Success)
			if tt.wantResult != "" {
				assert.Equal(t, tt.wantResult, result.Result)
			}
			if tt.wantError != "" {
				assert.Contains(t, result.Error.Error(), tt.wantError)
			}
		})
	}
}
