package main

import (
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go-auth-service/internal/config"
	"go-auth-service/internal/handlers"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Create handlers
	h := handlers.New(cfg)

	// Create mux
	mux := http.NewServeMux()

	// Setup routes - HMAC protected endpoints only
	mux.HandleFunc("/auth/puzzle", h.PuzzleHandler)       // Get puzzle challenge with HMAC requirements
	mux.HandleFunc("/auth/solve", h.SolveHandler)         // Submit HMAC protected solution
	mux.HandleFunc("/dev/test-solve", h.TestSolveHandler) // Generate test solution with HMAC (dev only)
	mux.HandleFunc("/auth/verify", h.VerifyHandler)       // Verify JWT token
	mux.HandleFunc("/auth/status", h.StatusHandler)       // Service status

	// Add Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Get port from environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("[go-auth-service] starting on :%s", port)
	log.Printf("[go-auth-service] algorithm: %s, memory: %dKB, time: %d, token expiry: %d minutes",
		cfg.Algorithm, cfg.Argon2Params.MemoryKB, cfg.Argon2Params.Time, cfg.TokenExpiryMinutes)
	log.Printf("[go-auth-service] metrics available at /metrics")

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
