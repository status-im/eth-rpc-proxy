package requestsrunner

import (
	"context"
	"time"

	"github.com/friofry/config-health-checker/rpcprovider"
)

// EVMMethodCaller defines the interface for calling EVM methods on RPC providers
type EVMMethodCaller interface {
	CallEVMMethod(
		ctx context.Context,
		provider rpcprovider.RpcProvider,
		method string,
		params []interface{},
		timeout time.Duration,
	) ProviderResult
}
