package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ServerConfig contains configuration for the HTTP server
type ServerConfig struct {
	Port                 string
	ProvidersPath        string
	DefaultProvidersPath string
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
}

// Server defines the interface for the configuration HTTP server
type Server interface {
	Start() error
	Stop() error
}

type httpServer struct {
	config        ServerConfig
	server        *http.Server
	logger        *slog.Logger
	lastReload    time.Time
	reloadTimeout time.Duration
}

func New(port, providersPath, defaultProvidersPath string) Server {
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	s := &httpServer{
		config: ServerConfig{
			Port:                 port,
			ProvidersPath:        providersPath,
			DefaultProvidersPath: defaultProvidersPath,
			ReadTimeout:          5 * time.Second,
			WriteTimeout:         10 * time.Second,
			IdleTimeout:          15 * time.Second,
		},
		server: srv,
		logger: slog.Default(),
	}

	mux.HandleFunc("/providers", s.providersHandler)
	mux.HandleFunc("/health", s.healthHandler)
	mux.Handle("/metrics", promhttp.Handler())

	return s
}

func (s *httpServer) Start() error {
	s.logger.Info("starting config HTTP server", "port", s.config.Port)
	s.server.ReadTimeout = s.config.ReadTimeout
	s.server.WriteTimeout = s.config.WriteTimeout
	s.server.IdleTimeout = s.config.IdleTimeout
	return s.server.ListenAndServe()
}

func (s *httpServer) Stop() error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(context.Background())
}

func (s *httpServer) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *httpServer) loadProviders(w io.Writer, path string) error {
	if !s.fileExists(path) {
		return os.ErrNotExist
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

func (s *httpServer) providersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Only try to load output providers if the file exists
	if s.fileExists(s.config.ProvidersPath) {
		err := s.loadProviders(w, s.config.ProvidersPath)
		if err == nil {
			return
		}
		s.logger.Info("failed to load output providers, falling back to default", "error", err)

		// Return HTTP error for file permission or corruption issues
		if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, io.EOF) {
			http.Error(w, fmt.Sprintf("Error reading output providers: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// If output providers don't exist or failed to load, use default providers
	if s.fileExists(s.config.DefaultProvidersPath) {
		err := s.loadProviders(w, s.config.DefaultProvidersPath)
		if err == nil {
			return
		}
		s.logger.Error("failed to load default providers", "error", err)

		// Return HTTP error for file permission or corruption issues
		if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, io.EOF) {
			http.Error(w, fmt.Sprintf("Error reading default providers: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// If no providers are available, return 404 Not Found
	http.Error(w, `{"error":"No providers found"}`, http.StatusNotFound)
}

func (s *httpServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
