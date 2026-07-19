// Package extraction provides deterministic P0 document extractors.
package extraction

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/aaraminds/dif/libs/ingestionruns"
	"github.com/aaraminds/dif/libs/sourceanchors"
)

const (
	FormatMarkdown Format = "md"
	FormatText     Format = "txt"
	FormatDOCX     Format = "docx"
	FormatJSON     Format = "json"

	NodeDocument NodeKind = "document"
	NodeSection  NodeKind = "section"
	NodeBlock    NodeKind = "block"

	EdgeContains EdgeKind = "CONTAINS"
)

// Format is a P0 document format.
type Format string

// NodeKind is the DIF document graph node kind.
type NodeKind string

// EdgeKind is the DIF document graph edge kind.
type EdgeKind string

// Options identifies the immutable document version being extracted.
type Options struct {
	CorpusID          string
	DocumentID        string
	DocumentVersionID string
	SourceID          string
	Path              string
}

// Document is the logical document record emitted by an extractor.
type Document struct {
	DocumentID        string
	CorpusID          string
	SourceID          string
	Path              string
	Format            Format
	DocumentVersionID string
	ContentHash       string
}

// Node is a document graph node.
type Node struct {
	NodeID            string
	CorpusID          string
	DocumentID        string
	DocumentVersionID string
	Kind              NodeKind
	ParentNodeID      string
	Ordinal           int
	HeadingPath       string
	AnchorID          string
	TextHash          string
	Text              string
}

// Edge is a document graph edge.
type Edge struct {
	EdgeID            string
	CorpusID          string
	DocumentVersionID string
	Kind              EdgeKind
	FromNodeID        string
	ToNodeID          string
	Confidence        string
	AnchorID          string
}

// Passage is a retrieval passage candidate. P0 creates one passage per block.
type Passage struct {
	PassageID         string
	CorpusID          string
	DocumentID        string
	DocumentVersionID string
	NodeID            string
	AnchorID          string
	SourceRef         string
	Text              string
	ContentHash       string
}

// Result contains deterministic extraction output for a single source file.
type Result struct {
	Document Document
	Nodes    []Node
	Edges    []Edge
	Anchors  []sourceanchors.Anchor
	Passages []Passage
	Caveats  []Caveat
}

// Caveat is a machine-readable extraction caveat.
type Caveat struct {
	Code     string
	Message  string
	JSONPath string
	Limit    int
	Observed int
}

// IngestionRun returns the run-count shape used by the P0 promotion guard.
func (r Result) IngestionRun(runID string, status ingestionruns.Status) ingestionruns.Run {
	documentCount := 0
	if strings.TrimSpace(r.Document.DocumentID) != "" {
		documentCount = 1
	}
	return ingestionruns.Run{
		RunID:         runID,
		CorpusID:      r.Document.CorpusID,
		SourceID:      r.Document.SourceID,
		Status:        status,
		DocumentCount: documentCount,
		NodeCount:     len(r.Nodes),
		EdgeCount:     len(r.Edges),
		AnchorCount:   len(r.Anchors),
		PassageCount:  len(r.Passages),
		CaveatCount:   len(r.Caveats),
	}
}

func (o Options) validate() error {
	var missing []string
	for _, field := range []struct {
		name  string
		value string
	}{
		{"corpus_id", o.CorpusID},
		{"document_id", o.DocumentID},
		{"document_version_id", o.DocumentVersionID},
		{"source_id", o.SourceID},
		{"path", o.Path},
	} {
		if strings.TrimSpace(field.value) == "" {
			missing = append(missing, field.name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing extraction options: %s", strings.Join(missing, ", "))
	}
	return nil
}

func newBaseResult(opts Options, format Format, lines []string) Result {
	text := strings.Join(lines, "\n")
	return Result{
		Document: Document{
			DocumentID:        strings.TrimSpace(opts.DocumentID),
			CorpusID:          strings.TrimSpace(opts.CorpusID),
			SourceID:          strings.TrimSpace(opts.SourceID),
			Path:              normalizePath(opts.Path),
			Format:            format,
			DocumentVersionID: strings.TrimSpace(opts.DocumentVersionID),
			ContentHash:       sourceanchors.ContentHash(text),
		},
	}
}

func (r *Result) addNode(opts Options, kind NodeKind, parentNodeID string, ordinal int, headingPath string, lineStart, lineEnd int, text string, anchorType sourceanchors.AnchorType) (Node, error) {
	var anchor sourceanchors.Anchor
	if strings.TrimSpace(text) != "" && lineStart > 0 && lineEnd >= lineStart {
		payload := fmt.Sprintf("%s#L%d-L%d", normalizePath(opts.Path), lineStart, lineEnd)
		sourceRef := sourceanchors.SourceRef{
			CorpusID:          strings.TrimSpace(opts.CorpusID),
			DocumentVersionID: strings.TrimSpace(opts.DocumentVersionID),
			AnchorType:        anchorType,
			Payload:           payload,
		}.String()
		anchor = sourceanchors.Anchor{
			AnchorID:          sourceanchors.ComputeAnchorID(opts.CorpusID, opts.DocumentVersionID, anchorType, payload),
			CorpusID:          strings.TrimSpace(opts.CorpusID),
			DocumentID:        strings.TrimSpace(opts.DocumentID),
			DocumentVersionID: strings.TrimSpace(opts.DocumentVersionID),
			AnchorType:        anchorType,
			SourceRef:         sourceRef,
			Path:              normalizePath(opts.Path),
			HeadingPath:       strings.TrimSpace(headingPath),
			LineStart:         lineStart,
			LineEnd:           lineEnd,
			ContentHash:       sourceanchors.ContentHash(text),
		}
		r.Anchors = append(r.Anchors, anchor)
	}

	node := Node{
		NodeID:            stableID("node", opts.CorpusID, opts.DocumentVersionID, string(kind), normalizePath(opts.Path), fmt.Sprint(ordinal), headingPath, fmt.Sprintf("%d-%d", lineStart, lineEnd)),
		CorpusID:          strings.TrimSpace(opts.CorpusID),
		DocumentID:        strings.TrimSpace(opts.DocumentID),
		DocumentVersionID: strings.TrimSpace(opts.DocumentVersionID),
		Kind:              kind,
		ParentNodeID:      parentNodeID,
		Ordinal:           ordinal,
		HeadingPath:       strings.TrimSpace(headingPath),
		AnchorID:          anchor.AnchorID,
		TextHash:          sourceanchors.ContentHash(text),
		Text:              text,
	}
	r.Nodes = append(r.Nodes, node)
	if parentNodeID != "" {
		r.addEdge(opts, parentNodeID, node.NodeID, anchor.AnchorID, len(r.Edges))
	}
	if kind == NodeBlock && anchor.AnchorID != "" {
		r.Passages = append(r.Passages, Passage{
			PassageID:         stableID("passage", opts.CorpusID, opts.DocumentVersionID, node.NodeID, anchor.AnchorID),
			CorpusID:          strings.TrimSpace(opts.CorpusID),
			DocumentID:        strings.TrimSpace(opts.DocumentID),
			DocumentVersionID: strings.TrimSpace(opts.DocumentVersionID),
			NodeID:            node.NodeID,
			AnchorID:          anchor.AnchorID,
			SourceRef:         anchor.SourceRef,
			Text:              text,
			ContentHash:       sourceanchors.ContentHash(text),
		})
	}
	return node, nil
}

func (r *Result) addEdge(opts Options, fromNodeID, toNodeID, anchorID string, ordinal int) {
	r.Edges = append(r.Edges, Edge{
		EdgeID:            stableID("edge", opts.CorpusID, opts.DocumentVersionID, fromNodeID, string(EdgeContains), toNodeID, fmt.Sprint(ordinal)),
		CorpusID:          strings.TrimSpace(opts.CorpusID),
		DocumentVersionID: strings.TrimSpace(opts.DocumentVersionID),
		Kind:              EdgeContains,
		FromNodeID:        fromNodeID,
		ToNodeID:          toNodeID,
		Confidence:        "exact",
		AnchorID:          anchorID,
	})
}

func lines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

func lineRangeText(lines []string, start, end int) string {
	if start < 1 || end < start || end > len(lines) {
		return ""
	}
	return strings.Join(lines[start-1:end], "\n")
}

func stableID(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(trimmed(parts), "\x00")))
	return hex.EncodeToString(sum[:])
}

func trimmed(values []string) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = strings.TrimSpace(value)
	}
	return result
}

func normalizePath(path string) string {
	return strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")
}

func ensureNonEmptyContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return errors.New("source content is empty")
	}
	return nil
}
