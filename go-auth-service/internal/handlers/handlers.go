package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"go-auth-service/internal/config"
	"go-auth-service/internal/jwt"
	"go-auth-service/internal/puzzle"
)

var cfg = config.Load()

func PuzzleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	p, err := puzzle.Generate(cfg.PuzzleDifficulty, cfg.TokenExpiryMinutes)
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
		"algorithm":           "argon2id",
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

func SolveHandler(w http.ResponseWriter, r *http.Request) {
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
		Difficulty: cfg.PuzzleDifficulty,
		ExpiresAt:  exp,
	}

	solution := &puzzle.Solution{
		Nonce: req.Nonce,
		Hash:  req.Hash,
	}

	// Validate the solution
	if !puzzle.Validate(puzzleObj, solution) {
		http.Error(w, "invalid solution", 400)
		return
	}

	// Generate JWT token
	token, expiresAt, err := jwt.Generate(cfg.JWTSecret, req.Challenge, cfg.TokenExpiryMinutes, cfg.RequestsPerToken)
	if err != nil {
		http.Error(w, "failed to generate token", 500)
		return
	}

	response := map[string]interface{}{
		"token":         token,
		"expires_at":    expiresAt.Format(time.RFC3339),
		"request_limit": cfg.RequestsPerToken,
		"algorithm":     "argon2id",
	}

	json.NewEncoder(w).Encode(response)
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"puzzle_difficulty":   cfg.PuzzleDifficulty,
		"token_expiry_min":    cfg.TokenExpiryMinutes,
		"requests_per_token":  cfg.RequestsPerToken,
		"jwt_secret_present":  cfg.JWTSecret != "",
		"algorithm":           "argon2id",
		"expected_iterations": puzzle.GetExpectedIterations(cfg.PuzzleDifficulty),
		"argon2_params": map[string]interface{}{
			"memory_kb": 32 * 1024,
			"time":      1,
			"threads":   4,
			"key_len":   32,
		},
	}

	json.NewEncoder(w).Encode(response)
}
