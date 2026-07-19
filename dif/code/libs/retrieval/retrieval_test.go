package retrieval

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aaraminds/dif/libs/admission"
	"github.com/aaraminds/dif/libs/extraction"
	"github.com/aaraminds/dif/libs/requestctx"
)

func TestGoldenQueriesReturnRequiredAnchoredResults(t *testing.T) {
	t.Parallel()

	searcher := goldenSearcher(t)
	queries := loadGoldenQueries(t)
	for _, query := range queries.Queries {
		query := query
		t.Run(query.QueryID, func(t *testing.T) {
			t.Parallel()
			response, err := searcher.SearchDocs(executionContext(t, query.CorpusID), Query{Text: query.Query, Limit: query.AcceptedResultCount.Max})
			if err != nil {
				t.Fatalf("search docs: %v", err)
			}
			if query.RequiredStatus != "" {
				if response.Status != Status(query.RequiredStatus) {
					t.Fatalf("expected status %q, got %+v", query.RequiredStatus, response)
				}
			} else if len(query.RequiredTopSourceRefs) == 0 {
				if response.Status != StatusNoEvidence || len(response.Results) != 0 {
					t.Fatalf("expected no_evidence with no results, got %+v", response)
				}
			} else {
				if response.Status != StatusOK {
					t.Fatalf("expected ok status, got %+v", response)
				}
			}
			if len(response.Results) < query.AcceptedResultCount.Min || len(response.Results) > query.AcceptedResultCount.Max {
				t.Fatalf("expected result count between %+v, got %+v", query.AcceptedResultCount, response.Results)
			}
			for index, sourceRef := range query.RequiredTopSourceRefs {
				if index >= len(response.Results) {
					t.Fatalf("missing required top source_ref %q in %+v", sourceRef, response.Results)
				}
				if response.Results[index].SourceRef != sourceRef {
					t.Fatalf("expected top source_ref %q, got %+v", sourceRef, response.Results)
				}
			}
			for _, result := range response.Results {
				assertResultShape(t, result)
			}
		})
	}
}

func TestRetrievalExcludesUnanchoredPassages(t *testing.T) {
	t.Parallel()

	result := mustExtractMarkdown(t)
	result.Passages = append([]extraction.Passage{{
		PassageID:         "unanchored-high-signal",
		CorpusID:          result.Document.CorpusID,
		DocumentID:        result.Document.DocumentID,
		DocumentVersionID: result.Document.DocumentVersionID,
		NodeID:            result.Nodes[0].NodeID,
		Text:              "Platform Architecture owns everything about the architecture service.",
		ContentHash:       "sha256:unanchored",
	}}, result.Passages...)

	index, err := NewIndex(result)
	if err != nil {
		t.Fatalf("new index: %v", err)
	}
	searcher := Searcher{Admission: goldenAdmission(t), Index: index}
	response, err := searcher.SearchDocs(executionContext(t, "golden-admitted"), Query{Text: "Who owns the architecture service?", Limit: 10})
	if err != nil {
		t.Fatalf("search docs: %v", err)
	}
	for _, result := range response.Results {
		if result.PassageID == "unanchored-high-signal" {
			t.Fatalf("unanchored passage leaked into results: %+v", response.Results)
		}
	}
}

func TestRetrievalRejectsSourceRefMismatchDuringIndexBuild(t *testing.T) {
	t.Parallel()

	result := mustExtractText(t)
	result.Passages[0].SourceRef = "golden-admitted@docver-runbook:txt:runbook.txt#L1-L1"
	if _, err := NewIndex(result); err == nil || !strings.Contains(err.Error(), "source_ref does not match anchor") {
		t.Fatalf("expected source_ref mismatch error, got %v", err)
	}
}

func TestSearchRequiresExecutionContextAndQuery(t *testing.T) {
	t.Parallel()

	searcher := goldenSearcher(t)
	if _, err := searcher.SearchDocs(context.Background(), Query{Text: "owner"}); err == nil {
		t.Fatal("expected missing execution context error")
	}
	if _, err := searcher.SearchDocs(executionContext(t, "golden-admitted"), Query{Text: " \n\t"}); err == nil {
		t.Fatal("expected blank query error")
	}
}

func goldenSearcher(t *testing.T) Searcher {
	t.Helper()
	searcher, err := NewSearcher(goldenAdmission(t), mustExtractMarkdown(t), mustExtractText(t), mustExtractDOCX(t), mustExtractJSON(t))
	if err != nil {
		t.Fatalf("new searcher: %v", err)
	}
	return searcher
}

func goldenAdmission(t *testing.T) admission.Catalog {
	t.Helper()
	catalog, err := admission.LoadGoldenManifest(filepath.Join("..", "..", "..", "evaluation", "golden", "manifest.json"))
	if err != nil {
		t.Fatalf("load golden manifest: %v", err)
	}
	return catalog
}

func mustExtractMarkdown(t *testing.T) extraction.Result {
	t.Helper()
	result, err := extraction.ExtractMarkdown(readGolden(t, "architecture-overview.md"), extraction.Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-architecture-overview",
		DocumentVersionID: "docver-architecture-overview",
		SourceID:          "src-golden-admitted-local",
		Path:              "architecture-overview.md",
	})
	if err != nil {
		t.Fatalf("extract markdown: %v", err)
	}
	return result
}

func mustExtractText(t *testing.T) extraction.Result {
	t.Helper()
	result, err := extraction.ExtractText(readGolden(t, "runbook.txt"), extraction.Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-runbook",
		DocumentVersionID: "docver-runbook",
		SourceID:          "src-golden-admitted-local",
		Path:              "runbook.txt",
	})
	if err != nil {
		t.Fatalf("extract text: %v", err)
	}
	return result
}

func mustExtractDOCX(t *testing.T) extraction.Result {
	t.Helper()
	result, err := extraction.ExtractDOCXParagraphFixture(readGolden(t, "requirements.docx.fixture.json"), extraction.Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-requirements",
		DocumentVersionID: "docver-requirements",
		SourceID:          "src-golden-admitted-local",
		Path:              "requirements.docx",
	})
	if err != nil {
		t.Fatalf("extract DOCX: %v", err)
	}
	return result
}

func mustExtractJSON(t *testing.T) extraction.Result {
	t.Helper()
	result, err := extraction.ExtractJSON(readGolden(t, "service-config.json"), extraction.Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-service-config",
		DocumentVersionID: "docver-service-config",
		SourceID:          "src-golden-admitted-local",
		Path:              "service-config.json",
	})
	if err != nil {
		t.Fatalf("extract JSON: %v", err)
	}
	return result
}

func readGolden(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "evaluation", "golden", "sources", "admitted", name))
	if err != nil {
		t.Fatalf("read golden fixture %s: %v", name, err)
	}
	return string(content)
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
		t.Fatalf("attach execution context: %v", err)
	}
	return ctx
}

func assertResultShape(t *testing.T, result Result) {
	t.Helper()
	if result.CorpusID == "" || result.DocumentID == "" || result.DocumentVersionID == "" || result.PassageID == "" ||
		result.Snippet == "" || result.AnchorID == "" || result.SourceRef == "" || result.Score <= 0 || result.Caveats == nil {
		t.Fatalf("retrieval result missing required fields: %+v", result)
	}
}

type goldenQueries struct {
	Queries []struct {
		QueryID               string   `json:"query_id"`
		CorpusID              string   `json:"corpus_id"`
		Query                 string   `json:"query"`
		AcceptedResultCount   bounds   `json:"accepted_result_count"`
		RequiredStatus        string   `json:"required_status"`
		RequiredTopSourceRefs []string `json:"required_top_source_refs"`
	} `json:"queries"`
}

type bounds struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

func loadGoldenQueries(t *testing.T) goldenQueries {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "evaluation", "golden", "golden-queries.json"))
	if err != nil {
		t.Fatalf("read golden queries: %v", err)
	}
	var queries goldenQueries
	if err := json.Unmarshal(content, &queries); err != nil {
		t.Fatalf("parse golden queries: %v", err)
	}
	return queries
}
