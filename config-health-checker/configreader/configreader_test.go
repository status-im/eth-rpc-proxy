package configreader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configJSON  string
		expectError bool
		expected    *CheckerConfig
	}{
		{
			name: "valid config",
			configJSON: `{
				"interval_seconds": 30,
				"default_providers_path": "custom_default.json",
				"reference_providers_path": "custom_reference.json",
				"output_providers_path": "custom_output.json",
				"tests_config_path": "custom_tests.json"
			}`,
			expectError: false,
			expected: &CheckerConfig{
				IntervalSeconds:        30,
				DefaultProvidersPath:   "custom_default.json",
				ReferenceProvidersPath: "custom_reference.json",
				OutputProvidersPath:    "custom_output.json",
				TestsConfigPath:        "custom_tests.json",
			},
		},
		{
			name: "default values",
			configJSON: `{
				"interval_seconds": 0,
				"default_providers_path": "",
				"reference_providers_path": "",
				"output_providers_path": "",
				"tests_config_path": ""
			}`,
			expectError: false,
			expected: &CheckerConfig{
				IntervalSeconds:        defaultIntervalSeconds,
				DefaultProvidersPath:   filepath.Join(defaultConfigDir, "default_providers.json"),
				ReferenceProvidersPath: filepath.Join(defaultConfigDir, "reference_providers.json"),
				OutputProvidersPath:    filepath.Join(defaultConfigDir, "providers.json"),
				TestsConfigPath:        filepath.Join(defaultConfigDir, "tests_config.json"),
			},
		},
		{
			name:        "invalid JSON",
			configJSON:  `{invalid}`,
			expectError: true,
		},
		{
			name:        "empty path",
			configJSON:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file for test
			tmpFile, err := os.CreateTemp("", "config_*.json")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if tt.configJSON != "" {
				if _, err := tmpFile.WriteString(tt.configJSON); err != nil {
					t.Fatalf("Failed to write test config: %v", err)
				}
			}
			tmpFile.Close()

			// Test ReadConfig
			config, err := ReadConfig(tmpFile.Name())
			if (err != nil) != tt.expectError {
				t.Errorf("ReadConfig() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				if config.IntervalSeconds != tt.expected.IntervalSeconds {
					t.Errorf("IntervalSeconds = %v, want %v", config.IntervalSeconds, tt.expected.IntervalSeconds)
				}
				if config.DefaultProvidersPath != tt.expected.DefaultProvidersPath {
					t.Errorf("DefaultProvidersPath = %v, want %v", config.DefaultProvidersPath, tt.expected.DefaultProvidersPath)
				}
				if config.ReferenceProvidersPath != tt.expected.ReferenceProvidersPath {
					t.Errorf("ReferenceProvidersPath = %v, want %v", config.ReferenceProvidersPath, tt.expected.ReferenceProvidersPath)
				}
				if config.OutputProvidersPath != tt.expected.OutputProvidersPath {
					t.Errorf("OutputProvidersPath = %v, want %v", config.OutputProvidersPath, tt.expected.OutputProvidersPath)
				}
				if config.TestsConfigPath != tt.expected.TestsConfigPath {
					t.Errorf("TestsConfigPath = %v, want %v", config.TestsConfigPath, tt.expected.TestsConfigPath)
				}
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *CheckerConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &CheckerConfig{
				IntervalSeconds:        60,
				DefaultProvidersPath:   "default.json",
				ReferenceProvidersPath: "reference.json",
				OutputProvidersPath:    "output.json",
				TestsConfigPath:        "tests.json",
				LogsPath:               "logs",
			},
			expectError: false,
		},
		{
			name: "invalid interval",
			config: &CheckerConfig{
				IntervalSeconds:        -1,
				DefaultProvidersPath:   "default.json",
				ReferenceProvidersPath: "reference.json",
				OutputProvidersPath:    "output.json",
				TestsConfigPath:        "tests.json",
			},
			expectError: true,
		},
		{
			name: "missing paths",
			config: &CheckerConfig{
				IntervalSeconds:        30,
				DefaultProvidersPath:   "",
				ReferenceProvidersPath: "",
				OutputProvidersPath:    "",
				TestsConfigPath:        "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.expectError {
				t.Errorf("validateConfig() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
