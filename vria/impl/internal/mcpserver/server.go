// Package mcpserver provides a minimal JSON-RPC-over-stdio tool server.
//
// Wire format (one JSON object per line, no framing headers):
//   Request:  {"id": <any>, "tool": "<name>", "input": {…}}
//   Response: {"id": <any>, "output": {…}}          // success
//             {"id": <any>, "error": {"code": "…", "message": "…"}} // failure
//
// Each call is executed under a per-call timeout (default 10 s). An AuditFn
// callback, when set, is invoked synchronously after every successful dispatch
// before the response is written.
package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// DefaultTimeout is applied to every tool call unless overridden in Config.
const DefaultTimeout = 10 * time.Second

// ErrorCode is a canonical error code returned in error responses.
type ErrorCode string

const (
	ErrInvalidInput      ErrorCode = "INVALID_INPUT"
	ErrMetricUnavailable ErrorCode = "METRIC_UNAVAILABLE"
	ErrNoEvidenceFound   ErrorCode = "NO_EVIDENCE_FOUND"
	ErrTimeout           ErrorCode = "TIMEOUT"
	ErrUnknownTool       ErrorCode = "UNKNOWN_TOOL"
	ErrInternalError     ErrorCode = "INTERNAL_ERROR"
)

// AuditRecord is passed to AuditFn after each successful tool call.
type AuditRecord struct {
	Tool      string
	InputJSON json.RawMessage
	CalledAt  time.Time
}

// AuditFn is a hook called after every successful dispatch. Failures before
// dispatch do not trigger the hook.
type AuditFn func(rec AuditRecord)

// ToolError is returned by a Handler to signal a structured failure.
type ToolError struct {
	Code    ErrorCode
	Message string
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Handler is a tool implementation. It receives the raw input JSON and returns
// either a serialisable output value or a *ToolError.
type Handler func(ctx context.Context, input json.RawMessage) (interface{}, *ToolError)

// Config carries server-level settings.
type Config struct {
	// Timeout per tool call. Zero means DefaultTimeout.
	Timeout time.Duration
	// Audit is called after every successful dispatch. May be nil.
	Audit AuditFn
}

// Server dispatches JSON-RPC-over-stdio tool calls.
type Server struct {
	tools   map[string]Handler
	timeout time.Duration
	audit   AuditFn
}

// New creates a Server from cfg.
func New(cfg Config) *Server {
	t := cfg.Timeout
	if t <= 0 {
		t = DefaultTimeout
	}
	return &Server{
		tools:   make(map[string]Handler),
		timeout: t,
		audit:   cfg.Audit,
	}
}

// Register adds a tool handler. Panics on duplicate registration.
func (s *Server) Register(name string, h Handler) {
	if _, exists := s.tools[name]; exists {
		panic("mcpserver: duplicate tool registration: " + name)
	}
	s.tools[name] = h
}

// ---- wire types ----

type request struct {
	ID    json.RawMessage `json:"id"`
	Tool  string          `json:"tool"`
	Input json.RawMessage `json:"input"`
}

type successResponse struct {
	ID     json.RawMessage `json:"id"`
	Output interface{}     `json:"output"`
}

type errorBody struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type errorResponse struct {
	ID    json.RawMessage `json:"id"`
	Error errorBody       `json:"error"`
}

// Serve reads lines from r, dispatches, writes responses to w. It returns only
// on EOF or a permanent read error — a single malformed line is skipped with an
// error response and serving continues.
func (s *Server) Serve(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	enc := json.NewEncoder(w)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		s.handleLine(line, enc)
	}
	return scanner.Err()
}

// ServeStdio is a convenience wrapper over os.Stdin / os.Stdout.
func (s *Server) ServeStdio() error {
	return s.Serve(os.Stdin, os.Stdout)
}

func (s *Server) handleLine(line []byte, enc *json.Encoder) {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		// No valid id available; use null.
		writeError(enc, nil, ErrInvalidInput, "malformed JSON request: "+err.Error())
		return
	}

	// Missing tool name.
	if req.Tool == "" {
		writeError(enc, req.ID, ErrInvalidInput, "missing field: tool")
		return
	}

	h, ok := s.tools[req.Tool]
	if !ok {
		writeError(enc, req.ID, ErrUnknownTool, "unknown tool: "+req.Tool)
		return
	}

	// Per-call timeout.
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	type result struct {
		out interface{}
		err *ToolError
	}
	ch := make(chan result, 1)
	calledAt := time.Now()

	go func() {
		out, terr := h(ctx, req.Input)
		ch <- result{out, terr}
	}()

	select {
	case <-ctx.Done():
		writeError(enc, req.ID, ErrTimeout, "tool call exceeded deadline")
	case res := <-ch:
		if res.err != nil {
			writeError(enc, req.ID, res.err.Code, res.err.Message)
			return
		}
		if s.audit != nil {
			s.audit(AuditRecord{
				Tool:      req.Tool,
				InputJSON: req.Input,
				CalledAt:  calledAt,
			})
		}
		_ = enc.Encode(successResponse{ID: req.ID, Output: res.out})
	}
}

func writeError(enc *json.Encoder, id json.RawMessage, code ErrorCode, msg string) {
	if id == nil {
		id = json.RawMessage("null")
	}
	_ = enc.Encode(errorResponse{
		ID:    id,
		Error: errorBody{Code: code, Message: msg},
	})
}
