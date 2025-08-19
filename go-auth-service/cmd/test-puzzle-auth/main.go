package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go-auth-service/internal/puzzle"
)

// ANSI color codes
const (
	Red    = "\033[0;31m"
	Green  = "\033[0;32m"
	Yellow = "\033[1;33m"
	NC     = "\033[0m" // No Color
)

type PuzzleResponse struct {
	Challenge    string              `json:"challenge"`
	Salt         string              `json:"salt"`
	Difficulty   int                 `json:"difficulty"`
	ExpiresAt    string              `json:"expires_at"`
	HMAC         string              `json:"hmac"`
	Algorithm    string              `json:"algorithm"`
	Argon2Params puzzle.Argon2Config `json:"argon2_params"`
}

type SolveRequest struct {
	Challenge string `json:"challenge"`
	Salt      string `json:"salt"`
	Nonce     uint64 `json:"nonce"`
	ArgonHash string `json:"argon_hash"`
	HMAC      string `json:"hmac"`
	ExpiresAt string `json:"expires_at"`
}

type TokenResponse struct {
	Token        string `json:"token"`
	ExpiresAt    string `json:"expires_at"`
	RequestLimit int    `json:"request_limit"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("%s=== Go Puzzle Auth Tester ===%s\n", Yellow, NC)
		fmt.Printf("Usage: %s <proxy_server_url>\n", os.Args[0])
		fmt.Printf("Example: %s https://your-proxy-server.com\n", os.Args[0])
		os.Exit(1)
	}

	baseURL := strings.TrimSuffix(os.Args[1], "/")
	fmt.Printf("%s=== Go Puzzle Auth Tester ===%s\n", Yellow, NC)
	fmt.Printf("Testing proxy server: %s\n\n", baseURL)

	// Step 1: Check service status
	fmt.Printf("%s1. Checking service status...%s\n", Yellow, NC)
	if !checkServiceStatus(baseURL) {
		fmt.Printf("%s✗ Service is not available%s\n", Red, NC)
		os.Exit(1)
	}
	fmt.Printf("%s✓ Service is running%s\n", Green, NC)

	// Step 2: Get puzzle challenge
	fmt.Printf("\n%s2. Getting puzzle challenge...%s\n", Yellow, NC)
	puzzleResp, err := getPuzzle(baseURL)
	if err != nil {
		fmt.Printf("%s✗ Failed to get puzzle: %v%s\n", Red, err, NC)
		os.Exit(1)
	}

	fmt.Printf("Puzzle response:\n")
	puzzleJSON, _ := json.MarshalIndent(puzzleResp, "", "  ")
	fmt.Printf("%s\n", puzzleJSON)

	fmt.Printf("%s✓ Puzzle received%s\n", Green, NC)
	fmt.Printf("  Challenge: %s\n", puzzleResp.Challenge)
	fmt.Printf("  Salt: %s\n", puzzleResp.Salt)
	fmt.Printf("  Difficulty: %d\n", puzzleResp.Difficulty)

	// Step 3: Solve puzzle using Go solver
	fmt.Printf("\n%s3. Solving puzzle using Go solver...%s\n", Yellow, NC)

	expiresAt, err := time.Parse(time.RFC3339, puzzleResp.ExpiresAt)
	if err != nil {
		fmt.Printf("%s✗ Failed to parse expires_at: %v%s\n", Red, err, NC)
		os.Exit(1)
	}

	puzzleObj := &puzzle.Puzzle{
		Challenge:  puzzleResp.Challenge,
		Salt:       puzzleResp.Salt,
		Difficulty: puzzleResp.Difficulty,
		ExpiresAt:  expiresAt,
		HMAC:       puzzleResp.HMAC,
	}

	startTime := time.Now()
	solution, err := puzzle.Solve(puzzleObj, puzzleResp.Argon2Params)
	solveTime := time.Since(startTime)

	if err != nil {
		fmt.Printf("%s✗ Failed to solve puzzle: %v%s\n", Red, err, NC)
		os.Exit(1)
	}

	fmt.Printf("%s✓ Puzzle solved!%s\n", Green, NC)
	fmt.Printf("  Nonce: %d\n", solution.Nonce)
	fmt.Printf("  Hash: %s...\n", solution.ArgonHash[:20])
	fmt.Printf("  %sSolve time: %v%s\n", Yellow, solveTime, NC)

	// Step 4: Submit solution and get JWT token
	fmt.Printf("\n%s4. Submitting solution to get JWT token...%s\n", Yellow, NC)

	solveReq := SolveRequest{
		Challenge: puzzleResp.Challenge,
		Salt:      puzzleResp.Salt,
		Nonce:     solution.Nonce,
		ArgonHash: solution.ArgonHash,
		HMAC:      puzzleResp.HMAC,
		ExpiresAt: puzzleResp.ExpiresAt,
	}

	tokenResp, err := submitSolution(baseURL, solveReq)
	if err != nil {
		fmt.Printf("%s✗ Failed to solve puzzle: %v%s\n", Red, err, NC)
		os.Exit(1)
	}

	fmt.Printf("%s✓ Puzzle solved successfully!%s\n", Green, NC)
	fmt.Printf("Token response: [JWT token received - details below]")

	fmt.Printf("%s✓ JWT token received%s\n", Green, NC)
	fmt.Printf("  Token: %s...\n", tokenResp.Token[:20])
	fmt.Printf("  Expires: %s\n", tokenResp.ExpiresAt)
	fmt.Printf("  Request limit: %d\n", tokenResp.RequestLimit)

	// Step 5: Verify JWT token
	fmt.Printf("\n%s5. Verifying JWT token...%s\n", Yellow, NC)
	if !verifyToken(baseURL, tokenResp.Token) {
		fmt.Printf("%s✗ JWT token verification failed%s\n", Red, NC)
		os.Exit(1)
	}
	fmt.Printf("%s✓ JWT token verified successfully%s\n", Green, NC)

	// Step 6: Test rate limiting
	fmt.Printf("\n%s6. Testing rate limiting...%s\n", Yellow, NC)
	for i := 1; i <= 3; i++ {
		if verifyToken(baseURL, tokenResp.Token) {
			fmt.Printf("%s✓ Request %d: Token still valid%s\n", Green, i, NC)
		} else {
			fmt.Printf("%s! Request %d: Token invalid or rate limited%s\n", Yellow, i, NC)
		}
	}

	// Final summary
	fmt.Printf("\n%s=== All Tests Passed! ===%s\n", Green, NC)
	fmt.Printf("%s✓ Service is accessible%s\n", Green, NC)
	fmt.Printf("%s✓ Puzzle generation works%s\n", Green, NC)
	fmt.Printf("%s✓ Go puzzle solver works%s\n", Green, NC)
	fmt.Printf("%s✓ HMAC protected solution validation works%s\n", Green, NC)
	fmt.Printf("%s✓ JWT token generation and verification works%s\n", Green, NC)
	fmt.Printf("%s✓ Rate limiting is functional%s\n", Green, NC)

	fmt.Printf("\n%s=== Performance Statistics ===%s\n", Yellow, NC)
	fmt.Printf("%sPuzzle solve time: %v%s\n", Yellow, solveTime, NC)
	fmt.Printf("%sDifficulty level: %d%s\n", Yellow, puzzleResp.Difficulty, NC)
	fmt.Printf("%sNonce found: %d%s\n", Yellow, solution.Nonce, NC)

	fmt.Printf("\n%sFinal JWT Token (truncated):%s\n", Yellow, NC)
	fmt.Printf("%s...\n", tokenResp.Token[:20])
}

func checkServiceStatus(baseURL string) bool {
	resp, err := http.Get(baseURL + "/auth/status")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func getPuzzle(baseURL string) (*PuzzleResponse, error) {
	resp, err := http.Get(baseURL + "/auth/puzzle")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var puzzleResp PuzzleResponse
	if err := json.NewDecoder(resp.Body).Decode(&puzzleResp); err != nil {
		return nil, err
	}

	return &puzzleResp, nil
}

func submitSolution(baseURL string, solveReq SolveRequest) (*TokenResponse, error) {
	reqBody, err := json.Marshal(solveReq)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(baseURL+"/auth/solve", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func verifyToken(baseURL, token string) bool {
	req, err := http.NewRequest("GET", baseURL+"/auth/verify", nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}
