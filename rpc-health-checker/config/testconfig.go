package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
)

// EVMMethodTestConfig contains configuration for testing an EVM method
type EVMMethodTestConfig struct {
	Method      string
	Params      []interface{}
	CompareFunc func(reference, result *big.Int) bool
	SkipChains  map[int64]bool // Chains to skip test (chainID -> true)
}

// EVMMethodTestJSON represents the JSON structure for EVM method test configuration
type EVMMethodTestJSON struct {
	Method        string        `json:"method"`
	Params        []interface{} `json:"params"`
	MaxDifference string        `json:"maxDifference"`
	SkipChains    []int64       `json:"skipChains,omitempty"` // Chain IDs to skip test
}

// ReadConfig reads and parses the EVM method test configuration from a JSON file
func ReadConfig(path string) ([]EVMMethodTestConfig, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var testConfigs []EVMMethodTestJSON
	if err := json.Unmarshal(data, &testConfigs); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert to EVMMethodTestConfig
	var configs []EVMMethodTestConfig
	for _, cfg := range testConfigs {
		// Parse max difference
		maxDiff, ok := new(big.Int).SetString(cfg.MaxDifference, 10)
		if !ok {
			return nil, fmt.Errorf("invalid maxDifference value: %s", cfg.MaxDifference)
		}

		// Create comparison function
		compareFunc := func(reference, result *big.Int) bool {
			diff := new(big.Int).Abs(new(big.Int).Sub(reference, result))
			return diff.Cmp(maxDiff) <= 0
		}

		skipChainsMap := make(map[int64]bool)
		for _, chainID := range cfg.SkipChains {
			skipChainsMap[chainID] = true
		}

		configs = append(configs, EVMMethodTestConfig{
			Method:      cfg.Method,
			Params:      cfg.Params,
			CompareFunc: compareFunc,
			SkipChains:  skipChainsMap,
		})
	}

	return configs, nil
}

// ValidateConfig validates the test configuration
func ValidateConfig(configs []EVMMethodTestConfig) error {
	if len(configs) == 0 {
		return errors.New("empty test configuration")
	}

	for _, cfg := range configs {
		if cfg.Method == "" {
			return errors.New("method name cannot be empty")
		}
	}

	return nil
}

// WriteConfig writes the given EVM method test configuration to a JSON file
func WriteConfig(path string, configs []EVMMethodTestJSON) error {
	bytes, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(path, bytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
