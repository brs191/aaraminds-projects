// Package assessment implements assessment generation (P4): it assembles a
// scoring.Input from the value hypothesis, registry context, and metric
// evidence, runs the deterministic scoring engine, and persists the result
// as an immutable ValueAssessment snapshot (contracts/17 §7).
//
// Assessments are append-only: regeneration always creates a new version
// with a new ID; an existing assessment is never mutated.
package assessment

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aaraminds/vria/internal/enums"
	"github.com/aaraminds/vria/internal/hypothesis"
	"github.com/aaraminds/vria/internal/scoring"
)

// ScoringRuleVersion pins every persisted assessment to the rule set that
// produced it (contracts/17 §7 scoring_rule_version, contracts/21 §4).
const ScoringRuleVersion = "v1.3"

var ErrNotFound = errors.New("not found")

// MetricSnapshot carries the operational metric values plus evidence
// metadata the scorer needs (contracts/17 §5–§6). Baseline and target
// normally come from the ValueHypothesis (field-ownership note in
// internal/hypothesis); provider values only fill gaps.
type MetricSnapshot struct {
	BaselineValue                   *float64
	CurrentValue                    *float64
	TargetValue                     *float64
	LowerIsBetter                   bool
	BaselinePeriodDefined           bool
	BaselinePlanApproved            bool
	EvidenceAuthority               enums.Authority
	EvidenceFreshness               enums.Freshness
	AllCitationsPresent             bool
	HasEvidenceSource               bool
	OwnerAcceptedStale              bool
	MaterialConfoundersUndocumented bool
	HasValueClaim                   bool
	EvidenceSourceIDs               []string
}

// MetricProvider is the boundary to metric/evidence storage (contracts/19
// metric_snapshots + evidence_sources). The in-memory implementation backs
// tests and local runs.
type MetricProvider interface {
	// Snapshot returns the latest snapshot for a use case's primary metric.
	// ok=false means no snapshot exists; scoring then sees missing values,
	// never inferred ones (GE-013).
	Snapshot(useCaseID, metricID string) (MetricSnapshot, bool)
	// SustainmentCheck returns the currently due post-Realized check.
	// ok=false means no snapshot arrived; the scheduler records that as a
	// failed check (contracts/20 §7).
	SustainmentCheck(useCaseID, metricID string) (scoring.SustainmentCheck, bool)
}

// HypothesisSource decouples this package from the hypothesis service;
// *hypothesis.Service satisfies it.
type HypothesisSource interface {
	Get(useCaseID string) (hypothesis.Hypothesis, []string, error)
}

// UseCaseContext carries registry fields the scorer needs that do not live
// on the hypothesis (contracts/17 §3).
type UseCaseContext struct {
	Sponsor                  string
	Tier                     enums.UseCaseTier
	DeliveryComplete         bool
	ApprovalBoundaryRecorded bool
	PolicyIssueUnresolved    bool
}

// UseCaseLookup resolves registry context; ok=false when the use case is
// not in the active registry.
type UseCaseLookup func(useCaseID string) (UseCaseContext, bool)

// AuditSink decouples this package from the audit store.
type AuditSink func(action, targetType, targetID, actorID string)

// Assessment is an immutable snapshot per contracts/17 §7 (fields persisted
// by this slice). Version increments per use case on every regeneration.
type Assessment struct {
	AssessmentID       string                  `json:"assessment_id"`
	UseCaseID          string                  `json:"use_case_id"`
	Version            int                     `json:"version"`
	ValueState         enums.ValueState        `json:"value_state"`
	RealizationScore   int                     `json:"realization_score"`
	PreCapScore        int                     `json:"pre_cap_score"`
	ScoreBreakdown     scoring.Breakdown       `json:"score_breakdown"`
	AppliedCaps        []string                `json:"applied_caps"`
	Recommendation     enums.Recommendation    `json:"recommendation"`
	Confidence         enums.Confidence        `json:"confidence"`
	SustainmentStatus  enums.SustainmentStatus `json:"sustainment_status"`
	MissingEvidence    []string                `json:"missing_evidence"`
	ApprovalState      enums.ArtifactState     `json:"approval_state"`
	ScoringRuleVersion string                  `json:"scoring_rule_version"`
	CreatedAt          time.Time               `json:"created_at"`
}

type Service struct {
	mu       sync.Mutex
	hyp      HypothesisSource
	provider MetricProvider
	lookup   UseCaseLookup
	audit    AuditSink
	now      func() time.Time
	byID     map[string]*Assessment
	byUC     map[string][]string
	checks   map[string][]scoring.SustainmentCheck
	ucOrder  []string
	seq      int
}

func NewService(hyp HypothesisSource, provider MetricProvider, lookup UseCaseLookup, audit AuditSink) *Service {
	if provider == nil {
		provider = NewMemProvider()
	}
	if lookup == nil {
		lookup = func(string) (UseCaseContext, bool) { return UseCaseContext{}, false }
	}
	if audit == nil {
		audit = func(string, string, string, string) {}
	}
	return &Service{
		hyp:      hyp,
		provider: provider,
		lookup:   lookup,
		audit:    audit,
		now:      func() time.Time { return time.Now().UTC() },
		byID:     map[string]*Assessment{},
		byUC:     map[string][]string{},
		checks:   map[string][]scoring.SustainmentCheck{},
	}
}

func (s *Service) nextID(p string) string {
	s.seq++
	return fmt.Sprintf("%s-%06d", p, s.seq)
}

// GenerateAssessment assembles the scoring input, runs scoring.Score, and
// persists a new immutable assessment in state Draft. Regeneration creates
// a new version; it never mutates a prior assessment.
func (s *Service) GenerateAssessment(useCaseID, actorID string) (Assessment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, _, herr := s.hyp.Get(useCaseID)
	ctx, ucOK := s.lookup(useCaseID)
	if herr != nil {
		if !ucOK {
			return Assessment{}, ErrNotFound
		}
		// Registry record without a hypothesis: score what exists (NotReady /
		// HypothesisOnly), never invent hypothesis fields.
		h = hypothesis.Hypothesis{
			UseCaseID:         useCaseID,
			AttributionMethod: enums.AttributionUnknown,
			NetValueCheck:     enums.NetUnknown,
			ApprovalState:     enums.ArtDraft,
		}
	}
	snap, ok := s.provider.Snapshot(useCaseID, h.PrimaryMetricID)
	if !ok {
		snap = MetricSnapshot{
			EvidenceAuthority: enums.AuthorityUnknown,
			EvidenceFreshness: enums.FreshnessUnknown,
		}
	}
	sus := scoring.EvaluateSustainment(s.checks[useCaseID])
	r := scoring.Score(buildInput(h, ctx, snap, sus))

	a := &Assessment{
		AssessmentID:       s.nextID("asm"),
		UseCaseID:          useCaseID,
		Version:            len(s.byUC[useCaseID]) + 1,
		ValueState:         r.ValueState,
		RealizationScore:   r.Score,
		PreCapScore:        r.PreCapScore,
		ScoreBreakdown:     r.Breakdown,
		AppliedCaps:        r.AppliedCaps,
		Recommendation:     r.Recommendation,
		Confidence:         r.Confidence,
		SustainmentStatus:  sus,
		MissingEvidence:    r.MissingEvidence,
		ApprovalState:      enums.ArtDraft,
		ScoringRuleVersion: ScoringRuleVersion,
		CreatedAt:          s.now(),
	}
	s.byID[a.AssessmentID] = a
	if len(s.byUC[useCaseID]) == 0 {
		s.ucOrder = append(s.ucOrder, useCaseID)
	}
	s.byUC[useCaseID] = append(s.byUC[useCaseID], a.AssessmentID)
	s.audit("assessment.generated", "ValueAssessment", a.AssessmentID, actorID)
	return *a, nil
}

// buildInput maps hypothesis + registry context + metric snapshot onto the
// scoring engine input (contracts/20 §2). Hypothesis owns baseline/target;
// provider values fill gaps only.
func buildInput(h hypothesis.Hypothesis, ctx UseCaseContext, snap MetricSnapshot, sus enums.SustainmentStatus) scoring.Input {
	baseline := h.BaselineValue
	if baseline == nil {
		baseline = snap.BaselineValue
	}
	target := h.TargetValue
	if target == nil {
		target = snap.TargetValue
	}
	return scoring.Input{
		ValueOwner:        h.ValueOwner,
		Sponsor:           ctx.Sponsor,
		BusinessObjective: h.BusinessObjective,
		BenefitType:       h.BenefitType,
		Tier:              ctx.Tier,
		PrimaryMetricID:   h.PrimaryMetricID,

		BaselineValue:         baseline,
		BaselinePeriodDefined: snap.BaselinePeriodDefined,
		BaselinePlanApproved:  snap.BaselinePlanApproved,

		CurrentValue:  snap.CurrentValue,
		TargetValue:   target,
		LowerIsBetter: snap.LowerIsBetter,

		EvidenceAuthority:   snap.EvidenceAuthority,
		EvidenceFreshness:   snap.EvidenceFreshness,
		AllCitationsPresent: snap.AllCitationsPresent,
		HasEvidenceSource:   snap.HasEvidenceSource,
		OwnerAcceptedStale:  snap.OwnerAcceptedStale,

		Attribution:                     h.AttributionMethod,
		ConfoundersDocumented:           len(h.KnownConfounders),
		MaterialConfoundersUndocumented: snap.MaterialConfoundersUndocumented,

		NetValue: h.NetValueCheck,

		Sustainment: sus,

		ApprovalBoundaryRecorded: ctx.ApprovalBoundaryRecorded,
		PolicyIssueUnresolved:    ctx.PolicyIssueUnresolved,
		ArtifactState:            h.ApprovalState,

		HasValueClaim:    snap.HasValueClaim,
		DeliveryComplete: ctx.DeliveryComplete,
	}
}

// Get returns one assessment by ID.
func (s *Service) Get(assessmentID string) (Assessment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.byID[assessmentID]
	if !ok {
		return Assessment{}, ErrNotFound
	}
	return *a, nil
}

// ListByUseCase returns all versions for a use case, oldest first.
func (s *Service) ListByUseCase(useCaseID string) []Assessment {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := s.byUC[useCaseID]
	out := make([]Assessment, 0, len(ids))
	for _, id := range ids {
		out = append(out, *s.byID[id])
	}
	return out
}

// Latest returns the newest assessment for a use case.
func (s *Service) Latest(useCaseID string) (Assessment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := s.byUC[useCaseID]
	if len(ids) == 0 {
		return Assessment{}, false
	}
	return *s.byID[ids[len(ids)-1]], true
}

// useCases returns the assessed use-case IDs in first-assessment order.
func (s *Service) useCases() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.ucOrder))
	copy(out, s.ucOrder)
	return out
}

// appendCheck records one sustainment check and returns the folded status
// per scoring.EvaluateSustainment. Check history is append-only.
func (s *Service) appendCheck(useCaseID string, c scoring.SustainmentCheck) enums.SustainmentStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks[useCaseID] = append(s.checks[useCaseID], c)
	return scoring.EvaluateSustainment(s.checks[useCaseID])
}

// CheckHistory returns a copy of the sustainment check history.
func (s *Service) CheckHistory(useCaseID string) []scoring.SustainmentCheck {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]scoring.SustainmentCheck, len(s.checks[useCaseID]))
	copy(out, s.checks[useCaseID])
	return out
}

// --- In-memory MetricProvider (tests and local runs) ---

type MemProvider struct {
	mu     sync.Mutex
	snaps  map[string]MetricSnapshot
	queues map[string][]scoring.SustainmentCheck
}

func NewMemProvider() *MemProvider {
	return &MemProvider{
		snaps:  map[string]MetricSnapshot{},
		queues: map[string][]scoring.SustainmentCheck{},
	}
}

func (p *MemProvider) SetSnapshot(useCaseID string, s MetricSnapshot) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.snaps[useCaseID] = s
}

// QueueCheck enqueues one sustainment check result; each scheduler pull
// consumes one. An empty queue reads as "no snapshot arrived".
func (p *MemProvider) QueueCheck(useCaseID string, c scoring.SustainmentCheck) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queues[useCaseID] = append(p.queues[useCaseID], c)
}

func (p *MemProvider) Snapshot(useCaseID, metricID string) (MetricSnapshot, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s, ok := p.snaps[useCaseID]
	return s, ok
}

func (p *MemProvider) SustainmentCheck(useCaseID, metricID string) (scoring.SustainmentCheck, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	q := p.queues[useCaseID]
	if len(q) == 0 {
		return scoring.SustainmentCheck{}, false
	}
	c := q[0]
	p.queues[useCaseID] = q[1:]
	return c, true
}
