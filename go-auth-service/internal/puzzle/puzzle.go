package puzzle

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
)

type Argon2Config struct {
	MemoryKB int `json:"memory_kb"`
	Time     int `json:"time"`
	Threads  int `json:"threads"`
	KeyLen   int `json:"key_len"`
}

type Puzzle struct {
	Challenge  string    `json:"challenge"`
	Salt       string    `json:"salt"`
	Difficulty int       `json:"difficulty"`
	ExpiresAt  time.Time `json:"expires_at"`
	HMAC       string    `json:"hmac"`
}

// Solution contains the Argon2 hash and nonce
type Solution struct {
	ArgonHash string `json:"argon_hash"`
	Nonce     uint64 `json:"nonce"`
}

// Generate creates a new Argon2id-based puzzle with HMAC
func Generate(difficulty int, ttlMinutes int, jwtSecret string) (*Puzzle, error) {
	// Generate random challenge (16 bytes)
	challengeBytes := make([]byte, 16)
	if _, err := rand.Read(challengeBytes); err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	// Generate random salt (16 bytes)
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	challenge := hex.EncodeToString(challengeBytes)
	salt := hex.EncodeToString(saltBytes)
	expiresAt := time.Now().Add(time.Duration(ttlMinutes) * time.Minute)

	// Create HMAC for puzzle verification (challenge + salt + difficulty + expires_at)
	puzzleData := fmt.Sprintf("%s%s%d%s", challenge, salt, difficulty, expiresAt.Format(time.RFC3339))
	puzzleHMAC := computeHMAC(puzzleData, jwtSecret)

	return &Puzzle{
		Challenge:  challenge,
		Salt:       salt,
		Difficulty: difficulty,
		ExpiresAt:  expiresAt,
		HMAC:       puzzleHMAC,
	}, nil
}

// ValidateHMACProtectedSolution validates a solution with HMAC protection (only secure method)
func ValidateHMACProtectedSolution(puzzle *Puzzle, solution *Solution, argon2Config Argon2Config, jwtSecret string) bool {
	// Step 1: Check HMAC signature of puzzle conditions FIRST (most important security check)
	puzzleData := fmt.Sprintf("%s%s%d%s", puzzle.Challenge, puzzle.Salt, puzzle.Difficulty, puzzle.ExpiresAt.Format(time.RFC3339))
	expectedHMAC := computeHMAC(puzzleData, jwtSecret)
	if !hmac.Equal([]byte(expectedHMAC), []byte(puzzle.HMAC)) {
		return false
	}

	// Step 2: Check if puzzle has expired
	if time.Now().After(puzzle.ExpiresAt) {
		return false
	}

	// Step 3: Recreate the Argon2 hash
	computedArgonHash := computeArgon2HashWithConfig(puzzle.Challenge, puzzle.Salt, solution.Nonce, puzzle.Difficulty, argon2Config)

	// Step 4: Verify the Argon2 hash matches
	if computedArgonHash != solution.ArgonHash {
		return false
	}

	// Step 5: Check difficulty requirement
	return checkDifficulty(computedArgonHash, puzzle.Difficulty)
}

// Solve creates a solution for the puzzle
func Solve(puzzle *Puzzle, argon2Config Argon2Config) (*Solution, error) {
	if time.Now().After(puzzle.ExpiresAt) {
		return nil, fmt.Errorf("puzzle has expired")
	}

	// Try different nonces until we find one that meets difficulty
	for nonce := uint64(0); nonce < 1000000; nonce++ {
		argonHash := computeArgon2HashWithConfig(puzzle.Challenge, puzzle.Salt, nonce, puzzle.Difficulty, argon2Config)

		if checkDifficulty(argonHash, puzzle.Difficulty) {
			return &Solution{
				ArgonHash: argonHash,
				Nonce:     nonce,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to solve puzzle within attempt limit")
}

// computeHMAC creates HMAC-SHA256 signature for the given hash
func computeHMAC(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// computeArgon2HashWithConfig computes Argon2id hash for given parameters and config
func computeArgon2HashWithConfig(challenge, salt string, nonce uint64, difficulty int, argon2Config Argon2Config) string {
	// Create input: challenge + salt + nonce
	input := fmt.Sprintf("%s%s%d", challenge, salt, nonce)

	// Decode salt from hex
	saltBytes, err := hex.DecodeString(salt)
	if err != nil {
		// Fallback to using salt as-is if decode fails
		saltBytes = []byte(salt)
	}

	// Use config parameters directly without difficulty adjustment
	memory := uint32(argon2Config.MemoryKB)
	time := uint32(argon2Config.Time)
	threads := uint8(argon2Config.Threads)
	keyLen := uint32(argon2Config.KeyLen)

	// Compute Argon2id hash
	hash := argon2.IDKey([]byte(input), saltBytes, time, memory, threads, keyLen)

	return hex.EncodeToString(hash)
}

// checkDifficulty verifies if hash meets the difficulty requirement
func checkDifficulty(hash string, difficulty int) bool {
	if len(hash) < difficulty {
		return false
	}

	// Check for leading zeros
	for i := 0; i < difficulty; i++ {
		if hash[i] != '0' {
			return false
		}
	}

	return true
}
