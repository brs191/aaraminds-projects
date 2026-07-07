package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aaraminds/vria/internal/assessment"
	"github.com/aaraminds/vria/internal/enums"
	"github.com/aaraminds/vria/internal/registry"
)

func f64(v float64) *float64 { return &v }

// seedHypothesis drives the existing HTTP workflow (draft → submit →
// approve+commit) so UC-20 has an Approved hypothesis.
func seedHypothesis(t *testing.T, srv *Server) {
	t.Helper()
	rec := doJSON(t, srv, http.MethodPost, "/api/v1/use-cases/UC-20/draft-update", "owner",
		map[string]interface{}{
			"proposed_changes": map[string]interface{}{
				"value_owner": "owner", "expected_benefit": "faster triage",
				"business_objective": "reduce MTTR", "benefit_type": "CycleTime",
				"primary_metric_id": "M-20", "baseline_value": 120.0, "target_value": 60.0,
				"attribution_method": "DirectMeasurement", "net_value_check": "Positive",
			},
			"submit": true,
		})
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed draft: %d %s", rec.Code, rec.Body.String())
	}
	var draft map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &draft)
	approvalID, _ := draft["approval_id"].(string)
	draftID, _ := draft["draft_id"].(string)
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/approvals/"+approvalID+"/decision", "portfolio-lead",
		map[string]interface{}{"decision": "approve", "draft_id": draftID})
	if rec.Code != http.StatusOK {
		t.Fatalf("seed approve: %d %s", rec.Code, rec.Body.String())
	}
}

// End-to-end over HTTP: generate assessment → create scorecard → publish
// without approval fails (GE-007) → submit+approve → publish succeeds →
// supersede works, with decision-log and audit effects verified.
func TestScorecardLifecycleOverHTTP(t *testing.T) {
	store := registry.NewMemStore()
	prov := assessment.NewMemProvider()
	srv := NewServer(store, WithMetricProvider(prov))
	seedHypothesis(t, srv)
	prov.SetSnapshot("UC-20", assessment.MetricSnapshot{
		CurrentValue:          f64(60),
		LowerIsBetter:         true,
		BaselinePeriodDefined: true,
		EvidenceAuthority:     enums.Authoritative,
		EvidenceFreshness:     enums.Fresh,
		AllCitationsPresent:   true,
		HasEvidenceSource:     true,
		HasValueClaim:         true,
	})

	// Assessment generation requires a principal.
	rec := doJSON(t, srv, http.MethodPost, "/api/v1/use-cases/UC-20/assessments", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("assessment without principal: %d, want 401", rec.Code)
	}
	// Unknown use case is a 404, never a fabricated assessment.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/use-cases/UC-404/assessments", "lead", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("assessment for unknown use case: %d, want 404", rec.Code)
	}
	// Generate the draft assessment.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/use-cases/UC-20/assessments", "lead", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("generate assessment: %d %s", rec.Code, rec.Body.String())
	}
	var asm map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &asm)
	assessmentID, _ := asm["assessment_id"].(string)
	if assessmentID == "" || asm["approval_state"] != "Draft" {
		t.Fatalf("assessment = %v", asm)
	}
	if asm["value_state"] != "Realized" || asm["scoring_rule_version"] != assessment.ScoringRuleVersion {
		t.Fatalf("value_state=%v rule_version=%v", asm["value_state"], asm["scoring_rule_version"])
	}
	// Read it back.
	rec = doJSON(t, srv, http.MethodGet, "/api/v1/assessments/"+assessmentID, "viewer", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get assessment: %d %s", rec.Code, rec.Body.String())
	}

	// Create the draft scorecard.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/scorecards", "lead", map[string]interface{}{
		"title":          "Q3 Value Scorecard",
		"period":         map[string]string{"start": "2026-07-01", "end": "2026-09-30"},
		"assessment_ids": []string{assessmentID},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create scorecard: %d %s", rec.Code, rec.Body.String())
	}
	var card map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &card)
	scorecardID, _ := card["scorecard_id"].(string)
	if scorecardID == "" || card["artifact_state"] != "Draft" {
		t.Fatalf("scorecard = %v", card)
	}
	cov, _ := card["evidence_coverage_summary"].(map[string]interface{})
	if cov["assessments_total"] != 1.0 || cov["with_citations"] != 1.0 || cov["with_gaps"] != 0.0 {
		t.Fatalf("coverage = %v", cov)
	}

	// GE-007: publish without an Approved request → 409 APPROVAL_REQUIRED,
	// and NO publication happens.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/scorecards/"+scorecardID+"/publish", "lead",
		map[string]interface{}{})
	if rec.Code != http.StatusConflict {
		t.Fatalf("ungated publish: %d, want 409", rec.Code)
	}
	var env ErrorEnvelope
	json.Unmarshal(rec.Body.Bytes(), &env)
	if env.ErrorCode != "APPROVAL_REQUIRED" || env.SafeState != "NoActionTaken" {
		t.Fatalf("envelope = %+v", env)
	}
	if c, _ := srv.sc.Get(scorecardID); c.ArtifactState != enums.ArtDraft || c.PublishedAt != nil {
		t.Fatalf("scorecard changed despite blocked publication: %+v", c)
	}
	if n := len(srv.sc.DecisionRecords()); n != 0 {
		t.Fatalf("decision log = %d records after blocked publish, want 0", n)
	}

	// Submit the publication approval.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/approvals", "lead", map[string]interface{}{
		"action_type": "ScorecardPublication", "target_id": scorecardID,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("submit approval: %d %s", rec.Code, rec.Body.String())
	}
	var sub map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &sub)
	pubApprovalID, _ := sub["approval_id"].(string)
	if pubApprovalID == "" || sub["approval_state"] != "Submitted" {
		t.Fatalf("submit = %v", sub)
	}
	// A pending (undecided) request still blocks publication.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/scorecards/"+scorecardID+"/publish", "lead",
		map[string]interface{}{"approval_id": pubApprovalID})
	if rec.Code != http.StatusConflict {
		t.Fatalf("publish with pending approval: %d, want 409", rec.Code)
	}

	// Approve as a different principal; decided_by comes from the header.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/approvals/"+pubApprovalID+"/decision", "sponsor",
		map[string]interface{}{"decision": "approve"})
	if rec.Code != http.StatusOK {
		t.Fatalf("approve publication: %d %s", rec.Code, rec.Body.String())
	}

	// Publish succeeds.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/scorecards/"+scorecardID+"/publish", "lead",
		map[string]interface{}{"approval_id": pubApprovalID})
	if rec.Code != http.StatusOK {
		t.Fatalf("gated publish: %d %s", rec.Code, rec.Body.String())
	}
	var published map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &published)
	if published["artifact_state"] != "Published" || published["published_at"] == nil {
		t.Fatalf("published = %v", published)
	}
	if p, _ := published["decision_log_pointer"].(string); p == "" {
		t.Fatal("published scorecard must carry a decision log pointer")
	}

	// Supersede: replacement draft, gated by its own approval on the old card.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/scorecards", "lead", map[string]interface{}{
		"title":          "Q3 Value Scorecard v2",
		"period":         map[string]string{"start": "2026-07-01", "end": "2026-09-30"},
		"assessment_ids": []string{assessmentID},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create replacement: %d %s", rec.Code, rec.Body.String())
	}
	var repl map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &repl)
	replacementID, _ := repl["scorecard_id"].(string)

	// Without approval the supersede gate holds.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/scorecards/"+scorecardID+"/supersede", "lead",
		map[string]interface{}{"replacement_scorecard_id": replacementID})
	if rec.Code != http.StatusConflict {
		t.Fatalf("ungated supersede: %d, want 409", rec.Code)
	}
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/approvals", "lead", map[string]interface{}{
		"action_type": "ScorecardSupersession", "target_id": scorecardID,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("submit supersession: %d %s", rec.Code, rec.Body.String())
	}
	json.Unmarshal(rec.Body.Bytes(), &sub)
	supApprovalID, _ := sub["approval_id"].(string)
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/approvals/"+supApprovalID+"/decision", "portfolio-lead",
		map[string]interface{}{"decision": "approve"})
	if rec.Code != http.StatusOK {
		t.Fatalf("approve supersession: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/scorecards/"+scorecardID+"/supersede", "lead",
		map[string]interface{}{"replacement_scorecard_id": replacementID, "approval_id": supApprovalID})
	if rec.Code != http.StatusOK {
		t.Fatalf("gated supersede: %d %s", rec.Code, rec.Body.String())
	}
	var superseded map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &superseded)
	if superseded["supersedes_scorecard_id"] != scorecardID {
		t.Fatalf("replacement link = %v, want %s", superseded["supersedes_scorecard_id"], scorecardID)
	}
	if c, _ := srv.sc.Get(scorecardID); c.ArtifactState != enums.ArtSuperseded || c.PublishedAt == nil {
		t.Fatalf("old scorecard = %+v, want Superseded with original published_at intact", c)
	}

	// Decision log: exactly publish + supersede, decided_by from principals.
	recs := srv.sc.DecisionRecords()
	if len(recs) != 2 {
		t.Fatalf("decision log = %d records, want 2", len(recs))
	}
	if recs[0].DecisionType != "ScorecardPublication" || recs[0].DecidedBy != "sponsor" {
		t.Fatalf("publish record = %+v", recs[0])
	}
	if recs[1].DecisionType != "ScorecardSupersession" || recs[1].DecidedBy != "portfolio-lead" {
		t.Fatalf("supersede record = %+v", recs[1])
	}

	// Audit trail carries the full lifecycle.
	want := map[string]bool{
		"assessment.generated": true, "scorecard.generated": true,
		"approval.submitted": true, "approval.decided": true,
		"scorecard.published": true, "scorecard.superseded": true,
	}
	for _, e := range store.AuditTrail() {
		delete(want, e.Action)
	}
	if len(want) != 0 {
		t.Fatalf("audit trail missing actions: %v", want)
	}
}
