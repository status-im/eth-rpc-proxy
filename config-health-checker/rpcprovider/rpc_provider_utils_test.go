package rpcprovider

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// RpcProviderTestSuite defines the structure of the test suite
type RpcProviderTestSuite struct {
	suite.Suite
	tempDir             string
	tempFile            string
	validJSON           string
	invalidJSON         string
	invalidAuthTypeJSON string
}

// SetupSuite is executed before running all tests in the suite
func (suite *RpcProviderTestSuite) SetupSuite() {
	// Create a temporary directory for tests
	dir, err := os.MkdirTemp("", "rpc_provider_test")
	if err != nil {
		suite.T().Fatalf("Failed to create temp dir: %v", err)
	}
	suite.tempDir = dir

	// Path to temporary files
	suite.tempFile = filepath.Join(suite.tempDir, "providers_test.json")

	// Define valid JSON
	suite.validJSON = `{
  "providers": [
    {
      "name": "InfuraMainnet",
      "url": "https://mainnet.infura.io/v3",
      "enabled": true,
      "authType": "token-auth",
      "authToken": "infura-token"
    },
    {
      "name": "AlchemyMainnet",
      "url": "https://eth-mainnet.alchemyapi.io/v2",
      "enabled": true,
      "authType": "token-auth",
      "authToken": "alchemy-token"
    },
    {
      "name": "Example",
      "url": "https://another-provider.example.io/v2",
      "enabled": true,
      "authType": "no-auth"
    }
  ]
}`

	// Define invalid JSON
	suite.invalidJSON = `{
  "providers": [
    {
      "name": "BadProvider",
      "url": "https://bad-provider.example.io",
      "enabled": true,
      "authType": "no-auth"
    }
  ]` // Note the missing comma and closing brace

	suite.invalidAuthTypeJSON = `{	
		"providers": [{
			"name": "InfuraMainnet",
			"url": "https://mainnet.infura.io/v3",
			"enabled": true,
			"authType": "invalid-auth"
		}]
	}`
}

// TearDownSuite is executed after all tests in the suite
func (suite *RpcProviderTestSuite) TearDownSuite() {
	// Remove the temporary directory and all its contents
	os.RemoveAll(suite.tempDir)
}

// SetupTest is executed before each test
func (suite *RpcProviderTestSuite) SetupTest() {
	// Clear the file before each test if it exists
	if _, err := os.Stat(suite.tempFile); err == nil {
		os.Remove(suite.tempFile)
	}
}

// TearDownTest is executed after each test
func (suite *RpcProviderTestSuite) TearDownTest() {
	// Additional actions can be added after each test if necessary
}

// TestReadRpcProvidersSuccess checks successful reading of a valid JSON file
func (suite *RpcProviderTestSuite) TestReadRpcProvidersSuccess() {
	// Write valid JSON to the temporary file
	err := os.WriteFile(suite.tempFile, []byte(suite.validJSON), 0644)
	suite.Require().NoError(err, "Failed to write valid JSON to temp file")

	// Read providers from the file
	providers, err := ReadRpcProviders(suite.tempFile)
	suite.Require().NoError(err, "ReadRpcProviders() returned an error")

	// Check the number of providers
	suite.Equal(3, len(providers), "Expected 3 providers")

	// Check the fields of the first provider
	first := providers[0]
	suite.Equal("InfuraMainnet", first.Name, "First provider name mismatch")
	suite.Equal("https://mainnet.infura.io/v3", first.URL, "First provider URL mismatch")
	suite.Equal(TokenAuth, first.AuthType, "First provider AuthType mismatch")
	suite.Equal("infura-token", first.AuthToken, "First provider AuthToken mismatch")
}

// TestReadRpcProvidersFileNotFound checks that the function returns an error for a non-existent file
func (suite *RpcProviderTestSuite) TestReadRpcProvidersFileNotFound() {
	_, err := ReadRpcProviders(filepath.Join(suite.tempDir, "non_existent.json"))
	suite.Error(err, "Expected error for non-existent file")
}

// TestReadRpcProvidersInvalidJSON checks that the function returns an error for invalid JSON
func (suite *RpcProviderTestSuite) TestReadRpcProvidersInvalidJSON() {
	// Write invalid JSON to the temporary file
	err := ioutil.WriteFile(suite.tempFile, []byte(suite.invalidJSON), 0644)
	suite.Require().NoError(err, "Failed to write invalid JSON to temp file")

	// Attempt to read providers from the file
	_, err = ReadRpcProviders(suite.tempFile)
	suite.Error(err, "Expected JSON parse error")
}

// TestInvalidAuthTypeJSON checks that the function returns an error for invalid JSON
func (suite *RpcProviderTestSuite) TestInvalidAuthTypeJSON() {
	err := ioutil.WriteFile(suite.tempFile, []byte(suite.invalidAuthTypeJSON), 0644)
	suite.Require().NoError(err, "Failed to write invalid JSON to temp file")

	// Attempt to read providers from the file
	_, err = ReadRpcProviders(suite.tempFile)
	suite.Error(err, "Expected JSON parse error")
}

// TestWriteRpcProvidersAndReadBack checks that writing and subsequent reading of providers works correctly
func (suite *RpcProviderTestSuite) TestWriteRpcProvidersAndReadBack() {
	// Create test providers
	wantProviders := []RpcProvider{
		{
			Name:      "TestProvider1",
			URL:       "https://test1.example.com",
			AuthType:  NoAuth,
			AuthToken: "",
		},
		{
			Name:      "TestProvider2",
			URL:       "https://test2.example.com",
			AuthType:  TokenAuth,
			AuthToken: "dummy_token",
		},
	}

	// Write providers to the file
	err := WriteRpcProviders(suite.tempFile, wantProviders)
	suite.Require().NoError(err, "WriteRpcProviders() returned an error")

	// Read providers from the file
	gotProviders, err := ReadRpcProviders(suite.tempFile)
	suite.Require().NoError(err, "ReadRpcProviders() returned an error")

	// Use assert to compare
	assert.Equal(suite.T(), wantProviders, gotProviders, "Providers read from file do not match written providers")
}

// TestWriteRpcProvidersHandlesEmptyList checks that the function correctly handles an empty list of providers
func (suite *RpcProviderTestSuite) TestWriteRpcProvidersHandlesEmptyList() {
	// Write an empty list of providers to the file
	err := WriteRpcProviders(suite.tempFile, []RpcProvider{})
	suite.Require().NoError(err, "WriteRpcProviders() returned an error for empty list")

	// Read providers from the file
	gotProviders, err := ReadRpcProviders(suite.tempFile)
	suite.Require().NoError(err, "ReadRpcProviders() returned an error for empty list")

	// Check that the list is empty
	suite.Empty(gotProviders, "Expected no providers in the file")
}

// TestWriteRpcProvidersInvalidPath checks that the function returns an error when trying to write to an invalid path
func (suite *RpcProviderTestSuite) TestWriteRpcProvidersInvalidPath() {
	// Use an invalid path (e.g., a directory instead of a file)
	err := WriteRpcProviders(suite.tempDir, []RpcProvider{})
	suite.Error(err, "Expected error when writing to a directory path")
}

// Run the test suite
func TestRpcProviderTestSuite(t *testing.T) {
	suite.Run(t, new(RpcProviderTestSuite))
}
