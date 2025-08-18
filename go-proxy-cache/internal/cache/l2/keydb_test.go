package l2

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"go-proxy-cache/internal/config"
	"go-proxy-cache/internal/interfaces/mock"
	"go-proxy-cache/internal/models"
)

func TestNewKeyDBCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger)

	assert.NotNil(t, cache)
	keydbCache, ok := cache.(*KeyDBCache)
	assert.True(t, ok)
	assert.Equal(t, mockClient, keydbCache.client)
	assert.Equal(t, cfg, keydbCache.config)
	assert.Equal(t, logger, keydbCache.logger)
}

func TestKeyDBCache_Get_Success_Fresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Prepare test data - fresh entry
	now := time.Now().Unix()
	entry := models.CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 100,
		StaleAt:   now + 100, // Fresh
		ExpiresAt: now + 200,
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	result, found := cache.Get("test-key")

	// Assert
	assert.True(t, found)
	assert.NotNil(t, result)
	assert.True(t, result.IsFresh())
	assert.Equal(t, []byte("test-data"), result.Data)
}

func TestKeyDBCache_Get_Success_Stale(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Prepare test data - stale but not expired
	now := time.Now().Unix()
	entry := models.CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 200,
		StaleAt:   now - 50,  // Stale
		ExpiresAt: now + 100, // Not expired
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	result, found := cache.Get("test-key")

	// Assert
	assert.True(t, found)
	assert.NotNil(t, result)
	assert.False(t, result.IsFresh())
	assert.Equal(t, []byte("test-data"), result.Data)
}

func TestKeyDBCache_Get_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Mock expectations
	stringCmd := redis.NewStringResult("", redis.Nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	result, found := cache.Get("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestKeyDBCache_Get_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Mock expectations
	stringCmd := redis.NewStringResult("", errors.New("redis error"))
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	result, found := cache.Get("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestKeyDBCache_Get_Expired(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Prepare test data - expired entry
	now := time.Now().Unix()
	entry := models.CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 300,
		StaleAt:   now - 200,
		ExpiresAt: now - 100, // Expired
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)
	// Expect delete call for expired entry
	intCmd := redis.NewIntResult(1, nil)
	mockClient.EXPECT().Del(gomock.Any(), "test-key").Return(intCmd)

	// Execute
	result, found := cache.Get("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestKeyDBCache_Get_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Mock expectations - return invalid JSON
	stringCmd := redis.NewStringResult("invalid-json", nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)
	// Expect delete call for corrupted entry
	intCmd := redis.NewIntResult(1, nil)
	mockClient.EXPECT().Del(gomock.Any(), "test-key").Return(intCmd)

	// Execute
	result, found := cache.Get("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestKeyDBCache_GetStale_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Prepare test data - stale but not expired
	now := time.Now().Unix()
	entry := models.CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 200,
		StaleAt:   now - 50,  // Stale
		ExpiresAt: now + 100, // Not expired
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	result, found := cache.GetStale("test-key")

	// Assert
	assert.True(t, found)
	assert.NotNil(t, result)
	assert.Equal(t, []byte("test-data"), result.Data)
}

func TestKeyDBCache_GetStale_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Mock expectations
	stringCmd := redis.NewStringResult("", redis.Nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	result, found := cache.GetStale("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestKeyDBCache_GetStale_Expired(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Prepare test data - completely expired
	now := time.Now().Unix()
	entry := models.CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 300,
		StaleAt:   now - 200,
		ExpiresAt: now - 100, // Expired
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)
	// Expect delete call for expired entry
	intCmd := redis.NewIntResult(1, nil)
	mockClient.EXPECT().Del(gomock.Any(), "test-key").Return(intCmd)

	// Execute
	result, found := cache.GetStale("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestKeyDBCache_Set_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	testData := []byte("test-data")
	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	// Mock expectations
	statusCmd := redis.NewStatusResult("OK", nil)
	mockClient.EXPECT().Set(gomock.Any(), "test-key", gomock.Any(), 90*time.Second).Return(statusCmd)

	// Execute
	cache.Set("test-key", testData, testTTL)

	// No assertions needed - just verify no panic
}

func TestKeyDBCache_Delete_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Mock expectations
	intCmd := redis.NewIntResult(1, nil)
	mockClient.EXPECT().Del(gomock.Any(), "test-key").Return(intCmd)

	// Execute
	cache.Delete("test-key")

	// No assertions needed - just verify no panic
}
