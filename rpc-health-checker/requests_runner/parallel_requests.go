package requestsrunner

import (
	"context"
	"fmt"
	"sync"
	"time"

	rpcprovider "github.com/status-im/eth-rpc-proxy/provider"
)

// ProviderResult contains information about the result of a provider check.
type ProviderResult struct {
	Success     bool          // Indicates if the request was successful
	Error       error         // Error if the request failed
	Response    []byte        // Response from the provider
	Result      string        // Result from the provider (if successful)
	ElapsedTime time.Duration // Duration taken to perform the request
}

// RequestFunc defines the type of function used to check a provider.
type RequestFunc func(ctx context.Context, provider rpcprovider.RPCProvider) ProviderResult

// ParallelCheckProviders performs concurrent checks on multiple RPC providers using the provided checker function.
// It does not limit the number of concurrent goroutines.
func ParallelCheckProviders(ctx context.Context, providers []rpcprovider.RPCProvider, timeout time.Duration, checker RequestFunc) map[string]ProviderResult {
	results := make(map[string]ProviderResult)
	resultsChan := make(chan struct {
		name   string
		result ProviderResult
	}, len(providers)) // Buffered channel to collect results

	// Create a child context with the specified timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var wg sync.WaitGroup

	for _, provider := range providers {
		// Increment the WaitGroup counter
		wg.Add(1)

		// Launch a goroutine for each provider
		go func(p rpcprovider.RPCProvider) {
			defer wg.Done()

			// Create a temporary channel to receive checker result
			tempChan := make(chan ProviderResult, 1)

			// Run the checker function in a separate goroutine
			go func() {
				result := checker(ctx, p)
				tempChan <- result
			}()

			// Wait for either the checker function to finish or the context to be done
			select {
			case res := <-tempChan:
				// Checker function completed
				resultsChan <- struct {
					name   string
					result ProviderResult
				}{name: p.Name, result: res}
			case <-ctx.Done():
				// Context canceled or timed out
				fmt.Println("Context canceled", p.Name)
				resultsChan <- struct {
					name   string
					result ProviderResult
				}{name: p.Name, result: ProviderResult{Success: false, Error: ctx.Err()}}
			}
		}(provider)
	}

	// Launch a goroutine to close the results channel once all checks are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results from the channel
	for entry := range resultsChan {
		results[entry.name] = entry.result
	}

	return results
}

// ParallelCallEVMMethods executes EVM methods in parallel across multiple providers
func ParallelCallEVMMethods(
	ctx context.Context,
	providers []rpcprovider.RPCProvider,
	method string,
	params []interface{},
	timeout time.Duration,
	caller MethodCaller,
) map[string]ProviderResult {
	// Create a RequestFunc that wraps CallMethod with the given method and params
	checker := func(ctx context.Context, provider rpcprovider.RPCProvider) ProviderResult {
		return caller.CallMethod(ctx, provider, method, params, timeout)
	}

	// Use ParallelCheckProviders to execute the calls in parallel
	return ParallelCheckProviders(ctx, providers, timeout, checker)
}
