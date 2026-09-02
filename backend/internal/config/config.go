package config

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURL  string
	DatabaseURL        string
	MigrationsPath     string
	EncryptionKey      []byte // decoded bytes, not string
}

func Load() (*Config, error) {
	// Load .env when running locally. In production, environment variables will normally be provided by the deployment environment.
	_ = godotenv.Load()

	rawKey := os.Getenv("ENCRYPTION_KEY")
	if rawKey == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY is not configured")
	}

	keyBytes, err := base64.StdEncoding.DecodeString(rawKey)
	if err != nil {
		return nil, fmt.Errorf("decode ENCRYPTION_KEY: %w", err)
	}

	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be 32 bytes, got %d", len(keyBytes))
	}

	cfg := &Config{
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		GitHubRedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		MigrationsPath:     os.Getenv("MIGRATIONS_PATH"),
		EncryptionKey:      keyBytes,
	}

	if cfg.GitHubClientID == "" {
		return nil, fmt.Errorf("GITHUB_CLIENT_ID is not configured")
	}

	if cfg.GitHubClientSecret == "" {
		return nil, fmt.Errorf("GITHUB_CLIENT_SECRET is not configured")
	}

	if cfg.GitHubRedirectURL == "" {
		return nil, fmt.Errorf("GITHUB_REDIRECT_URL is not configured")
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not configured")
	}

	if cfg.MigrationsPath == "" {
		cfg.MigrationsPath = "migrations" // default path
	}

	return cfg, nil
}
