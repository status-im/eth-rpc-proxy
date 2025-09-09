package cache

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"

	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/models"
)

// Ensure KeyBuilderImpl implements interfaces.KeyBuilder
var _ interfaces.KeyBuilder = (*KeyBuilderImpl)(nil)

// KeyBuilderImpl implements the KeyBuilder interface
type KeyBuilderImpl struct{}

// NewKeyBuilder creates a new KeyBuilder instance
func NewKeyBuilder() interfaces.KeyBuilder {
	return &KeyBuilderImpl{}
}

// Build creates a cache key for a single JSON-RPC request
func (kb *KeyBuilderImpl) Build(chain string, network string, req *models.JSONRPCRequest) (string, error) {
	if req == nil {
		return "", errors.New("request cannot be nil")
	}

	if req.Method == "" {
		return "", errors.New("request method cannot be empty")
	}

	if chain == "" {
		return "", errors.New("chain cannot be empty")
	}

	if network == "" {
		return "", errors.New("network cannot be empty")
	}

	// Generate hash for params if they exist
	var paramsHashStr string
	if req.Params != nil {
		paramsJSON, err := json.Marshal(req.Params)
		if err != nil {
			return "", fmt.Errorf("failed to marshal params: %w", err)
		}
		// Create MD5 hash of params (similar to Lua version)
		hasher := md5.New()
		hasher.Write(paramsJSON)
		paramsHashStr = fmt.Sprintf("%x", hasher.Sum(nil))
	}

	// Use JSONRPC version, default to "2.0" if not specified
	jsonrpc := req.Jsonrpc
	if jsonrpc == "" {
		jsonrpc = "2.0"
	}

	// Create normalized hash part (similar to Lua normalized_hash function)
	hashPart := fmt.Sprintf("%s:%s:%s", req.Method, jsonrpc, paramsHashStr)

	// Create final cache key: chain:network:hashPart
	key := fmt.Sprintf("%s:%s:%s", chain, network, hashPart)

	return key, nil
}

// BuildBatch creates cache keys for multiple JSON-RPC requests
func (kb *KeyBuilderImpl) BuildBatch(chain, network string, reqs []models.JSONRPCRequest) ([]string, error) {
	if len(reqs) == 0 {
		return nil, errors.New("requests slice cannot be empty")
	}

	keys := make([]string, len(reqs))

	for i, req := range reqs {
		key, err := kb.Build(chain, network, &req)
		if err != nil {
			return nil, fmt.Errorf("failed to build key for request %d: %w", i, err)
		}
		keys[i] = key
	}

	return keys, nil
}
