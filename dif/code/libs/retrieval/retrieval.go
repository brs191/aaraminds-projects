// Package retrieval implements the P0 source-anchored retrieval path.
package retrieval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/aaraminds/dif/libs/admission"
	"github.com/aaraminds/dif/libs/extraction"
	"github.com/aaraminds/dif/libs/requestctx"
	"github.com/aaraminds/dif/libs/sourceanchors"
)

const (
	StatusOK                Status = "ok"
	StatusNoEvidence        Status = "no_evidence"
	StatusCorpusNotAdmitted Status = "corpus_not_admitted"
)

var (
	tokenPattern = regexp.MustCompile(`[A-Za-z0-9]+`)
	stopwords    = map[string]bool{
		"a": true, "an": true, "and": true, "any": true, "does": true, "for": true,
		"is": true, "mentions": true, "mention": true, "of": true, "the": true,
		"what": true, "which": true, "who": true,
	}
)

// Status is the explicit retrieval response state.
type Status string

// Query is a P0 search_docs-style query.
type Query struct {
	Text  string
	Limit int
}

// Response is a grounded retrieval response. It never contains natural-language
// answers beyond retrieved evidence snippets.
type Response struct {
	Status  Status
	Results []Result
}

// Result is a source-anchored retrieval hit.
type Result struct {
	CorpusID          string
	DocumentID        string
	DocumentVersionID string
	PassageID         string
	Snippet           string
	AnchorID          string
	SourceRef         string
	Score             float64
	Caveats           []string
}

// Index is a deterministic in-memory P0 retrieval index. A later persistence
// prompt can back the same contract with Postgres FTS.
type Index struct {
	passages []indexedPassage
}

// Searcher combines corpus admission with an anchored passage index.
type Searcher struct {
	Admission admission.Catalog
	Index     Index
}

type indexedPassage struct {
	extraction.Passage
	Anchor      sourceanchors.Anchor
	SearchText  string
	TermCounts  map[string]int
	Specificity int
	Caveats     []string
}

type scoredPassage struct {
	passage indexedPassage
	score   float64
}

// NewIndex builds retrieval passages from extractor output and excludes any
// passage that is not source anchored.
func NewIndex(results ...extraction.Result) (Index, error) {
	var passages []indexedPassage
	for _, result := range results {
		anchors := map[string]sourceanchors.Anchor{}
		for _, anchor := range result.Anchors {
			if strings.TrimSpace(anchor.AnchorID) == "" {
				continue
			}
			anchors[anchor.AnchorID] = anchor
		}
		for _, passage := range result.Passages {
			if strings.TrimSpace(passage.AnchorID) == "" || strings.TrimSpace(passage.SourceRef) == "" {
				continue
			}
			anchor, ok := anchors[passage.AnchorID]
			if !ok {
				continue
			}
			if passage.SourceRef != anchor.SourceRef {
				return Index{}, fmt.Errorf("passage %q source_ref does not match anchor %q", passage.PassageID, passage.AnchorID)
			}
			searchText := strings.Join([]string{
				passage.Text,
				anchor.HeadingPath,
			}, "\n")
			passages = append(passages, indexedPassage{
				Passage:     passage,
				Anchor:      anchor,
				SearchText:  searchText,
				TermCounts:  termCounts(searchText),
				Specificity: anchorSpecificity(anchor),
				Caveats:     append([]string(nil), anchor.Caveats...),
			})
		}
	}
	sort.SliceStable(passages, func(i, j int) bool {
		return passages[i].PassageID < passages[j].PassageID
	})
	return Index{passages: passages}, nil
}

// NewSearcher builds a Searcher from an admission catalog and extractor output.
func NewSearcher(catalog admission.Catalog, results ...extraction.Result) (Searcher, error) {
	index, err := NewIndex(results...)
	if err != nil {
		return Searcher{}, err
	}
	return Searcher{Admission: catalog, Index: index}, nil
}

// SearchDocs executes deterministic P0 lexical retrieval over anchored passages.
func (s Searcher) SearchDocs(ctx context.Context, query Query) (Response, error) {
	text := strings.TrimSpace(query.Text)
	if text == "" {
		return Response{}, errors.New("query text is required")
	}
	exec, err := requestctx.RequireFromContext(ctx, requestctx.OperationRetrieval)
	if err != nil {
		return Response{}, err
	}
	decision := s.Admission.CheckCorpus(ctx, "search_docs", queryHash(text))
	if !decision.Allowed {
		return Response{Status: StatusCorpusNotAdmitted, Results: []Result{}}, nil
	}

	terms := queryTerms(text)
	if len(terms) == 0 {
		return Response{Status: StatusNoEvidence, Results: []Result{}}, nil
	}

	var scored []scoredPassage
	for _, passage := range s.Index.passages {
		if passage.CorpusID != exec.CorpusID {
			continue
		}
		score := scorePassage(terms, passage)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredPassage{passage: passage, score: score})
	}
	if len(scored) == 0 {
		return Response{Status: StatusNoEvidence, Results: []Result{}}, nil
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].passage.Specificity != scored[j].passage.Specificity {
			return scored[i].passage.Specificity > scored[j].passage.Specificity
		}
		return scored[i].passage.SourceRef < scored[j].passage.SourceRef
	})

	limit := query.Limit
	if limit <= 0 || limit > 10 {
		limit = 10
	}
	if len(scored) > limit {
		scored = scored[:limit]
	}

	results := make([]Result, 0, len(scored))
	for _, item := range scored {
		passage := item.passage
		if strings.TrimSpace(passage.AnchorID) == "" || strings.TrimSpace(passage.SourceRef) == "" {
			continue
		}
		results = append(results, Result{
			CorpusID:          passage.CorpusID,
			DocumentID:        passage.DocumentID,
			DocumentVersionID: passage.DocumentVersionID,
			PassageID:         passage.PassageID,
			Snippet:           passage.Text,
			AnchorID:          passage.AnchorID,
			SourceRef:         passage.SourceRef,
			Score:             roundScore(item.score),
			Caveats:           append([]string{}, passage.Caveats...),
		})
	}
	if len(results) == 0 {
		return Response{Status: StatusNoEvidence, Results: []Result{}}, nil
	}
	return Response{Status: StatusOK, Results: results}, nil
}

func scorePassage(queryTerms []string, passage indexedPassage) float64 {
	var matched int
	var weighted float64
	for _, term := range queryTerms {
		count := passage.TermCounts[term]
		if count == 0 {
			continue
		}
		matched++
		weighted += 1 + math.Log1p(float64(count))
	}
	if matched == 0 {
		return 0
	}
	coverage := float64(matched) / float64(len(queryTerms))
	score := weighted + coverage + (float64(passage.Specificity) * 0.01)
	if isCompositeJSONPassage(passage) {
		score -= 3
	}
	return score
}

func queryTerms(query string) []string {
	seen := map[string]bool{}
	var terms []string
	for _, token := range normalizeTokens(query) {
		if stopwords[token] || seen[token] {
			continue
		}
		seen[token] = true
		terms = append(terms, token)
	}
	sort.Strings(terms)
	return terms
}

func termCounts(text string) map[string]int {
	counts := map[string]int{}
	for _, token := range normalizeTokens(text) {
		counts[token]++
	}
	return counts
}

func normalizeTokens(text string) []string {
	raw := tokenPattern.FindAllString(text, -1)
	tokens := make([]string, 0, len(raw)*2)
	for _, value := range raw {
		token := normalizeToken(value)
		if token == "" {
			continue
		}
		tokens = append(tokens, token)
		for _, part := range splitCamelToken(value) {
			normalized := normalizeToken(part)
			if normalized != "" && normalized != token {
				tokens = append(tokens, normalized)
			}
		}
	}
	return tokens
}

func normalizeToken(value string) string {
	token := strings.ToLower(strings.TrimSpace(value))
	if token == "" {
		return ""
	}
	switch token {
	case "own", "owns", "owned", "owner", "owners":
		return "own"
	case "configured", "configuration":
		return "config"
	}
	if len(token) > 4 && strings.HasSuffix(token, "ies") {
		token = strings.TrimSuffix(token, "ies") + "y"
	}
	if len(token) > 4 && strings.HasSuffix(token, "ed") {
		token = strings.TrimSuffix(token, "ed")
	}
	if len(token) > 3 && strings.HasSuffix(token, "s") {
		token = strings.TrimSuffix(token, "s")
	}
	return token
}

func splitCamelToken(value string) []string {
	var parts []string
	start := 0
	runes := []rune(value)
	for i := 1; i < len(runes); i++ {
		if runes[i] >= 'A' && runes[i] <= 'Z' && runes[i-1] >= 'a' && runes[i-1] <= 'z' {
			parts = append(parts, string(runes[start:i]))
			start = i
		}
	}
	if start > 0 {
		parts = append(parts, string(runes[start:]))
	}
	return parts
}

func anchorSpecificity(anchor sourceanchors.Anchor) int {
	switch anchor.AnchorType {
	case sourceanchors.TypeJSON:
		return strings.Count(anchor.JSONPath, ".") + strings.Count(anchor.JSONPath, "[") + 10
	case sourceanchors.TypeDOCX:
		return 8
	case sourceanchors.TypeMarkdown, sourceanchors.TypeText:
		return 5
	default:
		return 0
	}
}

func isCompositeJSONPassage(passage indexedPassage) bool {
	return passage.Anchor.AnchorType == sourceanchors.TypeJSON && strings.Contains(passage.Text, "\n{")
}

func roundScore(score float64) float64 {
	return math.Round(score*1000000) / 1000000
}

func queryHash(query string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(query)))
	return "sha256:" + hex.EncodeToString(sum[:])
}
