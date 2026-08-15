package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/lhilove/security-copilot/internal/auth"
	"github.com/lhilove/security-copilot/internal/config"
	"github.com/lhilove/security-copilot/internal/database"
)

func main() {
	// load the configuration from environment variables or .env file
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// establish a connection to the PostgreSQL database using the provided database URL
	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Database connection established")

	githubAuth := auth.NewGitHubAuth(cfg) // Initialize GitHub OAuth with the loaded configuration

	router := gin.Default()

	//
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "security-copilot-api",
		})
	})

	// GitHub OAuth routes
	router.GET("/api/v1/auth/github", func(c *gin.Context) {
		// Generate a random state string for CSRF protection
		state, err := githubAuth.GenerateState()
		if err != nil {
			c.JSON(500, gin.H{
				"error": "failed to generate OAuth state",
			})
			return
		}
		// Set the state in a cookie for later validation
		c.SetCookie(
			"oauth_state", // Cookie name
			state,
			600, // 10 minutes
			"/", // Cookie path
			"",
			false,
			true,
		)

		c.Redirect(302, githubAuth.LoginURL(state))
	})

	// GitHub OAuth callback route
	router.GET("/api/v1/auth/github/callback", func(c *gin.Context) {
		code := c.Query("code")           // Get the authorization code from the query string
		receivedState := c.Query("state") // Get the state parameter from the query string

		// Validate the received state against the expected state
		if code == "" {
			c.JSON(400, gin.H{
				"error": "missing authorization code",
			})
			return
		}
		// Retrieve the expected state from the cookie
		expectedState, err := c.Cookie("oauth_state")
		if err != nil {
			c.JSON(400, gin.H{
				"error": "missing OAuth state",
			})
			return
		}

		// Validate the received state against the expected state
		if !auth.ValidateState(expectedState, receivedState) {
			c.JSON(400, gin.H{
				"error": "invalid OAuth state",
			})
			return
		}

		// State has been verified. It should not be reusable.
		c.SetCookie(
			"oauth_state",
			"",
			-1,  // Delete the cookie by setting a negative max age
			"/", // Cookie path
			"",
			false, // Not secure (for development purposes)
			true,  // HttpOnly
		)

		// Exchange the authorization code for an access token
		token, err := githubAuth.ExchangeCode(
			c.Request.Context(),
			code,
		)
		if err != nil {
			c.JSON(500, gin.H{
				"error": "failed to exchange authorization code",
			})
			return
		}

		// Retrieve the GitHub user information using the access token
		user, err := githubAuth.GetUser(
			c.Request.Context(),
			token,
		)
		if err != nil {
			c.JSON(500, gin.H{
				"error": "failed to retrieve GitHub user",
			})
			return
		}

		// return the successful response with the user information
		c.JSON(200, gin.H{
			"message": "GitHub connected successfully",
			"user":    user,
		})
	})

	log.Println("Security Copilot API running on :8080")

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
