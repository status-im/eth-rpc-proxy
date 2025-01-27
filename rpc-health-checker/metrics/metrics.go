package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/status-im/eth-rpc-proxy/config"
)

var (
	validationCycleDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "validation_cycle_duration_seconds",
		Help: "Duration of validation cycle in seconds",
	})

	workingProviders = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "working_providers_total",
		Help: "Number of working providers per chain",
	}, []string{"chain_name", "network"})

	nonWorkingProviders = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "non_working_providers_total",
		Help: "Number of non-working providers per chain",
	}, []string{"chain_name", "network"})

	rpcRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rpc_requests_total",
		Help: "Total number of RPC requests made for validation checks",
	}, []string{"chain_id", "provider_name", "provider_url", "method", "auth_token_masked"})
)

// RecordValidationCycleDuration records the duration of a complete validation cycle
func RecordValidationCycleDuration(duration time.Duration) {
	validationCycleDuration.Observe(duration.Seconds())
}

// RecordWorkingProviders records the number of working providers for each chain
func RecordWorkingProviders(validChains []config.ChainConfig) {
	for _, chain := range validChains {
		workingProviders.With(prometheus.Labels{
			"chain_name": chain.Name,
			"network":    chain.Network,
		}).Set(float64(len(chain.Providers)))
	}
}

// RecordNonWorkingProviders records the status of non-working providers
func RecordNonWorkingProviders(chainName, networkName string, providerResults map[string]struct {
	Valid bool
	URL   string
}) {
	for providerName, result := range providerResults {
		value := 0.0
		if !result.Valid {
			value = 1.0
		}
		nonWorkingProviders.With(prometheus.Labels{
			"chain_name": providerName,
			"network":    networkName,
		}).Set(value)
	}
}

// RecordRPCRequest records a single RPC request with its metadata
func RecordRPCRequest(chainId int64, providerName, providerURL, method, authToken string) {
	// Mask the auth token by keeping only first and last 4 characters if it's long enough
	maskedToken := maskAuthToken(authToken)
	rpcRequestsTotal.WithLabelValues(
		fmt.Sprintf("%d", chainId),
		providerName,
		providerURL,
		method,
		maskedToken,
	).Inc()
}

// maskAuthToken masks the auth token for security
func maskAuthToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
