package livebilling

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// mockHTTPClient simulates GitHub API responses for testing.
type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func TestFetchEntitlements_Success(t *testing.T) {
	cfg := Default()
	cfg.Enabled = true
	cfg.OrgSlug = "att-org"
	cfg.GitHubAPIURL = "https://api.github.com"

	// Mock a successful response
	mockResp := `{
		"data": {
			"viewer": {
				"organization": {
					"copilotQuota": 35000
				}
			}
		}
	}`

	fetcher := NewFetcher(cfg, "mock-token")
	// Replace the client with a mock
	fetcher.client = &http.Client{
		Transport: &mockTransport{
			responseBody: mockResp,
			statusCode:   200,
		},
	}

	quota, err := fetcher.FetchEntitlements(context.Background(), "att-org")
	if err != nil {
		t.Fatalf("FetchEntitlements failed: %v", err)
	}

	if quota != 35000 {
		t.Errorf("expected quota 35000, got %d", quota)
	}
}

func TestFetchEntitlements_DryRun(t *testing.T) {
	cfg := Default()
	cfg.Enabled = true
	cfg.DryRun = true
	cfg.OrgSlug = "att-org"

	fetcher := NewFetcher(cfg, "mock-token")
	_, err := fetcher.FetchEntitlements(context.Background(), "att-org")

	if err == nil {
		t.Fatal("expected error in dry-run mode")
	}
	if !strings.Contains(err.Error(), "dry-run") {
		t.Errorf("expected 'dry-run' in error, got: %v", err)
	}
}

func TestFetchEntitlements_NoOrgSlug(t *testing.T) {
	cfg := Default()
	cfg.Enabled = true

	fetcher := NewFetcher(cfg, "mock-token")
	_, err := fetcher.FetchEntitlements(context.Background(), "")

	if err == nil {
		t.Fatal("expected error with empty orgSlug")
	}
	if !strings.Contains(err.Error(), "orgSlug") {
		t.Errorf("expected 'orgSlug' in error, got: %v", err)
	}
}

func TestFetchEntitlements_NoToken(t *testing.T) {
	cfg := Default()
	cfg.Enabled = true
	cfg.OrgSlug = "att-org"

	fetcher := NewFetcher(cfg, "")
	_, err := fetcher.FetchEntitlements(context.Background(), "att-org")

	if err == nil {
		t.Fatal("expected error with empty token")
	}
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("expected 'token' in error, got: %v", err)
	}
}

func TestFetchEntitlements_GraphQLError(t *testing.T) {
	cfg := Default()
	cfg.Enabled = true
	cfg.OrgSlug = "att-org"

	mockResp := `{
		"errors": [
			{"message": "Authentication failed"}
		]
	}`

	fetcher := NewFetcher(cfg, "bad-token")
	fetcher.client = &http.Client{
		Transport: &mockTransport{
			responseBody: mockResp,
			statusCode:   200,
		},
	}

	_, err := fetcher.FetchEntitlements(context.Background(), "att-org")
	if err == nil {
		t.Fatal("expected GraphQL error")
	}
	if !strings.Contains(err.Error(), "GraphQL") {
		t.Errorf("expected 'GraphQL' in error, got: %v", err)
	}
}

func TestFetchEntitlements_HTTPError(t *testing.T) {
	cfg := Default()
	cfg.Enabled = true
	cfg.OrgSlug = "att-org"

	fetcher := NewFetcher(cfg, "mock-token")
	fetcher.client = &http.Client{
		Transport: &mockTransport{
			responseBody: "Unauthorized",
			statusCode:   401,
		},
	}

	_, err := fetcher.FetchEntitlements(context.Background(), "att-org")
	if err == nil {
		t.Fatal("expected HTTP error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected '401' in error, got: %v", err)
	}
}

// mockTransport implements http.RoundTripper for mocking HTTP responses.
type mockTransport struct {
	responseBody string
	statusCode   int
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(m.responseBody)),
		Header:     make(http.Header),
	}, nil
}
