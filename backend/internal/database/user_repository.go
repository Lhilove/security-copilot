package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

type User struct {
	ID          string
	GitHubID    int64
	GitHubLogin string
	Email       string
}

// UpsertUser inserts a new user or updates their GitHub login and email if they already exist.
func (r *UserRepository) UpsertUser(ctx context.Context, githubID int64, login string, email string) (*User, error) {
	query := `
		INSERT INTO users (github_id, github_login, email)
		VALUES ($1, $2, $3)
		ON CONFLICT (github_id)
		DO UPDATE SET
			github_login = EXCLUDED.github_login,
			email = EXCLUDED.email,
			updated_at = NOW()
		RETURNING id, github_id, github_login, COALESCE(email, '')
	`

	user := &User{}
	err := r.pool.QueryRow(ctx, query, githubID, login, email).Scan(
		&user.ID,
		&user.GitHubID,
		&user.GitHubLogin,
		&user.Email,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}

	return user, nil
}
