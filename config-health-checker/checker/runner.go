package checker

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/friofry/config-health-checker/chainconfig"
	"github.com/friofry/config-health-checker/configreader"
	requestsrunner "github.com/friofry/config-health-checker/requests-runner"
	"github.com/friofry/config-health-checker/rpcprovider"
	"github.com/friofry/config-health-checker/rpctestsconfig"
)

func loadChainsToMap(filePath string) (map[int64]chainconfig.ChainConfig, error) {
	chains, err := chainconfig.LoadChains(filePath)
	if err != nil {
		return nil, err
	}

	chainMap := make(map[int64]chainconfig.ChainConfig)
	for _, chain := range chains.Chains {
		chainMap[int64(chain.ChainId)] = chain
	}
	return chainMap, nil
}

func loadReferenceChainsToMap(filePath string) (map[int64]chainconfig.ReferenceChainConfig, error) {
	chains, err := chainconfig.LoadReferenceChains(filePath)
	if err != nil {
		return nil, err
	}

	chainMap := make(map[int64]chainconfig.ReferenceChainConfig)
	for _, chain := range chains.Chains {
		chainMap[int64(chain.ChainId)] = chain
	}
	return chainMap, nil
}

// ChainValidationRunner coordinates validation across multiple chains
type ChainValidationRunner struct {
	chainConfigs        map[int64]chainconfig.ChainConfig
	referenceChainCfgs  map[int64]chainconfig.ReferenceChainConfig
	methodConfigs       []rpctestsconfig.EVMMethodTestConfig
	caller              requestsrunner.EVMMethodCaller
	timeout             time.Duration
	outputProvidersPath string
	logger              *slog.Logger
}

// NewChainValidationRunner creates a new validation runner
func NewChainValidationRunner(
	chainCfgs map[int64]chainconfig.ChainConfig,
	referenceCfgs map[int64]chainconfig.ReferenceChainConfig,
	methodConfigs []rpctestsconfig.EVMMethodTestConfig,
	caller requestsrunner.EVMMethodCaller,
	timeout time.Duration,
	outputProvidersPath string,
	logPath string,
) *ChainValidationRunner {
	// Set up logging
	var logWriter io.Writer = os.Stdout
	if logPath != "" {
		file, err := os.Create(logPath)
		if err != nil {
			panic(fmt.Sprintf("failed to create log file: %v", err))
		}
		logWriter = file
	}

	logger := slog.New(slog.NewJSONHandler(logWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return &ChainValidationRunner{
		chainConfigs:        chainCfgs,
		referenceChainCfgs:  referenceCfgs,
		methodConfigs:       methodConfigs,
		caller:              caller,
		timeout:             timeout,
		outputProvidersPath: outputProvidersPath,
		logger:              logger,
	}
}

// Run executes validation across all configured chains and writes valid providers to output file
func (r *ChainValidationRunner) Run(ctx context.Context) {
	validChains, results := r.validateChains(ctx)
	r.logger.Info("validation results", "results", results)
	r.writeValidChains(validChains)
}

// validateChains runs validation for all chains and returns valid chains and validation results
func (r *ChainValidationRunner) validateChains(ctx context.Context) ([]chainconfig.ChainConfig, map[int64]map[string]ProviderValidationResult) {
	var validChains []chainconfig.ChainConfig
	results := make(map[int64]map[string]ProviderValidationResult)

	for chainId, chainCfg := range r.chainConfigs {
		if refCfg, exists := r.referenceChainCfgs[chainId]; exists {
			chainResults := r.validateChain(ctx, chainCfg, refCfg)
			results[chainId] = chainResults

			if validProviders := r.getValidProviders(chainCfg, chainResults); len(validProviders) > 0 {
				// Create a copy of the original chain config and update providers
				validChain := chainCfg
				validChain.Providers = validProviders
				validChains = append(validChains, validChain)
			}
		}
	}

	return validChains, results
}

// validateChain runs validation for a single chain
func (r *ChainValidationRunner) validateChain(
	ctx context.Context,
	chainCfg chainconfig.ChainConfig,
	refCfg chainconfig.ReferenceChainConfig,
) map[string]ProviderValidationResult {
	return ValidateMultipleEVMMethods(
		ctx,
		r.methodConfigs,
		r.caller,
		chainCfg.Providers,
		refCfg.Provider,
		r.timeout,
	)
}

// getValidProviders filters and returns valid providers from validation results
func (r *ChainValidationRunner) getValidProviders(
	chainCfg chainconfig.ChainConfig,
	results map[string]ProviderValidationResult,
) []rpcprovider.RpcProvider {
	var validProviders []rpcprovider.RpcProvider

	for providerName, result := range results {
		if result.Valid {
			if provider := r.findProviderByName(chainCfg.Providers, providerName); provider != nil {
				validProviders = append(validProviders, *provider)
			}
		}
	}

	return validProviders
}

// findProviderByName finds a provider by name in the list
func (r *ChainValidationRunner) findProviderByName(
	providers []rpcprovider.RpcProvider,
	name string,
) *rpcprovider.RpcProvider {
	for _, provider := range providers {
		if provider.Name == name {
			return &provider
		}
	}
	return nil
}

// writeValidChains writes valid chains to output file if path is specified
func (r *ChainValidationRunner) writeValidChains(validChains []chainconfig.ChainConfig) {
	if r.outputProvidersPath != "" {
		if err := chainconfig.WriteChains(r.outputProvidersPath, chainconfig.ChainsConfig{Chains: validChains}); err != nil {
			fmt.Printf("Failed to write valid providers: %v\n", err)
		}
	}
}

// NewRunnerFromConfig creates a new ChainValidationRunner from configreader.CheckerConfig
func NewRunnerFromConfig(
	cfg configreader.CheckerConfig,
	caller requestsrunner.EVMMethodCaller,
) (*ChainValidationRunner, error) {
	// Load reference chains
	referenceChains, err := loadReferenceChainsToMap(cfg.ReferenceProvidersPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load reference chains: %w", err)
	}

	// Load default chains
	defaultChains, err := loadChainsToMap(cfg.DefaultProvidersPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load default chains: %w", err)
	}

	// Load test configurations
	testConfigs, err := rpctestsconfig.ReadConfig(cfg.TestsConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load test configurations: %w", err)
	}

	return NewChainValidationRunner(
		defaultChains,
		referenceChains,
		testConfigs,
		caller,
		time.Duration(cfg.IntervalSeconds)*time.Second,
		cfg.OutputProvidersPath,
		"", // Empty log path for now
	), nil
}
