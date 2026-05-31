// Package jira is a thin client for the Jira Cloud REST API v3.
//
// It handles auth (API token via Basic), JSON encode/decode, actionable errors,
// and bounded 429 rate-limit backoff. It deliberately exposes only the
// operations the Code Intelligence Factory needs — not the whole Jira API.
package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Client talks to one Jira Cloud site.
type Client struct {
	baseURL string
	email   string
	token   string
	http    *http.Client
}

// NewClient builds a client for https://<site>.atlassian.net using an account
// email + API token (Basic auth). For production prefer OAuth 2.0 (3LO): swap
// the Authorization header in do() for "Bearer <access_token>".
func NewClient(baseURL, email, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		email:   email,
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// BrowseURL returns the human browser URL for an issue key.
func (c *Client) BrowseURL(key string) string {
	return c.baseURL + "/browse/" + key
}

// APIError carries a Jira error response in an actionable form.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("jira API error: HTTP %d: %s", e.Status, e.Body)
}

// do performs a JSON request with bounded retry on HTTP 429. body may be nil;
// out may be nil when no response body is expected (e.g. 204 No Content).
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var rdr io.Reader
		if body != nil {
			b, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("encode request body: %w", err)
			}
			rdr = bytes.NewReader(b)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		auth := base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.token))
		req.Header.Set("Authorization", "Basic "+auth)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http do: %w", err)
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = &APIError{Status: resp.StatusCode, Body: string(respBody)}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryAfter(resp.Header.Get("Retry-After"), attempt)):
			}
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return &APIError{Status: resp.StatusCode, Body: string(respBody)}
		}
		if out != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
		}
		return nil
	}
	return fmt.Errorf("exhausted retries: %w", lastErr)
}

// retryAfter honors the Retry-After header (seconds) or falls back to
// exponential backoff: 1s, 2s, 4s.
func retryAfter(header string, attempt int) time.Duration {
	if header != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(header)); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return time.Duration(1<<uint(attempt-1)) * time.Second
}
