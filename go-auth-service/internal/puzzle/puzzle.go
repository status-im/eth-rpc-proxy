package puzzle

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
)

type Argon2Config struct {
	MemoryKB int
	Time     int
	Threads  int
	KeyLen   int
}

type Puzzle struct {
	Challenge  string    `json:"challenge"`
	Salt       string    `json:"salt"`
	Difficulty int       `json:"difficulty"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type Solution struct {
	Nonce uint64 `json:"nonce"`
	Hash  string `json:"hash"`
}

// Generate creates a new Argon2id-based puzzle
func Generate(difficulty int, ttlMinutes int) (*Puzzle, error) {
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

	return &Puzzle{
		Challenge:  hex.EncodeToString(challengeBytes),
		Salt:       hex.EncodeToString(saltBytes),
		Difficulty: difficulty,
		ExpiresAt:  time.Now().Add(time.Duration(ttlMinutes) * time.Minute),
	}, nil
}

// ValidateWithConfig checks if the provided solution is correct for the puzzle
func ValidateWithConfig(puzzle *Puzzle, solution *Solution, argon2Config Argon2Config) bool {
	// Check if puzzle has expired
	if time.Now().After(puzzle.ExpiresAt) {
		return false
	}

	// Recreate the hash with provided nonce
	computedHash := computeArgon2HashWithConfig(puzzle.Challenge, puzzle.Salt, solution.Nonce, puzzle.Difficulty, argon2Config)

	// Verify the hash matches
	if computedHash != solution.Hash {
		return false
	}

	// Check if hash meets difficulty requirement (leading zeros)
	return checkDifficulty(computedHash, puzzle.Difficulty)
}

// SolveWithConfig attempts to find a valid nonce for the puzzle (for testing purposes)
func SolveWithConfig(puzzle *Puzzle, argon2Config Argon2Config) (*Solution, error) {
	if time.Now().After(puzzle.ExpiresAt) {
		return nil, fmt.Errorf("puzzle has expired")
	}

	// Try different nonces until we find one that meets difficulty
	for nonce := uint64(0); nonce < 1000000; nonce++ { // Limit attempts for safety
		hash := computeArgon2HashWithConfig(puzzle.Challenge, puzzle.Salt, nonce, puzzle.Difficulty, argon2Config)

		if checkDifficulty(hash, puzzle.Difficulty) {
			return &Solution{
				Nonce: nonce,
				Hash:  hash,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to solve puzzle within attempt limit")
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

	// Use config parameters
	memory := uint32(argon2Config.MemoryKB)
	time := uint32(argon2Config.Time) + uint32(difficulty-1) // Increase time with difficulty
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

// GetExpectedIterations returns estimated iterations needed for given difficulty
func GetExpectedIterations(difficulty int) uint64 {
	// For Argon2, iterations are more expensive than SHA256
	// Each difficulty level increases expected work significantly
	base := uint64(16)
	for i := 1; i < difficulty; i++ {
		base *= 16
	}
	return base
}
