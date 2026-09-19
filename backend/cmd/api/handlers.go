package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lhilove/security-copilot/internal/ai"
	"github.com/lhilove/security-copilot/internal/auth"
	"github.com/lhilove/security-copilot/internal/database"
	gh "github.com/lhilove/security-copilot/internal/github"
	"github.com/lhilove/security-copilot/internal/notifications"
	"github.com/lhilove/security-copilot/internal/risk"
)

// healthHandler godoc
// @Summary     Health check
// @Description Returns the service status
// @Tags        system
// @Produce     json
// @Success     200 {object} map[string]string
// @Router      /health [get]
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok", "service": "security-copilot-api"})
}

// githubLoginHandler godoc
// @Summary     Initiate GitHub OAuth
// @Description Redirects the user to GitHub for OAuth authorization
// @Tags        auth
// @Success     302
// @Router      /api/v1/auth/github [get]
func (s *Server) githubLoginHandler(c *gin.Context) {
	state, err := s.authService.GitHubAuth().GenerateState()
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate OAuth state"})
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		MaxAge:   600,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	c.Redirect(302, s.authService.GitHubAuth().LoginURL(state))
}

// githubCallbackHandler godoc
// @Summary     GitHub OAuth callback
// @Description Handles the GitHub OAuth callback, persists the user and encrypted token, returns a JWT
// @Tags        auth
// @Produce     json
// @Param       code  query string true "Authorization code from GitHub"
// @Param       state query string true "OAuth state parameter"
// @Success     200   {object} map[string]interface{}
// @Failure     400   {object} map[string]string
// @Failure     500   {object} map[string]string
// @Router      /api/v1/auth/github/callback [get]
func (s *Server) githubCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	receivedState := c.Query("state")

	if code == "" {
		c.JSON(400, gin.H{"error": "missing authorization code"})
		return
	}

	cookie, err := c.Request.Cookie("oauth_state")
	if err != nil {
		log.Printf("oauth_state cookie missing: %v", err)
		c.JSON(400, gin.H{"error": "missing OAuth state"})
		return
	}

	if !auth.ValidateState(cookie.Value, receivedState) {
		c.JSON(400, gin.H{"error": "invalid OAuth state"})
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	result, err := s.authService.ConnectGitHub(c.Request.Context(), code)
	if err != nil {
		log.Printf("connect github: %v", err)
		c.JSON(500, gin.H{"error": "authentication failed"})
		return
	}

	// Redirect to frontend callback page with JWT in URL
	// The frontend reads the token, stores it in session, and redirects to dashboard
	c.Redirect(302, fmt.Sprintf("%s/callback?token=%s", s.cfg.FrontendURL, result.Token))
}

// webhookHandler godoc
// @Summary     GitHub webhook receiver
// @Description Receives and processes GitHub security event webhooks. Verifies HMAC-SHA256 signature before processing.
// @Tags        webhooks
// @Accept      json
// @Produce     json
// @Param       X-Hub-Signature-256 header string true "GitHub webhook signature"
// @Param       X-GitHub-Event      header string true "GitHub event type"
// @Success     200 {object} map[string]string
// @Failure     400 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Router      /api/v1/webhooks/github [post]
func (s *Server) webhookHandler(c *gin.Context) {
	body, err := gh.ReadWebhookBody(c.Request)
	if err != nil {
		c.JSON(400, gin.H{"error": "failed to read request body"})
		return
	}

	if err := gh.VerifyWebhookSignature(body, c.GetHeader("X-Hub-Signature-256"), s.cfg.GitHubWebhookSecret); err != nil {
		log.Printf("webhook signature failed: %v", err)
		c.JSON(401, gin.H{"error": "invalid webhook signature"})
		return
	}

	eventType := c.GetHeader("X-GitHub-Event")

	switch eventType {
	case "code_scanning_alert":
		var event gh.CodeScanningAlertEvent
		if err := json.Unmarshal(body, &event); err != nil {
			c.JSON(400, gin.H{"error": "invalid payload"})
			return
		}
		if err := s.webhookSvc.HandleCodeScanning(c.Request.Context(), event); err != nil {
			log.Printf("webhook code scanning: %v", err)
			c.JSON(500, gin.H{"error": "failed to process event"})
			return
		}
	case "dependabot_alert":
		var event gh.DependabotAlertEvent
		if err := json.Unmarshal(body, &event); err != nil {
			c.JSON(400, gin.H{"error": "invalid payload"})
			return
		}
		if err := s.webhookSvc.HandleDependabot(c.Request.Context(), event); err != nil {
			log.Printf("webhook dependabot: %v", err)
			c.JSON(500, gin.H{"error": "failed to process event"})
			return
		}
	case "secret_scanning_alert":
		var event gh.SecretScanningAlertEvent
		if err := json.Unmarshal(body, &event); err != nil {
			c.JSON(400, gin.H{"error": "invalid payload"})
			return
		}
		if err := s.webhookSvc.HandleSecretScanning(c.Request.Context(), event); err != nil {
			log.Printf("webhook secret scanning: %v", err)
			c.JSON(500, gin.H{"error": "failed to process event"})
			return
		}
	case "ping":
		c.JSON(200, gin.H{"status": "ok"})
		return
	default:
		c.JSON(200, gin.H{"status": "ignored"})
		return
	}

	c.JSON(200, gin.H{"status": "processed"})
}

// listRepositoriesHandler godoc
// @Summary     List repositories
// @Description Syncs and returns all GitHub repositories for the authenticated user
// @Tags        repositories
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} map[string]interface{}
// @Failure     401 {object} map[string]string
// @Failure     500 {object} map[string]string
// @Router      /api/v1/repositories [get]
func (s *Server) listRepositoriesHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	repos, err := s.repoService.SyncRepositories(c.Request.Context(), userID)
	if err != nil {
		log.Printf("sync repositories: %v", err)
		c.JSON(500, gin.H{"error": "failed to fetch repositories"})
		return
	}
	c.JSON(200, gin.H{"repositories": repos, "count": len(repos)})
}

// selectRepositoryHandler godoc
// @Summary     Select repository for monitoring
// @Description Marks a repository as monitored by Security Copilot
// @Tags        repositories
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Repository UUID"
// @Success     200 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /api/v1/repositories/{id}/select [post]
func (s *Server) selectRepositoryHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	if err := s.repoService.SelectRepository(c.Request.Context(), userID, c.Param("id")); err != nil {
		log.Printf("select repository: %v", err)
		c.JSON(404, gin.H{"error": "repository not found"})
		return
	}
	c.JSON(200, gin.H{"message": "repository selected for monitoring"})

}

// deselectRepositoryHandler godoc
// @Summary     Deselect repository
// @Description Removes a repository from Security Copilot monitoring
// @Tags        repositories
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Repository UUID"
// @Success     200 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /api/v1/repositories/{id}/select [delete]
func (s *Server) deselectRepositoryHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	if err := s.repoService.DeselectRepository(c.Request.Context(), userID, c.Param("id")); err != nil {
		log.Printf("deselect repository: %v", err)
		c.JSON(404, gin.H{"error": "repository not found"})
		return
	}
	c.JSON(200, gin.H{"message": "repository deselected"})
}

// syncFindingsHandler godoc
// @Summary     Sync security findings
// @Description Fetches latest findings from GitHub Code Scanning, Dependabot, and Secret Scanning
// @Tags        findings
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Repository UUID"
// @Success     200 {object} map[string]interface{}
// @Failure     401 {object} map[string]string
// @Failure     500 {object} map[string]string
// @Router      /api/v1/repositories/{id}/sync [post]
func (s *Server) syncFindingsHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	repoID := c.Param("id")
	count, err := s.findingService.SyncRepository(c.Request.Context(), userID, repoID)
	if err != nil {
		log.Printf("sync findings: %v", err)
		c.JSON(500, gin.H{"error": "failed to sync findings"})
		return
	}
	c.JSON(200, gin.H{"synced": count})
}

// listFindingsHandler godoc
// @Summary     List findings
// @Description Returns all security findings for a repository, ordered by severity
// @Tags        findings
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Repository UUID"
// @Success     200 {object} map[string]interface{}
// @Failure     401 {object} map[string]string
// @Failure     500 {object} map[string]string
// @Router      /api/v1/repositories/{id}/findings [get]
func (s *Server) listFindingsHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	f, err := s.findingService.GetFindings(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		log.Printf("get findings: %v", err)
		c.JSON(500, gin.H{"error": "failed to retrieve findings"})
		return
	}
	c.JSON(200, gin.H{"findings": f, "count": len(f)})
}

// repositoryOverviewHandler godoc
// @Summary     Repository security overview
// @Description Returns risk score and finding severity breakdown for a repository
// @Tags        repositories
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Repository UUID"
// @Success     200 {object} map[string]interface{}
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /api/v1/repositories/{id}/overview [get]
func (s *Server) repositoryOverviewHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	repoID := c.Param("id")

	repo, err := s.repoRepo.GetRepositoryByID(c.Request.Context(), userID, repoID)
	if err != nil {
		c.JSON(404, gin.H{"error": "repository not found"})
		return
	}

	summary, err := s.findingService.GetSummary(c.Request.Context(), userID, repoID)
	if err != nil {
		log.Printf("get summary: %v", err)
		c.JSON(500, gin.H{"error": "failed to get overview"})
		return
	}

	score := risk.Calculate(summary)

	c.JSON(200, gin.H{
		"repository": repo.FullName,
		"monitored":  repo.Monitored,
		"security": gin.H{
			"risk_score": score.RiskScore,
			"findings": gin.H{
				"critical": score.Critical,
				"high":     score.High,
				"medium":   score.Medium,
				"low":      score.Low,
				"total":    score.Total,
			},
		},
	})
}

// analyzeFindingHandler godoc
// @Summary     Analyze finding with AI
// @Description Sends a security finding to DeepSeek AI for business impact analysis and remediation proposal
// @Tags        findings
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Finding UUID"
// @Success     200 {object} map[string]interface{}
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Failure     500 {object} map[string]string
// @Router      /api/v1/findings/{id}/analyze [post]
func (s *Server) analyzeFindingHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	findingID := c.Param("id")

	finding, err := s.findingRepo.GetFindingByID(c.Request.Context(), userID, findingID)
	if err != nil {
		log.Printf("get finding: %v", err)
		c.JSON(404, gin.H{"error": "finding not found"})
		return
	}

	// Get the repository for this finding
	repo, err := s.repoRepo.GetRepositoryByID(c.Request.Context(), userID, finding.RepositoryID)
	if err != nil {
		log.Printf("get repository: %v", err)
		c.JSON(500, gin.H{"error": "failed to get repository"})
		return
	}

	// Decrypt the GitHub token
	token, err := s.findingService.DecryptToken(c.Request.Context(), userID)
	if err != nil {
		log.Printf("decrypt token: %v", err)
		c.JSON(500, gin.H{"error": "failed to get github token"})
		return
	}

	req := ai.AnalysisRequest{
		Source:      finding.Source,
		Severity:    finding.Severity,
		Title:       finding.Title,
		Description: finding.Description,
		FilePath:    finding.FilePath,
		PackageName: finding.PackageName,
		CVEID:       finding.CVEID,
		SecretType:  finding.SecretType,
	}

	if finding.LineNumber != nil {
		req.LineNumber = *finding.LineNumber
	}

	// Fetch actual file content if we have a file path
	var fileSHA, fileContent string
	if finding.FilePath != "" && finding.LineNumber != nil {
		ghClient := gh.NewClient(token)
		content, sha, err := ghClient.GetFileContent(c.Request.Context(), repo.Owner, repo.Name, finding.FilePath)
		if err != nil {
			// Non-fatal: log and continue without code context
			log.Printf("fetch file content for %s: %v", finding.FilePath, err)
		} else {
			fileContent = content
			fileSHA = sha
			req.AffectedCode = gh.ExtractCodeContext(content, *finding.LineNumber, 10)
			req.Language = detectLanguage(finding.FilePath)
		}
	}

	result, err := s.aiProvider.Analyze(c.Request.Context(), req)
	if err != nil {
		log.Printf("ai analyze: %v", err)
		c.JSON(500, gin.H{"error": "analysis failed"})
		return
	}
	// Persist the remediation proposal
	remediation, err := s.remediationRepo.CreateRemediation(c.Request.Context(), database.Remediation{
		FindingID:    findingID,
		UserID:       userID,
		What:         result.What,
		Risk:         result.Risk,
		Fix:          result.Fix,
		ProposedCode: result.ProposedCode,
		CanAutoFix:   result.CanAutoFix,
		FileSHA:      fileSHA,     // from GetFileContent
		FileContent:  fileContent, // full file content
	})
	if err != nil {
		log.Printf("create remediation: %v", err)
		c.JSON(500, gin.H{"error": "failed to save remediation"})
		return
	}

	c.JSON(200, gin.H{
		"remediation_id": remediation.ID,
		"finding_id":     findingID,
		"what":           result.What,
		"risk":           result.Risk,
		"fix":            result.Fix,
		"proposed_code":  result.ProposedCode,
		"can_auto_fix":   result.CanAutoFix,
	})

	// After saving remediation, notify user
	go func() {
		notifTitle := fmt.Sprintf("New finding: %s", finding.Title)
		notifBody := fmt.Sprintf("Severity: %s\n\n%s\n\nRisk: %s", finding.Severity, result.What, result.Risk)
		s.notifDispatcher.FindingAlert(context.Background(), userID, findingID, notifTitle, notifBody)
	}()
}

// approveRemediationHandler godoc
// @Summary     Approve remediation
// @Description Developer approves the AI-proposed fix, triggering PR creation
// @Tags        remediations
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Finding UUID"
// @Success     200 {object} map[string]interface{}
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /api/v1/findings/{id}/approve [post]
func (s *Server) approveRemediationHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	findingID := c.Param("id")

	remediation, err := s.remediationRepo.ApproveRemediation(c.Request.Context(), findingID, userID)
	if err != nil {
		log.Printf("approve remediation: %v", err)
		c.JSON(404, gin.H{"error": "no pending remediation found for this finding"})
		return
	}

	// Tell the frontend whether a PR can be created automatically
	// so it can decide whether to call /pr or just show "marked as reviewed"
	canAutoFix := remediation.ProposedCode != ""

	// Also check if this is a dependabot finding with a patched version
	finding, err := s.findingRepo.GetFindingByID(c.Request.Context(), userID, findingID)
	if err == nil && finding.Source == "dependabot" && finding.PatchedVersion != nil && *finding.PatchedVersion != "" {
		canAutoFix = true
	}

	c.JSON(200, gin.H{
		"remediation_id": remediation.ID,
		"finding_id":     findingID,
		"status":         remediation.Status,
		"can_auto_fix":   canAutoFix,
		"message":        "remediation approved",
	})
}

// declineRemediationHandler godoc
// @Summary     Decline remediation
// @Description Developer declines the AI-proposed fix
// @Tags        remediations
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Finding UUID"
// @Success     200 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /api/v1/findings/{id}/decline [post]
func (s *Server) declineRemediationHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	findingID := c.Param("id")

	if err := s.remediationRepo.DeclineRemediation(c.Request.Context(), findingID, userID); err != nil {
		log.Printf("decline remediation: %v", err)
		c.JSON(404, gin.H{"error": "no pending remediation found for this finding"})
		return
	}

	c.JSON(200, gin.H{
		"finding_id": findingID,
		"status":     "declined",
		"message":    "remediation declined",
	})
}

// listRemediationsHandler godoc
// @Summary     List remediations
// @Description Returns all remediation proposals for the authenticated user
// @Tags        remediations
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} map[string]interface{}
// @Failure     401 {object} map[string]string
// @Router      /api/v1/remediations [get]
func (s *Server) listRemediationsHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	remediations, err := s.remediationRepo.GetRemediationsByUserID(c.Request.Context(), userID)
	if err != nil {
		log.Printf("list remediations: %v", err)
		c.JSON(500, gin.H{"error": "failed to retrieve remediations"})
		return
	}

	c.JSON(200, gin.H{
		"remediations": remediations,
		"count":        len(remediations),
	})
}

func detectLanguage(filePath string) string {
	switch {
	case strings.HasSuffix(filePath, ".py"):
		return "python"
	case strings.HasSuffix(filePath, ".go"):
		return "go"
	case strings.HasSuffix(filePath, ".js"), strings.HasSuffix(filePath, ".ts"):
		return "javascript"
	case strings.HasSuffix(filePath, ".java"):
		return "java"
	case strings.HasSuffix(filePath, ".rb"):
		return "ruby"
	case strings.HasSuffix(filePath, ".php"):
		return "php"
	default:
		return "unknown"
	}
}

// createPRHandler godoc
// @Summary     Create pull request
// @Description Creates a GitHub pull request with the approved remediation fix
// @Tags        remediations
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Finding UUID"
// @Success     200 {object} map[string]interface{}
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Failure     500 {object} map[string]string
// @Router      /api/v1/findings/{id}/pr [post]
func (s *Server) createPRHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	findingID := c.Param("id")

	// Get the approved remediation
	remediation, err := s.remediationRepo.GetRemediation(c.Request.Context(), findingID, userID)
	if err != nil {
		log.Printf("get remediation: %v", err)
		c.JSON(404, gin.H{"error": "no remediation found for this finding"})
		return
	}

	if remediation.Status != "approved" {
		c.JSON(400, gin.H{"error": "remediation must be approved before creating a PR"})
		return
	}

	// Get the finding
	finding, err := s.findingRepo.GetFindingByID(c.Request.Context(), userID, findingID)
	if err != nil {
		log.Printf("get finding: %v", err)
		c.JSON(404, gin.H{"error": "finding not found"})
		return
	}

	// Get the repository
	repo, err := s.repoRepo.GetRepositoryByID(c.Request.Context(), userID, finding.RepositoryID)
	if err != nil {
		log.Printf("get repository: %v", err)
		c.JSON(500, gin.H{"error": "failed to get repository"})
		return
	}

	// Decrypt token
	token, err := s.findingService.DecryptToken(c.Request.Context(), userID)
	if err != nil {
		log.Printf("decrypt token: %v", err)
		c.JSON(500, gin.H{"error": "failed to get github token"})
		return
	}

	ghClient := gh.NewClient(token)

	// Get default branch and its SHA
	defaultBranch, baseSHA, err := ghClient.GetDefaultBranchSHA(c.Request.Context(), repo.Owner, repo.Name)
	if err != nil {
		log.Printf("get default branch: %v", err)
		c.JSON(500, gin.H{"error": "failed to get repository branch info"})
		return
	}

	// Create a new branch for the fix
	branchName := fmt.Sprintf("security-copilot/fix-%s-%d", findingID[:8], time.Now().Unix())
	if err := ghClient.CreateBranch(c.Request.Context(), repo.Owner, repo.Name, branchName, baseSHA); err != nil {
		log.Printf("create branch: %v", err)
		c.JSON(500, gin.H{"error": "failed to create branch"})
		return
	}

	var (
		filePath      string
		newContent    string
		fileSHA       string
		commitMessage string
		prTitle       string
		prBody        string
	)

	// Dependabot: bump the dependency version in the manifest
	if finding.Source == "dependabot" && finding.PatchedVersion != nil && *finding.PatchedVersion != "" && finding.PackageName != "" {
		manifestPath, manifest, manifestSHA, err := fetchManifest(c.Request.Context(), ghClient, repo.Owner, repo.Name)
		if err != nil {
			log.Printf("fetch manifest: %v", err)
			c.JSON(500, gin.H{"error": "failed to read dependency manifest: " + err.Error()})
			return
		}

		bumped, err := bumpDependencyVersion(manifest, manifestPath, finding.PackageName, *finding.PatchedVersion)
		if err != nil {
			log.Printf("bump dependency: %v", err)
			c.JSON(500, gin.H{"error": "failed to apply version bump: " + err.Error()})
			return
		}

		filePath = manifestPath
		newContent = bumped
		fileSHA = manifestSHA
		commitMessage = fmt.Sprintf(
			"fix(deps): bump %s to %s\n\nAutomated fix by Security Copilot\nFinding: %s\nCVE: %s",
			finding.PackageName, *finding.PatchedVersion, findingID, finding.CVEID,
		)
		prTitle = fmt.Sprintf("fix(deps): bump %s to %s", finding.PackageName, *finding.PatchedVersion)
		prBody = fmt.Sprintf(
			"## Security Copilot — Dependency Update\n\n"+
				"**Package:** `%s`\n"+
				"**Patched version:** `%s`\n"+
				"**CVE:** %s\n\n"+
				"**Finding:** %s\n\n"+
				"**Risk:** %s\n\n"+
				"**Fix:** %s\n\n"+
				"---\n*This PR was generated by Security Copilot and approved by the repository owner.*",
			finding.PackageName,
			*finding.PatchedVersion,
			finding.CVEID,
			finding.Title,
			remediation.Risk,
			remediation.Fix,
		)

		// Code fix: apply AI proposed code patch
	} else if remediation.ProposedCode != "" && finding.FilePath != "" {
		if remediation.FileContent != "" {
			newContent = applyFix(remediation.FileContent, remediation.ProposedCode, finding.LineNumber)
		} else {
			newContent = remediation.ProposedCode
		}

		filePath = finding.FilePath
		fileSHA = remediation.FileSHA
		commitMessage = fmt.Sprintf(
			"fix: remediate %s in %s\n\nAutomated fix by Security Copilot\nFinding: %s\n\n%s",
			finding.Title, finding.FilePath, findingID, remediation.What,
		)
		prTitle = fmt.Sprintf("fix: remediate %s", finding.Title)
		prBody = fmt.Sprintf(
			"## Security Copilot Automated Fix\n\n"+
				"**Finding:** %s\n\n**What:** %s\n\n**Risk:** %s\n\n**Fix:** %s\n\n"+
				"---\n*This PR was generated by Security Copilot and approved by the repository owner.*",
			finding.Title, remediation.What, remediation.Risk, remediation.Fix,
		)

	} else {
		c.JSON(400, gin.H{"error": "no automated fix available for this finding"})
		return
	}

	// Commit the file
	if err := ghClient.UpdateFile(
		c.Request.Context(),
		repo.Owner,
		repo.Name,
		filePath,
		commitMessage,
		newContent,
		fileSHA,
		branchName,
	); err != nil {
		log.Printf("update file: %v", err)
		c.JSON(500, gin.H{"error": "failed to commit fix"})
		return
	}

	// Open the pull request
	pr, err := ghClient.CreatePullRequest(
		c.Request.Context(),
		repo.Owner,
		repo.Name,
		prTitle,
		prBody,
		branchName,
		defaultBranch,
	)
	if err != nil {
		log.Printf("create pr: %v", err)
		c.JSON(500, gin.H{"error": "failed to create pull request"})
		return
	}

	// Update remediation status
	if err := s.remediationRepo.UpdatePRDetails(
		c.Request.Context(),
		findingID,
		userID,
		pr.HTMLURL,
		pr.Number,
	); err != nil {
		log.Printf("update pr details: %v", err)
	}

	c.JSON(200, gin.H{
		"message":    "pull request created successfully",
		"pr_url":     pr.HTMLURL,
		"pr_number":  pr.Number,
		"branch":     branchName,
		"repository": repo.FullName,
	})
}

// fetchManifest tries common manifest files in order and returns the first one found.
func fetchManifest(ctx context.Context, ghClient *gh.Client, owner, repo string) (path, content, sha string, err error) {
	candidates := []string{
		"package.json",
		"go.mod",
		"requirements.txt",
		"pom.xml",
		"build.gradle",
		"Gemfile",
		"Cargo.toml",
		"composer.json",
		"pyproject.toml",
	}

	for _, candidate := range candidates {
		content, sha, err := ghClient.GetFileContent(ctx, owner, repo, candidate)
		if err == nil {
			return candidate, content, sha, nil
		}
	}

	return "", "", "", fmt.Errorf("no supported manifest file found in repository root")
}

// bumpDependencyVersion updates the version of a package in the manifest file.
// Supports: package.json, go.mod, requirements.txt, pom.xml, Gemfile, Cargo.toml, composer.json, pyproject.toml.
func bumpDependencyVersion(content, manifestPath, packageName, patchedVersion string) (string, error) {
	lines := strings.Split(content, "\n")

	switch {
	// package.json
	case manifestPath == "package.json":
		// Match "package-name": "^1.2.3" or "~1.2.3" or "1.2.3"
		// Preserves existing prefix (^, ~, >=, etc.)
		for i, line := range lines {
			if strings.Contains(line, `"`+packageName+`"`) {
				// Extract existing prefix
				re := regexp.MustCompile(`"([\^~>=<]*)([0-9][^"]*)"`)
				lines[i] = re.ReplaceAllStringFunc(line, func(match string) string {
					sub := re.FindStringSubmatch(match)
					if len(sub) < 2 {
						return match
					}
					prefix := sub[1]
					return `"` + prefix + patchedVersion + `"`
				})
			}
		}

	// go.mod
	case manifestPath == "go.mod":
		// Match: require github.com/foo/bar v1.2.3
		// or lines inside require ( ... ) block
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, packageName+" ") || strings.Contains(trimmed, packageName+"@") {
				parts := strings.Fields(trimmed)
				if len(parts) >= 2 {
					// Replace version (last field that starts with v)
					for j, part := range parts {
						if strings.HasPrefix(part, "v") && len(part) > 1 {
							parts[j] = "v" + strings.TrimPrefix(patchedVersion, "v")
							break
						}
					}
					// Preserve original indentation
					indent := line[:len(line)-len(strings.TrimLeft(line, "\t "))]
					lines[i] = indent + strings.Join(parts, " ")
				}
			}
		}

	// requirements.txt
	case manifestPath == "requirements.txt" || manifestPath == "pyproject.toml":
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Match: package==1.2.3 or package>=1.2.3 or package~=1.2.3
			if strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(packageName)) {
				re := regexp.MustCompile(`(?i)(` + regexp.QuoteMeta(packageName) + `)\s*([><=~!]+)\s*[\d][^\s#]*`)
				if re.MatchString(trimmed) {
					lines[i] = re.ReplaceAllString(trimmed, "${1}=="+patchedVersion)
				}
			}
		}

	// pom.xml
	case manifestPath == "pom.xml":
		// Match <version>x.y.z</version> after the package artifactId
		inArtifact := false
		for i, line := range lines {
			if strings.Contains(line, "<artifactId>"+packageName+"</artifactId>") {
				inArtifact = true
			}
			if inArtifact && strings.Contains(line, "<version>") {
				re := regexp.MustCompile(`<version>[^<]*</version>`)
				lines[i] = re.ReplaceAllString(line, "<version>"+patchedVersion+"</version>")
				inArtifact = false
			}
		}

	// Gemfile
	case manifestPath == "Gemfile":
		for i, line := range lines {
			if strings.Contains(line, `gem '`+packageName+`'`) || strings.Contains(line, `gem "`+packageName+`"`) {
				re := regexp.MustCompile(`(['"])(\d[^'"]*)\1`)
				lines[i] = re.ReplaceAllString(line, `'`+patchedVersion+`'`)
			}
		}

	// Cargo.toml
	case manifestPath == "Cargo.toml":
		for i, line := range lines {
			if strings.Contains(line, packageName) && strings.Contains(line, "=") {
				re := regexp.MustCompile(`"[\^~>=<]*[\d][^"]*"`)
				lines[i] = re.ReplaceAllString(line, `"`+patchedVersion+`"`)
			}
		}

	default:
		return "", fmt.Errorf("unsupported manifest format: %s", manifestPath)
	}

	return strings.Join(lines, "\n"), nil
}

// applyFix replaces the code around the vulnerable line with the proposed fix.
// This is a simple line-based replacement. A future version will use proper
// AST-based patching for more precise changes.
func applyFix(originalContent, proposedCode string, lineNumber *int) string {
	if lineNumber == nil || *lineNumber <= 0 {
		return proposedCode
	}

	lines := strings.Split(originalContent, "\n")
	if *lineNumber > len(lines) {
		return originalContent
	}

	// Find the vulnerable line and replace with proposed fix
	// Insert the fix after the vulnerable line as a comment + fix block
	fixLines := strings.Split(proposedCode, "\n")
	result := make([]string, 0, len(lines)+len(fixLines))

	for i, line := range lines {
		if i == *lineNumber-1 {
			// Add a comment marking the original vulnerable line
			result = append(result, "# SECURITY COPILOT: replaced vulnerable code below")
			result = append(result, "# Original: "+line)
			result = append(result, fixLines...)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// setupSecurityHandler godoc
// @Summary     Enable security features and run AI scan
// @Description Auto-enables GitHub Dependabot, Secret Scanning, and CodeQL for a repository,
// @Description then immediately runs an AI-powered direct scan of the codebase for instant findings.
// @Description Runs asynchronously — returns 202 immediately and populates findings in the background.
// @Tags        repositories
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Repository UUID"
// @Success     202 {object} map[string]string
// @Failure     401 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Failure     500 {object} map[string]string
// @Router      /api/v1/repositories/{id}/setup-security [post]
func (s *Server) setupSecurityHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	repoID := c.Param("id")

	// Use existing repo lookup that already scopes by userID
	repo, err := s.repoRepo.GetRepositoryByID(c.Request.Context(), userID, repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Decrypt token via findingService (same pattern as analyzeFindingHandler)
	token, err := s.findingService.DecryptToken(c.Request.Context(), userID)
	if err != nil {
		log.Printf("setup-security: decrypt token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve access token"})
		return
	}

	ghClient := gh.NewClient(token)

	// Run in background — AI scan can take several minutes on large repos
	go func() {
		bgCtx := context.Background()
		result, err := s.securitySetup.EnableAndScan(bgCtx, ghClient, repoID, repo.Owner, repo.Name)
		if err != nil {
			log.Printf("setup-security: %s/%s: %v", repo.Owner, repo.Name, err)
			return
		}
		log.Printf("setup-security: %s/%s done — dependabot=%v secretscanning=%v codescanning=%v aifindings=%d errors=%v",
			repo.Owner, repo.Name,
			result.DependabotEnabled, result.SecretScanningEnabled,
			result.CodeScanningEnabled, result.AIFindingsCount, result.Errors)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message":    "Security setup started. AI scan is running in the background.",
		"info":       "Findings will appear shortly. Refresh your findings list in a minute.",
		"repository": repo.FullName,
	})
}

// listNotificationsHandler godoc
// @Summary     List notifications
// @Description Returns all notifications for the authenticated user
// @Tags        notifications
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} map[string]interface{}
// @Router      /api/v1/notifications [get]
func (s *Server) listNotificationsHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	items, err := s.notifRepo.ListUnread(c.Request.Context(), userID)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to retrieve notifications"})
		return
	}
	count, _ := s.notifRepo.UnreadCount(c.Request.Context(), userID)
	c.JSON(200, gin.H{"notifications": items, "unread_count": count})
}

// markNotificationReadHandler godoc
// @Summary     Mark notification as read
// @Tags        notifications
// @Security    BearerAuth
// @Param       id path string true "Notification UUID"
// @Success     200 {object} map[string]string
// @Router      /api/v1/notifications/{id}/read [post]
func (s *Server) markNotificationReadHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	if err := s.notifRepo.MarkRead(c.Request.Context(), c.Param("id"), userID); err != nil {
		c.JSON(500, gin.H{"error": "failed to mark notification read"})
		return
	}
	c.JSON(200, gin.H{"status": "read"})
}

// getNotificationSettingsHandler godoc
// @Summary     Get notification settings
// @Tags        notifications
// @Security    BearerAuth
// @Success     200 {object} map[string]interface{}
// @Router      /api/v1/notifications/settings [get]
func (s *Server) getNotificationSettingsHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	settings, err := s.notifSettingsRepo.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		// No settings yet — return empty defaults
		c.JSON(200, gin.H{
			"email_enabled":    false,
			"slack_enabled":    false,
			"discord_enabled":  false,
			"telegram_enabled": false,
		})
		return
	}
	c.JSON(200, gin.H{
		"email":               settings.Email,
		"slack_webhook_url":   settings.SlackWebhookURL,
		"discord_webhook_url": settings.DiscordWebhookURL,
		"telegram_chat_id":    settings.TelegramChatID,
		"email_enabled":       settings.EmailEnabled,
		"slack_enabled":       settings.SlackEnabled,
		"discord_enabled":     settings.DiscordEnabled,
		"telegram_enabled":    settings.TelegramEnabled,
	})
}

// saveNotificationSettingsHandler godoc
// @Summary     Save notification settings
// @Tags        notifications
// @Security    BearerAuth
// @Success     200 {object} map[string]string
// @Router      /api/v1/notifications/settings [post]
func (s *Server) saveNotificationSettingsHandler(c *gin.Context) {
	userID := c.GetString("user_id")

	var body struct {
		Email             string `json:"email"`
		SlackWebhookURL   string `json:"slack_webhook_url"`
		DiscordWebhookURL string `json:"discord_webhook_url"`
		TelegramBotToken  string `json:"telegram_bot_token"`
		TelegramChatID    string `json:"telegram_chat_id"`
		EmailEnabled      bool   `json:"email_enabled"`
		SlackEnabled      bool   `json:"slack_enabled"`
		DiscordEnabled    bool   `json:"discord_enabled"`
		TelegramEnabled   bool   `json:"telegram_enabled"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	settings := &database.Settings{
		UserID:            userID,
		Email:             body.Email,
		SlackWebhookURL:   body.SlackWebhookURL,
		DiscordWebhookURL: body.DiscordWebhookURL,
		TelegramBotToken:  body.TelegramBotToken,
		TelegramChatID:    body.TelegramChatID,
		EmailEnabled:      body.EmailEnabled,
		SlackEnabled:      body.SlackEnabled,
		DiscordEnabled:    body.DiscordEnabled,
		TelegramEnabled:   body.TelegramEnabled,
	}

	if err := s.notifSettingsRepo.Upsert(c.Request.Context(), settings); err != nil {
		c.JSON(500, gin.H{"error": "failed to save settings"})
		return
	}

	c.JSON(200, gin.H{"message": "notification settings saved"})
}

// approvalCallbackHandler godoc
// @Summary     Handle approval/decline from notification channels
// @Description Public endpoint — validates signed JWT from email/Slack/Discord/Telegram button
// @Tags        notifications
// @Param       token query string true "Signed approval JWT"
// @Success     200 {object} map[string]string
// @Failure     400 {object} map[string]string
// @Router      /api/v1/approve [get]
func (s *Server) approvalCallbackHandler(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing token"})
		return
	}

	claims, err := notifications.ValidateApprovalToken(tokenStr, s.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired approval link"})
		return
	}

	switch claims.Action {
	case notifications.ActionApprove:
		_, err = s.remediationRepo.ApproveRemediation(c.Request.Context(), claims.FindingID, claims.UserID)
		if err != nil {
			log.Printf("approval callback: approve: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve remediation"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":    "Remediation approved. A pull request will be created shortly.",
			"finding_id": claims.FindingID,
			"channel":    claims.Channel,
		})

	case notifications.ActionDecline:
		err = s.remediationRepo.DeclineRemediation(c.Request.Context(), claims.FindingID, claims.UserID)
		if err != nil {
			log.Printf("approval callback: decline: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decline remediation"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":    "Remediation declined.",
			"finding_id": claims.FindingID,
			"channel":    claims.Channel,
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown action"})
	}
}
