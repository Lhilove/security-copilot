package findings

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/lhilove/security-copilot/internal/crypto"
	"github.com/lhilove/security-copilot/internal/database"
	gh "github.com/lhilove/security-copilot/internal/github"
)

type Service struct {
	findings *database.FindingRepository
	repos    *database.RepositoryRepository
	conns    *database.GitHubConnectionRepository
	key      []byte
}

func NewService(
	findings *database.FindingRepository,
	repos *database.RepositoryRepository,
	conns *database.GitHubConnectionRepository,
	encryptionKey []byte,
) *Service {
	return &Service{
		findings: findings,
		repos:    repos,
		conns:    conns,
		key:      encryptionKey,
	}
}

// SyncRepository fetches findings from all GitHub security sources for a
// monitored repository and persists them. It decrypts the stored token,
// calls GitHub, normalizes the results, and upserts into the database.
func (s *Service) SyncRepository(ctx context.Context, userID, repoID string) (int, error) {
	repo, err := s.repos.GetRepositoryByID(ctx, userID, repoID)
	if err != nil {
		return 0, fmt.Errorf("get repository: %w", err)
	}

	token, err := s.DecryptToken(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("decrypt token: %w", err)
	}

	client := gh.NewClient(token)

	var findings []database.Finding
	var rawData []map[string]any

	codeAlerts, err := client.ListCodeScanningAlerts(ctx, repo.Owner, repo.Name)
	if err != nil {
		// Non-fatal: log and continue with other sources
		fmt.Printf("code scanning unavailable for %s: %v\n", repo.FullName, err)
	} else {
		for _, a := range codeAlerts {
			lineNum := a.MostRecentInstance.Location.StartLine
			findings = append(findings, database.Finding{
				RepositoryID:  repoID,
				Source:        "code_scanning",
				SourceAlertID: fmt.Sprintf("%d", a.Number),
				Severity:      NormalizeSeverity(a.Rule.Severity),
				Title:         a.Rule.ID,
				Description:   a.Rule.Description,
				State:         a.State,
				FilePath:      a.MostRecentInstance.Location.Path,
				LineNumber:    &lineNum,
			})
			rawData = append(rawData, a.RawData)
		}
	}

	depAlerts, err := client.ListDependabotAlerts(ctx, repo.Owner, repo.Name)
	if err != nil {
		fmt.Printf("dependabot unavailable for %s: %v\n", repo.FullName, err)
	} else {
		for _, a := range depAlerts {
			var patchedVersion string
			if a.SecurityVulnerability.FirstPatchedVersion != nil {
				patchedVersion = a.SecurityVulnerability.FirstPatchedVersion.Identifier
			}
			findings = append(findings, database.Finding{
				RepositoryID:   repoID,
				Source:         "dependabot",
				SourceAlertID:  fmt.Sprintf("%d", a.Number),
				Severity:       NormalizeSeverity(a.SecurityVulnerability.Severity),
				Title:          a.SecurityAdvisory.Summary,
				Description:    a.SecurityAdvisory.Description,
				State:          a.State,
				PackageName:    a.SecurityVulnerability.Package.Name,
				CVEID:          a.SecurityAdvisory.CVEId,
				PatchedVersion: patchedVersion,
			})
			rawData = append(rawData, a.RawData)
		}
	}

	secretAlerts, err := client.ListSecretScanningAlerts(ctx, repo.Owner, repo.Name)
	if err != nil {
		fmt.Printf("secret scanning unavailable for %s: %v\n", repo.FullName, err)
	} else {
		for _, a := range secretAlerts {
			findings = append(findings, database.Finding{
				RepositoryID:  repoID,
				Source:        "secret_scanning",
				SourceAlertID: fmt.Sprintf("%d", a.Number),
				Severity:      "critical",
				Title:         fmt.Sprintf("Secret detected: %s", a.SecretType),
				State:         a.State,
				SecretType:    a.SecretType,
			})
			rawData = append(rawData, a.RawData)
		}
	}

	if len(findings) == 0 {
		return 0, nil
	}

	if err := s.findings.UpsertFindings(ctx, findings, rawData); err != nil {
		return 0, fmt.Errorf("upsert findings: %w", err)
	}

	return len(findings), nil
}

// GetFindings returns all findings for a repository, verifying ownership.
func (s *Service) GetFindings(ctx context.Context, userID, repoID string) ([]database.Finding, error) {
	if _, err := s.repos.GetRepositoryByID(ctx, userID, repoID); err != nil {
		return nil, fmt.Errorf("repository not found: %w", err)
	}

	return s.findings.GetFindingsByRepositoryID(ctx, repoID)
}

// GetSummary returns the finding severity breakdown for a repository.
func (s *Service) GetSummary(ctx context.Context, userID, repoID string) (*database.FindingSummary, error) {
	if _, err := s.repos.GetRepositoryByID(ctx, userID, repoID); err != nil {
		return nil, fmt.Errorf("repository not found: %w", err)
	}

	return s.findings.GetFindingSummary(ctx, repoID)
}

// ProcessWebhookFinding upserts a single finding received via webhook.
func (s *Service) ProcessWebhookFinding(ctx context.Context, finding database.Finding, raw map[string]any) error {
	return s.findings.UpsertFindings(ctx, []database.Finding{finding}, []map[string]any{raw})
}

// NormalizeSeverity maps GitHub severity strings to our internal values.
func NormalizeSeverity(s string) string {
	switch s {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium", "moderate":
		return "medium"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

func (s *Service) DecryptToken(ctx context.Context, userID string) (string, error) {
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
