// Package generator — pr.go
//
// PR workflow: validates the TerraformPlan via ValidateBeforeEmit and, on a
// clean gate result, creates a GitHub PR in the AT&T infrastructure repository.
//
// Hard rule: GITHUB_TOKEN MUST NOT appear in any log line or error message.
// All errors redact the token string; the token is only placed in the
// Authorization header of the outbound HTTP request.
package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// ErrGateFailed is returned by CreatePR when validation.Approved == false.
var ErrGateFailed = errors.New("generate_topology: ValidateBeforeEmit gate failed — no PR created")

// GitHubClient abstracts GitHub PR creation for testability.
type GitHubClient interface {
	// CreatePull creates a PR in the infrastructure repository.
	// title: PR title string
	// body:  PR body Markdown
	// branch: source branch name (e.g. "att-nettopo/abc12345")
	// Returns the PR HTML URL on success.
	CreatePull(ctx context.Context, title, body, branch string) (string, error)
}

// RealGitHubClient calls the GitHub REST API.
// GITHUB_TOKEN and INFRA_REPO (owner/repo format) are read from env vars.
// [VERIFY V-04] INFRA_REPO: confirm AT&T infrastructure repository name.
// [VERIFY V-05] GitHub App token vs PAT — confirm which is provisioned.
type RealGitHubClient struct{}

// CreatePull creates a GitHub pull request via the REST API.
// The GITHUB_TOKEN is read from env and placed only in the Authorization header —
// it never appears in any error message or log line.
func (c *RealGitHubClient) CreatePull(ctx context.Context, title, body, branch string) (string, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return "", fmt.Errorf("GITHUB_TOKEN env var is not set")
	}
	infraRepo := os.Getenv("INFRA_REPO")
	if infraRepo == "" {
		return "", fmt.Errorf("INFRA_REPO env var is not set (expected owner/repo format)")
	}

	payload := map[string]string{
		"title": title,
		"body":  body,
		"head":  branch,
		"base":  "main",
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal PR payload: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/pulls", infraRepo)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("create PR request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("GitHub PR API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read GitHub PR response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Do NOT include response body in error — it may echo request fields.
		return "", fmt.Errorf("GitHub PR API returned status %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse GitHub PR response: %w", err)
	}

	htmlURL, ok := result["html_url"].(string)
	if !ok || htmlURL == "" {
		return "", fmt.Errorf("GitHub PR response missing html_url field")
	}
	return htmlURL, nil
}

// StubGitHubClient returns a deterministic PR URL and captures call arguments.
// Used in tests and when GENERATOR_MODE=stub.
type StubGitHubClient struct {
	LastTitle  string
	LastBody   string
	LastBranch string
	Called     bool
}

// CreatePull records arguments and returns the deterministic stub PR URL.
func (c *StubGitHubClient) CreatePull(_ context.Context, title, body, branch string) (string, error) {
	c.Called = true
	c.LastTitle = title
	c.LastBody = body
	c.LastBranch = branch
	return "https://github.com/att-infra/network/pull/42", nil
}

// CreatePR creates a GitHub PR if validation.Approved == true.
// Returns ErrGateFailed immediately if validation.Approved == false.
// intent is the original architect NL description (stored in the PR body).
func CreatePR(
	ctx context.Context,
	plan TerraformPlan,
	validation ValidationResult,
	intent string,
	ghClient GitHubClient,
) (string, error) {
	if !validation.Approved {
		return "", ErrGateFailed
	}

	// Derive branch name from the SpecHash prefix (8 hex chars).
	hashPrefix := plan.SpecHash
	if len(hashPrefix) > 8 {
		hashPrefix = hashPrefix[:8]
	}
	branch := "att-nettopo/" + hashPrefix
	title := fmt.Sprintf("[nettopo] %s — topology generation (validated)", hashPrefix)

	// Build sorted file-name list.
	fileNames := make([]string, 0, len(plan.Files))
	for fname := range plan.Files {
		fileNames = append(fileNames, fname)
	}
	sort.Strings(fileNames)

	var fileListSB strings.Builder
	for _, fname := range fileNames {
		fileListSB.WriteString(fmt.Sprintf("- `%s`\n", fname))
	}
	fileList := fileListSB.String()

	// Build advisory findings table (Medium / Low / Informational only).
	advisorySection := buildAdvisorySection(validation)

	registrySHA := plan.RegistrySnapshotSHA
	if registrySHA == "" {
		registrySHA = "N/A"
	}

	body := buildPRBody(intent, fileList, advisorySection, plan.SpecHash, registrySHA)

	return ghClient.CreatePull(ctx, title, body, branch)
}

// buildAdvisorySection renders the advisory findings table for the PR body.
func buildAdvisorySection(validation ValidationResult) string {
	var rows []string
	for _, f := range validation.Findings {
		if f.Severity == "Critical" || f.Severity == "High" {
			continue
		}
		rows = append(rows, fmt.Sprintf("| %s | %s | %s | %s |",
			f.Severity, f.Type, f.Resource, f.Evidence))
	}
	if len(rows) == 0 {
		return "No advisory findings."
	}
	var sb strings.Builder
	sb.WriteString("| Severity | Type | Resource | Evidence |\n")
	sb.WriteString("|----------|------|----------|----------|\n")
	for _, row := range rows {
		sb.WriteString(row + "\n")
	}
	return sb.String()
}

// buildPRBody constructs the Markdown PR body from its components.
func buildPRBody(intent, fileList, advisorySection, specHash, registrySHA string) string {
	return "## Network Topology Generation — Automated PR\n\n" +
		"**Intent:** " + intent + "\n\n" +
		"**Gate Result:** ✅ PASS — zero Critical/High findings\n\n" +
		"## Terraform Files\n" +
		fileList + "\n" +
		"## ValidateBeforeEmit Findings\n" +
		advisorySection + "\n\n" +
		"## Audit Trail\n" +
		"- SpecHash: `" + specHash + "`\n" +
		"- RegistrySnapshotSHA: `" + registrySHA + "`\n" +
		"- Generated by: azure-network-topology-reviewer (generate_topology MCP tool)\n\n" +
		"> ⚠️ Human approval required before `terraform apply`. This PR was created automatically.\n" +
		"> The AT&T network team must review and approve before merge.\n"
}
