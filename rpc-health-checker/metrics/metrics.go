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
	}, []string{
		"chain_id",
		"chain_name",
		"provider_name",
		"provider_url",
		"method",
		"auth_token_masked",
		"request_err",    // Error message if request failed, "none" if successful
		"http_status",    // HTTP status code, "0" if request failed before getting response
		"evm_error_code", // EVM error code from JSON-RPC response, "0" if successful
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
	// Mask the auth token by keeping only first and last 4 characters if it's long enough
	maskedToken := maskAuthToken(metrics.AuthToken)

	// Format error message, use "none" if no error
	errMsg := "none"
	if metrics.RequestErr != nil {
		errMsg = metrics.RequestErr.Error()
	}

	rpcRequestsTotal.WithLabelValues(
		fmt.Sprintf("%d", metrics.ChainID),
		metrics.ChainName,
		metrics.ProviderName,
		metrics.ProviderURL,
		metrics.Method,
		maskedToken,
		errMsg,
		fmt.Sprintf("%d", metrics.HTTPStatus),
		fmt.Sprintf("%d", metrics.EVMErrorCode),
	).Inc()
}

// maskAuthToken masks the auth token for security
func maskAuthToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
