package metrics

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		httpStatus   int
		evmErrorCode int
		want         ErrorCategory
	}{
		{
			name:         "nil error",
			err:          nil,
			httpStatus:   0,
			evmErrorCode: 0,
			want:         NoError,
		},
		{
			name:         "network timeout",
			err:          &timeoutError{},
			httpStatus:   0,
			evmErrorCode: 0,
			want:         NetworkError,
		},
		{
			name:         "connection refused",
			err:          errors.New("connection refused"),
			httpStatus:   0,
			evmErrorCode: 0,
			want:         NetworkError,
		},
		{
			name:         "connection reset",
			err:          errors.New("connection reset"),
			httpStatus:   0,
			evmErrorCode: 0,
			want:         NetworkError,
		},
		{
			name:         "context deadline exceeded",
			err:          context.DeadlineExceeded,
			httpStatus:   0,
			evmErrorCode: 0,
			want:         NetworkError,
		},
		{
			name:         "context canceled",
			err:          context.Canceled,
			httpStatus:   0,
			evmErrorCode: 0,
			want:         NetworkError,
		},
		{
			name:         "http error 400",
			err:          nil,
			httpStatus:   http.StatusBadRequest,
			evmErrorCode: 0,
			want:         HTTPError,
		},
		{
			name:         "http error 500",
			err:          nil,
			httpStatus:   http.StatusInternalServerError,
			evmErrorCode: 0,
			want:         HTTPError,
		},
		{
			name:         "jsonrpc invalid request",
			err:          nil,
			httpStatus:   0,
			evmErrorCode: JSONRPCInvalidRequest,
			want:         JSONRPCError,
		},
		{
			name:         "jsonrpc method not found",
			err:          nil,
			httpStatus:   0,
			evmErrorCode: JSONRPCMethodNotFound,
			want:         JSONRPCError,
		},
		{
			name:         "jsonrpc invalid params",
			err:          nil,
			httpStatus:   0,
			evmErrorCode: JSONRPCInvalidParams,
			want:         JSONRPCError,
		},
		{
			name:         "jsonrpc parse error",
			err:          nil,
			httpStatus:   0,
			evmErrorCode: JSONRPCParseError,
			want:         JSONRPCError,
		},
		{
			name:         "evm error",
			err:          nil,
			httpStatus:   0,
			evmErrorCode: -32000,
			want:         EVMError,
		},
		{
			name:         "unknown error",
			err:          errors.New("some unknown error"),
			httpStatus:   0,
			evmErrorCode: 0,
			want:         UnknownError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CategorizeError(tt.err, tt.httpStatus, tt.evmErrorCode)
			if got != tt.want {
				t.Errorf("CategorizeError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// timeoutError implements net.Error interface
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
