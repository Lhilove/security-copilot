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
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Run migrations before opening the connection pool
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

	log.Println("Database connection established")

	githubAuth := auth.NewGitHubAuth(cfg)

	router := gin.Default()
	router.SetTrustedProxies(nil) // no proxies trusted in development

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

		// Delete the oauth_state cookie after validation
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

		// Upsert the user
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

		// Encrypt the token
		encryptedToken, err := crypto.Encrypt(cfg.EncryptionKey, []byte(token.AccessToken))
		if err != nil {
			c.JSON(500, gin.H{"error": "failed to encrypt token"})
			return
		}

		// Base64-encode for safe storage in TEXT column
		encodedToken := base64.StdEncoding.EncodeToString(encryptedToken)

		// Store the connection
		if err := connRepo.UpsertConnection(
			c.Request.Context(),
			dbUser.ID,
			encodedToken,
		); err != nil {
			c.JSON(500, gin.H{"error": "failed to save github connection"})
			return
		}

		// Store the connection
		if err := connRepo.UpsertConnection(
			c.Request.Context(),
			dbUser.ID,
			encodedToken,
		); err != nil {
			log.Printf("failed to save github connection: %v", err)
			c.JSON(500, gin.H{"error": "failed to save github connection"})
			return
		}

		c.JSON(200, gin.H{
			"message": "GitHub connected successfully",
			"user":    dbUser,
		})
	})

	log.Println("Security Copilot API running on :8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
