package metrics

import (
	"errors"
	"net"
	"net/http"
	"strings"
)

// ErrorCategory represents the type of error that occurred
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

// JSON-RPC error codes
const (
	JSONRPCInvalidRequest = -32600
	JSONRPCMethodNotFound = -32601
	JSONRPCInvalidParams  = -32602
	JSONRPCParseError     = -32700
)

// CategorizeError determines the category of an error based on its type and content
func CategorizeError(err error, httpStatus int, evmErrorCode int) ErrorCategory {
	// Check for HTTP errors first
	if httpStatus != 0 && httpStatus != http.StatusOK {
		return HTTPError
	}

	// Check for JSON-RPC errors
	if evmErrorCode == JSONRPCInvalidRequest ||
		evmErrorCode == JSONRPCMethodNotFound ||
		evmErrorCode == JSONRPCInvalidParams ||
		evmErrorCode == JSONRPCParseError {
		return JSONRPCError
	}

	// Check for other EVM errors
	if evmErrorCode != 0 {
		return EVMError
	}

	// If there's no error and no error codes, return NoError
	if err == nil {
		return NoError
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return NetworkError
		}
	}

	// Check for common network errors from standard library
	errMsg := err.Error()
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "context canceled") ||
		strings.Contains(errMsg, "context deadline exceeded") {
		return NetworkError
	}

	// If we can't categorize the error, return unknown
	return UnknownError
}
