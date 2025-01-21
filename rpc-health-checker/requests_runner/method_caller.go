package requestsrunner

import (
	"context"
	"time"

	"github.com/status-im/eth-rpc-proxy/provider"
)

// MethodCaller defines the interface for calling EVM methods on RPC providers
type MethodCaller interface {
	CallMethod(
		ctx context.Context,
		provider provider.RPCProvider,
		method string,
		params []interface{},
		timeout time.Duration,
	) ProviderResult
}
