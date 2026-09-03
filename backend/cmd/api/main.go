package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lhilove/security-copilot/internal/auth"
	"github.com/lhilove/security-copilot/internal/config"
	"github.com/lhilove/security-copilot/internal/database"
	"github.com/lhilove/security-copilot/internal/findings"
	gh "github.com/lhilove/security-copilot/internal/github"
	"github.com/lhilove/security-copilot/internal/repositories"
	"github.com/lhilove/security-copilot/internal/risk"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if err := database.Migrate(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	log.Println("Database migrations applied")

	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("Database connection established")

	// Repositories (data access)
	userRepo := database.NewUserRepository(db)
	connRepo := database.NewGitHubConnectionRepository(db)
	repoRepo := database.NewRepositoryRepository(db)
	findingRepo := database.NewFindingRepository(db)

	// Services (business logic)
	authService := auth.NewService(cfg, userRepo, connRepo)
	repoService := repositories.NewService(repoRepo, connRepo, cfg.EncryptionKey)
	findingService := findings.NewService(findingRepo, repoRepo, connRepo, cfg.EncryptionKey)
	webhookHandler := gh.NewWebhookHandler(repoRepo, findingRepo, findings.NormalizeSeverity)

	router := gin.Default()
	router.SetTrustedProxies(nil)

	// Public routes
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "security-copilot-api"})
	})

	router.GET("/api/v1/auth/github", func(c *gin.Context) {
		state, err := authService.GitHubAuth().GenerateState()
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
		c.Redirect(302, authService.GitHubAuth().LoginURL(state))
	})

	router.GET("/api/v1/auth/github/callback", func(c *gin.Context) {
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

		result, err := authService.ConnectGitHub(c.Request.Context(), code)
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
	})

	router.POST("/api/v1/webhooks/github", func(c *gin.Context) {
		body, err := gh.ReadWebhookBody(c.Request)
		if err != nil {
			c.JSON(400, gin.H{"error": "failed to read request body"})
			return
		}

		if err := gh.VerifyWebhookSignature(body, c.GetHeader("X-Hub-Signature-256"), cfg.GitHubWebhookSecret); err != nil {
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
			if err := webhookHandler.HandleCodeScanning(c.Request.Context(), event); err != nil {
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
			if err := webhookHandler.HandleDependabot(c.Request.Context(), event); err != nil {
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
			if err := webhookHandler.HandleSecretScanning(c.Request.Context(), event); err != nil {
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
	})

	// Authenticated routes
	authorized := router.Group("/api/v1")
	authorized.Use(auth.RequireAuth(cfg.JWTSecret))

	authorized.GET("/repositories", func(c *gin.Context) {
		userID := c.GetString("user_id")
		repos, err := repoService.SyncRepositories(c.Request.Context(), userID)
		if err != nil {
			log.Printf("sync repositories: %v", err)
			c.JSON(500, gin.H{"error": "failed to fetch repositories"})
			return
		}
		c.JSON(200, gin.H{"repositories": repos, "count": len(repos)})
	})

	authorized.POST("/repositories/:id/select", func(c *gin.Context) {
		userID := c.GetString("user_id")
		if err := repoService.SelectRepository(c.Request.Context(), userID, c.Param("id")); err != nil {
			log.Printf("select repository: %v", err)
			c.JSON(404, gin.H{"error": "repository not found"})
			return
		}
		c.JSON(200, gin.H{"message": "repository selected for monitoring"})
	})

	authorized.DELETE("/repositories/:id/select", func(c *gin.Context) {
		userID := c.GetString("user_id")
		if err := repoService.DeselectRepository(c.Request.Context(), userID, c.Param("id")); err != nil {
			log.Printf("deselect repository: %v", err)
			c.JSON(404, gin.H{"error": "repository not found"})
			return
		}
		c.JSON(200, gin.H{"message": "repository deselected"})
	})

	authorized.POST("/repositories/:id/sync", func(c *gin.Context) {
		userID := c.GetString("user_id")
		repoID := c.Param("id")
		count, err := findingService.SyncRepository(c.Request.Context(), userID, repoID)
		if err != nil {
			log.Printf("sync findings: %v", err)
			c.JSON(500, gin.H{"error": "failed to sync findings"})
			return
		}
		c.JSON(200, gin.H{"synced": count})
	})

	authorized.GET("/repositories/:id/findings", func(c *gin.Context) {
		userID := c.GetString("user_id")
		f, err := findingService.GetFindings(c.Request.Context(), userID, c.Param("id"))
		if err != nil {
			log.Printf("get findings: %v", err)
			c.JSON(500, gin.H{"error": "failed to retrieve findings"})
			return
		}
		c.JSON(200, gin.H{"findings": f, "count": len(f)})
	})

	authorized.GET("/repositories/:id/overview", func(c *gin.Context) {
		userID := c.GetString("user_id")
		repoID := c.Param("id")

		repo, err := repoRepo.GetRepositoryByID(c.Request.Context(), userID, repoID)
		if err != nil {
			c.JSON(404, gin.H{"error": "repository not found"})
			return
		}

		summary, err := findingService.GetSummary(c.Request.Context(), userID, repoID)
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
	})

	log.Println("Security Copilot API running on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
