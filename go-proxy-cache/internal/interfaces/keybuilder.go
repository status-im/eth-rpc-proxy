package interfaces

import "go-proxy-cache/internal/models"

//go:generate mockgen -package=mock -source=keybuilder.go -destination=mock/keybuilder.go

// KeyBuilder canonizes requests into deterministic cache keys
type KeyBuilder interface {
	// For single request
	Build(chain string, network string, req *models.JSONRPCRequest) (string, error)
	// For batch, returns per-item keys aligned by index
	BuildBatch(chain, network string, reqs []models.JSONRPCRequest) ([]string, error)
}
