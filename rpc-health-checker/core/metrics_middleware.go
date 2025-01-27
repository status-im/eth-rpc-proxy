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

	// Record provider statuses for all chains
	for chainId, chainResults := range results {
		if chainConfig, exists := chainConfigs[chainId]; exists {
			providerStatus := make(map[string]struct {
				Valid     bool
				URL       string
				AuthToken string
			})

			// Create a map of provider names to their URLs and auth tokens
			providerInfo := make(map[string]struct {
				URL       string
				AuthToken string
			})
			for _, provider := range chainConfig.Providers {
				providerInfo[provider.Name] = struct {
					URL       string
					AuthToken string
				}{
					URL:       provider.URL,
					AuthToken: provider.AuthToken,
				}
			}

			for providerName, result := range chainResults {
				info := providerInfo[providerName]
				providerStatus[providerName] = struct {
					Valid     bool
					URL       string
					AuthToken string
				}{
					Valid:     result.Valid,
					URL:       info.URL,
					AuthToken: info.AuthToken,
				}
			}
			metrics.RecordProviderStatuses(chainId, chainConfig.Name, chainConfig.Network, providerStatus)
		}
	}
}
