package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxRawBodyBytes caps how much of a request body the raw-tool-call shim will
// buffer before handing off to the SDK handler.
const maxRawBodyBytes = 1 << 20 // 1 MiB, matching ingestion's decodeBody cap

func main() {
	var addr string
	flag.StringVar(&addr, "addr", envOr("MCP_SERVER_ADDR", ":8081"), "HTTP listen address")
	flag.Parse()

	app, err := NewApp(context.Background(), Config{
		DatabaseURL:     envOr("DATABASE_URL", ""),
		EmbeddingURL:    envOr("EMBEDDING_SERVICE_URL", "http://127.0.0.1:8000/embed"),
		AgentServiceURL: envOr("AGENT_SERVICE_URL", ""),
		AuditLogPath:    envOr("AUDIT_LOG_PATH", "./audit.log"),
		FixtureMode:     strings.EqualFold(envOr("MCP_FIXTURE_MODE", "false"), "true"),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	server := mcp.NewServer(&mcp.Implementation{Name: "rif-mcp-server", Version: "v0.1.0"}, nil)
	registerTools(server, app)
	streamableHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)

	mux := newMux(app, streamableHandler)

	log.Printf("rif mcp server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

// newMux builds the production HTTP mux: raw-tool-call shim first, SDK
// streamable handler as fall-through. Extracted so integration tests exercise
// the same routing the deployed binary uses (the C1 regression lived in the
// gap between the test mux and this one).
func newMux(app *App, streamableHandler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			if handled := serveRawToolCall(app, w, r); handled {
				return
			}
		}
		streamableHandler.ServeHTTP(w, r)
	})
	return mux
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type rawToolRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Method  string `json:"method"`
	Params  struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"params"`
}

type rawToolResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Result  map[string]any `json:"result,omitempty"`
	Error   map[string]any `json:"error,omitempty"`
}

func serveRawToolCall(app *App, w http.ResponseWriter, r *http.Request) bool {
	// Buffer the body before sniffing. The previous implementation decoded
	// r.Body directly, so when the request was NOT a raw tools/call (e.g. an
	// MCP SDK client sending initialize or tools/list) it fell through to the
	// SDK handler with an already-drained body, breaking the MCP session
	// handshake for every real client. Buffering lets us restore the body for
	// the fall-through path.
	body, readErr := io.ReadAll(io.LimitReader(r.Body, maxRawBodyBytes+1))
	if readErr != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return true
	}
	if int64(len(body)) > maxRawBodyBytes {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return true
	}
	// Restore the body so the SDK handler sees the full request if we decline.
	r.Body = io.NopCloser(bytes.NewReader(body))

	var req rawToolRequest
	if err := json.Unmarshal(body, &req); err != nil {
		// Not decodable as our raw shape — let the SDK handler judge it.
		return false
	}
	if req.Method != "tools/call" || strings.TrimSpace(req.Params.Name) == "" {
		return false
	}

	var (
		result *mcp.CallToolResult
		err    error
	)
	switch req.Params.Name {
	case "search_code":
		var in SearchCodeInput
		err = json.Unmarshal(req.Params.Arguments, &in)
		if err == nil {
			result, _, err = app.handleSearchCode(r.Context(), nil, in)
		}
	case "find_callers":
		var in FindCallersInput
		err = json.Unmarshal(req.Params.Arguments, &in)
		if err == nil {
			result, _, err = app.handleFindCallers(r.Context(), nil, in)
		}
	case "impact_analysis":
		var in ImpactAnalysisInput
		err = json.Unmarshal(req.Params.Arguments, &in)
		if err == nil {
			result, _, err = app.handleImpactAnalysis(r.Context(), nil, in)
		}
	case "explain_architecture":
		var in ExplainArchitectureInput
		err = json.Unmarshal(req.Params.Arguments, &in)
		if err == nil {
			result, _, err = app.handleExplainArchitecture(r.Context(), nil, in)
		}
	case "dependency_analysis":
		var in DependencyAnalysisInput
		err = json.Unmarshal(req.Params.Arguments, &in)
		if err == nil {
			result, _, err = app.handleDependencyAnalysis(r.Context(), nil, in)
		}
	default:
		http.Error(w, "unknown tool", http.StatusBadRequest)
		return true
	}

	w.Header().Set("Content-Type", "application/json")
	resp := rawToolResponse{JSONRPC: "2.0", ID: req.ID}
	if err != nil {
		resp.Error = map[string]any{"code": -32000, "message": err.Error()}
		_ = json.NewEncoder(w).Encode(resp)
		return true
	}

	content := make([]map[string]any, 0, len(result.Content))
	for _, item := range result.Content {
		if text, ok := item.(*mcp.TextContent); ok {
			content = append(content, map[string]any{"type": "text", "text": text.Text})
		}
	}
	resp.Result = map[string]any{"content": content}
	_ = json.NewEncoder(w).Encode(resp)
	return true
}
