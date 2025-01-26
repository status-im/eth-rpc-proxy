package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/status-im/eth-rpc-proxy/config"
)

var (
	validationCycleDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "validation_cycle_duration_seconds",
			Help:    "Duration of complete validation cycles in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	workingProvidersCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "working_providers_count",
			Help: "Number of working providers per chain",
		},
		[]string{"chainName", "networkName"},
	)

	nonWorkingProviders = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "non_working_providers",
			Help: "Status of non-working providers (1 if provider is not working, 0 if working)",
		},
		[]string{"chainName", "networkName", "providerName", "providerUrl"},
	)
)

func init() {
	prometheus.MustRegister(validationCycleDuration)
	prometheus.MustRegister(workingProvidersCount)
	prometheus.MustRegister(nonWorkingProviders)
}

// RecordValidationCycleDuration records the duration of a complete validation cycle
func RecordValidationCycleDuration(duration time.Duration) {
	validationCycleDuration.Observe(duration.Seconds())
}

// RecordWorkingProviders records the number of working providers for each chain
func RecordWorkingProviders(validChains []config.ChainConfig) {
	for _, chain := range validChains {
		workingProvidersCount.With(prometheus.Labels{
			"chainName":   chain.Name,
			"networkName": chain.Network,
		}).Set(float64(len(chain.Providers)))
	}
}

// RecordNonWorkingProviders records the status of non-working providers
func RecordNonWorkingProviders(chainName, networkName string, providerResults map[string]struct {
	Valid bool
	URL   string
}) {
	for providerName, result := range providerResults {
		value := 0.0
		if !result.Valid {
			value = 1.0
		}
		nonWorkingProviders.With(prometheus.Labels{
			"chainName":    chainName,
			"networkName":  networkName,
			"providerName": providerName,
			"providerUrl":  result.URL,
		}).Set(value)
	}
}
