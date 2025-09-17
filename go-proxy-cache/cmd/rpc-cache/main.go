package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	// Initialize composition root with all dependencies
	root, err := NewCompositionRoot()
	if err != nil {
		fmt.Printf("Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// Ensure cleanup on exit
	defer func() {
		if err := root.Cleanup(); err != nil {
			root.Logger.Error("Failed to cleanup resources", zap.Error(err))
		}
	}()

	// Get socket path
	socketPath := root.GetSocketPath()

	// Start server on Unix socket
	root.Logger.Info("Starting cache server on Unix socket", zap.String("socket_path", socketPath))
	go func() {
		if err := root.HTTPServer.StartUnixSocket(socketPath); err != nil {
			root.Logger.Error("Server failed to start on Unix socket", zap.Error(err))
		}
	}()

	// Start metrics server on HTTP port
	metricsPort := root.GetMetricsPort()
	root.Logger.Info("Starting metrics server", zap.String("port", metricsPort))
	go func() {
		if err := root.MetricsServer.Start(metricsPort); err != nil {
			root.Logger.Error("Metrics server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	root.Logger.Info("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown servers
	if err := root.HTTPServer.Stop(ctx); err != nil {
		root.Logger.Error("HTTP server forced to shutdown", zap.Error(err))
	}
	if err := root.MetricsServer.Stop(ctx); err != nil {
		root.Logger.Error("Metrics server forced to shutdown", zap.Error(err))
	}

	root.Logger.Info("Server exited")
}
