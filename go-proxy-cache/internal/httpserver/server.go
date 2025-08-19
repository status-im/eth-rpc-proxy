package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"go-proxy-cache/internal/cache/service"
)

// Server represents the HTTP cache server
type Server struct {
	cacheService *service.CacheService
	logger       *zap.Logger
	server       *http.Server
}

// NewServer creates a new cache HTTP server
func NewServer(cacheService *service.CacheService, logger *zap.Logger) *Server {
	return &Server{
		cacheService: cacheService,
		logger:       logger,
	}
}

// StartUnixSocket starts the HTTP server on a Unix socket
func (s *Server) StartUnixSocket(socketPath string) error {
	// Remove existing socket file
	if err := os.RemoveAll(socketPath); err != nil {
		s.logger.Warn("Failed to remove existing socket file", zap.String("path", socketPath), zap.Error(err))
	}

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}

	// Set socket permissions (readable/writable by owner and group)
	if err := os.Chmod(socketPath, 0660); err != nil {
		s.logger.Warn("Failed to set socket permissions", zap.String("path", socketPath), zap.Error(err))
	}

	router := s.createRouter()

	s.server = &http.Server{
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting cache HTTP server on Unix socket", zap.String("socket_path", socketPath))
	return s.server.Serve(listener)
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping cache HTTP server")
	return s.server.Shutdown(ctx)
}

// createRouter creates and configures the HTTP router
func (s *Server) createRouter() *mux.Router {
	router := mux.NewRouter()

	// Cache endpoints
	router.HandleFunc("/cache/get", s.handleGet).Methods("POST")
	router.HandleFunc("/cache/set", s.handleSet).Methods("POST")

	// Health check
	router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Cache info endpoint (equivalent to cache rules check)
	router.HandleFunc("/cache/info", s.handleCacheInfo).Methods("POST")

	// Prometheus metrics endpoint
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	return router
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeResponse(w, map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().UTC(),
	})
}

// parseRequest parses JSON request body
func (s *Server) parseRequest(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.Unmarshal(body, v)
}

// writeResponse writes JSON response
func (s *Server) writeResponse(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.logger.Error("Failed to write response", zap.Error(err))
	}
}

// writeErrorResponse writes error response
func (s *Server) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := map[string]interface{}{
		"success": false,
		"error":   message,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to write error response", zap.Error(err))
	}
}
