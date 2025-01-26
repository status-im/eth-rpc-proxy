package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	validationCycleDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "validation_cycle_duration_seconds",
			Help:    "Duration of complete validation cycles in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func init() {
	prometheus.MustRegister(validationCycleDuration)
}

// RecordValidationCycleDuration records the duration of a complete validation cycle
func RecordValidationCycleDuration(duration time.Duration) {
	validationCycleDuration.Observe(duration.Seconds())
}
