package codeentities

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/aaraminds/dif/libs/rifcompat"
)

func compatibleReport() rifcompat.Report {
	return rifcompat.Report{
		Status:       rifcompat.StatusCompatible,
		ShadowStatus: rifcompat.ShadowEmpty,
		Matches: []rifcompat.Entity{
			{
				NodeID:        rifcompat.NodeID("repo-1", "com.acme.billing.InvoiceService", "CLASS"),
				RepoID:        "repo-1",
				Kind:          "CLASS",
				QualifiedName: "com.acme.billing.InvoiceService",
				SimpleName:    "InvoiceService",
				SourceRef:     "src/main/java/com/acme/billing/InvoiceService.java#L10-L120",
				Origin:        "age",
				Confidence:    rifcompat.ConfidenceExact,
			},
			{
				NodeID:        rifcompat.NodeID("repo-1", "com.acme.billing.InvoiceService.render", "METHOD"),
				RepoID:        "repo-1",
				Kind:          "METHOD",
				QualifiedName: "com.acme.billing.InvoiceService.render",
				SimpleName:    "render",
				SourceRef:     "src/main/java/com/acme/billing/InvoiceService.java#L42-L60",
				Origin:        "age",
				Confidence:    rifcompat.ConfidenceExact,
			},
			{
				NodeID:        rifcompat.NodeID("repo-1", "src/main/java/com/acme/billing/InvoiceService.java", "FILE"),
				RepoID:        "repo-1",
				Kind:          "FILE",
				QualifiedName: "src/main/java/com/acme/billing/InvoiceService.java",
				SimpleName:    "InvoiceService.java",
				SourceRef:     "src/main/java/com/acme/billing/InvoiceService.java",
				Origin:        "age",
				Confidence:    rifcompat.ConfidenceExact,
			},
			{
				NodeID:        rifcompat.NodeID("repo-1", "com.acme.payments.RetryPolicy", "CLASS"),
				RepoID:        "repo-1",
				Kind:          "CLASS",
				QualifiedName: "com.acme.payments.RetryPolicy",
				SimpleName:    "RetryPolicy",
				SourceRef:     "src/main/java/com/acme/payments/RetryPolicy.java#L5-L80",
				Origin:        "age",
				Confidence:    rifcompat.ConfidenceExact,
			},
			{
				NodeID:        rifcompat.NodeID("repo-2", "com.other.RetryPolicy", "CLASS"),
				RepoID:        "repo-2",
				Kind:          "CLASS",
				QualifiedName: "com.other.RetryPolicy",
				SimpleName:    "RetryPolicy",
				SourceRef:     "src/com/other/RetryPolicy.java#L1-L50",
				Origin:        "age",
				Confidence:    rifcompat.ConfidenceExact,
			},
		},
	}
}

func resolverCandidate(id, text string, kind CandidateKind, mode MatchMode, confidence Confidence, caveats ...string) Candidate {
	return Candidate{
		CandidateID:       id,
		CorpusID:          "corpus-1",
		DocumentID:        "doc-1",
		DocumentVersionID: "docv-1",
		NodeID:            "node-" + id,
		AnchorID:          "anchor-" + id,
		SourceRef:         "docs/design.md#L10-L20",
		CandidateText:     text,
		CandidateKind:     kind,
		MatchStatus:       StatusUnresolved,
		MatchMode:         mode,
		Confidence:        confidence,
		Caveats:           caveats,
	}
}

func TestResolveQualifiedNameMethodIsExact(t *testing.T) {
	t.Parallel()

	candidate := resolverCandidate("c1", "com.acme.billing.InvoiceService.render()", KindMethod, ModeQualifiedName, ConfidenceExact)
	outcome, err := Resolve(compatibleReport(), []Candidate{candidate})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got := outcome.Candidates[0].Candidate
	if got.MatchStatus != StatusResolved {
		t.Fatalf("match_status = %q, want resolved (caveats %v)", got.MatchStatus, got.Caveats)
	}
	want := rifcompat.NodeID("repo-1", "com.acme.billing.InvoiceService.render", "METHOD")
	if got.ResolvedRIFNodeID != want {
		t.Fatalf("resolved_rif_node_id = %q, want %q", got.ResolvedRIFNodeID, want)
	}
	if got.Confidence != ConfidenceExact {
		t.Fatalf("confidence = %q, want exact", got.Confidence)
	}
	if len(outcome.Edges) != 1 {
		t.Fatalf("expected one DESCRIBES edge, got %d", len(outcome.Edges))
	}
	edge := outcome.Edges[0]
	if edge.EdgeID != rifcompat.EdgeID(candidate.NodeID, EdgeKindDescribes, want) {
		t.Fatalf("edge_id does not use shared RIF/DIF algorithm")
	}
	if edge.RepoID != "repo-1" || edge.CodeSourceRef == "" || edge.AnchorID != candidate.AnchorID {
		t.Fatalf("edge missing ADR-016 minimum fields: %+v", edge)
	}
}

func TestResolveSourcePathResolvesFile(t *testing.T) {
	t.Parallel()

	candidate := resolverCandidate("c2", "src/main/java/com/acme/billing/InvoiceService.java", KindFilePath, ModeSourcePath, ConfidenceExact)
	outcome, err := Resolve(compatibleReport(), []Candidate{candidate})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got := outcome.Candidates[0].Candidate
	if got.MatchStatus != StatusResolved || got.Confidence != ConfidenceExact {
		t.Fatalf("unexpected resolution: %+v", got)
	}
	if len(outcome.Edges) != 1 {
		t.Fatalf("expected one edge, got %d", len(outcome.Edges))
	}
}

func TestResolveSimpleNameSingleMatchIsInferred(t *testing.T) {
	t.Parallel()

	candidate := resolverCandidate("c3", "InvoiceService", KindClass, ModeSimpleName, ConfidenceInferred)
	outcome, err := Resolve(compatibleReport(), []Candidate{candidate})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got := outcome.Candidates[0].Candidate
	if got.MatchStatus != StatusResolved {
		t.Fatalf("match_status = %q, want resolved", got.MatchStatus)
	}
	if got.Confidence != ConfidenceInferred {
		t.Fatalf("simple-name resolution must stay inferred, got %q", got.Confidence)
	}
}

func TestResolveAmbiguousSimpleNameCreatesNoEdge(t *testing.T) {
	t.Parallel()

	candidate := resolverCandidate("c4", "RetryPolicy", KindClass, ModeSimpleName, ConfidenceInferred)
	outcome, err := Resolve(compatibleReport(), []Candidate{candidate})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got := outcome.Candidates[0].Candidate
	if got.MatchStatus != StatusAmbiguous {
		t.Fatalf("match_status = %q, want ambiguous", got.MatchStatus)
	}
	if got.ResolvedRIFNodeID != "" {
		t.Fatalf("ambiguous candidate must not carry resolved_rif_node_id")
	}
	if !containsString(got.Caveats, CaveatAmbiguousMatch) {
		t.Fatalf("ambiguous caveat missing: %v", got.Caveats)
	}
	if len(outcome.Edges) != 0 {
		t.Fatalf("ambiguous match must not create DESCRIBES edges")
	}
	if len(outcome.Candidates[0].Matches) != 2 {
		t.Fatalf("ambiguous evidence should list both matches, got %d", len(outcome.Candidates[0].Matches))
	}
}

func TestResolveFuzzySnakeCaseResolvesInferred(t *testing.T) {
	t.Parallel()

	candidate := resolverCandidate("c5", "invoice_service", KindUnknown, ModeFuzzy, ConfidenceInferred, CaveatIdentifierHeuristic)
	outcome, err := Resolve(compatibleReport(), []Candidate{candidate})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got := outcome.Candidates[0].Candidate
	if got.MatchStatus != StatusResolved {
		t.Fatalf("match_status = %q, want resolved via PascalCase fallback", got.MatchStatus)
	}
	if got.Confidence != ConfidenceInferred {
		t.Fatalf("fuzzy resolution must stay inferred")
	}
	if !containsString(got.Caveats, CaveatFuzzyMatch) {
		t.Fatalf("fuzzy caveat missing: %v", got.Caveats)
	}
}

func TestResolveUnknownStaysUnresolved(t *testing.T) {
	t.Parallel()

	candidate := resolverCandidate("c6", "com.acme.unknown.Thing", KindClass, ModeQualifiedName, ConfidenceExact)
	outcome, err := Resolve(compatibleReport(), []Candidate{candidate})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	got := outcome.Candidates[0].Candidate
	if got.MatchStatus != StatusUnresolved {
		t.Fatalf("match_status = %q, want unresolved", got.MatchStatus)
	}
	if len(outcome.Edges) != 0 {
		t.Fatalf("unresolved candidates must not create edges")
	}
}

func TestResolveNonCompatibleReportIsExplicit(t *testing.T) {
	t.Parallel()

	for _, status := range []rifcompat.Status{rifcompat.StatusNotDeployed, rifcompat.StatusIncompatible} {
		report := rifcompat.Report{Status: status}
		candidate := resolverCandidate("c7", "InvoiceService", KindClass, ModeSimpleName, ConfidenceInferred)
		outcome, err := Resolve(report, []Candidate{candidate})
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		got := outcome.Candidates[0].Candidate
		if got.MatchStatus != StatusRIFUnavailable {
			t.Fatalf("status %s: match_status = %q, want rif_unavailable", status, got.MatchStatus)
		}
		if !containsString(got.Caveats, CaveatRIFStatusPrefix+string(status)) {
			t.Fatalf("status %s: explicit RIF status caveat missing: %v", status, got.Caveats)
		}
		if len(outcome.Edges) != 0 {
			t.Fatalf("status %s: no edges may exist without a compatible report", status)
		}
		if outcome.Metrics[0].RIFUnavailable != 1 {
			t.Fatalf("status %s: rif_unavailable must be measured", status)
		}
	}
}

func TestResolveRejectsAlreadyResolvedInput(t *testing.T) {
	t.Parallel()

	candidate := resolverCandidate("c8", "InvoiceService", KindClass, ModeSimpleName, ConfidenceInferred)
	candidate.MatchStatus = StatusResolved
	candidate.ResolvedRIFNodeID = "somewhere"
	if _, err := Resolve(compatibleReport(), []Candidate{candidate}); err == nil {
		t.Fatalf("resolver must reject non-unresolved input")
	}
}

func TestResolveMetricsPerCorpus(t *testing.T) {
	t.Parallel()

	first := resolverCandidate("c9", "com.acme.billing.InvoiceService.render()", KindMethod, ModeQualifiedName, ConfidenceExact)
	second := resolverCandidate("c10", "RetryPolicy", KindClass, ModeSimpleName, ConfidenceInferred)
	third := resolverCandidate("c11", "com.acme.unknown.Thing", KindClass, ModeQualifiedName, ConfidenceExact)
	other := resolverCandidate("c12", "InvoiceService", KindClass, ModeSimpleName, ConfidenceInferred)
	other.CorpusID = "corpus-2"

	outcome, err := Resolve(compatibleReport(), []Candidate{first, second, third, other})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	want := []ResolutionMetrics{
		{CorpusID: "corpus-1", Total: 3, Resolved: 1, Ambiguous: 1, Unresolved: 1},
		{CorpusID: "corpus-2", Total: 1, Resolved: 1},
	}
	if !reflect.DeepEqual(outcome.Metrics, want) {
		t.Fatalf("metrics = %+v, want %+v", outcome.Metrics, want)
	}
	if rate := outcome.Metrics[0].ResolutionRate(); rate < 0.333 || rate > 0.334 {
		t.Fatalf("corpus-1 resolution rate = %f", rate)
	}
	if rate := outcome.Metrics[1].ResolutionRate(); rate != 1.0 {
		t.Fatalf("corpus-2 resolution rate = %f", rate)
	}
}

func TestResolveIsDeterministic(t *testing.T) {
	t.Parallel()

	candidates := []Candidate{
		resolverCandidate("d1", "com.acme.billing.InvoiceService.render()", KindMethod, ModeQualifiedName, ConfidenceExact),
		resolverCandidate("d2", "RetryPolicy", KindClass, ModeSimpleName, ConfidenceInferred),
		resolverCandidate("d3", "invoice_service", KindUnknown, ModeFuzzy, ConfidenceInferred, CaveatIdentifierHeuristic),
	}
	first, err := Resolve(compatibleReport(), candidates)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	second, err := Resolve(compatibleReport(), candidates)
	if err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("resolver output is not deterministic")
	}
}

func TestNewDescribesEdgeRequiresEvidence(t *testing.T) {
	t.Parallel()

	report := compatibleReport()
	unresolved := resolverCandidate("e1", "InvoiceService", KindClass, ModeSimpleName, ConfidenceInferred)
	if _, err := NewDescribesEdge(unresolved, report.Matches[0]); err == nil {
		t.Fatalf("edge must require a resolved candidate")
	}

	mismatched := unresolved
	mismatched.MatchStatus = StatusResolved
	mismatched.ResolvedRIFNodeID = "not-the-entity"
	if _, err := NewDescribesEdge(mismatched, report.Matches[0]); err == nil {
		t.Fatalf("edge must require candidate/entity node ID agreement")
	}
}

func TestSQLEdgeStoreWritesDescribesShape(t *testing.T) {
	t.Parallel()

	outcome, err := Resolve(compatibleReport(), []Candidate{
		resolverCandidate("s1", "com.acme.billing.InvoiceService.render()", KindMethod, ModeQualifiedName, ConfidenceExact),
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	execer := &recordingExecer{}
	store := SQLEdgeStore{Execer: execer}
	if err := store.WriteDescribesEdges(context.Background(), outcome.Edges); err != nil {
		t.Fatalf("write edges: %v", err)
	}
	if len(execer.calls) != 1 {
		t.Fatalf("expected one SQL call, got %d", len(execer.calls))
	}
	call := execer.calls[0]
	if !strings.Contains(call.query, "INSERT INTO dif_meta.edges") {
		t.Fatalf("unexpected query: %s", call.query)
	}
	if !strings.Contains(call.query, "'DESCRIBES'") || !strings.Contains(call.query, "'rif'") {
		t.Fatalf("DESCRIBES/rif literals missing: %s", call.query)
	}
	if strings.Contains(strings.ToLower(call.query), "rif_meta.") || strings.Contains(strings.ToLower(call.query), " rif.") {
		t.Fatalf("edge writer must not touch RIF-owned schemas: %s", call.query)
	}
	if !strings.Contains(call.query, "ON CONFLICT (edge_id)") {
		t.Fatalf("edge upsert must be idempotent: %s", call.query)
	}
}

func TestUpdateResolutionsWritesResolverOwnedFields(t *testing.T) {
	t.Parallel()

	outcome, err := Resolve(compatibleReport(), []Candidate{
		resolverCandidate("u1", "com.acme.billing.InvoiceService.render()", KindMethod, ModeQualifiedName, ConfidenceExact),
		resolverCandidate("u2", "RetryPolicy", KindClass, ModeSimpleName, ConfidenceInferred),
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	execer := &recordingExecer{}
	store := SQLStore{Execer: execer}
	if err := store.UpdateResolutions(context.Background(), outcome.Candidates); err != nil {
		t.Fatalf("update resolutions: %v", err)
	}
	if len(execer.calls) != 2 {
		t.Fatalf("expected two SQL calls, got %d", len(execer.calls))
	}
	for _, call := range execer.calls {
		if !strings.Contains(call.query, "UPDATE dif_meta.code_entity_candidates") {
			t.Fatalf("unexpected query: %s", call.query)
		}
		if strings.Contains(call.query, "INSERT") {
			t.Fatalf("resolution update must not insert candidates: %s", call.query)
		}
	}
	if execer.calls[0].args[1] != string(StatusResolved) {
		t.Fatalf("first candidate should be resolved, got %v", execer.calls[0].args[1])
	}
	if execer.calls[1].args[1] != string(StatusAmbiguous) {
		t.Fatalf("second candidate should be ambiguous, got %v", execer.calls[1].args[1])
	}
	if execer.calls[1].args[2] != nil {
		t.Fatalf("ambiguous candidate must persist NULL resolved_rif_node_id")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
