package github

import (
	"context"
	"fmt"

	"github.com/lhilove/security-copilot/internal/database"
)

// WebhookHandler processes incoming GitHub webhook events and persists
// the relevant security findings. All logic here is GitHub-specific.
type WebhookHandler struct {
	repos     *database.RepositoryRepository
	findings  *database.FindingRepository
	normalize func(string) string
}

func NewWebhookHandler(
	repos *database.RepositoryRepository,
	findings *database.FindingRepository,
	normalize func(string) string,
) *WebhookHandler {
	return &WebhookHandler{
		repos:     repos,
		findings:  findings,
		normalize: normalize,
	}
}

func (h *WebhookHandler) HandleCodeScanning(ctx context.Context, event CodeScanningAlertEvent) error {
	repo, err := h.repos.GetRepositoryByGitHubID(ctx, event.Repository.ID)
	if err != nil {
		return nil // not monitored, ignore silently
	}

	lineNum := event.Alert.MostRecentInstance.Location.StartLine
	finding := database.Finding{
		RepositoryID:  repo.ID,
		Source:        "code_scanning",
		SourceAlertID: fmt.Sprintf("%d", event.Alert.Number),
		Severity:      h.normalize(event.Alert.Rule.Severity),
		Title:         event.Alert.Rule.ID,
		Description:   event.Alert.Rule.Description,
		State:         event.Alert.State,
		FilePath:      event.Alert.MostRecentInstance.Location.Path,
		LineNumber:    &lineNum,
	}

	raw := map[string]any{"action": event.Action, "alert_number": event.Alert.Number}
	return h.findings.UpsertFindings(ctx, []database.Finding{finding}, []map[string]any{raw})
}

func (h *WebhookHandler) HandleDependabot(ctx context.Context, event DependabotAlertEvent) error {
	repo, err := h.repos.GetRepositoryByGitHubID(ctx, event.Repository.ID)
	if err != nil {
		return nil
	}

	finding := database.Finding{
		RepositoryID:  repo.ID,
		Source:        "dependabot",
		SourceAlertID: fmt.Sprintf("%d", event.Alert.Number),
		Severity:      h.normalize(event.Alert.SecurityVulnerability.Severity),
		Title:         event.Alert.SecurityAdvisory.Summary,
		Description:   event.Alert.SecurityAdvisory.Description,
		State:         event.Alert.State,
		PackageName:   event.Alert.SecurityVulnerability.Package.Name,
		CVEID:         event.Alert.SecurityAdvisory.CVEId,
	}

	raw := map[string]any{"action": event.Action, "alert_number": event.Alert.Number}
	return h.findings.UpsertFindings(ctx, []database.Finding{finding}, []map[string]any{raw})
}

func (h *WebhookHandler) HandleSecretScanning(ctx context.Context, event SecretScanningAlertEvent) error {
	repo, err := h.repos.GetRepositoryByGitHubID(ctx, event.Repository.ID)
	if err != nil {
		return nil
	}

	finding := database.Finding{
		RepositoryID:  repo.ID,
		Source:        "secret_scanning",
		SourceAlertID: fmt.Sprintf("%d", event.Alert.Number),
		Severity:      "critical",
		Title:         fmt.Sprintf("Secret detected: %s", event.Alert.SecretType),
		State:         event.Alert.State,
		SecretType:    event.Alert.SecretType,
	}

	raw := map[string]any{"action": event.Action, "alert_number": event.Alert.Number}
	return h.findings.UpsertFindings(ctx, []database.Finding{finding}, []map[string]any{raw})
}
