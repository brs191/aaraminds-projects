// Command vria-mcp-metrics runs the get_metric_snapshot MCP tool server over
// stdio.
//
// Configuration (environment variables):
//   VRIA_METRICS_CSV  — required; path to the metrics CSV file.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aaraminds/vria/internal/mcpserver"
)

func main() {
	csvPath := os.Getenv("VRIA_METRICS_CSV")
	if csvPath == "" {
		fmt.Fprintln(os.Stderr, "vria-mcp-metrics: VRIA_METRICS_CSV must be set")
		os.Exit(1)
	}

	auditLog := log.New(os.Stderr, "[audit] ", log.LstdFlags|log.LUTC)

	srv := mcpserver.New(mcpserver.Config{
		Audit: func(rec mcpserver.AuditRecord) {
			auditLog.Printf("tool=%s called_at=%s", rec.Tool, rec.CalledAt.UTC().Format("2006-01-02T15:04:05Z"))
		},
	})

	srv.Register("get_metric_snapshot", mcpserver.NewMetricSnapshotHandler(mcpserver.MetricsConfig{
		CSVPath: csvPath,
	}))

	if err := srv.ServeStdio(); err != nil {
		fmt.Fprintf(os.Stderr, "vria-mcp-metrics: serve error: %v\n", err)
		os.Exit(1)
	}
}
