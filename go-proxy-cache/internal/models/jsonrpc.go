package models

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Jsonrpc string      `json:"jsonrpc"`
}
