package main

import (
	"log"
	"net/http"
	"os"

	"go-auth-service/internal/config"
	handlers "go-auth-service/internal/handlers"
)

func main() {
	// Load configuration from JSON file
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Create handlers instance with configuration
	h := handlers.New(cfg)

	mux := http.NewServeMux()

	mux.HandleFunc("/auth/puzzle", h.PuzzleHandler)
	mux.HandleFunc("/auth/solve", h.SolveHandler)
	mux.HandleFunc("/auth/verify", h.VerifyHandler)
	mux.HandleFunc("/auth/status", h.StatusHandler)

	// Get port from environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("[go-auth-service] starting on :%s", port)
	log.Printf("[go-auth-service] algorithm: %s, difficulty: %d, token expiry: %d minutes",
		cfg.Algorithm, cfg.PuzzleDifficulty, cfg.TokenExpiryMinutes)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
