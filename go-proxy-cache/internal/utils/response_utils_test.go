package utils

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestIsNullResult(t *testing.T) {
	tests := []struct {
		name         string
		responseData string
		expected     bool
		description  string
	}{
		{
			name:         "null result",
			responseData: `{"jsonrpc":"2.0","id":1,"result":null}`,
			expected:     true,
			description:  "should return true when result is null",
		},
		{
			name:         "non-null result - object",
			responseData: `{"jsonrpc":"2.0","id":1,"result":{"blockNumber":"0x123"}}`,
			expected:     false,
			description:  "should return false when result is an object",
		},
		{
			name:         "non-null result - string",
			responseData: `{"jsonrpc":"2.0","id":1,"result":"0x123"}`,
			expected:     false,
			description:  "should return false when result is a string",
		},
		{
			name:         "non-null result - number",
			responseData: `{"jsonrpc":"2.0","id":1,"result":123}`,
			expected:     false,
			description:  "should return false when result is a number",
		},
		{
			name:         "non-null result - array",
			responseData: `{"jsonrpc":"2.0","id":1,"result":[]}`,
			expected:     false,
			description:  "should return false when result is an array",
		},
		{
			name:         "non-null result - boolean",
			responseData: `{"jsonrpc":"2.0","id":1,"result":true}`,
			expected:     false,
			description:  "should return false when result is a boolean",
		},
		{
			name:         "error response",
			responseData: `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid Request"}}`,
			expected:     false,
			description:  "should return false for error response without result field",
		},
		{
			name:         "empty response",
			responseData: "",
			expected:     false,
			description:  "should return false for empty response",
		},
		{
			name:         "invalid JSON",
			responseData: "not valid json",
			expected:     false,
			description:  "should return false for invalid JSON",
		},
		{
			name:         "missing result field",
			responseData: `{"jsonrpc":"2.0","id":1}`,
			expected:     false,
			description:  "should return false when result field is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNullResult(tt.responseData)
			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, result)
			}
		})
	}
}

func TestParseJSONRPCRequest(t *testing.T) {
	tests := []struct {
		name        string
		rawBody     string
		expectError bool
		method      string
	}{
		{
			name:        "valid request",
			rawBody:     `{"method":"eth_getBalance","params":["0x123",true],"jsonrpc":"2.0","id":1}`,
			expectError: false,
			method:      "eth_getBalance",
		},
		{
			name:        "empty body",
			rawBody:     "",
			expectError: true,
		},
		{
			name:        "invalid JSON",
			rawBody:     "not valid json",
			expectError: true,
		},
		{
			name:        "missing method",
			rawBody:     `{"params":["0x123"],"jsonrpc":"2.0","id":1}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := ParseJSONRPCRequest(tt.rawBody)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if request.Method != tt.method {
					t.Errorf("Expected method %s, got %s", tt.method, request.Method)
				}
			}
		})
	}
}

func TestFixResponseID(t *testing.T) {
	tests := []struct {
		name           string
		cachedResponse string
		requestID      interface{}
		expected       string
		isValidJSON    bool // whether we expect valid JSON output
	}{
		{
			name:           "fix integer ID",
			cachedResponse: `{"jsonrpc":"2.0","id":1,"result":"0x123"}`,
			requestID:      float64(2), // JSON numbers are decoded as float64
			expected:       `{"id":2,"jsonrpc":"2.0","result":"0x123"}`,
			isValidJSON:    true,
		},
		{
			name:           "same ID - no change needed",
			cachedResponse: `{"jsonrpc":"2.0","id":1,"result":"0x123"}`,
			requestID:      float64(1),
			expected:       `{"jsonrpc":"2.0","id":1,"result":"0x123"}`,
			isValidJSON:    true,
		},
		{
			name:           "empty response",
			cachedResponse: "",
			requestID:      float64(1),
			expected:       "",
			isValidJSON:    false,
		},
		{
			name:           "nil request ID",
			cachedResponse: `{"jsonrpc":"2.0","id":1,"result":"0x123"}`,
			requestID:      nil,
			expected:       `{"jsonrpc":"2.0","id":1,"result":"0x123"}`,
			isValidJSON:    true,
		},
		{
			name:           "invalid JSON",
			cachedResponse: "not valid json",
			requestID:      float64(1),
			expected:       "not valid json",
			isValidJSON:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FixResponseID(tt.cachedResponse, tt.requestID)

			if tt.isValidJSON {
				// Compare unmarshaled JSON to be insensitive to key ordering
				var expectedMap, resultMap map[string]interface{}
				if err := json.Unmarshal([]byte(tt.expected), &expectedMap); err != nil {
					t.Fatalf("Failed to unmarshal expected JSON: %v", err)
				}
				if err := json.Unmarshal([]byte(result), &resultMap); err != nil {
					t.Fatalf("Failed to unmarshal result JSON: %v", err)
				}
				if !reflect.DeepEqual(expectedMap, resultMap) {
					t.Errorf("Expected %v, got %v", expectedMap, resultMap)
				}
			} else {
				// For non-JSON cases, compare strings directly
				if result != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func BenchmarkIsNullResult(b *testing.B) {
	responseData := `{"jsonrpc":"2.0","id":1,"result":null}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsNullResult(responseData)
	}
}
