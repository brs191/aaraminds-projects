package ingestionruns

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGoldenPromotionCasesMatchDegenerateRunExpectations(t *testing.T) {
	t.Parallel()

	expectations := loadGoldenExpectations(t)
	promoted := 0
	for _, testCase := range expectations.PromotionCases {
		run := testCase.Run()
		decision := run.PromotionDecision()
		if decision.CanPromote != testCase.ExpectedCanPromote {
			t.Fatalf("%s: expected can_promote=%v, got %+v", testCase.CaseID, testCase.ExpectedCanPromote, decision)
		}
		if decision.Reason != Reason(testCase.ExpectedReason) {
			t.Fatalf("%s: expected reason %q, got %+v", testCase.CaseID, testCase.ExpectedReason, decision)
		}
		if decision.CanPromote {
			promoted++
			if decision.Err != nil {
				t.Fatalf("%s: promotable run should not have error: %v", testCase.CaseID, decision.Err)
			}
		} else if !IsNonPromotable(decision.Err) {
			t.Fatalf("%s: expected explicit non-promotable error, got %T %v", testCase.CaseID, decision.Err, decision.Err)
		}

		record := run.ToRecord()
		if record.Promoted != testCase.ExpectedCanPromote {
			t.Fatalf("%s: record promoted mismatch: %+v", testCase.CaseID, record)
		}
		if record.RunMetrics["promotion_reason"] != testCase.ExpectedReason {
			t.Fatalf("%s: record reason mismatch: %+v", testCase.CaseID, record.RunMetrics)
		}
		if !testCase.ExpectedCanPromote && record.RunMetrics["promotion_decision"] != PromotionDecisionDeny {
			t.Fatalf("%s: non-promotable record must deny promotion: %+v", testCase.CaseID, record.RunMetrics)
		}
	}
	if promoted != 1 {
		t.Fatalf("expected exactly one promotable golden case, got %d", promoted)
	}
}

func TestPromotionGuardRejectsFailedRunningAndCancelledRuns(t *testing.T) {
	t.Parallel()

	for _, status := range []Status{StatusFailed, StatusRunning, StatusCancelled} {
		run := healthyRun()
		run.Status = status
		decision := run.PromotionDecision()
		if decision.CanPromote {
			t.Fatalf("status %q should not promote", status)
		}
		if decision.Reason != ReasonRunNotCompleted {
			t.Fatalf("status %q expected %q, got %+v", status, ReasonRunNotCompleted, decision)
		}
	}
}

func TestPromotionGuardRejectsDegenerateCompletedRuns(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*Run)
		reason Reason
	}{
		{"no-documents", func(run *Run) { run.DocumentCount = 0 }, ReasonDegenerateNoDocuments},
		{"no-nodes", func(run *Run) { run.NodeCount = 0 }, ReasonDegenerateNoNodes},
		{"no-anchors", func(run *Run) { run.AnchorCount = 0 }, ReasonDegenerateNoAnchors},
		{"no-passages", func(run *Run) { run.PassageCount = 0 }, ReasonDegenerateNoPassages},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			run := healthyRun()
			testCase.mutate(&run)
			decision := run.PromotionDecision()
			if decision.CanPromote || decision.Reason != testCase.reason {
				t.Fatalf("expected blocked reason %q, got %+v", testCase.reason, decision)
			}
			if record := run.ToRecord(); record.Promoted {
				t.Fatalf("degenerate run must not produce promoted record: %+v", record)
			}
		})
	}
}

func TestInvalidRunReturnsStructuredValidationError(t *testing.T) {
	t.Parallel()

	run := healthyRun()
	run.RunID = ""
	run.DocumentCount = -1
	run.Status = Status("unknown")
	decision := run.PromotionDecision()
	if decision.CanPromote {
		t.Fatal("invalid run must not promote")
	}
	if decision.Reason != ReasonInvalidRun {
		t.Fatalf("expected invalid_run, got %+v", decision)
	}
	if !IsValidationError(decision.Err) {
		t.Fatalf("expected validation error, got %T %v", decision.Err, decision.Err)
	}
}

func healthyRun() Run {
	return Run{
		RunID:         "run-healthy",
		CorpusID:      "golden-admitted",
		SourceID:      "src-golden-admitted-local",
		Status:        StatusCompleted,
		DocumentCount: 4,
		NodeCount:     12,
		EdgeCount:     8,
		AnchorCount:   5,
		PassageCount:  5,
		CaveatCount:   0,
	}
}

type goldenExpectations struct {
	PromotionCases []goldenPromotionCase `json:"promotion_cases"`
}

type goldenPromotionCase struct {
	CaseID             string `json:"case_id"`
	RunID              string `json:"run_id"`
	CorpusID           string `json:"corpus_id"`
	SourceID           string `json:"source_id"`
	Status             string `json:"status"`
	DocumentCount      int    `json:"document_count"`
	NodeCount          int    `json:"node_count"`
	EdgeCount          int    `json:"edge_count"`
	AnchorCount        int    `json:"anchor_count"`
	PassageCount       int    `json:"passage_count"`
	CaveatCount        int    `json:"caveat_count"`
	ErrorMessage       string `json:"error_message"`
	ExpectedCanPromote bool   `json:"expected_can_promote"`
	ExpectedReason     string `json:"expected_reason"`
}

func (c goldenPromotionCase) Run() Run {
	return Run{
		RunID:         c.RunID,
		CorpusID:      c.CorpusID,
		SourceID:      c.SourceID,
		Status:        Status(c.Status),
		DocumentCount: c.DocumentCount,
		NodeCount:     c.NodeCount,
		EdgeCount:     c.EdgeCount,
		AnchorCount:   c.AnchorCount,
		PassageCount:  c.PassageCount,
		CaveatCount:   c.CaveatCount,
		ErrorMessage:  c.ErrorMessage,
	}
}

func loadGoldenExpectations(t *testing.T) goldenExpectations {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("..", "..", "..", "evaluation", "golden", "expected-degenerate-runs.json"))
	if err != nil {
		t.Fatalf("read expected degenerate runs: %v", err)
	}
	var expectations goldenExpectations
	if err := json.Unmarshal(content, &expectations); err != nil {
		t.Fatalf("parse expected degenerate runs: %v", err)
	}
	return expectations
}
