package mocks

import (
	"context"
	"time"

	"github.com/status-im/eth-rpc-proxy/provider"
	requestsrunner "github.com/status-im/eth-rpc-proxy/requests_runner"
)

// EVMMethodCaller implements the EVMMethodCaller interface for testing
type EVMMethodCaller struct {
	Responses       map[string]requestsrunner.ProviderResult
	MethodResponses map[string]map[string]requestsrunner.ProviderResult
}

func (m *EVMMethodCaller) CallMethod(
	ctx context.Context,
	provider provider.RPCProvider,
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
