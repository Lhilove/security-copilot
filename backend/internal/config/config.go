package config

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GitHubClientID      string
	GitHubClientSecret  string
	GitHubRedirectURL   string
	DatabaseURL         string
	MigrationsPath      string
	EncryptionKey       []byte
	JWTSecret           []byte
	GitHubWebhookSecret string
	NvidiaAPIKey        string
	NvidiaBaseURL       string
	AIModel             string
	AIProvider          string
	OllamaBaseURL       string
	FrontendURL         string
	BaseURL             string
	SendByteAPIKey      string
	EmailFrom           string
}

func Load() (*Config, error) {
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

	rawJWT := os.Getenv("JWT_SECRET")
	if rawJWT == "" {
		return nil, fmt.Errorf("JWT_SECRET is not configured")
	}
	jwtBytes, err := base64.StdEncoding.DecodeString(rawJWT)
	if err != nil {
		return nil, fmt.Errorf("decode JWT_SECRET: %w", err)
	}

	aiProvider := os.Getenv("AI_PROVIDER")
	if aiProvider == "" {
		aiProvider = "ollama"
	}

	cfg := &Config{
		GitHubClientID:      os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret:  os.Getenv("GITHUB_CLIENT_SECRET"),
		GitHubRedirectURL:   os.Getenv("GITHUB_REDIRECT_URL"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		MigrationsPath:      os.Getenv("MIGRATIONS_PATH"),
		GitHubWebhookSecret: os.Getenv("GITHUB_WEBHOOK_SECRET"),
		NvidiaAPIKey:        os.Getenv("NVIDIA_API_KEY"),
		NvidiaBaseURL:       os.Getenv("NVIDIA_BASE_URL"),
		AIModel:             os.Getenv("AI_MODEL"),
		AIProvider:          aiProvider,
		OllamaBaseURL:       os.Getenv("OLLAMA_BASE_URL"),
		FrontendURL:         os.Getenv("FRONTEND_URL"),
		BaseURL:             os.Getenv("BASE_URL"),
		SendByteAPIKey:      os.Getenv("SENDBYTE_API_KEY"),
		EmailFrom:           os.Getenv("EMAIL_FROM"),
		EncryptionKey:       keyBytes,
		JWTSecret:           jwtBytes,
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
	if cfg.GitHubWebhookSecret == "" {
		return nil, fmt.Errorf("GITHUB_WEBHOOK_SECRET is not configured")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("BASE_URL is not configured")
	}
	if cfg.MigrationsPath == "" {
		cfg.MigrationsPath = "migrations"
	}
	if cfg.AIProvider == "nvidia" && cfg.NvidiaAPIKey == "" {
		return nil, fmt.Errorf("NVIDIA_API_KEY is not configured")
	}
	if cfg.FrontendURL == "" {
		cfg.FrontendURL = "http://localhost:5173"
	}
	if cfg.OllamaBaseURL == "" {
		cfg.OllamaBaseURL = "http://localhost:11434"
	}

	return cfg, nil
}
