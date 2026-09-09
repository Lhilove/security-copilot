package scanner

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lhilove/security-copilot/internal/ai"
	gh "github.com/lhilove/security-copilot/internal/github"
)

// Finding represents a vulnerability found during an AI scan.
type Finding struct {
	SourceAlertID string
	Severity      string
	Title         string
	Description   string
	FilePath      string
	LineNumber    int
	AffectedCode  string
	Language      string
	AIResult      *ai.AnalysisResult
}

// Scanner performs direct AI-powered scans of repository code.
type Scanner struct {
	ghClient   *gh.Client
	aiProvider ai.Provider
}

func New(ghClient *gh.Client, aiProvider ai.Provider) *Scanner {
	return &Scanner{
		ghClient:   ghClient,
		aiProvider: aiProvider,
	}
}

// ScanRepository fetches all scannable files and runs AI analysis on each.
// Returns findings with AI analysis already attached.
func (s *Scanner) ScanRepository(ctx context.Context, owner, repo, treeSHA string) ([]Finding, error) {
	files, err := s.ghClient.ListRepoFiles(ctx, owner, repo, treeSHA)
	if err != nil {
		return nil, fmt.Errorf("list repo files: %w", err)
	}

	log.Printf("AI scan: found %d scannable files in %s/%s", len(files), owner, repo)

	var findings []Finding

	for _, filePath := range files {
		// Respect context cancellation
		select {
		case <-ctx.Done():
			return findings, ctx.Err()
		default:
		}

		content, _, err := s.ghClient.GetFileContent(ctx, owner, repo, filePath)
		if err != nil {
			log.Printf("AI scan: skip %s: %v", filePath, err)
			continue
		}

		// Skip files over 50KB — too large for useful analysis
		if len(content) > 50000 {
			log.Printf("AI scan: skip %s (too large: %d bytes)", filePath, len(content))
			continue
		}

		fileFindings := s.analyzeFile(ctx, filePath, content)
		findings = append(findings, fileFindings...)

		// Rate limit: avoid hammering Ollama or GitHub API
		time.Sleep(500 * time.Millisecond)
	}

	log.Printf("AI scan: completed %s/%s — %d findings", owner, repo, len(findings))
	return findings, nil
}

// analyzeFile sends a file to the AI for security analysis.
func (s *Scanner) analyzeFile(ctx context.Context, filePath, content string) []Finding {
	lang := detectLanguage(filePath)

	req := ai.AnalysisRequest{
		Source:       "ai_scan",
		Severity:     "unknown", // AI will determine severity
		Title:        fmt.Sprintf("AI scan: %s", filePath),
		Description:  "Direct AI security analysis of source file",
		FilePath:     filePath,
		AffectedCode: truncate(content, 6000), // stay well within context window
		Language:     lang,
	}

	result, err := s.aiProvider.Analyze(ctx, req)
	if err != nil {
		log.Printf("AI scan: analyze %s: %v", filePath, err)
		return nil
	}

	// Skip if AI found nothing significant
	if result.What == "" || strings.Contains(strings.ToLower(result.What), "no vulnerability") {
		return nil
	}

	// Derive severity from the AI result risk field
	severity := inferSeverity(result.Risk)

	finding := Finding{
		SourceAlertID: alertID(filePath, result.What),
		Severity:      severity,
		Title:         result.What,
		Description:   result.Risk,
		FilePath:      filePath,
		AffectedCode:  req.AffectedCode,
		Language:      lang,
		AIResult:      result,
	}

	return []Finding{finding}
}

// alertID creates a stable unique ID for an AI scan finding.
func alertID(filePath, what string) string {
	h := sha256.Sum256([]byte(filePath + "|" + what))
	return fmt.Sprintf("ai-%x", h[:8])
}

// inferSeverity maps AI risk descriptions to severity levels.
func inferSeverity(risk string) string {
	risk = strings.ToLower(risk)
	switch {
	case strings.Contains(risk, "critical") || strings.Contains(risk, "remote code") ||
		strings.Contains(risk, "authentication bypass"):
		return "critical"
	case strings.Contains(risk, "high") || strings.Contains(risk, "data breach") ||
		strings.Contains(risk, "injection") || strings.Contains(risk, "privilege"):
		return "high"
	case strings.Contains(risk, "medium") || strings.Contains(risk, "information disclosure") ||
		strings.Contains(risk, "sensitive"):
		return "medium"
	default:
		return "low"
	}
}

// detectLanguage maps file extension to language name.
func detectLanguage(path string) string {
	switch {
	case strings.HasSuffix(path, ".go"):
		return "Go"
	case strings.HasSuffix(path, ".js"), strings.HasSuffix(path, ".jsx"):
		return "JavaScript"
	case strings.HasSuffix(path, ".ts"), strings.HasSuffix(path, ".tsx"):
		return "TypeScript"
	case strings.HasSuffix(path, ".py"):
		return "Python"
	case strings.HasSuffix(path, ".java"):
		return "Java"
	case strings.HasSuffix(path, ".rb"):
		return "Ruby"
	case strings.HasSuffix(path, ".php"):
		return "PHP"
	case strings.HasSuffix(path, ".cs"):
		return "C#"
	case strings.HasSuffix(path, ".rs"):
		return "Rust"
	case strings.HasSuffix(path, ".swift"):
		return "Swift"
	default:
		return "unknown"
	}
}

// truncate limits content to maxLen characters to stay within AI context window.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... [truncated for analysis]"
}
