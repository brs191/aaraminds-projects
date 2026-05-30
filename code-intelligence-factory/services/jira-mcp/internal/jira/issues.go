package jira

import (
	"context"
	"net/url"
	"strings"
)

// Issue is the subset of a Jira issue CIF cares about. Fields is left as a raw
// map so callers pull only what they need without coupling to every field type.
type Issue struct {
	ID     string         `json:"id,omitempty"`
	Key    string         `json:"key,omitempty"`
	Self   string         `json:"self,omitempty"`
	Fields map[string]any `json:"fields,omitempty"`
}

// CreateIssueInput describes a new issue. CustomFields keys are raw Jira field
// IDs (e.g. "customfield_10031"); resolve them per instance via GET /rest/api/3/field.
type CreateIssueInput struct {
	ProjectKey   string
	IssueType    string // e.g. "Story", "Bug"
	Summary      string
	Description  string
	CustomFields map[string]any
}

// CreateIssue -> POST /rest/api/3/issue
func (c *Client) CreateIssue(ctx context.Context, in CreateIssueInput) (*Issue, error) {
	fields := map[string]any{
		"project":   map[string]string{"key": in.ProjectKey},
		"issuetype": map[string]string{"name": in.IssueType},
		"summary":   in.Summary,
	}
	if in.Description != "" {
		fields["description"] = adfDoc(in.Description)
	}
	for k, v := range in.CustomFields {
		fields[k] = v
	}
	var out Issue
	if err := c.do(ctx, "POST", "/rest/api/3/issue", map[string]any{"fields": fields}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetIssue -> GET /rest/api/3/issue/{key}
func (c *Client) GetIssue(ctx context.Context, key string, fields []string) (*Issue, error) {
	path := "/rest/api/3/issue/" + url.PathEscape(key)
	if len(fields) > 0 {
		path += "?fields=" + url.QueryEscape(strings.Join(fields, ","))
	}
	var out Issue
	if err := c.do(ctx, "GET", path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchResult is the token-paginated response of /search/jql.
// NOTE: Jira removed `total` from this endpoint; iterate using NextPageToken
// until IsLast is true rather than computing pages from a count.
type SearchResult struct {
	Issues        []Issue `json:"issues"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
	IsLast        bool    `json:"isLast"`
}

// SearchJQL -> POST /rest/api/3/search/jql
//
// This replaces the removed GET /rest/api/3/search endpoint (shut down in
// Oct 2025). Pagination is cursor-based via nextPageToken.
func (c *Client) SearchJQL(ctx context.Context, jql string, fields []string, nextPageToken string, maxResults int) (*SearchResult, error) {
	if maxResults <= 0 || maxResults > 100 {
		maxResults = 50
	}
	body := map[string]any{
		"jql":        jql,
		"maxResults": maxResults,
	}
	if len(fields) > 0 {
		body["fields"] = fields
	}
	if nextPageToken != "" {
		body["nextPageToken"] = nextPageToken
	}
	var out SearchResult
	if err := c.do(ctx, "POST", "/rest/api/3/search/jql", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TransitionIssue -> POST /rest/api/3/issue/{key}/transitions
// transitionID comes from GET /rest/api/3/issue/{key}/transitions.
func (c *Client) TransitionIssue(ctx context.Context, key, transitionID string) error {
	path := "/rest/api/3/issue/" + url.PathEscape(key) + "/transitions"
	body := map[string]any{"transition": map[string]string{"id": transitionID}}
	return c.do(ctx, "POST", path, body, nil)
}

// LinkIssues -> POST /rest/api/3/issueLink
// linkType is a link-type name such as "Relates", "Blocks", "Cloners".
func (c *Client) LinkIssues(ctx context.Context, inwardKey, outwardKey, linkType string) error {
	body := map[string]any{
		"type":         map[string]string{"name": linkType},
		"inwardIssue":  map[string]string{"key": inwardKey},
		"outwardIssue": map[string]string{"key": outwardKey},
	}
	return c.do(ctx, "POST", "/rest/api/3/issueLink", body, nil)
}

// adfDoc wraps plain text in a minimal Atlassian Document Format document, which
// the v3 API requires for rich-text fields such as description.
func adfDoc(text string) map[string]any {
	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{"type": "text", "text": text},
				},
			},
		},
	}
}
