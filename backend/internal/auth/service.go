package auth

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/lhilove/security-copilot/internal/config"
	"github.com/lhilove/security-copilot/internal/crypto"
	"github.com/lhilove/security-copilot/internal/database"
)

type Service struct {
	githubAuth *GitHubAuth
	users      *database.UserRepository
	conns      *database.GitHubConnectionRepository
	key        []byte
	jwtSecret  []byte
}

func NewService(
	cfg *config.Config,
	users *database.UserRepository,
	conns *database.GitHubConnectionRepository,
) *Service {
	return &Service{
		githubAuth: NewGitHubAuth(cfg),
		users:      users,
		conns:      conns,
		key:        cfg.EncryptionKey,
		jwtSecret:  cfg.JWTSecret,
	}
}

func (s *Service) GitHubAuth() *GitHubAuth {
	return s.githubAuth
}

type ConnectResult struct {
	Token string
	User  *database.User
}

// ConnectGitHub persists the GitHub identity and encrypted token,
// then issues a JWT for the session.
func (s *Service) ConnectGitHub(ctx context.Context, code string) (*ConnectResult, error) {
	token, err := s.githubAuth.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	githubUser, err := s.githubAuth.GetUser(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("get github user: %w", err)
	}

	dbUser, err := s.users.UpsertUser(ctx, githubUser.ID, githubUser.Login, githubUser.Email)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}

	encryptedToken, err := crypto.Encrypt(s.key, []byte(token.AccessToken))
	if err != nil {
		return nil, fmt.Errorf("encrypt token: %w", err)
	}

	encodedToken := base64.StdEncoding.EncodeToString(encryptedToken)

	if err := s.conns.UpsertConnection(ctx, dbUser.ID, encodedToken); err != nil {
		return nil, fmt.Errorf("upsert connection: %w", err)
	}

	jwtToken, err := IssueToken(dbUser.ID, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &ConnectResult{
		Token: jwtToken,
		User:  dbUser,
	}, nil
}
