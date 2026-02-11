package main

import (
	"time"

	"github.com/status-im/proxy-common/cache"
	cachemetrics "github.com/status-im/proxy-common/cache/metrics"
	"go.uber.org/zap"
)

// ZapLogger adapts zap.Logger to cache.Logger interface
type ZapLogger struct {
	logger *zap.Logger
}

// NewZapLogger creates a new ZapLogger adapter
func NewZapLogger(logger *zap.Logger) cache.Logger {
	return &ZapLogger{logger: logger}
}

// Debug logs a debug message
func (z *ZapLogger) Debug(msg string, keysAndValues ...interface{}) {
	z.logger.Debug(msg, toZapFields(keysAndValues)...)
}

// Info logs an info message
func (z *ZapLogger) Info(msg string, keysAndValues ...interface{}) {
	z.logger.Info(msg, toZapFields(keysAndValues)...)
}

// Warn logs a warning message
func (z *ZapLogger) Warn(msg string, keysAndValues ...interface{}) {
	z.logger.Warn(msg, toZapFields(keysAndValues)...)
}

// Error logs an error message
func (z *ZapLogger) Error(msg string, keysAndValues ...interface{}) {
	z.logger.Error(msg, toZapFields(keysAndValues)...)
}

// toZapFields converts key-value pairs to zap fields
func toZapFields(keysAndValues []interface{}) []zap.Field {
	fields := make([]zap.Field, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		fields = append(fields, zap.Any(key, keysAndValues[i+1]))
	}
	return fields
}

// PrometheusMetrics adapts the cache metrics to cache.MetricsRecorder interface
type PrometheusMetrics struct {
	metrics *cachemetrics.CacheMetrics
}

// NewPrometheusMetrics creates a new PrometheusMetrics adapter
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		metrics: cachemetrics.New(cachemetrics.Config{
			Namespace: "eth_rpc_proxy",
			Subsystem: "cache",
		}),
	}
}

// RecordCacheError records a cache error with level and kind
func (p *PrometheusMetrics) RecordCacheError(level, kind string) {
	p.metrics.RecordCacheError(level, kind)
}

// UpdateL1CacheCapacity updates L1 cache capacity metrics
func (p *PrometheusMetrics) UpdateL1CacheCapacity(capacity, used int64) {
	p.metrics.UpdateL1CacheCapacity(capacity, used)
}

// UpdateCacheKeys updates the number of keys in cache
func (p *PrometheusMetrics) UpdateCacheKeys(level string, count int64) {
	p.metrics.UpdateCacheKeys(level, count)
}

// RecordCacheHit records a cache hit with enhanced labels and age tracking
func (p *PrometheusMetrics) RecordCacheHit(cacheType, level, chain, network, rpcMethod string, itemAge time.Duration) {
	p.metrics.RecordCacheHit(cacheType, level, chain, network, rpcMethod, itemAge)
}

// RecordCacheMiss records a cache miss with enhanced labels
func (p *PrometheusMetrics) RecordCacheMiss(cacheType, chain, network, rpcMethod string) {
	p.metrics.RecordCacheMiss(cacheType, chain, network, rpcMethod)
}

// RecordCacheSet records a cache set operation with size tracking
func (p *PrometheusMetrics) RecordCacheSet(level, cacheType, chain, network string, dataSize int) {
	p.metrics.RecordCacheSet(level, cacheType, chain, network, dataSize)
}

// RecordCacheBytesRead records bytes read from cache
func (p *PrometheusMetrics) RecordCacheBytesRead(level, cacheType, chain, network string, bytesRead int) {
	p.metrics.RecordCacheBytesRead(level, cacheType, chain, network, bytesRead)
}

// TimeCacheOperation returns a timer function for measuring cache operation duration
func (p *PrometheusMetrics) TimeCacheOperation(operation, level string) func() {
	return p.metrics.TimeCacheOperation(operation, level)
}

// InitializeAllowedMethods initializes the allowed methods whitelist from cache rules
func (p *PrometheusMetrics) InitializeAllowedMethods(methods []string) {
	p.metrics.InitializeAllowedMethods(methods)
}
