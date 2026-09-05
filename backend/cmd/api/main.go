package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/lhilove/security-copilot/docs"
	"github.com/lhilove/security-copilot/internal/ai"
	"github.com/lhilove/security-copilot/internal/auth"
	"github.com/lhilove/security-copilot/internal/config"
	"github.com/lhilove/security-copilot/internal/database"
	"github.com/lhilove/security-copilot/internal/findings"
	gh "github.com/lhilove/security-copilot/internal/github"
	"github.com/lhilove/security-copilot/internal/repositories"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Security Copilot API
// @version         1.0
// @description     AI-powered application security remediation for GitHub repositories.
// @description     Aggregates GitHub Code Scanning, Dependabot, and Secret Scanning findings,
// @description     provides AI-generated business impact analysis and remediation proposals,
// @description     and manages a controlled developer approval workflow.

// @contact.name    Adewole Oluwapelumi
// @contact.url     https://linkedin.com/in/adewole-oluwapelumi
// @contact.email   pelumiade92@gmail.com

// @license.name    MIT

// @host            localhost:8080
// @BasePath        /

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 JWT token. Format: "Bearer <token>"

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

	// Data access
	userRepo := database.NewUserRepository(db)
	connRepo := database.NewGitHubConnectionRepository(db)
	repoRepo := database.NewRepositoryRepository(db)
	findingRepo := database.NewFindingRepository(db)
	remediationRepo := database.NewRemediationRepository(db)

	// Services
	srv := &Server{
		cfg:            cfg,
		authService:    auth.NewService(cfg, userRepo, connRepo),
		repoService:    repositories.NewService(repoRepo, connRepo, cfg.EncryptionKey),
		findingService: findings.NewService(findingRepo, repoRepo, connRepo, cfg.EncryptionKey),
		webhookSvc:     gh.NewWebhookHandler(repoRepo, findingRepo, findings.NormalizeSeverity),
		repoRepo:       repoRepo,
		findingRepo:    findingRepo,
		// aiProvider:      ai.NewNvidiaProvider(cfg.NvidiaAPIKey, cfg.NvidiaBaseURL, cfg.AIModel),
		aiProvider:      ai.NewMockProvider(),
		remediationRepo: remediationRepo,
	}

	router := gin.Default()
	router.SetTrustedProxies(nil)

	// Docs
	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Public routes
	router.GET("/health", srv.healthHandler)
	router.GET("/api/v1/auth/github", srv.githubLoginHandler)
	router.GET("/api/v1/auth/github/callback", srv.githubCallbackHandler)
	router.POST("/api/v1/webhooks/github", srv.webhookHandler)

	// Authenticated routes
	authorized := router.Group("/api/v1")
	authorized.Use(auth.RequireAuth(cfg.JWTSecret))

	authorized.GET("/repositories", srv.listRepositoriesHandler)
	authorized.POST("/repositories/:id/select", srv.selectRepositoryHandler)
	authorized.DELETE("/repositories/:id/select", srv.deselectRepositoryHandler)
	authorized.POST("/repositories/:id/sync", srv.syncFindingsHandler)
	authorized.GET("/repositories/:id/findings", srv.listFindingsHandler)
	authorized.GET("/repositories/:id/overview", srv.repositoryOverviewHandler)
	authorized.POST("/findings/:id/analyze", srv.analyzeFindingHandler)
	authorized.POST("/findings/:id/approve", srv.approveRemediationHandler)
	authorized.POST("/findings/:id/decline", srv.declineRemediationHandler)
	authorized.GET("/remediations", srv.listRemediationsHandler)
	authorized.POST("/findings/:id/pr", srv.createPRHandler)

	log.Println("Security Copilot API running on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
