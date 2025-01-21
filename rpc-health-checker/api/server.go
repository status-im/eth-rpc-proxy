package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// ServerConfig contains configuration for the HTTP server
type ServerConfig struct {
	Port          string
	ProvidersPath string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	IdleTimeout   time.Duration
}

// Provider represents a configuration provider
type Provider struct {
	URL        string `json:"url"`
	AuthHeader string `json:"auth_header"`
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
	providers     []Provider
	lastReload    time.Time
	reloadTimeout time.Duration
}

func New(port, providersPath string) Server {
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	s := &httpServer{
		config: ServerConfig{
			Port:          port,
			ProvidersPath: providersPath,
			ReadTimeout:   5 * time.Second,
			WriteTimeout:  10 * time.Second,
			IdleTimeout:   15 * time.Second,
		},
		server: srv,
		logger: slog.Default(),
	}

	mux.HandleFunc("/providers", s.providersHandler)
	mux.HandleFunc("/health", s.healthHandler)

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

func (s *httpServer) providersHandler(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open(s.config.ProvidersPath)
	if err != nil {
		s.logger.Error("failed to open providers file", "error", err)
		http.Error(w, "failed to open providers file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/json")
	if _, err := io.Copy(w, f); err != nil {
		s.logger.Error("failed to read providers file", "error", err)
		http.Error(w, "failed to read providers file", http.StatusInternalServerError)
		return
	}
}

func (s *httpServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
