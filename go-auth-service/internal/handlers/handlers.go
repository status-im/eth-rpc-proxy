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
	"go-auth-service/internal/metrics"
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

func (h *Handlers) PuzzleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	p, err := puzzle.Generate(h.config.PuzzleDifficulty, h.config.TokenExpiryMinutes, h.config.JWTSecret)
	if err != nil {
		http.Error(w, "failed to generate puzzle", 500)
		return
	}

	response := map[string]interface{}{
		"challenge":     p.Challenge,
		"salt":          p.Salt,
		"difficulty":    p.Difficulty,
		"expires_at":    p.ExpiresAt.Format(time.RFC3339),
		"hmac":          p.HMAC,
		"algorithm":     h.config.Algorithm,
		"argon2_params": h.config.Argon2Params,
		"solve_request_format": map[string]interface{}{
			"required_fields": []string{"challenge", "salt", "nonce", "argon_hash", "hmac", "expires_at"},
		},
	}

	json.NewEncoder(w).Encode(response)
}

// SolveRequest for puzzle solving
type SolveRequest struct {
	Challenge string `json:"challenge"`
	Salt      string `json:"salt"`
	Nonce     uint64 `json:"nonce"`
	ArgonHash string `json:"argon_hash"`
	HMAC      string `json:"hmac"`
	ExpiresAt string `json:"expires_at"`
}

// SolveHandler handles HMAC protected solutions only
func (h *Handlers) SolveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req SolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		metrics.RecordPuzzleAttempt("invalid_request")
		http.Error(w, "bad request", 400)
		return
	}

	// Validate required fields
	if req.Challenge == "" || req.Salt == "" || req.ArgonHash == "" || req.HMAC == "" {
		metrics.RecordPuzzleAttempt("missing_fields")
		http.Error(w, "missing required fields: challenge, salt, argon_hash, hmac", 400)
		return
	}

	// Parse expiration time
	exp, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		metrics.RecordPuzzleAttempt("invalid_expiry")
		http.Error(w, "bad expires_at format", 400)
		return
	}

	// Check if puzzle has expired
	if time.Now().After(exp) {
		metrics.RecordPuzzleAttempt("expired")
		http.Error(w, "puzzle has expired", 400)
		return
	}

	// Create puzzle struct with HMAC from request
	puzzleObj := &puzzle.Puzzle{
		Challenge:  req.Challenge,
		Salt:       req.Salt,
		Difficulty: h.config.PuzzleDifficulty,
		ExpiresAt:  exp,
		HMAC:       req.HMAC,
	}

	solution := &puzzle.Solution{
		Nonce:     req.Nonce,
		ArgonHash: req.ArgonHash,
	}

	// Validate with HMAC protection
	if !puzzle.ValidateHMACProtectedSolution(puzzleObj, solution, h.config.Argon2Params, h.config.JWTSecret) {
		metrics.RecordPuzzleAttempt("invalid_solution")
		http.Error(w, "invalid solution or HMAC verification failed", 400)
		return
	}

	// Record successful puzzle solve
	metrics.RecordPuzzleAttempt("success")
	metrics.IncrementPuzzlesSolved()

	// Generate JWT token
	token, expiresAt, err := jwt.Generate(h.config.JWTSecret, req.Challenge, h.config.TokenExpiryMinutes, h.config.RequestsPerToken)
	if err != nil {
		http.Error(w, "failed to generate token", 500)
		return
	}

	// Record token issuance
	metrics.IncrementTokensIssued()

	response := map[string]interface{}{
		"token":         token,
		"expires_at":    expiresAt.Format(time.RFC3339),
		"request_limit": h.config.RequestsPerToken,
	}

	json.NewEncoder(w).Encode(response)
}

// TestSolveHandler provides a test endpoint that generates a valid solution with HMAC
func (h *Handlers) TestSolveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Generate a test puzzle
	p, err := puzzle.Generate(h.config.PuzzleDifficulty, h.config.TokenExpiryMinutes, h.config.JWTSecret)
	if err != nil {
		http.Error(w, "failed to generate test puzzle", 500)
		return
	}

	// Solve the puzzle
	solution, err := puzzle.Solve(p, h.config.Argon2Params)
	if err != nil {
		http.Error(w, "failed to solve test puzzle", 500)
		return
	}

	response := map[string]interface{}{
		"test_puzzle": map[string]interface{}{
			"challenge":  p.Challenge,
			"salt":       p.Salt,
			"difficulty": p.Difficulty,
			"expires_at": p.ExpiresAt.Format(time.RFC3339),
		},
		"example_request": map[string]interface{}{
			"challenge":  p.Challenge,
			"salt":       p.Salt,
			"nonce":      solution.Nonce,
			"argon_hash": solution.ArgonHash,
			"hmac":       p.HMAC,
			"expires_at": p.ExpiresAt.Format(time.RFC3339),
		},
	}

	json.NewEncoder(w).Encode(response)
}

// VerifyHandler handles JWT token verification for nginx auth_request
func (h *Handlers) VerifyHandler(w http.ResponseWriter, r *http.Request) {
	// Extract Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		metrics.RecordTokenVerification("missing_token")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Check if it's a Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		metrics.RecordTokenVerification("invalid_format")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokenString := parts[1]

	// Verify JWT token
	claims, err := jwt.Verify(tokenString, h.config.JWTSecret)
	if err != nil {
		metrics.RecordTokenVerification("invalid_token")
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
			metrics.RecordTokenVerification("rate_limited")
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

	// Record successful verification
	metrics.RecordTokenVerification("success")

	// Token is valid
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"puzzle_difficulty":  h.config.PuzzleDifficulty,
		"token_expiry_min":   h.config.TokenExpiryMinutes,
		"requests_per_token": h.config.RequestsPerToken,
		"jwt_secret_present": h.config.JWTSecret != "",
		"algorithm":          h.config.Algorithm,
		"argon2_params":      h.config.Argon2Params,
		"endpoints": map[string]interface{}{
			"puzzle": "/auth/puzzle",
			"solve":  "/auth/solve",
			"verify": "/auth/verify",
			"status": "/auth/status",
		},
	}

	json.NewEncoder(w).Encode(response)
}
