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

	// Prepare test data
	now := time.Now().Unix()
	entry := CacheEntry{
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
	val, fresh, found := cache.Get("test-key")

	// Assert
	assert.True(t, found)
	assert.True(t, fresh)
	assert.Equal(t, []byte("test-data"), val)
}

func TestKeyDBCache_Get_Success_Stale(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Prepare test data
	now := time.Now().Unix()
	entry := CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 200,
		StaleAt:   now - 50, // Stale but not expired
		ExpiresAt: now + 100,
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	val, fresh, found := cache.Get("test-key")

	// Assert
	assert.True(t, found)
	assert.False(t, fresh)
	assert.Equal(t, []byte("test-data"), val)
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
	val, fresh, found := cache.Get("test-key")

	// Assert
	assert.False(t, found)
	assert.False(t, fresh)
	assert.Nil(t, val)
}

func TestKeyDBCache_Get_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Mock expectations
	stringCmd := redis.NewStringResult("", errors.New("connection error"))
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	val, fresh, found := cache.Get("test-key")

	// Assert
	assert.False(t, found)
	assert.False(t, fresh)
	assert.Nil(t, val)
}

func TestKeyDBCache_Get_Expired(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Prepare test data
	now := time.Now().Unix()
	entry := CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 300,
		StaleAt:   now - 200,
		ExpiresAt: now - 100, // Expired
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	intCmd := redis.NewIntResult(1, nil)
	mockClient.EXPECT().Del(gomock.Any(), "test-key").Return(intCmd)

	// Execute
	val, fresh, found := cache.Get("test-key")

	// Assert
	assert.False(t, found)
	assert.False(t, fresh)
	assert.Nil(t, val)
}

func TestKeyDBCache_Get_CorruptedEntry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Mock expectations - return invalid JSON
	stringCmd := redis.NewStringResult("invalid-json", nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	intCmd := redis.NewIntResult(1, nil)
	mockClient.EXPECT().Del(gomock.Any(), "test-key").Return(intCmd)

	// Execute
	val, fresh, found := cache.Get("test-key")

	// Assert
	assert.False(t, found)
	assert.False(t, fresh)
	assert.Nil(t, val)
}

func TestKeyDBCache_GetStale_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Prepare test data
	now := time.Now().Unix()
	entry := CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 200,
		StaleAt:   now - 50, // Stale but not expired
		ExpiresAt: now + 100,
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	val, found := cache.GetStale("test-key")

	// Assert
	assert.True(t, found)
	assert.Equal(t, []byte("test-data"), val)
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
	val, found := cache.GetStale("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, val)
}

func TestKeyDBCache_GetStale_Expired(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Prepare test data
	now := time.Now().Unix()
	entry := CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 300,
		StaleAt:   now - 200,
		ExpiresAt: now - 100, // Expired
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	intCmd := redis.NewIntResult(1, nil)
	mockClient.EXPECT().Del(gomock.Any(), "test-key").Return(intCmd)

	// Execute
	val, found := cache.GetStale("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, val)
}

func TestKeyDBCache_Set_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	ttl := models.TTL{
		Fresh: 60 * time.Second,
		Stale: 30 * time.Second,
	}

	// Mock expectations
	statusCmd := redis.NewStatusResult("OK", nil)
	mockClient.EXPECT().Set(gomock.Any(), "test-key", gomock.Any(), 90*time.Second).Return(statusCmd)

	// Execute
	cache.Set("test-key", []byte("test-data"), ttl)

	// No assertions needed as Set doesn't return anything
	// The test passes if no panic occurs and mock expectations are met
}

func TestKeyDBCache_Set_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	ttl := models.TTL{
		Fresh: 60 * time.Second,
		Stale: 30 * time.Second,
	}

	// Mock expectations
	statusCmd := redis.NewStatusResult("", errors.New("set error"))
	mockClient.EXPECT().Set(gomock.Any(), "test-key", gomock.Any(), 90*time.Second).Return(statusCmd)

	// Execute
	cache.Set("test-key", []byte("test-data"), ttl)

	// No assertions needed as Set doesn't return anything
	// The test passes if no panic occurs and mock expectations are met
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

	// No assertions needed as Delete doesn't return anything
	// The test passes if no panic occurs and mock expectations are met
}

func TestKeyDBCache_Delete_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Mock expectations
	intCmd := redis.NewIntResult(0, errors.New("delete error"))
	mockClient.EXPECT().Del(gomock.Any(), "test-key").Return(intCmd)

	// Execute
	cache.Delete("test-key")

	// No assertions needed as Delete doesn't return anything
	// The test passes if no panic occurs and mock expectations are met
}

func TestKeyDBCache_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	// Mock expectations
	mockClient.EXPECT().Close().Return(nil)

	// Execute
	err := cache.Close()

	// Assert
	assert.NoError(t, err)
}

func TestKeyDBCache_Close_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &config.Config{}
	logger := zap.NewNop()

	cache := NewKeyDBCache(cfg, mockClient, logger).(*KeyDBCache)

	expectedErr := errors.New("close error")

	// Mock expectations
	mockClient.EXPECT().Close().Return(expectedErr)

	// Execute
	err := cache.Close()

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}
