package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// CodeScanningAlert represents a GitHub code scanning alert.
type CodeScanningAlert struct {
	Number int    `json:"number"`
	State  string `json:"state"`
	Rule   struct {
		ID          string `json:"id"`
		Description string `json:"description"`
		Severity    string `json:"severity"`
	} `json:"rule"`
	MostRecentInstance struct {
		Location struct {
			Path      string `json:"path"`
			StartLine int    `json:"start_line"`
		} `json:"location"`
	} `json:"most_recent_instance"`
	RawData map[string]any
}

// DependabotAlert represents a GitHub Dependabot alert.
type DependabotAlert struct {
	Number                int    `json:"number"`
	State                 string `json:"state"`
	SecurityVulnerability struct {
		Package struct {
			Name string `json:"name"`
		} `json:"package"`
		Severity            string `json:"severity"`
		FirstPatchedVersion *struct {
			Identifier string `json:"identifier"`
		} `json:"first_patched_version"`
	} `json:"security_vulnerability"`
	SecurityAdvisory struct {
		Summary     string `json:"summary"`
		Description string `json:"description"`
		CVEId       string `json:"cve_id"`
	} `json:"security_advisory"`
	RawData map[string]any
}

// SecretScanningAlert represents a GitHub secret scanning alert.
type SecretScanningAlert struct {
	Number     int    `json:"number"`
	State      string `json:"state"`
	SecretType string `json:"secret_type"`
	Secret     string `json:"secret"`
	RawData    map[string]any
}

// ListCodeScanningAlerts fetches all code scanning alerts for a repository.
func (c *Client) ListCodeScanningAlerts(ctx context.Context, owner, repo string) ([]CodeScanningAlert, error) {
	var all []CodeScanningAlert
	page := 1

	for {
		path := fmt.Sprintf("/repos/%s/%s/code-scanning/alerts?per_page=100&page=%d&state=open", owner, repo, page)

		resp, err := c.do(ctx, http.MethodGet, path)
		if err != nil {
			return nil, fmt.Errorf("fetch code scanning alerts page %d: %w", page, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			// Code scanning not enabled for this repo
			return nil, nil
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
		}

		var rawAlerts []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&rawAlerts); err != nil {
			return nil, fmt.Errorf("decode code scanning alerts: %w", err)
		}

		var pageAlerts []CodeScanningAlert
		for _, raw := range rawAlerts {
			data, _ := json.Marshal(raw)
			var alert CodeScanningAlert
			json.Unmarshal(data, &alert)
			alert.RawData = raw
			pageAlerts = append(pageAlerts, alert)
		}

		all = append(all, pageAlerts...)

		if len(pageAlerts) < 100 {
			break
		}
		page++
	}

	return all, nil
}

// ListDependabotAlerts fetches all Dependabot alerts for a repository.
func (c *Client) ListDependabotAlerts(ctx context.Context, owner, repo string) ([]DependabotAlert, error) {
	var all []DependabotAlert
	path := fmt.Sprintf("/repos/%s/%s/dependabot/alerts?per_page=100&state=open", owner, repo)

	for {
		resp, err := c.do(ctx, http.MethodGet, path)
		if err != nil {
			return nil, fmt.Errorf("fetch dependabot alerts: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("github api returned status %d: %s", resp.StatusCode, string(body))
		}

		var rawAlerts []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&rawAlerts); err != nil {
			return nil, fmt.Errorf("decode dependabot alerts: %w", err)
		}

		var pageAlerts []DependabotAlert
		for _, raw := range rawAlerts {
			data, _ := json.Marshal(raw)
			var alert DependabotAlert
			json.Unmarshal(data, &alert)
			alert.RawData = raw
			pageAlerts = append(pageAlerts, alert)
		}

		all = append(all, pageAlerts...)

		// Dependabot uses Link header for cursor-based pagination
		linkHeader := resp.Header.Get("Link")
		nextURL := extractNextURL(linkHeader)
		if nextURL == "" {
			break
		}

		// Use the full next URL directly
		path = strings.TrimPrefix(nextURL, baseURL)
	}

	return all, nil
}

// ListSecretScanningAlerts fetches all secret scanning alerts for a repository.
func (c *Client) ListSecretScanningAlerts(ctx context.Context, owner, repo string) ([]SecretScanningAlert, error) {
	var all []SecretScanningAlert
	page := 1

	for {
		path := fmt.Sprintf("/repos/%s/%s/secret-scanning/alerts?per_page=100&page=%d&state=open", owner, repo, page)

		resp, err := c.do(ctx, http.MethodGet, path)
		if err != nil {
			return nil, fmt.Errorf("fetch secret scanning alerts page %d: %w", page, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
		}

		var rawAlerts []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&rawAlerts); err != nil {
			return nil, fmt.Errorf("decode secret scanning alerts: %w", err)
		}

		var pageAlerts []SecretScanningAlert
		for _, raw := range rawAlerts {
			data, _ := json.Marshal(raw)
			var alert SecretScanningAlert
			json.Unmarshal(data, &alert)
			alert.RawData = raw
			pageAlerts = append(pageAlerts, alert)
		}

		all = append(all, pageAlerts...)

		if len(pageAlerts) < 100 {
			break
		}
		page++
	}

	return all, nil
}

// extractNextURL parses the Link header and returns the next page URL if present.
func extractNextURL(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}
	// Link header format: <https://api.github.com/...>; rel="next", <...>; rel="last"
	parts := strings.Split(linkHeader, ",")
	for _, part := range parts {
		sections := strings.Split(strings.TrimSpace(part), ";")
		if len(sections) == 2 && strings.TrimSpace(sections[1]) == `rel="next"` {
			url := strings.TrimSpace(sections[0])
			return strings.Trim(url, "<>")
		}
	}
	return ""
}
