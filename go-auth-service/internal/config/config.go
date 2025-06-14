package config

import (
	"os"
	"strconv"
)

type Config struct {
	PuzzleDifficulty   int
	TokenExpiryMinutes int
	RequestsPerToken   int
	JWTSecret          string
}

func Load() *Config {
	return &Config{
		PuzzleDifficulty:   getEnvInt("PUZZLE_DIFFICULTY", 2),
		TokenExpiryMinutes: getEnvInt("TOKEN_EXPIRY_MINUTES", 10),
		RequestsPerToken:   getEnvInt("REQUESTS_PER_TOKEN", 100),
		JWTSecret:          getEnv("JWT_SECRET", "supersecret"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
