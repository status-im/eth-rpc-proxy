package httpserver

import "go-proxy-cache/internal/models"

// CacheRequest represents a cache operation request
type CacheRequest struct {
	Chain    string `json:"chain"`
	Network  string `json:"network"`
	RawBody  string `json:"raw_body"`            // Raw JSON-RPC request body
	Data     string `json:"data,omitempty"`      // For SET operations - response data
	TTL      *int   `json:"ttl,omitempty"`       // TTL in seconds, optional for SET
	StaleTTL *int   `json:"stale_ttl,omitempty"` // Stale TTL in seconds, optional for SET
}

// CacheResponse represents a cache operation response
type CacheResponse struct {
	Success     bool               `json:"success"`
	Found       bool               `json:"found,omitempty"`
	Fresh       bool               `json:"fresh,omitempty"`
	Data        string             `json:"data,omitempty"`
	Key         string             `json:"key,omitempty"`
	CacheType   string             `json:"cache_type,omitempty"`
	TTL         int                `json:"ttl,omitempty"`
	Error       string             `json:"error,omitempty"`
	CacheStatus models.CacheStatus `json:"cache_status,omitempty"` // HIT, MISS, or BYPASS
	CacheLevel  models.CacheLevel  `json:"cache_level,omitempty"`  // L1, L2, or MISS
}
