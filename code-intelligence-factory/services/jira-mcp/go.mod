module github.com/aaraminds/code-intelligence-factory/services/jira-mcp

go 1.23

// Pin to the latest stable v1.x (>= v1.4.0 tracks MCP spec 2025-11-25).
// Run `go get github.com/modelcontextprotocol/go-sdk@latest` to resolve the
// exact patch version, then `go mod tidy`. [VERIFY] the version below.
require github.com/modelcontextprotocol/go-sdk v1.5.0
