package requestsrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/status-im/eth-rpc-proxy/metrics"

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
	var httpStatus int
	var requestErr error
	var evmErrorCode int

	// Defer the metrics recording
	defer func() {
		metrics.RecordRPCRequest(metrics.RPCRequestMetrics{
			ChainID:      provider.ChainID,
			ChainName:    provider.ChainName,
			ProviderName: provider.Name,
			ProviderURL:  provider.URL,
			Method:       method,
			AuthToken:    provider.AuthToken,
			RequestErr:   requestErr,
			HTTPStatus:   httpStatus,
			EVMErrorCode: evmErrorCode,
		})
	}()

	// Create JSON-RPC 2.0 request body
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		requestErr = fmt.Errorf("failed to marshal request body: %w", err)
		return ProviderResult{
			Success:     false,
			Error:       requestErr,
			ElapsedTime: time.Since(startTime),
		}
	}

	// Create HTTP client with timeout from context
	client := &http.Client{}
	req, err := http.NewRequest("POST", provider.URL, bytes.NewBuffer(jsonBody))
	if err != nil {
		requestErr = fmt.Errorf("failed to create request: %w", err)
		return ProviderResult{
			Success:     false,
			Error:       requestErr,
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
		req.URL.Path = strings.TrimRight(req.URL.Path, "/") + fmt.Sprintf("/%s", provider.AuthToken)
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		requestErr = fmt.Errorf("request failed: %w", err)
		return ProviderResult{
			Success:     false,
			Error:       requestErr,
			ElapsedTime: time.Since(startTime),
		}
	}
	defer resp.Body.Close()

	httpStatus = resp.StatusCode

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		requestErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return ProviderResult{
			Success:     false,
			Error:       requestErr,
			ElapsedTime: time.Since(startTime),
		}
	}

	// Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		requestErr = fmt.Errorf("failed to read response: %w", err)
		return ProviderResult{
			Success:     false,
			Error:       requestErr,
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
		requestErr = fmt.Errorf("failed to parse JSON response: %w", err)
		return ProviderResult{
			Success:     false,
			Error:       requestErr,
			ElapsedTime: time.Since(startTime),
		}
	}

	// Check for JSON-RPC error
	if jsonResponse.Error.Code != 0 {
		evmErrorCode = jsonResponse.Error.Code
		requestErr = fmt.Errorf("JSON-RPC error: %s (code %d)", jsonResponse.Error.Message, jsonResponse.Error.Code)
		return ProviderResult{
			Success:     false,
			Error:       requestErr,
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

var _ MethodCaller = (*RequestsRunner)(nil)
