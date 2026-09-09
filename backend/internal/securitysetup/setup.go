package securitysetup

import (
	"context"
	"fmt"
	"log"

	"github.com/lhilove/security-copilot/internal/ai"
	"github.com/lhilove/security-copilot/internal/database"
	gh "github.com/lhilove/security-copilot/internal/github"
	"github.com/lhilove/security-copilot/internal/scanner"
)

// Result reports what was enabled and whether AI scan ran as fallback.
type Result struct {
	DependabotEnabled     bool
	SecretScanningEnabled bool
	CodeScanningEnabled   bool
	AIFindingsCount       int
	Errors                []string
}

// Service handles enabling GitHub security features and running AI fallback scans.
type Service struct {
	aiProvider  ai.Provider
	findingRepo *database.FindingRepository
}

func NewService(aiProvider ai.Provider, findingRepo *database.FindingRepository) *Service {
	return &Service{
		aiProvider:  aiProvider,
		findingRepo: findingRepo,
	}
}

// EnableAndScan enables all GitHub security features for a repo,
// then runs an AI scan immediately for instant findings.
func (s *Service) EnableAndScan(ctx context.Context, ghClient *gh.Client, repoID, owner, repo string) (*Result, error) {
	result := &Result{}

	// Get default branch SHA for CodeQL workflow and file tree
	defaultBranch, treeSHA, err := ghClient.GetDefaultBranchSHA(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("get default branch: %w", err)
	}

	// Enable Dependabot
	if err := ghClient.EnableDependabot(ctx, owner, repo); err != nil {
		log.Printf("security setup: dependabot enable failed for %s/%s: %v", owner, repo, err)
		result.Errors = append(result.Errors, fmt.Sprintf("Dependabot: %v", err))
	} else {
		result.DependabotEnabled = true
		log.Printf("security setup: Dependabot enabled for %s/%s", owner, repo)
	}

	// Enable Secret Scanning
	if err := ghClient.EnableSecretScanning(ctx, owner, repo); err != nil {
		log.Printf("security setup: secret scanning enable failed for %s/%s: %v", owner, repo, err)
		result.Errors = append(result.Errors, fmt.Sprintf("Secret Scanning: %v", err))
	} else {
		result.SecretScanningEnabled = true
		log.Printf("security setup: Secret Scanning enabled for %s/%s", owner, repo)
	}

	// Enable Code Scanning (commits CodeQL workflow)
	if err := ghClient.EnableCodeScanning(ctx, owner, repo, defaultBranch); err != nil {
		log.Printf("security setup: code scanning enable failed for %s/%s: %v", owner, repo, err)
		result.Errors = append(result.Errors, fmt.Sprintf("Code Scanning: %v", err))
	} else {
		result.CodeScanningEnabled = true
		log.Printf("security setup: Code Scanning enabled for %s/%s", owner, repo)
	}

	// Always run AI scan immediately — CodeQL takes time, AI gives results now
	log.Printf("security setup: starting AI scan for %s/%s", owner, repo)
	sc := scanner.New(ghClient, s.aiProvider)
	aiFindings, err := sc.ScanRepository(ctx, owner, repo, treeSHA)
	if err != nil {
		log.Printf("security setup: AI scan failed for %s/%s: %v", owner, repo, err)
		result.Errors = append(result.Errors, fmt.Sprintf("AI Scan: %v", err))
		return result, nil
	}

	if len(aiFindings) == 0 {
		log.Printf("security setup: AI scan found no issues in %s/%s", owner, repo)
		return result, nil
	}

	// Convert scanner findings to database findings + raw data
	dbFindings := make([]database.Finding, 0, len(aiFindings))
	rawData := make([]map[string]any, 0, len(aiFindings))

	for _, f := range aiFindings {
		dbFindings = append(dbFindings, database.Finding{
			RepositoryID:  repoID,
			Source:        "ai_scan",
			SourceAlertID: f.SourceAlertID,
			Severity:      f.Severity,
			Title:         f.Title,
			Description:   f.Description,
			State:         "open",
			FilePath:      f.FilePath,
			LineNumber:    nil, // AI scan is file-level, not line-level
		})

		rawData = append(rawData, map[string]any{
			"source":       "ai_scan",
			"file":         f.FilePath,
			"language":     f.Language,
			"what":         f.AIResult.What,
			"risk":         f.AIResult.Risk,
			"fix":          f.AIResult.Fix,
			"can_auto_fix": f.AIResult.CanAutoFix,
		})
	}

	if err := s.findingRepo.UpsertFindings(ctx, dbFindings, rawData); err != nil {
		log.Printf("security setup: store findings for %s/%s: %v", owner, repo, err)
		result.Errors = append(result.Errors, fmt.Sprintf("Store findings: %v", err))
	} else {
		result.AIFindingsCount = len(aiFindings)
		log.Printf("security setup: stored %d AI findings for %s/%s", len(aiFindings), owner, repo)
	}

	return result, nil
}
