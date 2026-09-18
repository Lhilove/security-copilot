package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FindingRepository struct {
	pool *pgxpool.Pool
}

func NewFindingRepository(pool *pgxpool.Pool) *FindingRepository {
	return &FindingRepository{pool: pool}
}

type Finding struct {
	ID             string
	RepositoryID   string
	Source         string
	SourceAlertID  string
	Severity       string
	Title          string
	Description    string
	State          string
	FilePath       string
	LineNumber     *int
	PackageName    string
	CVEID          string
	SecretType     string
	PatchedVersion *string
}

type FindingSummary struct {
	RepositoryID string
	Critical     int
	High         int
	Medium       int
	Low          int
	Total        int
}

// UpsertFindings inserts or updates a batch of findings for a repository.
func (r *FindingRepository) UpsertFindings(ctx context.Context, findings []Finding, rawData []map[string]any) error {
	query := `
		INSERT INTO findings (
			repository_id, source, source_alert_id, severity, title,
			description, state, file_path, line_number, package_name,
			cve_id, secret_type, raw_data, patched_version
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (repository_id, source, source_alert_id)
		DO UPDATE SET
			severity      = EXCLUDED.severity,
			title         = EXCLUDED.title,
			description   = EXCLUDED.description,
			state         = EXCLUDED.state,
			file_path     = EXCLUDED.file_path,
			line_number   = EXCLUDED.line_number,
			package_name  = EXCLUDED.package_name,
			cve_id        = EXCLUDED.cve_id,
			secret_type   = EXCLUDED.secret_type,
			raw_data      = EXCLUDED.raw_data,
			patched_version = EXCLUDED.patched_version,
			updated_at    = NOW()
	`

	for i, f := range findings {
		raw, err := json.Marshal(rawData[i])
		if err != nil {
			return fmt.Errorf("marshal raw data for %s alert %s: %w", f.Source, f.SourceAlertID, err)
		}

		_, err = r.pool.Exec(ctx, query,
			f.RepositoryID,
			f.Source,
			f.SourceAlertID,
			f.Severity,
			f.Title,
			f.Description,
			f.State,
			f.FilePath,
			f.LineNumber,
			f.PackageName,
			f.CVEID,
			f.SecretType,
			raw,
			f.PatchedVersion,
		)
		if err != nil {
			return fmt.Errorf("upsert finding %s/%s: %w", f.Source, f.SourceAlertID, err)
		}
	}

	return nil
}

// GetFindingsByRepositoryID returns all findings for a repository.
func (r *FindingRepository) GetFindingsByRepositoryID(ctx context.Context, repositoryID string) ([]Finding, error) {
	query := `
		SELECT id, repository_id, source, source_alert_id, severity,
			title, description, state, file_path, line_number,
			package_name, cve_id, secret_type, patched_version
		FROM findings
		WHERE repository_id = $1
		ORDER BY
			CASE severity
				WHEN 'critical' THEN 1
				WHEN 'high'     THEN 2
				WHEN 'medium'   THEN 3
				WHEN 'low'      THEN 4
				ELSE 5
			END,
			updated_at DESC
	`

	rows, err := r.pool.Query(ctx, query, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("query findings: %w", err)
	}
	defer rows.Close()

	var findings []Finding
	for rows.Next() {
		var f Finding
		if err := rows.Scan(
			&f.ID,
			&f.RepositoryID,
			&f.Source,
			&f.SourceAlertID,
			&f.Severity,
			&f.Title,
			&f.Description,
			&f.State,
			&f.FilePath,
			&f.LineNumber,
			&f.PackageName,
			&f.CVEID,
			&f.SecretType,
			&f.PatchedVersion,
		); err != nil {
			return nil, fmt.Errorf("scan finding: %w", err)
		}
		findings = append(findings, f)
	}

	return findings, nil
}

// GetFindingSummary returns a severity breakdown and risk score for a repository.
func (r *FindingRepository) GetFindingSummary(ctx context.Context, repositoryID string) (*FindingSummary, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE severity = 'critical') AS critical,
			COUNT(*) FILTER (WHERE severity = 'high')     AS high,
			COUNT(*) FILTER (WHERE severity = 'medium')   AS medium,
			COUNT(*) FILTER (WHERE severity = 'low')      AS low,
			COUNT(*)                                       AS total
		FROM findings
		WHERE repository_id = $1
		AND state = 'open'
	`

	s := &FindingSummary{RepositoryID: repositoryID}
	err := r.pool.QueryRow(ctx, query, repositoryID).Scan(
		&s.Critical,
		&s.High,
		&s.Medium,
		&s.Low,
		&s.Total,
	)
	if err != nil {
		return nil, fmt.Errorf("get finding summary: %w", err)
	}

	return s, nil
}

// Add PatchedVersion to GetFindingByID query and scan
// Replace the existing GetFindingByID function with this one

func (r *FindingRepository) GetFindingByID(ctx context.Context, userID, findingID string) (*Finding, error) {
	query := `
		SELECT f.id, f.repository_id, f.source, f.source_alert_id,
			f.severity, f.title, f.description, f.state,
			f.file_path, f.line_number, f.package_name, f.cve_id, f.secret_type,
			f.patched_version
		FROM findings f
		JOIN repositories r ON r.id = f.repository_id
		WHERE f.id = $1
		AND r.user_id = $2
	`

	var f Finding
	err := r.pool.QueryRow(ctx, query, findingID, userID).Scan(
		&f.ID,
		&f.RepositoryID,
		&f.Source,
		&f.SourceAlertID,
		&f.Severity,
		&f.Title,
		&f.Description,
		&f.State,
		&f.FilePath,
		&f.LineNumber,
		&f.PackageName,
		&f.CVEID,
		&f.SecretType,
		&f.PatchedVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("get finding: %w", err)
	}

	return &f, nil
}
