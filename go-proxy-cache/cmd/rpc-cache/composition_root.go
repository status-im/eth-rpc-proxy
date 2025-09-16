package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"

	"go-proxy-cache/internal/cache"
	"go-proxy-cache/internal/cache/l1"
	"go-proxy-cache/internal/cache/l2"
	"go-proxy-cache/internal/cache/noop"
	"go-proxy-cache/internal/cache/service"
	"go-proxy-cache/internal/cache_rules"
	"go-proxy-cache/internal/config"
	"go-proxy-cache/internal/httpserver"
	"go-proxy-cache/internal/interfaces"
)

// CompositionRoot holds all application dependencies and provides a centralized
// place for dependency injection and service initialization.
// This pattern helps with:
// - Centralized dependency management
// - Easier testing (can inject mocks)
// - Clear separation of concerns
// - Proper resource cleanup
type CompositionRoot struct {
	// Configuration
	Config     *config.Config
	Logger     *zap.Logger
	CacheRules interfaces.CacheRulesClassifier

	// Cache components
	L1Cache    interfaces.Cache
	L2Cache    interfaces.Cache
	KeyBuilder interfaces.KeyBuilder

	// Services
	CacheService *service.CacheService
	HTTPServer   *httpserver.Server
}

// NewCompositionRoot creates and initializes all application dependencies.
// It follows the dependency injection pattern where all services are created
// and wired together in the correct order.
//
// Initialization order:
// 1. Logger (needed by all other components)
// 2. Configuration (defines how components should be configured)
// 3. Cache rules (defines caching policies)
// 4. Cache components (L1, L2, KeyBuilder)
// 5. Services (CacheService with metrics)
// 6. HTTP Server (uses all above components)
func NewCompositionRoot() (*CompositionRoot, error) {
	root := &CompositionRoot{}

	// Initialize logger first
	if err := root.initLogger(); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Load configuration
	if err := root.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Load cache rules
	if err := root.loadCacheRules(); err != nil {
		return nil, fmt.Errorf("failed to load cache rules: %w", err)
	}

	// Initialize cache components
	if err := root.initCacheComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize cache components: %w", err)
	}

	// Initialize services
	if err := root.initServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	// Initialize HTTP server
	if err := root.initHTTPServer(); err != nil {
		return nil, fmt.Errorf("failed to initialize HTTP server: %w", err)
	}

	return root, nil
}

// initLogger initializes the application logger
func (r *CompositionRoot) initLogger() error {
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}
	r.Logger = logger
	return nil
}

// loadConfig loads the application configuration
func (r *CompositionRoot) loadConfig() error {
	configPath := os.Getenv("CACHE_CONFIG_FILE")
	if configPath == "" {
		configPath = "/app/cache_config.yaml"
	}

	cfg, err := config.LoadConfig(configPath, r.Logger)
	if err != nil {
		return err
	}

	r.Config = cfg
	return nil
}

// loadCacheRules loads cache rules configuration
func (r *CompositionRoot) loadCacheRules() error {
	rulesPath := os.Getenv("CACHE_RULES_FILE")
	if rulesPath == "" {
		rulesPath = "/app/cache_rules.yaml"
	}

	cacheRules, err := cache_rules.LoadCacheRulesConfig(rulesPath, r.Logger)
	if err != nil {
		return err
	}

	// Create classifier from the loaded config
	r.CacheRules = cache_rules.NewClassifier(r.Logger, cacheRules)
	return nil
}

// initCacheComponents initializes all cache-related components
func (r *CompositionRoot) initCacheComponents() error {
	// Initialize L1 cache (BigCache)
	if err := r.initL1Cache(); err != nil {
		return fmt.Errorf("failed to initialize L1 cache: %w", err)
	}

	// Initialize L2 cache (KeyDB)
	if err := r.initL2Cache(); err != nil {
		return fmt.Errorf("failed to initialize L2 cache: %w", err)
	}

	// Initialize key builder
	r.KeyBuilder = cache.NewKeyBuilder()

	return nil
}

// initL1Cache initializes the L1 cache (BigCache)
func (r *CompositionRoot) initL1Cache() error {
	if r.Config.BigCache.Enabled {
		l1Cache, err := l1.NewBigCache(&r.Config.BigCache, r.Logger)
		if err != nil {
			return err
		}
		r.L1Cache = l1Cache
		r.Logger.Info("BigCache (L1) initialized", zap.Int("size_mb", r.Config.BigCache.Size))
	} else {
		r.L1Cache = noop.NewNoOpCache()
		r.Logger.Info("BigCache (L1) disabled")
	}
	return nil
}

// initL2Cache initializes the L2 cache (KeyDB)
func (r *CompositionRoot) initL2Cache() error {
	if r.Config.KeyDB.Enabled {
		keydbURL := GetKeyDBURL(r.Logger)

		// Create KeyDB client
		keydbClient, err := l2.NewRedisKeyDbClient(&r.Config.KeyDB, keydbURL, r.Logger)
		if err != nil {
			r.Logger.Warn("Failed to connect to KeyDB, falling back to no L2 cache",
				zap.String("keydb_url", keydbURL),
				zap.Error(err))
			r.L2Cache = noop.NewNoOpCache()
			return nil
		}

		// Create L2 cache with the client
		r.L2Cache = l2.NewKeyDBCache(&r.Config.KeyDB, keydbClient, r.Logger)
		r.Logger.Info("KeyDB (L2) initialized", zap.String("keydb_url", keydbURL))
	} else {
		r.L2Cache = noop.NewNoOpCache()
		r.Logger.Info("KeyDB (L2) disabled")
	}
	return nil
}

// initServices initializes application services
func (r *CompositionRoot) initServices() error {
	// Initialize cache service
	r.CacheService = service.NewCacheService(
		r.L1Cache,
		r.L2Cache,
		r.CacheRules,
		r.Config.MultiCache.EnablePropagation,
		r.Logger,
	)

	return nil
}

// initHTTPServer initializes the HTTP server
func (r *CompositionRoot) initHTTPServer() error {
	r.HTTPServer = httpserver.NewServer(
		r.CacheService,
		r.Logger,
	)

	return nil
}

// Cleanup performs cleanup of all resources
func (r *CompositionRoot) Cleanup() error {
	var errors []error

	// Sync logger
	if r.Logger != nil {
		if err := r.Logger.Sync(); err != nil {
			errors = append(errors, fmt.Errorf("failed to sync logger: %w", err))
		}
	}

	// Close L1 cache
	if r.L1Cache != nil {
		if l1BigCache, ok := r.L1Cache.(*l1.BigCache); ok {
			if err := l1BigCache.Close(); err != nil {
				errors = append(errors, fmt.Errorf("failed to close L1 cache: %w", err))
			}
		}
	}

	// Close L2 cache
	if r.L2Cache != nil {
		if l2KeyDBCache, ok := r.L2Cache.(*l2.KeyDBCache); ok {
			if err := l2KeyDBCache.Close(); err != nil {
				errors = append(errors, fmt.Errorf("failed to close L2 cache: %w", err))
			}
		}
	}

	// Return first error if any
	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}

// GetSocketPath returns the Unix socket path for the server
func (r *CompositionRoot) GetSocketPath() string {
	socketPath := os.Getenv("CACHE_SOCKET_PATH")
	if socketPath == "" {
		socketPath = "/tmp/cache.sock"
	}
	return socketPath
}
