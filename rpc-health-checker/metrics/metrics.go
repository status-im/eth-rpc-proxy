package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (

	// cardinality: 1
	validationCycleDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "validation_cycle_duration_seconds",
		Help: "Duration of validation cycle in seconds",
	})

	providerStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "provider_status",
		Help: "Status of providers (1 = working, 0 = not working)",
	}, []string{"chain_id", "chain_name", "network_name", "provider_name", "provider_url", "auth_token_masked"})

	// approximate cardinality: 10 (chain_id) × 10 (provider_name) × 6 (error_type) × 20 (status_code) = 12k
	rpcRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rpc_requests_total",
		Help: "Total number of RPC requests made for validation checks",
	}, []string{
		"chain_id",      // Chain ID for identification
		"provider_name", // Provider name for identification
		"error_type",    // Categorized error type (none, network_error, http_error, jsonrpc_error, evm_error, unknown_error)
		"status_code",   // Combined status code: HTTP status or EVM error code
	})
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

// RPCRequestMetrics contains all the parameters needed for recording RPC request metrics
type RPCRequestMetrics struct {
	ChainID      int64
	ChainName    string
	ProviderName string
	ProviderURL  string
	Method       string
	AuthToken    string
	RequestErr   error
	HTTPStatus   int
	EVMErrorCode int
}

// RecordRPCRequest records a single RPC request with its metadata and error information
func RecordRPCRequest(metrics RPCRequestMetrics) {
	// Categorize the error
	errCategory := CategorizeError(metrics.RequestErr, metrics.HTTPStatus, metrics.EVMErrorCode)

	// Determine status code based on error type
	statusCode := "0"
	if errCategory == HTTPError {
		statusCode = fmt.Sprintf("http_%d", metrics.HTTPStatus)
	} else if errCategory == EVMError {
		statusCode = fmt.Sprintf("evm_%d", metrics.EVMErrorCode)
	}

	rpcRequestsTotal.WithLabelValues(
		fmt.Sprintf("%d", metrics.ChainID),
		metrics.ProviderName,
		string(errCategory),
		statusCode,
	).Inc()
}

// maskAuthToken masks the auth token for security
func maskAuthToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
