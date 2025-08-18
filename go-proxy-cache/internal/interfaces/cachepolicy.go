package interfaces

import (
	"time"
)

//go:generate mockgen -package=mock -source=cachepolicy.go -destination=mock/cachepolicy.go

// TTL represents cache time-to-live configuration
type TTL struct {
	Fresh time.Duration // How long the data is considered fresh
	Stale time.Duration // How long stale data can be served (stale-if-error)
}

// CachePolicy determines cache behavior based on request characteristics
type CachePolicy interface {
	Resolve(method string, params interface{}) TTL
}
