package codeentities

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"reflect"
	"strings"
	"testing"

	"github.com/aaraminds/dif/libs/extraction"
	"github.com/aaraminds/dif/libs/sourceanchors"
)

func TestDetectFindsUnresolvedAnchoredCodeEntityCandidates(t *testing.T) {
	t.Parallel()

	result := fixtureResult("Use `PaymentService`, com.example.orders.OrderController, com.example.orders.OrderRepository.findById, OrderRepository.findById, src/main/java/com/example/orders/OrderRepository.java, GET /orders/{id}, and feature_flag_enabled.")
	candidates, err := Detect(result)
	if err != nil {
		t.Fatalf("detect candidates: %v", err)
	}

	assertCandidate(t, candidates, "PaymentService", KindClass, ModeSimpleName, ConfidenceInferred, StatusUnresolved)
	assertCandidate(t, candidates, "com.example.orders.OrderController", KindClass, ModeQualifiedName, ConfidenceExact, StatusUnresolved)
	assertCandidate(t, candidates, "com.example.orders.OrderRepository.findById", KindMethod, ModeQualifiedName, ConfidenceExact, StatusUnresolved)
	assertCandidate(t, candidates, "OrderRepository.findById", KindMethod, ModeQualifiedName, ConfidenceExact, StatusUnresolved)
	assertCandidate(t, candidates, "src/main/java/com/example/orders/OrderRepository.java", KindFilePath, ModeSourcePath, ConfidenceExact, StatusUnresolved)
	assertCandidate(t, candidates, "/orders/{id}", KindService, ModeSimpleName, ConfidenceInferred, StatusUnresolved)
	assertCandidate(t, candidates, "feature_flag_enabled", KindUnknown, ModeFuzzy, ConfidenceInferred, StatusUnresolved)

	for _, candidate := range candidates {
		if candidate.AnchorID == "" || candidate.SourceRef == "" {
			t.Fatalf("candidate must preserve source anchor: %+v", candidate)
		}
		if candidate.ResolvedRIFNodeID != "" {
			t.Fatalf("detector must not resolve RIF nodes: %+v", candidate)
		}
	}
}

func TestDetectIsDeterministicAndDeduplicates(t *testing.T) {
	t.Parallel()

	result := fixtureResult("PaymentService calls PaymentService and `PaymentService`.")
	first, err := Detect(result)
	if err != nil {
		t.Fatalf("detect first: %v", err)
	}
	second, err := Detect(result)
	if err != nil {
		t.Fatalf("detect second: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("candidate count changed: %d vs %d", len(first), len(second))
	}
	seenPaymentService := 0
	for i := range first {
		if !reflect.DeepEqual(first[i], second[i]) {
			t.Fatalf("candidate %d changed:\nfirst:  %+v\nsecond: %+v", i, first[i], second[i])
		}
		if first[i].CandidateText == "PaymentService" {
			seenPaymentService++
		}
	}
	if seenPaymentService != 1 {
		t.Fatalf("expected one deduplicated PaymentService candidate, got %d in %+v", seenPaymentService, first)
	}
}

func TestDetectRejectsCandidateWithoutSourceAnchor(t *testing.T) {
	t.Parallel()

	result := fixtureResult("PaymentService")
	result.Nodes[0].AnchorID = ""
	_, err := Detect(result)
	if err == nil || !strings.Contains(err.Error(), "no source anchor") {
		t.Fatalf("expected source-anchor error, got %v", err)
	}
}

func TestCandidateValidatePreventsSuccessShapedUnresolvedRows(t *testing.T) {
	t.Parallel()

	candidate := fixtureCandidate()
	candidate.ResolvedRIFNodeID = "rif-node-1"
	if err := candidate.Validate(); err == nil || !strings.Contains(err.Error(), "unresolved candidate") {
		t.Fatalf("expected unresolved shape error, got %v", err)
	}

	candidate = fixtureCandidate()
	candidate.MatchStatus = StatusResolved
	if err := candidate.Validate(); err == nil || !strings.Contains(err.Error(), "resolved candidate") {
		t.Fatalf("expected resolved shape error, got %v", err)
	}
}

func TestSQLStoreWritesUnresolvedCandidateShape(t *testing.T) {
	t.Parallel()

	execer := &recordingExecer{}
	store := SQLStore{Execer: execer}
	candidate := fixtureCandidate()
	if err := store.WriteCandidates(context.Background(), []Candidate{candidate}); err != nil {
		t.Fatalf("write candidates: %v", err)
	}
	if len(execer.calls) != 1 {
		t.Fatalf("expected one SQL call, got %d", len(execer.calls))
	}
	call := execer.calls[0]
	if !strings.Contains(call.query, "INSERT INTO dif_meta.code_entity_candidates") {
		t.Fatalf("unexpected query: %s", call.query)
	}
	if !strings.Contains(call.query, "ON CONFLICT (candidate_id)") {
		t.Fatalf("expected idempotent upsert query: %s", call.query)
	}
	if !strings.Contains(call.query, "WHERE dif_meta.code_entity_candidates.match_status = 'unresolved'") {
		t.Fatalf("upsert must not overwrite resolver-owned rows: %s", call.query)
	}
	if !strings.Contains(call.query, "resolved_rif_node_id IS NULL") {
		t.Fatalf("upsert must preserve resolved RIF evidence: %s", call.query)
	}
	if got := call.args[8]; got != string(StatusUnresolved) {
		t.Fatalf("match_status arg = %v, want unresolved", got)
	}
	if got := call.args[9]; got != nil {
		t.Fatalf("resolved_rif_node_id arg = %v, want nil", got)
	}
	if got := call.args[12]; got != `["identifier_heuristic"]` {
		t.Fatalf("caveats arg = %v", got)
	}
}

func assertCandidate(t *testing.T, candidates []Candidate, text string, kind CandidateKind, mode MatchMode, confidence Confidence, status MatchStatus) {
	t.Helper()
	for _, candidate := range candidates {
		if candidate.CandidateText == text {
			if candidate.CandidateKind != kind || candidate.MatchMode != mode || candidate.Confidence != confidence || candidate.MatchStatus != status {
				t.Fatalf("candidate %q shape mismatch: %+v", text, candidate)
			}
			return
		}
	}
	t.Fatalf("missing candidate %q in %+v", text, candidates)
}

func fixtureResult(text string) extraction.Result {
	anchor := sourceanchors.Anchor{
		AnchorID:          "anchor-1",
		CorpusID:          "corpus-1",
		DocumentID:        "doc-1",
		DocumentVersionID: "docver-1",
		AnchorType:        sourceanchors.TypeMarkdown,
		SourceRef:         "corpus-1@docver-1:md:architecture.md#L1-L1",
		Path:              "architecture.md",
		LineStart:         1,
		LineEnd:           1,
		ContentHash:       sourceanchors.ContentHash(text),
	}
	return extraction.Result{
		Document: extraction.Document{
			DocumentID:        "doc-1",
			CorpusID:          "corpus-1",
			SourceID:          "source-1",
			Path:              "architecture.md",
			Format:            extraction.FormatMarkdown,
			DocumentVersionID: "docver-1",
			ContentHash:       sourceanchors.ContentHash(text),
		},
		Nodes: []extraction.Node{{
			NodeID:            "node-1",
			CorpusID:          "corpus-1",
			DocumentID:        "doc-1",
			DocumentVersionID: "docver-1",
			Kind:              extraction.NodeBlock,
			Ordinal:           1,
			AnchorID:          "anchor-1",
			TextHash:          sourceanchors.ContentHash(text),
			Text:              text,
		}},
		Anchors: []sourceanchors.Anchor{anchor},
	}
}

func fixtureCandidate() Candidate {
	return Candidate{
		CandidateID:       "candidate-1",
		CorpusID:          "corpus-1",
		DocumentID:        "doc-1",
		DocumentVersionID: "docver-1",
		NodeID:            "node-1",
		AnchorID:          "anchor-1",
		SourceRef:         "corpus-1@docver-1:md:architecture.md#L1-L1",
		CandidateText:     "feature_flag_enabled",
		CandidateKind:     KindUnknown,
		MatchStatus:       StatusUnresolved,
		MatchMode:         ModeFuzzy,
		Confidence:        ConfidenceInferred,
		Caveats:           []string{CaveatIdentifierHeuristic},
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
	return driver.RowsAffected(1), nil
}
