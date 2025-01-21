package testutils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// MockRPCServer represents a mock RPC server for testing
type MockRPCServer struct {
	Port      int
	Handler   http.Handler
	server    *http.Server
	wg        sync.WaitGroup
	responses map[string]map[string]interface{} // method name -> response
	mu        sync.Mutex
}

// NewMockRPCServer creates a new mock RPC server
func NewMockRPCServer(port int) *MockRPCServer {
	mux := http.NewServeMux()
	server := &MockRPCServer{
		Port:      port,
		Handler:   mux,
		responses: make(map[string]map[string]interface{}),
	}

	// Setup default RPC endpoints
	mux.HandleFunc("/", server.handleRPCRequest)

	return server
}

// AddResponse adds a response for a specific method
func (s *MockRPCServer) AddResponse(method string, response map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses[method] = response
}

// ClearResponses clears all responses
func (s *MockRPCServer) ClearResponses() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses = make(map[string]map[string]interface{})
}

// Start starts the mock RPC server
func (s *MockRPCServer) Start() error {
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.Port),
		Handler: s.Handler,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.server.ListenAndServe()
	}()

	// Wait briefly to ensure server is up
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Stop stops the mock RPC server
func (s *MockRPCServer) Stop() error {
	if s.server != nil {
		err := s.server.Close()
		s.wg.Wait()
		return err
	}
	return nil
}

// handleRPCRequest handles incoming RPC requests
func (s *MockRPCServer) handleRPCRequest(w http.ResponseWriter, r *http.Request) {
	var request struct {
		JSONRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
		ID      interface{}   `json:"id"`
	}

	// Parse request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, -32700, "Parse error", nil)
		return
	}

	// Check for predefined response first
	s.mu.Lock()
	if response, exists := s.responses[request.Method]; exists {
		s.mu.Unlock()
		writeJSONResponse(w, response)
		return
	}
	s.mu.Unlock()

	// Handle different methods
	switch request.Method {
	case "eth_blockNumber":
		writeSuccess(w, request.ID, "0x123456")
	case "eth_getBalance":
		writeSuccess(w, request.ID, "0x1000000000000000000")
	default:
		writeError(w, -32601, "Method not found", nil)
	}
}

// writeJSONResponse writes a JSON response with the appropriate headers
func writeJSONResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeSuccess writes a successful JSON-RPC response
func writeSuccess(w http.ResponseWriter, id interface{}, result interface{}) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	writeJSONResponse(w, response)
}

// writeError writes a JSON-RPC error response
func writeError(w http.ResponseWriter, code int, message string, data interface{}) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"data":    data,
		},
	}
	writeJSONResponse(w, response)
}
