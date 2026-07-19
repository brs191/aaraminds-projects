package searchdocs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aaraminds/dif/libs/admission"
	"github.com/aaraminds/dif/libs/extraction"
	"github.com/aaraminds/dif/libs/retrieval"
)

func TestGoldenQueriesReturnAnchoredServiceResults(t *testing.T) {
	t.Parallel()

	service := goldenService(t)
	queries := loadGoldenQueries(t)
	for _, query := range queries.Queries {
		query := query
		t.Run(query.QueryID, func(t *testing.T) {
			t.Parallel()
			response, err := service.SearchDocs(context.Background(), goldenRequest(query.CorpusID, query.Query, query.AcceptedResultCount.Max))
			if err != nil {
				t.Fatalf("search docs service: %v", err)
			}
			expectedStatus := StatusOK
			if query.RequiredStatus != "" {
				expectedStatus = Status(query.RequiredStatus)
			} else if len(query.RequiredTopSourceRefs) == 0 {
				expectedStatus = StatusNoEvidence
			}
			if response.Status != expectedStatus {
				t.Fatalf("expected status %q, got %+v", expectedStatus, response)
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
				assertServiceResultShape(t, result)
			}
		})
	}
}

func TestServiceFailsClosedBeforeRetrieverForNonAdmittedCorpus(t *testing.T) {
	t.Parallel()

	fake := &recordingRetriever{}
	service := Service{Admission: goldenAdmission(t), Retriever: fake}
	response, err := service.SearchDocs(context.Background(), goldenRequest("golden-restricted", "restricted", 5))
	if err != nil {
		t.Fatalf("search docs service: %v", err)
	}
	if response.Status != StatusCorpusNotAdmitted || len(response.Results) != 0 {
		t.Fatalf("expected corpus_not_admitted with no results, got %+v", response)
	}
	if fake.called {
		t.Fatal("retriever was called before admission rejected the corpus")
	}
	if response.AuditIntent == nil || response.AuditIntent.Outcome != admission.AuditOutcomeDenied {
		t.Fatalf("expected denied audit intent, got %+v", response.AuditIntent)
	}
}

func TestServiceInvalidRequestReportsMissingFields(t *testing.T) {
	t.Parallel()

	service := goldenService(t)
	response, err := service.SearchDocs(context.Background(), Request{
		RequestID:   "request-001",
		PrincipalID: "principal-001",
		TenantID:    "tenant-a",
		ProjectID:   "dif-p0-golden",
		CorpusID:    "golden-admitted",
		Query:       " \n\t",
	})
	if err != nil {
		t.Fatalf("search docs service: %v", err)
	}
	if response.Status != StatusInvalidRequest || response.Error == nil {
		t.Fatalf("expected invalid_request error response, got %+v", response)
	}
	if !contains(response.Error.Fields, "query") {
		t.Fatalf("expected missing query field, got %+v", response.Error.Fields)
	}
}

func TestServiceFailsClosedForUnanchoredRetrieverResult(t *testing.T) {
	t.Parallel()

	service := Service{
		Admission: goldenAdmission(t),
		Retriever: staticRetriever{response: retrieval.Response{
			Status: retrieval.StatusOK,
			Results: []retrieval.Result{{
				CorpusID:          "golden-admitted",
				DocumentID:        "doc-architecture-overview",
				DocumentVersionID: "docver-architecture-overview",
				PassageID:         "passage-001",
				Snippet:           "Platform Architecture owns the service.",
				Score:             1,
			}},
		}},
	}
	response, err := service.SearchDocs(context.Background(), goldenRequest("golden-admitted", "owner", 5))
	if err != nil {
		t.Fatalf("search docs service: %v", err)
	}
	if response.Status != StatusInvalidResult || len(response.Results) != 0 {
		t.Fatalf("expected invalid_result with no leaked results, got %+v", response)
	}
}

func TestServiceResponseDoesNotExposeFreeFormAnswer(t *testing.T) {
	t.Parallel()

	service := goldenService(t)
	response, err := service.SearchDocs(context.Background(), goldenRequest("golden-admitted", "Who owns the architecture service?", 3))
	if err != nil {
		t.Fatalf("search docs service: %v", err)
	}
	rendered, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if strings.Contains(string(rendered), `"answer"`) {
		t.Fatalf("service response exposed a free-form answer field: %s", rendered)
	}
}

type recordingRetriever struct {
	called bool
}

func (r *recordingRetriever) SearchDocs(context.Context, retrieval.Query) (retrieval.Response, error) {
	r.called = true
	return retrieval.Response{Status: retrieval.StatusNoEvidence, Results: []retrieval.Result{}}, nil
}

type staticRetriever struct {
	response retrieval.Response
}

func (r staticRetriever) SearchDocs(context.Context, retrieval.Query) (retrieval.Response, error) {
	return r.response, nil
}

func goldenService(t *testing.T) Service {
	t.Helper()
	searcher, err := retrieval.NewSearcher(goldenAdmission(t), mustExtractMarkdown(t), mustExtractText(t), mustExtractDOCX(t), mustExtractJSON(t))
	if err != nil {
		t.Fatalf("new searcher: %v", err)
	}
	return Service{Admission: goldenAdmission(t), Retriever: searcher}
}

func goldenAdmission(t *testing.T) admission.Catalog {
	t.Helper()
	catalog, err := admission.LoadGoldenManifest(filepath.Join("..", "..", "..", "evaluation", "golden", "manifest.json"))
	if err != nil {
		t.Fatalf("load golden manifest: %v", err)
	}
	return catalog
}

func goldenRequest(corpusID, query string, limit int) Request {
	return Request{
		RequestID:   "request-001",
		PrincipalID: "principal-001",
		TenantID:    "tenant-a",
		ProjectID:   "dif-p0-golden",
		CorpusID:    corpusID,
		Query:       query,
		Limit:       limit,
	}
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

func assertServiceResultShape(t *testing.T, result Result) {
	t.Helper()
	if result.CorpusID == "" || result.DocumentID == "" || result.DocumentVersionID == "" || result.PassageID == "" ||
		result.Snippet == "" || result.AnchorID == "" || result.SourceRef == "" || result.Score <= 0 || result.Caveats == nil {
		t.Fatalf("service result missing required fields: %+v", result)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
