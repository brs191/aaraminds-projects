// server_test.go — contract tests for the mcpserver framework.
//
// Coverage:
//   - Round-trip valid request routes to handler and output contains expected keys.
//   - Missing required field → INVALID_INPUT, no partial data.
//   - Malformed input JSON → error, server keeps serving next request.
//   - Timeout path: handler exceeding deadline returns TIMEOUT.
//   - Every successful call invokes the audit hook.
//   - Unknown tool → UNKNOWN_TOOL error.
package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ---- helpers ----

// parseResponse decodes a single JSON line from buf into a generic map.
func parseResponse(t *testing.T, buf *bytes.Buffer) map[string]interface{} {
	t.Helper()
	line, err := buf.ReadString('\n')
	if err != nil && line == "" {
		t.Fatalf("no response line: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &m); err != nil {
		t.Fatalf("cannot parse response %q: %v", line, err)
	}
	return m
}

func assertKey(t *testing.T, m map[string]interface{}, key string) {
	t.Helper()
	if _, ok := m[key]; !ok {
		t.Errorf("expected key %q in response, got keys: %v", key, mapKeys(m))
	}
}

func assertNoKey(t *testing.T, m map[string]interface{}, key string) {
	t.Helper()
	if _, ok := m[key]; ok {
		t.Errorf("expected no key %q in response, got keys: %v", key, mapKeys(m))
	}
}

func assertErrorCode(t *testing.T, m map[string]interface{}, want ErrorCode) {
	t.Helper()
	errField, ok := m["error"]
	if !ok {
		t.Fatalf("expected error field, got: %v", m)
	}
	errMap, ok := errField.(map[string]interface{})
	if !ok {
		t.Fatalf("error field is not an object: %T", errField)
	}
	code, _ := errMap["code"].(string)
	if code != string(want) {
		t.Errorf("error code: want %q, got %q", want, code)
	}
}

func mapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// echoHandler echoes input back as output.
func echoHandler(_ context.Context, input json.RawMessage) (interface{}, *ToolError) {
	var v interface{}
	if err := json.Unmarshal(input, &v); err != nil {
		return nil, &ToolError{Code: ErrInvalidInput, Message: err.Error()}
	}
	return v, nil
}

// ---- tests ----

func TestRoundTrip_ValidRequest(t *testing.T) {
	var auditCalled int32
	srv := New(Config{
		Audit: func(rec AuditRecord) { atomic.AddInt32(&auditCalled, 1) },
	})
	srv.Register("echo", echoHandler)

	req := `{"id":1,"tool":"echo","input":{"hello":"world"}}` + "\n"
	in := strings.NewReader(req)
	var out bytes.Buffer

	if err := srv.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	resp := parseResponse(t, &out)
	assertKey(t, resp, "id")
	assertKey(t, resp, "output")
	assertNoKey(t, resp, "error")

	if atomic.LoadInt32(&auditCalled) != 1 {
		t.Errorf("audit hook called %d times, want 1", auditCalled)
	}
}

func TestAuditHook_CalledOnEverySuccess(t *testing.T) {
	var count int32
	srv := New(Config{
		Audit: func(rec AuditRecord) { atomic.AddInt32(&count, 1) },
	})
	srv.Register("echo", echoHandler)

	var reqBuf strings.Builder
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&reqBuf, `{"id":%d,"tool":"echo","input":{}}`, i)
		reqBuf.WriteString("\n")
	}

	in := strings.NewReader(reqBuf.String())
	var out bytes.Buffer
	if err := srv.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if atomic.LoadInt32(&count) != 5 {
		t.Errorf("audit hook called %d times, want 5", count)
	}
}

func TestMalformedJSON_KeepsServing(t *testing.T) {
	srv := New(Config{})
	srv.Register("echo", echoHandler)

	// malformed line followed by valid line
	input := "NOT JSON AT ALL\n" +
		`{"id":2,"tool":"echo","input":{"ok":true}}` + "\n"
	in := strings.NewReader(input)
	var out bytes.Buffer
	if err := srv.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// First response: error for the malformed line.
	resp1 := parseResponse(t, &out)
	assertKey(t, resp1, "error")

	// Second response: successful echo.
	resp2 := parseResponse(t, &out)
	assertKey(t, resp2, "output")
	assertNoKey(t, resp2, "error")
}

func TestTimeout_ExceedingDeadlineReturnsError(t *testing.T) {
	srv := New(Config{Timeout: 50 * time.Millisecond})
	srv.Register("slow", func(ctx context.Context, _ json.RawMessage) (interface{}, *ToolError) {
		select {
		case <-ctx.Done():
			return nil, &ToolError{Code: ErrTimeout, Message: "deadline exceeded"}
		case <-time.After(5 * time.Second):
			return map[string]string{"done": "true"}, nil
		}
	})

	req := `{"id":99,"tool":"slow","input":{}}` + "\n"
	in := strings.NewReader(req)
	var out bytes.Buffer
	if err := srv.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	resp := parseResponse(t, &out)
	assertErrorCode(t, resp, ErrTimeout)
}

func TestUnknownTool_ReturnsError(t *testing.T) {
	srv := New(Config{})
	req := `{"id":5,"tool":"no_such_tool","input":{}}` + "\n"
	in := strings.NewReader(req)
	var out bytes.Buffer
	if err := srv.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	resp := parseResponse(t, &out)
	assertErrorCode(t, resp, ErrUnknownTool)
}

func TestMissingToolField_ReturnsInvalidInput(t *testing.T) {
	srv := New(Config{})
	req := `{"id":6,"input":{}}` + "\n"
	in := strings.NewReader(req)
	var out bytes.Buffer
	if err := srv.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	resp := parseResponse(t, &out)
	assertErrorCode(t, resp, ErrInvalidInput)
}

// ---- CSV metrics tests ----

// writeTempCSV writes a metrics CSV to a temp dir and returns the file path.
// The caller is responsible for removing the returned directory when done.
func writeTempCSV(t *testing.T, rows [][]string) (path string, cleanup func()) {
	t.Helper()
	dir, err := ioutil.TempDir("", "vria-metrics-test")
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}

	csvPath := dir + "/metrics.csv"
	f, err := os.Create(csvPath)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("Create: %v", err)
	}

	header := "metric_id,use_case_id,period_start,period_end,baseline_value,current_value,target_value,metric_unit,source_system,source_owner,authority,freshness,cost,currency\n"
	if _, err := f.WriteString(header); err != nil {
		f.Close()
		os.RemoveAll(dir)
		t.Fatalf("write header: %v", err)
	}
	for _, row := range rows {
		line := strings.Join(row, ",") + "\n"
		if _, err := f.WriteString(line); err != nil {
			f.Close()
			os.RemoveAll(dir)
			t.Fatalf("write row: %v", err)
		}
	}
	f.Close()
	return csvPath, func() { os.RemoveAll(dir) }
}

func sendMetricRequest(t *testing.T, srv *Server, metricID, useCaseID string) map[string]interface{} {
	t.Helper()
	input := fmt.Sprintf(`{"metric_id":%q,"use_case_id":%q,"period":{"start":"2025-01-01","end":"2025-12-31"}}`,
		metricID, useCaseID)
	req := fmt.Sprintf(`{"id":1,"tool":"get_metric_snapshot","input":%s}`, input) + "\n"
	in := strings.NewReader(req)
	var out bytes.Buffer
	if err := srv.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	return parseResponse(t, &out)
}

func TestMetricSnapshot_ValidRoundTrip(t *testing.T) {
	csvPath, cleanup := writeTempCSV(t, [][]string{
		{"M001", "UC001", "2025-01-01", "2025-12-31", "100", "120", "150", "percent", "DataWarehouse", "alice", "Authoritative", "Fresh", "5000", "USD"},
	})
	defer cleanup()

	var auditCalled int32
	srv := New(Config{
		Audit: func(rec AuditRecord) { atomic.AddInt32(&auditCalled, 1) },
	})
	srv.Register("get_metric_snapshot", NewMetricSnapshotHandler(MetricsConfig{CSVPath: csvPath}))

	resp := sendMetricRequest(t, srv, "M001", "UC001")

	assertNoKey(t, resp, "error")
	assertKey(t, resp, "output")

	out, ok := resp["output"].(map[string]interface{})
	if !ok {
		t.Fatalf("output is not an object: %T", resp["output"])
	}
	// Contract §3.5 required output keys.
	for _, key := range []string{"metric_snapshot", "initiative_cost_period", "source_owner", "freshness", "authority", "audit_id"} {
		assertKey(t, out, key)
	}
	snap, ok := out["metric_snapshot"].(map[string]interface{})
	if !ok {
		t.Fatalf("metric_snapshot is not an object")
	}
	for _, key := range []string{"metric_id", "use_case_id", "period_start", "period_end", "baseline_value", "current_value", "target_value", "metric_unit", "source_system"} {
		assertKey(t, snap, key)
	}

	if atomic.LoadInt32(&auditCalled) != 1 {
		t.Errorf("audit hook called %d times, want 1", auditCalled)
	}
}

func TestMetricSnapshot_MissingMetric_ReturnsUnavailable(t *testing.T) {
	csvPath, cleanup := writeTempCSV(t, [][]string{
		{"M001", "UC001", "2025-01-01", "2025-12-31", "100", "120", "150", "percent", "DW", "alice", "Authoritative", "Fresh", "0", "USD"},
	})
	defer cleanup()

	srv := New(Config{})
	srv.Register("get_metric_snapshot", NewMetricSnapshotHandler(MetricsConfig{CSVPath: csvPath}))

	resp := sendMetricRequest(t, srv, "M999", "UC001")

	assertErrorCode(t, resp, ErrMetricUnavailable)
	assertNoKey(t, resp, "output")
}

func TestMetricSnapshot_MissingMetricID_ReturnsInvalidInput(t *testing.T) {
	csvPath, cleanup := writeTempCSV(t, nil)
	defer cleanup()

	srv := New(Config{})
	srv.Register("get_metric_snapshot", NewMetricSnapshotHandler(MetricsConfig{CSVPath: csvPath}))

	req := `{"id":1,"tool":"get_metric_snapshot","input":{"use_case_id":"UC001"}}` + "\n"
	in := strings.NewReader(req)
	var out bytes.Buffer
	if err := srv.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	resp := parseResponse(t, &out)
	assertErrorCode(t, resp, ErrInvalidInput)
}

// ---- Evidence document tests ----

// writeTempDocs creates a temp dir with .md files and returns the dir path
// and a cleanup function. The caller must call cleanup when done.
func writeTempDocs(t *testing.T, docs map[string]string) (dir string, cleanup func()) {
	t.Helper()
	dir, err := ioutil.TempDir("", "vria-evidence-test")
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}

	for name, content := range docs {
		path := dir + "/" + name
		if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
			os.RemoveAll(dir)
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}
	return dir, func() { os.RemoveAll(dir) }
}

func sendEvidenceRequest(t *testing.T, srv *Server, query string, topK int) map[string]interface{} {
	t.Helper()
	input := fmt.Sprintf(`{"query":%q,"top_k":%d}`, query, topK)
	req := fmt.Sprintf(`{"id":1,"tool":"search_evidence_documents","input":%s}`, input) + "\n"
	in := strings.NewReader(req)
	var out bytes.Buffer
	if err := srv.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	return parseResponse(t, &out)
}

func TestEvidenceSearch_ValidRoundTrip(t *testing.T) {
	dir, cleanup := writeTempDocs(t, map[string]string{
		"doc1.md":  "# Cost Reduction\nThe automation reduced costs by 30 percent in Q3.\nEvidence from BI dashboard.\n",
		"doc2.txt": "No matching keywords here.\n",
	})
	defer cleanup()

	var auditCalled int32
	srv := New(Config{
		Audit: func(rec AuditRecord) { atomic.AddInt32(&auditCalled, 1) },
	})
	srv.Register("search_evidence_documents", NewSearchEvidenceHandler(EvidenceConfig{EvidenceDir: dir}))

	resp := sendEvidenceRequest(t, srv, "cost reduction automation", 5)

	assertNoKey(t, resp, "error")
	assertKey(t, resp, "output")

	out, ok := resp["output"].(map[string]interface{})
	if !ok {
		t.Fatalf("output is not an object: %T", resp["output"])
	}
	assertKey(t, out, "results")
	assertKey(t, out, "audit_id")

	results, ok := out["results"].([]interface{})
	if !ok {
		t.Fatalf("results is not an array: %T", out["results"])
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	// Every result must have a non-empty citation_pointer.
	for i, r := range results {
		rm, ok := r.(map[string]interface{})
		if !ok {
			t.Fatalf("result[%d] is not an object", i)
		}
		for _, key := range []string{"document_id", "title", "citation_pointer", "authority", "freshness", "evidence_quality", "snippet"} {
			assertKey(t, rm, key)
		}
		cp, _ := rm["citation_pointer"].(string)
		if cp == "" {
			t.Errorf("result[%d] has empty citation_pointer", i)
		}
	}

	if atomic.LoadInt32(&auditCalled) != 1 {
		t.Errorf("audit hook called %d times, want 1", auditCalled)
	}
}

func TestEvidenceSearch_EmptyResults_NoFabricatedCitations(t *testing.T) {
	dir, cleanup := writeTempDocs(t, map[string]string{
		"doc1.md": "This document has nothing relevant.\n",
	})
	defer cleanup()

	srv := New(Config{})
	srv.Register("search_evidence_documents", NewSearchEvidenceHandler(EvidenceConfig{EvidenceDir: dir}))

	resp := sendEvidenceRequest(t, srv, "zzz_no_match_xyzzy_impossible_token", 5)

	// Empty search: output with empty results array and error_code NO_EVIDENCE_FOUND.
	assertNoKey(t, resp, "error")
	assertKey(t, resp, "output")

	out, ok := resp["output"].(map[string]interface{})
	if !ok {
		t.Fatalf("output is not an object")
	}
	results, ok := out["results"].([]interface{})
	if !ok {
		t.Fatalf("results is not an array: %T", out["results"])
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
	errCode, _ := out["error_code"].(string)
	if errCode != string(ErrNoEvidenceFound) {
		t.Errorf("error_code: want %q, got %q", ErrNoEvidenceFound, errCode)
	}
	// Assert audit_id is present even for empty results.
	assertKey(t, out, "audit_id")
}

func TestEvidenceSearch_MissingQuery_ReturnsInvalidInput(t *testing.T) {
	dir, cleanup := writeTempDocs(t, nil)
	defer cleanup()

	srv := New(Config{})
	srv.Register("search_evidence_documents", NewSearchEvidenceHandler(EvidenceConfig{EvidenceDir: dir}))

	req := `{"id":1,"tool":"search_evidence_documents","input":{"top_k":5}}` + "\n"
	in := strings.NewReader(req)
	var out bytes.Buffer
	if err := srv.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	resp := parseResponse(t, &out)
	assertErrorCode(t, resp, ErrInvalidInput)
}

func TestEvidenceSearch_CitationPointerFormat(t *testing.T) {
	dir, cleanup := writeTempDocs(t, map[string]string{
		"evidence.md": "Line one.\nMachine learning reduces cost dramatically.\nLine three.\n",
	})
	defer cleanup()

	srv := New(Config{})
	srv.Register("search_evidence_documents", NewSearchEvidenceHandler(EvidenceConfig{EvidenceDir: dir}))

	resp := sendEvidenceRequest(t, srv, "machine learning cost", 3)

	out := resp["output"].(map[string]interface{})
	results := out["results"].([]interface{})
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	for i, r := range results {
		rm := r.(map[string]interface{})
		cp, _ := rm["citation_pointer"].(string)
		if cp == "" {
			t.Errorf("result[%d]: empty citation_pointer", i)
		}
		// citation_pointer must contain a colon (path:line_number format).
		if !strings.Contains(cp, ":") {
			t.Errorf("result[%d]: citation_pointer %q lacks line number separator", i, cp)
		}
	}
}

func TestMetricSnapshot_MalformedInputJSON(t *testing.T) {
	csvPath, cleanup := writeTempCSV(t, [][]string{
		{"M001", "UC001", "2025-01-01", "2025-12-31", "100", "120", "150", "pct", "DW", "alice", "Authoritative", "Fresh", "0", "USD"},
	})
	defer cleanup()

	srv := New(Config{})
	srv.Register("get_metric_snapshot", NewMetricSnapshotHandler(MetricsConfig{CSVPath: csvPath}))

	// Malformed input JSON in the input field, but valid outer JSON.
	req := `{"id":1,"tool":"get_metric_snapshot","input":"not-an-object"}` + "\n"
	in := strings.NewReader(req)
	var out bytes.Buffer
	if err := srv.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	resp := parseResponse(t, &out)
	assertErrorCode(t, resp, ErrInvalidInput)
}
