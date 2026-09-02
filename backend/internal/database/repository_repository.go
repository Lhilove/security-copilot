package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RepositoryRepository struct {
	pool *pgxpool.Pool
}

func NewRepositoryRepository(pool *pgxpool.Pool) *RepositoryRepository {
	return &RepositoryRepository{pool: pool}
}

type Repository struct {
	ID           string
	UserID       string
	GitHubRepoID int64
	Owner        string
	Name         string
	FullName     string
	Private      bool
	Monitored    bool
}

// UpsertRepositories inserts or updates a batch of repositories for a user.
func (r *RepositoryRepository) UpsertRepositories(ctx context.Context, userID string, repos []Repository) error {
	query := `
		INSERT INTO repositories (user_id, github_repo_id, owner, name, full_name, private)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, github_repo_id)
		DO UPDATE SET
			owner     = EXCLUDED.owner,
			name      = EXCLUDED.name,
			full_name = EXCLUDED.full_name,
			private   = EXCLUDED.private,
			updated_at = NOW()
	`

	for _, repo := range repos {
		_, err := r.pool.Exec(ctx, query,
			userID,
			repo.GitHubRepoID,
			repo.Owner,
			repo.Name,
			repo.FullName,
			repo.Private,
		)
		if err != nil {
			return fmt.Errorf("upsert repository %s: %w", repo.FullName, err)
		}
	}

	return nil
}

// GetRepositoriesByUserID returns all stored repositories for a user.
func (r *RepositoryRepository) GetRepositoriesByUserID(ctx context.Context, userID string) ([]Repository, error) {
	query := `
		SELECT id, user_id, github_repo_id, owner, name, full_name, private, monitored
		FROM repositories
		WHERE user_id = $1
		ORDER BY full_name ASC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query repositories: %w", err)
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		var repo Repository
		if err := rows.Scan(
			&repo.ID,
			&repo.UserID,
			&repo.GitHubRepoID,
			&repo.Owner,
			&repo.Name,
			&repo.FullName,
			&repo.Private,
			&repo.Monitored,
		); err != nil {
			return nil, fmt.Errorf("scan repository: %w", err)
		}
		repos = append(repos, repo)
	}

	return repos, nil
}

// SelectRepository marks a repository as monitored for a user.
func (r *RepositoryRepository) SelectRepository(ctx context.Context, userID string, repoID string) error {
	query := `
		UPDATE repositories
		SET monitored = TRUE, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`

	result, err := r.pool.Exec(ctx, query, repoID, userID)
	if err != nil {
		return fmt.Errorf("select repository: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("repository not found or does not belong to user")
	}

	return nil
}

// DeselectRepository unmarks a repository as monitored.
func (r *RepositoryRepository) DeselectRepository(ctx context.Context, userID string, repoID string) error {
	query := `
		UPDATE repositories
		SET monitored = FALSE, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`

	result, err := r.pool.Exec(ctx, query, repoID, userID)
	if err != nil {
		return fmt.Errorf("deselect repository: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("repository not found or does not belong to user")
	}

	return nil
}

func (r *RepositoryRepository) GetRepositoryByID(ctx context.Context, userID string, repoID string) (*Repository, error) {
	query := `
        SELECT id, user_id, github_repo_id, owner, name, full_name, private, monitored
        FROM repositories
        WHERE id = $1 AND user_id = $2
    `

	repo := &Repository{}
	err := r.pool.QueryRow(ctx, query, repoID, userID).Scan(
		&repo.ID,
		&repo.UserID,
		&repo.GitHubRepoID,
		&repo.Owner,
		&repo.Name,
		&repo.FullName,
		&repo.Private,
		&repo.Monitored,
	)
	if err != nil {
		return nil, fmt.Errorf("get repository: %w", err)
	}

	return repo, nil
}
