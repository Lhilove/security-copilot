package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lhilove/security-copilot/internal/ai"
	"github.com/lhilove/security-copilot/internal/auth"
	gh "github.com/lhilove/security-copilot/internal/github"
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

	c.JSON(200, gin.H{
		"message": "GitHub connected successfully",
		"token":   result.Token,
		"user":    result.User,
	})
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

	result, err := s.aiProvider.Analyze(c.Request.Context(), req)
	if err != nil {
		log.Printf("ai analyze: %v", err)
		c.JSON(500, gin.H{"error": "analysis failed"})
		return
	}

	c.JSON(200, gin.H{
		"finding_id":    findingID,
		"what":          result.What,
		"risk":          result.Risk,
		"fix":           result.Fix,
		"proposed_code": result.ProposedCode,
		"can_auto_fix":  result.CanAutoFix,
	})
}
