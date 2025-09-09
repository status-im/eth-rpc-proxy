package cache

import (
	"testing"

	"go-proxy-cache/internal/models"
)

func TestKeyBuilder_Build(t *testing.T) {
	kb := NewKeyBuilder()

	tests := []struct {
		name      string
		chain     string
		network   string
		request   *models.JSONRPCRequest
		wantKey   string
		wantError bool
	}{
		{
			name:    "basic request",
			chain:   "ethereum",
			network: "mainnet",
			request: &models.JSONRPCRequest{
				ID:      1,
				Method:  "eth_blockNumber",
				Params:  []interface{}{},
				Jsonrpc: "2.0",
			},
			wantKey:   "ethereum:mainnet:eth_blockNumber:2.0:d751713988987e9331980363e24189ce",
			wantError: false,
		},
		{
			name:    "request with params",
			chain:   "polygon",
			network: "mainnet",
			request: &models.JSONRPCRequest{
				ID:     2,
				Method: "eth_getBlockByNumber",
				Params: []interface{}{"0x1", true},
			},
			wantError: false,
		},
		{
			name:      "nil request",
			chain:     "ethereum",
			network:   "mainnet",
			request:   nil,
			wantError: true,
		},
		{
			name:    "empty method",
			chain:   "ethereum",
			network: "mainnet",
			request: &models.JSONRPCRequest{
				ID:     1,
				Method: "",
			},
			wantError: true,
		},
		{
			name:    "empty chain",
			chain:   "",
			network: "mainnet",
			request: &models.JSONRPCRequest{
				ID:     1,
				Method: "eth_blockNumber",
			},
			wantError: true,
		},
		{
			name:    "empty network",
			chain:   "ethereum",
			network: "",
			request: &models.JSONRPCRequest{
				ID:     1,
				Method: "eth_blockNumber",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, gotErr := kb.Build(tt.chain, tt.network, tt.request)

			if tt.wantError {
				if gotErr == nil {
					t.Errorf("Build() expected error, but got none")
				}
				if gotKey != "" {
					t.Errorf("Build() gotKey = %v, want empty string when error expected", gotKey)
				}
				return
			}

			if gotErr != nil {
				t.Errorf("Build() unexpected error: %v", gotErr)
				return
			}

			// For successful cases, check that key starts with expected prefix
			if tt.wantKey != "" {
				if gotKey != tt.wantKey {
					t.Errorf("Build() gotKey = %v, want %v", gotKey, tt.wantKey)
				}
			} else {
				// Check that key starts with expected prefix for cases without exact match
				expectedPrefix := tt.chain + ":" + tt.network + ":" + tt.request.Method + ":"
				if len(gotKey) < len(expectedPrefix) || gotKey[:len(expectedPrefix)] != expectedPrefix {
					t.Errorf("Build() gotKey = %v, want to start with %v", gotKey, expectedPrefix)
				}
			}
		})
	}
}

func TestKeyBuilder_BuildBatch(t *testing.T) {
	kb := NewKeyBuilder()

	t.Run("successful batch", func(t *testing.T) {
		requests := []models.JSONRPCRequest{
			{
				ID:      1,
				Method:  "eth_blockNumber",
				Params:  []interface{}{},
				Jsonrpc: "2.0",
			},
			{
				ID:     2,
				Method: "eth_getBlockByNumber",
				Params: []interface{}{"0x1", true},
			},
		}

		keys, err := kb.BuildBatch("ethereum", "mainnet", requests)

		if err != nil {
			t.Errorf("BuildBatch() unexpected error: %v", err)
			return
		}

		if len(keys) != len(requests) {
			t.Errorf("BuildBatch() got %d keys, want %d", len(keys), len(requests))
		}

		// Check that each key is properly formed
		for i, key := range keys {
			expectedPrefix := "ethereum:mainnet:" + requests[i].Method + ":"
			if len(key) < len(expectedPrefix) || key[:len(expectedPrefix)] != expectedPrefix {
				t.Errorf("BuildBatch() key[%d] = %v, want to start with %v", i, key, expectedPrefix)
			}
		}
	})

	t.Run("empty batch", func(t *testing.T) {
		requests := []models.JSONRPCRequest{}

		keys, err := kb.BuildBatch("ethereum", "mainnet", requests)

		if err == nil {
			t.Errorf("BuildBatch() expected error for empty batch, but got none")
		}

		if keys != nil {
			t.Errorf("BuildBatch() expected nil keys for empty batch, got %v", keys)
		}
	})

	t.Run("batch with invalid request", func(t *testing.T) {
		requests := []models.JSONRPCRequest{
			{
				ID:      1,
				Method:  "eth_blockNumber",
				Params:  []interface{}{},
				Jsonrpc: "2.0",
			},
			{
				ID:     2,
				Method: "", // Invalid empty method
			},
		}

		keys, err := kb.BuildBatch("ethereum", "mainnet", requests)

		if err == nil {
			t.Errorf("BuildBatch() expected error for batch with invalid request, but got none")
		}

		if keys != nil {
			t.Errorf("BuildBatch() expected nil keys for invalid batch, got %v", keys)
		}
	})
}

func TestKeyBuilder_ConsistentKeys(t *testing.T) {
	kb := NewKeyBuilder()

	// Same request should produce same key regardless of ID
	req1 := &models.JSONRPCRequest{
		ID:      1,
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
		Jsonrpc: "2.0",
	}

	req2 := &models.JSONRPCRequest{
		ID:      999,
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
		Jsonrpc: "2.0",
	}

	key1, err1 := kb.Build("ethereum", "mainnet", req1)
	if err1 != nil {
		t.Errorf("Build() unexpected error for req1: %v", err1)
		return
	}

	key2, err2 := kb.Build("ethereum", "mainnet", req2)
	if err2 != nil {
		t.Errorf("Build() unexpected error for req2: %v", err2)
		return
	}

	if key1 != key2 {
		t.Errorf("Build() should produce same key for same request with different IDs, got %v and %v", key1, key2)
	}
}

func TestKeyBuilder_DifferentParams(t *testing.T) {
	kb := NewKeyBuilder()

	req1 := &models.JSONRPCRequest{
		ID:     1,
		Method: "eth_getBlockByNumber",
		Params: []interface{}{"0x1", true},
	}

	req2 := &models.JSONRPCRequest{
		ID:     1,
		Method: "eth_getBlockByNumber",
		Params: []interface{}{"0x2", true},
	}

	key1, err1 := kb.Build("ethereum", "mainnet", req1)
	if err1 != nil {
		t.Errorf("Build() unexpected error for req1: %v", err1)
		return
	}

	key2, err2 := kb.Build("ethereum", "mainnet", req2)
	if err2 != nil {
		t.Errorf("Build() unexpected error for req2: %v", err2)
		return
	}

	if key1 == key2 {
		t.Errorf("Build() should produce different keys for different params, got same key: %v", key1)
	}
}
