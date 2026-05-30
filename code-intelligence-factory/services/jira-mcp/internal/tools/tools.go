// Package tools registers the Code Intelligence Factory's Jira MCP tools.
//
// The surface is deliberately scoped to what CIF needs — create/read/search/
// transition/link issues — plus one composite workflow tool that creates a
// story AND writes its BR-/US- traceability fields in a single call (the kind
// of high-signal tool a generic Jira server would not give you).
//
// Tool-annotation note: the MCP spec defines hints (readOnlyHint, destructiveHint,
// idempotentHint, openWorldHint) and mcp-builder recommends setting them. They
// are shown commented below; confirm the exact field shapes for your pinned SDK
// with `go doc github.com/modelcontextprotocol/go-sdk/mcp.ToolAnnotations`, then
// uncomment. They are omitted from the compiled surface only to keep this
// scaffold building against the API surface that was verified.
package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/aaraminds/code-intelligence-factory/services/jira-mcp/internal/jira"
)

// Config carries instance-specific custom-field IDs resolved from env at startup.
type Config struct {
	FieldUSID   string // Jira custom-field id storing the US- story id, e.g. customfield_10031
	FieldBRLink string // Jira custom-field id storing the BR- requirement id, e.g. customfield_10032
}

// Register wires every CIF Jira tool onto the server.
func Register(server *mcp.Server, client *jira.Client, cfg Config) {
	registerCreateIssue(server, client)
	registerGetIssue(server, client)
	registerSearch(server, client)
	registerTransition(server, client)
	registerLink(server, client)
	registerCreateStoryWithTrace(server, client, cfg)
}

// ---- jira_create_issue ----

type createIssueInput struct {
	ProjectKey  string `json:"project_key" jsonschema:"Jira project key, e.g. PROJ"`
	IssueType   string `json:"issue_type" jsonschema:"Issue type name, e.g. Story or Bug"`
	Summary     string `json:"summary" jsonschema:"One-line issue summary"`
	Description string `json:"description,omitempty" jsonschema:"Plain-text description; wrapped in ADF automatically"`
}

type createIssueOutput struct {
	Key string `json:"key" jsonschema:"Created issue key, e.g. PROJ-123"`
	URL string `json:"url" jsonschema:"Browser URL of the created issue"`
}

func registerCreateIssue(server *mcp.Server, client *jira.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "jira_create_issue",
		Description: "Create a Jira issue (Story, Bug, Task, …) in a project. Returns the new issue key.",
		// Annotations: &mcp.ToolAnnotations{OpenWorldHint: true}, // not read-only
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createIssueInput) (*mcp.CallToolResult, createIssueOutput, error) {
		if in.ProjectKey == "" || in.IssueType == "" || in.Summary == "" {
			return nil, createIssueOutput{}, fmt.Errorf("project_key, issue_type, and summary are required")
		}
		iss, err := client.CreateIssue(ctx, jira.CreateIssueInput{
			ProjectKey:  in.ProjectKey,
			IssueType:   in.IssueType,
			Summary:     in.Summary,
			Description: in.Description,
		})
		if err != nil {
			return nil, createIssueOutput{}, fmt.Errorf("create issue in %s: %w", in.ProjectKey, err)
		}
		return nil, createIssueOutput{Key: iss.Key, URL: client.BrowseURL(iss.Key)}, nil
	})
}

// ---- jira_get_issue ----

type getIssueInput struct {
	Key    string   `json:"key" jsonschema:"Issue key, e.g. PROJ-123"`
	Fields []string `json:"fields,omitempty" jsonschema:"Fields to return; defaults to summary and status"`
}

type getIssueOutput struct {
	Key     string `json:"key"`
	Summary string `json:"summary"`
	Status  string `json:"status"`
	URL     string `json:"url"`
}

func registerGetIssue(server *mcp.Server, client *jira.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "jira_get_issue",
		Description: "Fetch a single Jira issue's summary and status by key.",
		// Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in getIssueInput) (*mcp.CallToolResult, getIssueOutput, error) {
		if in.Key == "" {
			return nil, getIssueOutput{}, fmt.Errorf("key is required, e.g. PROJ-123")
		}
		fields := in.Fields
		if len(fields) == 0 {
			fields = []string{"summary", "status"}
		}
		iss, err := client.GetIssue(ctx, in.Key, fields)
		if err != nil {
			return nil, getIssueOutput{}, fmt.Errorf("get issue %s: %w", in.Key, err)
		}
		return nil, getIssueOutput{
			Key:     iss.Key,
			Summary: issueSummary(iss.Fields),
			Status:  issueStatus(iss.Fields),
			URL:     client.BrowseURL(iss.Key),
		}, nil
	})
}

// ---- jira_search ----

type searchInput struct {
	JQL           string   `json:"jql" jsonschema:"JQL query, e.g. project = PROJ AND status = Open ORDER BY created DESC"`
	Fields        []string `json:"fields,omitempty" jsonschema:"Fields to return; defaults to summary and status"`
	NextPageToken string   `json:"next_page_token,omitempty" jsonschema:"Cursor from a previous call; omit for the first page"`
	MaxResults    int      `json:"max_results,omitempty" jsonschema:"Page size 1-100 (default 50)"`
}

type searchHit struct {
	Key     string `json:"key"`
	Summary string `json:"summary"`
	Status  string `json:"status"`
}

type searchOutput struct {
	Issues        []searchHit `json:"issues"`
	NextPageToken string      `json:"next_page_token,omitempty" jsonschema:"Pass back as next_page_token to fetch the next page"`
	IsLast        bool        `json:"is_last" jsonschema:"True when there are no more pages"`
}

func registerSearch(server *mcp.Server, client *jira.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "jira_search",
		Description: "Search issues with JQL. Cursor-paginated: pass next_page_token from the previous result until is_last is true.",
		// Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in searchInput) (*mcp.CallToolResult, searchOutput, error) {
		if in.JQL == "" {
			return nil, searchOutput{}, fmt.Errorf("jql is required, e.g. project = PROJ AND statusCategory != Done")
		}
		fields := in.Fields
		if len(fields) == 0 {
			fields = []string{"summary", "status"}
		}
		res, err := client.SearchJQL(ctx, in.JQL, fields, in.NextPageToken, in.MaxResults)
		if err != nil {
			return nil, searchOutput{}, fmt.Errorf("search jql: %w", err)
		}
		hits := make([]searchHit, 0, len(res.Issues))
		for _, iss := range res.Issues {
			hits = append(hits, searchHit{
				Key:     iss.Key,
				Summary: issueSummary(iss.Fields),
				Status:  issueStatus(iss.Fields),
			})
		}
		return nil, searchOutput{Issues: hits, NextPageToken: res.NextPageToken, IsLast: res.IsLast}, nil
	})
}

// ---- jira_transition_issue ----

type transitionInput struct {
	Key          string `json:"key" jsonschema:"Issue key, e.g. PROJ-123"`
	TransitionID string `json:"transition_id" jsonschema:"Transition id from GET /issue/{key}/transitions"`
}

type statusOutput struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func registerTransition(server *mcp.Server, client *jira.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "jira_transition_issue",
		Description: "Move an issue through a workflow transition (e.g. To Do -> In Progress).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in transitionInput) (*mcp.CallToolResult, statusOutput, error) {
		if in.Key == "" || in.TransitionID == "" {
			return nil, statusOutput{}, fmt.Errorf("key and transition_id are required")
		}
		if err := client.TransitionIssue(ctx, in.Key, in.TransitionID); err != nil {
			return nil, statusOutput{}, fmt.Errorf("transition %s: %w", in.Key, err)
		}
		return nil, statusOutput{OK: true, Message: "transitioned " + in.Key}, nil
	})
}

// ---- jira_link_issues ----

type linkInput struct {
	InwardKey  string `json:"inward_key" jsonschema:"Inward issue key, e.g. the blocked issue"`
	OutwardKey string `json:"outward_key" jsonschema:"Outward issue key, e.g. the blocking issue"`
	LinkType   string `json:"link_type" jsonschema:"Link-type name, e.g. Relates, Blocks"`
}

func registerLink(server *mcp.Server, client *jira.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "jira_link_issues",
		Description: "Create a typed link between two issues (e.g. Blocks, Relates).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in linkInput) (*mcp.CallToolResult, statusOutput, error) {
		if in.InwardKey == "" || in.OutwardKey == "" || in.LinkType == "" {
			return nil, statusOutput{}, fmt.Errorf("inward_key, outward_key, and link_type are required")
		}
		if err := client.LinkIssues(ctx, in.InwardKey, in.OutwardKey, in.LinkType); err != nil {
			return nil, statusOutput{}, fmt.Errorf("link %s -> %s: %w", in.InwardKey, in.OutwardKey, err)
		}
		return nil, statusOutput{OK: true, Message: fmt.Sprintf("linked %s %s %s", in.InwardKey, in.LinkType, in.OutwardKey)}, nil
	})
}

// ---- cif_create_story_with_trace (composite) ----

type createStoryTraceInput struct {
	ProjectKey  string `json:"project_key" jsonschema:"Jira project key"`
	Summary     string `json:"summary" jsonschema:"Story summary"`
	Description string `json:"description,omitempty" jsonschema:"Plain-text description"`
	BRID        string `json:"br_id" jsonschema:"Business requirement id this story traces to, e.g. BR-0007"`
	USID        string `json:"us_id" jsonschema:"User story id in the traceability manifest, e.g. US-0045"`
}

func registerCreateStoryWithTrace(server *mcp.Server, client *jira.Client, cfg Config) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "cif_create_story_with_trace",
		Description: "Create a Story AND write its BR-/US- traceability fields in one atomic call. The CIF Business Analyst uses this so a story can never enter Jira without its requirement link.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createStoryTraceInput) (*mcp.CallToolResult, createIssueOutput, error) {
		if cfg.FieldUSID == "" || cfg.FieldBRLink == "" {
			return nil, createIssueOutput{}, fmt.Errorf("traceability custom fields not configured: set JIRA_FIELD_US_ID and JIRA_FIELD_BR_LINK env vars to the custom-field ids for this instance")
		}
		if in.ProjectKey == "" || in.Summary == "" || in.BRID == "" || in.USID == "" {
			return nil, createIssueOutput{}, fmt.Errorf("project_key, summary, br_id, and us_id are all required")
		}
		iss, err := client.CreateIssue(ctx, jira.CreateIssueInput{
			ProjectKey:  in.ProjectKey,
			IssueType:   "Story",
			Summary:     in.Summary,
			Description: in.Description,
			CustomFields: map[string]any{
				cfg.FieldUSID:   in.USID,
				cfg.FieldBRLink: in.BRID,
			},
		})
		if err != nil {
			return nil, createIssueOutput{}, fmt.Errorf("create traced story in %s: %w", in.ProjectKey, err)
		}
		return nil, createIssueOutput{Key: iss.Key, URL: client.BrowseURL(iss.Key)}, nil
	})
}

// ---- field extraction helpers ----

func issueSummary(fields map[string]any) string {
	if fields == nil {
		return ""
	}
	s, _ := fields["summary"].(string)
	return s
}

func issueStatus(fields map[string]any) string {
	if fields == nil {
		return ""
	}
	st, ok := fields["status"].(map[string]any)
	if !ok {
		return ""
	}
	name, _ := st["name"].(string)
	return name
}
