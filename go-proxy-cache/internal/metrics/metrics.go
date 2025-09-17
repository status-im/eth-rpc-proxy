package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// White list of allowed RPC methods to prevent cardinality explosion
// Initialized from cache_rules.yaml via InitializeAllowedMethods()
var allowedMethods map[string]bool

var (
	// Core request/hit/miss counters with unified labels
	CacheRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eth_rpc_proxy_cache_requests_total",
			Help: "Total number of cache requests",
		},
		[]string{"cache_type", "level", "network", "rpc_method"},
	)

	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eth_rpc_proxy_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type", "level", "network", "rpc_method"},
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eth_rpc_proxy_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type", "level", "network", "rpc_method"},
	)

	// New metrics for enhanced monitoring
	CacheSets = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eth_rpc_proxy_cache_sets_total",
			Help: "Total number of cache set operations",
		},
		[]string{"level", "cache_type", "network"},
	)

	CacheEvictions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eth_rpc_proxy_cache_evictions_total",
			Help: "Total number of cache evictions",
		},
		[]string{"level", "cache_type", "network"},
	)

	CacheErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eth_rpc_proxy_cache_errors_total",
			Help: "Cache errors by kind",
		},
		[]string{"level", "kind"},
	)

	CacheBytesRead = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eth_rpc_proxy_cache_bytes_read_total",
			Help: "Bytes read from cache",
		},
		[]string{"level", "cache_type", "network"},
	)

	CacheBytesWritten = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "eth_rpc_proxy_cache_bytes_written_total",
			Help: "Bytes written to cache",
		},
		[]string{"level", "cache_type", "network"},
	)

	// Extended operation latency for get and set operations
	CacheOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "eth_rpc_proxy_cache_operation_duration_seconds",
			Help:    "Duration of cache operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "level"}, // operation: get|set, level: l1|l2|multi
	)

	// Cache keys count (mainly for L1)
	CacheKeys = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "eth_rpc_proxy_cache_keys",
			Help: "Current number of keys in cache",
		},
		[]string{"level"},
	)

	// Cache item age at hit time (for TTL effectiveness analysis)
	CacheItemAge = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "eth_rpc_proxy_cache_item_age_seconds",
			Help:    "Age of item at hit time",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300, 600, 1800, 3600}, // up to 1 hour
		},
		[]string{"level", "cache_type"},
	)

	// L1 capacity metrics only (if L1 is in-memory)
	CacheCapacity = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "eth_rpc_proxy_cache_capacity_bytes",
			Help: "L1 cache capacity in bytes",
		},
		[]string{"level"}, // only "l1"
	)

	CacheUsed = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "eth_rpc_proxy_cache_used_bytes",
			Help: "L1 cache used space in bytes",
		},
		[]string{"level"}, // only "l1"
	)
)

// InitializeAllowedMethods initializes the allowed methods whitelist from cache rules
func InitializeAllowedMethods(methods []string) {
	allowedMethods = make(map[string]bool)

	// Add all configured methods to whitelist
	for _, method := range methods {
		allowedMethods[method] = true
	}
}

// normalizeRPCMethod returns the method name if it's in the whitelist, otherwise "other"
func normalizeRPCMethod(method string) string {
	if allowedMethods != nil && allowedMethods[method] {
		return method
	}
	return "other"
}

// normalizeNetwork creates a network identifier from chain and network
func normalizeNetwork(chain, network string) string {
	if chain == "" || network == "" {
		return "unknown"
	}
	return chain + ":" + network
}

// RecordCacheHit records a cache hit with enhanced labels and age tracking
func RecordCacheHit(cacheType, level, chain, network, rpcMethod string, itemAge time.Duration) {
	normalizedNetwork := normalizeNetwork(chain, network)
	normalizedMethod := normalizeRPCMethod(rpcMethod)

	// Record request and hit with proper level
	CacheRequests.WithLabelValues(cacheType, level, normalizedNetwork, normalizedMethod).Inc()
	CacheHits.WithLabelValues(cacheType, level, normalizedNetwork, normalizedMethod).Inc()

	// Record item age for TTL effectiveness analysis
	if itemAge > 0 {
		CacheItemAge.WithLabelValues(level, cacheType).Observe(itemAge.Seconds())
	}
}

// RecordCacheMiss records a cache miss with enhanced labels
func RecordCacheMiss(cacheType, chain, network, rpcMethod string) {
	normalizedNetwork := normalizeNetwork(chain, network)
	normalizedMethod := normalizeRPCMethod(rpcMethod)

	// For miss, we don't know the level, so we use "miss" as level
	// This represents requests that didn't hit any cache level
	CacheRequests.WithLabelValues(cacheType, "miss", normalizedNetwork, normalizedMethod).Inc()
	CacheMisses.WithLabelValues(cacheType, "miss", normalizedNetwork, normalizedMethod).Inc()
}

// RecordCacheSet records a cache set operation with size tracking
func RecordCacheSet(level, cacheType, chain, network string, dataSize int) {
	normalizedNetwork := normalizeNetwork(chain, network)

	CacheSets.WithLabelValues(level, cacheType, normalizedNetwork).Inc()
	if dataSize > 0 {
		CacheBytesWritten.WithLabelValues(level, cacheType, normalizedNetwork).Add(float64(dataSize))
	}
}

// RecordCacheEviction records a cache eviction
func RecordCacheEviction(level, cacheType, chain, network string) {
	normalizedNetwork := normalizeNetwork(chain, network)
	CacheEvictions.WithLabelValues(level, cacheType, normalizedNetwork).Inc()
}

// RecordCacheError records a cache error
func RecordCacheError(level, kind string) {
	CacheErrors.WithLabelValues(level, kind).Inc()
}

// RecordCacheBytesRead records bytes read from cache
func RecordCacheBytesRead(level, cacheType, chain, network string, bytesRead int) {
	if bytesRead > 0 {
		normalizedNetwork := normalizeNetwork(chain, network)
		CacheBytesRead.WithLabelValues(level, cacheType, normalizedNetwork).Add(float64(bytesRead))
	}
}

// UpdateL1CacheCapacity updates L1 cache capacity metrics
func UpdateL1CacheCapacity(capacity, used int64) {
	CacheCapacity.WithLabelValues("l1").Set(float64(capacity))
	CacheUsed.WithLabelValues("l1").Set(float64(used))
}

// UpdateCacheKeys updates the number of keys in cache
func UpdateCacheKeys(level string, count int64) {
	CacheKeys.WithLabelValues(level).Set(float64(count))
}

// TimeCacheOperation returns a timer function for measuring cache operation duration
func TimeCacheOperation(operation, level string) func() {
	timer := prometheus.NewTimer(CacheOperationDuration.WithLabelValues(operation, level))
	return func() {
		timer.ObserveDuration()
	}
}

// TimeCacheGetOperation returns a timer function for measuring cache get operation duration (backward compatibility)
func TimeCacheGetOperation(level string) func() {
	return TimeCacheOperation("get", level)
}
