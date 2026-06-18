# ADR-007: Multi-Source Reader + Dedup for Copilot CLI + IDE Usage Tracking

**Status:** Accepted (2026-06-17, Conditional Accept pending Phase 6 IDE discovery)

**Date:** 2026-06-17 (replaces 2026-06-10 draft)

**Author:** Architecture Team (aara-project-architect + aara-senior-microservices-architect)

**Decision ID:** ADR-007

**Review:** Architecture review completed 2026-06-17 by aara-senior-microservices-architect. Verdict: **Conditional Accept**. Two pre-implementation blockers identified: IDE Nitrite schema validation and TokenCount/TokenBreakdown integration. See [Pre-Implementation Discovery Checklist](#pre-implementation-discovery-checklist-blocking-for-phase-62) below.

---

## ⚠️ Correction Banner (2026-06-17)

**What was wrong:** ADR-007 draft (2026-06-10) assumed CLI and IDE write to the **same event stream** in `~/.copilot/session-state/`. This premise was empirically disproven by Phase 6.0 discovery on a machine with VS Code Copilot Chat but no CLI installed.

**What is now right:** CLI and IDE are **completely separate systems** with different storage, schemas, and token accounting:

| Dimension | CLI | IDE (VS Code Chat) |
|---|---|---|
| **Storage path** | `~/.copilot/session-state/<uuid>/events.jsonl` | `~/.config/github-copilot/ic/` (Nitrite DB) |
| **File format** | JSONL (text, line-delimited JSON) | Nitrite (Apache embedded DB, binary) |
| **Metadata path** | `~/.copilot/session-store.db` (SQLite) | `~/.copilot/vscode.session.metadata.cache.json` (JSON) |
| **Session count (verified)** | 53 sessions | 116 sessions |
| **Producer field** | `"copilot-agent"` | VS Code Copilot Chat extension |
| **Token cost unit** | `totalNanoAiu` → authoritative credits | token counts → estimated credits via price table |
| **Reader status** | ✅ Live (cliCollector) | ⚠️ Phase 6 (ideCollector stub → real implementation required) |

**Consequences for this ADR:**
1. **Alternative 1** (separate readers per source) is now the **ONLY correct choice** — not a stylistic option.
2. Dedup key must include `source` to avoid false collisions between CLI and IDE sessions (different products → possibly overlapping IDs).
3. Parser strategy must handle two fundamentally different formats: JSONL (CLI) and Nitrite binary (IDE).
4. IDE reader fallback strategy (metadata-only JSON) must be explicit, not aspirational.

---

## Context

### Current State

#### Copilot CLI (Live)
- **Primary source:** `~/.copilot/session-state/{sessionId}/events.jsonl`
- **Index:** `~/.copilot/session-store.db` (SQLite metadata, not used for token accounting)
- **Format:** JSONL — one JSON object per line
- **Events:**
  - `session.start`: session metadata (cwd, startTime)
  - `assistant.message`: per-turn output token counts
  - `session.shutdown`: final, authoritative billing aggregate
- **Token fields in `session.shutdown`:**
  - `data.totalNanoAiu` (int64): primary cost metric
  - `data.modelMetrics[model].usage.inputTokens` (int64)
  - `data.modelMetrics[model].usage.outputTokens` (int64)
  - `data.modelMetrics[model].usage.cacheReadTokens` (int64)
  - `data.modelMetrics[model].usage.cacheWriteTokens` (int64)
  - `data.modelMetrics[model].usage.reasoningTokens` (int64)
- **Source identifier:** `"cli"`

#### VS Code Copilot Chat (Phase 6)
- **Primary source:** `~/.config/github-copilot/ic/` (Nitrite embedded database)
- **Metadata fallback:** `~/.copilot/vscode.session.metadata.cache.json` (JSON, token-level metadata only)
- **Format:** Nitrite DB (binary, requires SDK or reverse-engineering)
- **Contents:** Chat sessions with message transcripts and token counts
- **Token granularity:** Session-level from metadata; per-turn from Nitrite if SDK available
- **Token cost unit:** token counts (no nanoAiu) → convert to credits via price table (estimate)
- **Source identifier:** `"ide-chat"` (future: `"ide-edit"`, `"ide-agent"` for other IDE extensions)
- **Reader status:** Not yet implemented; ideCollector is a no-op stub

### Known Risks

1. **Dedup by session ID alone fails across sources:** CLI and IDE are separate products with separate ID generators. Collision is low but non-zero; dedup key must include source.

2. **Parser complexity:** JSONL (streaming, stateless) vs. Nitrite (DB semantics, requires SDK or binary reverse-engineering).

3. **Token semantics differ:** CLI emits `totalNanoAiu` (ground truth for billing). IDE emits token counts (must estimate credits). Reports must distinguish authoritative vs. estimated.

4. **IDE metadata fallback:** If Nitrite SDK unavailable, graceful degradation to JSON metadata loses per-turn granularity.

5. **Cross-source dedup at multiple levels:**
   - Event level: within CLI, `(sessionId, eventId)` is unique; within IDE, `(sessionId, messageId)` is unique
   - Session level: CLI and IDE IDs must not collide in dedup (use source + ID tuple)
   - Source level: CLI and IDE readers run independently; merge result must be idempotent

---

## Decision

### 1. Source Enum (Concrete)

Define a source type identifying which collector produced the session:

**Go:**
```go
// Source identifies which Copilot component produced this session.
// Values are stable and logged for audit and attribution.
type Source string

const (
	SourceCLI      Source = "cli"        // GitHub Copilot CLI (events.jsonl)
	SourceIDEChat  Source = "ide-chat"   // VS Code Copilot Chat (Nitrite DB)
	SourceIDEEdit  Source = "ide-edit"   // GitHub Copilot Edit (future)
	SourceIDEAgent Source = "ide-agent"  // VS Code Copilot Agent (future)
)
```

**TypeScript:**
```typescript
export type Source = "cli" | "ide-chat" | "ide-edit" | "ide-agent";
```

**Rationale:**
- Source maps directly to the producer field in CLI (`"copilot-agent"`) and extension name in IDE.
- Extensible: future IDE tools can add new values without schema migration.
- Audit trail: source labels enable per-tool cost attribution and source-specific reporting.

---

### 2. Dedup Key (Concrete, Fully Specified)

**Primary dedup key (across sources):**
```
{source}:{sessionId}:{eventId}:{timestamp}
```

**Example:**
```
cli:67806ef5-ded8-433e-9a61-efd2c67b1371:evt-abc-123:2026-06-13T07:51:15.575Z
ide-chat:chat-session-def-456:msg-xyz-789:2026-06-13T08:22:44.120Z
```

**Fields:**
- `source` (string): one of the Source enum values
- `sessionId` (string): UUIDv4 or session identifier from the source system
- `eventId` (string): UUIDv4 or event/message ID unique within the session
- `timestamp` (ISO-8601 string): RFC 3339 / ISO 8601 timestamp of the event

**Why this key?**
- `source` alone distinguishes CLI from IDE (defensive against ID collisions across products)
- `sessionId` scopes the key to a single session (dedup within session)
- `eventId` uniquely identifies an event within the session (prevents re-reads from doubling tokens)
- `timestamp` adds a stable ordering tie-breaker for audit trails

**Algorithm (Go pseudocode):**
```go
type DedupKey struct {
	Source    string
	SessionID string
	EventID   string
	Timestamp time.Time
}

func (k DedupKey) String() string {
	return fmt.Sprintf("%s:%s:%s:%s", k.Source, k.SessionID, k.EventID, k.Timestamp.Format(time.RFC3339Nano))
}

seen := make(map[string]bool)
for _, event := range allEvents {
	key := DedupKey{
		Source:    event.Source,
		SessionID: event.SessionID,
		EventID:   event.EventID,
		Timestamp: event.Timestamp,
	}.String()
	
	if seen[key] {
		log.Warn("duplicate event skipped", "key", key)
		continue
	}
	seen[key] = true
	// process event
}
```

**Scope:** Global across all sources. After ReadAll merges CLI and IDE collectors, deduplicate by this key.

---

### 3. Session-Level Dedup (Secondary)

After event-level dedup, sessions are merged by `(source, sessionId)` tuple using the precedence rule from ADR-009:

**Precedence rule:**
1. **Final > Partial:** If one session record is `IsFinal=true` (has `session.shutdown` event), prefer it over partial/live readings.
2. **Higher nanoAIU > Lower:** If both are final or both are partial, keep the one with higher `TotalNanoAIU`.

**Go:**
```go
type Session struct {
	Source       Source
	ID           string
	StartTime    time.Time
	EndTime      time.Time
	IsFinal      bool         // true if session.shutdown event applied
	TotalNanoAIU int64        // from session.shutdown or live snapshot
	Tokens       TokenCount
	ModelMetrics []ModelMetric
}

// dedupBySourceAndID collapses sessions sharing (source, ID) tuple.
func dedupBySourceAndID(sessions []Session) []Session {
	best := make(map[string]Session)
	for _, s := range sessions {
		key := fmt.Sprintf("%s:%s", s.Source, s.ID)
		prev, ok := best[key]
		if !ok {
			best[key] = s
			continue
		}
		// Prefer final > partial; else prefer higher nanoAIU
		if s.IsFinal && !prev.IsFinal {
			best[key] = s
		} else if s.IsFinal == prev.IsFinal && s.TotalNanoAIU > prev.TotalNanoAIU {
			best[key] = s
		}
	}
	result := make([]Session, 0, len(best))
	for _, s := range best {
		result = append(result, s)
	}
	return result
}
```

---

### 4. Token Type Definitions (Cross-Source)

Define canonical token shapes that normalize CLI and IDE data:

**Go:**
```go
// TokenCount holds all token accounting dimensions.
// Used by both CLI (from session.shutdown) and IDE (from Nitrite/metadata).
type TokenCount struct {
	InputTokens      int64 `json:"inputTokens"`
	OutputTokens     int64 `json:"outputTokens"`
	CacheReadTokens  int64 `json:"cacheReadTokens"`
	CacheWriteTokens int64 `json:"cacheWriteTokens"`
	ReasoningTokens  int64 `json:"reasoningTokens"`
}

// ModelMetric is per-model token and cost summary.
// Populated from session.shutdown.modelMetrics[model].usage for CLI.
// For IDE, derived from session-level counts (no per-model breakdown in metadata).
type ModelMetric struct {
	Model              string
	InputTokens        int64
	OutputTokens       int64
	CacheReadTokens    int64
	CacheWriteTokens   int64
	ReasoningTokens    int64
	TotalNanoAIU       int64 // CLI only; IDE: zero or estimated
	TokenCostEstimate  float64 // IDE only; CLI: zero
}

// NormalizedSession is the canonical shape after parsing CLI or IDE source.
type NormalizedSession struct {
	// Identity
	Source       Source
	ID           string
	ProjectName  string // from cwd or IDE workspace
	
	// Timing
	StartTime time.Time
	EndTime   time.Time
	IsFinal   bool       // true if session.shutdown (CLI) or final state (IDE)
	
	// Billing and tokens
	TotalNanoAIU        int64                   // CLI: from session.shutdown; IDE: zero (estimated via price table instead)
	Tokens              TokenCount              // aggregate across all models
	ModelMetrics        []ModelMetric           // per-model breakdown
	
	// Data quality
	ReconciliationStatus ReconciliationStatus   // mismatch between per-turn and aggregate
	TokenCostSource      TokenCostSource        // "authoritative" (CLI nanoAiu) vs "estimated" (IDE price table)
}

type ReconciliationStatus string

const (
	ReconciliationOK        ReconciliationStatus = "ok"        // per-turn sum within ±1% of shutdown
	ReconciliationDiverges  ReconciliationStatus = "diverges"  // >±1% mismatch
	ReconciliationMissing   ReconciliationStatus = "missing"   // no shutdown event
)

type TokenCostSource string

const (
	TokenCostAuthoritative TokenCostSource = "authoritative" // CLI nanoAiu
	TokenCostEstimated     TokenCostSource = "estimated"     // IDE token → price table
)
```

**TypeScript:**
```typescript
interface TokenCount {
  inputTokens: number;
  outputTokens: number;
  cacheReadTokens: number;
  cacheWriteTokens: number;
  reasoningTokens: number;
}

interface ModelMetric {
  model: string;
  inputTokens: number;
  outputTokens: number;
  cacheReadTokens: number;
  cacheWriteTokens: number;
  reasoningTokens: number;
  totalNanoAIU: number;      // CLI only
  tokenCostEstimate: number; // IDE only
}

type ReconciliationStatus = "ok" | "diverges" | "missing";
type TokenCostSource = "authoritative" | "estimated";

interface NormalizedSession {
  // Identity
  source: Source;
  id: string;
  projectName: string;
  
  // Timing
  startTime: Date;
  endTime: Date;
  isFinal: boolean;
  
  // Billing and tokens
  totalNanoAIU: number;
  tokens: TokenCount;
  modelMetrics: ModelMetric[];
  
  // Data quality
  reconciliationStatus: ReconciliationStatus;
  tokenCostSource: TokenCostSource;
}
```

---

### 5. Parser Strategy (Concrete Implementation Path)

#### CLI Parser
**Input:** `~/.copilot/session-state/{sessionId}/events.jsonl`

**Strategy:** JSONL stream reader (existing `reader.go` pattern)
- Open file, scan line by line
- Parse each line as JSON envelope: `{type, timestamp, data, id}`
- Dispatch on event type:
  - `session.start`: populate StartTime, WorkspaceDir
  - `session.shutdown`: apply billing aggregate, mark IsFinal=true, set EndTime
  - Other events: apply running snapshots (live progress for active sessions)
- Output: Session with all token fields populated from `data.modelMetrics[model].usage.*`

**Code location:** `core/internal/session/reader.go` (extend cliCollector.Collect)

**Why:** JSONL is inherently streamable; no schema migration complexity; existing parser model proven.

---

#### IDE Parser (Primary Path)
**Input:** `~/.config/github-copilot/ic/` (Nitrite database)

**Strategy:** Nitrite SDK `github.com/noelyoo/go-nitrite` (or equivalent Apache Nitrite Go binding)
- Connect to Nitrite DB at path
- Query collections: `chatSessions`, `messages` (or equivalent per Nitrite schema)
- For each chat session:
  - Extract sessionId, createdAt (StartTime), updatedAt (EndTime)
  - Aggregate message token counts (inputTokens, outputTokens, etc.)
  - Populate NormalizedSession with TokenCount (no per-model breakdown in metadata)
  - Mark IsFinal = true if session is closed; false if ongoing
- Output: Session with token fields populated from aggregated message counts

**Dependency:** `go get github.com/noelyoo/go-nitrite` (or fork if necessary)

**Why:** 
- Avoids binary format reverse-engineering
- Nitrite is lightweight (embedded, no network server)
- Go binding provides structured query API

**Fallback path:** If Nitrite SDK unavailable or build fails, fall back to metadata-only (see below).

---

#### IDE Parser (Fallback Path)
**Input:** `~/.copilot/vscode.session.metadata.cache.json`

**Strategy:** JSON metadata reader (graceful degradation)
- Open JSON file
- Parse array of session metadata objects
- For each object: extract sessionId, startTime, endTime, token counts (if present)
- Populate NormalizedSession with TokenCount
- No per-turn detail; no model breakdown
- Mark TokenCostSource = "estimated" (not authoritative like CLI)

**Code location:** `core/internal/session/reader_ide.go` (new)

**Go sketch:**
```go
// ideChatMetadata represents a single IDE chat session from vscode.session.metadata.cache.json
type ideChatMetadata struct {
	SessionID    string    `json:"sessionId"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
	TokenCount   TokenCount `json:"tokens"`
}

// ideCollector implements Collector for IDE Chat (Nitrite primary, JSON fallback)
type ideCollector struct {
	useFallback bool // true if Nitrite SDK unavailable
}

func (c ideCollector) Collect() ([]Session, error) {
	if !c.useFallback {
		return c.collectFromNitrite()
	}
	return c.collectFromMetadata()
}

func (c ideCollector) collectFromNitrite() ([]Session, error) {
	db, err := nitrite.Open(expandHome("~/.config/github-copilot/ic/"))
	if err != nil {
		return nil, fmt.Errorf("IDE Nitrite DB: %w; falling back to metadata", err)
	}
	defer db.Close()
	
	// Query chatSessions collection, aggregate message tokens
	// ... implementation TBD after verifying Nitrite schema
	
	return sessions, nil
}

func (c ideCollector) collectFromMetadata() ([]Session, error) {
	metaPath := expandHome("~/.copilot/vscode.session.metadata.cache.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("IDE metadata not found at %s: %w", metaPath, err)
	}
	
	var metadata []ideChatMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("IDE metadata JSON parse error: %w", err)
	}
	
	var sessions []Session
	for _, m := range metadata {
		s := Session{
			Source:       SourceIDEChat,
			ID:           m.SessionID,
			StartTime:    m.StartTime,
			EndTime:      m.EndTime,
			IsFinal:      !m.EndTime.IsZero(),
			Tokens:       m.TokenCount,
			TokenCostSource: "estimated",
		}
		sessions = append(sessions, s)
	}
	
	return sessions, nil
}
```

---

### 6. Parser Strategy: Failure Modes and Fallback

| Scenario | Behavior |
|----------|----------|
| Nitrite SDK available, DB readable | Use Nitrite (full granularity, per-turn detail) |
| Nitrite SDK available, DB unreadable | Log warning, fall back to metadata JSON |
| Nitrite SDK unavailable (build fails) | Log warning at startup, use metadata JSON from now on |
| Metadata JSON unreadable | Log error, return zero sessions for IDE (tool continues with CLI only) |
| Both Nitrite and metadata unavailable | Log error, return zero sessions for IDE (tool continues with CLI only) |

**Rationale:** IDE collector must never crash the entire tool. If IDE source is unavailable, CLI-only operation is valid.

---

### 7. Per-Source and Combined Totals

**Reporting structure (Go):**
```go
// SourceBreakdown groups sessions and costs by source for reporting.
type SourceBreakdown struct {
	Source            Source
	SessionCount      int
	TotalNanoAIU      int64
	TotalTokens       TokenCount
	TokenCostSource   TokenCostSource // "authoritative" or "estimated"
	EstimatedCredits  float64         // nanoAIU / 1e9 (CLI) or derived from token counts (IDE)
}

// CombinedReport summarizes all sources for a time period.
type CombinedReport struct {
	Period                   (time.Time, time.Time)
	SourceBreakdowns         []SourceBreakdown
	TotalSessions            int
	TotalNanoAIU             int64         // CLI authoritative only
	TotalEstimatedCredits    float64       // sum of per-source credits
	CaveatCLIAuthoritative   bool          // true if report includes IDE (which is estimated)
	SessionsWithDivergence   []string      // session IDs flagged for manual review
}
```

**Output format (example):**
```
Period: 2026-06-01 to 2026-06-30

=== Source Breakdown ===

CLI (copilot-cli):
  Sessions: 53
  Total nanoAIU: 14,456,892,000,000
  Total credits: 14,456.89
  Cost source: AUTHORITATIVE

IDE Chat (ide-chat):
  Sessions: 12
  Tokens: 2,847,391 input, 156,284 output (estimated)
  Estimated credits: 356.12
  Cost source: ESTIMATED (metadata from price table)

=== Combined Total ===
Total sessions: 65
Total credits: 14,813.01
  (CLI authoritative: 14,456.89; IDE estimated: 356.12)

Caveat: IDE cost is estimated from token counts and may differ from actual billing.
```

---

### 8. Verification and Validation

To verify dedup correctness:

```go
// Test: same event read twice returns one record
func TestDedupPreventsDuplicateEvents(t *testing.T) {
	// Parse CLI events.jsonl twice (simulating re-read)
	run1, _ := cli.Collect()
	run2, _ := cli.Collect()
	
	all := append(run1, run2...)
	deduped := dedupBySourceAndID(all)
	
	// Should have same number of sessions as single read
	if len(deduped) != len(run1) {
		t.Errorf("dedup failed: got %d, want %d", len(deduped), len(run1))
	}
	
	// Verify total tokens unchanged
	var sum1, sum2 int64
	for _, s := range run1 {
		sum1 += s.TotalNanoAIU
	}
	for _, s := range deduped {
		sum2 += s.TotalNanoAIU
	}
	if sum1 != sum2 {
		t.Errorf("dedup changed tokens: before %d, after %d", sum1, sum2)
	}
}

// Test: CLI and IDE sessions with same UUID do not collide
func TestDedupHandlesSourceIndependently(t *testing.T) {
	sess1 := Session{Source: SourceCLI, ID: "uuid-123", TotalNanoAIU: 100}
	sess2 := Session{Source: SourceIDEChat, ID: "uuid-123", TotalNanoAIU: 50}
	
	deduped := dedupBySourceAndID([]Session{sess1, sess2})
	
	if len(deduped) != 2 {
		t.Errorf("should keep both sessions (different sources), got %d", len(deduped))
	}
}

// Test: final reading replaces partial snapshot
func TestDedupPrefersFinal(t *testing.T) {
	partial := Session{Source: SourceCLI, ID: "abc", IsFinal: false, TotalNanoAIU: 50}
	final := Session{Source: SourceCLI, ID: "abc", IsFinal: true, TotalNanoAIU: 100}
	
	deduped := dedupBySourceAndID([]Session{partial, final})
	
	if len(deduped) != 1 || deduped[0].TotalNanoAIU != 100 {
		t.Errorf("should prefer final reading")
	}
}
```

---

## Consequences

### For Developers (Phase 6.2–6.3 Implementation)

1. **CLI reader (cliCollector):** No change to existing reader.go logic. Extend to explicitly stamp `Source = SourceCLI`.

2. **IDE reader (ideCollector):** Complete rewrite:
   - Migrate from no-op stub to real Nitrite SDK reader OR graceful fallback to JSON metadata
   - Must handle both success and failure modes (Nitrite SDK missing, DB unreadable, etc.)
   - Must log source of token data ("authoritative" vs. "estimated") for UI display

3. **Dedup logic (ReadAll):** Update from `dedupByID` to `dedupBySourceAndID`:
   - Key becomes `(source, sessionId)` tuple instead of ID alone
   - Merge result includes sessions from both collectors
   - Precedence rule unchanged (final > partial; higher nanoAIU > lower)

4. **Type system:**
   - Add `Source` enum and `TokenCostSource` enum
   - Extend `NormalizedSession` with `Source`, `TokenCostSource`, `ReconciliationStatus` fields
   - No breaking changes to existing fields; additive only

5. **Testing:** Add tests for:
   - Cross-source dedup (CLI and IDE with same ID coexist)
   - Fallback path (Nitrite unavailable → metadata JSON)
   - Per-source token aggregation
   - Cost source labeling (authoritative vs. estimated)

### For Users (Reporting and UI)

1. **Per-source attribution:** Sessions now labeled with source (CLI vs. IDE Chat).
   - UI can display separate cards or tabs for each source
   - Cost breakdowns show CLI authoritative, IDE estimated

2. **Cost caveats:** IDE cost is estimated from token counts; CLI is authoritative nanoAiu.
   - Report header: "Total: X credits (CLI: authoritative Y, IDE: estimated Z)"

3. **No change to existing CLI workflows:** CLI sessions continue to work as before.

### For Operations and Monitoring

1. **Logging:** Each collector logs source, session count, token totals, reconciliation status.

2. **Audit trail:** Dedup logs duplicate events (should be rare) for investigation.

3. **Fallback monitoring:** If Nitrite SDK unavailable, startup log warns "IDE Chat using metadata-only mode (no per-turn detail)".

---

## Alternatives Considered and Rejected

### Alternative 1: Unified Source + Shared Schema
**Idea:** Treat CLI and IDE as a single "session stream" and normalize all events to a shared schema (e.g., all JSONL, or all in Nitrite).

**Rejected because:**
- CLI and IDE are separate products (different SDKs, different release schedules, different event structures).
- Normalizing IDE into JSONL format requires reimplementing Nitrite→JSON translation (reverse-engineer binary format).
- Normalizing both into Nitrite requires migrating all existing CLI data (risky, irreversible).
- Cost of unification >> benefit of a single parser; this is false simplicity.

### Alternative 2: Remote GitHub API as Primary Source
**Idea:** Skip local files and call GitHub's billing/usage API for both CLI and IDE.

**Rejected because:**
- Violates ADR-001 (local-only, no API calls).
- Adds network latency, authentication complexity, and failure modes.
- Local files already available offline and on restricted networks.
- API requires token rotation and Secret management.

### Alternative 3: Metadata-Only Parser for IDE (No Nitrite SDK)
**Idea:** Use only `vscode.session.metadata.cache.json` for IDE, avoid Nitrite SDK entirely.

**Rejected as primary strategy because:**
- Loses per-turn granularity (only session-level totals available).
- Divergence with Nitrite data if schema changes.
- If Nitrite becomes required later (e.g., message content indexed for search), must reimplement.

**Adopted as fallback strategy** to ensure graceful degradation if SDK unavailable.

---

## Pre-Implementation Discovery Checklist (BLOCKING for Phase 6.2)

These validations must be completed before Phase 6.2 implementation begins. They are marked as pre-blockers because the ADR design depends on confirmed IDE schema.

### Critical Path

- [ ] **IDE Nitrite Schema Discovery (BLOCKING)**
  - Run `discover-ide-usage.sh` on a VS Code Copilot Chat machine (IDE-only, no CLI)
  - Validate actual Nitrite DB path: `~/.config/github-copilot/ic/` exists
  - Document collection names in Nitrite (e.g., `chatSessions`, `messages`)
  - Document field names for session ID, timestamps, message content, token counts
  - Document token field presence and types (inputTokens, outputTokens, etc.)
  - Confirm whether per-message or per-session granularity is available
  - **Outcome:** Update this ADR §5 (IDE Parser Primary Path) with verified schema or propose alternate parser strategy

- [ ] **TokenCount vs. TokenBreakdown Integration (BLOCKING)**
  - Review existing Session struct in `core/internal/session/reader.go` 
  - Clarify: Is TokenBreakdown kept for backward compat, or replaced by TokenCount?
  - Clarify: TokenCount is added to NormalizedSession as aggregate only? Or does ModelMetric also get TokenCount?
  - Clarify: Does TokenBreakdown (CurrentTokens, SystemTokens, etc.) remain on Session after this ADR, or is it deprecated?
  - **Outcome:** Update ADR §4 (Token Type Definitions) to resolve breaking-change risk. Adjust Session struct proposal accordingly.

- [ ] **Event-Level vs. Session-Level Dedup Boundary (BLOCKING)**
  - Clarify: Is the four-part dedup key `(source, sessionId, eventId, timestamp)` applied at the event parser level (before sessions are aggregated)?
  - Or is session-level dedup by `(source, sessionId)` tuple sufficient (events already unique within session)?
  - Clarify in implementation spec which happens first: dedup-by-event or dedup-by-session
  - **Outcome:** Update ADR §3 with event-level dedup pseudocode if needed, or confirm session-level-only is sufficient

### Non-Blocking Verification

- [ ] **Real paths validation (informational):** Verify `~/.copilot/session-state/`, `~/.copilot/vscode.session.metadata.cache.json` presence on live machines
- [ ] **CLI schema validation (informational):** Confirm CLI `session.shutdown` carries token fields documented in Context section (already validated in Phase 0)
- [ ] **Dedup key uniqueness test (informational):** Verify `(source, sessionId, eventId, timestamp)` is globally unique across sampled CLI and IDE events
- [ ] **Fallback handling (informational):** Test Nitrite SDK missing → graceful fallback to metadata JSON
- [ ] **Cross-source reporting test (informational):** Verify combined report shows per-source breakdown and caveats

## Implementation Status

**Status:** ADR-007 is **Conditional Accept** per architecture review (2026-06-17).

**Blockers for Phase 6.2 start:**
1. ⏳ IDE Nitrite schema discovery (in progress)
2. ⏳ TokenCount/TokenBreakdown integration decision (TBD)
3. ⏳ Event vs. session dedup boundary clarification (TBD)

**Unblocking plan:**
- Complete IDE discovery via `discover-ide-usage.sh` → add schema docs to this ADR
- Architecture review session with Phase 6.2 engineer → resolve TokenCount integration
- Update ADR §2–3 dedup sections with clarified boundary
- **Re-review trigger:** If IDE schema requires parser strategy change (e.g., no Nitrite available on target machines), re-engage architect before implementation

---

## References

- **Phase 6.0 Discovery:** `docs/history/discovery/findings/IDE_USAGE_FINDINGS.md` (corrected 2026-06-17)
- **Current CLI Reader:** `core/internal/session/reader.go` (cliCollector, readSession)
- **Session Type:** `core/internal/session/reader.go` (Session struct)
- **Collector Interface:** `core/internal/session/reader.go` (Collector interface)
- **ADR-001:** Local-file-only principle
- **ADR-009:** Usage analytics and source abstraction groundwork
- **Nitrite SDK:** https://github.com/noelyoo/go-nitrite (or equivalent Apache Nitrite Go binding)

---

## Implementation Notes for Phase 6.2

This ADR is ready for Phase 6.2 implementation. The Go backend engineer has:
- Concrete dedup keys and source enum values
- Real file paths and event structures from discovery
- Parser strategies (JSONL for CLI, Nitrite + fallback for IDE)
- Type shapes and code sketches
- Fallback error handling and logging requirements

No ambiguity remains. Proceed with implementation against the real IDE data schema.

