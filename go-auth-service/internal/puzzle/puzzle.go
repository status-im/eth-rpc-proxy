package puzzle

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
)

// Argon2 parameters for puzzle system
const (
	// Memory usage in KB (32MB for difficulty balance)
	ArgonMemory = 32 * 1024
	// Number of iterations (time parameter)
	ArgonTime = 1
	// Number of threads
	ArgonThreads = 4
	// Key length in bytes
	ArgonKeyLen = 32
)

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

// Validate checks if the provided solution is correct for the puzzle
func Validate(puzzle *Puzzle, solution *Solution) bool {
	// Check if puzzle has expired
	if time.Now().After(puzzle.ExpiresAt) {
		return false
	}

	// Recreate the hash with provided nonce
	computedHash := computeArgon2Hash(puzzle.Challenge, puzzle.Salt, solution.Nonce, puzzle.Difficulty)

	// Verify the hash matches
	if computedHash != solution.Hash {
		return false
	}

	// Check if hash meets difficulty requirement (leading zeros)
	return checkDifficulty(computedHash, puzzle.Difficulty)
}

// Solve attempts to find a valid nonce for the puzzle (for testing purposes)
func Solve(puzzle *Puzzle) (*Solution, error) {
	if time.Now().After(puzzle.ExpiresAt) {
		return nil, fmt.Errorf("puzzle has expired")
	}

	// Try different nonces until we find one that meets difficulty
	for nonce := uint64(0); nonce < 1000000; nonce++ { // Limit attempts for safety
		hash := computeArgon2Hash(puzzle.Challenge, puzzle.Salt, nonce, puzzle.Difficulty)

		if checkDifficulty(hash, puzzle.Difficulty) {
			return &Solution{
				Nonce: nonce,
				Hash:  hash,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to solve puzzle within attempt limit")
}

// computeArgon2Hash computes Argon2id hash for given parameters
func computeArgon2Hash(challenge, salt string, nonce uint64, difficulty int) string {
	// Create input: challenge + salt + nonce
	input := fmt.Sprintf("%s%s%d", challenge, salt, nonce)

	// Decode salt from hex
	saltBytes, err := hex.DecodeString(salt)
	if err != nil {
		// Fallback to using salt as-is if decode fails
		saltBytes = []byte(salt)
	}

	// Adjust Argon2 parameters based on difficulty
	memory := uint32(ArgonMemory)
	time := ArgonTime + uint32(difficulty-1) // Increase time with difficulty

	// Compute Argon2id hash
	hash := argon2.IDKey([]byte(input), saltBytes, time, memory, ArgonThreads, ArgonKeyLen)

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
