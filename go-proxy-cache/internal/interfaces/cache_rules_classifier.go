package interfaces

import "go-proxy-cache/internal/models"

//go:generate mockgen -package=mock -source=cache_rules_classifier.go -destination=mock/cache_rules_classifier.go

// CacheRulesClassifier defines the interface for classifying requests and getting cache info
type CacheRulesClassifier interface {
	// GetTtl analyzes the request and returns cache information including TTL and cache type
	GetTtl(chain, network string, request *models.JSONRPCRequest) models.CacheInfo
}
