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

	// approximate cardinality: 10 (chain_id) × 10 (provider_name) = 100
	providerStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "provider_status",
		Help: "Status of providers (1 = working, 0 = not working)",
	}, []string{
		"chain_id",      // Chain ID for identification
		"provider_name", // Provider name for identification
	})

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
		providerStatus.WithLabelValues(
			fmt.Sprintf("%d", chainId),
			providerName,
		).Set(value)
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
	var statusCode string
	switch errCategory {
	case HTTPError:
		statusCode = fmt.Sprintf("http_%d", metrics.HTTPStatus)
	case EVMError:
		statusCode = fmt.Sprintf("evm_%d", metrics.EVMErrorCode)
	case JSONRPCError:
		statusCode = "jsonrpc_error"
	case NetworkError:
		statusCode = "network_error"
	case UnknownError:
		statusCode = "unknown_error"
	default:
		statusCode = "success"
	}

	rpcRequestsTotal.WithLabelValues(
		fmt.Sprintf("%d", metrics.ChainID),
		metrics.ProviderName,
		string(errCategory),
		statusCode,
	).Inc()
}
