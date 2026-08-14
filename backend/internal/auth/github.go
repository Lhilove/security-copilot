package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lhilove/security-copilot/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// GitHub auth
type GitHubAuth struct {
	config *oauth2.Config
}

// NewGitHubAuth creates a new GitHub auth instance
func NewGitHubAuth(cfg *config.Config) *GitHubAuth {
	return &GitHubAuth{
		config: &oauth2.Config{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			RedirectURL:  cfg.GitHubRedirectURL,
			Endpoint:     github.Endpoint,
			Scopes: []string{ //
				"read:user",
				"user:email",
				"repo",
			},
		},
	}
}

// GenerateState generates a random state string for CSRF protection
func (g *GitHubAuth) GenerateState() (string, error) {
	// Generate a random 32-byte state string
	stateBytes := make([]byte, 32)

	// generating 256 bits of randomness for every OAuth attempt
	if _, err := rand.Read(stateBytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(stateBytes), nil // Return the encoded state string
}

// LoginURL returns the GitHub login URL with the provided state parameter
func (g *GitHubAuth) LoginURL(state string) string {
	return g.config.AuthCodeURL(state) // Return the GitHub login URL with the provided state parameter
}

// ValidateState compares the expected and received state strings in constant time to prevent timing attacks
func ValidateState(expected, received string) bool {
	if expected == "" || received == "" {
		return false
	}

	// Compare the expected and received state strings in constant time
	return subtle.ConstantTimeCompare(
		[]byte(expected),
		[]byte(received),
	) == 1
}

// ExchangeCode exchanges the authorization code for an access token
func (g *GitHubAuth) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.config.Exchange(ctx, code)
}

// GetUser retrieves the GitHub user information using the access token
func (g *GitHubAuth) GetUser(ctx context.Context, token *oauth2.Token) (map[string]any, error) {
	client := g.config.Client(ctx, token) // Create an HTTP client using the access token

	// Make a request to the GitHub API to get user information
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check if the response status is OK
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var user map[string]any // Use map[string]any to hold the user data

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return user, nil
}
