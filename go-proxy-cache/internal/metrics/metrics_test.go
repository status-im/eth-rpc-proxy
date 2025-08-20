package metrics

import (
	"testing"
)

func TestCacheMetrics(t *testing.T) {
	// Note: Metrics are now package-level variables, automatically registered
	// This test just verifies the functions don't panic

	t.Run("RecordCacheRequest", func(t *testing.T) {
		// This should not panic
		RecordCacheRequest("permanent")
	})

	t.Run("RecordCacheHit", func(t *testing.T) {
		// This should not panic
		RecordCacheHit("permanent", "l1")
		RecordCacheHit("permanent", "l2")
	})

	t.Run("RecordCacheMiss", func(t *testing.T) {
		// This should not panic
		RecordCacheMiss("permanent")
	})

	t.Run("UpdateL1CacheCapacity", func(t *testing.T) {
		// This should not panic
		UpdateL1CacheCapacity(1000000, 500000)
	})

	t.Run("TimeCacheGetOperation", func(t *testing.T) {
		// This should not panic
		timer := TimeCacheGetOperation("l1")
		timer() // Call the returned function
	})
}
