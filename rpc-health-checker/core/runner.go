package core

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/status-im/eth-rpc-proxy/config"
	"github.com/status-im/eth-rpc-proxy/provider"
	requestsrunner "github.com/status-im/eth-rpc-proxy/requests_runner"
)

// ChainValidationRunner coordinates validation across multiple chains
type ChainValidationRunner struct {
	chainConfigs          map[int64]config.ChainConfig
	referenceChainConfigs map[int64]config.ReferenceChainConfig
	methodConfigs         []config.EVMMethodTestConfig
	caller                requestsrunner.MethodCaller
	timeout               time.Duration
	outputProvidersPath   string
	logger                *slog.Logger
}

// NewChainValidationRunner creates a new validation runner
func NewChainValidationRunner(
	chainConfigs map[int64]config.ChainConfig,
	referenceConfigs map[int64]config.ReferenceChainConfig,
	methodConfigs []config.EVMMethodTestConfig,
	caller requestsrunner.MethodCaller,
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
		chainConfigs:          chainConfigs,
		referenceChainConfigs: referenceConfigs,
		methodConfigs:         methodConfigs,
		caller:                caller,
		timeout:               timeout,
		outputProvidersPath:   outputProvidersPath,
		logger:                logger,
	}
}

// Run executes validation across all configured chains and writes valid providers to output file
func (r *ChainValidationRunner) Run(ctx context.Context) {
	startTime := time.Now()
	validChains, results := r.validateChains(ctx)
	defer RecordValidationMetrics(startTime, validChains, results, r.chainConfigs)
	r.writeValidChains(validChains)
}

// validateChains runs validation for all chains and returns valid chains and validation results (chainID -> provider -> result)
func (r *ChainValidationRunner) validateChains(ctx context.Context) ([]config.ChainConfig, map[int64]map[string]ProviderValidationResult) {
	var validChains []config.ChainConfig
	results := make(map[int64]map[string]ProviderValidationResult)

	for chainId, chainCfg := range r.chainConfigs {
		refCfg, exists := r.referenceChainConfigs[chainId]
		if !exists {
			r.logger.Warn("reference provider not found for chain, skipping validation",
				slog.Int64("chain_id", chainId),
			)
			continue
		}

		filteredMethods := filterMethodsForChain(r.methodConfigs, chainId)

		chainResults := ValidateMultipleEVMMethods(
			ctx,
			filteredMethods,
			r.caller,
			chainCfg.Providers,
			refCfg.Provider,
			r.timeout,
		)
		results[chainId] = chainResults

		validProviders := r.getValidProviders(chainCfg, chainResults)

		if len(validProviders) > 0 {
			// Create a copy of the original chain config and update providers
			validChain := chainCfg
			validChain.Providers = validProviders
			validChains = append(validChains, validChain)
		}
	}

	return validChains, results
}

// getValidProviders filters and returns valid providers from validation results
func (r *ChainValidationRunner) getValidProviders(
	chainCfg config.ChainConfig,
	results map[string]ProviderValidationResult,
) []provider.RPCProvider {
	var validProviders []provider.RPCProvider

	for _, providerConfigs := range chainCfg.Providers {
		if result, exists := results[providerConfigs.Name]; exists && result.Valid {
			validProviders = append(validProviders, providerConfigs)
		}
	}

	return validProviders
}

// writeValidChains writes valid chains to output file if path is specified
func (r *ChainValidationRunner) writeValidChains(validChains []config.ChainConfig) {
	if r.outputProvidersPath != "" {
		if len(validChains) == 0 {
			if err := os.Remove(r.outputProvidersPath); err != nil {
				fmt.Printf("Failed to remove output providers file: %v\n", err)
			}
			return
		}
		if err := config.WriteChains(r.outputProvidersPath, config.ChainsConfig{Chains: validChains}); err != nil {
			fmt.Printf("Failed to write valid providers: %v\n", err)
		}
	}
}

// filterMethodsForChain filters out methods that should be skipped for a specific chain
func filterMethodsForChain(configs []config.EVMMethodTestConfig, chainId int64) []config.EVMMethodTestConfig {
	var filtered []config.EVMMethodTestConfig
	for _, cfg := range configs {
		if cfg.SkipChains == nil || !cfg.SkipChains[chainId] {
			filtered = append(filtered, cfg)
		}
	}
	return filtered
}

func loadChainsToMap(filePath string) (map[int64]config.ChainConfig, error) {
	chains, err := config.LoadChains(filePath)
	if err != nil {
		return nil, err
	}

	chainMap := make(map[int64]config.ChainConfig)
	for _, chain := range chains.Chains {
		chainMap[int64(chain.ChainID)] = chain
	}
	return chainMap, nil
}

func loadReferenceChainsToMap(filePath string) (map[int64]config.ReferenceChainConfig, error) {
	chains, err := config.LoadReferenceChains(filePath)
	if err != nil {
		return nil, err
	}

	chainMap := make(map[int64]config.ReferenceChainConfig)
	for _, chain := range chains.Chains {
		chainMap[int64(chain.ChainId)] = chain
	}
	return chainMap, nil
}

// NewRunnerFromConfig creates a new ChainValidationRunner from config.CheckerConfig
func NewRunnerFromConfig(
	cfg config.CheckerConfig,
	caller requestsrunner.MethodCaller,
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
	testConfigs, err := config.ReadConfig(cfg.TestsConfigPath)
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
