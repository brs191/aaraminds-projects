// Command vria-mcp-evidence runs the search_evidence_documents MCP tool server
// over stdio.
//
// Configuration (environment variables):
//   VRIA_EVIDENCE_DIR  — required; path to the directory of .md / .txt files.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aaraminds/vria/internal/mcpserver"
)

func main() {
	evidenceDir := os.Getenv("VRIA_EVIDENCE_DIR")
	if evidenceDir == "" {
		fmt.Fprintln(os.Stderr, "vria-mcp-evidence: VRIA_EVIDENCE_DIR must be set")
		os.Exit(1)
	}

	auditLog := log.New(os.Stderr, "[audit] ", log.LstdFlags|log.LUTC)

	srv := mcpserver.New(mcpserver.Config{
		Audit: func(rec mcpserver.AuditRecord) {
			auditLog.Printf("tool=%s called_at=%s", rec.Tool, rec.CalledAt.UTC().Format("2006-01-02T15:04:05Z"))
		},
	})

	srv.Register("search_evidence_documents", mcpserver.NewSearchEvidenceHandler(mcpserver.EvidenceConfig{
		EvidenceDir: evidenceDir,
	}))

	if err := srv.ServeStdio(); err != nil {
		fmt.Fprintf(os.Stderr, "vria-mcp-evidence: serve error: %v\n", err)
		os.Exit(1)
	}
}
