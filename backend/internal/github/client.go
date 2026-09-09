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

// EnableDependabot enables Dependabot alerts for a repository.
func (c *Client) EnableDependabot(ctx context.Context, owner, repo string) error {
	path := fmt.Sprintf("/repos/%s/%s/vulnerability-alerts", owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create dependabot request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("enable dependabot: %w", err)
	}
	defer resp.Body.Close()

	// 204 = success, 403 = not allowed (org setting), both are acceptable
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusForbidden {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("enable dependabot returned status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

// EnableSecretScanning enables secret scanning for a repository.
func (c *Client) EnableSecretScanning(ctx context.Context, owner, repo string) error {
	path := fmt.Sprintf("/repos/%s/%s", owner, repo)

	body := map[string]any{
		"security_and_analysis": map[string]any{
			"secret_scanning": map[string]string{
				"status": "enabled",
			},
			"secret_scanning_push_protection": map[string]string{
				"status": "enabled",
			},
		},
	}

	bodyBytes, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create secret scanning request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("enable secret scanning: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusForbidden {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("enable secret scanning returned status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

// EnableCodeScanning creates a default CodeQL workflow in the repository.
// This is the only way to enable CodeQL via API — GitHub has no direct toggle.
func (c *Client) EnableCodeScanning(ctx context.Context, owner, repo, defaultBranch string) error {
	path := fmt.Sprintf("/repos/%s/%s/contents/.github/workflows/codeql.yml", owner, repo)

	workflow := fmt.Sprintf(`name: CodeQL Analysis
on:
  push:
    branches: [ "%s" ]
  pull_request:
    branches: [ "%s" ]
  schedule:
    - cron: '0 6 * * 1'

jobs:
  analyze:
    name: Analyze
    runs-on: ubuntu-latest
    permissions:
      actions: read
      contents: read
      security-events: write
    strategy:
      fail-fast: false
      matrix:
        language: ['javascript', 'python']
    steps:
    - name: Checkout repository
      uses: actions/checkout@v4
    - name: Initialize CodeQL
      uses: github/codeql-action/init@v3
      with:
        languages: ${{ matrix.language }}
    - name: Autobuild
      uses: github/codeql-action/autobuild@v3
    - name: Perform CodeQL Analysis
      uses: github/codeql-action/analyze@v3
`, defaultBranch, defaultBranch)

	encoded := base64.StdEncoding.EncodeToString([]byte(workflow))

	// Check if file already exists to get its SHA
	existingContent, existingSHA, err := c.GetFileContent(ctx, owner, repo, ".github/workflows/codeql.yml")
	_ = existingContent

	var bodyMap map[string]string
	if err == nil && existingSHA != "" {
		// File exists, update it
		bodyMap = map[string]string{
			"message": "chore: add CodeQL analysis workflow [Security Copilot]",
			"content": encoded,
			"sha":     existingSHA,
			"branch":  defaultBranch,
		}
	} else {
		// New file
		bodyMap = map[string]string{
			"message": "chore: add CodeQL analysis workflow [Security Copilot]",
			"content": encoded,
			"branch":  defaultBranch,
		}
	}

	bodyBytes, _ := json.Marshal(bodyMap)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create codeql workflow request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("enable code scanning: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("enable code scanning returned status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

// ListRepoFiles returns all file paths in a repository up to maxFiles.
// Used for AI direct scanning when CodeQL is not available.
func (c *Client) ListRepoFiles(ctx context.Context, owner, repo, treeSHA string) ([]string, error) {
	path := fmt.Sprintf("/repos/%s/%s/git/trees/%s?recursive=1", owner, repo, treeSHA)

	resp, err := c.do(ctx, http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("fetch file tree: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var result struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
		Truncated bool `json:"truncated"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode file tree: %w", err)
	}

	// Only return files (blobs), not directories, and only scannable extensions
	scannable := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
		".py": true, ".java": true, ".rb": true, ".php": true, ".cs": true,
		".cpp": true, ".c": true, ".h": true, ".rs": true, ".swift": true,
	}

	var files []string
	for _, item := range result.Tree {
		if item.Type != "blob" {
			continue
		}
		for ext := range scannable {
			if strings.HasSuffix(item.Path, ext) {
				files = append(files, item.Path)
				break
			}
		}
	}

	return files, nil
}
