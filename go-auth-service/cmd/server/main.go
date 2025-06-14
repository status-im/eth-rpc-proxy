package main

import (
	"log"
	"net/http"
	"os"

	handlers "go-auth-service/internal/handlers"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/auth/puzzle", handlers.PuzzleHandler)
	mux.HandleFunc("/auth/solve", handlers.SolveHandler)
	mux.HandleFunc("/auth/status", handlers.StatusHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("[go-auth-service] starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
