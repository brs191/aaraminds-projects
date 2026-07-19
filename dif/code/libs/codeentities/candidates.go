// Package codeentities detects unresolved code-entity candidates in anchored
// DIF document blocks.
package codeentities

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/aaraminds/dif/libs/extraction"
	"github.com/aaraminds/dif/libs/sourceanchors"
)

const (
	KindClass    CandidateKind = "class"
	KindMethod   CandidateKind = "method"
	KindFilePath CandidateKind = "file_path"
	KindService  CandidateKind = "service"
	KindUnknown  CandidateKind = "unknown"

	StatusUnresolved     MatchStatus = "unresolved"
	StatusResolved       MatchStatus = "resolved"
	StatusAmbiguous      MatchStatus = "ambiguous"
	StatusRIFUnavailable MatchStatus = "rif_unavailable"

	ModeQualifiedName MatchMode = "qualified-name"
	ModeSourcePath    MatchMode = "source-path"
	ModeSimpleName    MatchMode = "simple-name"
	ModeFuzzy         MatchMode = "fuzzy"

	ConfidenceExact    Confidence = "exact"
	ConfidenceInferred Confidence = "inferred"

	CaveatBacktickSpan        = "backtick_span"
	CaveatCodeFence           = "code_fence"
	CaveatIdentifierHeuristic = "identifier_heuristic"
	CaveatCandidateTruncated  = "candidate_text_truncated"

	maxCandidateTextLength = 256
)

var (
	filePathPattern   = regexp.MustCompile(`(^|[\s("'=])((?:[A-Za-z0-9_.-]+/)+(?:[A-Za-z0-9_.-]+\.(?:go|java|py|js|ts|tsx|jsx|json|ya?ml|md|txt|xml|sql|proto|kt|cs|rb|rs|cpp|hpp|h|c|scala)))`)
	httpRoutePattern  = regexp.MustCompile(`\b(?:GET|POST|PUT|PATCH|DELETE)\s+(/[A-Za-z0-9_./{}:-]+)`)
	methodPattern     = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)+\(\)|[A-Z][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)+|[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*){2,})\b`)
	qualifiedPattern  = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*){2,})\b`)
	classPattern      = regexp.MustCompile(`\b([A-Z][A-Za-z0-9]*(?:[A-Z][a-z0-9]+)[A-Za-z0-9]*)\b`)
	identifierPattern = regexp.MustCompile(
		`\b([a-z][a-z0-9]+(?:_[a-z0-9]+)+|[A-Z][A-Z0-9]+(?:_[A-Z0-9]+)+)\b`,
	)
	backtickPattern    = regexp.MustCompile("`([^`\n]{1,512})`")
	codeFencePattern   = regexp.MustCompile("(?s)```[A-Za-z0-9_-]*\n(.*?)```")
	serviceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]+(?:-[a-z0-9]+)+$`)
)

// CandidateKind mirrors dif_meta.code_entity_candidates.candidate_kind.
type CandidateKind string

// MatchStatus mirrors dif_meta.code_entity_candidates.match_status.
type MatchStatus string

// MatchMode mirrors dif_meta.code_entity_candidates.match_mode.
type MatchMode string

// Confidence mirrors dif_meta.code_entity_candidates.confidence.
type Confidence string

// Candidate is an unresolved code reference detected in anchored document text.
// A candidate is resolver input only; it is not a RIF node or DESCRIBES edge.
type Candidate struct {
	CandidateID       string
	CorpusID          string
	DocumentID        string
	DocumentVersionID string
	NodeID            string
	AnchorID          string
	SourceRef         string
	CandidateText     string
	CandidateKind     CandidateKind
	MatchStatus       MatchStatus
	ResolvedRIFNodeID string
	MatchMode         MatchMode
	Confidence        Confidence
	Caveats           []string
}

// Store persists code-entity candidates.
type Store interface {
	WriteCandidates(context.Context, []Candidate) error
}

// SQLStore writes unresolved candidates to dif_meta.code_entity_candidates
// using a caller-owned database handle or transaction.
type SQLStore struct {
	Execer Execer
}

// Execer is implemented by *sql.DB and *sql.Tx.
type Execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// MemoryStore records candidates in memory for tests and local harnesses.
type MemoryStore struct {
	Candidates []Candidate
}

// Detect returns deterministic unresolved code-entity candidates from anchored
// block nodes in an extraction result.
func Detect(result extraction.Result) ([]Candidate, error) {
	anchorsByID := map[string]sourceanchors.Anchor{}
	for _, anchor := range result.Anchors {
		anchorsByID[anchor.AnchorID] = anchor
	}

	var candidates []detectedCandidate
	for _, node := range sortedBlockNodes(result.Nodes) {
		if strings.TrimSpace(node.Text) == "" {
			continue
		}
		nodeCandidates := detectInText(node.Text)
		if len(nodeCandidates) == 0 {
			continue
		}
		if strings.TrimSpace(node.AnchorID) == "" {
			return nil, fmt.Errorf("node %q has code-entity candidates but no source anchor", node.NodeID)
		}
		anchor, ok := anchorsByID[node.AnchorID]
		if !ok {
			return nil, fmt.Errorf("node %q references unknown anchor_id %q", node.NodeID, node.AnchorID)
		}
		for _, detected := range nodeCandidates {
			candidates = append(candidates, detected.withNode(result, node, anchor))
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].candidate.NodeID != candidates[j].candidate.NodeID {
			return candidates[i].candidate.NodeID < candidates[j].candidate.NodeID
		}
		if candidates[i].start != candidates[j].start {
			return candidates[i].start < candidates[j].start
		}
		if candidates[i].candidate.CandidateText != candidates[j].candidate.CandidateText {
			return candidates[i].candidate.CandidateText < candidates[j].candidate.CandidateText
		}
		return candidates[i].candidate.CandidateKind < candidates[j].candidate.CandidateKind
	})

	out := make([]Candidate, 0, len(candidates))
	for ordinal, detected := range candidates {
		candidate := detected.candidate
		candidate.CandidateID = computeCandidateID(candidate, ordinal)
		if err := candidate.Validate(); err != nil {
			return nil, err
		}
		out = append(out, candidate)
	}
	return out, nil
}

// WriteCandidates writes validated candidates to SQL. Existing rows are kept
// unresolved unless a later resolver updates them.
func (s SQLStore) WriteCandidates(ctx context.Context, candidates []Candidate) error {
	if s.Execer == nil {
		return errors.New("codeentities SQL store requires an execer")
	}
	for _, candidate := range candidates {
		if err := candidate.Validate(); err != nil {
			return err
		}
		caveats, err := json.Marshal(candidate.Caveats)
		if err != nil {
			return err
		}
		_, err = s.Execer.ExecContext(ctx, `
INSERT INTO dif_meta.code_entity_candidates (
    candidate_id, corpus_id, document_id, document_version_id, node_id,
    anchor_id, candidate_text, candidate_kind, match_status,
    resolved_rif_node_id, match_mode, confidence, caveats
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::jsonb)
ON CONFLICT (candidate_id) DO UPDATE SET
    candidate_text = EXCLUDED.candidate_text,
    candidate_kind = EXCLUDED.candidate_kind,
    match_mode = EXCLUDED.match_mode,
    confidence = EXCLUDED.confidence,
    caveats = EXCLUDED.caveats
WHERE dif_meta.code_entity_candidates.match_status = 'unresolved'
  AND dif_meta.code_entity_candidates.resolved_rif_node_id IS NULL`,
			candidate.CandidateID,
			candidate.CorpusID,
			candidate.DocumentID,
			candidate.DocumentVersionID,
			candidate.NodeID,
			candidate.AnchorID,
			candidate.CandidateText,
			string(candidate.CandidateKind),
			string(candidate.MatchStatus),
			emptyToNil(candidate.ResolvedRIFNodeID),
			string(candidate.MatchMode),
			string(candidate.Confidence),
			string(caveats),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// WriteCandidates records validated candidates in memory.
func (s *MemoryStore) WriteCandidates(_ context.Context, candidates []Candidate) error {
	for _, candidate := range candidates {
		if err := candidate.Validate(); err != nil {
			return err
		}
	}
	s.Candidates = append(s.Candidates, candidates...)
	return nil
}

// Validate enforces the unresolved-candidate shape before persistence.
func (c Candidate) Validate() error {
	required := []struct {
		name  string
		value string
	}{
		{"candidate_id", c.CandidateID},
		{"corpus_id", c.CorpusID},
		{"document_id", c.DocumentID},
		{"document_version_id", c.DocumentVersionID},
		{"node_id", c.NodeID},
		{"anchor_id", c.AnchorID},
		{"source_ref", c.SourceRef},
		{"candidate_text", c.CandidateText},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
	}
	if len(c.CandidateText) > maxCandidateTextLength {
		return fmt.Errorf("candidate_text exceeds %d bytes", maxCandidateTextLength)
	}
	if !allowedKind(c.CandidateKind) {
		return fmt.Errorf("unsupported candidate_kind %q", c.CandidateKind)
	}
	if !allowedStatus(c.MatchStatus) {
		return fmt.Errorf("unsupported match_status %q", c.MatchStatus)
	}
	if !allowedMode(c.MatchMode) {
		return fmt.Errorf("unsupported match_mode %q", c.MatchMode)
	}
	if !allowedConfidence(c.Confidence) {
		return fmt.Errorf("unsupported confidence %q", c.Confidence)
	}
	if c.MatchStatus == StatusResolved && strings.TrimSpace(c.ResolvedRIFNodeID) == "" {
		return errors.New("resolved candidate requires resolved_rif_node_id")
	}
	if c.MatchStatus != StatusResolved && strings.TrimSpace(c.ResolvedRIFNodeID) != "" {
		return errors.New("unresolved candidate must not include resolved_rif_node_id")
	}
	return nil
}

type detectedCandidate struct {
	candidate Candidate
	start     int
}

func (d detectedCandidate) withNode(result extraction.Result, node extraction.Node, anchor sourceanchors.Anchor) detectedCandidate {
	candidate := d.candidate
	candidate.CorpusID = result.Document.CorpusID
	candidate.DocumentID = result.Document.DocumentID
	candidate.DocumentVersionID = result.Document.DocumentVersionID
	candidate.NodeID = node.NodeID
	candidate.AnchorID = node.AnchorID
	candidate.SourceRef = anchor.SourceRef
	candidate.MatchStatus = StatusUnresolved
	candidate.ResolvedRIFNodeID = ""
	d.candidate = candidate
	return d
}

func detectInText(text string) []detectedCandidate {
	seen := map[string]bool{}
	var out []detectedCandidate
	add := func(start int, raw string, kind CandidateKind, mode MatchMode, confidence Confidence, caveats ...string) {
		candidateText, candidateCaveats := normalizeCandidateText(raw, caveats)
		if candidateText == "" {
			return
		}
		key := fmt.Sprintf("%s\x00%s\x00%s", candidateText, kind, mode)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, detectedCandidate{
			start: start,
			candidate: Candidate{
				CandidateText: candidateText,
				CandidateKind: kind,
				MatchMode:     mode,
				Confidence:    confidence,
				Caveats:       candidateCaveats,
			},
		})
	}

	for _, match := range codeFencePattern.FindAllStringSubmatchIndex(text, -1) {
		if len(match) >= 4 {
			for _, nested := range scanGeneral(text[match[2]:match[3]], match[2], CaveatCodeFence) {
				add(nested.start, nested.candidate.CandidateText, nested.candidate.CandidateKind, nested.candidate.MatchMode, nested.candidate.Confidence, nested.candidate.Caveats...)
			}
		}
	}
	for _, match := range backtickPattern.FindAllStringSubmatchIndex(text, -1) {
		if len(match) >= 4 {
			raw := text[match[2]:match[3]]
			kind, mode, confidence, ok := classifyBacktick(raw)
			if ok {
				add(match[2], raw, kind, mode, confidence, CaveatBacktickSpan)
			}
		}
	}
	for _, detected := range scanGeneral(text, 0, "") {
		add(detected.start, detected.candidate.CandidateText, detected.candidate.CandidateKind, detected.candidate.MatchMode, detected.candidate.Confidence, detected.candidate.Caveats...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].start != out[j].start {
			return out[i].start < out[j].start
		}
		return out[i].candidate.CandidateText < out[j].candidate.CandidateText
	})
	return out
}

func scanGeneral(text string, offset int, contextCaveat string) []detectedCandidate {
	var out []detectedCandidate
	add := func(match []int, group int, kind CandidateKind, mode MatchMode, confidence Confidence, caveats ...string) {
		if len(match) <= group*2+1 || match[group*2] < 0 {
			return
		}
		allCaveats := append([]string{}, caveats...)
		if contextCaveat != "" {
			allCaveats = append(allCaveats, contextCaveat)
		}
		out = append(out, detectedCandidate{
			start: offset + match[group*2],
			candidate: Candidate{
				CandidateText: text[match[group*2]:match[group*2+1]],
				CandidateKind: kind,
				MatchMode:     mode,
				Confidence:    confidence,
				Caveats:       sortedStrings(allCaveats),
			},
		})
	}
	for _, match := range filePathPattern.FindAllStringSubmatchIndex(text, -1) {
		add(match, 2, KindFilePath, ModeSourcePath, ConfidenceExact)
	}
	for _, match := range httpRoutePattern.FindAllStringSubmatchIndex(text, -1) {
		add(match, 1, KindService, ModeSimpleName, ConfidenceInferred)
	}
	for _, match := range methodPattern.FindAllStringSubmatchIndex(text, -1) {
		raw := text[match[2]:match[3]]
		if isMethodLike(raw) {
			add(match, 1, KindMethod, ModeQualifiedName, ConfidenceExact)
		}
	}
	for _, match := range qualifiedPattern.FindAllStringSubmatchIndex(text, -1) {
		raw := text[match[2]:match[3]]
		if !isMethodLike(raw) {
			add(match, 1, classifyQualifiedName(raw), ModeQualifiedName, ConfidenceExact)
		}
	}
	for _, match := range classPattern.FindAllStringSubmatchIndex(text, -1) {
		add(match, 1, KindClass, ModeSimpleName, ConfidenceInferred)
	}
	for _, match := range identifierPattern.FindAllStringSubmatchIndex(text, -1) {
		add(match, 1, KindUnknown, ModeFuzzy, ConfidenceInferred, CaveatIdentifierHeuristic)
	}
	return out
}

func classifyBacktick(raw string) (CandidateKind, MatchMode, Confidence, bool) {
	text, _ := normalizeCandidateText(raw, nil)
	if text == "" || strings.Contains(text, " ") {
		return "", "", "", false
	}
	switch {
	case filePathPattern.MatchString(" " + text):
		return KindFilePath, ModeSourcePath, ConfidenceExact, true
	case serviceNamePattern.MatchString(text):
		return KindService, ModeSimpleName, ConfidenceInferred, true
	case strings.Contains(text, ".") && isMethodLike(text):
		return KindMethod, ModeQualifiedName, ConfidenceExact, true
	case strings.Count(text, ".") >= 2:
		return classifyQualifiedName(text), ModeQualifiedName, ConfidenceExact, true
	case classPattern.MatchString(text):
		return KindClass, ModeSimpleName, ConfidenceInferred, true
	case identifierPattern.MatchString(text):
		return KindUnknown, ModeFuzzy, ConfidenceInferred, true
	default:
		return "", "", "", false
	}
}

func classifyQualifiedName(text string) CandidateKind {
	last := text[strings.LastIndex(text, ".")+1:]
	if last != "" && last[0] >= 'A' && last[0] <= 'Z' {
		return KindClass
	}
	return KindUnknown
}

func isMethodLike(text string) bool {
	if strings.HasSuffix(text, "()") {
		return true
	}
	last := text[strings.LastIndex(text, ".")+1:]
	if last == "" {
		return false
	}
	return last[0] >= 'a' && last[0] <= 'z'
}

func normalizeCandidateText(raw string, caveats []string) (string, []string) {
	text := strings.TrimSpace(raw)
	text = strings.Trim(text, `"'“”‘’.,;:`)
	text = strings.TrimSpace(text)
	if text == "" {
		return "", nil
	}
	normalizedCaveats := sortedStrings(caveats)
	if len(text) > maxCandidateTextLength {
		text = text[:maxCandidateTextLength]
		normalizedCaveats = sortedStrings(append(normalizedCaveats, CaveatCandidateTruncated))
	}
	return text, normalizedCaveats
}

func sortedBlockNodes(nodes []extraction.Node) []extraction.Node {
	var blocks []extraction.Node
	for _, node := range nodes {
		if node.Kind == extraction.NodeBlock {
			blocks = append(blocks, node)
		}
	}
	sort.SliceStable(blocks, func(i, j int) bool {
		if blocks[i].Ordinal != blocks[j].Ordinal {
			return blocks[i].Ordinal < blocks[j].Ordinal
		}
		return blocks[i].NodeID < blocks[j].NodeID
	})
	return blocks
}

func computeCandidateID(candidate Candidate, ordinal int) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(candidate.CorpusID),
		strings.TrimSpace(candidate.DocumentVersionID),
		strings.TrimSpace(candidate.NodeID),
		strings.TrimSpace(candidate.AnchorID),
		string(candidate.CandidateKind),
		strings.TrimSpace(candidate.CandidateText),
		fmt.Sprint(ordinal),
	}, "\x00")))
	return hex.EncodeToString(sum[:])
}

func allowedKind(kind CandidateKind) bool {
	switch kind {
	case KindClass, KindMethod, KindFilePath, KindService, KindUnknown:
		return true
	default:
		return false
	}
}

func allowedStatus(status MatchStatus) bool {
	switch status {
	case StatusUnresolved, StatusResolved, StatusAmbiguous, StatusRIFUnavailable:
		return true
	default:
		return false
	}
}

func allowedMode(mode MatchMode) bool {
	switch mode {
	case ModeQualifiedName, ModeSourcePath, ModeSimpleName, ModeFuzzy:
		return true
	default:
		return false
	}
}

func allowedConfidence(confidence Confidence) bool {
	switch confidence {
	case ConfidenceExact, ConfidenceInferred:
		return true
	default:
		return false
	}
}

func sortedStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func emptyToNil(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}
