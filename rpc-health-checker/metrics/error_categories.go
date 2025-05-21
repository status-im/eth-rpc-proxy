package metrics

// ErrorCategory represents a categorized error type for RPC requests
type ErrorCategory string

const (
	// NoError indicates a successful request
	NoError ErrorCategory = "none"

	// NetworkError indicates network-related issues (timeouts, connection resets, etc.)
	NetworkError ErrorCategory = "network_error"

	// HTTPError indicates HTTP-level errors (non-200 status codes)
	HTTPError ErrorCategory = "http_error"

	// JSONRPCError indicates JSON-RPC protocol errors
	JSONRPCError ErrorCategory = "jsonrpc_error"

	// EVMError indicates EVM-specific errors (non-zero EVM error codes)
	EVMError ErrorCategory = "evm_error"

	// UnknownError indicates unclassified errors
	UnknownError ErrorCategory = "unknown_error"
)

// CategorizeError takes an error and returns the appropriate ErrorCategory
func CategorizeError(err error, httpStatus, evmErrorCode int) ErrorCategory {
	if err == nil {
		return NoError
	}

	errStr := err.Error()

	// Check for network-related errors
	if errStr == "context deadline exceeded" ||
		errStr == "context canceled" ||
		errStr == "connection reset by peer" ||
		errStr == "connection refused" ||
		errStr == "no such host" ||
		errStr == "i/o timeout" {
		return NetworkError
	}

	// Check for HTTP errors
	if httpStatus != 0 && httpStatus != 200 {
		return HTTPError
	}

	// Check for EVM errors
	if evmErrorCode != 0 {
		return EVMError
	}

	// Check for JSON-RPC errors
	if errStr == "invalid json" ||
		errStr == "invalid request" ||
		errStr == "method not found" ||
		errStr == "invalid params" {
		return JSONRPCError
	}

	// Default to unknown error
	return UnknownError
}
