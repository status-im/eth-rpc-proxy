package interfaces

import (
	"github.com/status-im/proxy-common/models"

	localModels "go-proxy-cache/internal/models"
)

//go:generate mockgen -package=mock -source=cache_rules_classifier.go -destination=mock/cache_rules_classifier.go

// CacheRulesClassifier defines the interface for classifying requests and getting cache info
type CacheRulesClassifier interface {
	// GetTtl analyzes the request and returns cache information including TTL and cache type
	GetTtl(chain, network string, request *localModels.JSONRPCRequest) models.CacheInfo
	// ShouldSkipNullCache returns true if null results should not be cached for this method
	ShouldSkipNullCache(method string) bool
}
