// Package auditusage writes DIF audit and usage records without storing raw
// request parameters or document text.
package auditusage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	ToolVersionP0 = "p0"

	OutcomeSuccess Outcome = "success"
	OutcomeError   Outcome = "error"
	OutcomeDenied  Outcome = "denied"

	EventTypeMCPToolCall EventType = "mcp_tool_call"
)

var allowedUsageEventTypes = map[EventType]bool{
	"ingestion_run":      true,
	"document_indexed":   true,
	"embedding_batch":    true,
	EventTypeMCPToolCall: true,
	"agent_request":      true,
	"connector_sync":     true,
}

// Outcome mirrors dif_meta.audit_log.outcome.
type Outcome string

// EventType mirrors dif_meta.usage_events.event_type.
type EventType string

// Store persists audit and usage events separately.
type Store interface {
	WriteAuditEvent(context.Context, AuditEvent) error
	WriteUsageEvent(context.Context, UsageEvent) error
}

// Recorder validates and writes paired governance records for one operation.
type Recorder struct {
	Store Store
}

// AuditEvent is the safe write shape for dif_meta.audit_log.
type AuditEvent struct {
	PrincipalID    string
	TenantID       string
	ProjectID      string
	CorpusID       string
	ToolName       string
	ToolVersion    string
	ParametersHash string
	Outcome        Outcome
	LatencyMS      int
	SourceRefs     []string
	ErrorClass     string
}

// UsageEvent is the non-PII write shape for dif_meta.usage_events.
type UsageEvent struct {
	EventType      EventType
	TenantID       string
	ProjectID      string
	CorpusID       string
	ConnectorID    string
	Counts         map[string]int
	LatencyMS      int
	TokenUnits     *int
	EmbeddingUnits *int
	ErrorClass     string
}

// MCPToolCall is the input for creating one audit event and one usage event.
type MCPToolCall struct {
	PrincipalID    string
	TenantID       string
	ProjectID      string
	CorpusID       string
	ToolName       string
	ToolVersion    string
	ParametersHash string
	Outcome        Outcome
	Latency        time.Duration
	SourceRefs     []string
	ErrorClass     string
	Counts         map[string]int
}

// SQLStore writes to dif_meta.audit_log and dif_meta.usage_events using a
// caller-owned database handle or transaction.
type SQLStore struct {
	Execer Execer
}

// Execer is implemented by *sql.DB and *sql.Tx.
type Execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// MemoryStore records writes in memory for tests and local harnesses.
type MemoryStore struct {
	AuditEvents []AuditEvent
	UsageEvents []UsageEvent
}

// Enabled reports whether the recorder has a backing store.
func (r Recorder) Enabled() bool {
	return r.Store != nil
}

// RecordMCPToolCall builds, validates, and writes separate audit and usage
// events for one MCP/API operation.
func (r Recorder) RecordMCPToolCall(ctx context.Context, call MCPToolCall) (AuditEvent, UsageEvent, error) {
	if r.Store == nil {
		return AuditEvent{}, UsageEvent{}, errors.New("auditusage recorder requires a store")
	}
	auditEvent, usageEvent, err := EventsFromMCPToolCall(call)
	if err != nil {
		return AuditEvent{}, UsageEvent{}, err
	}
	if err := r.Store.WriteAuditEvent(ctx, auditEvent); err != nil {
		return AuditEvent{}, UsageEvent{}, err
	}
	if err := r.Store.WriteUsageEvent(ctx, usageEvent); err != nil {
		return AuditEvent{}, UsageEvent{}, err
	}
	return auditEvent, usageEvent, nil
}

// EventsFromMCPToolCall creates the separated safe write shapes without
// persisting them.
func EventsFromMCPToolCall(call MCPToolCall) (AuditEvent, UsageEvent, error) {
	latencyMS := int(call.Latency.Milliseconds())
	if latencyMS < 0 {
		latencyMS = 0
	}
	auditEvent := AuditEvent{
		PrincipalID:    strings.TrimSpace(call.PrincipalID),
		TenantID:       strings.TrimSpace(call.TenantID),
		ProjectID:      strings.TrimSpace(call.ProjectID),
		CorpusID:       strings.TrimSpace(call.CorpusID),
		ToolName:       strings.TrimSpace(call.ToolName),
		ToolVersion:    strings.TrimSpace(call.ToolVersion),
		ParametersHash: strings.TrimSpace(call.ParametersHash),
		Outcome:        call.Outcome,
		LatencyMS:      latencyMS,
		SourceRefs:     sortedNonEmpty(call.SourceRefs),
		ErrorClass:     strings.TrimSpace(call.ErrorClass),
	}
	usageEvent := UsageEvent{
		EventType:  EventTypeMCPToolCall,
		TenantID:   strings.TrimSpace(call.TenantID),
		ProjectID:  strings.TrimSpace(call.ProjectID),
		CorpusID:   strings.TrimSpace(call.CorpusID),
		Counts:     normalizedCounts(call.Counts),
		LatencyMS:  latencyMS,
		ErrorClass: strings.TrimSpace(call.ErrorClass),
	}
	if auditEvent.ToolVersion == "" {
		auditEvent.ToolVersion = ToolVersionP0
	}
	if auditEvent.ParametersHash == "" {
		return AuditEvent{}, UsageEvent{}, errors.New("parameters_hash is required")
	}
	if err := auditEvent.Validate(); err != nil {
		return AuditEvent{}, UsageEvent{}, err
	}
	if err := usageEvent.Validate(); err != nil {
		return AuditEvent{}, UsageEvent{}, err
	}
	return auditEvent, usageEvent, nil
}

// WriteAuditEvent writes one validated audit event.
func (s SQLStore) WriteAuditEvent(ctx context.Context, event AuditEvent) error {
	if s.Execer == nil {
		return errors.New("auditusage SQL store requires an execer")
	}
	if err := event.Validate(); err != nil {
		return err
	}
	sourceRefs, err := json.Marshal(event.SourceRefs)
	if err != nil {
		return err
	}
	_, err = s.Execer.ExecContext(ctx, `
INSERT INTO dif_meta.audit_log (
    principal_id, tenant_id, project_id, corpus_id, tool_name, tool_version,
    parameters_hash, outcome, latency_ms, source_refs, error_class
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11)`,
		event.PrincipalID,
		emptyToNil(event.TenantID),
		event.ProjectID,
		event.CorpusID,
		event.ToolName,
		emptyToNil(event.ToolVersion),
		event.ParametersHash,
		string(event.Outcome),
		event.LatencyMS,
		string(sourceRefs),
		emptyToNil(event.ErrorClass),
	)
	return err
}

// WriteUsageEvent writes one validated non-PII usage event.
func (s SQLStore) WriteUsageEvent(ctx context.Context, event UsageEvent) error {
	if s.Execer == nil {
		return errors.New("auditusage SQL store requires an execer")
	}
	if err := event.Validate(); err != nil {
		return err
	}
	counts, err := json.Marshal(event.Counts)
	if err != nil {
		return err
	}
	_, err = s.Execer.ExecContext(ctx, `
INSERT INTO dif_meta.usage_events (
    event_type, tenant_id, project_id, corpus_id, connector_id, counts,
    latency_ms, token_units, embedding_units, error_class
) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10)`,
		string(event.EventType),
		emptyToNil(event.TenantID),
		event.ProjectID,
		event.CorpusID,
		emptyToNil(event.ConnectorID),
		string(counts),
		event.LatencyMS,
		optionalInt(event.TokenUnits),
		optionalInt(event.EmbeddingUnits),
		emptyToNil(event.ErrorClass),
	)
	return err
}

// WriteAuditEvent records one audit event in memory.
func (s *MemoryStore) WriteAuditEvent(_ context.Context, event AuditEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}
	s.AuditEvents = append(s.AuditEvents, event)
	return nil
}

// WriteUsageEvent records one usage event in memory.
func (s *MemoryStore) WriteUsageEvent(_ context.Context, event UsageEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}
	s.UsageEvents = append(s.UsageEvents, event)
	return nil
}

// Validate checks the audit event against schema and safety constraints.
func (e AuditEvent) Validate() error {
	var errs []error
	required := map[string]string{
		"principal_id":    e.PrincipalID,
		"project_id":      e.ProjectID,
		"corpus_id":       e.CorpusID,
		"tool_name":       e.ToolName,
		"parameters_hash": e.ParametersHash,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			errs = append(errs, fmt.Errorf("%s is required", field))
		}
	}
	if !validOutcome(e.Outcome) {
		errs = append(errs, fmt.Errorf("invalid audit outcome %q", e.Outcome))
	}
	if e.LatencyMS < 0 {
		errs = append(errs, errors.New("audit latency_ms must be non-negative"))
	}
	if e.Outcome == OutcomeDenied && len(e.SourceRefs) > 0 {
		errs = append(errs, errors.New("denied audit event must not include source_refs"))
	}
	return errors.Join(errs...)
}

// Validate checks the usage event against schema and non-PII constraints.
func (e UsageEvent) Validate() error {
	var errs []error
	if !allowedUsageEventTypes[e.EventType] {
		errs = append(errs, fmt.Errorf("invalid usage event_type %q", e.EventType))
	}
	required := map[string]string{
		"project_id": e.ProjectID,
		"corpus_id":  e.CorpusID,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			errs = append(errs, fmt.Errorf("%s is required", field))
		}
	}
	if e.Counts == nil {
		errs = append(errs, errors.New("usage counts are required"))
	}
	for key, value := range e.Counts {
		if strings.TrimSpace(key) == "" {
			errs = append(errs, errors.New("usage count keys must be non-empty"))
		}
		if value < 0 {
			errs = append(errs, fmt.Errorf("usage count %q must be non-negative", key))
		}
	}
	if e.LatencyMS < 0 {
		errs = append(errs, errors.New("usage latency_ms must be non-negative"))
	}
	if e.TokenUnits != nil && *e.TokenUnits < 0 {
		errs = append(errs, errors.New("token_units must be non-negative"))
	}
	if e.EmbeddingUnits != nil && *e.EmbeddingUnits < 0 {
		errs = append(errs, errors.New("embedding_units must be non-negative"))
	}
	return errors.Join(errs...)
}

// HashParameters returns a canonical SHA-256 hash for safe parameters. Callers
// must pass only low-risk metadata, not raw queries, snippets, or document text.
func HashParameters(parameters map[string]any) (string, error) {
	encoded, err := json.Marshal(parameters)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func validOutcome(outcome Outcome) bool {
	switch outcome {
	case OutcomeSuccess, OutcomeError, OutcomeDenied:
		return true
	default:
		return false
	}
}

func sortedNonEmpty(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func normalizedCounts(counts map[string]int) map[string]int {
	if counts == nil {
		return nil
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]int, len(counts))
	for _, key := range keys {
		out[strings.TrimSpace(key)] = counts[key]
	}
	return out
}

func emptyToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func optionalInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}
