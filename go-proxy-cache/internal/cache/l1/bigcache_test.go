package l1

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"go-proxy-cache/internal/models"
)

func TestNewBigCache(t *testing.T) {
	logger := zap.NewNop()

	cache, err := NewBigCache(10, logger)

	assert.NoError(t, err)
	assert.NotNil(t, cache)

	bigCache, ok := cache.(*BigCache)
	assert.True(t, ok)
	assert.NotNil(t, bigCache.cache)
	assert.Equal(t, logger, bigCache.logger)
}

func TestBigCache_Set_And_Get_Fresh(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	assert.NoError(t, err)

	testData := []byte("test-value")
	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	// Set the value
	cache.Set("test-key", testData, testTTL)

	// Get the value immediately (should be fresh)
	result, found := cache.Get("test-key")

	assert.True(t, found)
	assert.NotNil(t, result)
	assert.True(t, result.IsFresh())
	assert.Equal(t, testData, result.Data)
}

func TestBigCache_Get_NotFound(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	assert.NoError(t, err)

	// Try to get non-existent key
	result, found := cache.Get("non-existent-key")

	assert.False(t, found)
	assert.Nil(t, result)
}

func TestBigCache_Set_And_Get_Stale(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	assert.NoError(t, err)

	// Create a cache entry that's already stale
	now := time.Now().Unix()
	testData := []byte("test-value")

	// Manually create a stale entry by setting timestamps in the past
	bigCache := cache.(*BigCache)
	entry := models.CacheEntry{
		Data:      testData,
		CreatedAt: now - 200,
		StaleAt:   now - 50,  // Already stale
		ExpiresAt: now + 100, // Not expired
	}

	// Manually marshal and set the entry
	entryJSON, _ := json.Marshal(entry)
	bigCache.cache.Set("test-key", entryJSON)

	// Get the value (should be stale but not expired)
	result, found := cache.Get("test-key")

	assert.True(t, found)
	assert.NotNil(t, result)
	assert.False(t, result.IsFresh())
	assert.Equal(t, testData, result.Data)
}

func TestBigCache_Set_And_Get_Expired(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	assert.NoError(t, err)

	// Create a cache entry that's already expired
	now := time.Now().Unix()
	testData := []byte("test-value")

	// Manually create an expired entry
	bigCache := cache.(*BigCache)
	entry := models.CacheEntry{
		Data:      testData,
		CreatedAt: now - 300,
		StaleAt:   now - 200,
		ExpiresAt: now - 100, // Already expired
	}

	// Manually marshal and set the entry
	entryJSON, _ := json.Marshal(entry)
	bigCache.cache.Set("test-key", entryJSON)

	// Get the value (should be expired and not found)
	result, found := cache.Get("test-key")

	assert.False(t, found)
	assert.Nil(t, result)
}

func TestBigCache_GetStale_Success(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	assert.NoError(t, err)

	// Create a cache entry that's stale but not expired
	now := time.Now().Unix()
	testData := []byte("test-value")

	// Manually create a stale entry
	bigCache := cache.(*BigCache)
	entry := models.CacheEntry{
		Data:      testData,
		CreatedAt: now - 200,
		StaleAt:   now - 50,  // Already stale
		ExpiresAt: now + 100, // Not expired
	}

	// Manually marshal and set the entry
	entryJSON, _ := json.Marshal(entry)
	bigCache.cache.Set("test-key", entryJSON)

	// Get stale value
	result, found := cache.GetStale("test-key")

	assert.True(t, found)
	assert.NotNil(t, result)
	assert.Equal(t, testData, result.Data)
}

func TestBigCache_GetStale_NotFound(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	assert.NoError(t, err)

	// Try to get stale value for non-existent key
	result, found := cache.GetStale("non-existent-key")

	assert.False(t, found)
	assert.Nil(t, result)
}

func TestBigCache_GetStale_Expired(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	assert.NoError(t, err)

	// Create a cache entry that's completely expired
	now := time.Now().Unix()
	testData := []byte("test-value")

	// Manually create an expired entry
	bigCache := cache.(*BigCache)
	entry := models.CacheEntry{
		Data:      testData,
		CreatedAt: now - 300,
		StaleAt:   now - 200,
		ExpiresAt: now - 100, // Already expired
	}

	// Manually marshal and set the entry
	entryJSON, _ := json.Marshal(entry)
	bigCache.cache.Set("test-key", entryJSON)

	// Try to get stale value (should be expired)
	result, found := cache.GetStale("test-key")

	assert.False(t, found)
	assert.Nil(t, result)
}

func TestBigCache_Delete(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	assert.NoError(t, err)

	testData := []byte("test-value")
	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	// Set the value
	cache.Set("test-key", testData, testTTL)

	// Verify it exists
	result, found := cache.Get("test-key")
	assert.True(t, found)
	assert.NotNil(t, result)

	// Delete it
	cache.Delete("test-key")

	// Verify it's gone
	result, found = cache.Get("test-key")
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestBigCache_Delete_NonExistent(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	assert.NoError(t, err)

	// Delete non-existent key (should not panic)
	cache.Delete("non-existent-key")
}

func TestBigCache_Multiple_Keys(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	assert.NoError(t, err)

	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	// Set multiple keys
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))
		cache.Set(key, value, testTTL)
	}

	// Verify all keys exist
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key-%d", i)
		expectedValue := []byte(fmt.Sprintf("value-%d", i))

		result, found := cache.Get(key)
		assert.True(t, found)
		assert.NotNil(t, result)
		assert.Equal(t, expectedValue, result.Data)
	}
}

func TestBigCache_Concurrent_Access(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	assert.NoError(t, err)

	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}
	numGoroutines := 10
	numOperations := 100

	// Run concurrent operations
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent-key-%d-%d", id, j)
				value := []byte(fmt.Sprintf("value-%d-%d", id, j))

				// Set
				cache.Set(key, value, testTTL)

				// Get
				result, found := cache.Get(key)
				if found {
					assert.NotNil(t, result)
					assert.Equal(t, value, result.Data)
				}

				// Delete
				cache.Delete(key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestBigCache_Edge_Cases(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	assert.NoError(t, err)

	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	t.Run("empty key", func(t *testing.T) {
		cache.Set("", []byte("value"), testTTL)
		result, found := cache.Get("")
		assert.True(t, found)
		assert.NotNil(t, result)
		assert.Equal(t, []byte("value"), result.Data)
	})

	t.Run("empty value", func(t *testing.T) {
		cache.Set("empty-value-key", []byte(""), testTTL)
		result, found := cache.Get("empty-value-key")
		assert.True(t, found)
		assert.NotNil(t, result)
		assert.Equal(t, []byte(""), result.Data)
	})

	t.Run("nil value", func(t *testing.T) {
		cache.Set("nil-value-key", nil, testTTL)
		result, found := cache.Get("nil-value-key")
		assert.True(t, found)
		assert.NotNil(t, result)
		assert.Nil(t, result.Data)
	})
}
