package session

import (
	"testing"
)

// These tests exercise readSession's parsing of cache-read, cache-write, and
// reasoning token metrics from session.shutdown / running-snapshot events. They
// are source-agnostic: readSession is the shared parser for every Collector, so
// the coverage applies to the CLI source today and any future source.
//
// NOTE: the IDE collector (ideCollector) is a no-op stub — it does NOT read
// vscode.metadata.json or any marker file (see ADR-007 corrected and
// TestIDECollectorIsNoOp). These tests therefore do not create marker files or
// assert any marker-driven behavior; doing so would contradict the stub.

// assistantMessageEvent returns a synthetic assistant.message event with cache and reasoning tokens.
func assistantMessageEvent(ts, apiCallID string, inputTokens, outputTokens int64,
	cacheReadTokens, cacheWriteTokens, reasoningTokens int64, eventID, parentID string) map[string]any {
	return map[string]any{
		"type":      "assistant.message",
		"timestamp": ts,
		"id":        eventID,
		"parentId":  parentID,
		"data": map[string]any{
			"model":            "claude-sonnet-4.6",
			"inputTokens":      inputTokens,
			"outputTokens":     outputTokens,
			"cacheReadTokens":  cacheReadTokens,
			"cacheWriteTokens": cacheWriteTokens,
			"reasoningTokens":  reasoningTokens,
			"apiCallId":        apiCallID,
			"timestamp":        ts,
		},
	}
}

// shutdownEventWithCache returns a session.shutdown event with cache and reasoning token metrics.
func shutdownEventWithCache(endTime string, nanoAIU int64, model string, systemTokens, currentTokens int64,
	cacheReadTokens, cacheWriteTokens, reasoningTokens int64) map[string]any {
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
						"inputTokens":      int64(100000),
						"outputTokens":     int64(5000),
						"cacheReadTokens":  cacheReadTokens,
						"cacheWriteTokens": cacheWriteTokens,
						"reasoningTokens":  reasoningTokens,
					},
				},
			},
		},
	}
}

// TestReadSession_ParsesCacheReasoningTokens verifies readSession extracts cache
// and reasoning tokens onto ModelMetric from a shutdown event.
func TestReadSession_ParsesCacheReasoningTokens(t *testing.T) {
	root := t.TempDir()
	uuid := "session-001-parse-tokens"

	start := "2026-06-13T08:00:00.000Z"
	end := "2026-06-13T10:00:00.000Z"

	events := []map[string]any{
		startEvent(start, "/home/user/project"),
		shutdownEventWithCache(end, 500_000_000_000, "claude-sonnet-4.6", 12000, 35000,
			28_009_533, 697_780, 14_618),
	}

	dir := makeSessionDir(t, root, uuid, events, false)

	s, err := readSession(uuid, dir)
	if err != nil {
		t.Fatalf("readSession: %v", err)
	}

	if len(s.ModelMetrics) != 1 {
		t.Fatalf("expected 1 model metric, got %d", len(s.ModelMetrics))
	}

	mm := s.ModelMetrics[0]
	if mm.CacheReadTokens != 28_009_533 {
		t.Errorf("CacheReadTokens = %d, want 28009533", mm.CacheReadTokens)
	}
	if mm.CacheWriteTokens != 697_780 {
		t.Errorf("CacheWriteTokens = %d, want 697780", mm.CacheWriteTokens)
	}
	if mm.ReasoningTokens != 14_618 {
		t.Errorf("ReasoningTokens = %d, want 14618", mm.ReasoningTokens)
	}

	// Session-level aggregates must surface the same values (single model).
	if s.TotalCacheReadTokens() != 28_009_533 {
		t.Errorf("TotalCacheReadTokens = %d, want 28009533", s.TotalCacheReadTokens())
	}
	if s.TotalCacheWriteTokens() != 697_780 {
		t.Errorf("TotalCacheWriteTokens = %d, want 697780", s.TotalCacheWriteTokens())
	}
	if s.TotalReasoningTokens() != 14_618 {
		t.Errorf("TotalReasoningTokens = %d, want 14618", s.TotalReasoningTokens())
	}
}

// TestIDECollectorIsNoOp verifies the ideCollector is a hermetic no-op stub.
//
// The real VS Code Copilot Chat reader is a SEPARATE source (chatSessions/transcripts
// under VS Code user data, NOT ~/.copilot) and will be implemented in Phase 6 against
// the real Chat schema — see ADR-007 (corrected). Until then the collector must return
// no error and no sessions, and must NOT touch the real ~/.copilot.
func TestIDECollectorIsNoOp(t *testing.T) {
	collector := ideCollector{}

	if got := collector.Name(); got != "copilot-ide" {
		t.Errorf("Name() = %q, want %q", got, "copilot-ide")
	}

	sessions, err := collector.Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v, want nil (no-op stub)", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Collect() returned %d sessions, want 0 (no-op stub)", len(sessions))
	}
}

// TestReadSession_ConcurrentReads verifies no data races with concurrent readSession calls.
func TestReadSession_ConcurrentReads(t *testing.T) {
	root := t.TempDir()
	uuid := "race-test-001"

	start := "2026-06-13T08:00:00.000Z"
	end := "2026-06-13T10:00:00.000Z"

	events := []map[string]any{
		startEvent(start, "/home/user/race"),
		shutdownEventWithCache(end, 500_000_000_000, "claude-sonnet-4.6", 12000, 35000, 500, 50, 10),
	}

	dir := makeSessionDir(t, root, uuid, events, false)

	done := make(chan bool, 2)
	for i := 0; i < 2; i++ {
		go func() {
			s, _ := readSession(uuid, dir)
			_ = s.ID
			_ = s.TotalNanoAIU
			done <- true
		}()
	}

	<-done
	<-done
	// If we got here without a data race (run with -race), the test passes.
}

// TestReadSession_ActiveWithoutShutdown verifies active sessions without a shutdown event.
func TestReadSession_ActiveWithoutShutdown(t *testing.T) {
	root := t.TempDir()
	uuid := "active-session"

	start := "2026-06-13T08:00:00.000Z"

	events := []map[string]any{
		startEvent(start, "/home/user/active"),
		// No shutdown event; session is still active. assistant.message is not a
		// billing-bearing snapshot, so it leaves billing at zero.
		assistantMessageEvent("2026-06-13T08:30:00.000Z", "msg_active_001", 1000, 100, 400, 50, 8,
			"evt-active-001", "sess-active"),
	}

	dir := makeSessionDir(t, root, uuid, events, true) // lock file present

	s, err := readSession(uuid, dir)
	if err != nil {
		t.Fatalf("readSession: %v", err)
	}

	if s.IsFinal {
		t.Error("IsFinal = true, want false (no shutdown event, active session)")
	}
	if !s.IsActive {
		t.Error("IsActive = false, want true (lock file present)")
	}
	if s.TotalNanoAIU != 0 {
		t.Errorf("TotalNanoAIU = %d, want 0 (no shutdown event)", s.TotalNanoAIU)
	}
}

// TestReadSession_MultipleModels verifies sessions using multiple models aggregate correctly.
func TestReadSession_MultipleModels(t *testing.T) {
	root := t.TempDir()
	uuid := "multimodel"

	start := "2026-06-13T08:00:00.000Z"
	end := "2026-06-13T10:00:00.000Z"

	shutdownWithMultiModels := map[string]any{
		"type":      "session.shutdown",
		"timestamp": end,
		"id":        "evt-shutdown-multi",
		"parentId":  "sess-multi",
		"data": map[string]any{
			"totalNanoAiu":          700_000_000_000,
			"currentModel":          "claude-opus-4.6",
			"systemTokens":          int64(15000),
			"currentTokens":         int64(40000),
			"conversationTokens":    int64(6000),
			"toolDefinitionsTokens": int64(4000),
			"modelMetrics": map[string]any{
				"claude-sonnet-4.6": map[string]any{
					"totalNanoAiu": int64(300_000_000_000),
					"usage": map[string]any{
						"inputTokens":      int64(50000),
						"outputTokens":     int64(2500),
						"cacheReadTokens":  int64(10000),
						"cacheWriteTokens": int64(500),
						"reasoningTokens":  int64(5),
					},
				},
				"claude-opus-4.6": map[string]any{
					"totalNanoAiu": int64(400_000_000_000),
					"usage": map[string]any{
						"inputTokens":      int64(80000),
						"outputTokens":     int64(4000),
						"cacheReadTokens":  int64(20000),
						"cacheWriteTokens": int64(1000),
						"reasoningTokens":  int64(10),
					},
				},
			},
		},
	}

	events := []map[string]any{
		startEvent(start, "/home/user/multimodel"),
		shutdownWithMultiModels,
	}

	dir := makeSessionDir(t, root, uuid, events, false)

	s, err := readSession(uuid, dir)
	if err != nil {
		t.Fatalf("readSession: %v", err)
	}

	if len(s.ModelMetrics) != 2 {
		t.Fatalf("expected 2 model metrics, got %d", len(s.ModelMetrics))
	}

	// PrimaryModel should be the one with highest NanoAIU.
	if s.PrimaryModel != "claude-opus-4.6" {
		t.Errorf("PrimaryModel = %q, want %q", s.PrimaryModel, "claude-opus-4.6")
	}

	var sonnet, opus *ModelMetric
	for i := range s.ModelMetrics {
		if s.ModelMetrics[i].Model == "claude-sonnet-4.6" {
			sonnet = &s.ModelMetrics[i]
		}
		if s.ModelMetrics[i].Model == "claude-opus-4.6" {
			opus = &s.ModelMetrics[i]
		}
	}

	if sonnet == nil || sonnet.CacheReadTokens != 10000 {
		t.Errorf("sonnet CacheReadTokens = %v, want 10000", sonnet)
	}
	if opus == nil || opus.CacheReadTokens != 20000 {
		t.Errorf("opus CacheReadTokens = %v, want 20000", opus)
	}

	// Session aggregate sums both models.
	if s.TotalCacheReadTokens() != 30000 {
		t.Errorf("TotalCacheReadTokens = %d, want 30000", s.TotalCacheReadTokens())
	}
}

// TestReadSession_ZeroCacheTokens verifies handling of sessions with zero cache/reasoning tokens.
func TestReadSession_ZeroCacheTokens(t *testing.T) {
	root := t.TempDir()
	uuid := "zero-cache"

	start := "2026-06-13T08:00:00.000Z"
	end := "2026-06-13T10:00:00.000Z"

	events := []map[string]any{
		startEvent(start, "/home/user/zerocache"),
		shutdownEventWithCache(end, 500_000_000_000, "claude-sonnet-4.6", 12000, 35000,
			0, 0, 0), // zero cache and reasoning tokens
	}

	dir := makeSessionDir(t, root, uuid, events, false)

	s, err := readSession(uuid, dir)
	if err != nil {
		t.Fatalf("readSession: %v", err)
	}

	if len(s.ModelMetrics) != 1 {
		t.Fatalf("expected 1 model metric, got %d", len(s.ModelMetrics))
	}

	mm := s.ModelMetrics[0]
	if mm.CacheReadTokens != 0 {
		t.Errorf("CacheReadTokens = %d, want 0", mm.CacheReadTokens)
	}
	if mm.CacheWriteTokens != 0 {
		t.Errorf("CacheWriteTokens = %d, want 0", mm.CacheWriteTokens)
	}
	if mm.ReasoningTokens != 0 {
		t.Errorf("ReasoningTokens = %d, want 0", mm.ReasoningTokens)
	}
}
