package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RemediationRepository struct {
	pool *pgxpool.Pool
}

func NewRemediationRepository(pool *pgxpool.Pool) *RemediationRepository {
	return &RemediationRepository{pool: pool}
}

type Remediation struct {
	ID           string
	FindingID    string
	UserID       string
	What         string
	Risk         string
	Fix          string
	ProposedCode string
	CanAutoFix   bool
	Status       string
	PRUrl        string
	PRNumber     *int
	ApprovedAt   *string
	DeclinedAt   *string
	CreatedAt    string
	UpdatedAt    string
	FileSHA      string
	FileContent  string
}

// CreateRemediation stores a new AI-generated remediation proposal.
func (r *RemediationRepository) CreateRemediation(ctx context.Context, rem Remediation) (*Remediation, error) {
	query := `
		INSERT INTO remediations (
			finding_id, user_id, what, risk, fix, proposed_code, can_auto_fix, status, file_sha, file_content
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9)
		ON CONFLICT (finding_id, user_id)
		DO UPDATE SET
			what          = EXCLUDED.what,
			risk          = EXCLUDED.risk,
			fix           = EXCLUDED.fix,
			proposed_code = EXCLUDED.proposed_code,
			can_auto_fix  = EXCLUDED.can_auto_fix,
			status        = 'pending',
			approved_at   = NULL,
			declined_at   = NULL,
			pr_url        = NULL,
			pr_number     = NULL,
			file_sha      = EXCLUDED.file_sha,
 			file_content  = EXCLUDED.file_content,
			updated_at    = NOW()
		RETURNING id, finding_id, user_id, what, risk, fix,
			proposed_code, can_auto_fix, status, created_at::text, updated_at::text
	`

	result := &Remediation{}
	err := r.pool.QueryRow(ctx, query,
		rem.FindingID,
		rem.UserID,
		rem.What,
		rem.Risk,
		rem.Fix,
		rem.ProposedCode,
		rem.CanAutoFix,
		rem.FileSHA,
		rem.FileContent,
	).Scan(
		&result.ID,
		&result.FindingID,
		&result.UserID,
		&result.What,
		&result.Risk,
		&result.Fix,
		&result.ProposedCode,
		&result.CanAutoFix,
		&result.Status,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create remediation: %w", err)
	}

	return result, nil
}

// GetRemediation retrieves a remediation by finding ID and user ID.
func (r *RemediationRepository) GetRemediation(ctx context.Context, findingID, userID string) (*Remediation, error) {
	query := `
		SELECT id, finding_id, user_id, what, risk, fix,
			proposed_code, can_auto_fix, status,
			COALESCE(pr_url, ''),
			approved_at::text,
			declined_at::text,
			created_at::text,
			updated_at::text,
			COALESCE(file_sha, ''),
			COALESCE(file_content, '')
		FROM remediations
		WHERE finding_id = $1 AND user_id = $2
	`

	result := &Remediation{}
	err := r.pool.QueryRow(ctx, query, findingID, userID).Scan(
		&result.ID,
		&result.FindingID,
		&result.UserID,
		&result.What,
		&result.Risk,
		&result.Fix,
		&result.ProposedCode,
		&result.CanAutoFix,
		&result.Status,
		&result.PRUrl,
		&result.ApprovedAt,
		&result.DeclinedAt,
		&result.CreatedAt,
		&result.UpdatedAt,
		&result.FileSHA,
		&result.FileContent,
	)
	if err != nil {
		return nil, fmt.Errorf("get remediation: %w", err)
	}

	return result, nil
}

// ApproveRemediation marks a remediation as approved.
func (r *RemediationRepository) ApproveRemediation(ctx context.Context, findingID, userID string) (*Remediation, error) {
	query := `
		UPDATE remediations
		SET status      = 'approved',
			approved_at = NOW(),
			updated_at  = NOW()
		WHERE finding_id = $1
		AND user_id      = $2
		AND status       = 'pending'
		RETURNING id, finding_id, user_id, what, risk, fix,
			proposed_code, can_auto_fix, status, created_at::text, updated_at::text
	`

	result := &Remediation{}
	err := r.pool.QueryRow(ctx, query, findingID, userID).Scan(
		&result.ID,
		&result.FindingID,
		&result.UserID,
		&result.What,
		&result.Risk,
		&result.Fix,
		&result.ProposedCode,
		&result.CanAutoFix,
		&result.Status,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("approve remediation: %w", err)
	}

	return result, nil
}

// DeclineRemediation marks a remediation as declined.
func (r *RemediationRepository) DeclineRemediation(ctx context.Context, findingID, userID string) error {
	query := `
		UPDATE remediations
		SET status      = 'declined',
			declined_at = NOW(),
			updated_at  = NOW()
		WHERE finding_id = $1
		AND user_id      = $2
		AND status       = 'pending'
	`

	result, err := r.pool.Exec(ctx, query, findingID, userID)
	if err != nil {
		return fmt.Errorf("decline remediation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no pending remediation found for this finding")
	}

	return nil
}

// UpdatePRDetails stores the PR URL and number after a PR is created.
func (r *RemediationRepository) UpdatePRDetails(ctx context.Context, findingID, userID, prURL string, prNumber int) error {
	query := `
		UPDATE remediations
		SET status     = 'pr_created',
			pr_url     = $3,
			pr_number  = $4,
			updated_at = NOW()
		WHERE finding_id = $1
		AND user_id      = $2
		AND status       = 'approved'
	`

	result, err := r.pool.Exec(ctx, query, findingID, userID, prURL, prNumber)
	if err != nil {
		return fmt.Errorf("update pr details: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no approved remediation found for this finding")
	}

	return nil
}

// GetRemediationsByUserID returns all remediations for a user with their status.
func (r *RemediationRepository) GetRemediationsByUserID(ctx context.Context, userID string) ([]Remediation, error) {
	query := `
		SELECT id, finding_id, user_id, what, risk, fix,
			proposed_code, can_auto_fix, status,
			COALESCE(pr_url, ''),
			created_at::text,
			updated_at::text
		FROM remediations
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get remediations: %w", err)
	}
	defer rows.Close()

	var remediations []Remediation
	for rows.Next() {
		var rem Remediation
		if err := rows.Scan(
			&rem.ID,
			&rem.FindingID,
			&rem.UserID,
			&rem.What,
			&rem.Risk,
			&rem.Fix,
			&rem.ProposedCode,
			&rem.CanAutoFix,
			&rem.Status,
			&rem.PRUrl,
			&rem.CreatedAt,
			&rem.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan remediation: %w", err)
		}
		remediations = append(remediations, rem)
	}

	return remediations, nil
}
