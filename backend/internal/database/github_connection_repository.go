package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GitHubConnectionRepository struct {
	pool *pgxpool.Pool
}

func NewGitHubConnectionRepository(pool *pgxpool.Pool) *GitHubConnectionRepository {
	return &GitHubConnectionRepository{pool: pool}
}

type GitHubConnection struct {
	UserID               string
	AccessTokenEncrypted string
}

// UpsertConnection stores an encrypted GitHub access token for a user.
func (r *GitHubConnectionRepository) UpsertConnection(ctx context.Context, userID string, encryptedToken string) error {
	query := `
		INSERT INTO github_connections (user_id, access_token_encrypted)
		VALUES ($1, $2)
		ON CONFLICT (user_id)
		DO UPDATE SET
			access_token_encrypted = EXCLUDED.access_token_encrypted,
			updated_at = NOW()
	`

	_, err := r.pool.Exec(ctx, query, userID, encryptedToken)
	if err != nil {
		return fmt.Errorf("upsert github connection: %w", err)
	}

	return nil
}

// GetConnection retrieves the stored GitHub connection for a user.
func (r *GitHubConnectionRepository) GetConnection(ctx context.Context, userID string) (*GitHubConnection, error) {
	query := `
		SELECT user_id, access_token_encrypted
		FROM github_connections
		WHERE user_id = $1
	`

	conn := &GitHubConnection{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&conn.UserID,
		&conn.AccessTokenEncrypted,
	)
	if err != nil {
		return nil, fmt.Errorf("get github connection: %w", err)
	}

	return conn, nil
}
