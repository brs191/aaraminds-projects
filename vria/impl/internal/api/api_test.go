package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aaraminds/vria/internal/registry"
)

func doJSON(t *testing.T, srv *Server, method, path, actor string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if actor != "" {
		req.Header.Set("X-VRIA-Principal", actor)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func TestImportRequiresPrincipal(t *testing.T) {
	srv := NewServer(registry.NewMemStore())
	rec := doJSON(t, srv, http.MethodPost, "/api/v1/use-cases/import", "", importRequest{SourceID: "s"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rec.Code)
	}
}

func TestImportStagesWithoutPromoting(t *testing.T) {
	store := registry.NewMemStore()
	srv := NewServer(store)
	rows := []map[string]string{
		{"use_case_id": "UC-1", "name": "A", "tier": "Tool", "delivery_status": "PTB"},
		{"use_case_id": "UC-2", "name": "B", "tier": "Agent", "delivery_status": "PTB/PTO mixed"},
	}
	rec := doJSON(t, srv, http.MethodPost, "/api/v1/use-cases/import", "lead",
		importRequest{SourceID: "test", Rows: rows})
	if rec.Code != http.StatusCreated {
		t.Fatalf("code = %d body=%s", rec.Code, rec.Body.String())
	}
	var res registry.ImportResult
	json.Unmarshal(rec.Body.Bytes(), &res)
	if res.RecordsLoaded != 1 || res.RecordsRejected != 1 {
		t.Fatalf("loaded=%d rejected=%d, want 1/1", res.RecordsLoaded, res.RecordsRejected)
	}
	if res.AuditID == "" {
		t.Fatal("import must produce an audit record")
	}
	// Nothing in the active registry until explicit promotion (09 §3.1).
	if n := len(store.ListUseCases()); n != 0 {
		t.Fatalf("registry has %d records before promotion, want 0", n)
	}

	// Promote: only the clean record lands; the rejected one stays staged.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/import-batches/"+res.ImportBatchID+"/promote", "lead", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("promote code = %d body=%s", rec.Code, rec.Body.String())
	}
	if n := len(store.ListUseCases()); n != 1 {
		t.Fatalf("registry has %d records, want 1", n)
	}
	// Re-promotion is refused.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/import-batches/"+res.ImportBatchID+"/promote", "lead", nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("re-promote code = %d, want 409", rec.Code)
	}
}

func TestGetUseCaseAndErrorEnvelope(t *testing.T) {
	store := registry.NewMemStore()
	srv := NewServer(store)
	rec := doJSON(t, srv, http.MethodGet, "/api/v1/use-cases/UC-404", "viewer", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", rec.Code)
	}
	var env ErrorEnvelope
	json.Unmarshal(rec.Body.Bytes(), &env)
	if env.ErrorCode != "NOT_FOUND" || env.SafeState != "NoActionTaken" {
		t.Fatalf("bad envelope: %+v", env)
	}
}

// parseInventory extracts rows from the internal/99 markdown inventory table.
func parseInventory(t *testing.T, path string) []map[string]string {
	t.Helper()
	fh, err := os.Open(path)
	if err != nil {
		t.Skipf("internal inventory not present: %v", err)
	}
	defer fh.Close()
	var rows []map[string]string
	sc := bufio.NewScanner(fh)
	inTable := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "| SN |") {
			inTable = true
			continue
		}
		if inTable {
			if !strings.HasPrefix(line, "|") {
				break // end of the inventory table; ignore later tables
			}
			cells := strings.Split(strings.Trim(line, "|"), "|")
			if len(cells) < 5 || strings.HasPrefix(strings.TrimSpace(cells[0]), "-") {
				continue
			}
			rows = append(rows, map[string]string{
				"use_case_id":     "UC-" + strings.TrimSpace(cells[2]),
				"name":            strings.TrimSpace(cells[1]),
				"tier":            "", // tier assignment happens at Gate A triage
				"delivery_status": strings.TrimSpace(cells[3]),
			})
		}
	}
	return rows
}

// End-to-end: import the real source inventory. Done-when (prompts.md P1.2):
// 17 staged records with correct rejects.
func TestImportRealInventory(t *testing.T) {
	rows := parseInventory(t, "../../../internal/99_Source_AI_Use_Case_Inventory.md")
	if len(rows) != 17 {
		t.Fatalf("parsed %d inventory rows, want 17", len(rows))
	}
	store := registry.NewMemStore()
	srv := NewServer(store)
	rec := doJSON(t, srv, http.MethodPost, "/api/v1/use-cases/import", "portfolio-lead",
		importRequest{SourceID: "internal/99", Rows: rows})
	if rec.Code != http.StatusCreated {
		t.Fatalf("code = %d body=%s", rec.Code, rec.Body.String())
	}
	var res registry.ImportResult
	json.Unmarshal(rec.Body.Bytes(), &res)
	if res.RecordsLoaded+res.RecordsRejected != 17 {
		t.Fatalf("loaded %d + rejected %d != 17", res.RecordsLoaded, res.RecordsRejected)
	}
	// 9 rows carry mixed PTB/PTO status and must be rejected for human triage,
	// never guessed (rows 3-11 of the inventory).
	if res.RecordsRejected != 9 {
		t.Fatalf("rejected = %d, want 9 (mixed-status rows)", res.RecordsRejected)
	}
	// Promote the clean 8.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/import-batches/"+res.ImportBatchID+"/promote", "portfolio-lead", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("promote code = %d body=%s", rec.Code, rec.Body.String())
	}
	if n := len(store.ListUseCases()); n != 8 {
		t.Fatalf("registry has %d, want 8 clean records", n)
	}
	// Full audit trail exists: staging + promotion.
	if n := len(store.AuditTrail()); n < 2 {
		t.Fatalf("audit trail has %d events, want >= 2", n)
	}
}

// End-to-end over HTTP: draft → submit → approve+commit → read hypothesis.
func TestHypothesisWorkflowOverHTTP(t *testing.T) {
	srv := NewServer(registry.NewMemStore())
	rec := doJSON(t, srv, http.MethodPost, "/api/v1/use-cases/UC-9/draft-update", "owner",
		map[string]interface{}{
			"proposed_changes": map[string]interface{}{
				"value_owner": "owner", "expected_benefit": "faster triage",
				"business_objective": "MTTR", "benefit_type": "CycleTime",
				"primary_metric_id": "M-9", "baseline_value": 120.0, "target_value": 60.0,
			},
			"submit": true,
		})
	if rec.Code != http.StatusCreated {
		t.Fatalf("draft: %d %s", rec.Code, rec.Body.String())
	}
	var draft map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &draft)
	approvalID, _ := draft["approval_id"].(string)
	draftID, _ := draft["draft_id"].(string)
	if approvalID == "" || draftID == "" {
		t.Fatalf("missing ids: %v", draft)
	}
	// Approve (+commit) as a different principal.
	rec = doJSON(t, srv, http.MethodPost, "/api/v1/approvals/"+approvalID+"/decision", "portfolio-lead",
		map[string]interface{}{"decision": "approve", "draft_id": draftID})
	if rec.Code != http.StatusOK {
		t.Fatalf("decision: %d %s", rec.Code, rec.Body.String())
	}
	// Read back.
	rec = doJSON(t, srv, http.MethodGet, "/api/v1/use-cases/UC-9/hypothesis", "viewer", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get hypothesis: %d %s", rec.Code, rec.Body.String())
	}
	var got map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got["approval_state"] != "Approved" {
		t.Fatalf("approval_state = %v, want Approved", got["approval_state"])
	}
	if missing, ok := got["missing_required_fields"].([]interface{}); ok && len(missing) > 0 {
		t.Fatalf("unexpected gaps: %v", missing)
	}
}
