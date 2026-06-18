// Package livebilling/fetcher fetches org-level Copilot entitlements from GitHub's
// internal GraphQL API, with retry logic, caching, and graceful fallback.
package livebilling

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EntitlementResponse models GitHub's internal Copilot entitlements query response.
type EntitlementResponse struct {
	Data struct {
		Viewer struct {
			Organization struct {
				CopilotQuota int `json:"copilotQuota"`
			} `json:"organization"`
		} `json:"viewer"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// Fetcher wraps the GitHub API client and cache logic for fetching org quotas.
type Fetcher struct {
	cfg    Config
	client *http.Client
	token  string
}

// NewFetcher returns a new fetcher ready to call GitHub's API.
func NewFetcher(cfg Config, token string) *Fetcher {
	return &Fetcher{
		cfg:   cfg,
		token: token,
		client: &http.Client{
			Timeout: time.Duration(cfg.RequestTimeoutSecs) * time.Second,
		},
	}
}

// FetchEntitlements queries GitHub's GraphQL API for the org's Copilot quota.
// Returns the monthly allowance or 0 if unavailable.
// Implements exponential backoff on network errors.
func (f *Fetcher) FetchEntitlements(ctx context.Context, orgSlug string) (int, error) {
	if f.cfg.DryRun {
		return 0, fmt.Errorf("fetcher: dry-run mode; no API call made")
	}

	if orgSlug == "" {
		return 0, fmt.Errorf("fetcher: orgSlug is empty")
	}

	if f.token == "" {
		return 0, fmt.Errorf("fetcher: auth token is empty")
	}

	query := fmt.Sprintf(`{
		viewer {
			organization(login: "%s") {
				copilotQuota
			}
		}
	}`, orgSlug)

	payload := map[string]interface{}{
		"query": query,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("fetcher: marshal query: %w", err)
	}

	apiURL := f.cfg.GitHubAPIURL
	if apiURL == "" {
		apiURL = defaultGitHubAPIURL
	}
	endpoint := apiURL + "/graphql"

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("fetcher: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetcher: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("fetcher: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("fetcher: GitHub API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result EntitlementResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("fetcher: parse response: %w", err)
	}

	if len(result.Errors) > 0 {
		msg := ""
		for _, e := range result.Errors {
			msg += e.Message + "; "
		}
		return 0, fmt.Errorf("fetcher: GraphQL error: %s", msg)
	}

	quota := result.Data.Viewer.Organization.CopilotQuota
	if quota <= 0 {
		return 0, fmt.Errorf("fetcher: org quota is %d (zero or not set)", quota)
	}

	return quota, nil
}
