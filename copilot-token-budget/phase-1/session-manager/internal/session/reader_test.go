package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// makeSessionDir creates a minimal session directory in tmp with the provided events.
func makeSessionDir(t *testing.T, root, uuid string, events []map[string]any, activeLock bool) string {
	t.Helper()
	dir := filepath.Join(root, uuid)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("makeSessionDir: %v", err)
	}

	// Write events.jsonl.
	f, err := os.Create(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatalf("create events.jsonl: %v", err)
	}
	enc := json.NewEncoder(f)
	for _, ev := range events {
		if err := enc.Encode(ev); err != nil {
			t.Fatalf("encode event: %v", err)
		}
	}
	f.Close()

	if activeLock {
		lf, err := os.Create(filepath.Join(dir, "inuse.1234.lock"))
		if err != nil {
			t.Fatalf("create lock: %v", err)
		}
		lf.Close()
	}
	return dir
}

// startEvent returns a synthetic session.start event payload.
func startEvent(startTime, cwd string) map[string]any {
	return map[string]any{
		"type":      "session.start",
		"timestamp": startTime,
		"id":        "evt-start",
		"parentId":  "sess-001",
		"data": map[string]any{
			"sessionId": "sess-001",
			"startTime": startTime,
			"context":   map[string]any{"cwd": cwd},
		},
	}
}

// shutdownEvent returns a synthetic session.shutdown event payload.
func shutdownEvent(endTime string, nanoAIU int64, model string, systemTokens, currentTokens int64) map[string]any {
	return map[string]any{
		"type":      "session.shutdown",
		"timestamp": endTime,
		"id":        "evt-shutdown",
		"parentId":  "sess-001",
		"data": map[string]any{
			"totalNanoAiu":          nanoAIU,
			"currentModel":          model,
			"systemTokens":          systemTokens,
			"currentTokens":         currentTokens,
			"conversationTokens":    int64(5000),
			"toolDefinitionsTokens": int64(3000),
			"modelMetrics": map[string]any{
				model: map[string]any{
					"totalNanoAiu": nanoAIU,
					"usage": map[string]any{
						"inputTokens":  int64(100000),
						"outputTokens": int64(5000),
					},
				},
			},
		},
	}
}

// snapshotEvent returns a synthetic non-start/non-shutdown event whose data
// carries a running billing/token snapshot (the live in-progress reading).
func snapshotEvent(ts string, nanoAIU int64, model string, systemTokens, currentTokens int64) map[string]any {
	return map[string]any{
		"type":      "model.usage", // any non-start/non-shutdown type
		"timestamp": ts,
		"id":        "evt-snapshot",
		"parentId":  "sess-001",
		"data": map[string]any{
			"totalNanoAiu":          nanoAIU,
			"currentModel":          model,
			"systemTokens":          systemTokens,
			"currentTokens":         currentTokens,
			"conversationTokens":    int64(4000),
			"toolDefinitionsTokens": int64(2000),
			"modelMetrics": map[string]any{
				model: map[string]any{
					"totalNanoAiu": nanoAIU,
					"usage": map[string]any{
						"inputTokens":  int64(50000),
						"outputTokens": int64(2500),
					},
				},
			},
		},
	}
}

func TestReadAll_FullSession(t *testing.T) {
	root := t.TempDir()
	uuid := "aaaaaaaa-0000-0000-0000-000000000001"

	start := "2026-06-13T08:00:00.000Z"
	end := "2026-06-13T10:00:00.000Z"
	makeSessionDir(t, root, uuid, []map[string]any{
		startEvent(start, "/home/user/myproject"),
		shutdownEvent(end, 500_000_000_000, "claude-sonnet-4.6", 12000, 35000),
	}, false)

	sessions, err := readAll(root)
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.ID != uuid {
		t.Errorf("ID = %q, want %q", s.ID, uuid)
	}
	if s.WorkspaceDir != "/home/user/myproject" {
		t.Errorf("WorkspaceDir = %q, want %q", s.WorkspaceDir, "/home/user/myproject")
	}
	if s.ProjectName != "myproject" {
		t.Errorf("ProjectName = %q, want %q", s.ProjectName, "myproject")
	}
	if s.TotalNanoAIU != 500_000_000_000 {
		t.Errorf("TotalNanoAIU = %d, want 500000000000", s.TotalNanoAIU)
	}
	if s.PrimaryModel != "claude-sonnet-4.6" {
		t.Errorf("PrimaryModel = %q, want %q", s.PrimaryModel, "claude-sonnet-4.6")
	}
	if s.Tokens.SystemTokens != 12000 {
		t.Errorf("SystemTokens = %d, want 12000", s.Tokens.SystemTokens)
	}
	if s.Tokens.CurrentTokens != 35000 {
		t.Errorf("CurrentTokens = %d, want 35000", s.Tokens.CurrentTokens)
	}
	if s.IsActive {
		t.Error("IsActive = true, want false (no lock file)")
	}
	if s.StartTime.IsZero() {
		t.Error("StartTime is zero")
	}
	if s.EndTime.IsZero() {
		t.Error("EndTime is zero")
	}
}

func TestReadAll_ActiveSession(t *testing.T) {
	root := t.TempDir()
	uuid := "aaaaaaaa-0000-0000-0000-000000000002"
	makeSessionDir(t, root, uuid, []map[string]any{
		startEvent("2026-06-13T09:00:00.000Z", "/home/user/active"),
		// No shutdown event — session is still running
	}, true) // lock file present

	sessions, err := readAll(root)
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	s := sessions[0]
	if !s.IsActive {
		t.Error("IsActive = false, want true (lock file present)")
	}
	if s.TotalNanoAIU != 0 {
		t.Errorf("TotalNanoAIU = %d, want 0 (no shutdown event)", s.TotalNanoAIU)
	}
	if s.IsFinal {
		t.Error("IsFinal = true, want false (no shutdown event)")
	}
}

func TestReadAll_FinalSession_IsFinal(t *testing.T) {
	root := t.TempDir()
	uuid := "aaaaaaaa-0000-0000-0000-00000000000f"
	makeSessionDir(t, root, uuid, []map[string]any{
		startEvent("2026-06-13T08:00:00.000Z", "/home/user/done"),
		shutdownEvent("2026-06-13T10:00:00.000Z", 400_000_000_000, "claude-sonnet-4.6", 1000, 5000),
	}, false)

	sessions, err := readAll(root)
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	if !sessions[0].IsFinal {
		t.Error("IsFinal = false, want true (shutdown event present)")
	}
}

// TestReadAll_RunningSnapshot covers BUG 1: an active session with a running
// snapshot event but no shutdown must surface live billing with IsFinal=false.
func TestReadAll_RunningSnapshot(t *testing.T) {
	root := t.TempDir()
	uuid := "bbbbbbbb-0000-0000-0000-000000000001"
	makeSessionDir(t, root, uuid, []map[string]any{
		startEvent("2026-06-13T09:00:00.000Z", "/home/user/live"),
		snapshotEvent("2026-06-13T09:30:00.000Z", 250_000_000_000, "claude-sonnet-4.6", 8000, 22000),
		// No shutdown — still running.
	}, true)

	sessions, err := readAll(root)
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	s := sessions[0]
	if s.IsFinal {
		t.Error("IsFinal = true, want false (no shutdown, snapshot only)")
	}
	if s.TotalNanoAIU != 250_000_000_000 {
		t.Errorf("TotalNanoAIU = %d, want 250000000000 (from snapshot)", s.TotalNanoAIU)
	}
	if s.Tokens.CurrentTokens != 22000 {
		t.Errorf("CurrentTokens = %d, want 22000 (from snapshot)", s.Tokens.CurrentTokens)
	}
	if s.Tokens.SystemTokens != 8000 {
		t.Errorf("SystemTokens = %d, want 8000 (from snapshot)", s.Tokens.SystemTokens)
	}
	if s.PrimaryModel != "claude-sonnet-4.6" {
		t.Errorf("PrimaryModel = %q, want %q", s.PrimaryModel, "claude-sonnet-4.6")
	}
	if got := s.TotalInputTokens(); got != 50000 {
		t.Errorf("TotalInputTokens = %d, want 50000 (from snapshot)", got)
	}
}

// TestReadAll_SnapshotThenShutdown covers BUG 1: a final shutdown reading must
// win over an earlier running snapshot, and IsFinal must be true.
func TestReadAll_SnapshotThenShutdown(t *testing.T) {
	root := t.TempDir()
	uuid := "bbbbbbbb-0000-0000-0000-000000000002"
	makeSessionDir(t, root, uuid, []map[string]any{
		startEvent("2026-06-13T09:00:00.000Z", "/home/user/settled"),
		snapshotEvent("2026-06-13T09:30:00.000Z", 250_000_000_000, "claude-sonnet-4.6", 8000, 22000),
		shutdownEvent("2026-06-13T10:00:00.000Z", 600_000_000_000, "claude-sonnet-4.6", 12000, 35000),
	}, false)

	sessions, err := readAll(root)
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	s := sessions[0]
	if !s.IsFinal {
		t.Error("IsFinal = false, want true (shutdown applied)")
	}
	if s.TotalNanoAIU != 600_000_000_000 {
		t.Errorf("TotalNanoAIU = %d, want 600000000000 (final shutdown wins)", s.TotalNanoAIU)
	}
	if s.Tokens.CurrentTokens != 35000 {
		t.Errorf("CurrentTokens = %d, want 35000 (final shutdown wins)", s.Tokens.CurrentTokens)
	}
	// ModelMetrics must not accumulate duplicates across snapshot + shutdown.
	if len(s.ModelMetrics) != 1 {
		t.Errorf("ModelMetrics len = %d, want 1 (no duplicate accumulation)", len(s.ModelMetrics))
	}
	if got := s.TotalInputTokens(); got != 100000 {
		t.Errorf("TotalInputTokens = %d, want 100000 (final shutdown wins)", got)
	}
}

// TestBillingTime covers BUG 2: EndTime is preferred, StartTime is the fallback.
func TestBillingTime(t *testing.T) {
	start := time.Date(2026, 5, 31, 23, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC)

	// Finalized session → BillingTime is EndTime.
	finalized := Session{StartTime: start, EndTime: end}
	if got := finalized.BillingTime(); !got.Equal(end) {
		t.Errorf("BillingTime (finalized) = %v, want EndTime %v", got, end)
	}
	// Active session (no EndTime) → BillingTime falls back to StartTime.
	active := Session{StartTime: start}
	if got := active.BillingTime(); !got.Equal(start) {
		t.Errorf("BillingTime (active) = %v, want StartTime %v", got, start)
	}
}

func TestReadAll_SortedNewestFirst(t *testing.T) {
	root := t.TempDir()

	makeSessionDir(t, root, "older-session-aaaa", []map[string]any{
		startEvent("2026-06-10T08:00:00.000Z", "/proj/old"),
		shutdownEvent("2026-06-10T09:00:00.000Z", 100_000_000_000, "claude-sonnet-4.6", 1000, 5000),
	}, false)

	makeSessionDir(t, root, "newer-session-bbbb", []map[string]any{
		startEvent("2026-06-13T08:00:00.000Z", "/proj/new"),
		shutdownEvent("2026-06-13T09:00:00.000Z", 200_000_000_000, "claude-sonnet-4.6", 2000, 6000),
	}, false)

	sessions, err := readAll(root)
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	// Newest first
	if !sessions[0].StartTime.After(sessions[1].StartTime) {
		t.Errorf("sessions not sorted newest-first: [0]=%v [1]=%v",
			sessions[0].StartTime, sessions[1].StartTime)
	}
}

func TestReadAll_BadDirSkipped(t *testing.T) {
	root := t.TempDir()

	// Session with no events.jsonl — should be skipped, not abort
	bad := filepath.Join(root, "bad-session-cccc")
	if err := os.MkdirAll(bad, 0755); err != nil {
		t.Fatal(err)
	}

	// Good session alongside it
	makeSessionDir(t, root, "good-session-dddd", []map[string]any{
		startEvent("2026-06-13T08:00:00.000Z", "/proj/good"),
		shutdownEvent("2026-06-13T09:00:00.000Z", 300_000_000_000, "claude-sonnet-4.6", 3000, 7000),
	}, false)

	sessions, err := readAll(root)
	if err != nil {
		t.Fatalf("readAll returned error: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 good session, got %d", len(sessions))
	}
	if sessions[0].ID != "good-session-dddd" {
		t.Errorf("unexpected session ID: %q", sessions[0].ID)
	}
}

func TestReadThisMonth(t *testing.T) {
	// ReadThisMonth calls ReadAll which uses platform.SessionStateDir().
	// We test the filter logic directly by calling readAll with synthetic data.
	root := t.TempDir()
	now := time.Now()

	// This month
	thisMonth := now.Format("2006-01") + "-01T00:00:00.000Z"
	makeSessionDir(t, root, "this-month-eeee", []map[string]any{
		startEvent(thisMonth, "/proj/current"),
		shutdownEvent(thisMonth, 100_000_000_000, "model", 1000, 5000),
	}, false)

	// Last month
	lastMonth := now.AddDate(0, -1, 0).Format("2006-01") + "-01T00:00:00.000Z"
	makeSessionDir(t, root, "last-month-ffff", []map[string]any{
		startEvent(lastMonth, "/proj/old"),
		shutdownEvent(lastMonth, 200_000_000_000, "model", 2000, 6000),
	}, false)

	all, err := readAll(root)
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}

	// Apply the same filter ReadThisMonth uses
	var thisMonthSessions []Session
	for _, s := range all {
		if s.StartTime.Year() == now.Year() && s.StartTime.Month() == now.Month() {
			thisMonthSessions = append(thisMonthSessions, s)
		}
	}

	if len(thisMonthSessions) != 1 {
		t.Errorf("expected 1 this-month session, got %d", len(thisMonthSessions))
	}
	if len(thisMonthSessions) > 0 && thisMonthSessions[0].ID != "this-month-eeee" {
		t.Errorf("unexpected session ID: %q", thisMonthSessions[0].ID)
	}
}

func TestReadSince(t *testing.T) {
	root := t.TempDir()
	cutoff := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)

	makeSessionDir(t, root, "before-gggg", []map[string]any{
		startEvent("2026-06-11T08:00:00.000Z", "/proj/old"),
		shutdownEvent("2026-06-11T09:00:00.000Z", 100_000_000_000, "model", 1000, 5000),
	}, false)

	makeSessionDir(t, root, "on-cutoff-hhhh", []map[string]any{
		startEvent("2026-06-12T00:00:00.000Z", "/proj/cutoff"),
		shutdownEvent("2026-06-12T01:00:00.000Z", 200_000_000_000, "model", 2000, 6000),
	}, false)

	makeSessionDir(t, root, "after-iiii", []map[string]any{
		startEvent("2026-06-13T08:00:00.000Z", "/proj/new"),
		shutdownEvent("2026-06-13T09:00:00.000Z", 300_000_000_000, "model", 3000, 7000),
	}, false)

	all, err := readAll(root)
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}

	var since []Session
	for _, s := range all {
		if !s.StartTime.Before(cutoff) {
			since = append(since, s)
		}
	}

	if len(since) != 2 {
		t.Errorf("expected 2 sessions at/after cutoff, got %d", len(since))
	}
}

func TestTokenHelpers(t *testing.T) {
	s := Session{
		ModelMetrics: []ModelMetric{
			{Model: "a", InputTokens: 1000, OutputTokens: 200},
			{Model: "b", InputTokens: 500, OutputTokens: 100},
		},
	}
	if got := s.TotalInputTokens(); got != 1500 {
		t.Errorf("TotalInputTokens = %d, want 1500", got)
	}
	if got := s.TotalOutputTokens(); got != 300 {
		t.Errorf("TotalOutputTokens = %d, want 300", got)
	}
}

func TestReadAll_SetsCLISource(t *testing.T) {
	root := t.TempDir()
	makeSessionDir(t, root, "src-session-aaaa", []map[string]any{
		startEvent("2026-06-13T08:00:00.000Z", "/proj/src"),
		shutdownEvent("2026-06-13T09:00:00.000Z", 100_000_000_000, "claude-sonnet-4.6", 1000, 5000),
	}, false)

	sessions, err := readAll(root)
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Source != "copilot-cli" {
		t.Errorf("Source = %q, want %q", sessions[0].Source, "copilot-cli")
	}
}

func TestIDECollector_IsHermeticNoOp(t *testing.T) {
	// The ideCollector is a no-op stub: the real VS Code Copilot Chat reader is a
	// SEPARATE source (chatSessions/transcripts under VS Code user data, NOT
	// ~/.copilot) and is deferred to Phase 6 (see ADR-007 corrected). Until then
	// it must be hermetic — return no error and no sessions, and never read the
	// real ~/.copilot or any marker file.
	c := ideCollector{}
	if c.Name() != "copilot-ide" {
		t.Errorf("Name = %q, want %q", c.Name(), "copilot-ide")
	}
	got, err := c.Collect()
	if err != nil {
		t.Fatalf("ideCollector.Collect: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Collect() returned %d sessions, want 0 (no-op stub)", len(got))
	}
}

func TestDedupByID(t *testing.T) {
	cases := []struct {
		name     string
		in       []Session
		wantLen  int
		wantNano map[string]int64 // ID -> expected surviving TotalNanoAIU
	}{
		{
			name: "final beats live snapshot regardless of magnitude",
			in: []Session{
				{ID: "x", Source: "copilot-ide", IsFinal: false, TotalNanoAIU: 900},
				{ID: "x", Source: "copilot-cli", IsFinal: true, TotalNanoAIU: 100},
			},
			wantLen:  1,
			wantNano: map[string]int64{"x": 100},
		},
		{
			name: "neither final -> higher nano wins",
			in: []Session{
				{ID: "y", Source: "copilot-cli", IsFinal: false, TotalNanoAIU: 200},
				{ID: "y", Source: "copilot-ide", IsFinal: false, TotalNanoAIU: 500},
			},
			wantLen:  1,
			wantNano: map[string]int64{"y": 500},
		},
		{
			name: "both final -> higher nano wins",
			in: []Session{
				{ID: "z", IsFinal: true, TotalNanoAIU: 300},
				{ID: "z", IsFinal: true, TotalNanoAIU: 800},
			},
			wantLen:  1,
			wantNano: map[string]int64{"z": 800},
		},
		{
			name: "distinct ids all kept",
			in: []Session{
				{ID: "a", TotalNanoAIU: 1},
				{ID: "b", TotalNanoAIU: 2},
			},
			wantLen:  2,
			wantNano: map[string]int64{"a": 1, "b": 2},
		},
		{
			name: "empty id passes through untouched",
			in: []Session{
				{ID: "", TotalNanoAIU: 7},
				{ID: "", TotalNanoAIU: 9},
			},
			wantLen: 2,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := dedupByID(c.in)
			if len(out) != c.wantLen {
				t.Fatalf("len = %d, want %d", len(out), c.wantLen)
			}
			for id, want := range c.wantNano {
				var found bool
				for _, s := range out {
					if s.ID == id {
						found = true
						if s.TotalNanoAIU != want {
							t.Errorf("id %q TotalNanoAIU = %d, want %d", id, s.TotalNanoAIU, want)
						}
					}
				}
				if !found {
					t.Errorf("id %q not found in output", id)
				}
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	cases := []struct {
		input string
		wantZ bool // want zero time
	}{
		{"2026-06-13T08:43:04.057Z", false},
		{"2026-06-13T08:43:04Z", false},
		{"", true},
		{"not-a-date", true},
	}
	for _, c := range cases {
		got := parseTime(c.input)
		if c.wantZ != got.IsZero() {
			t.Errorf("parseTime(%q): IsZero=%v, want %v", c.input, got.IsZero(), c.wantZ)
		}
	}
}
