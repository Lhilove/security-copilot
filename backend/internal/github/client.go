package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const baseURL = "https://api.github.com"

type Client struct {
	accessToken string
	httpClient  *http.Client
}

func NewClient(accessToken string) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient:  &http.Client{},
	}
}

func (c *Client) do(ctx context.Context, method, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	return c.httpClient.Do(req)
}

// Repository represents a GitHub repository.
type Repository struct {
	ID    int64 `json:"id"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
}

// ListRepositories fetches all repositories for the authenticated user, handling pagination.
func (c *Client) ListRepositories(ctx context.Context) ([]Repository, error) {
	var all []Repository
	page := 1

	for {
		path := fmt.Sprintf("/user/repos?per_page=100&page=%d&sort=updated", page)

		resp, err := c.do(ctx, http.MethodGet, path)
		if err != nil {
			return nil, fmt.Errorf("fetch repositories page %d: %w", page, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("github api returned status %d on page %d", resp.StatusCode, page)
		}

		var repos []Repository
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			return nil, fmt.Errorf("decode repositories page %d: %w", page, err)
		}

		all = append(all, repos...)

		// GitHub returns fewer than per_page results on the last page
		if len(repos) < 100 {
			break
		}

		page++
	}

	return all, nil
}
