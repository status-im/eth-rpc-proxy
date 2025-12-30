package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/status-im/eth-rpc-proxy/api"
	"github.com/status-im/eth-rpc-proxy/config"
	"github.com/status-im/eth-rpc-proxy/core"
	"github.com/status-im/eth-rpc-proxy/e2e/testutils"
	rpcprovider "github.com/status-im/eth-rpc-proxy/provider"
	requestsrunner "github.com/status-im/eth-rpc-proxy/requests_runner"
	"github.com/status-im/eth-rpc-proxy/scheduler"
	"github.com/stretchr/testify/suite"
)

type E2ETestSuite struct {
	suite.Suite
	cfg           config.CheckerConfig
	providerSetup *testutils.ProviderSetup
}

const (
	testPort        = "8081"
	testConfigFile  = "test_config.json"
	testTempDir     = "testdata"
	shutdownTimeout = 5 * time.Second
)

func (s *E2ETestSuite) SetupSuite() {
	// Create test directory
	err := os.MkdirAll(testTempDir, 0755)
	if err != nil {
		s.FailNow("failed to create test directory", err)
	}

	// Initialize provider setup
	s.providerSetup = testutils.NewProviderSetup()

	// Create test config files
	s.cfg = config.CheckerConfig{
		IntervalSeconds:        1,
		DefaultProvidersPath:   filepath.Join(testTempDir, "default_providers.json"),
		ReferenceProvidersPath: filepath.Join(testTempDir, "reference_providers.json"),
		OutputProvidersPath:    filepath.Join(testTempDir, "output_providers.json"),
		TestsConfigPath:        filepath.Join(testTempDir, "test_methods.json"),
	}

	// Create mock servers and update provider URLs
	basePort := 8545
	defaultProviders := make([]rpcprovider.RPCProvider, 0)
	referenceProviders := make([]rpcprovider.RPCProvider, 0)

	// Create mock servers for default providers
	// Responses for first default provider (different from reference)
	firstProviderResponses := map[string]map[string]interface{}{
		"eth_blockNumber": {
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0x654321",
		},
		"eth_getBalance": {
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0x2000000000000000000",
		},
	}

	// Responses for reference provider
	referenceResponses := map[string]map[string]interface{}{
		"eth_blockNumber": {
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0x123456",
		},
		"eth_getBalance": {
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0x1000000000000000000",
		},
	}

	// First default provider
	s.providerSetup.AddProvider(basePort, firstProviderResponses)
	defaultProviders = append(defaultProviders, rpcprovider.RPCProvider{
		Type:     "test_provider1",
		Name:     "testprovider1",
		URL:      fmt.Sprintf("http://localhost:%d", basePort),
		AuthType: "no-auth",
		ChainID:  1,
	})

	// Second default provider
	s.providerSetup.AddProvider(basePort+2, referenceResponses)
	defaultProviders = append(defaultProviders, rpcprovider.RPCProvider{
		Type:     "test_provider2",
		Name:     "testprovider2",
		URL:      fmt.Sprintf("http://localhost:%d", basePort+2),
		AuthType: "no-auth",
		ChainID:  1,
	})

	// Third default provider that returns errors
	errorResponses := map[string]map[string]interface{}{
		"eth_blockNumber": {
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]interface{}{
				"code":    -32000,
				"message": "server error",
			},
		},
		"eth_getBalance": {
			"jsonrpc": "2.0",
			"id":      1,
			"error": map[string]interface{}{
				"code":    -32000,
				"message": "server error",
			},
		},
	}
	s.providerSetup.AddProvider(basePort+4, errorResponses)
	defaultProviders = append(defaultProviders, rpcprovider.RPCProvider{
		Type:     "test_provider3",
		Name:     "testprovider3",
		URL:      fmt.Sprintf("http://localhost:%d", basePort+4),
		AuthType: "no-auth",
		ChainID:  1,
	})

	// Fourth default provider that returns 404
	s.providerSetup.Add404Provider(basePort + 6)
	defaultProviders = append(defaultProviders, rpcprovider.RPCProvider{
		Type:     "test_provider4",
		Name:     "testprovider4",
		URL:      fmt.Sprintf("http://localhost:%d", basePort+6),
		AuthType: "no-auth",
		ChainID:  1,
	})

	// Fifth default provider that returns malformed JSON
	malformedResponses := map[string]map[string]interface{}{
		"eth_blockNumber": {
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "{invalid json",
		},
		"eth_getBalance": {
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "{invalid json",
		},
	}
	s.providerSetup.AddProvider(basePort+8, malformedResponses)
	defaultProviders = append(defaultProviders, rpcprovider.RPCProvider{
		Type:     "test_provider5",
		Name:     "testprovider5",
		URL:      fmt.Sprintf("http://localhost:%d", basePort+8),
		AuthType: "no-auth",
		ChainID:  1,
	})

	// Create mock server for reference provider
	s.providerSetup.AddProvider(basePort+1, referenceResponses)
	referenceProviders = append(referenceProviders, rpcprovider.RPCProvider{
		Type:     "reference_provider",
		Name:     "reference-testprovider",
		URL:      fmt.Sprintf("http://localhost:%d", basePort+1),
		AuthType: "no-auth",
		ChainID:  1,
	})

	// Start all mock servers
	if err := s.providerSetup.StartAll(); err != nil {
		s.FailNow("failed to start mock servers", err)
	}

	// Write default providers using ChainsConfig
	defaultChains := config.ChainsConfig{
		Chains: []config.ChainConfig{
			{
				Name:      "testchain",
				Network:   "testnet",
				ChainID:   1,
				Providers: defaultProviders,
			},
		},
	}
	err = config.WriteChains(s.cfg.DefaultProvidersPath, defaultChains)
	if err != nil {
		s.FailNow("failed to write default providers", err)
	}

	// Write reference providers using ReferenceChainsConfig
	referenceChains := config.ReferenceChainsConfig{
		Chains: []config.ReferenceChainConfig{
			{
				Name:     "testchain",
				Network:  "testnet",
				ChainId:  1,
				Provider: referenceProviders[0],
			},
		},
	}
	err = config.WriteReferenceChains(s.cfg.ReferenceProvidersPath, referenceChains)
	if err != nil {
		s.FailNow("failed to write reference providers", err)
	}

	// Write test methods
	testMethods := []config.EVMMethodTestJSON{
		{
			Method:        "eth_blockNumber",
			Params:        []interface{}{},
			MaxDifference: "0",
		},
		{
			Method:        "eth_getBalance",
			Params:        []interface{}{"0x0000000000000000000000000000000000000000", "latest"},
			MaxDifference: "0",
		},
	}
	err = config.WriteConfig(filepath.Join(testTempDir, "test_methods.json"), testMethods)
	if err != nil {
		s.FailNow("failed to write test methods", err)
	}

	// Write checker config
	s.writeJSONFile(filepath.Join(testTempDir, testConfigFile), s.cfg)
}

func (s *E2ETestSuite) TearDownSuite() {
	// Stop all mock servers
	if err := s.providerSetup.Close(); err != nil {
		s.T().Logf("error stopping mock servers: %v", err)
	}
	if err := os.RemoveAll(testTempDir); err != nil {
		s.T().Logf("error removing temp directory: %v", err)
	}
}

func (s *E2ETestSuite) writeJSONFile(path string, data interface{}) {
	bytes, err := json.Marshal(data)
	if err != nil {
		s.FailNow("failed to marshal JSON", err)
	}
	err = os.WriteFile(path, bytes, 0644)
	if err != nil {
		s.FailNow("failed to write file", err, path)
	}
}

func TestE2E(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

func (s *E2ETestSuite) TestE2E() {
	// Start application
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create EVM method caller
	caller := requestsrunner.NewRequestsRunner()

	// Create periodic task
	validationTask := scheduler.New(
		time.Duration(s.cfg.IntervalSeconds)*time.Second,
		func() {
			runner, err := core.NewRunnerFromConfig(s.cfg, caller)
			if err != nil {
				fmt.Printf("failed to create runner: %v\n", err)
				return
			}
			runner.Run(context.Background())
		},
	)

	// Start HTTP server
	server := api.New(testPort, s.cfg.OutputProvidersPath, s.cfg.DefaultProvidersPath)
	serverDone := make(chan error)
	go func() {
		serverDone <- server.Start()
	}()

	// Start periodic task
	validationTask.Start()
	defer validationTask.Stop()

	// Wait for first run to complete and output file to be created
	maxWait := 10 * time.Second
	start := time.Now()

	for {
		if time.Since(start) > maxWait {
			s.FailNow("timeout waiting for output file to be created")
		}

		// Check if file exists and has content
		outputBytes, err := os.ReadFile(s.cfg.OutputProvidersPath)
		if err == nil && len(outputBytes) > 0 {
			// Try to parse JSON to ensure it's valid
			var testOutput config.ChainsConfig
			if err := json.Unmarshal(outputBytes, &testOutput); err == nil {
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Test provider connectivity
	s.Run("Test provider is accessible", func() {
		// Test default providers
		client := &http.Client{Timeout: 1 * time.Second}

		// Test first default provider
		_, err := client.Get("http://localhost:8545")
		if err != nil {
			s.Fail("first default provider not accessible", err)
		}

		// Test second default provider
		_, err = client.Get("http://localhost:8547")
		if err != nil {
			s.Fail("second default provider not accessible", err)
		}

		// Test reference provider
		_, err = client.Get("http://localhost:8546")
		if err != nil {
			s.Fail("reference provider not accessible", err)
		}

		// Test third default provider
		_, err = client.Get("http://localhost:8549")
		if err != nil {
			s.Fail("third default provider not accessible", err)
		}

		// Test fourth default provider (404)
		resp, err := client.Get("http://localhost:8551")
		if err != nil {
			s.Fail("fourth default provider not accessible", err)
		}
		s.Equal(http.StatusNotFound, resp.StatusCode)
	})

	// Test HTTP API endpoint
	s.Run("HTTP API returns providers", func() {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/providers", testPort))
		s.NoError(err)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				s.T().Logf("error closing response body: %v", err)
			}
		}()

		s.Equal(http.StatusOK, resp.StatusCode)

		var providers map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&providers)
		s.NoError(err)
		s.NotEmpty(providers)
	})

	// Test output file contains second provider
	s.Run("Output file contains second provider", func() {
		// Read output file
		outputBytes, err := os.ReadFile(s.cfg.OutputProvidersPath)
		s.NoError(err)

		// Parse output
		var output config.ChainsConfig
		err = json.Unmarshal(outputBytes, &output)
		s.NoError(err)

		// Verify second provider exists
		s.NotEmpty(output.Chains)
		providers := output.Chains[0].Providers
		s.Equal(len(providers), 1)
		s.Equal("testprovider2", providers[0].Name)
		s.Equal("test_provider2", providers[0].Type)
		s.Equal("http://localhost:8547", providers[0].URL)
	})

	// Cleanup server
	if err := server.Stop(); err != nil {
		s.T().Logf("error stopping server: %v", err)
	}
	cancel()
	select {
	case err := <-serverDone:
		if err != nil && err != http.ErrServerClosed {
			s.T().Logf("server stopped with error: %v", err)
		}
	case <-time.After(shutdownTimeout):
		s.T().Log("server shutdown timeout")
	}
}
