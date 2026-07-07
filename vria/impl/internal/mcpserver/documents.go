// documents.go implements the search_evidence_documents tool (09 §3.6).
//
// Reference adapter: a directory of .md / .txt files.  Retrieval uses naive
// keyword scoring (term-frequency count).  Every result MUST carry a non-empty
// citation_pointer (file path + line number of the best-matching line).
// Results without a citation_pointer are dropped before returning.
//
// Empty result set → output with empty results array and error code
// NO_EVIDENCE_FOUND.  Citations are never fabricated.
//
// Document content is treated as untrusted: snippets are returned verbatim but
// the server never interprets instructions embedded in them.
package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aaraminds/vria/internal/enums"
)

// EvidenceConfig configures the search_evidence_documents handler.
type EvidenceConfig struct {
	// EvidenceDir is the directory containing .md and .txt evidence files.
	EvidenceDir string
}

// --- input / output types (mirrors §3.6 JSON contract) ---

type searchEvidenceInput struct {
	Query   string          `json:"query"`
	Filters *evidenceFilter `json:"filters"`
	TopK    int             `json:"top_k"`
}

type evidenceFilter struct {
	UseCaseID    string   `json:"use_case_id"`
	DocumentType []string `json:"document_type"`
	DateFrom     string   `json:"date_from"`
	DateTo       string   `json:"date_to"`
}

type searchEvidenceOutput struct {
	Results []evidenceResult `json:"results"`
	AuditID string           `json:"audit_id"`
	// ErrorCode is populated when the result set is empty (NO_EVIDENCE_FOUND).
	// A non-empty results array never carries an error_code.
	ErrorCode string `json:"error_code,omitempty"`
}

type evidenceResult struct {
	DocumentID      string          `json:"document_id"`
	Title           string          `json:"title"`
	CitationPointer string          `json:"citation_pointer"`
	Authority       enums.Authority `json:"authority"`
	Freshness       enums.Freshness `json:"freshness"`
	EvidenceQuality string          `json:"evidence_quality"`
	Snippet         string          `json:"snippet"`
}

// candidateDoc accumulates scoring data for one file.
type candidateDoc struct {
	path        string
	score       int
	bestLineNum int
	bestLine    string
}

// NewSearchEvidenceHandler returns a Handler for search_evidence_documents
// backed by a directory of .md / .txt files.
func NewSearchEvidenceHandler(cfg EvidenceConfig) Handler {
	return func(ctx context.Context, input json.RawMessage) (interface{}, *ToolError) {
		var req searchEvidenceInput
		if err := json.Unmarshal(input, &req); err != nil {
			return nil, &ToolError{Code: ErrInvalidInput, Message: "cannot parse input: " + err.Error()}
		}
		if strings.TrimSpace(req.Query) == "" {
			return nil, &ToolError{Code: ErrInvalidInput, Message: "missing required field: query"}
		}
		topK := req.TopK
		if topK <= 0 {
			topK = 10
		}

		terms := tokenise(req.Query)
		candidates, err := scoreDocuments(cfg.EvidenceDir, terms)
		if err != nil {
			return nil, &ToolError{Code: ErrInternalError, Message: "cannot scan evidence dir: " + err.Error()}
		}

		// Sort descending by score, then by path for determinism.
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].score != candidates[j].score {
				return candidates[i].score > candidates[j].score
			}
			return candidates[i].path < candidates[j].path
		})

		// Build results, dropping any without a citation_pointer.
		var results []evidenceResult
		for _, c := range candidates {
			if c.score == 0 {
				break // remaining candidates have zero score
			}
			if c.bestLine == "" || c.bestLineNum == 0 {
				// No citation available — drop per guardrail.
				continue
			}
			citationPointer := fmt.Sprintf("%s:%d", c.path, c.bestLineNum)
			snippet := c.bestLine
			if len(snippet) > 300 {
				snippet = snippet[:300]
			}

			results = append(results, evidenceResult{
				DocumentID:      newAuditID(), // deterministic uuid not possible without registry; use random id
				Title:           filepath.Base(c.path),
				CitationPointer: citationPointer,
				Authority:       enums.AuthorityUnknown,
				Freshness:       enums.FreshnessUnknown,
				EvidenceQuality: "Low", // naive keyword search; quality always Low without richer metadata
				Snippet:         snippet,
			})
			if len(results) >= topK {
				break
			}
		}

		// Contract §3.6: empty result → output with empty results array and
		// error_code NO_EVIDENCE_FOUND; no fabricated citations.
		if len(results) == 0 {
			return searchEvidenceOutput{
				Results:   []evidenceResult{},
				AuditID:   newAuditID(),
				ErrorCode: string(ErrNoEvidenceFound),
			}, nil
		}

		return searchEvidenceOutput{
			Results: results,
			AuditID: newAuditID(),
		}, nil
	}
}

// tokenise lowercases and splits a query into non-empty terms.
func tokenise(query string) []string {
	raw := strings.Fields(strings.ToLower(query))
	out := raw[:0]
	for _, t := range raw {
		t = strings.Trim(t, ".,;:!?\"'()")
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// scoreDocuments walks EvidenceDir and scores each .md / .txt file against
// the query terms. Returns one candidateDoc per file (score may be zero).
func scoreDocuments(dir string, terms []string) ([]candidateDoc, error) {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var candidates []candidateDoc
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".md" && ext != ".txt" {
			continue
		}
		fullPath := filepath.Join(dir, name)
		c, err := scoreFile(fullPath, terms)
		if err != nil {
			continue // skip unreadable files silently
		}
		candidates = append(candidates, c)
	}
	return candidates, nil
}

// scoreFile scores a single file against terms. It counts term occurrences
// per line, accumulates a total score, and tracks the line with the highest
// per-line score as the citation anchor.
func scoreFile(path string, terms []string) (candidateDoc, error) {
	f, err := os.Open(path)
	if err != nil {
		return candidateDoc{}, err
	}
	defer f.Close()

	c := candidateDoc{path: path}
	scanner := bufio.NewScanner(f)
	// Evidence documents can contain long lines (minified exports, wide
	// tables); the default 64KB token limit would silently drop the whole
	// file. Raise it so such documents are still searchable.
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	lineNum := 0
	bestLineScore := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lower := strings.ToLower(line)
		lineScore := 0
		for _, term := range terms {
			lineScore += strings.Count(lower, term)
		}
		c.score += lineScore
		if lineScore > bestLineScore {
			bestLineScore = lineScore
			c.bestLineNum = lineNum
			c.bestLine = line
		}
	}
	return c, scanner.Err()
}
