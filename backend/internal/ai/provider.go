package ai

import (
	"context"
	"fmt"
)

// AnalysisRequest contains everything the AI needs to analyze a finding.
// Repository content is treated as untrusted data, never as instructions.
type AnalysisRequest struct {
	Source       string // code_scanning, dependabot, secret_scanning
	Severity     string
	Title        string
	Description  string
	FilePath     string
	LineNumber   int
	PackageName  string
	CVEID        string
	SecretType   string
	AffectedCode string // raw code snippet from the file, if available
	Language     string
}

// AnalysisResult is what the AI returns.
type AnalysisResult struct {
	What         string // one line: what the vulnerability is
	Risk         string // one line: business impact
	Fix          string // brief fix description
	ProposedCode string // the actual fixed code, if applicable
	CanAutoFix   bool   // whether the AI is confident enough to propose a PR
}

// Provider is the interface every AI backend must implement.
// Swap implementations in main.go without touching anything else.
type Provider interface {
	Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResult, error)
}

// validateRequest guards against obviously malformed or injection-attempt requests.
// Repository content is never trusted as instructions at any layer.
func validateRequest(req AnalysisRequest) error {
	if req.Source == "" {
		return fmt.Errorf("source is required")
	}
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	// Cap code snippet size to prevent context stuffing attacks
	if len(req.AffectedCode) > 8000 {
		return fmt.Errorf("affected code exceeds maximum allowed size")
	}
	return nil
}
