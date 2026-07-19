// Package searchdocs implements the service-layer search_docs contract.
package searchdocs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/aaraminds/dif/libs/admission"
	"github.com/aaraminds/dif/libs/requestctx"
	"github.com/aaraminds/dif/libs/retrieval"
	"github.com/aaraminds/dif/libs/sourceanchors"
)

const (
	ToolName = "search_docs"

	StatusOK                Status = "ok"
	StatusNoEvidence        Status = "no_evidence"
	StatusCorpusNotAdmitted Status = "corpus_not_admitted"
	StatusInvalidRequest    Status = "invalid_request"
	StatusInvalidResult     Status = "invalid_result"
)

// Status is the explicit service response state.
type Status string

// Retriever is the retrieval dependency used by the service layer.
type Retriever interface {
	SearchDocs(context.Context, retrieval.Query) (retrieval.Response, error)
}

// Service validates request scope, enforces admission, and returns only
// source-anchored retrieval evidence. It does not generate free-form answers.
type Service struct {
	Admission admission.Catalog
	Retriever Retriever
}

// Request is the service-layer search_docs input.
type Request struct {
	RequestID   string
	PrincipalID string
	TenantID    string
	ProjectID   string
	CorpusID    string
	Query       string
	Limit       int
}

// Response is the service-layer search_docs output.
type Response struct {
	Status      Status                 `json:"status"`
	Results     []Result               `json:"results"`
	Error       *ErrorDetail           `json:"error,omitempty"`
	AuditIntent *admission.AuditIntent `json:"audit_intent,omitempty"`
	Usage       Usage                  `json:"usage"`
}

// Result is a source-anchored evidence hit.
type Result struct {
	CorpusID          string   `json:"corpus_id"`
	DocumentID        string   `json:"document_id"`
	DocumentVersionID string   `json:"document_version_id"`
	PassageID         string   `json:"passage_id"`
	Snippet           string   `json:"snippet"`
	AnchorID          string   `json:"anchor_id"`
	SourceRef         string   `json:"source_ref"`
	Score             float64  `json:"score"`
	Caveats           []string `json:"caveats"`
}

// ErrorDetail is a structured fail-closed service error.
type ErrorDetail struct {
	Class   string   `json:"class"`
	Message string   `json:"message"`
	Fields  []string `json:"fields,omitempty"`
}

// Usage captures non-PII service metering dimensions.
type Usage struct {
	ProjectID   string `json:"project_id"`
	CorpusID    string `json:"corpus_id"`
	EventType   string `json:"event_type"`
	QueryHash   string `json:"query_hash"`
	ResultCount int    `json:"result_count"`
	Limit       int    `json:"limit"`
	Status      Status `json:"status"`
}

// SearchDocs executes the P0 service-layer search_docs contract.
func (s Service) SearchDocs(ctx context.Context, request Request) (Response, error) {
	if s.Retriever == nil {
		return Response{}, errors.New("searchdocs service requires a retriever")
	}

	normalized := normalizeRequest(request)
	queryHash := parametersHash(normalized.Query)
	usage := Usage{
		ProjectID: normalized.ProjectID,
		CorpusID:  normalized.CorpusID,
		EventType: ToolName,
		QueryHash: queryHash,
		Limit:     normalized.Limit,
	}

	if missing := missingRequestFields(normalized); len(missing) > 0 {
		return withStatus(Response{
			Results: []Result{},
			Error: &ErrorDetail{
				Class:   string(StatusInvalidRequest),
				Message: "missing required search_docs request fields",
				Fields:  missing,
			},
			Usage: usage,
		}, StatusInvalidRequest), nil
	}

	exec := requestctx.ExecutionContext{
		RequestID:   normalized.RequestID,
		PrincipalID: normalized.PrincipalID,
		TenantID:    normalized.TenantID,
		ProjectID:   normalized.ProjectID,
		CorpusID:    normalized.CorpusID,
		ToolName:    ToolName,
	}
	scopedCtx, err := requestctx.WithExecutionContext(ctx, exec, requestctx.OperationRetrieval)
	if err != nil {
		return withStatus(Response{
			Results: []Result{},
			Error: &ErrorDetail{
				Class:   string(StatusInvalidRequest),
				Message: err.Error(),
			},
			Usage: usage,
		}, StatusInvalidRequest), nil
	}

	decision := s.Admission.CheckCorpus(scopedCtx, ToolName, queryHash)
	if !decision.Allowed {
		return withStatus(Response{
			Results:     []Result{},
			AuditIntent: decision.AuditIntent,
			Error: &ErrorDetail{
				Class:   string(StatusCorpusNotAdmitted),
				Message: firstNonEmpty(decision.FailureReason, "corpus is not admitted"),
			},
			Usage: usage,
		}, StatusCorpusNotAdmitted), nil
	}

	retrievalResponse, err := s.Retriever.SearchDocs(scopedCtx, retrieval.Query{Text: normalized.Query, Limit: normalized.Limit})
	if err != nil {
		return Response{}, err
	}
	response, err := fromRetrievalResponse(retrievalResponse, usage)
	if err != nil {
		return Response{}, err
	}
	return response, nil
}

func fromRetrievalResponse(response retrieval.Response, usage Usage) (Response, error) {
	switch response.Status {
	case retrieval.StatusOK:
		results, err := convertResults(response.Results)
		if err != nil {
			return withStatus(Response{
				Results: []Result{},
				Error: &ErrorDetail{
					Class:   string(StatusInvalidResult),
					Message: err.Error(),
				},
				Usage: usage,
			}, StatusInvalidResult), nil
		}
		out := Response{Results: results, Usage: usage}
		return withStatus(out, StatusOK), nil
	case retrieval.StatusNoEvidence:
		return withStatus(Response{Results: []Result{}, Usage: usage}, StatusNoEvidence), nil
	case retrieval.StatusCorpusNotAdmitted:
		return withStatus(Response{Results: []Result{}, Usage: usage}, StatusCorpusNotAdmitted), nil
	default:
		return Response{}, fmt.Errorf("unsupported retrieval status %q", response.Status)
	}
}

func convertResults(results []retrieval.Result) ([]Result, error) {
	converted := make([]Result, 0, len(results))
	for _, result := range results {
		if err := validateAnchoredResult(result); err != nil {
			return nil, err
		}
		converted = append(converted, Result{
			CorpusID:          strings.TrimSpace(result.CorpusID),
			DocumentID:        strings.TrimSpace(result.DocumentID),
			DocumentVersionID: strings.TrimSpace(result.DocumentVersionID),
			PassageID:         strings.TrimSpace(result.PassageID),
			Snippet:           strings.TrimSpace(result.Snippet),
			AnchorID:          strings.TrimSpace(result.AnchorID),
			SourceRef:         strings.TrimSpace(result.SourceRef),
			Score:             result.Score,
			Caveats:           append([]string{}, result.Caveats...),
		})
	}
	if len(converted) == 0 {
		return nil, errors.New("ok retrieval response contained no anchored results")
	}
	return converted, nil
}

func validateAnchoredResult(result retrieval.Result) error {
	required := map[string]string{
		"corpus_id":           result.CorpusID,
		"document_id":         result.DocumentID,
		"document_version_id": result.DocumentVersionID,
		"passage_id":          result.PassageID,
		"snippet":             result.Snippet,
		"anchor_id":           result.AnchorID,
		"source_ref":          result.SourceRef,
	}
	var missing []string
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("retrieval result missing required anchored fields: %s", strings.Join(missing, ", "))
	}
	if result.Score <= 0 {
		return errors.New("retrieval result score must be positive")
	}
	parsed, err := sourceanchors.ParseSourceRef(result.SourceRef)
	if err != nil {
		return fmt.Errorf("retrieval result source_ref is invalid: %w", err)
	}
	if parsed.CorpusID != strings.TrimSpace(result.CorpusID) {
		return fmt.Errorf("retrieval result source_ref corpus %q does not match result corpus %q", parsed.CorpusID, result.CorpusID)
	}
	if parsed.DocumentVersionID != strings.TrimSpace(result.DocumentVersionID) {
		return fmt.Errorf("retrieval result source_ref document_version_id %q does not match result document_version_id %q", parsed.DocumentVersionID, result.DocumentVersionID)
	}
	return nil
}

func normalizeRequest(request Request) Request {
	return Request{
		RequestID:   strings.TrimSpace(request.RequestID),
		PrincipalID: strings.TrimSpace(request.PrincipalID),
		TenantID:    strings.TrimSpace(request.TenantID),
		ProjectID:   strings.TrimSpace(request.ProjectID),
		CorpusID:    strings.TrimSpace(request.CorpusID),
		Query:       strings.TrimSpace(request.Query),
		Limit:       request.Limit,
	}
}

func missingRequestFields(request Request) []string {
	values := map[string]string{
		"request_id":   request.RequestID,
		"principal_id": request.PrincipalID,
		"tenant_id":    request.TenantID,
		"project_id":   request.ProjectID,
		"corpus_id":    request.CorpusID,
		"query":        request.Query,
	}
	fields := []string{"request_id", "principal_id", "tenant_id", "project_id", "corpus_id", "query"}
	var missing []string
	for _, field := range fields {
		if values[field] == "" {
			missing = append(missing, field)
		}
	}
	return missing
}

func withStatus(response Response, status Status) Response {
	response.Status = status
	response.Usage.Status = status
	response.Usage.ResultCount = len(response.Results)
	return response
}

func parametersHash(query string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(query)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
