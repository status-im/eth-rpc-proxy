package core

import (
	"time"

	"github.com/status-im/eth-rpc-proxy/config"
	"github.com/status-im/eth-rpc-proxy/metrics"
)

// RecordValidationMetrics records validation cycle metrics
func RecordValidationMetrics(
	startTime time.Time,
	validChains []config.ChainConfig,
	results map[int64]map[string]ProviderValidationResult,
	chainConfigs map[int64]config.ChainConfig,
) {
	duration := time.Since(startTime)
	metrics.RecordValidationCycleDuration(duration)
	metrics.RecordWorkingProviders(validChains)

	// Record non-working providers metrics
	for chainId, chainResults := range results {
		if chainConfig, exists := chainConfigs[chainId]; exists {
			providerStatus := make(map[string]struct {
				Valid bool
				URL   string
			})

			// Create a map of provider names to their URLs
			providerUrls := make(map[string]string)
			for _, provider := range chainConfig.Providers {
				providerUrls[provider.Name] = provider.URL
			}

			for providerName, result := range chainResults {
				providerStatus[providerName] = struct {
					Valid bool
					URL   string
				}{
					Valid: result.Valid,
					URL:   providerUrls[providerName],
				}
			}
			metrics.RecordNonWorkingProviders(chainConfig.Name, chainConfig.Network, providerStatus)
		}
	}
}
