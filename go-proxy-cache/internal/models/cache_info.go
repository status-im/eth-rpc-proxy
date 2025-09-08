package models

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// CacheType represents the type of caching for a method
type CacheType string

const (
	CacheTypePermanent CacheType = "permanent"
	CacheTypeShort     CacheType = "short"
	CacheTypeMinimal   CacheType = "minimal"
	CacheTypeNone      CacheType = "none"
)

// CacheStatus represents the cache status for a request
type CacheStatus string

const (
	CacheStatusHit    CacheStatus = "HIT"
	CacheStatusMiss   CacheStatus = "MISS"
	CacheStatusBypass CacheStatus = "BYPASS"
)

// String returns the string representation of the cache status
func (cs CacheStatus) String() string {
	return string(cs)
}

// IsValid checks if the cache status is one of the valid values
func (cs CacheStatus) IsValid() bool {
	switch cs {
	case CacheStatusHit, CacheStatusMiss, CacheStatusBypass:
		return true
	default:
		return false
	}
}

// UnmarshalYAML implements custom YAML unmarshaling for CacheType
func (c *CacheType) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err != nil {
		return err
	}

	switch str {
	case "permanent", "short", "minimal", "none":
		*c = CacheType(str)
		return nil
	default:
		return fmt.Errorf("invalid cache type '%s': must be one of 'permanent', 'short', 'minimal', 'none'", str)
	}
}

// CacheInfo contains cache configuration information
type CacheInfo struct {
	TTL       time.Duration `json:"ttl"`
	CacheType CacheType     `json:"cache_type"`
}

// TTL represents cache time-to-live configuration
type TTL struct {
	Fresh time.Duration // How long the data is considered fresh
	Stale time.Duration // How long stale data can be served (stale-if-error)
}

// CacheLevel represents the cache level where data was found
type CacheLevel string

const (
	CacheLevelL1   CacheLevel = "L1"
	CacheLevelL2   CacheLevel = "L2"
	CacheLevelMiss CacheLevel = "MISS"
)

// String returns the string representation of the cache level
func (cl CacheLevel) String() string {
	return string(cl)
}

// CacheLevelFromIndex creates a CacheLevel from a cache index
// Index 0 returns L1, index 1 returns L2, higher indices return L3, L4, etc.
// Negative indices are treated as L1 (fallback to first level)
func CacheLevelFromIndex(index int) CacheLevel {
	if index < 0 {
		return CacheLevelL1
	}

	switch index {
	case 0:
		return CacheLevelL1
	case 1:
		return CacheLevelL2
	default:
		return CacheLevel(fmt.Sprintf("L%d", index+1))
	}
}

// CacheResult represents the result of a cache operation with level information
type CacheResult struct {
	Entry *CacheEntry `json:"entry,omitempty"`
	Found bool        `json:"found"`
	Level CacheLevel  `json:"level"`
}

// CacheEntry represents an entry in the cache with TTL information
type CacheEntry struct {
	Data      []byte `json:"data"`
	ExpiresAt int64  `json:"expires_at"`
	StaleAt   int64  `json:"stale_at"`
	CreatedAt int64  `json:"created_at"`
}

// IsExpired checks if the cache entry is completely expired
func (ce *CacheEntry) IsExpired() bool {
	now := time.Now().Unix()
	return now > ce.ExpiresAt
}

// IsFresh checks if the cache entry is still fresh
func (ce *CacheEntry) IsFresh() bool {
	now := time.Now().Unix()
	return now <= ce.StaleAt
}

// RemainingTTL calculates the remaining TTL for this cache entry
func (ce *CacheEntry) RemainingTTL() TTL {
	now := time.Now().Unix()

	// Calculate remaining fresh time
	freshRemaining := ce.StaleAt - now
	if freshRemaining < 0 {
		freshRemaining = 0
	}

	// Calculate remaining stale time
	staleRemaining := ce.ExpiresAt - ce.StaleAt
	if staleRemaining < 0 {
		staleRemaining = 0
	}

	return TTL{
		Fresh: time.Duration(freshRemaining) * time.Second,
		Stale: time.Duration(staleRemaining) * time.Second,
	}
}
