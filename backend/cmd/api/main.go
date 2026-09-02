package main

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lhilove/security-copilot/internal/auth"
	"github.com/lhilove/security-copilot/internal/config"
	"github.com/lhilove/security-copilot/internal/crypto"
	"github.com/lhilove/security-copilot/internal/database"
	github "github.com/lhilove/security-copilot/internal/github"
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

	userRepo := database.NewUserRepository(db)
	connRepo := database.NewGitHubConnectionRepository(db)
	repoRepo := database.NewRepositoryRepository(db)

	log.Println("Database connection established")

	githubAuth := auth.NewGitHubAuth(cfg)

	router := gin.Default()
	router.SetTrustedProxies(nil)

	// Public routes
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "security-copilot-api",
		})
	})

	router.GET("/api/v1/auth/github", func(c *gin.Context) {
		state, err := githubAuth.GenerateState()
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
		c.Redirect(302, githubAuth.LoginURL(state))
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

		expectedState := cookie.Value

		if !auth.ValidateState(expectedState, receivedState) {
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

		token, err := githubAuth.ExchangeCode(c.Request.Context(), code)
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to exchange authorization code"})
			return
		}

		githubUser, err := githubAuth.GetUser(c.Request.Context(), token)
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to retrieve GitHub user"})
			return
		}

		dbUser, err := userRepo.UpsertUser(
			c.Request.Context(),
			githubUser.ID,
			githubUser.Login,
			githubUser.Email,
		)
		if err != nil {
			log.Printf("failed to save user: %v", err)
			c.JSON(500, gin.H{"error": "failed to save user"})
			return
		}

		encryptedToken, err := crypto.Encrypt(cfg.EncryptionKey, []byte(token.AccessToken))
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to encrypt token"})
			return
		}

		encodedToken := base64.StdEncoding.EncodeToString(encryptedToken)

		if err := connRepo.UpsertConnection(
			c.Request.Context(),
			dbUser.ID,
			encodedToken,
		); err != nil {
			log.Printf("failed to save github connection: %v", err)
			c.JSON(500, gin.H{"error": "failed to save github connection"})
			return
		}

		tokenString, err := auth.IssueToken(dbUser.ID, cfg.JWTSecret)
		if err != nil {
			log.Printf("failed to issue token: %v", err)
			c.JSON(500, gin.H{"error": "failed to issue token"})
			return
		}

		c.JSON(200, gin.H{
			"message": "GitHub connected successfully",
			"token":   tokenString,
			"user":    dbUser,
		})
	})

	// Authenticated routes
	authorized := router.Group("/api/v1")
	authorized.Use(auth.RequireAuth(cfg.JWTSecret))

	authorized.GET("/repositories", func(c *gin.Context) {
		userID := c.GetString("user_id")

		conn, err := connRepo.GetConnection(c.Request.Context(), userID)
		if err != nil {
			log.Printf("get connection: %v", err)
			c.JSON(500, gin.H{"error": "failed to retrieve github connection"})
			return
		}

		encryptedBytes, err := base64.StdEncoding.DecodeString(conn.AccessTokenEncrypted)
		if err != nil {
			log.Printf("decode token: %v", err)
			c.JSON(500, gin.H{"error": "failed to decode token"})
			return
		}

		tokenBytes, err := crypto.Decrypt(cfg.EncryptionKey, encryptedBytes)
		if err != nil {
			log.Printf("decrypt token: %v", err)
			c.JSON(500, gin.H{"error": "failed to decrypt token"})
			return
		}

		ghClient := github.NewClient(string(tokenBytes))
		ghRepos, err := ghClient.ListRepositories(c.Request.Context())
		if err != nil {
			log.Printf("list repositories: %v", err)
			c.JSON(500, gin.H{"error": "failed to fetch repositories"})
			return
		}

		var repos []database.Repository
		for _, r := range ghRepos {
			repos = append(repos, database.Repository{
				GitHubRepoID: r.ID,
				Owner:        r.Owner.Login,
				Name:         r.Name,
				FullName:     r.FullName,
				Private:      r.Private,
			})
		}

		if err := repoRepo.UpsertRepositories(c.Request.Context(), userID, repos); err != nil {
			log.Printf("upsert repositories: %v", err)
			c.JSON(500, gin.H{"error": "failed to save repositories"})
			return
		}

		stored, err := repoRepo.GetRepositoriesByUserID(c.Request.Context(), userID)
		if err != nil {
			log.Printf("get repositories: %v", err)
			c.JSON(500, gin.H{"error": "failed to retrieve repositories"})
			return
		}

		c.JSON(200, gin.H{
			"repositories": stored,
			"count":        len(stored),
		})
	})

	log.Println("Security Copilot API running on :8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
