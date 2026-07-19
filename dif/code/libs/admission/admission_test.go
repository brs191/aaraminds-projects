package admission

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aaraminds/dif/libs/requestctx"
)

func TestGoldenManifestAdmissionMatchesExpectedCorpora(t *testing.T) {
	t.Parallel()

	catalog := loadGolden(t)

	admitted := catalog.CheckCorpus(executionContext(t, "golden-admitted"), "search_docs", "sha256:params")
	if !admitted.Allowed || admitted.Status != StatusOK {
		t.Fatalf("expected golden-admitted to pass admission, got %+v", admitted)
	}
	if admitted.AuditIntent != nil {
		t.Fatalf("did not expect audit denial intent for admitted corpus: %+v", admitted.AuditIntent)
	}

	restricted := catalog.CheckCorpus(executionContext(t, "golden-restricted"), "search_docs", "sha256:params")
	if restricted.Allowed || restricted.Status != StatusCorpusNotAdmitted {
		t.Fatalf("expected golden-restricted to fail closed, got %+v", restricted)
	}
	if restricted.AuditIntent == nil || restricted.AuditIntent.Outcome != AuditOutcomeDenied {
		t.Fatalf("expected denied audit intent, got %+v", restricted.AuditIntent)
	}
}

func TestMissingCorpusFailsClosedWithCorpusNotAdmitted(t *testing.T) {
	t.Parallel()

	catalog := loadGolden(t)
	decision := catalog.CheckCorpus(executionContext(t, "missing-corpus"), "search_docs", "sha256:params")

	if decision.Allowed {
		t.Fatal("expected missing corpus to fail closed")
	}
	if decision.Status != StatusCorpusNotAdmitted {
		t.Fatalf("expected %q, got %q", StatusCorpusNotAdmitted, decision.Status)
	}
	if decision.AuditIntent == nil || decision.AuditIntent.ErrorClass != string(StatusCorpusNotAdmitted) {
		t.Fatalf("expected corpus_not_admitted audit intent, got %+v", decision.AuditIntent)
	}
}

func TestNonUniformReadableCorpusIsRejected(t *testing.T) {
	t.Parallel()

	corpus := Corpus{
		CorpusID:         "mixed-corpus",
		ProjectID:        "dif-p0-golden",
		DisplayName:      "Mixed Corpus",
		AdmissionStatus:  AdmissionAdmitted,
		ReadabilityModel: ReadabilityModel("mixed_permissions"),
	}
	err := corpus.Admit()
	if err == nil {
		t.Fatal("expected non-uniform readable corpus to be rejected")
	}
	if !strings.Contains(err.Error(), "uniform_readable") {
		t.Fatalf("expected uniform_readable error, got %q", err.Error())
	}
}

func TestProjectMismatchFailsClosed(t *testing.T) {
	t.Parallel()

	catalog := loadGolden(t)
	ctx, err := requestctx.WithExecutionContext(context.Background(), requestctx.ExecutionContext{
		RequestID:   "request-001",
		PrincipalID: "principal-001",
		TenantID:    "tenant-a",
		ProjectID:   "wrong-project",
		CorpusID:    "golden-admitted",
		ToolName:    "search_docs",
	}, requestctx.OperationRetrieval)
	if err != nil {
		t.Fatalf("attach context: %v", err)
	}

	decision := catalog.CheckCorpus(ctx, "search_docs", "sha256:params")
	if decision.Allowed || decision.Status != StatusCorpusNotAdmitted {
		t.Fatalf("expected project mismatch to fail closed, got %+v", decision)
	}
}

func TestSourceAdmissionUsesCorpusBoundary(t *testing.T) {
	t.Parallel()

	catalog := loadGolden(t)

	admitted := catalog.CheckSource(executionContext(t, "golden-admitted"), "src-golden-admitted-local", "search_docs", "sha256:params")
	if !admitted.Allowed || admitted.SourceID != "src-golden-admitted-local" {
		t.Fatalf("expected admitted source, got %+v", admitted)
	}

	restricted := catalog.CheckSource(executionContext(t, "golden-restricted"), "src-golden-restricted-local", "search_docs", "sha256:params")
	if restricted.Allowed || restricted.Status != StatusCorpusNotAdmitted {
		t.Fatalf("expected restricted source corpus to fail at corpus gate, got %+v", restricted)
	}

	missing := catalog.CheckSource(executionContext(t, "golden-admitted"), "missing-source", "search_docs", "sha256:params")
	if missing.Allowed || missing.Status != StatusSourceNotAdmitted {
		t.Fatalf("expected missing source to fail closed, got %+v", missing)
	}
}

func TestMissingExecutionContextFailsClosed(t *testing.T) {
	t.Parallel()

	decision := Catalog{}.CheckCorpus(context.Background(), "search_docs", "sha256:params")
	if decision.Allowed || decision.Status != StatusCorpusNotAdmitted {
		t.Fatalf("expected missing execution context to fail closed, got %+v", decision)
	}
	if decision.AuditIntent == nil || decision.AuditIntent.ErrorClass != "missing_execution_context" {
		t.Fatalf("expected missing execution context audit intent, got %+v", decision.AuditIntent)
	}
}

func loadGolden(t *testing.T) Catalog {
	t.Helper()

	catalog, err := LoadGoldenManifest(filepath.Join("..", "..", "..", "evaluation", "golden", "manifest.json"))
	if err != nil {
		t.Fatalf("load golden manifest: %v", err)
	}
	return catalog
}

func executionContext(t *testing.T, corpusID string) context.Context {
	t.Helper()

	ctx, err := requestctx.WithExecutionContext(context.Background(), requestctx.ExecutionContext{
		RequestID:   "request-001",
		PrincipalID: "principal-001",
		TenantID:    "tenant-a",
		ProjectID:   "dif-p0-golden",
		CorpusID:    corpusID,
		ToolName:    "search_docs",
	}, requestctx.OperationRetrieval)
	if err != nil {
		t.Fatalf("attach context: %v", err)
	}
	return ctx
}
