package mcpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aaraminds/dif/libs/auditusage"
	"github.com/aaraminds/dif/libs/searchdocs"
)

func TestInvokeSearchDocsRequiresAuthBeforeService(t *testing.T) {
	t.Parallel()

	service := &recordingSearchDocsService{}
	server := testServer(service)
	response, err := server.InvokeSearchDocs(context.Background(), validRequest("Bearer wrong-token"))
	if err != nil {
		t.Fatalf("invoke search_docs: %v", err)
	}
	if response.Status != StatusUnauthorized || response.Error == nil {
		t.Fatalf("expected unauthorized response, got %+v", response)
	}
	if service.called {
		t.Fatal("service was called for unauthorized request")
	}
}

func TestInvokeSearchDocsWritesDeniedGovernanceForAuthFailure(t *testing.T) {
	t.Parallel()

	store := &auditusage.MemoryStore{}
	service := &recordingSearchDocsService{}
	server := testServer(service)
	server.Governance = auditusage.Recorder{Store: store}
	response, err := server.InvokeSearchDocs(context.Background(), validRequest("Bearer wrong-token"))
	if err != nil {
		t.Fatalf("invoke search_docs: %v", err)
	}
	if response.Status != StatusUnauthorized {
		t.Fatalf("expected unauthorized response, got %+v", response)
	}
	if service.called {
		t.Fatal("service was called for unauthorized request")
	}
	if len(store.AuditEvents) != 1 || len(store.UsageEvents) != 1 {
		t.Fatalf("expected denied audit and usage writes, got %+v %+v", store.AuditEvents, store.UsageEvents)
	}
	if store.AuditEvents[0].PrincipalID != unauthenticatedPrincipal || store.AuditEvents[0].Outcome != auditusage.OutcomeDenied ||
		store.AuditEvents[0].ErrorClass != string(StatusUnauthorized) || len(store.AuditEvents[0].SourceRefs) != 0 {
		t.Fatalf("unexpected unauthorized audit event: %+v", store.AuditEvents[0])
	}
	if store.AuditEvents[0].ProjectID != unknownProjectID || store.AuditEvents[0].CorpusID != unknownCorpusID {
		t.Fatalf("unauthorized audit must use FK-safe sentinel scope, got %+v", store.AuditEvents[0])
	}
	if store.UsageEvents[0].Counts["denied_count"] != 1 || store.UsageEvents[0].Counts["result_count"] != 0 {
		t.Fatalf("unexpected unauthorized usage event: %+v", store.UsageEvents[0])
	}
}

func TestInvokeSearchDocsUsesSentinelScopeForUnauthorizedBogusCorpus(t *testing.T) {
	t.Parallel()

	store := &auditusage.MemoryStore{}
	service := &recordingSearchDocsService{}
	server := testServer(service)
	server.Governance = auditusage.Recorder{Store: store}
	request := validRequest("Bearer wrong-token")
	request.ProjectID = "attacker-project"
	request.CorpusID = "does-not-exist"

	response, err := server.InvokeSearchDocs(context.Background(), request)
	if err != nil {
		t.Fatalf("invoke search_docs: %v", err)
	}
	if response.Status != StatusUnauthorized {
		t.Fatalf("expected unauthorized response, got %+v", response)
	}
	if len(store.AuditEvents) != 1 || len(store.UsageEvents) != 1 {
		t.Fatalf("expected denied audit and usage writes, got %+v %+v", store.AuditEvents, store.UsageEvents)
	}
	if store.AuditEvents[0].ProjectID != unknownProjectID || store.AuditEvents[0].CorpusID != unknownCorpusID ||
		store.UsageEvents[0].ProjectID != unknownProjectID || store.UsageEvents[0].CorpusID != unknownCorpusID {
		t.Fatalf("expected sentinel scope for bogus unauthorized request, got %+v %+v", store.AuditEvents[0], store.UsageEvents[0])
	}
}

func TestInvokeSearchDocsValidatesRequiredFieldsBeforeService(t *testing.T) {
	t.Parallel()

	service := &recordingSearchDocsService{}
	server := testServer(service)
	request := validRequest("Bearer test-token")
	request.Query = " \t\n"
	response, err := server.InvokeSearchDocs(context.Background(), request)
	if err != nil {
		t.Fatalf("invoke search_docs: %v", err)
	}
	if response.Status != StatusInvalidRequest || response.Error == nil {
		t.Fatalf("expected invalid_request response, got %+v", response)
	}
	if !contains(response.Error.Fields, "query") {
		t.Fatalf("expected missing query field, got %+v", response.Error.Fields)
	}
	if service.called {
		t.Fatal("service was called for invalid transport request")
	}
}

func TestInvokeSearchDocsRoutesToServiceWithAuthenticatedPrincipal(t *testing.T) {
	t.Parallel()

	service := &recordingSearchDocsService{response: groundedServiceResponse()}
	server := testServer(service)
	response, err := server.InvokeSearchDocs(context.Background(), validRequest("Bearer test-token"))
	if err != nil {
		t.Fatalf("invoke search_docs: %v", err)
	}
	if response.Status != Status(searchdocs.StatusOK) {
		t.Fatalf("expected ok status, got %+v", response)
	}
	if len(response.Results) != 1 || response.Results[0].SourceRef == "" || response.Results[0].AnchorID == "" {
		t.Fatalf("expected grounded source-anchored result, got %+v", response.Results)
	}
	if !service.called {
		t.Fatal("service was not called")
	}
	if service.request.PrincipalID != "principal-from-token" {
		t.Fatalf("expected authenticated principal, got %+v", service.request)
	}
	if service.request.Query != "Who owns the architecture service?" {
		t.Fatalf("expected routed query, got %+v", service.request)
	}
}

func TestHTTPRequiresAuthBeforeMethodAndBodyValidation(t *testing.T) {
	t.Parallel()

	service := &recordingSearchDocsService{response: groundedServiceResponse()}
	server := testServer(service)
	req := httptest.NewRequest(http.MethodGet, "/search_docs", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	server.ServeSearchDocsHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized before method/body validation, got %d body=%s", rec.Code, rec.Body.String())
	}
	if service.called {
		t.Fatal("service was called for unauthenticated HTTP request")
	}
}

func TestHTTPWritesDeniedGovernanceForAuthFailure(t *testing.T) {
	t.Parallel()

	store := &auditusage.MemoryStore{}
	service := &recordingSearchDocsService{response: groundedServiceResponse()}
	server := testServer(service)
	server.Governance = auditusage.Recorder{Store: store}
	req := httptest.NewRequest(http.MethodGet, "/search_docs", strings.NewReader("{"))
	rec := httptest.NewRecorder()

	server.ServeSearchDocsHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d body=%s", rec.Code, rec.Body.String())
	}
	if service.called {
		t.Fatal("service was called for unauthenticated HTTP request")
	}
	if len(store.AuditEvents) != 1 || len(store.UsageEvents) != 1 {
		t.Fatalf("expected denied audit and usage writes, got %+v %+v", store.AuditEvents, store.UsageEvents)
	}
	if store.AuditEvents[0].ProjectID != unknownProjectID || store.AuditEvents[0].CorpusID != unknownCorpusID ||
		store.AuditEvents[0].PrincipalID != unauthenticatedPrincipal || store.AuditEvents[0].Outcome != auditusage.OutcomeDenied {
		t.Fatalf("unexpected HTTP unauthorized audit event: %+v", store.AuditEvents[0])
	}
}

func TestHTTPReturnsStructuredMissingFieldError(t *testing.T) {
	t.Parallel()

	service := &recordingSearchDocsService{response: groundedServiceResponse()}
	server := testServer(service)
	body := bytes.NewBufferString(`{"request_id":"request-001","tenant_id":"tenant-a","project_id":"dif-p0-golden","corpus_id":"golden-admitted"}`)
	req := httptest.NewRequest(http.MethodPost, "/search_docs", body)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	server.ServeSearchDocsHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d body=%s", rec.Code, rec.Body.String())
	}
	var response Response
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != StatusInvalidRequest || response.Error == nil || !contains(response.Error.Fields, "query") {
		t.Fatalf("expected structured missing query error, got %+v", response)
	}
	if service.called {
		t.Fatal("service was called for missing required HTTP field")
	}
}

func TestHTTPDoesNotExposeFreeFormAnswer(t *testing.T) {
	t.Parallel()

	service := &recordingSearchDocsService{response: groundedServiceResponse()}
	server := testServer(service)
	body := bytes.NewBufferString(`{"request_id":"request-001","tenant_id":"tenant-a","project_id":"dif-p0-golden","corpus_id":"golden-admitted","query":"Who owns the architecture service?","limit":3}`)
	req := httptest.NewRequest(http.MethodPost, "/search_docs", body)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	server.ServeSearchDocsHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected OK, got %d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"answer"`) {
		t.Fatalf("HTTP response exposed free-form answer field: %s", rec.Body.String())
	}
	var response Response
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Results) != 1 || response.Results[0].SourceRef == "" {
		t.Fatalf("expected grounded source ref, got %+v", response)
	}
}

func TestInvokeSearchDocsWritesAuditAndUsageEvents(t *testing.T) {
	t.Parallel()

	store := &auditusage.MemoryStore{}
	service := &recordingSearchDocsService{response: groundedServiceResponse()}
	server := testServer(service)
	server.Governance = auditusage.Recorder{Store: store}

	response, err := server.InvokeSearchDocs(context.Background(), validRequest("Bearer test-token"))
	if err != nil {
		t.Fatalf("invoke search_docs: %v", err)
	}
	if response.Status != Status(searchdocs.StatusOK) {
		t.Fatalf("expected ok response, got %+v", response)
	}
	if len(store.AuditEvents) != 1 || len(store.UsageEvents) != 1 {
		t.Fatalf("expected one audit and one usage event, got %+v %+v", store.AuditEvents, store.UsageEvents)
	}
	auditEvent := store.AuditEvents[0]
	if auditEvent.PrincipalID != "principal-from-token" || auditEvent.ToolName != searchdocs.ToolName ||
		auditEvent.Outcome != auditusage.OutcomeSuccess || len(auditEvent.SourceRefs) != 1 {
		t.Fatalf("unexpected audit event: %+v", auditEvent)
	}
	if strings.Contains(auditEvent.ParametersHash, "Who owns") || strings.TrimSpace(auditEvent.ParametersHash) == "" {
		t.Fatalf("parameters hash leaked query or was blank: %+v", auditEvent)
	}
	usageEvent := store.UsageEvents[0]
	if usageEvent.EventType != auditusage.EventTypeMCPToolCall || usageEvent.Counts["result_count"] != 1 ||
		usageEvent.Counts["source_ref_count"] != 1 || usageEvent.Counts["denied_count"] != 0 {
		t.Fatalf("unexpected usage event: %+v", usageEvent)
	}
	rendered, err := json.Marshal(store)
	if err != nil {
		t.Fatalf("marshal store: %v", err)
	}
	if strings.Contains(string(rendered), "Platform Architecture") || strings.Contains(string(rendered), "Who owns") {
		t.Fatalf("governance records leaked raw query/snippet text: %s", rendered)
	}
}

func TestInvokeSearchDocsWritesDeniedAuditAndUsageEvents(t *testing.T) {
	t.Parallel()

	store := &auditusage.MemoryStore{}
	service := &recordingSearchDocsService{response: searchdocs.Response{
		Status:  searchdocs.StatusCorpusNotAdmitted,
		Results: []searchdocs.Result{},
		Error: &searchdocs.ErrorDetail{
			Class:   string(searchdocs.StatusCorpusNotAdmitted),
			Message: "corpus is not admitted",
		},
		Usage: searchdocs.Usage{
			ProjectID:   "dif-p0-golden",
			CorpusID:    "golden-restricted",
			EventType:   searchdocs.ToolName,
			QueryHash:   "sha256:restricted",
			ResultCount: 0,
			Limit:       3,
			Status:      searchdocs.StatusCorpusNotAdmitted,
		},
	}}
	server := testServer(service)
	server.Governance = auditusage.Recorder{Store: store}
	request := validRequest("Bearer test-token")
	request.CorpusID = "golden-restricted"

	response, err := server.InvokeSearchDocs(context.Background(), request)
	if err != nil {
		t.Fatalf("invoke search_docs: %v", err)
	}
	if response.Status != Status(searchdocs.StatusCorpusNotAdmitted) {
		t.Fatalf("expected corpus_not_admitted response, got %+v", response)
	}
	if len(store.AuditEvents) != 1 || len(store.UsageEvents) != 1 {
		t.Fatalf("expected one audit and one usage event, got %+v %+v", store.AuditEvents, store.UsageEvents)
	}
	if store.AuditEvents[0].Outcome != auditusage.OutcomeDenied || len(store.AuditEvents[0].SourceRefs) != 0 {
		t.Fatalf("expected denied audit with no source refs, got %+v", store.AuditEvents[0])
	}
	if store.UsageEvents[0].Counts["denied_count"] != 1 || store.UsageEvents[0].Counts["result_count"] != 0 {
		t.Fatalf("expected denied usage counts, got %+v", store.UsageEvents[0])
	}
}

type recordingSearchDocsService struct {
	called   bool
	request  searchdocs.Request
	response searchdocs.Response
}

func (s *recordingSearchDocsService) SearchDocs(_ context.Context, request searchdocs.Request) (searchdocs.Response, error) {
	s.called = true
	s.request = request
	if s.response.Status != "" {
		return s.response, nil
	}
	return searchdocs.Response{Status: searchdocs.StatusNoEvidence, Results: []searchdocs.Result{}}, nil
}

func testServer(service SearchDocsService) Server {
	return Server{
		Authenticator: BearerAuthenticator{
			Token:       "test-token",
			PrincipalID: "principal-from-token",
		},
		SearchDocs: service,
	}
}

func validRequest(authorization string) SearchDocsRequest {
	return SearchDocsRequest{
		Authorization: authorization,
		RequestID:     "request-001",
		TenantID:      "tenant-a",
		ProjectID:     "dif-p0-golden",
		CorpusID:      "golden-admitted",
		Query:         "Who owns the architecture service?",
		Limit:         3,
	}
}

func groundedServiceResponse() searchdocs.Response {
	return searchdocs.Response{
		Status: searchdocs.StatusOK,
		Results: []searchdocs.Result{{
			CorpusID:          "golden-admitted",
			DocumentID:        "doc-architecture-overview",
			DocumentVersionID: "docver-architecture-overview",
			PassageID:         "passage-architecture-overview",
			Snippet:           "The platform architecture service is owned by Platform Architecture.",
			AnchorID:          "anchor-architecture-overview",
			SourceRef:         "golden-admitted@docver-architecture-overview:md:architecture-overview.md#L5-L8",
			Score:             1,
			Caveats:           []string{},
		}},
		Usage: searchdocs.Usage{
			ProjectID:   "dif-p0-golden",
			CorpusID:    "golden-admitted",
			EventType:   searchdocs.ToolName,
			QueryHash:   "sha256:test",
			ResultCount: 1,
			Limit:       3,
			Status:      searchdocs.StatusOK,
		},
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
