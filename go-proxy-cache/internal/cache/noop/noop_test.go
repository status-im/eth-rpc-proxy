package noop

import (
	"testing"
	"time"

	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/models"
)

func TestNewNoOpCache(t *testing.T) {
	cache := NewNoOpCache()

	// Verify it implements the Cache interface
	var _ interfaces.Cache = cache

	// Verify it returns a NoOpCache instance
	if _, ok := cache.(*NoOpCache); !ok {
		t.Errorf("NewNoOpCache() should return a *NoOpCache instance")
	}
}

func TestNoOpCache_Get(t *testing.T) {
	cache := NewNoOpCache()

	// Test with various keys
	testCases := []string{
		"test-key",
		"",
		"very-long-key-with-special-characters-!@#$%^&*()",
		"key-with-numbers-123456789",
	}

	for _, key := range testCases {
		t.Run("key="+key, func(t *testing.T) {
			val, fresh, found := cache.Get(key)

			if val != nil {
				t.Errorf("Get(%q) val = %v, want nil", key, val)
			}
			if fresh {
				t.Errorf("Get(%q) fresh = %v, want false", key, fresh)
			}
			if found {
				t.Errorf("Get(%q) found = %v, want false", key, found)
			}
		})
	}
}

func TestNoOpCache_GetStale(t *testing.T) {
	cache := NewNoOpCache()

	// Test with various keys
	testCases := []string{
		"test-key",
		"",
		"very-long-key-with-special-characters-!@#$%^&*()",
		"key-with-numbers-123456789",
	}

	for _, key := range testCases {
		t.Run("key="+key, func(t *testing.T) {
			val, found := cache.GetStale(key)

			if val != nil {
				t.Errorf("GetStale(%q) val = %v, want nil", key, val)
			}
			if found {
				t.Errorf("GetStale(%q) found = %v, want false", key, found)
			}
		})
	}
}

func TestNoOpCache_Set(t *testing.T) {
	cache := NewNoOpCache()

	// Test setting various values
	testCases := []struct {
		key string
		val []byte
		ttl models.TTL
	}{
		{"test-key", []byte("test-value"), models.TTL{Fresh: 60 * time.Second, Stale: 120 * time.Second}},
		{"", []byte(""), models.TTL{Fresh: 0, Stale: 0}},
		{"binary-key", []byte{0x01, 0x02, 0x03, 0xFF}, models.TTL{Fresh: 3600 * time.Second, Stale: 7200 * time.Second}},
		{"json-key", []byte(`{"test": "value"}`), models.TTL{Fresh: 300 * time.Second, Stale: 600 * time.Second}},
	}

	for _, tc := range testCases {
		t.Run("key="+tc.key, func(t *testing.T) {
			// Set should not panic and should be a no-op
			cache.Set(tc.key, tc.val, tc.ttl)

			// Verify it's still a cache miss after setting
			val, fresh, found := cache.Get(tc.key)
			if val != nil || fresh || found {
				t.Errorf("After Set(%q, %v, %v), Get() = (%v, %v, %v), want (nil, false, false)",
					tc.key, tc.val, tc.ttl, val, fresh, found)
			}
		})
	}
}

func TestNoOpCache_Delete(t *testing.T) {
	cache := NewNoOpCache()

	// Test deleting various keys
	testCases := []string{
		"test-key",
		"",
		"non-existent-key",
		"very-long-key-with-special-characters-!@#$%^&*()",
	}

	for _, key := range testCases {
		t.Run("key="+key, func(t *testing.T) {
			// Delete should not panic and should be a no-op
			cache.Delete(key)

			// Verify it's still a cache miss after deleting
			val, fresh, found := cache.Get(key)
			if val != nil || fresh || found {
				t.Errorf("After Delete(%q), Get() = (%v, %v, %v), want (nil, false, false)",
					key, val, fresh, found)
			}
		})
	}
}

func TestNoOpCache_InterfaceCompliance(t *testing.T) {
	cache := NewNoOpCache()

	// Verify all interface methods work as expected
	key := "test-key"
	value := []byte("test-value")
	ttl := models.TTL{Fresh: 60 * time.Second, Stale: 120 * time.Second}

	// Test the complete workflow
	cache.Set(key, value, ttl)

	val, fresh, found := cache.Get(key)
	if val != nil || fresh || found {
		t.Errorf("Get() after Set() = (%v, %v, %v), want (nil, false, false)", val, fresh, found)
	}

	val, found = cache.GetStale(key)
	if val != nil || found {
		t.Errorf("GetStale() after Set() = (%v, %v), want (nil, false)", val, found)
	}

	cache.Delete(key)

	val, fresh, found = cache.Get(key)
	if val != nil || fresh || found {
		t.Errorf("Get() after Delete() = (%v, %v, %v), want (nil, false, false)", val, fresh, found)
	}
}

func TestNoOpCache_ConcurrentAccess(t *testing.T) {
	cache := NewNoOpCache()

	// Test concurrent access to ensure no race conditions
	done := make(chan bool)

	// Start multiple goroutines performing operations
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			key := "concurrent-key"
			value := []byte("concurrent-value")
			ttl := models.TTL{Fresh: 60 * time.Second, Stale: 120 * time.Second}

			// Perform various operations
			cache.Set(key, value, ttl)
			cache.Get(key)
			cache.GetStale(key)
			cache.Delete(key)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify cache is still in expected state
	val, fresh, found := cache.Get("concurrent-key")
	if val != nil || fresh || found {
		t.Errorf("After concurrent operations, Get() = (%v, %v, %v), want (nil, false, false)", val, fresh, found)
	}
}
