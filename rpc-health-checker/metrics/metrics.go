package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	validationCycleDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "validation_cycle_duration_seconds",
		Help: "Duration of validation cycle in seconds",
	})

	providerStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "provider_status",
		Help: "Status of providers (1 = working, 0 = not working)",
	}, []string{"chain_id", "chain_name", "network_name", "provider_name", "provider_url", "auth_token_masked"})

	rpcRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rpc_requests_total",
		Help: "Total number of RPC requests made for validation checks",
	}, []string{"chain_id", "provider_name", "provider_url", "method", "auth_token_masked"})
)

// RecordValidationCycleDuration records the duration of a complete validation cycle
func RecordValidationCycleDuration(duration time.Duration) {
	validationCycleDuration.Observe(duration.Seconds())
}

// RecordProviderStatuses records the status of all providers for each chain
func RecordProviderStatuses(chainId int64, chainName, networkName string, providerResults map[string]struct {
	Valid     bool
	URL       string
	AuthToken string
}) {
	for providerName, result := range providerResults {
		value := 0.0
		if result.Valid {
			value = 1.0
		}
		maskedToken := maskAuthToken(result.AuthToken)
		providerStatus.With(prometheus.Labels{
			"chain_id":          fmt.Sprintf("%d", chainId),
			"chain_name":        chainName,
			"network_name":      networkName,
			"provider_name":     providerName,
			"provider_url":      result.URL,
			"auth_token_masked": maskedToken,
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
