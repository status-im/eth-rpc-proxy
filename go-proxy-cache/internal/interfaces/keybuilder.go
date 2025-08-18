package interfaces

//go:generate mockgen -package=mock -source=keybuilder.go -destination=mock/keybuilder.go

// KeyBuilder canonizes requests into deterministic cache keys
type KeyBuilder interface {
	// For single request
	Build(chain string, network string, req *JSONRPCRequest) (key string, paramsHash uint32)
	// For batch, returns per-item keys aligned by index
	BuildBatch(chain, network string, reqs []JSONRPCRequest) ([]string, []uint32)
}
