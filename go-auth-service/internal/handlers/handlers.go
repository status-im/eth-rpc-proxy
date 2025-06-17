package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-auth-service/internal/config"
	"go-auth-service/internal/jwt"
	"go-auth-service/internal/puzzle"
)

// Handlers struct holds configuration and state
type Handlers struct {
	config     *config.Config
	tokenUsage map[string]int
	tokenMutex sync.RWMutex
}

// New creates a new Handlers instance with the given configuration
func New(cfg *config.Config) *Handlers {
	return &Handlers{
		config:     cfg,
		tokenUsage: make(map[string]int),
	}
}

// configToArgon2Config converts config.Argon2Params to puzzle.Argon2Config
func (h *Handlers) configToArgon2Config() puzzle.Argon2Config {
	return puzzle.Argon2Config{
		MemoryKB: h.config.Argon2Params.MemoryKB,
		Time:     h.config.Argon2Params.Time,
		Threads:  h.config.Argon2Params.Threads,
		KeyLen:   h.config.Argon2Params.KeyLen,
	}
}

func (h *Handlers) PuzzleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	p, err := puzzle.Generate(h.config.PuzzleDifficulty, h.config.TokenExpiryMinutes)
	if err != nil {
		http.Error(w, "failed to generate puzzle", 500)
		return
	}

	// Add expected iterations info for client
	response := map[string]interface{}{
		"challenge":           p.Challenge,
		"salt":                p.Salt,
		"difficulty":          p.Difficulty,
		"expires_at":          p.ExpiresAt.Format(time.RFC3339),
		"expected_iterations": puzzle.GetExpectedIterations(p.Difficulty),
		"algorithm":           h.config.Algorithm,
		"argon2_params": map[string]interface{}{
			"memory_kb": h.config.Argon2Params.MemoryKB,
			"time":      h.config.Argon2Params.Time,
			"threads":   h.config.Argon2Params.Threads,
			"key_len":   h.config.Argon2Params.KeyLen,
		},
	}

	json.NewEncoder(w).Encode(response)
}

type SolveRequest struct {
	Challenge string `json:"challenge"`
	Salt      string `json:"salt"`
	Nonce     uint64 `json:"nonce"`
	Hash      string `json:"hash"`
	ExpiresAt string `json:"expires_at"`
}

func (h *Handlers) SolveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req SolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	// Parse expiration time
	exp, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		http.Error(w, "bad expires_at format", 400)
		return
	}

	// Create puzzle and solution structs
	puzzleObj := &puzzle.Puzzle{
		Challenge:  req.Challenge,
		Salt:       req.Salt,
		Difficulty: h.config.PuzzleDifficulty,
		ExpiresAt:  exp,
	}

	solution := &puzzle.Solution{
		Nonce: req.Nonce,
		Hash:  req.Hash,
	}

	// Validate the solution using config parameters
	argon2Config := h.configToArgon2Config()
	if !puzzle.ValidateWithConfig(puzzleObj, solution, argon2Config) {
		http.Error(w, "invalid solution", 400)
		return
	}

	// Generate JWT token
	token, expiresAt, err := jwt.Generate(h.config.JWTSecret, req.Challenge, h.config.TokenExpiryMinutes, h.config.RequestsPerToken)
	if err != nil {
		http.Error(w, "failed to generate token", 500)
		return
	}

	response := map[string]interface{}{
		"token":         token,
		"expires_at":    expiresAt.Format(time.RFC3339),
		"request_limit": h.config.RequestsPerToken,
		"algorithm":     h.config.Algorithm,
	}

	json.NewEncoder(w).Encode(response)
}

// VerifyHandler handles JWT token verification for nginx auth_request
func (h *Handlers) VerifyHandler(w http.ResponseWriter, r *http.Request) {
	// Extract Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Check if it's a Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokenString := parts[1]

	// Verify JWT token
	claims, err := jwt.Verify(tokenString, h.config.JWTSecret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Check rate limiting
	tokenID := claims.ID
	if tokenID != "" {
		h.tokenMutex.Lock()
		currentUsage := h.tokenUsage[tokenID]
		if currentUsage >= h.config.RequestsPerToken {
			h.tokenMutex.Unlock()
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", h.config.RequestsPerToken))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		h.tokenUsage[tokenID] = currentUsage + 1
		newUsage := h.tokenUsage[tokenID]
		h.tokenMutex.Unlock()

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", h.config.RequestsPerToken))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", h.config.RequestsPerToken-newUsage))
	}

	// Token is valid
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"puzzle_difficulty":   h.config.PuzzleDifficulty,
		"token_expiry_min":    h.config.TokenExpiryMinutes,
		"requests_per_token":  h.config.RequestsPerToken,
		"jwt_secret_present":  h.config.JWTSecret != "",
		"algorithm":           h.config.Algorithm,
		"expected_iterations": puzzle.GetExpectedIterations(h.config.PuzzleDifficulty),
		"argon2_params": map[string]interface{}{
			"memory_kb": h.config.Argon2Params.MemoryKB,
			"time":      h.config.Argon2Params.Time,
			"threads":   h.config.Argon2Params.Threads,
			"key_len":   h.config.Argon2Params.KeyLen,
		},
	}

	json.NewEncoder(w).Encode(response)
}
