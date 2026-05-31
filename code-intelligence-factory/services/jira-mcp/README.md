# CIF Jira MCP Server

A **custom Go MCP server** over the **Jira Cloud REST API v3**, built for the Code Intelligence Factory. It replaces the hosted Atlassian server so the tool surface is scoped to exactly what CIF needs — including a composite tool that can never let a story enter Jira without its requirement link. See the platform design in [`../../docs/ARCHITECTURE.md`](../../docs/ARCHITECTURE.md) (§3, §5.2, §8, §12).

Built on the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) (v1.4.0+ tracks MCP spec 2025-11-25).

## Tool surface

| Tool | Purpose | Read-only | Key inputs |
| --- | --- | --- | --- |
| `jira_create_issue` | Create a Story/Bug/Task | no | `project_key`, `issue_type`, `summary` |
| `jira_get_issue` | Fetch one issue's summary + status | yes | `key` |
| `jira_search` | JQL search, cursor-paginated | yes | `jql`, `next_page_token` |
| `jira_transition_issue` | Move through a workflow transition | no | `key`, `transition_id` |
| `jira_link_issues` | Create a typed issue link | no | `inward_key`, `outward_key`, `link_type` |
| `cif_create_story_with_trace` | **Composite** — create a Story *and* write its `BR-`/`US-` trace fields atomically | no | `project_key`, `summary`, `br_id`, `us_id` |

The composite is the CIF-specific tool: the Business Analyst agent calls it so a story is created with its traceability fields set in a single API call — no window where a story exists in Jira unlinked.

## Auth & configuration (env)

| Variable | Required | Purpose |
| --- | --- | --- |
| `JIRA_BASE_URL` | yes | e.g. `https://your-site.atlassian.net` |
| `JIRA_EMAIL` | yes | Account email for API-token (Basic) auth |
| `JIRA_API_TOKEN` | yes | Jira API token — store in **Azure Key Vault**, inject via managed identity |
| `JIRA_FIELD_US_ID` | for composite | Custom-field id holding the `US-` id, e.g. `customfield_10031` |
| `JIRA_FIELD_BR_LINK` | for composite | Custom-field id holding the `BR-` id, e.g. `customfield_10032` |

Resolve the custom-field ids per instance with `GET /rest/api/3/field`. For production, prefer **OAuth 2.0 (3LO)** over an API token — swap the `Authorization` header in `internal/jira/client.go` `do()` for a bearer token.

## REST v3 mapping

- Create: `POST /rest/api/3/issue` (description wrapped in ADF automatically)
- Get: `GET /rest/api/3/issue/{key}`
- Search: `POST /rest/api/3/search/jql` — **the old `GET /search` was removed in Oct 2025.** Pagination is cursor-based via `nextPageToken`; there is no `total`, so iterate until `isLast`.
- Transition: `POST /rest/api/3/issue/{key}/transitions`
- Link: `POST /rest/api/3/issueLink`

429s are retried with `Retry-After`/exponential backoff (bounded, 3 attempts).

## Build, run, verify

```bash
# from services/jira-mcp/
go get github.com/modelcontextprotocol/go-sdk@latest   # resolve exact version
go mod tidy
go vet ./...
go build ./...                                          # produces ./jira-mcp

# run (stdio transport)
export JIRA_BASE_URL=https://your-site.atlassian.net
export JIRA_EMAIL=you@example.com
export JIRA_API_TOKEN=•••
./jira-mcp
```

Interactive test with the MCP Inspector:

```bash
npx @modelcontextprotocol/inspector ./jira-mcp
```

## Follow-ups / `[VERIFY]`

- **Pin the SDK version** in `go.mod` (currently `v1.5.0`) to whatever `go get @latest` resolves; the SDK guarantees API compatibility within v1.x.
- **Enable tool annotations.** `readOnlyHint`/`idempotentHint`/etc. are shown commented in `internal/tools/tools.go`. Confirm field shapes with `go doc github.com/modelcontextprotocol/go-sdk/mcp.ToolAnnotations`, then uncomment — recommended by mcp-builder for better client behavior.
- **Custom-field ids** are instance-specific; the composite tool returns an actionable error until `JIRA_FIELD_US_ID` / `JIRA_FIELD_BR_LINK` are set.
- **Evaluations** in `evals/jira_evals.xml` need answers filled against a frozen fixture project (`CIFTEST`) before they can gate CI.
- **Streamable HTTP** transport for remote deployment: swap `&mcp.StdioTransport{}` in `main.go` for the SDK's HTTP handler, behind the control plane's auth.

## Layout

```
services/jira-mcp/
├── go.mod
├── main.go                 # server wiring + stdio transport + env config
├── internal/
│   ├── jira/
│   │   ├── client.go       # auth, JSON request, 429 backoff
│   │   └── issues.go       # create / get / search(jql) / transition / link + ADF
│   └── tools/
│       └── tools.go        # MCP tool defs + handlers (typed I/O)
└── evals/
    └── jira_evals.xml      # evaluation questions (fill answers vs fixture)
```
