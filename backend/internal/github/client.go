package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// GetFileContent fetches the content and SHA of a file from a GitHub repository.
// The SHA is required for updating the file via the Contents API.
func (c *Client) GetFileContent(ctx context.Context, owner, repo, path string) (content string, sha string, err error) {
	apiPath := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)

	resp, err := c.do(ctx, http.MethodGet, apiPath)
	if err != nil {
		return "", "", fmt.Errorf("fetch file content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", "", fmt.Errorf("file not found: %s", path)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var result struct {
		Content  string `json:"content"`
		SHA      string `json:"sha"`
		Encoding string `json:"encoding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("decode file content: %w", err)
	}

	if result.Encoding != "base64" {
		return "", "", fmt.Errorf("unexpected encoding: %s", result.Encoding)
	}

	// GitHub returns base64 with newlines, strip them before decoding
	cleaned := strings.ReplaceAll(result.Content, "\n", "")
	decoded, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return "", "", fmt.Errorf("decode base64 content: %w", err)
	}

	return string(decoded), result.SHA, nil
}

// ExtractCodeContext returns the lines around a specific line number for AI context.
// It returns up to contextLines before and after the target line.
func ExtractCodeContext(content string, lineNumber int, contextLines int) string {
	if lineNumber <= 0 {
		return ""
	}

	lines := strings.Split(content, "\n")
	total := len(lines)

	start := lineNumber - contextLines - 1
	if start < 0 {
		start = 0
	}

	end := lineNumber + contextLines
	if end > total {
		end = total
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		sb.WriteString(fmt.Sprintf("%d: %s\n", i+1, lines[i]))
	}

	return sb.String()
}

// CreateBranch creates a new branch from the default branch of a repository.
func (c *Client) CreateBranch(ctx context.Context, owner, repo, branchName, baseSHA string) error {
	path := fmt.Sprintf("/repos/%s/%s/git/refs", owner, repo)

	body := map[string]string{
		"ref": "refs/heads/" + branchName,
		"sha": baseSHA,
	}

	bodyBytes, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create branch request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("create branch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create branch returned status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

// GetDefaultBranchSHA returns the SHA of the latest commit on the default branch.
func (c *Client) GetDefaultBranchSHA(ctx context.Context, owner, repo string) (string, string, error) {
	path := fmt.Sprintf("/repos/%s/%s", owner, repo)

	resp, err := c.do(ctx, http.MethodGet, path)
	if err != nil {
		return "", "", fmt.Errorf("get repo: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("decode repo: %w", err)
	}

	// Get the SHA of the default branch
	refPath := fmt.Sprintf("/repos/%s/%s/git/refs/heads/%s", owner, repo, result.DefaultBranch)
	resp2, err := c.do(ctx, http.MethodGet, refPath)
	if err != nil {
		return "", "", fmt.Errorf("get ref: %w", err)
	}
	defer resp2.Body.Close()

	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&ref); err != nil {
		return "", "", fmt.Errorf("decode ref: %w", err)
	}

	return result.DefaultBranch, ref.Object.SHA, nil
}

// UpdateFile commits a file change to a branch.
func (c *Client) UpdateFile(ctx context.Context, owner, repo, path, message, content, sha, branch string) error {
	apiPath := fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path)

	encoded := base64.StdEncoding.EncodeToString([]byte(content))

	body := map[string]string{
		"message": message,
		"content": encoded,
		"sha":     sha,
		"branch":  branch,
	}

	bodyBytes, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, baseURL+apiPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("update file request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("update file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update file returned status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

// PullRequest represents a created GitHub pull request.
type PullRequest struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	Title   string `json:"title"`
}

// CreatePullRequest opens a pull request from a branch.
func (c *Client) CreatePullRequest(ctx context.Context, owner, repo, title, body, head, base string) (*PullRequest, error) {
	apiPath := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)

	payload := map[string]string{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  base,
	}

	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+apiPath, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create pr request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("create pr: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create pr returned status %d: %s", resp.StatusCode, string(b))
	}

	var pr PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("decode pr: %w", err)
	}

	return &pr, nil
}
