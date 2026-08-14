package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURL  string
}

func Load() (*Config, error) {
	// Load .env when running locally. In production, environment variables will normally be provided by the deployment environment.
	_ = godotenv.Load()

	cfg := &Config{
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		GitHubRedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
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

	return cfg, nil
}
