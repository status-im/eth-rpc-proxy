package metrics

import (
	"testing"
	"time"
)

func TestCacheMetrics(t *testing.T) {
	// Note: Metrics are now package-level variables, automatically registered
	// This test just verifies the functions don't panic

	t.Run("RecordCacheHit", func(t *testing.T) {
		// This should not panic
		RecordCacheHit("permanent", "l1", "ethereum", "mainnet", "eth_getBlockByHash", time.Second*30)
		RecordCacheHit("permanent", "l2", "ethereum", "mainnet", "eth_getBlockByHash", time.Second*60)
	})

	t.Run("RecordCacheMiss", func(t *testing.T) {
		// This should not panic
		RecordCacheMiss("permanent", "ethereum", "mainnet", "eth_getBlockByHash")
	})

	t.Run("RecordCacheSet", func(t *testing.T) {
		// This should not panic
		RecordCacheSet("l1", "permanent", "ethereum", "mainnet", 1024)
	})

	t.Run("RecordCacheError", func(t *testing.T) {
		// This should not panic
		RecordCacheError("l1", "encode")
	})

	t.Run("UpdateL1CacheCapacity", func(t *testing.T) {
		// This should not panic
		UpdateL1CacheCapacity(1000000, 500000)
	})

	t.Run("UpdateCacheKeys", func(t *testing.T) {
		// This should not panic
		UpdateCacheKeys("l1", 1000)
	})

	t.Run("TimeCacheOperation", func(t *testing.T) {
		// This should not panic
		timer := TimeCacheOperation("get", "l1")
		timer() // Call the returned function
	})

	t.Run("TimeCacheGetOperation", func(t *testing.T) {
		// This should not panic (backward compatibility)
		timer := TimeCacheGetOperation("l1")
		timer() // Call the returned function
	})

	t.Run("NormalizeRPCMethod", func(t *testing.T) {
		// Initialize test methods
		testMethods := []string{"eth_getBlockByHash", "eth_call", "net_version"}
		InitializeAllowedMethods(testMethods)

		// Test whitelisted method
		if normalizeRPCMethod("eth_getBlockByHash") != "eth_getBlockByHash" {
			t.Error("Expected whitelisted method to be preserved")
		}

		// Test non-whitelisted method
		if normalizeRPCMethod("custom_method") != "other" {
			t.Error("Expected non-whitelisted method to be normalized to 'other'")
		}
	})

	t.Run("NormalizeNetwork", func(t *testing.T) {
		// Test valid network
		if normalizeNetwork("ethereum", "mainnet") != "ethereum:mainnet" {
			t.Error("Expected valid network to be formatted correctly")
		}

		// Test empty network
		if normalizeNetwork("", "") != "unknown" {
			t.Error("Expected empty network to be normalized to 'unknown'")
		}
	})
}
