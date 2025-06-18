package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// TokensIssued tracks the total number of JWT tokens issued
	TokensIssued = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auth_tokens_issued_total",
		Help: "The total number of JWT tokens issued",
	})

	// PuzzlesSolved tracks the total number of puzzles solved successfully
	PuzzlesSolved = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auth_puzzles_solved_total",
		Help: "The total number of puzzles solved successfully",
	})

	// PuzzleAttempts tracks puzzle solution attempts (including failed ones)
	PuzzleAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_puzzle_attempts_total",
		Help: "The total number of puzzle solution attempts",
	}, []string{"status"}) // status: "success", "failed", "invalid_hmac", "expired"

	// TokenVerifications tracks JWT token verification attempts
	TokenVerifications = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_token_verifications_total",
		Help: "The total number of token verification attempts",
	}, []string{"status"}) // status: "success", "failed", "expired", "rate_limited"
)

// IncrementTokensIssued increments the tokens issued counter
func IncrementTokensIssued() {
	TokensIssued.Inc()
}

// IncrementPuzzlesSolved increments the puzzles solved counter
func IncrementPuzzlesSolved() {
	PuzzlesSolved.Inc()
}

// RecordPuzzleAttempt records a puzzle solution attempt
func RecordPuzzleAttempt(status string) {
	PuzzleAttempts.WithLabelValues(status).Inc()
}

// RecordTokenVerification records a token verification attempt
func RecordTokenVerification(status string) {
	TokenVerifications.WithLabelValues(status).Inc()
}
