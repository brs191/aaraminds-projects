// Package mcpapi provides the thin P0 MCP/API boundary for search_docs.
package mcpapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aaraminds/dif/libs/admission"
	"github.com/aaraminds/dif/libs/auditusage"
	"github.com/aaraminds/dif/libs/searchdocs"
)

const (
	StatusUnauthorized   Status = "unauthorized"
	StatusInvalidRequest Status = "invalid_request"
	StatusInternalError  Status = "internal_error"

	unauthenticatedPrincipal = "unauthenticated"
	unknownProjectID         = "dif-auth-unknown-project"
	unknownCorpusID          = "dif-auth-unknown-corpus"
)

// Status is a transport-layer response status. It also carries service-layer
// statuses such as ok, no_evidence, and corpus_not_admitted.
type Status string

// SearchDocsService is the service-layer dependency used by the transport.
type SearchDocsService interface {
	SearchDocs(context.Context, searchdocs.Request) (searchdocs.Response, error)
}

// Server validates auth and request shape, then routes to the searchdocs
// service. It intentionally does not duplicate retrieval, ranking, graph
// traversal, or answer generation logic.
type Server struct {
	Authenticator BearerAuthenticator
	SearchDocs    SearchDocsService
	Governance    auditusage.Recorder
}

// BearerAuthenticator implements the P0 internal bearer-token gate. Pilot and
// remote deployments must replace this with OAuth 2.1 + PKCE.
type BearerAuthenticator struct {
	Token       string
	PrincipalID string
}

// SearchDocsRequest is the transport/MCP tool input.
type SearchDocsRequest struct {
	Authorization string `json:"authorization,omitempty"`
	RequestID     string `json:"request_id"`
	TenantID      string `json:"tenant_id"`
	ProjectID     string `json:"project_id"`
	CorpusID      string `json:"corpus_id"`
	Query         string `json:"query"`
	Limit         int    `json:"limit,omitempty"`
}

// Response is the transport response envelope.
type Response struct {
	Status      Status                 `json:"status"`
	Results     []searchdocs.Result    `json:"results"`
	Error       *ErrorDetail           `json:"error,omitempty"`
	AuditIntent *admission.AuditIntent `json:"audit_intent,omitempty"`
	Usage       searchdocs.Usage       `json:"usage"`
}

// ErrorDetail is a structured transport error.
type ErrorDetail struct {
	Class   string   `json:"class"`
	Message string   `json:"message"`
	Fields  []string `json:"fields,omitempty"`
}

// AuthResult is the authenticated principal.
type AuthResult struct {
	PrincipalID string
}

// Authenticate validates a bearer token using constant-time comparison of
// fixed-length hashes.
func (a BearerAuthenticator) Authenticate(header string) (AuthResult, bool) {
	expectedToken := strings.TrimSpace(a.Token)
	principalID := strings.TrimSpace(a.PrincipalID)
	if expectedToken == "" || principalID == "" {
		return AuthResult{}, false
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return AuthResult{}, false
	}
	presentedToken := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if presentedToken == "" {
		return AuthResult{}, false
	}

	expectedHash := sha256.Sum256([]byte(expectedToken))
	presentedHash := sha256.Sum256([]byte(presentedToken))
	if subtle.ConstantTimeCompare(expectedHash[:], presentedHash[:]) != 1 {
		return AuthResult{}, false
	}
	return AuthResult{PrincipalID: principalID}, true
}

// InvokeSearchDocs executes the MCP/tool-style search_docs call.
func (s Server) InvokeSearchDocs(ctx context.Context, request SearchDocsRequest) (Response, error) {
	started := time.Now()
	if s.SearchDocs == nil {
		return Response{}, errors.New("mcpapi server requires search_docs service")
	}

	auth, ok := s.Authenticator.Authenticate(strings.TrimSpace(request.Authorization))
	if !ok {
		if err := s.recordUnauthorized(ctx, request, time.Since(started)); err != nil {
			return Response{}, err
		}
		return response(StatusUnauthorized, ErrorDetail{
			Class:   string(StatusUnauthorized),
			Message: "missing or invalid bearer token",
		}), nil
	}

	normalized := normalizeRequest(request)
	if missing := missingRequestFields(normalized); len(missing) > 0 {
		return response(StatusInvalidRequest, ErrorDetail{
			Class:   string(StatusInvalidRequest),
			Message: "missing required search_docs transport fields",
			Fields:  missing,
		}), nil
	}

	serviceResponse, err := s.SearchDocs.SearchDocs(ctx, searchdocs.Request{
		RequestID:   normalized.RequestID,
		PrincipalID: auth.PrincipalID,
		TenantID:    normalized.TenantID,
		ProjectID:   normalized.ProjectID,
		CorpusID:    normalized.CorpusID,
		Query:       normalized.Query,
		Limit:       normalized.Limit,
	})
	if err != nil {
		return Response{}, err
	}
	out := fromServiceResponse(serviceResponse)
	if s.Governance.Enabled() {
		if _, _, err := s.Governance.RecordMCPToolCall(ctx, auditusage.MCPToolCall{
			PrincipalID:    auth.PrincipalID,
			TenantID:       normalized.TenantID,
			ProjectID:      normalized.ProjectID,
			CorpusID:       normalized.CorpusID,
			ToolName:       searchdocs.ToolName,
			ToolVersion:    auditusage.ToolVersionP0,
			ParametersHash: governanceParametersHash(normalized, serviceResponse),
			Outcome:        governanceOutcome(out.Status),
			Latency:        time.Since(started),
			SourceRefs:     sourceRefs(out.Results),
			ErrorClass:     governanceErrorClass(out),
			Counts:         governanceCounts(out),
		}); err != nil {
			return Response{}, err
		}
	}
	return out, nil
}

// ServeSearchDocsHTTP exposes a JSON HTTP skeleton for search_docs. The same
// service routing is used by InvokeSearchDocs.
func (s Server) ServeSearchDocsHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	authHeader := r.Header.Get("Authorization")
	if _, ok := s.Authenticator.Authenticate(authHeader); !ok {
		if err := s.recordUnauthorized(r.Context(), SearchDocsRequest{}, 0); err != nil {
			out := response(StatusInternalError, ErrorDetail{
				Class:   string(StatusInternalError),
				Message: err.Error(),
			})
			writeJSON(w, http.StatusInternalServerError, out)
			return
		}
		out := response(StatusUnauthorized, ErrorDetail{
			Class:   string(StatusUnauthorized),
			Message: "missing or invalid bearer token",
		})
		writeJSON(w, http.StatusUnauthorized, out)
		return
	}

	request := SearchDocsRequest{Authorization: authHeader}
	if r.Body != nil {
		defer r.Body.Close()
	}
	if r.Method != http.MethodPost {
		out := response(StatusInvalidRequest, ErrorDetail{
			Class:   string(StatusInvalidRequest),
			Message: "search_docs requires POST",
		})
		writeJSON(w, http.StatusBadRequest, out)
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		out := response(StatusInvalidRequest, ErrorDetail{
			Class:   string(StatusInvalidRequest),
			Message: fmt.Sprintf("invalid JSON request: %v", err),
		})
		writeJSON(w, http.StatusBadRequest, out)
		return
	}
	request.Authorization = authHeader

	out, err := s.InvokeSearchDocs(r.Context(), request)
	if err != nil {
		out = response(StatusInternalError, ErrorDetail{
			Class:   string(StatusInternalError),
			Message: err.Error(),
		})
		writeJSON(w, http.StatusInternalServerError, out)
		return
	}
	writeJSON(w, statusCode(out.Status), out)
}

func fromServiceResponse(serviceResponse searchdocs.Response) Response {
	var err *ErrorDetail
	if serviceResponse.Error != nil {
		err = &ErrorDetail{
			Class:   serviceResponse.Error.Class,
			Message: serviceResponse.Error.Message,
			Fields:  append([]string{}, serviceResponse.Error.Fields...),
		}
	}
	return Response{
		Status:      Status(serviceResponse.Status),
		Results:     append([]searchdocs.Result{}, serviceResponse.Results...),
		Error:       err,
		AuditIntent: serviceResponse.AuditIntent,
		Usage:       serviceResponse.Usage,
	}
}

func normalizeRequest(request SearchDocsRequest) SearchDocsRequest {
	return SearchDocsRequest{
		Authorization: strings.TrimSpace(request.Authorization),
		RequestID:     strings.TrimSpace(request.RequestID),
		TenantID:      strings.TrimSpace(request.TenantID),
		ProjectID:     strings.TrimSpace(request.ProjectID),
		CorpusID:      strings.TrimSpace(request.CorpusID),
		Query:         strings.TrimSpace(request.Query),
		Limit:         request.Limit,
	}
}

func missingRequestFields(request SearchDocsRequest) []string {
	values := map[string]string{
		"request_id": request.RequestID,
		"tenant_id":  request.TenantID,
		"project_id": request.ProjectID,
		"corpus_id":  request.CorpusID,
		"query":      request.Query,
	}
	fields := []string{"request_id", "tenant_id", "project_id", "corpus_id", "query"}
	var missing []string
	for _, field := range fields {
		if values[field] == "" {
			missing = append(missing, field)
		}
	}
	return missing
}

func response(status Status, detail ErrorDetail) Response {
	return Response{
		Status:  status,
		Results: []searchdocs.Result{},
		Error:   &detail,
	}
}

func statusCode(status Status) int {
	switch status {
	case StatusUnauthorized:
		return http.StatusUnauthorized
	case StatusInvalidRequest:
		return http.StatusBadRequest
	case StatusInternalError:
		return http.StatusInternalServerError
	case Status(searchdocs.StatusCorpusNotAdmitted):
		return http.StatusForbidden
	default:
		return http.StatusOK
	}
}

func writeJSON(w http.ResponseWriter, code int, response Response) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(response)
}

func (s Server) recordUnauthorized(ctx context.Context, request SearchDocsRequest, latency time.Duration) error {
	if !s.Governance.Enabled() {
		return nil
	}
	normalized := normalizeRequest(request)
	parametersHash, err := auditusage.HashParameters(map[string]any{
		"auth_status": "unauthorized",
		"project_id":  unknownProjectID,
		"corpus_id":   unknownCorpusID,
		"limit":       normalized.Limit,
	})
	if err != nil {
		return err
	}
	_, _, err = s.Governance.RecordMCPToolCall(ctx, auditusage.MCPToolCall{
		PrincipalID:    unauthenticatedPrincipal,
		TenantID:       normalized.TenantID,
		ProjectID:      unknownProjectID,
		CorpusID:       unknownCorpusID,
		ToolName:       searchdocs.ToolName,
		ToolVersion:    auditusage.ToolVersionP0,
		ParametersHash: parametersHash,
		Outcome:        auditusage.OutcomeDenied,
		Latency:        latency,
		ErrorClass:     string(StatusUnauthorized),
		Counts: map[string]int{
			"request_count":    1,
			"result_count":     0,
			"source_ref_count": 0,
			"denied_count":     1,
		},
	})
	return err
}

func governanceParametersHash(request SearchDocsRequest, response searchdocs.Response) string {
	hash, err := auditusage.HashParameters(map[string]any{
		"corpus_id":  request.CorpusID,
		"query_hash": response.Usage.QueryHash,
		"limit":      request.Limit,
	})
	if err != nil {
		return ""
	}
	return hash
}

func governanceOutcome(status Status) auditusage.Outcome {
	switch status {
	case Status(searchdocs.StatusCorpusNotAdmitted), StatusUnauthorized:
		return auditusage.OutcomeDenied
	case Status(searchdocs.StatusOK), Status(searchdocs.StatusNoEvidence):
		return auditusage.OutcomeSuccess
	default:
		return auditusage.OutcomeError
	}
}

func governanceErrorClass(response Response) string {
	if response.Error != nil {
		return response.Error.Class
	}
	if response.Status == Status(searchdocs.StatusCorpusNotAdmitted) {
		return string(searchdocs.StatusCorpusNotAdmitted)
	}
	return ""
}

func governanceCounts(response Response) map[string]int {
	denied := 0
	if governanceOutcome(response.Status) == auditusage.OutcomeDenied {
		denied = 1
	}
	return map[string]int{
		"request_count":    1,
		"result_count":     len(response.Results),
		"source_ref_count": len(sourceRefs(response.Results)),
		"denied_count":     denied,
	}
}

func sourceRefs(results []searchdocs.Result) []string {
	refs := make([]string, 0, len(results))
	for _, result := range results {
		if strings.TrimSpace(result.SourceRef) != "" {
			refs = append(refs, strings.TrimSpace(result.SourceRef))
		}
	}
	return refs
}
