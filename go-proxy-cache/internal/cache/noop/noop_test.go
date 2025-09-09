package noop

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go-proxy-cache/internal/models"
)

func TestNewNoOpCache(t *testing.T) {
	cache := NewNoOpCache()

	assert.NotNil(t, cache)
	noOpCache, ok := cache.(*NoOpCache)
	assert.True(t, ok)
	assert.NotNil(t, noOpCache)
}

func TestNoOpCache_Get_AlwaysReturnsNotFound(t *testing.T) {
	cache := NewNoOpCache()

	testCases := []string{
		"test-key",
		"another-key",
		"",
		"very-long-key-name-that-should-still-return-not-found",
	}

	for _, key := range testCases {
		t.Run("key: "+key, func(t *testing.T) {
			result, found := cache.Get(key)
			assert.False(t, found)
			assert.Nil(t, result)
		})
	}
}

func TestNoOpCache_GetStale_AlwaysReturnsNotFound(t *testing.T) {
	cache := NewNoOpCache()

	testCases := []string{
		"test-key",
		"another-key",
		"",
		"stale-key",
	}

	for _, key := range testCases {
		t.Run("key: "+key, func(t *testing.T) {
			result, found := cache.GetStale(key)
			assert.False(t, found)
			assert.Nil(t, result)
		})
	}
}

func TestNoOpCache_Set_DoesNothing(t *testing.T) {
	cache := NewNoOpCache()

	testData := []byte("test-value")
	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	// Set should not panic
	cache.Set("test-key", testData, testTTL)

	// Verify it still returns not found
	result, found := cache.Get("test-key")
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestNoOpCache_Delete_DoesNothing(t *testing.T) {
	cache := NewNoOpCache()

	// Delete should not panic
	cache.Delete("test-key")
	cache.Delete("")
	cache.Delete("non-existent-key")
}

func TestNoOpCache_Multiple_Operations(t *testing.T) {
	cache := NewNoOpCache()

	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	// Perform multiple operations
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))

		// Set
		cache.Set(key, value, testTTL)

		// Get
		result, found := cache.Get(key)
		assert.False(t, found)
		assert.Nil(t, result)

		// GetStale
		result, found = cache.GetStale(key)
		assert.False(t, found)
		assert.Nil(t, result)

		// Delete
		cache.Delete(key)
	}
}

func TestNoOpCache_Concurrent_Operations(t *testing.T) {
	cache := NewNoOpCache()

	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}
	numGoroutines := 10
	numOperations := 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent-key-%d-%d", id, j)
				value := []byte(fmt.Sprintf("value-%d-%d", id, j))

				// All operations should be safe and not panic
				cache.Set(key, value, testTTL)
				cache.Get(key)
				cache.GetStale(key)
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
