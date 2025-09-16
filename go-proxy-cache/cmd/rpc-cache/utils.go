package main

import (
	"os"
	"strings"

	"go.uber.org/zap"
)

const (
	DefaultKeyDBURL = "redis://keydb:6379"
)

// GetKeyDBURL returns KeyDB URL with the following priority:
// 1. KEYDB_URL environment variable
// 2. CACHE_KEYDB_URL_FILE file content
// 3. Default value
func GetKeyDBURL(logger *zap.Logger) string {
	// Priority 1: Environment variable
	if keydbURL := strings.TrimSpace(os.Getenv("KEYDB_URL")); keydbURL != "" {
		logger.Info("Using KeyDB URL from environment variable", zap.String("url", keydbURL))
		return keydbURL
	}

	// Priority 2: Configurable connection file path
	connectionFile := os.Getenv("CACHE_KEYDB_URL_FILE")
	if connectionFile == "" {
		connectionFile = "/app/.keydb-url"
	}

	if content, err := os.ReadFile(connectionFile); err == nil {
		keydbURL := strings.TrimSpace(string(content))
		if len(keydbURL) > 0 {
			logger.Info("Using KeyDB URL from connection file", zap.String("file", connectionFile), zap.String("url", keydbURL))
			return keydbURL
		} else {
			logger.Warn("KeyDB connection file is empty", zap.String("file", connectionFile))
		}
	} else {
		logger.Debug("KeyDB connection file not found", zap.String("file", connectionFile), zap.Error(err))
	}

	// Priority 3: Default
	logger.Info("Using default KeyDB URL", zap.String("url", DefaultKeyDBURL))
	return DefaultKeyDBURL
}
