package main

import (
	"os"
	"strings"

	"go.uber.org/zap"
)

// GetKeyDBURL returns KeyDB URL with the following priority:
// 1. KEYDB_URL environment variable
// 2. CACHE_KEYDB_URL_FILE file content
// 3. Default value
func GetKeyDBURL(logger *zap.Logger) (string, error) {
	// Priority 1: Environment variable
	if keydbURL := os.Getenv("KEYDB_URL"); keydbURL != "" {
		logger.Debug("Using KeyDB URL from environment variable")
		return keydbURL, nil
	}

	// Priority 2: Configurable connection file path
	connectionFile := os.Getenv("CACHE_KEYDB_URL_FILE")
	if connectionFile == "" {
		connectionFile = "/app/.keydb-url"
	}

	if content, err := os.ReadFile(connectionFile); err == nil {
		keydbURL := strings.TrimSpace(string(content))
		if len(keydbURL) > 0 {
			logger.Debug("Using KeyDB URL from connection file", zap.String("file", connectionFile))
			return keydbURL, nil
		}
	} else {
		logger.Debug("KeyDB connection file not found or empty", zap.String("file", connectionFile))
	}

	// Priority 3: Default
	logger.Debug("Using default KeyDB URL")
	return "redis://keydb:6379", nil
}
