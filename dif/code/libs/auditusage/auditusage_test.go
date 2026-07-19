package auditusage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGoldenMCPCallsProduceSeparatedSafeAuditAndUsageEvents(t *testing.T) {
	t.Parallel()

	expectations := loadExpectations(t)
	for _, item := range expectations.MCPCallExpectations {
		item := item
		t.Run(item.CaseID, func(t *testing.T) {
			t.Parallel()

			parametersHash, err := HashParameters(item.Parameters)
			if err != nil {
				t.Fatalf("hash parameters: %v", err)
			}
			store := &MemoryStore{}
			recorder := Recorder{Store: store}
			auditEvent, usageEvent, err := recorder.RecordMCPToolCall(context.Background(), MCPToolCall{
				PrincipalID:    item.PrincipalID,
				TenantID:       item.TenantID,
				ProjectID:      item.ProjectID,
				CorpusID:       item.CorpusID,
				ToolName:       item.ToolName,
				ToolVersion:    item.ToolVersion,
				ParametersHash: parametersHash,
				Outcome:        Outcome(item.Outcome),
				Latency:        time.Duration(item.LatencyMS) * time.Millisecond,
				SourceRefs:     item.SourceRefs,
				ErrorClass:     item.ErrorClass,
				Counts:         item.UsageCounts,
			})
			if err != nil {
				t.Fatalf("record MCP call: %v", err)
			}

			if len(store.AuditEvents) != 1 || len(store.UsageEvents) != 1 {
				t.Fatalf("expected separate audit and usage writes, got %+v %+v", store.AuditEvents, store.UsageEvents)
			}
			assertAuditEvent(t, auditEvent, item, parametersHash)
			assertUsageEvent(t, usageEvent, item)
			assertSafeJSONRecord(t, auditEvent, expectations.ProhibitedRecordFields, expectedFixtureExcerpts(t))
			assertSafeJSONRecord(t, usageEvent, expectations.ProhibitedRecordFields, expectedFixtureExcerpts(t))
		})
	}
}

func TestDeniedAuditEventRejectsSourceRefs(t *testing.T) {
	t.Parallel()

	_, _, err := EventsFromMCPToolCall(MCPToolCall{
		PrincipalID:    "principal-p0-qa",
		ProjectID:      "dif-p0-golden",
		CorpusID:       "golden-restricted",
		ToolName:       "search_docs",
		ParametersHash: "hash",
		Outcome:        OutcomeDenied,
		SourceRefs:     []string{"golden-admitted@docver:md:file.md#L1-L1"},
		Counts:         map[string]int{"request_count": 1},
	})
	if err == nil || !strings.Contains(err.Error(), "denied audit event must not include source_refs") {
		t.Fatalf("expected denied source_refs validation error, got %v", err)
	}
}

func TestUsageEventRejectsNegativeCounts(t *testing.T) {
	t.Parallel()

	event := UsageEvent{
		EventType: EventTypeMCPToolCall,
		ProjectID: "dif-p0-golden",
		CorpusID:  "golden-admitted",
		Counts:    map[string]int{"request_count": -1},
	}
	if err := event.Validate(); err == nil || !strings.Contains(err.Error(), "must be non-negative") {
		t.Fatalf("expected negative count validation error, got %v", err)
	}
}

func TestSQLStoreWritesSeparateTablesWithoutRawParameters(t *testing.T) {
	t.Parallel()

	execer := &recordingExecer{}
	store := SQLStore{Execer: execer}
	auditEvent := AuditEvent{
		PrincipalID:    "principal-p0-qa",
		TenantID:       "tenant-p0",
		ProjectID:      "dif-p0-golden",
		CorpusID:       "golden-admitted",
		ToolName:       "search_docs",
		ToolVersion:    ToolVersionP0,
		ParametersHash: "parameters-hash",
		Outcome:        OutcomeSuccess,
		LatencyMS:      42,
		SourceRefs:     []string{"golden-admitted@docver-architecture-overview:md:architecture-overview.md#L5-L8"},
	}
	usageEvent := UsageEvent{
		EventType: EventTypeMCPToolCall,
		TenantID:  "tenant-p0",
		ProjectID: "dif-p0-golden",
		CorpusID:  "golden-admitted",
		Counts: map[string]int{
			"request_count":    1,
			"result_count":     1,
			"source_ref_count": 1,
			"denied_count":     0,
		},
		LatencyMS: 42,
	}

	if err := store.WriteAuditEvent(context.Background(), auditEvent); err != nil {
		t.Fatalf("write audit: %v", err)
	}
	if err := store.WriteUsageEvent(context.Background(), usageEvent); err != nil {
		t.Fatalf("write usage: %v", err)
	}
	if len(execer.calls) != 2 {
		t.Fatalf("expected two SQL writes, got %+v", execer.calls)
	}
	if !strings.Contains(execer.calls[0].query, "dif_meta.audit_log") || !strings.Contains(execer.calls[1].query, "dif_meta.usage_events") {
		t.Fatalf("expected separate table writes, got %+v", execer.calls)
	}
	allArgs, err := json.Marshal(execer.calls)
	if err != nil {
		t.Fatalf("marshal calls: %v", err)
	}
	rendered := string(allArgs)
	for _, prohibited := range []string{"parameters", "query", "snippet", "document_text", "raw_text", "content"} {
		if strings.Contains(rendered, prohibited) {
			t.Fatalf("SQL write shape leaked prohibited field %q in %s", prohibited, rendered)
		}
	}
}

type recordingExecer struct {
	calls []sqlCall
}

type sqlCall struct {
	query string
	args  []any
}

func (e *recordingExecer) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	e.calls = append(e.calls, sqlCall{query: query, args: append([]any{}, args...)})
	return fakeResult(1), nil
}

type fakeResult int64

func (r fakeResult) LastInsertId() (int64, error) { return 0, driver.ErrSkip }
func (r fakeResult) RowsAffected() (int64, error) { return int64(r), nil }

func assertAuditEvent(t *testing.T, event AuditEvent, expected mcpCallExpectation, parametersHash string) {
	t.Helper()
	if event.PrincipalID != expected.PrincipalID || event.TenantID != expected.TenantID || event.ProjectID != expected.ProjectID ||
		event.CorpusID != expected.CorpusID || event.ToolName != expected.ToolName || event.ToolVersion != expected.ToolVersion {
		t.Fatalf("audit dimensions mismatch: got %+v expected %+v", event, expected)
	}
	if event.ParametersHash != parametersHash {
		t.Fatalf("expected parameters hash %q, got %+v", parametersHash, event)
	}
	if string(event.Outcome) != expected.Outcome || event.LatencyMS != expected.LatencyMS || event.ErrorClass != expected.ErrorClass {
		t.Fatalf("audit outcome/latency/error mismatch: got %+v expected %+v", event, expected)
	}
	if strings.Join(event.SourceRefs, "\n") != strings.Join(expected.SourceRefs, "\n") {
		t.Fatalf("source refs mismatch: got %+v expected %+v", event.SourceRefs, expected.SourceRefs)
	}
}

func assertUsageEvent(t *testing.T, event UsageEvent, expected mcpCallExpectation) {
	t.Helper()
	if string(event.EventType) != expected.UsageEventType || event.TenantID != expected.TenantID ||
		event.ProjectID != expected.ProjectID || event.CorpusID != expected.CorpusID {
		t.Fatalf("usage dimensions mismatch: got %+v expected %+v", event, expected)
	}
	if event.LatencyMS != expected.LatencyMS || event.ErrorClass != expected.ErrorClass {
		t.Fatalf("usage latency/error mismatch: got %+v expected %+v", event, expected)
	}
	if len(event.Counts) != len(expected.UsageCounts) {
		t.Fatalf("usage counts mismatch: got %+v expected %+v", event.Counts, expected.UsageCounts)
	}
	for key, value := range expected.UsageCounts {
		if event.Counts[key] != value {
			t.Fatalf("usage count %q mismatch: got %+v expected %+v", key, event.Counts, expected.UsageCounts)
		}
	}
}

func assertSafeJSONRecord(t *testing.T, record any, prohibitedFields []string, prohibitedLiterals []string) {
	t.Helper()
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	rendered := string(encoded)
	for _, field := range prohibitedFields {
		if strings.Contains(rendered, `"`+field+`"`) {
			t.Fatalf("record stores prohibited field %q: %s", field, rendered)
		}
	}
	for _, literal := range prohibitedLiterals {
		if literal != "" && strings.Contains(rendered, literal) {
			t.Fatalf("record leaks raw fixture text %q: %s", literal, rendered)
		}
	}
}

func loadExpectations(t *testing.T) auditUsageExpectations {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "evaluation", "golden", "expected-audit-usage.json"))
	if err != nil {
		t.Fatalf("read expected audit/usage: %v", err)
	}
	var expectations auditUsageExpectations
	if err := json.Unmarshal(content, &expectations); err != nil {
		t.Fatalf("parse expected audit/usage: %v", err)
	}
	return expectations
}

func expectedFixtureExcerpts(t *testing.T) []string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "evaluation", "golden", "expected-anchors.json"))
	if err != nil {
		t.Fatalf("read expected anchors: %v", err)
	}
	var payload struct {
		Anchors []struct {
			ExpectedExcerpt string `json:"expected_excerpt"`
		} `json:"anchors"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("parse expected anchors: %v", err)
	}
	var excerpts []string
	for _, anchor := range payload.Anchors {
		excerpts = append(excerpts, anchor.ExpectedExcerpt)
	}
	return excerpts
}

type auditUsageExpectations struct {
	MCPCallExpectations    []mcpCallExpectation `json:"mcp_call_expectations"`
	ProhibitedRecordFields []string             `json:"prohibited_record_fields"`
}

type mcpCallExpectation struct {
	CaseID         string         `json:"case_id"`
	PrincipalID    string         `json:"principal_id"`
	TenantID       string         `json:"tenant_id"`
	ProjectID      string         `json:"project_id"`
	CorpusID       string         `json:"corpus_id"`
	ToolName       string         `json:"tool_name"`
	ToolVersion    string         `json:"tool_version"`
	Parameters     map[string]any `json:"parameters"`
	Outcome        string         `json:"outcome"`
	LatencyMS      int            `json:"latency_ms"`
	SourceRefs     []string       `json:"source_refs"`
	ErrorClass     string         `json:"error_class"`
	UsageEventType string         `json:"usage_event_type"`
	UsageCounts    map[string]int `json:"usage_counts"`
}
