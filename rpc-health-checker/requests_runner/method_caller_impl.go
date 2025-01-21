package requestsrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	rpcprovider "github.com/status-im/eth-rpc-proxy/provider"
)

// RequestsRunner implements MethodCaller interface
type RequestsRunner struct{}

// NewRequestsRunner creates a new instance of RequestsRunner
func NewRequestsRunner() *RequestsRunner {
	return &RequestsRunner{}
}

// CallEVMMethod makes an HTTP POST request to an RPC provider for a specific EVM method
// Implements the MethodCaller interface
func (r *RequestsRunner) CallMethod(
	ctx context.Context,
	provider rpcprovider.RPCProvider,
	method string,
	params []interface{},
	timeout time.Duration,
) ProviderResult {
	startTime := time.Now()

	// Create JSON-RPC 2.0 request body
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return ProviderResult{
			Success:     false,
			Error:       fmt.Errorf("failed to marshal request body: %w", err),
			ElapsedTime: time.Since(startTime),
		}
	}

	// Create HTTP client with timeout from context
	client := &http.Client{}
	req, err := http.NewRequest("POST", provider.URL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return ProviderResult{
			Success:     false,
			Error:       fmt.Errorf("failed to create request: %w", err),
			ElapsedTime: time.Since(startTime),
		}
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Set authentication based on provider type
	switch provider.AuthType {
	case rpcprovider.BasicAuth:
		req.SetBasicAuth(provider.AuthLogin, provider.AuthPassword)
	case rpcprovider.TokenAuth:
		req.URL.Path += fmt.Sprintf("/%s", provider.AuthToken)
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return ProviderResult{
			Success:     false,
			Error:       fmt.Errorf("request failed: %w", err),
			ElapsedTime: time.Since(startTime),
		}
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		return ProviderResult{
			Success:     false,
			Error:       fmt.Errorf("unexpected status code: %d", resp.StatusCode),
			ElapsedTime: time.Since(startTime),
		}
	}

	// Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ProviderResult{
			Success:     false,
			Error:       fmt.Errorf("failed to read response: %w", err),
			ElapsedTime: time.Since(startTime),
		}
	}

	// Parse JSON-RPC response
	var jsonResponse struct {
		Result interface{} `json:"result"`
		Error  struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return ProviderResult{
			Success:     false,
			Error:       fmt.Errorf("failed to parse JSON response: %w", err),
			ElapsedTime: time.Since(startTime),
		}
	}

	// Check for JSON-RPC error
	if jsonResponse.Error.Code != 0 {
		return ProviderResult{
			Success:     false,
			Error:       fmt.Errorf("JSON-RPC error: %s (code %d)", jsonResponse.Error.Message, jsonResponse.Error.Code),
			Response:    body,
			ElapsedTime: time.Since(startTime),
		}
	}

	// Convert result to string
	resultStr := fmt.Sprintf("%v", jsonResponse.Result)

	return ProviderResult{
		Success:     true,
		Error:       nil,
		Result:      resultStr,
		Response:    body,
		ElapsedTime: time.Since(startTime),
	}
}
