package repositories

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/lhilove/security-copilot/internal/crypto"
	"github.com/lhilove/security-copilot/internal/database"
	gh "github.com/lhilove/security-copilot/internal/github"
)

type Service struct {
	repos *database.RepositoryRepository
	conns *database.GitHubConnectionRepository
	key   []byte
}

func NewService(
	repos *database.RepositoryRepository,
	conns *database.GitHubConnectionRepository,
	encryptionKey []byte,
) *Service {
	return &Service{
		repos: repos,
		conns: conns,
		key:   encryptionKey,
	}
}

// SyncRepositories fetches all repositories from GitHub and upserts them.
func (s *Service) SyncRepositories(ctx context.Context, userID string) ([]database.Repository, error) {
	token, err := s.decryptToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("decrypt token: %w", err)
	}

	client := gh.NewClient(token)
	ghRepos, err := client.ListRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}

	var repos []database.Repository
	for _, r := range ghRepos {
		repos = append(repos, database.Repository{
			GitHubRepoID: r.ID,
			Owner:        r.Owner.Login,
			Name:         r.Name,
			FullName:     r.FullName,
			Private:      r.Private,
		})
	}

	if err := s.repos.UpsertRepositories(ctx, userID, repos); err != nil {
		return nil, fmt.Errorf("upsert repositories: %w", err)
	}

	return s.repos.GetRepositoriesByUserID(ctx, userID)
}

// SelectRepository marks a repository as monitored.
func (s *Service) SelectRepository(ctx context.Context, userID, repoID string) error {
	return s.repos.SelectRepository(ctx, userID, repoID)
}

// DeselectRepository unmarks a repository as monitored.
func (s *Service) DeselectRepository(ctx context.Context, userID, repoID string) error {
	return s.repos.DeselectRepository(ctx, userID, repoID)
}

// GetRepositories returns all stored repositories for a user.
func (s *Service) GetRepositories(ctx context.Context, userID string) ([]database.Repository, error) {
	return s.repos.GetRepositoriesByUserID(ctx, userID)
}

func (s *Service) decryptToken(ctx context.Context, userID string) (string, error) {
	conn, err := s.conns.GetConnection(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get connection: %w", err)
	}

	encryptedBytes, err := base64.StdEncoding.DecodeString(conn.AccessTokenEncrypted)
	if err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}

	tokenBytes, err := crypto.Decrypt(s.key, encryptedBytes)
	if err != nil {
		return "", fmt.Errorf("decrypt token: %w", err)
	}

	return string(tokenBytes), nil
}
