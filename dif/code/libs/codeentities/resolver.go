package codeentities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/aaraminds/dif/libs/rifcompat"
)

// P1-02 resolver: resolves unresolved code-entity candidates through the RIF
// compatibility layer (ADR-016) and creates DESCRIBES edges only when
// resolver evidence is sufficient.
//
// Guardrails:
//   - Populated rif_meta shadows are never assumed; the resolver consumes a
//     rifcompat.Report, which already encodes the AGE fallback when shadows
//     are empty (ADR-016 §7).
//   - RIF-owned schemas are never mutated; edges are written to
//     dif_meta.edges only.
//   - Non-compatible RIF statuses mark candidates rif_unavailable explicitly;
//     there is no success-shaped empty result.

const (
	CaveatFuzzyMatch      = "fuzzy_match"
	CaveatAmbiguousMatch  = "ambiguous_match"
	CaveatRIFStatusPrefix = "rif_status:"

	// ExternalSystemRIF is the only external system DESCRIBES edges target.
	ExternalSystemRIF = "rif"

	// EdgeKindDescribes mirrors dif_meta.edges.edge_kind = 'DESCRIBES'.
	EdgeKindDescribes = "DESCRIBES"
)

// DescribesEdge is a doc->code edge backed by resolver evidence (ADR-016 §9).
type DescribesEdge struct {
	EdgeID            string
	CorpusID          string
	DocumentID        string
	DocumentVersionID string
	FromNodeID        string
	ToRIFNodeID       string
	RepoID            string
	CandidateID       string
	AnchorID          string
	SourceRef         string
	CodeSourceRef     string
	MatchMode         MatchMode
	Confidence        Confidence
	Caveats           []string
}

// ResolvedCandidate pairs a candidate's post-resolution state with the
// resolver evidence that produced it.
type ResolvedCandidate struct {
	Candidate Candidate
	Matches   []rifcompat.Entity
}

// ResolutionMetrics is the measured per-corpus resolution outcome. Values are
// counted from actual resolver results, never estimated.
type ResolutionMetrics struct {
	CorpusID       string
	Total          int
	Resolved       int
	Ambiguous      int
	Unresolved     int
	RIFUnavailable int
}

// ResolutionRate returns resolved/total for the corpus, or 0 for no candidates.
func (m ResolutionMetrics) ResolutionRate() float64 {
	if m.Total == 0 {
		return 0
	}
	return float64(m.Resolved) / float64(m.Total)
}

// ResolutionOutcome is the full result of one resolver pass.
type ResolutionOutcome struct {
	RIFStatus  rifcompat.Status
	Candidates []ResolvedCandidate
	Edges      []DescribesEdge
	Metrics    []ResolutionMetrics
}

// EdgeStore persists DESCRIBES edges.
type EdgeStore interface {
	WriteDescribesEdges(context.Context, []DescribesEdge) error
}

// MemoryEdgeStore records DESCRIBES edges in memory for tests and harnesses.
type MemoryEdgeStore struct {
	Edges []DescribesEdge
}

// SQLEdgeStore writes DESCRIBES edges to dif_meta.edges. It never mutates
// RIF-owned schemas.
type SQLEdgeStore struct {
	Execer Execer
}

// Resolve resolves candidates against a RIF compatibility report and returns
// updated candidates, DESCRIBES edges for sufficiently evidenced matches,
// and measured per-corpus resolution metrics.
func Resolve(report rifcompat.Report, candidates []Candidate) (ResolutionOutcome, error) {
	outcome := ResolutionOutcome{RIFStatus: report.Status}
	metrics := map[string]*ResolutionMetrics{}

	for _, candidate := range candidates {
		if err := candidate.Validate(); err != nil {
			return ResolutionOutcome{}, fmt.Errorf("candidate %q: %w", candidate.CandidateID, err)
		}
		if candidate.MatchStatus != StatusUnresolved {
			return ResolutionOutcome{}, fmt.Errorf("candidate %q: resolver input must be unresolved, got %q", candidate.CandidateID, candidate.MatchStatus)
		}
		corpus := metricsFor(metrics, candidate.CorpusID)
		corpus.Total++

		if report.Status != rifcompat.StatusCompatible {
			candidate.MatchStatus = StatusRIFUnavailable
			candidate.Caveats = sortedStrings(append(candidate.Caveats, CaveatRIFStatusPrefix+string(report.Status)))
			corpus.RIFUnavailable++
			outcome.Candidates = append(outcome.Candidates, ResolvedCandidate{Candidate: candidate, Matches: []rifcompat.Entity{}})
			continue
		}

		resolved := resolveOne(report, candidate)
		switch resolved.Candidate.MatchStatus {
		case StatusResolved:
			corpus.Resolved++
			edge, err := NewDescribesEdge(resolved.Candidate, resolved.Matches[0])
			if err != nil {
				return ResolutionOutcome{}, err
			}
			outcome.Edges = append(outcome.Edges, edge)
		case StatusAmbiguous:
			corpus.Ambiguous++
		default:
			corpus.Unresolved++
		}
		outcome.Candidates = append(outcome.Candidates, resolved)
	}

	outcome.Metrics = sortedMetrics(metrics)
	sort.SliceStable(outcome.Edges, func(i, j int) bool {
		return outcome.Edges[i].EdgeID < outcome.Edges[j].EdgeID
	})
	return outcome, nil
}

// NewDescribesEdge builds a DESCRIBES edge from a resolved candidate and the
// matched RIF entity. It refuses to build edges without resolver evidence.
func NewDescribesEdge(candidate Candidate, entity rifcompat.Entity) (DescribesEdge, error) {
	if candidate.MatchStatus != StatusResolved {
		return DescribesEdge{}, fmt.Errorf("DESCRIBES edge requires a resolved candidate, got %q", candidate.MatchStatus)
	}
	if strings.TrimSpace(candidate.ResolvedRIFNodeID) == "" || candidate.ResolvedRIFNodeID != entity.NodeID {
		return DescribesEdge{}, errors.New("DESCRIBES edge requires matching resolver evidence (candidate.resolved_rif_node_id == entity.node_id)")
	}
	edge := DescribesEdge{
		EdgeID:            rifcompat.EdgeID(candidate.NodeID, EdgeKindDescribes, entity.NodeID),
		CorpusID:          candidate.CorpusID,
		DocumentID:        candidate.DocumentID,
		DocumentVersionID: candidate.DocumentVersionID,
		FromNodeID:        candidate.NodeID,
		ToRIFNodeID:       entity.NodeID,
		RepoID:            entity.RepoID,
		CandidateID:       candidate.CandidateID,
		AnchorID:          candidate.AnchorID,
		SourceRef:         candidate.SourceRef,
		CodeSourceRef:     entity.SourceRef,
		MatchMode:         candidate.MatchMode,
		Confidence:        candidate.Confidence,
		Caveats:           sortedStrings(candidate.Caveats),
	}
	if err := edge.Validate(); err != nil {
		return DescribesEdge{}, err
	}
	return edge, nil
}

// Validate enforces the DESCRIBES edge shape before persistence.
func (e DescribesEdge) Validate() error {
	required := []struct {
		name  string
		value string
	}{
		{"edge_id", e.EdgeID},
		{"corpus_id", e.CorpusID},
		{"document_id", e.DocumentID},
		{"document_version_id", e.DocumentVersionID},
		{"from_node_id", e.FromNodeID},
		{"to_rif_node_id", e.ToRIFNodeID},
		{"repo_id", e.RepoID},
		{"candidate_id", e.CandidateID},
		{"anchor_id", e.AnchorID},
		{"source_ref", e.SourceRef},
		{"code_source_ref", e.CodeSourceRef},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("DESCRIBES edge %s is required", field.name)
		}
	}
	if e.EdgeID != rifcompat.EdgeID(e.FromNodeID, EdgeKindDescribes, e.ToRIFNodeID) {
		return errors.New("DESCRIBES edge_id must use the shared RIF/DIF edge ID algorithm")
	}
	if !allowedMode(e.MatchMode) {
		return fmt.Errorf("unsupported DESCRIBES match_mode %q", e.MatchMode)
	}
	if !allowedConfidence(e.Confidence) {
		return fmt.Errorf("unsupported DESCRIBES confidence %q", e.Confidence)
	}
	return nil
}

// WriteDescribesEdges records validated edges in memory.
func (s *MemoryEdgeStore) WriteDescribesEdges(_ context.Context, edges []DescribesEdge) error {
	for _, edge := range edges {
		if err := edge.Validate(); err != nil {
			return err
		}
	}
	s.Edges = append(s.Edges, edges...)
	return nil
}

// WriteDescribesEdges upserts validated DESCRIBES edges into dif_meta.edges.
func (s SQLEdgeStore) WriteDescribesEdges(ctx context.Context, edges []DescribesEdge) error {
	if s.Execer == nil {
		return errors.New("codeentities SQL edge store requires an execer")
	}
	for _, edge := range edges {
		if err := edge.Validate(); err != nil {
			return err
		}
		caveats, err := marshalCaveats(edge.Caveats)
		if err != nil {
			return err
		}
		_, err = s.Execer.ExecContext(ctx, `
INSERT INTO dif_meta.edges (
    edge_id, corpus_id, document_version_id, edge_kind, from_node_id,
    to_node_id, to_external_node_id, external_system, repo_id, candidate_id,
    match_mode, code_source_ref, confidence, anchor_id, caveats
) VALUES ($1, $2, $3, 'DESCRIBES', $4, NULL, $5, 'rif', $6, $7, $8, $9, $10, $11, $12::jsonb)
ON CONFLICT (edge_id) DO UPDATE SET
    match_mode = EXCLUDED.match_mode,
    code_source_ref = EXCLUDED.code_source_ref,
    confidence = EXCLUDED.confidence,
    caveats = EXCLUDED.caveats`,
			edge.EdgeID,
			edge.CorpusID,
			edge.DocumentVersionID,
			edge.FromNodeID,
			edge.ToRIFNodeID,
			edge.RepoID,
			edge.CandidateID,
			string(edge.MatchMode),
			edge.CodeSourceRef,
			string(edge.Confidence),
			edge.AnchorID,
			caveats,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateResolutions persists resolver outcomes onto existing candidate rows.
// It only updates resolver-owned fields and never inserts new candidates.
func (s SQLStore) UpdateResolutions(ctx context.Context, resolved []ResolvedCandidate) error {
	if s.Execer == nil {
		return errors.New("codeentities SQL store requires an execer")
	}
	for _, item := range resolved {
		candidate := item.Candidate
		if err := candidate.Validate(); err != nil {
			return err
		}
		caveats, err := marshalCaveats(candidate.Caveats)
		if err != nil {
			return err
		}
		_, err = s.Execer.ExecContext(ctx, `
UPDATE dif_meta.code_entity_candidates SET
    match_status = $2,
    resolved_rif_node_id = $3,
    match_mode = $4,
    confidence = $5,
    caveats = $6::jsonb,
    resolved_at = CASE WHEN $2 = 'resolved' THEN now() ELSE resolved_at END
WHERE candidate_id = $1`,
			candidate.CandidateID,
			string(candidate.MatchStatus),
			emptyToNil(candidate.ResolvedRIFNodeID),
			string(candidate.MatchMode),
			string(candidate.Confidence),
			caveats,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func resolveOne(report rifcompat.Report, candidate Candidate) ResolvedCandidate {
	lookup, caveats := lookupForCandidate(report, candidate)
	caveats = append(caveats, lookup.Caveats...)

	switch {
	case lookup.Status != rifcompat.LookupResolved || len(lookup.Matches) == 0:
		candidate.MatchStatus = StatusUnresolved
		candidate.Caveats = sortedStrings(append(candidate.Caveats, caveats...))
		return ResolvedCandidate{Candidate: candidate, Matches: []rifcompat.Entity{}}
	case len(lookup.Matches) > 1:
		candidate.MatchStatus = StatusAmbiguous
		candidate.Caveats = sortedStrings(append(candidate.Caveats, append(caveats, CaveatAmbiguousMatch)...))
		return ResolvedCandidate{Candidate: candidate, Matches: lookup.Matches}
	default:
		match := lookup.Matches[0]
		candidate.MatchStatus = StatusResolved
		candidate.ResolvedRIFNodeID = match.NodeID
		candidate.Confidence = confidenceFrom(lookup.Confidence, candidate.MatchMode)
		candidate.Caveats = sortedStrings(append(candidate.Caveats, caveats...))
		return ResolvedCandidate{Candidate: candidate, Matches: []rifcompat.Entity{match}}
	}
}

func lookupForCandidate(report rifcompat.Report, candidate Candidate) (rifcompat.LookupResult, []string) {
	text := strings.TrimSpace(candidate.CandidateText)
	switch candidate.MatchMode {
	case ModeQualifiedName:
		return rifcompat.ResolveLookup(report, rifcompat.LookupQualifiedName, strings.TrimSuffix(text, "()")), nil
	case ModeSourcePath:
		return rifcompat.ResolveLookup(report, rifcompat.LookupSourcePath, text), nil
	case ModeSimpleName:
		return rifcompat.ResolveLookup(report, rifcompat.LookupSimpleName, text), nil
	case ModeFuzzy:
		caveats := []string{CaveatFuzzyMatch}
		result := rifcompat.ResolveLookup(report, rifcompat.LookupSimpleName, text)
		if result.Status == rifcompat.LookupResolved {
			return result, caveats
		}
		pascal := snakeToPascal(text)
		if pascal != text {
			return rifcompat.ResolveLookup(report, rifcompat.LookupSimpleName, pascal), caveats
		}
		return result, caveats
	default:
		return rifcompat.LookupResult{Status: rifcompat.LookupUnresolved, Matches: []rifcompat.Entity{}}, nil
	}
}

// confidenceFrom keeps fuzzy and simple-name resolutions inferred regardless
// of the underlying lookup confidence.
func confidenceFrom(lookup rifcompat.Confidence, mode MatchMode) Confidence {
	if mode == ModeFuzzy || mode == ModeSimpleName {
		return ConfidenceInferred
	}
	if lookup == rifcompat.ConfidenceInferred {
		return ConfidenceInferred
	}
	return ConfidenceExact
}

func snakeToPascal(text string) string {
	parts := strings.Split(strings.ToLower(text), "_")
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		builder.WriteString(part[1:])
	}
	return builder.String()
}

func metricsFor(metrics map[string]*ResolutionMetrics, corpusID string) *ResolutionMetrics {
	if existing, ok := metrics[corpusID]; ok {
		return existing
	}
	created := &ResolutionMetrics{CorpusID: corpusID}
	metrics[corpusID] = created
	return created
}

func sortedMetrics(metrics map[string]*ResolutionMetrics) []ResolutionMetrics {
	out := make([]ResolutionMetrics, 0, len(metrics))
	for _, value := range metrics {
		out = append(out, *value)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CorpusID < out[j].CorpusID
	})
	return out
}

func marshalCaveats(caveats []string) (string, error) {
	if caveats == nil {
		caveats = []string{}
	}
	data, err := json.Marshal(caveats)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
