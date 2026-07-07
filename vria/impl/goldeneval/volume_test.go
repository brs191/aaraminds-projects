// Package goldeneval — volume evaluation dataset tests.
// Implements 07_VRIA_Golden_Eval_Set.md §3 release gate metrics
// against the labeled volume dataset (≥50 records).
//
// Gate thresholds:
//   value-state classification accuracy  ≥ 90%
//   schema validation pass rate          100% (all records unmarshal cleanly)
//   recommendation accuracy              ≥ 90% (within recommendation_one_of)
//   normalize accuracy                   ≥ 95%
package goldeneval

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/aaraminds/vria/internal/enums"
	"github.com/aaraminds/vria/internal/registry"
	"github.com/aaraminds/vria/internal/scoring"
)

// --- Dataset schema structs ---

// volumeInput mirrors scoring.Input for JSON unmarshaling.
// Pointer fields handle null JSON values.
type volumeInput struct {
	ValueOwner        string `json:"ValueOwner"`
	Sponsor           string `json:"Sponsor"`
	BusinessObjective string `json:"BusinessObjective"`
	BenefitType       string `json:"BenefitType"`
	Tier              string `json:"Tier"`
	PrimaryMetricID   string `json:"PrimaryMetricID"`

	BaselineValue         *float64 `json:"BaselineValue"`
	BaselinePeriodDefined bool     `json:"BaselinePeriodDefined"`
	BaselinePlanApproved  bool     `json:"BaselinePlanApproved"`

	CurrentValue  *float64 `json:"CurrentValue"`
	TargetValue   *float64 `json:"TargetValue"`
	LowerIsBetter bool     `json:"LowerIsBetter"`

	EvidenceAuthority   string `json:"EvidenceAuthority"`
	EvidenceFreshness   string `json:"EvidenceFreshness"`
	AllCitationsPresent bool   `json:"AllCitationsPresent"`
	HasEvidenceSource   bool   `json:"HasEvidenceSource"`
	OwnerAcceptedStale  bool   `json:"OwnerAcceptedStale"`

	Attribution                     string `json:"Attribution"`
	ConfoundersDocumented           int    `json:"ConfoundersDocumented"`
	MaterialConfoundersUndocumented bool   `json:"MaterialConfoundersUndocumented"`

	NetValue          string `json:"NetValue"`
	NetValueRationale string `json:"NetValueRationale"`

	Sustainment string `json:"Sustainment"`

	ApprovalBoundaryRecorded bool   `json:"ApprovalBoundaryRecorded"`
	PolicyIssueUnresolved    bool   `json:"PolicyIssueUnresolved"`
	ArtifactState            string `json:"ArtifactState"`

	HasValueClaim    bool `json:"HasValueClaim"`
	DeliveryComplete bool `json:"DeliveryComplete"`
}

type volumeExpected struct {
	ValueState          string   `json:"value_state"`
	RecommendationOneOf []string `json:"recommendation_one_of"`
	Confidence          string   `json:"confidence"`
	ScoreMin            int      `json:"score_min"`
	ScoreMax            int      `json:"score_max"`
}

type volumeRecord struct {
	RecordID   string         `json:"record_id"`
	LabelNotes string         `json:"label_notes"`
	Input      volumeInput    `json:"input"`
	Expected   volumeExpected `json:"expected"`
}

type volumeDataset struct {
	DatasetVersion     string         `json:"dataset_version"`
	ScoringRuleVersion string         `json:"scoring_rule_version"`
	Records            []volumeRecord `json:"records"`
}

// --- Normalize cases schema structs ---

type normalizeCase struct {
	CaseID             string `json:"case_id"`
	Raw                string `json:"raw"`
	ExpectedStatus     string `json:"expected_status"`
	ExpectedErrorClass string `json:"expected_error_class"`
	Notes              string `json:"notes"`
}

type normalizeCasesFile struct {
	DatasetVersion string          `json:"dataset_version"`
	Cases          []normalizeCase `json:"cases"`
}

// --- Helpers ---

func toScoringInput(v volumeInput) scoring.Input {
	return scoring.Input{
		ValueOwner:                      v.ValueOwner,
		Sponsor:                         v.Sponsor,
		BusinessObjective:               v.BusinessObjective,
		BenefitType:                     v.BenefitType,
		Tier:                            enums.UseCaseTier(v.Tier),
		PrimaryMetricID:                 v.PrimaryMetricID,
		BaselineValue:                   v.BaselineValue,
		BaselinePeriodDefined:           v.BaselinePeriodDefined,
		BaselinePlanApproved:            v.BaselinePlanApproved,
		CurrentValue:                    v.CurrentValue,
		TargetValue:                     v.TargetValue,
		LowerIsBetter:                   v.LowerIsBetter,
		EvidenceAuthority:               enums.Authority(v.EvidenceAuthority),
		EvidenceFreshness:               enums.Freshness(v.EvidenceFreshness),
		AllCitationsPresent:             v.AllCitationsPresent,
		HasEvidenceSource:               v.HasEvidenceSource,
		OwnerAcceptedStale:              v.OwnerAcceptedStale,
		Attribution:                     enums.AttributionMethod(v.Attribution),
		ConfoundersDocumented:           v.ConfoundersDocumented,
		MaterialConfoundersUndocumented: v.MaterialConfoundersUndocumented,
		NetValue:                        enums.NetValueCheck(v.NetValue),
		NetValueRationale:               v.NetValueRationale,
		Sustainment:                     enums.SustainmentStatus(v.Sustainment),
		ApprovalBoundaryRecorded:        v.ApprovalBoundaryRecorded,
		PolicyIssueUnresolved:           v.PolicyIssueUnresolved,
		ArtifactState:                   enums.ArtifactState(v.ArtifactState),
		HasValueClaim:                   v.HasValueClaim,
		DeliveryComplete:                v.DeliveryComplete,
	}
}

func containsStr(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// --- Main volume test ---

// TestVolume loads the volume dataset and normalize cases, runs each record
// through the engine, and checks all four §3 gates. It prints per-gate
// percentages via t.Logf and fails with a clear message if any gate is missed.
func TestVolume(t *testing.T) {
	// Load dataset.json
	dsBytes, err := ioutil.ReadFile("volume/dataset.json")
	if err != nil {
		t.Fatalf("cannot read volume/dataset.json: %v", err)
	}
	var ds volumeDataset
	if err := json.Unmarshal(dsBytes, &ds); err != nil {
		t.Fatalf("cannot parse volume/dataset.json: %v", err)
	}

	// Load normalize_cases.json
	ncBytes, err := ioutil.ReadFile("volume/normalize_cases.json")
	if err != nil {
		t.Fatalf("cannot read volume/normalize_cases.json: %v", err)
	}
	var ncFile normalizeCasesFile
	if err := json.Unmarshal(ncBytes, &ncFile); err != nil {
		t.Fatalf("cannot parse volume/normalize_cases.json: %v", err)
	}

	t.Logf("dataset_version=%s  scoring_rule_version=%s  records=%d",
		ds.DatasetVersion, ds.ScoringRuleVersion, len(ds.Records))

	// --- Gate 1: Schema validation pass rate = 100% ---
	// Every record successfully unmarshaled above; count records with
	// valid record_id (non-empty) as the schema check proxy.
	schemaTotal := len(ds.Records)
	schemaPassed := 0
	for _, rec := range ds.Records {
		if rec.RecordID != "" {
			schemaPassed++
		}
	}
	schemaRate := 100.0 * float64(schemaPassed) / float64(schemaTotal)
	t.Logf("GATE schema_validation_pass_rate: %.1f%% (%d/%d)", schemaRate, schemaPassed, schemaTotal)
	if schemaPassed != schemaTotal {
		t.Errorf("schema_validation_pass_rate gate FAILED: %.1f%% < 100%%", schemaRate)
	}

	// --- Gate 2: Value-state classification accuracy >= 90% ---
	stateTotal := len(ds.Records)
	statePassed := 0
	var stateFailIDs []string
	for _, rec := range ds.Records {
		in := toScoringInput(rec.Input)
		result := scoring.Score(in)
		got := string(result.ValueState)
		want := rec.Expected.ValueState
		if got == want {
			statePassed++
		} else {
			stateFailIDs = append(stateFailIDs, rec.RecordID)
			t.Logf("  state MISMATCH %s: got=%s want=%s notes=%s", rec.RecordID, got, want, rec.LabelNotes)
		}
	}
	stateRate := 100.0 * float64(statePassed) / float64(stateTotal)
	t.Logf("GATE value_state_classification_accuracy: %.1f%% (%d/%d)", stateRate, statePassed, stateTotal)
	if len(stateFailIDs) > 0 {
		t.Logf("  failing record ids: %s", strings.Join(stateFailIDs, ", "))
	}
	if stateRate < 90.0 {
		t.Errorf("value_state_classification_accuracy gate FAILED: %.1f%% < 90%%", stateRate)
	}

	// --- Gate 3: Recommendation accuracy >= 90% ---
	recTotal := len(ds.Records)
	recPassed := 0
	var recFailIDs []string
	for _, rec := range ds.Records {
		in := toScoringInput(rec.Input)
		result := scoring.Score(in)
		got := string(result.Recommendation)
		if containsStr(rec.Expected.RecommendationOneOf, got) {
			recPassed++
		} else {
			recFailIDs = append(recFailIDs, rec.RecordID)
			t.Logf("  rec MISMATCH %s: got=%s want_one_of=%v notes=%s",
				rec.RecordID, got, rec.Expected.RecommendationOneOf, rec.LabelNotes)
		}
	}
	recRate := 100.0 * float64(recPassed) / float64(recTotal)
	t.Logf("GATE recommendation_accuracy: %.1f%% (%d/%d)", recRate, recPassed, recTotal)
	if len(recFailIDs) > 0 {
		t.Logf("  failing record ids: %s", strings.Join(recFailIDs, ", "))
	}
	if recRate < 90.0 {
		t.Errorf("recommendation_accuracy gate FAILED: %.1f%% < 90%%", recRate)
	}

	// --- Gate 4: Normalize accuracy >= 95% ---
	normTotal := len(ncFile.Cases)
	normPassed := 0
	var normFailIDs []string
	for _, nc := range ncFile.Cases {
		status, normErr := registry.NormalizeDeliveryStatus(nc.Raw)
		gotClass := "ok"
		if normErr != nil {
			errStr := normErr.Error()
			if strings.Contains(errStr, "ambiguous") {
				gotClass = "ambiguous"
			} else {
				gotClass = "unmapped"
			}
		}
		classOK := gotClass == nc.ExpectedErrorClass
		statusOK := nc.ExpectedErrorClass != "ok" || string(status) == nc.ExpectedStatus
		if classOK && statusOK {
			normPassed++
		} else {
			normFailIDs = append(normFailIDs, nc.CaseID)
			t.Logf("  normalize MISMATCH %s: raw=%q gotClass=%s wantClass=%s gotStatus=%s wantStatus=%s",
				nc.CaseID, nc.Raw, gotClass, nc.ExpectedErrorClass, string(status), nc.ExpectedStatus)
		}
	}
	normRate := 100.0 * float64(normPassed) / float64(normTotal)
	t.Logf("GATE normalize_accuracy: %.1f%% (%d/%d)", normRate, normPassed, normTotal)
	if len(normFailIDs) > 0 {
		t.Logf("  failing case ids: %s", strings.Join(normFailIDs, ", "))
	}
	if normRate < 95.0 {
		t.Errorf("normalize_accuracy gate FAILED: %.1f%% < 95%%", normRate)
	}
}
