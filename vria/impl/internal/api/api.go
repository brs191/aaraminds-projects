// Package api implements the registry slice of contracts/21 (§2 conventions,
// §2a endpoints) over net/http. Auth middleware is a boundary stub: real
// deployments terminate Entra ID OIDC at the gateway and pass the verified
// principal in X-VRIA-Principal (never trusted from external callers).
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/aaraminds/vria/internal/approval"
	"github.com/aaraminds/vria/internal/assessment"
	"github.com/aaraminds/vria/internal/enums"
	"github.com/aaraminds/vria/internal/hypothesis"
	"github.com/aaraminds/vria/internal/registry"
	"github.com/aaraminds/vria/internal/scorecard"
)

// ErrorEnvelope is the standard error shape (contracts/21 §3).
type ErrorEnvelope struct {
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
	SafeState string `json:"safe_state"`
	TraceID   string `json:"trace_id,omitempty"`
	Retryable bool   `json:"retryable"`
}

type Server struct {
	svc   *registry.Service
	hyp   *hypothesis.Service
	asmt  *assessment.Service
	sc    *scorecard.Service
	sched *assessment.Scheduler
	store registry.Store
	mux   *http.ServeMux
}

// Option customizes server construction without breaking existing callers.
type Option func(*serverConfig)

type serverConfig struct {
	provider assessment.MetricProvider
}

// WithMetricProvider swaps the metric/evidence source backing assessment
// generation and sustainment checks (default: empty in-memory provider).
func WithMetricProvider(p assessment.MetricProvider) Option {
	return func(c *serverConfig) { c.provider = p }
}

func NewServer(store registry.Store, opts ...Option) *Server {
	cfg := serverConfig{provider: assessment.NewMemProvider()}
	for _, o := range opts {
		o(&cfg)
	}
	audit := func(action, targetType, targetID, actorID string) {
		store.AppendAudit(registry.AuditEvent{
			ActorID: actorID, Action: action, TargetType: targetType, TargetID: targetID,
		})
	}
	hyp := hypothesis.NewService(audit)
	lookup := func(id string) (assessment.UseCaseContext, bool) {
		uc, err := store.GetUseCase(id)
		if err != nil {
			return assessment.UseCaseContext{}, false
		}
		return assessment.UseCaseContext{
			Sponsor: uc.Sponsor,
			Tier:    uc.Tier,
			DeliveryComplete: uc.DeliveryStatus == enums.DSPTOApproved ||
				uc.DeliveryStatus == enums.DSProduction,
			ApprovalBoundaryRecorded: uc.ApprovalState == enums.ArtApproved ||
				uc.ApprovalState == enums.ArtPublished,
		}, true
	}
	asmt := assessment.NewService(hyp, cfg.provider, lookup, audit)
	sc := scorecard.NewService(func(id string) (scorecard.AssessmentInfo, bool) {
		a, err := asmt.Get(id)
		if err != nil {
			return scorecard.AssessmentInfo{}, false
		}
		return scorecard.AssessmentInfo{AssessmentID: a.AssessmentID, MissingEvidence: a.MissingEvidence}, true
	}, audit)
	sched := assessment.NewScheduler(asmt, assessment.DefaultReportingWindow,
		func(useCaseID, valueOwner string, status enums.SustainmentStatus) {
			// Owner notification hook: audited here; a mail/Teams adapter
			// plugs in at deployment time.
			audit("sustainment.owner_notified", "UseCase", useCaseID, assessment.SchedulerActor)
		})
	s := &Server{
		svc: registry.NewService(store), hyp: hyp, asmt: asmt, sc: sc,
		sched: sched, store: store, mux: http.NewServeMux(),
	}
	s.mux.HandleFunc("/api/v1/use-cases", s.handleUseCases)
	s.mux.HandleFunc("/api/v1/use-cases/", s.handleUseCaseByID)
	s.mux.HandleFunc("/api/v1/use-cases/import", s.handleImport)
	s.mux.HandleFunc("/api/v1/import-batches/", s.handleBatch)
	s.mux.HandleFunc("/api/v1/approvals", s.handleSubmitApproval)
	s.mux.HandleFunc("/api/v1/approvals/", s.handleApprovalDecision)
	s.mux.HandleFunc("/api/v1/assessments/", s.handleAssessmentByID)
	s.mux.HandleFunc("/api/v1/scorecards", s.handleScorecards)
	s.mux.HandleFunc("/api/v1/scorecards/", s.handleScorecardAction)
	s.mux.HandleFunc("/api/v1/decision-log", s.handleDecisionLog)
	return s
}

// GET /api/v1/decision-log — read the append-only decision log (contracts/21
// §2a). Role-filtered at the gateway; read-only here.
func (s *Server) handleDecisionLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", r.Method, "NoActionTaken")
		return
	}
	records := s.sc.DecisionRecords()
	if target := r.URL.Query().Get("target_id"); target != "" {
		filtered := records[:0:0]
		for _, rec := range records {
			if rec.TargetID == target {
				filtered = append(filtered, rec)
			}
		}
		records = filtered
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"decision_records": records})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// RunSustainment drives the sustainment scheduler (P4.2). Deployments call
// this from a timer trigger; tests call it with a fixed clock.
func (s *Server) RunSustainment(now time.Time) []assessment.CheckOutcome {
	return s.sched.RunDue(now)
}

func principal(r *http.Request) string {
	if p := r.Header.Get("X-VRIA-Principal"); p != "" {
		return p
	}
	return ""
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, errCode, msg, safeState string) {
	writeJSON(w, code, ErrorEnvelope{ErrorCode: errCode, Message: msg, SafeState: safeState})
}

// GET /api/v1/use-cases
func (s *Server) handleUseCases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", r.Method, "NoActionTaken")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"use_cases": s.store.ListUseCases()})
}

// GET /api/v1/use-cases/{id}
// GET /api/v1/use-cases/{id}/hypothesis
// POST /api/v1/use-cases/{id}/draft-update
// POST /api/v1/use-cases/{id}/draft-update/submit
func (s *Server) handleUseCaseByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/use-cases/")
	if rest == "import" { // routed separately
		s.handleImport(w, r)
		return
	}
	parts := strings.Split(rest, "/")
	id := parts[0]
	if len(parts) >= 2 {
		switch parts[1] {
		case "hypothesis":
			s.handleGetHypothesis(w, r, id)
			return
		case "draft-update":
			s.handleDraftUpdate(w, r, id)
			return
		case "assessments":
			s.handleGenerateAssessment(w, r, id)
			return
		}
	}
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", r.Method, "NoActionTaken")
		return
	}
	uc, err := s.store.GetUseCase(id)
	if errors.Is(err, registry.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", id, "NoActionTaken")
		return
	}
	writeJSON(w, http.StatusOK, uc)
}

// GET /api/v1/use-cases/{id}/hypothesis — get_value_hypothesis (09 §3.3).
func (s *Server) handleGetHypothesis(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", r.Method, "NoActionTaken")
		return
	}
	h, missing, err := s.hyp.Get(id)
	if errors.Is(err, hypothesis.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", id, "NoActionTaken")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"value_hypothesis":        h,
		"version":                 h.RecordVersion,
		"approval_state":          h.ApprovalState,
		"missing_required_fields": missing,
	})
}

type draftUpdateRequest struct {
	ProposedChanges map[string]interface{} `json:"proposed_changes"`
	Submit          bool                   `json:"submit"` // also open the approval request
}

// POST /api/v1/use-cases/{id}/draft-update — draft_use_case_update (09 §3.4).
func (s *Server) handleDraftUpdate(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", r.Method, "NoActionTaken")
		return
	}
	actor := principal(r)
	if actor == "" {
		writeErr(w, http.StatusUnauthorized, "UNAUTHENTICATED", "missing principal", "NoActionTaken")
		return
	}
	var req draftUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), "NoActionTaken")
		return
	}
	d, err := s.hyp.CreateDraft(id, actor, req.ProposedChanges)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "DRAFT_FAILED", err.Error(), "NoActionTaken")
		return
	}
	resp := map[string]interface{}{
		"draft_id":             d.DraftID,
		"validation_status":    d.ValidationStatus,
		"validation_errors":    d.ValidationErrors,
		"requires_approval":    true,
		"approval_action_type": "RegistryUpdate",
	}
	if req.Submit && d.ValidationStatus == "Valid" {
		ar, err := s.hyp.SubmitForApproval(d.DraftID, actor)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "SUBMIT_FAILED", err.Error(), "Draft")
			return
		}
		resp["approval_id"] = ar.ApprovalID
		resp["approval_state"] = ar.State
	}
	writeJSON(w, http.StatusCreated, resp)
}

type decisionRequest struct {
	Decision string `json:"decision"` // approve | reject | request_changes
	Comments string `json:"comments"`
}

// decisionVerbs are the only transitions an approver may drive through this
// endpoint (contracts/21: "Approver only"). Requester-only verbs (resubmit,
// withdraw) must not be reachable here.
var decisionVerbs = map[string]bool{"approve": true, "reject": true, "request_changes": true}

// POST /api/v1/approvals/{id}/decision — approve_or_reject_draft (18 §4).
// decided_by is always the authenticated principal, never payload data.
func (s *Server) handleApprovalDecision(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/approvals/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "decision" || r.Method != http.MethodPost {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", r.URL.Path, "NoActionTaken")
		return
	}
	actor := principal(r)
	if actor == "" {
		writeErr(w, http.StatusUnauthorized, "UNAUTHENTICATED", "missing principal", "NoActionTaken")
		return
	}
	var req decisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), "NoActionTaken")
		return
	}
	if !decisionVerbs[req.Decision] {
		writeErr(w, http.StatusBadRequest, "INVALID_DECISION",
			"decision must be approve, reject, or request_changes", "NoActionTaken")
		return
	}
	ar, err := s.hyp.Decide(parts[0], req.Decision, actor)
	if errors.Is(err, hypothesis.ErrNotFound) {
		// Not a registry-update approval: try the scorecard lifecycle.
		ar, err = s.sc.Decide(parts[0], req.Decision, actor)
	}
	switch {
	case errors.Is(err, scorecard.ErrNotFound):
		writeErr(w, http.StatusNotFound, "NOT_FOUND", parts[0], "NoActionTaken")
		return
	case errors.Is(err, approval.ErrSelfApproval), errors.Is(err, approval.ErrNotDesignatedApprover):
		writeErr(w, http.StatusForbidden, "SEPARATION_OF_DUTIES", err.Error(), "NoActionTaken")
		return
	case err != nil:
		writeErr(w, http.StatusConflict, "INVALID_TRANSITION", err.Error(), "NoActionTaken")
		return
	}
	resp := map[string]interface{}{"approval_id": ar.ApprovalID, "approval_state": ar.State}
	// An approved RegistryUpdate commits its draft in the same action. The
	// commit target is the request's own TargetID — never trusted from the
	// payload — so a valid approval can always be committed (no wedge).
	if req.Decision == "approve" && ar.ActionType == "RegistryUpdate" {
		h, err := s.hyp.Commit(ar.TargetID, ar.ApprovalID, actor)
		if err != nil {
			writeErr(w, http.StatusConflict, "COMMIT_FAILED", err.Error(), "NoActionTaken")
			return
		}
		resp["value_hypothesis"] = h
	}
	writeJSON(w, http.StatusOK, resp)
}

type importRequest struct {
	SourceID string              `json:"source_id"`
	Rows     []map[string]string `json:"rows"`
}

// POST /api/v1/use-cases/import — stages only; promotion is separate and audited.
func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", r.Method, "NoActionTaken")
		return
	}
	actor := principal(r)
	if actor == "" {
		writeErr(w, http.StatusUnauthorized, "UNAUTHENTICATED", "missing principal", "NoActionTaken")
		return
	}
	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), "NoActionTaken")
		return
	}
	res, err := s.svc.Import(req.SourceID, actor, req.Rows)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "IMPORT_FAILED", err.Error(), "NoActionTaken")
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

// POST /api/v1/import-batches/{id}/promote
func (s *Server) handleBatch(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/import-batches/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "promote" || r.Method != http.MethodPost {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", r.URL.Path, "NoActionTaken")
		return
	}
	actor := principal(r)
	if actor == "" {
		writeErr(w, http.StatusUnauthorized, "UNAUTHENTICATED", "missing principal", "NoActionTaken")
		return
	}
	promoted, err := s.svc.Promote(parts[0], actor)
	switch {
	case errors.Is(err, registry.ErrNotFound):
		writeErr(w, http.StatusNotFound, "NOT_FOUND", parts[0], "NoActionTaken")
		return
	case errors.Is(err, registry.ErrAlreadyPromoted):
		writeErr(w, http.StatusConflict, "ALREADY_PROMOTED", parts[0], "NoActionTaken")
		return
	case errors.Is(err, registry.ErrDuplicateUseCase):
		writeErr(w, http.StatusConflict, "DUPLICATE_USE_CASE", err.Error(), "NoActionTaken")
		return
	case errors.Is(err, registry.ErrNothingToPromote):
		writeErr(w, http.StatusUnprocessableEntity, "NOTHING_TO_PROMOTE", err.Error(), "NoActionTaken")
		return
	case err != nil:
		writeErr(w, http.StatusInternalServerError, "PROMOTION_FAILED", err.Error(), "NoActionTaken")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"promoted": promoted})
}

// POST /api/v1/use-cases/{id}/assessments — generate_assessment (21 §2a).
// Produces a Draft assessment only; publication is a separate gated flow.
func (s *Server) handleGenerateAssessment(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", r.Method, "NoActionTaken")
		return
	}
	actor := principal(r)
	if actor == "" {
		writeErr(w, http.StatusUnauthorized, "UNAUTHENTICATED", "missing principal", "NoActionTaken")
		return
	}
	a, err := s.asmt.GenerateAssessment(id, actor)
	if errors.Is(err, assessment.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", id, "NoActionTaken")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "ASSESSMENT_FAILED", err.Error(), "NoActionTaken")
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// GET /api/v1/assessments/{id}
func (s *Server) handleAssessmentByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/assessments/")
	if r.Method != http.MethodGet || id == "" || strings.Contains(id, "/") {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", r.URL.Path, "NoActionTaken")
		return
	}
	a, err := s.asmt.Get(id)
	if errors.Is(err, assessment.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", id, "NoActionTaken")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

type scorecardRequest struct {
	Title         string           `json:"title"`
	Period        scorecard.Period `json:"period"`
	AssessmentIDs []string         `json:"assessment_ids"`
}

// POST /api/v1/scorecards — create draft scorecard (21 §2a; approval before publish).
func (s *Server) handleScorecards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", r.Method, "NoActionTaken")
		return
	}
	actor := principal(r)
	if actor == "" {
		writeErr(w, http.StatusUnauthorized, "UNAUTHENTICATED", "missing principal", "NoActionTaken")
		return
	}
	var req scorecardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), "NoActionTaken")
		return
	}
	c, err := s.sc.CreateDraft(req.Title, req.Period, req.AssessmentIDs, actor)
	if errors.Is(err, scorecard.ErrUnknownAssessment) {
		writeErr(w, http.StatusBadRequest, "UNKNOWN_ASSESSMENT", err.Error(), "NoActionTaken")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "SCORECARD_FAILED", err.Error(), "NoActionTaken")
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

type submitApprovalRequest struct {
	ActionType string `json:"action_type"`
	TargetID   string `json:"target_id"`
	Rationale  string `json:"rationale"`
}

// POST /api/v1/approvals — submit_for_approval (18 §4, 21 §2a). Creates a
// request only; nothing executes until an approver decides.
func (s *Server) handleSubmitApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", r.Method, "NoActionTaken")
		return
	}
	actor := principal(r)
	if actor == "" {
		writeErr(w, http.StatusUnauthorized, "UNAUTHENTICATED", "missing principal", "NoActionTaken")
		return
	}
	var req submitApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), "NoActionTaken")
		return
	}
	ar, err := s.sc.SubmitForApproval(req.TargetID, req.ActionType, actor)
	switch {
	case errors.Is(err, scorecard.ErrNotFound):
		writeErr(w, http.StatusNotFound, "NOT_FOUND", req.TargetID, "NoActionTaken")
		return
	case errors.Is(err, scorecard.ErrBadActionType):
		writeErr(w, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), "NoActionTaken")
		return
	case err != nil:
		writeErr(w, http.StatusInternalServerError, "SUBMIT_FAILED", err.Error(), "NoActionTaken")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"approval_id": ar.ApprovalID, "approval_state": ar.State, "action_type": ar.ActionType,
	})
}

type publishRequest struct {
	ApprovalID string `json:"approval_id"`
}

type supersedeRequest struct {
	ReplacementScorecardID string `json:"replacement_scorecard_id"`
	ApprovalID             string `json:"approval_id"`
}

// POST /api/v1/scorecards/{id}/publish   — GE-007 gated
// POST /api/v1/scorecards/{id}/supersede — approval-gated, links replacement
func (s *Server) handleScorecardAction(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/scorecards/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || r.Method != http.MethodPost {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", r.URL.Path, "NoActionTaken")
		return
	}
	actor := principal(r)
	if actor == "" {
		writeErr(w, http.StatusUnauthorized, "UNAUTHENTICATED", "missing principal", "NoActionTaken")
		return
	}
	id := parts[0]
	switch parts[1] {
	case "publish":
		var req publishRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), "NoActionTaken")
			return
		}
		c, err := s.sc.Publish(id, req.ApprovalID, actor)
		if !s.writeScorecardErr(w, id, err) {
			writeJSON(w, http.StatusOK, c)
		}
	case "supersede":
		var req supersedeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "INVALID_PAYLOAD", err.Error(), "NoActionTaken")
			return
		}
		c, err := s.sc.Supersede(id, req.ReplacementScorecardID, req.ApprovalID, actor)
		if !s.writeScorecardErr(w, id, err) {
			writeJSON(w, http.StatusOK, c)
		}
	default:
		writeErr(w, http.StatusNotFound, "NOT_FOUND", r.URL.Path, "NoActionTaken")
	}
}

// writeScorecardErr maps scorecard lifecycle errors onto the standard
// envelope. GE-007: a missing or undecided approval yields 409
// APPROVAL_REQUIRED and no state change.
func (s *Server) writeScorecardErr(w http.ResponseWriter, id string, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, scorecard.ErrApprovalRequired):
		writeErr(w, http.StatusConflict, "APPROVAL_REQUIRED", "action requires an Approved approval request", "NoActionTaken")
	case errors.Is(err, scorecard.ErrNotFound):
		writeErr(w, http.StatusNotFound, "NOT_FOUND", id, "NoActionTaken")
	default:
		writeErr(w, http.StatusConflict, "INVALID_TRANSITION", err.Error(), "NoActionTaken")
	}
	return true
}
