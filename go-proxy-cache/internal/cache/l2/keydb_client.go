package l2

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"go-proxy-cache/internal/config"
	"go-proxy-cache/internal/interfaces"
)

// Ensure RedisKeyDbClient implements interfaces.KeyDbClient
var _ interfaces.KeyDbClient = (*RedisKeyDbClient)(nil)

// RedisKeyDbClient wraps redis.Client to implement KeyDbClient interface
type RedisKeyDbClient struct {
	client *redis.Client
	logger *zap.Logger
}

// NewRedisKeyDbClient creates a new RedisKeyDbClient instance
func NewRedisKeyDbClient(keydbCfg *config.KeyDBConfig, keydbURL string, logger *zap.Logger) (interfaces.KeyDbClient, error) {
	// Parse KeyDB URL
	parsedURL, err := url.Parse(keydbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse KeyDB URL: %w", err)
	}

	// Extract host and port
	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "6379" // Default Redis port
	}

	// Create Redis client options
	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		DialTimeout:  keydbCfg.Connection.ConnectTimeout,
		ReadTimeout:  keydbCfg.Connection.ReadTimeout,
		WriteTimeout: keydbCfg.Connection.SendTimeout,
		PoolSize:     keydbCfg.Keepalive.PoolSize,
		IdleTimeout:  keydbCfg.Keepalive.MaxIdleTimeout,
	}

	// Handle password if present in URL
	if parsedURL.User != nil {
		if password, ok := parsedURL.User.Password(); ok {
			opts.Password = password
		}
	}

	// Handle database number if present in URL path
	if parsedURL.Path != "" && len(parsedURL.Path) > 1 {
		if db, err := strconv.Atoi(parsedURL.Path[1:]); err == nil {
			opts.DB = db
		}
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), keydbCfg.Connection.ConnectTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close() // Clean up the client
		return nil, fmt.Errorf("failed to connect to KeyDB at %s: %w", opts.Addr, err)
	}

	logger.Info("Connected to KeyDB",
		zap.String("address", opts.Addr),
		zap.Duration("connect_timeout", keydbCfg.Connection.ConnectTimeout),
		zap.Int("pool_size", keydbCfg.Keepalive.PoolSize))

	return &RedisKeyDbClient{
		client: client,
		logger: logger,
	}, nil
}

// Get retrieves a value by key
func (r *RedisKeyDbClient) Get(ctx context.Context, key string) *redis.StringCmd {
	return r.client.Get(ctx, key)
}

// Set stores a value with expiration
func (r *RedisKeyDbClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return r.client.Set(ctx, key, value, expiration)
}

// Del deletes one or more keys
func (r *RedisKeyDbClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return r.client.Del(ctx, keys...)
}

// Ping tests connectivity
func (r *RedisKeyDbClient) Ping(ctx context.Context) *redis.StatusCmd {
	return r.client.Ping(ctx)
}

// Close closes the client connection
func (r *RedisKeyDbClient) Close() error {
	return r.client.Close()
}
