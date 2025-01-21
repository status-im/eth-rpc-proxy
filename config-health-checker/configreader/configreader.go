package configreader

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultIntervalSeconds = 60
	defaultConfigDir       = "."
)

// CheckerConfig represents the configuration for the health checker
type CheckerConfig struct {
	IntervalSeconds        int    `json:"interval_seconds"`         // Interval between health checks in seconds
	DefaultProvidersPath   string `json:"default_providers_path"`   // Path to default providers JSON file
	ReferenceProvidersPath string `json:"reference_providers_path"` // Path to reference providers JSON file
	OutputProvidersPath    string `json:"output_providers_path"`    // Path to output providers JSON file
	TestsConfigPath        string `json:"tests_config_path"`        // Path to tests configuration JSON file
	LogsPath               string `json:"logs_path"`                // Path to store log files
}

// ReadConfig reads and validates the configuration from the specified path
func ReadConfig(path string) (*CheckerConfig, error) {
	if path == "" {
		return nil, errors.New("config path cannot be empty")
	}

	configData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config CheckerConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Set default values and validate
	if config.IntervalSeconds <= 0 {
		config.IntervalSeconds = defaultIntervalSeconds
	}

	config.DefaultProvidersPath = resolvePath(config.DefaultProvidersPath, "default_providers.json")
	config.ReferenceProvidersPath = resolvePath(config.ReferenceProvidersPath, "reference_providers.json")
	config.OutputProvidersPath = resolvePath(config.OutputProvidersPath, "providers.json")
	config.TestsConfigPath = resolvePath(config.TestsConfigPath, "tests_config.json")
	config.LogsPath = resolvePath(config.LogsPath, "logs")

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// resolvePath returns the default path if the provided path is empty,
// otherwise returns the absolute path of the provided path
func resolvePath(path, defaultPath string) string {
	if path == "" {
		return filepath.Join(defaultConfigDir, defaultPath)
	}
	return filepath.Clean(path)
}

// validateConfig performs validation of the configuration values
func validateConfig(config *CheckerConfig) error {
	if config.IntervalSeconds <= 0 {
		return errors.New("interval_seconds must be positive")
	}

	if config.DefaultProvidersPath == "" || config.ReferenceProvidersPath == "" ||
		config.OutputProvidersPath == "" || config.TestsConfigPath == "" || config.LogsPath == "" {
		return errors.New("all paths must be specified")
	}

	return nil
}
