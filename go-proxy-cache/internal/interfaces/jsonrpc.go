package interfaces

//go:generate mockgen -package=mock -source=jsonrpc.go -destination=mock/jsonrpc.go

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Jsonrpc string      `json:"jsonrpc"`
}
