package l1

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go-proxy-cache/internal/models"
)

func TestNewBigCache(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name    string
		sizeMB  int
		wantErr bool
	}{
		{
			name:    "valid size",
			sizeMB:  10,
			wantErr: false,
		},
		{
			name:    "zero size",
			sizeMB:  0,
			wantErr: false,
		},
		{
			name:    "large size",
			sizeMB:  1000,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewBigCache(tt.sizeMB, logger)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cache)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cache)

				// Clean up
				if cache != nil {
					if bc, ok := cache.(*BigCache); ok {
						bc.Close()
					}
				}
			}
		})
	}
}

func TestBigCache_SetAndGet(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	require.NoError(t, err)
	defer func() {
		if bc, ok := cache.(*BigCache); ok {
			bc.Close()
		}
	}()

	key := "test-key"
	value := []byte("test-value")
	ttl := models.TTL{
		Fresh: 5 * time.Second,
		Stale: 10 * time.Second,
	}

	// Test Set and Get
	cache.Set(key, value, ttl)

	val, fresh, found := cache.Get(key)
	assert.True(t, found)
	assert.True(t, fresh)
	assert.Equal(t, value, val)
}

func TestBigCache_GetNonExistent(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	require.NoError(t, err)
	defer func() {
		if bc, ok := cache.(*BigCache); ok {
			bc.Close()
		}
	}()

	val, fresh, found := cache.Get("non-existent-key")
	assert.False(t, found)
	assert.False(t, fresh)
	assert.Nil(t, val)
}

func TestBigCache_GetStale(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	require.NoError(t, err)
	defer func() {
		if bc, ok := cache.(*BigCache); ok {
			bc.Close()
		}
	}()

	key := "test-key"
	value := []byte("test-value")
	ttl := models.TTL{
		Fresh: 1 * time.Second,
		Stale: 2 * time.Second,
	}

	// Set value
	cache.Set(key, value, ttl)

	// Wait for fresh period to expire but not stale period
	time.Sleep(1500 * time.Millisecond)

	// Should be stale but still available
	val, fresh, found := cache.Get(key)
	assert.True(t, found)
	assert.False(t, fresh) // Should be stale
	assert.Equal(t, value, val)

	// GetStale should still return the value
	staleVal, staleFound := cache.GetStale(key)
	assert.True(t, staleFound)
	assert.Equal(t, value, staleVal)
}

func TestBigCache_ExpiredEntry(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	require.NoError(t, err)
	defer func() {
		if bc, ok := cache.(*BigCache); ok {
			bc.Close()
		}
	}()

	key := "test-key"
	value := []byte("test-value")
	// Use very short TTL to ensure expiration
	ttl := models.TTL{
		Fresh: 500 * time.Millisecond,
		Stale: 500 * time.Millisecond,
	}

	// Set value
	cache.Set(key, value, ttl)

	// Immediately check that it's fresh
	val, fresh, found := cache.Get(key)
	assert.True(t, found)
	assert.True(t, fresh)
	assert.Equal(t, value, val)

	// Wait for both fresh and stale periods to expire (total: 1 second)
	time.Sleep(1200 * time.Millisecond)

	// Should not be found
	val, fresh, found = cache.Get(key)
	assert.False(t, found, "Entry should be expired and not found")
	assert.False(t, fresh)
	assert.Nil(t, val)

	// GetStale should also not find it
	staleVal, staleFound := cache.GetStale(key)
	assert.False(t, staleFound, "Expired entry should not be found even with GetStale")
	assert.Nil(t, staleVal)
}

func TestBigCache_Delete(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	require.NoError(t, err)
	defer func() {
		if bc, ok := cache.(*BigCache); ok {
			bc.Close()
		}
	}()

	key := "test-key"
	value := []byte("test-value")
	ttl := models.TTL{
		Fresh: 5 * time.Second,
		Stale: 10 * time.Second,
	}

	// Set and verify
	cache.Set(key, value, ttl)
	val, fresh, found := cache.Get(key)
	assert.True(t, found)
	assert.True(t, fresh)
	assert.Equal(t, value, val)

	// Delete and verify
	cache.Delete(key)
	val, fresh, found = cache.Get(key)
	assert.False(t, found)
	assert.False(t, fresh)
	assert.Nil(t, val)
}

func TestBigCache_DeleteNonExistent(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	require.NoError(t, err)
	defer func() {
		if bc, ok := cache.(*BigCache); ok {
			bc.Close()
		}
	}()

	// Should not panic or error
	cache.Delete("non-existent-key")
}

func TestBigCache_MultipleEntries(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	require.NoError(t, err)
	defer func() {
		if bc, ok := cache.(*BigCache); ok {
			bc.Close()
		}
	}()

	ttl := models.TTL{
		Fresh: 5 * time.Second,
		Stale: 10 * time.Second,
	}

	// Set multiple entries
	entries := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	for key, value := range entries {
		cache.Set(key, value, ttl)
	}

	// Verify all entries
	for key, expectedValue := range entries {
		val, fresh, found := cache.Get(key)
		assert.True(t, found, "Key %s should be found", key)
		assert.True(t, fresh, "Key %s should be fresh", key)
		assert.Equal(t, expectedValue, val, "Key %s should have correct value", key)
	}
}

func TestBigCache_MediumValue(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	require.NoError(t, err)
	defer func() {
		if bc, ok := cache.(*BigCache); ok {
			bc.Close()
		}
	}()

	key := "medium-key"
	// Create a medium-sized value that BigCache can handle
	mediumValue := make([]byte, 1024) // 1KB
	for i := range mediumValue {
		mediumValue[i] = byte(i % 256)
	}

	ttl := models.TTL{
		Fresh: 5 * time.Second,
		Stale: 10 * time.Second,
	}

	cache.Set(key, mediumValue, ttl)

	val, fresh, found := cache.Get(key)
	assert.True(t, found)
	assert.True(t, fresh)
	assert.Equal(t, mediumValue, val)
}

func TestBigCache_ZeroTTL(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	require.NoError(t, err)
	defer func() {
		if bc, ok := cache.(*BigCache); ok {
			bc.Close()
		}
	}()

	key := "zero-ttl-key"
	value := []byte("zero-ttl-value")
	ttl := models.TTL{
		Fresh: 0,
		Stale: 0,
	}

	cache.Set(key, value, ttl)

	// With zero TTL, the entry is still fresh at the exact same timestamp
	// but becomes stale/expired very quickly
	val, fresh, found := cache.Get(key)

	// The entry might be found and fresh at the exact same timestamp
	// or might be expired if there's any time difference
	if found {
		assert.Equal(t, value, val)
		// Could be fresh or stale depending on timing
	} else {
		// Entry was expired immediately
		assert.False(t, fresh)
		assert.Nil(t, val)
	}
}

func TestBigCache_CorruptedEntry(t *testing.T) {
	logger := zap.NewNop()
	bigCache, err := NewBigCache(10, logger)
	require.NoError(t, err)
	defer func() {
		if bc, ok := bigCache.(*BigCache); ok {
			bc.Close()
		}
	}()

	bc := bigCache.(*BigCache)
	key := "corrupted-key"

	// Manually set corrupted data
	corruptedData := []byte("not-json")
	err = bc.cache.Set(key, corruptedData)
	require.NoError(t, err)

	// Should handle corrupted entry gracefully
	val, fresh, found := bc.Get(key)
	assert.False(t, found)
	assert.False(t, fresh)
	assert.Nil(t, val)

	// Entry should be removed after corruption detection
	val, fresh, found = bc.Get(key)
	assert.False(t, found)
}

func TestBigCache_CorruptedEntryGetStale(t *testing.T) {
	logger := zap.NewNop()
	bigCache, err := NewBigCache(10, logger)
	require.NoError(t, err)
	defer func() {
		if bc, ok := bigCache.(*BigCache); ok {
			bc.Close()
		}
	}()

	bc := bigCache.(*BigCache)
	key := "corrupted-key-stale"

	// Manually set corrupted data
	corruptedData := []byte("not-json")
	err = bc.cache.Set(key, corruptedData)
	require.NoError(t, err)

	// Should handle corrupted entry gracefully in GetStale
	val, found := bc.GetStale(key)
	assert.False(t, found)
	assert.Nil(t, val)
}

func TestCacheEntry_Serialization(t *testing.T) {
	now := time.Now().Unix()
	entry := CacheEntry{
		Data:      []byte("test-data"),
		ExpiresAt: now + 100,
		StaleAt:   now + 50,
		CreatedAt: now,
	}

	// Test marshaling
	data, err := json.Marshal(entry)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test unmarshaling
	var unmarshaled CacheEntry
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, entry, unmarshaled)
}

func TestBigCache_Close(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	require.NoError(t, err)

	// Should close without error
	bc, ok := cache.(*BigCache)
	require.True(t, ok)
	err = bc.Close()
	assert.NoError(t, err)
}

func TestBigCache_ConcurrentAccess(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	require.NoError(t, err)
	defer func() {
		if bc, ok := cache.(*BigCache); ok {
			bc.Close()
		}
	}()

	ttl := models.TTL{
		Fresh: 5 * time.Second,
		Stale: 10 * time.Second,
	}

	// Test concurrent writes and reads
	done := make(chan bool, 10)

	// Start multiple goroutines writing
	for i := 0; i < 5; i++ {
		go func(id int) {
			key := "concurrent-key-" + string(rune(id+'0'))
			value := []byte("concurrent-value-" + string(rune(id+'0')))
			cache.Set(key, value, ttl)
			done <- true
		}(i)
	}

	// Start multiple goroutines reading
	for i := 0; i < 5; i++ {
		go func(id int) {
			key := "concurrent-key-" + string(rune(id+'0'))
			cache.Get(key) // Don't care about result, just testing for race conditions
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestBigCache_EdgeCases(t *testing.T) {
	logger := zap.NewNop()
	cache, err := NewBigCache(10, logger)
	require.NoError(t, err)
	defer func() {
		if bc, ok := cache.(*BigCache); ok {
			bc.Close()
		}
	}()

	ttl := models.TTL{
		Fresh: 1 * time.Second,
		Stale: 1 * time.Second,
	}

	t.Run("empty key", func(t *testing.T) {
		cache.Set("", []byte("empty-key-value"), ttl)
		val, fresh, found := cache.Get("")
		assert.True(t, found)
		assert.True(t, fresh)
		assert.Equal(t, []byte("empty-key-value"), val)
	})

	t.Run("empty value", func(t *testing.T) {
		cache.Set("empty-value-key", []byte{}, ttl)
		val, fresh, found := cache.Get("empty-value-key")
		assert.True(t, found)
		assert.True(t, fresh)
		assert.Equal(t, []byte{}, val)
	})

	t.Run("nil value", func(t *testing.T) {
		cache.Set("nil-value-key", nil, ttl)
		val, fresh, found := cache.Get("nil-value-key")
		assert.True(t, found)
		assert.True(t, fresh)
		assert.Nil(t, val)
	})
}
