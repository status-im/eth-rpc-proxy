package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// MetricsServer represents a separate HTTP server for metrics
type MetricsServer struct {
	logger *zap.Logger
	server *http.Server
}

// NewMetricsServer creates a new metrics HTTP server
func NewMetricsServer(logger *zap.Logger) *MetricsServer {
	return &MetricsServer{
		logger: logger,
	}
}

// Start starts the metrics HTTP server on the specified port
func (ms *MetricsServer) Start(port string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	ms.server = &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ms.logger.Info("Starting metrics HTTP server", zap.String("port", port))
	return ms.server.ListenAndServe()
}

// Stop stops the metrics HTTP server
func (ms *MetricsServer) Stop(ctx context.Context) error {
	if ms.server == nil {
		return nil
	}

	ms.logger.Info("Stopping metrics HTTP server")
	return ms.server.Shutdown(ctx)
}
