package main

import (
	"github.com/status-im/proxy-common/cache"
	"go.uber.org/zap"

	"go-proxy-cache/internal/metrics"
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

// PrometheusMetrics adapts the eth-rpc-proxy metrics package to cache.MetricsRecorder interface
type PrometheusMetrics struct{}

// NewPrometheusMetrics creates a new PrometheusMetrics adapter
func NewPrometheusMetrics() cache.MetricsRecorder {
	return &PrometheusMetrics{}
}

// RecordCacheError records a cache error with level and kind
func (p *PrometheusMetrics) RecordCacheError(level, kind string) {
	metrics.RecordCacheError(level, kind)
}

// UpdateL1CacheCapacity updates L1 cache capacity metrics
func (p *PrometheusMetrics) UpdateL1CacheCapacity(capacity, used int64) {
	metrics.UpdateL1CacheCapacity(capacity, used)
}

// UpdateCacheKeys updates the number of keys in cache
func (p *PrometheusMetrics) UpdateCacheKeys(level string, count int64) {
	metrics.UpdateCacheKeys(level, count)
}
