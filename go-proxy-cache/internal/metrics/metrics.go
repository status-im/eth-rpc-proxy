package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Core request/hit/miss counters
	CacheRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_requests_total",
			Help: "Total number of cache requests",
		},
		[]string{"cache_type"},
	)

	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type"},
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type"},
	)

	// L1/L2 specific hits (separate counters for simplicity)
	L1CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "l1_cache_hits_total",
			Help: "Total number of L1 cache hits",
		},
		[]string{"cache_type"},
	)

	L2CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "l2_cache_hits_total",
			Help: "Total number of L2 cache hits",
		},
		[]string{"cache_type"},
	)

	// Get operation latency only
	CacheOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_operation_duration_seconds",
			Help:    "Duration of cache get operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "level"}, // simplified labels
	)

	// L1 capacity metrics only (if L1 is in-memory)
	CacheCapacity = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_capacity_bytes",
			Help: "L1 cache capacity in bytes",
		},
		[]string{"level"}, // only "l1"
	)

	CacheUsed = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_used_bytes",
			Help: "L1 cache used space in bytes",
		},
		[]string{"level"}, // only "l1"
	)
)

// RecordCacheRequest records a cache request
func RecordCacheRequest(cacheType string) {
	CacheRequests.WithLabelValues(cacheType).Inc()
}

// RecordCacheHit records a cache hit
func RecordCacheHit(cacheType string, level string) {
	CacheHits.WithLabelValues(cacheType).Inc()

	switch level {
	case "l1":
		L1CacheHits.WithLabelValues(cacheType).Inc()
	case "l2":
		L2CacheHits.WithLabelValues(cacheType).Inc()
	}
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss(cacheType string) {
	CacheMisses.WithLabelValues(cacheType).Inc()
}

// UpdateL1CacheCapacity updates L1 cache capacity metrics only
func UpdateL1CacheCapacity(capacity, used int64) {
	CacheCapacity.WithLabelValues("l1").Set(float64(capacity))
	CacheUsed.WithLabelValues("l1").Set(float64(used))
}

// TimeCacheGetOperation returns a timer function for measuring cache get operation duration
func TimeCacheGetOperation(level string) func() {
	timer := prometheus.NewTimer(CacheOperationDuration.WithLabelValues("get", level))
	return func() {
		timer.ObserveDuration()
	}
}
