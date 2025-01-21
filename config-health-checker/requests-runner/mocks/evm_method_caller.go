package mocks

import (
	"context"
	"time"

	requestsrunner "github.com/friofry/config-health-checker/requests-runner"
	"github.com/friofry/config-health-checker/rpcprovider"
)

// EVMMethodCaller implements the EVMMethodCaller interface for testing
type EVMMethodCaller struct {
	Responses       map[string]requestsrunner.ProviderResult
	MethodResponses map[string]map[string]requestsrunner.ProviderResult
}

func (m *EVMMethodCaller) CallEVMMethod(
	ctx context.Context,
	provider rpcprovider.RpcProvider,
	method string,
	params []interface{},
	timeout time.Duration,
) requestsrunner.ProviderResult {
	// Check if there are method-specific responses
	if methodResponses, ok := m.MethodResponses[provider.Name]; ok {
		if response, ok := methodResponses[method]; ok {
			return response
		}
	}
	// Fall back to general provider response
	return m.Responses[provider.Name]
}
