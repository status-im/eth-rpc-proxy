package models

import "time"

// CacheInfo contains cache configuration information
type CacheInfo struct {
	TTL       time.Duration `json:"ttl"`
	CacheType string        `json:"cache_type"`
}

// TTL represents cache time-to-live configuration
type TTL struct {
	Fresh time.Duration // How long the data is considered fresh
	Stale time.Duration // How long stale data can be served (stale-if-error)
}
