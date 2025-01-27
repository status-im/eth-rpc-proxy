package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	rpcprovider "github.com/status-im/eth-rpc-proxy/provider"
)

const (
	minProviders = 1
	configKey    = "chains"
)

// ChainConfigurer defines common behavior for chain configurations
type ChainConfigurer interface {
	GetName() string
	GetNetwork() string
	GetChainID() int
	Validate() error
	Normalize()
}

// ChainsConfig represents a collection of chain configurations
type ChainsConfig struct {
	Chains []ChainConfig `json:"chains" validate:"required,dive"`
}

// ReferenceChainsConfig represents a collection of reference chain configurations
type ReferenceChainsConfig struct {
	Chains []ReferenceChainConfig `json:"chains" validate:"required,dive"`
}

// ChainConfig represents configuration for a blockchain network
type ChainConfig struct {
	Name      string                    `json:"name" validate:"required,lowercase"`
	Network   string                    `json:"network" validate:"required,lowercase"`
	ChainID   int                       `json:"chainId" validate:"required"`
	Providers []rpcprovider.RPCProvider `json:"providers" validate:"required,dive"`
}

// GetName returns the chain name
func (c ChainConfig) GetName() string {
	return c.Name
}

// GetNetwork returns the chain network
func (c ChainConfig) GetNetwork() string {
	return c.Network
}

// GetChainID returns the chain ID
func (c ChainConfig) GetChainID() int {
	return c.ChainID
}

// Validate validates the chain configuration
func (c ChainConfig) Validate() error {
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("invalid chain configuration: %w", err)
	}
	if len(c.Providers) < minProviders {
		return fmt.Errorf("at least %d provider(s) required", minProviders)
	}
	return nil
}

// ReferenceChainConfig represents configuration for reference providers
type ReferenceChainConfig struct {
	Name     string                  `json:"name" validate:"required,lowercase"`
	Network  string                  `json:"network" validate:"required,lowercase"`
	ChainId  int                     `json:"chainId" validate:"required"`
	Provider rpcprovider.RPCProvider `json:"provider" validate:"required"`
}

// GetName returns the chain name
func (c ReferenceChainConfig) GetName() string {
	return c.Name
}

// GetNetwork returns the chain network
func (c ReferenceChainConfig) GetNetwork() string {
	return c.Network
}

// GetChainID returns the chain ID
func (c ReferenceChainConfig) GetChainID() int {
	return c.ChainId
}

// Validate validates the reference chain configuration
func (c ReferenceChainConfig) Validate() error {
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("invalid reference chain configuration: %w", err)
	}
	if c.Provider.Name == "" {
		return errors.New("provider name is required")
	}
	if c.Provider.URL == "" {
		return errors.New("provider URL is required")
	}
	return nil
}

// LoadChains loads and validates chain configurations from a JSON file
func LoadChains(filePath string) (ChainsConfig, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return ChainsConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ChainsConfig
	if err := json.Unmarshal(file, &config); err != nil {
		return ChainsConfig{}, fmt.Errorf("failed to parse chains config: %w", err)
	}

	if len(config.Chains) == 0 {
		return ChainsConfig{}, errors.New("no chains configured")
	}

	// Validate and normalize each chain
	for i := range config.Chains {
		config.Chains[i].normalize()
		if err := config.Chains[i].Validate(); err != nil {
			return ChainsConfig{}, fmt.Errorf("invalid chain configuration: %w", err)
		}
	}

	return config, nil
}

// LoadReferenceChains loads and validates reference provider configurations from a JSON file
func LoadReferenceChains(filePath string) (ReferenceChainsConfig, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return ReferenceChainsConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ReferenceChainsConfig
	if err := json.Unmarshal(file, &config); err != nil {
		return ReferenceChainsConfig{}, fmt.Errorf("failed to parse reference chains config: %w", err)
	}

	if len(config.Chains) == 0 {
		return ReferenceChainsConfig{}, errors.New("no reference chains configured")
	}

	// Validate and normalize each reference chain
	for i := range config.Chains {
		config.Chains[i].normalize()
		// Set ChainID for the provider
		config.Chains[i].Provider.ChainID = int64(config.Chains[i].ChainId)
		if err := config.Chains[i].Validate(); err != nil {
			return ReferenceChainsConfig{}, fmt.Errorf("invalid reference chain configuration: %w", err)
		}
	}

	return config, nil
}

// GetChainByNameAndNetwork finds a chain by name and network
func GetChainByNameAndNetwork(chains []ChainConfig, name, network string) (*ChainConfig, error) {
	for _, chain := range chains {
		if chain.Name == name && chain.Network == network {
			return &chain, nil
		}
	}
	return nil, fmt.Errorf("chain %s (%s) not found", name, network)
}

// GetReferenceProvider finds a reference provider by name and network
func GetReferenceProvider(chains []ReferenceChainConfig, name, network string) (*rpcprovider.RPCProvider, error) {
	for _, chain := range chains {
		if chain.Name == name && chain.Network == network {
			return &chain.Provider, nil
		}
	}
	return nil, fmt.Errorf("reference provider for %s (%s) not found", name, network)
}

// normalize ensures chain name and network are lowercase
func (c *ChainConfig) normalize() {
	c.Name = strings.ToLower(c.Name)
	c.Network = strings.ToLower(c.Network)
}

// normalize ensures reference chain name and network are lowercase
func (c *ReferenceChainConfig) normalize() {
	c.Name = strings.ToLower(c.Name)
	c.Network = strings.ToLower(c.Network)
}

// WriteChains writes chain configurations to a JSON file
func WriteChains(filePath string, config ChainsConfig) error {
	// Validate each chain configuration
	for _, chain := range config.Chains {
		if err := validateChainConfig(chain); err != nil {
			return fmt.Errorf("invalid chain configuration: %w", err)
		}
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal chains: %w", err)
	}

	// Write to file with proper permissions
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write chains file: %w", err)
	}

	return nil
}

// WriteReferenceChains writes reference chain configurations to a JSON file
func WriteReferenceChains(filePath string, config ReferenceChainsConfig) error {
	// Validate each reference chain configuration
	for _, chain := range config.Chains {
		if err := chain.Validate(); err != nil {
			return fmt.Errorf("invalid reference chain configuration: %w", err)
		}
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal reference chains: %w", err)
	}

	// Write to file with proper permissions
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write reference chains file: %w", err)
	}

	return nil
}

var validate = validator.New()

// validateChainConfig validates required fields in chain configuration
func validateChainConfig(chain ChainConfig) error {
	// Validate struct fields
	if err := validate.Struct(chain); err != nil {
		return err
	}

	// Additional custom validation
	if len(chain.Providers) == 0 {
		return errors.New("at least one provider is required")
	}

	return nil
}
