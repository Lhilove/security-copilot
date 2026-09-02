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
