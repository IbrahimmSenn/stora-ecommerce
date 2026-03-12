package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	BcryptCost  int
	JWTSecret   string
	Port        string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Port:        os.Getenv("PORT"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	costStr := os.Getenv("BCRYPT_COST")
	if costStr == "" {
		cfg.BcryptCost = 10
	} else {
		cost, err := strconv.Atoi(costStr)
		if err != nil {
			return nil, fmt.Errorf("BCRYPT_COST must be a valid integer: %w", err)
		}
		cfg.BcryptCost = cost
	}

	return cfg, nil
}
