// Package session reads GitHub Copilot CLI session state from the local file system.
// All data is sourced from ~/.copilot/session-state/<uuid>/events.jsonl — no network calls.
package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aaraminds/copilot-token-budget/internal/livebilling"
	"github.com/aaraminds/copilot-token-budget/internal/platform"
)

// Session holds the billing and token data for a single Copilot CLI session.
type Session struct {
	ID           string
	WorkspaceDir string
	ProjectName  string // filepath.Base(WorkspaceDir)
	StartTime    time.Time
	EndTime      time.Time
	IsActive     bool  // true when an inuse.*.lock file is present in the session dir
	TotalNanoAIU int64 // from session.shutdown → data.totalNanoAiu, or latest running snapshot
	// TotalPremiumRequests is the count of premium (paid-tier) requests this
	// session made, from session.shutdown → data.totalPremiumRequests. It is
	// only carried by the final/shutdown event; running snapshots leave it zero.
	TotalPremiumRequests int64
	PrimaryModel         string
	Tokens               TokenBreakdown
	ModelMetrics         []ModelMetric

	// IsFinal reports whether the billing/token figures are authoritative.
	// true  → a session.shutdown event has been applied (final, settled billing).
	// false → the figures are a partial/live reading taken from a running
	//         snapshot event (or are still zero for an active session that has
	//         not yet emitted any billing-bearing event). The UI must label
	//         these as live/partial rather than presenting them as final.
	IsFinal bool

	// Source identifies which Collector produced this session. Known values:
	//   "cli" — GitHub Copilot CLI session-state (events.jsonl).
	//   "ide-chat" — VS Code Copilot Chat (Nitrite DB).
	//   "ide-edit" — VS Code Copilot Edit extension (future).
	//   "ide-agent" — VS Code Copilot Agent (future).
	// Source lets the dedup step in ReadAll reason about cross-source overlap and
	// lets the UI attribute spend to the originating tool.
	Source string

	// TokenCostSource indicates the trustworthiness of the cost figures.
	// Known values:
	//   "authoritative" — Cost is from CLI session.shutdown event (ground truth).
	//   "estimated" — Cost is computed from IDE token counts via pricing table (Phase 6 limitation).
	// Callers must use this label when reporting costs to distinguish settled charges
	// from estimates pending GitHub API enrichment (Phase 7).
	TokenCostSource string

	// OrgBillingSnapshot carries optional org-aggregate live billing metadata.
	// It is nil in the default local-first path and only populated when the
	// user explicitly opts in to Phase 8 live billing enrichment.
	OrgBillingSnapshot *livebilling.OrgBillingSnapshot
}

// TokenBreakdown holds the last-known context-window token counts.
type TokenBreakdown struct {
	CurrentTokens         int64
	SystemTokens          int64 // instruction file overhead — key metric for Phase 1
	ConversationTokens    int64
	ToolDefinitionsTokens int64
}

// ModelMetric is the per-model billing summary for one session.
type ModelMetric struct {
	Model            string
	InputTokens      int64
	OutputTokens     int64
	NanoAIU          int64
	CacheReadTokens  int64 // phase 6: prompt caching reads
	CacheWriteTokens int64 // phase 6: prompt caching writes
	ReasoningTokens  int64 // phase 6: extended thinking tokens
}

// BillingTime returns the time used to attribute a session to a calendar month.
// It is the EndTime (shutdown time) when set, otherwise StartTime.
//
// Per the data-source discovery findings, spend is attributed to the calendar month in which it is
// finalized — i.e. when the session shuts down — not when it started. A long
// session that begins late in one month but settles in the next belongs to the
// later month's budget. Active sessions have no EndTime yet; they fall back to
// StartTime, which is correct because in-progress sessions belong to the current
// month regardless.
func (s Session) BillingTime() time.Time {
	if !s.EndTime.IsZero() {
		return s.EndTime
	}
	return s.StartTime
}

// TotalInputTokens returns the sum of input tokens across all models used in the session.
func (s Session) TotalInputTokens() int64 {
	var n int64
	for _, m := range s.ModelMetrics {
		n += m.InputTokens
	}
	return n
}

// TotalOutputTokens returns the sum of output tokens across all models used in the session.
func (s Session) TotalOutputTokens() int64 {
	var n int64
	for _, m := range s.ModelMetrics {
		n += m.OutputTokens
	}
	return n
}

// TotalCacheReadTokens returns the sum of prompt-cache read tokens across all
// models used in the session.
func (s Session) TotalCacheReadTokens() int64 {
	var n int64
	for _, m := range s.ModelMetrics {
		n += m.CacheReadTokens
	}
	return n
}

// TotalCacheWriteTokens returns the sum of prompt-cache write tokens across all
// models used in the session.
func (s Session) TotalCacheWriteTokens() int64 {
	var n int64
	for _, m := range s.ModelMetrics {
		n += m.CacheWriteTokens
	}
	return n
}

// TotalReasoningTokens returns the sum of extended-thinking (reasoning) tokens
// across all models used in the session.
func (s Session) TotalReasoningTokens() int64 {
	var n int64
	for _, m := range s.ModelMetrics {
		n += m.ReasoningTokens
	}
	return n
}

// Collector is a source of sessions. Each implementation knows how to read one
// upstream data source (Copilot CLI state, IDE usage, etc.) and return sessions
// already stamped with their Source. ReadAll runs every registered Collector and
// merges the results. Collect must never panic; a source that cannot be read
// returns an error and the others still contribute.
type Collector interface {
	// Name returns the stable Source identifier this collector stamps on sessions.
	Name() string
	// Collect reads and returns this source's sessions. An error aborts only this
	// source; ReadAll logs it and continues with the remaining collectors.
	Collect() ([]Session, error)
}

// cliCollector reads GitHub Copilot CLI session-state via the existing readAll
// core. It is the primary live source.
type cliCollector struct{}

// Name implements Collector.
func (cliCollector) Name() string { return "cli" }

// Collect implements Collector by reading every session under SessionStateDir.
func (cliCollector) Collect() ([]Session, error) {
	stateDir, err := platform.SessionStateDir()
	if err != nil {
		return nil, fmt.Errorf("session: cannot determine session-state directory: %w", err)
	}
	sessions, err := readAll(stateDir)
	if err != nil {
		return nil, err
	}
	// Stamp TokenCostSource as authoritative for CLI (from session.shutdown billing).
	for i := range sessions {
		sessions[i].TokenCostSource = "authoritative"
	}
	return sessions, nil
}

// ideCollector reads GitHub Copilot IDE Chat sessions via the Nitrite SDK
// or falls back to metadata-only parsing from JSON cache.
// Implementation details are in ide_collector.go.
type ideCollector struct{}

// Name implements Collector.
func (ideCollector) Name() string { return "ide-chat" }

// Collect implements Collector by reading IDE sessions from Nitrite DB or JSON metadata.
// If the IDE DB is missing, it returns nil error (not a hard failure — IDE is optional in Phase 6).
// See ide_collector.go for implementation.
func (ideCollector) Collect() ([]Session, error) {
	return newIDECollector().Collect()
}

// collectors is the ordered set of sources ReadAll merges. CLI first so that, all
// else equal, a CLI record is encountered before an IDE record for the same id.
var collectors = []Collector{cliCollector{}, ideCollector{}}

// ReadAll runs every registered Collector, concatenates their sessions,
// deduplicates by {source}:{sessionId} across all sources, and returns the survivors
// sorted by BillingTime descending (newest first).
//
// A collector that fails is logged to stderr and skipped — it does not abort the
// merge. A single unreadable session directory inside a collector is likewise
// logged and skipped. ReadAll returns an error only if it cannot produce any
// result at all (today: only when the sole live collector fails).
//
// Dedup rule: sessions are keyed by {source}:{ID} to prevent false collapse of
// CLI and IDE sessions that may share an ID across different products.
// When two records share a {source}:{ID} key the winner is:
//  1. the one with IsFinal == true (settled billing beats a live snapshot); else
//  2. the one with the higher TotalNanoAIU (the more complete reading).
//
// For CLI-only data every ID is unique per source, so this is a no-op and existing
// behavior is preserved.
func ReadAll() ([]Session, error) {
	var merged []Session
	var firstErr error
	for _, c := range collectors {
		got, err := c.Collect()
		if err != nil {
			// Log IDE collection errors but don't fail ReadAll; CLI is the authoritative source.
			log.Printf("session: collector %q failed: %v", c.Name(), err)
			if firstErr == nil && c.Name() != "copilot-ide" {
				// Only record non-IDE errors as critical
				firstErr = err
			}
			continue
		}
		// Stamp Source defensively; collectors should set it, but never trust.
		for i := range got {
			if got[i].Source == "" {
				got[i].Source = c.Name()
			}
		}
		merged = append(merged, got...)
	}

	// If nothing was collected and at least one collector errored, surface it so
	// callers can distinguish "no sessions" from "could not read sessions".
	if len(merged) == 0 && firstErr != nil {
		return nil, firstErr
	}

	deduped := dedupBySourceAndID(merged)

	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].BillingTime().After(deduped[j].BillingTime())
	})

	return deduped, nil
}

// dedupBySourceAndID collapses sessions sharing a {source}:{ID} tuple to a single
// record per the ReadAll dedup rule (final wins; else higher TotalNanoAIU).
// Sessions with an empty ID are passed through untouched (they cannot be keyed).
// Order of survivors is not guaranteed; ReadAll sorts afterwards.
func dedupBySourceAndID(sessions []Session) []Session {
	best := make(map[string]Session, len(sessions))
	var unkeyed []Session
	var order []string

	for _, s := range sessions {
		if s.ID == "" {
			unkeyed = append(unkeyed, s)
			continue
		}
		key := fmt.Sprintf("%s:%s", s.Source, s.ID)
		prev, ok := best[key]
		if !ok {
			best[key] = s
			order = append(order, key)
			continue
		}
		if preferSession(prev, s) {
			best[key] = s
		}
	}

	out := make([]Session, 0, len(order)+len(unkeyed))
	for _, key := range order {
		out = append(out, best[key])
	}
	out = append(out, unkeyed...)
	return out
}

// preferSession reports whether candidate should replace current under the dedup
// rule: a final reading beats a non-final one; otherwise the higher TotalNanoAIU
// wins. Ties keep the incumbent (deterministic).
func preferSession(current, candidate Session) bool {
	if candidate.IsFinal != current.IsFinal {
		return candidate.IsFinal // promote candidate only if it is the final one
	}
	return candidate.TotalNanoAIU > current.TotalNanoAIU
}

// ReadThisMonth returns sessions whose BillingTime falls in the current calendar month.
// Billing is attributed to the month a session finalizes (EndTime), falling back to
// StartTime for active sessions — see Session.BillingTime. Both year AND month are
// checked to handle year boundaries correctly.
func ReadThisMonth() ([]Session, error) {
	all, err := ReadAll()
	if err != nil {
		return nil, err
	}
	// Compare in UTC to match the analytics bucketing (which normalizes
	// BillingTime to UTC). Using local time here would mis-attribute sessions
	// near a month boundary relative to the buckets.
	now := time.Now().UTC()
	var result []Session
	for _, s := range all {
		bt := s.BillingTime().UTC()
		if bt.Year() == now.Year() && bt.Month() == now.Month() {
			result = append(result, s)
		}
	}
	return result, nil
}

// ReadSince returns sessions whose StartTime is at or after t.
func ReadSince(t time.Time) ([]Session, error) {
	all, err := ReadAll()
	if err != nil {
		return nil, err
	}
	var result []Session
	for _, s := range all {
		if !s.StartTime.Before(t) { // equivalent to After(t) || Equal(t)
			result = append(result, s)
		}
	}
	return result, nil
}

// readAll is the testable core of ReadAll — accepts an explicit stateDir.
func readAll(stateDir string) ([]Session, error) {
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		return nil, fmt.Errorf("session: cannot read %q: %w", stateDir, err)
	}

	var sessions []Session
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sessionDir := filepath.Join(stateDir, entry.Name())
		s, err := readSession(entry.Name(), sessionDir)
		if err != nil {
			log.Printf("session: skipping %s: %v", entry.Name(), err)
			continue
		}
		sessions = append(sessions, s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartTime.After(sessions[j].StartTime)
	})

	return sessions, nil
}

// readSession parses one session directory into a Session.
func readSession(uuid, sessionDir string) (Session, error) {
	s := Session{ID: uuid, Source: "cli"}

	// Detect active session: presence of any inuse.*.lock file.
	locks, err := filepath.Glob(filepath.Join(sessionDir, "inuse.*.lock"))
	if err == nil && len(locks) > 0 {
		s.IsActive = true
	}

	// workspace.yaml provides WorkspaceDir without parsing JSONL — fast fallback.
	s.WorkspaceDir = readWorkspaceCWD(sessionDir)

	// Parse events.jsonl for billing and timing fields.
	if err := parseEventsFile(sessionDir, &s); err != nil {
		return s, err
	}

	if s.WorkspaceDir != "" {
		s.ProjectName = filepath.Base(s.WorkspaceDir)
	}

	return s, nil
}

// parseEventsFile scans events.jsonl and populates billing fields on s.
// Uses a 1 MB scanner buffer to handle large sessions without truncation.
func parseEventsFile(sessionDir string, s *Session) error {
	eventsFile := filepath.Join(sessionDir, "events.jsonl")
	f, err := os.Open(eventsFile)
	if err != nil {
		return fmt.Errorf("open events.jsonl: %w", err)
	}
	defer f.Close()

	const bufSize = 1 << 20 // 1 MB
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, bufSize), bufSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Decode the envelope cheaply to read the type field.
		var envelope struct {
			Type      string          `json:"type"`
			Timestamp string          `json:"timestamp"`
			Data      json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			continue // skip malformed lines silently
		}

		switch envelope.Type {
		case "session.start":
			applyStartEvent(envelope.Data, s)
		case "session.shutdown":
			applyShutdownEvent(envelope.Data, envelope.Timestamp, s)
		default:
			// Any other event may carry a running billing/token snapshot.
			// Apply it as the latest live reading so active sessions display
			// in-progress spend instead of zeros. Events are appended
			// chronologically, so last-write-wins is correct. A partial
			// snapshot must never overwrite a final (shutdown) reading.
			applySnapshotEvent(envelope.Data, s)
		}
	}

	return scanner.Err()
}

// applyStartEvent populates StartTime and WorkspaceDir from a session.start event.
func applyStartEvent(raw json.RawMessage, s *Session) {
	var data struct {
		StartTime string `json:"startTime"`
		Context   struct {
			CWD string `json:"cwd"`
		} `json:"context"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return
	}
	if s.StartTime.IsZero() {
		if t := parseTime(data.StartTime); !t.IsZero() {
			s.StartTime = t
		}
	}
	if s.WorkspaceDir == "" && data.Context.CWD != "" {
		s.WorkspaceDir = data.Context.CWD
	}
}

// billingData is the shared shape of the billing/token payload carried by both
// session.shutdown events and running-snapshot events. Decoding through one
// struct keeps the field set identical between the final and live code paths.
type billingData struct {
	TotalNanoAiu          int64                      `json:"totalNanoAiu"`
	TotalPremiumRequests  int64                      `json:"totalPremiumRequests"` // only on session.shutdown
	SessionStartTime      int64                      `json:"sessionStartTime"`     // Unix ms fallback
	CurrentModel          string                     `json:"currentModel"`
	CurrentTokens         int64                      `json:"currentTokens"`
	SystemTokens          int64                      `json:"systemTokens"`
	ConversationTokens    int64                      `json:"conversationTokens"`
	ToolDefinitionsTokens int64                      `json:"toolDefinitionsTokens"`
	ModelMetrics          map[string]json.RawMessage `json:"modelMetrics"`
}

// hasBillingSignal reports whether the payload actually carries spend/usage data
// worth applying. Used to ignore non-billing events on the snapshot path.
func (d billingData) hasBillingSignal() bool {
	return d.TotalNanoAiu > 0 || d.CurrentTokens > 0
}

// applyBilling overwrites TotalNanoAIU, Tokens, ModelMetrics and PrimaryModel on s
// from the decoded payload. Shared by the shutdown and snapshot code paths.
func applyBilling(data billingData, s *Session) {
	s.TotalNanoAIU = data.TotalNanoAiu
	s.Tokens = TokenBreakdown{
		CurrentTokens:         data.CurrentTokens,
		SystemTokens:          data.SystemTokens,
		ConversationTokens:    data.ConversationTokens,
		ToolDefinitionsTokens: data.ToolDefinitionsTokens,
	}

	// Build ModelMetrics; derive PrimaryModel as the model with highest NanoAIU.
	// Reset first so re-applying a later snapshot does not accumulate duplicates.
	s.ModelMetrics = nil
	var bestNano int64
	s.PrimaryModel = data.CurrentModel // sensible default
	for modelName, mRaw := range data.ModelMetrics {
		var m struct {
			TotalNanoAiu int64 `json:"totalNanoAiu"`
			Usage        struct {
				InputTokens      int64 `json:"inputTokens"`
				OutputTokens     int64 `json:"outputTokens"`
				CacheReadTokens  int64 `json:"cacheReadTokens"`
				CacheWriteTokens int64 `json:"cacheWriteTokens"`
				ReasoningTokens  int64 `json:"reasoningTokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(mRaw, &m); err != nil {
			continue
		}
		s.ModelMetrics = append(s.ModelMetrics, ModelMetric{
			Model:            modelName,
			InputTokens:      m.Usage.InputTokens,
			OutputTokens:     m.Usage.OutputTokens,
			NanoAIU:          m.TotalNanoAiu,
			CacheReadTokens:  m.Usage.CacheReadTokens,
			CacheWriteTokens: m.Usage.CacheWriteTokens,
			ReasoningTokens:  m.Usage.ReasoningTokens,
		})
		if m.TotalNanoAiu > bestNano {
			bestNano = m.TotalNanoAiu
			s.PrimaryModel = modelName
		}
	}
}

// applyShutdownEvent populates all billing fields from a session.shutdown event
// and marks the reading as final/authoritative.
func applyShutdownEvent(raw json.RawMessage, topTimestamp string, s *Session) {
	var data billingData
	if err := json.Unmarshal(raw, &data); err != nil {
		return
	}

	applyBilling(data, s)
	// totalPremiumRequests is only carried by the shutdown event, so capture it
	// here rather than in the shared applyBilling (snapshot events do not have it).
	s.TotalPremiumRequests = data.TotalPremiumRequests
	s.IsFinal = true // shutdown billing is settled and authoritative

	// EndTime from the shutdown event's top-level timestamp.
	if t := parseTime(topTimestamp); !t.IsZero() {
		s.EndTime = t
	}

	// Fallback StartTime from shutdown payload (Unix epoch milliseconds).
	if s.StartTime.IsZero() && data.SessionStartTime > 0 {
		s.StartTime = time.UnixMilli(data.SessionStartTime).UTC()
	}
}

// applySnapshotEvent applies a running billing/token snapshot from a non-start,
// non-shutdown event. It does nothing if the session already has a final
// (shutdown) reading, or if the event carries no billing signal. This lets
// active sessions display live in-progress spend without depending on knowing
// every event name; the last billing-bearing event wins.
func applySnapshotEvent(raw json.RawMessage, s *Session) {
	if s.IsFinal {
		return // never let a partial snapshot overwrite a final reading
	}
	var data billingData
	if err := json.Unmarshal(raw, &data); err != nil {
		return
	}
	if !data.hasBillingSignal() {
		return
	}
	applyBilling(data, s) // leaves IsFinal == false (partial/live)
}

// readWorkspaceCWD reads the cwd field from workspace.yaml using a simple line scan.
// Returns empty string if the file is absent or does not contain a cwd field.
// No YAML library needed — the field format is always "cwd: <path>".
func readWorkspaceCWD(sessionDir string) string {
	f, err := os.Open(filepath.Join(sessionDir, "workspace.yaml"))
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cwd:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "cwd:"))
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("session: error reading workspace.yaml in %s: %v", filepath.Dir(f.Name()), err)
	}
	return ""
}

// parseTime parses an ISO 8601 / RFC 3339 timestamp string.
// Tries RFC3339Nano first (handles milliseconds), then RFC3339.
// Returns zero time on failure — callers check IsZero().
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}
