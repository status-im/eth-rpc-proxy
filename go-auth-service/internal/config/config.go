package config

import (
	"encoding/json"
	"os"

	"go-auth-service/internal/puzzle"
)

type Config struct {
	Algorithm          string              `json:"algorithm"`
	JWTSecret          string              `json:"jwt_secret"`
	PuzzleDifficulty   int                 `json:"puzzle_difficulty"`
	RequestsPerToken   int                 `json:"requests_per_token"`
	TokenExpiryMinutes int                 `json:"token_expiry_minutes"`
	Argon2Params       puzzle.Argon2Config `json:"argon2_params"`
}

func Load() (*Config, error) {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "config.json" // default path
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
