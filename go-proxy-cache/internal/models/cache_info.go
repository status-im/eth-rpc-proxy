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
