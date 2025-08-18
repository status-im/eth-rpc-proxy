package utils

import (
	"encoding/json"
	"fmt"

	"go-proxy-cache/internal/models"
)

// ParseJSONRPCRequest parses raw JSON-RPC request body
func ParseJSONRPCRequest(rawBody string) (*models.JSONRPCRequest, error) {
	if rawBody == "" {
		return nil, fmt.Errorf("empty request body")
	}

	var request models.JSONRPCRequest
	if err := json.Unmarshal([]byte(rawBody), &request); err != nil {
		return nil, fmt.Errorf("failed to parse JSON-RPC request: %w", err)
	}

	if request.Method == "" {
		return nil, fmt.Errorf("missing method in JSON-RPC request")
	}

	return &request, nil
}

// FixResponseID fixes the ID in cached response to match the current request
func FixResponseID(cachedResponse string, requestID interface{}) string {
	if cachedResponse == "" || requestID == nil {
		return cachedResponse
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal([]byte(cachedResponse), &responseData); err != nil {
		return cachedResponse
	}

	// If ID already matches, return original response
	if responseData["id"] == requestID {
		return cachedResponse
	}

	// Update the ID to match the current request
	responseData["id"] = requestID

	if fixedBytes, err := json.Marshal(responseData); err == nil {
		return string(fixedBytes)
	}

	return cachedResponse
}
