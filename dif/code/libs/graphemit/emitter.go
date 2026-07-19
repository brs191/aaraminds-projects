// Package graphemit turns deterministic extractor output into byte-stable
// NDJSON records for downstream loading and service-level tests.
package graphemit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/aaraminds/dif/libs/extraction"
	"github.com/aaraminds/dif/libs/sourceanchors"
)

const (
	defaultExtractorName    = "dif-p0-extractor"
	defaultExtractorVersion = "p0"
	passageKindStructural   = "structural"
	passageKindJSONSubtree  = "json_subtree"
)

// Options controls metadata added to emitted records.
type Options struct {
	ExtractorName    string
	ExtractorVersion string
}

// EmitNDJSON validates extraction output and writes byte-stable NDJSON.
func EmitNDJSON(result extraction.Result) ([]byte, error) {
	return EmitNDJSONWithOptions(result, Options{})
}

// EmitNDJSONWithOptions validates extraction output and writes byte-stable
// NDJSON with caller-provided extractor metadata.
func EmitNDJSONWithOptions(result extraction.Result, opts Options) ([]byte, error) {
	if err := Validate(result); err != nil {
		return nil, err
	}
	opts = opts.withDefaults()

	var buffer bytes.Buffer
	if err := writeRecord(&buffer, documentRecordFrom(result.Document, opts)); err != nil {
		return nil, err
	}
	for _, anchor := range sortedAnchors(result.Anchors) {
		if err := writeRecord(&buffer, sourceAnchorRecordFrom(anchor, result.Document.SourceID, opts)); err != nil {
			return nil, err
		}
	}
	for _, node := range sortedNodes(result.Nodes) {
		if err := writeRecord(&buffer, nodeRecordFrom(node)); err != nil {
			return nil, err
		}
	}
	for _, edge := range sortedEdges(result.Edges) {
		if err := writeRecord(&buffer, edgeRecordFrom(edge)); err != nil {
			return nil, err
		}
	}
	for _, passage := range sortedPassages(result.Passages) {
		anchor := anchorByID(result.Anchors)[passage.AnchorID]
		if err := writeRecord(&buffer, passageRecordFrom(passage, anchor)); err != nil {
			return nil, err
		}
	}
	for _, caveat := range sortedCaveats(result.Caveats) {
		if err := writeRecord(&buffer, caveatRecordFrom(result.Document, caveat)); err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}

// Validate enforces the P0 graph invariants before any records are emitted.
func Validate(result extraction.Result) error {
	if strings.TrimSpace(result.Document.DocumentID) == "" {
		return errors.New("document_id is required")
	}
	if strings.TrimSpace(result.Document.CorpusID) == "" {
		return errors.New("corpus_id is required")
	}
	if strings.TrimSpace(result.Document.SourceID) == "" {
		return errors.New("source_id is required")
	}
	if strings.TrimSpace(result.Document.DocumentVersionID) == "" {
		return errors.New("document_version_id is required")
	}
	if strings.TrimSpace(result.Document.Path) == "" {
		return errors.New("document path is required")
	}

	nodes := map[string]extraction.Node{}
	for _, node := range result.Nodes {
		if strings.TrimSpace(node.NodeID) == "" {
			return errors.New("node_id is required")
		}
		if _, exists := nodes[node.NodeID]; exists {
			return fmt.Errorf("duplicate node_id %q", node.NodeID)
		}
		if node.Kind != extraction.NodeDocument && node.Kind != extraction.NodeSection && node.Kind != extraction.NodeBlock {
			return fmt.Errorf("unsupported node kind %q for node %q", node.Kind, node.NodeID)
		}
		nodes[node.NodeID] = node
	}

	anchors := map[string]sourceanchors.Anchor{}
	for _, anchor := range result.Anchors {
		if strings.TrimSpace(anchor.AnchorID) == "" {
			return errors.New("anchor_id is required")
		}
		if strings.TrimSpace(anchor.SourceRef) == "" {
			return fmt.Errorf("source_ref is required for anchor %q", anchor.AnchorID)
		}
		if strings.TrimSpace(anchor.ContentHash) == "" {
			return fmt.Errorf("content_hash is required for anchor %q", anchor.AnchorID)
		}
		if _, exists := anchors[anchor.AnchorID]; exists {
			return fmt.Errorf("duplicate anchor_id %q", anchor.AnchorID)
		}
		anchors[anchor.AnchorID] = anchor
	}

	for _, node := range result.Nodes {
		if node.ParentNodeID != "" {
			if _, ok := nodes[node.ParentNodeID]; !ok {
				return fmt.Errorf("node %q has unknown parent_node_id %q", node.NodeID, node.ParentNodeID)
			}
		}
		if node.AnchorID != "" {
			if _, ok := anchors[node.AnchorID]; !ok {
				return fmt.Errorf("node %q references unknown anchor_id %q", node.NodeID, node.AnchorID)
			}
		}
	}

	for _, edge := range result.Edges {
		if strings.TrimSpace(edge.EdgeID) == "" {
			return errors.New("edge_id is required")
		}
		if edge.Kind != extraction.EdgeContains {
			return fmt.Errorf("unsupported edge kind %q for edge %q", edge.Kind, edge.EdgeID)
		}
		if _, ok := nodes[edge.FromNodeID]; !ok {
			return fmt.Errorf("edge %q has unknown from_node_id %q", edge.EdgeID, edge.FromNodeID)
		}
		if _, ok := nodes[edge.ToNodeID]; !ok {
			return fmt.Errorf("edge %q has unknown to_node_id %q", edge.EdgeID, edge.ToNodeID)
		}
		if edge.AnchorID != "" {
			if _, ok := anchors[edge.AnchorID]; !ok {
				return fmt.Errorf("edge %q references unknown anchor_id %q", edge.EdgeID, edge.AnchorID)
			}
		}
	}

	for _, passage := range result.Passages {
		if strings.TrimSpace(passage.PassageID) == "" {
			return errors.New("passage_id is required")
		}
		if strings.TrimSpace(passage.AnchorID) == "" {
			return fmt.Errorf("passage %q missing anchor_id", passage.PassageID)
		}
		if strings.TrimSpace(passage.SourceRef) == "" {
			return fmt.Errorf("passage %q missing source_ref", passage.PassageID)
		}
		anchor, ok := anchors[passage.AnchorID]
		if !ok {
			return fmt.Errorf("passage %q references unknown anchor_id %q", passage.PassageID, passage.AnchorID)
		}
		if passage.SourceRef != anchor.SourceRef {
			return fmt.Errorf("passage %q source_ref does not match anchor %q", passage.PassageID, passage.AnchorID)
		}
		if _, ok := nodes[passage.NodeID]; !ok {
			return fmt.Errorf("passage %q has unknown node_id %q", passage.PassageID, passage.NodeID)
		}
		if strings.TrimSpace(passage.Text) == "" {
			return fmt.Errorf("passage %q text is empty", passage.PassageID)
		}
	}
	for _, caveat := range result.Caveats {
		if strings.TrimSpace(caveat.Code) == "" {
			return errors.New("caveat code is required")
		}
	}
	return nil
}

func (o Options) withDefaults() Options {
	if strings.TrimSpace(o.ExtractorName) == "" {
		o.ExtractorName = defaultExtractorName
	}
	if strings.TrimSpace(o.ExtractorVersion) == "" {
		o.ExtractorVersion = defaultExtractorVersion
	}
	return o
}

func writeRecord(buffer *bytes.Buffer, record any) error {
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(record)
}

type documentRecord struct {
	RecordType        string            `json:"record_type"`
	DocumentID        string            `json:"document_id"`
	CorpusID          string            `json:"corpus_id"`
	SourceID          string            `json:"source_id"`
	Path              string            `json:"path"`
	Format            extraction.Format `json:"format"`
	DocumentVersionID string            `json:"document_version_id"`
	ContentHash       string            `json:"content_hash"`
	ExtractorName     string            `json:"extractor_name"`
	ExtractorVersion  string            `json:"extractor_version"`
}

type nodeRecord struct {
	RecordType        string              `json:"record_type"`
	NodeID            string              `json:"node_id"`
	CorpusID          string              `json:"corpus_id"`
	DocumentID        string              `json:"document_id"`
	DocumentVersionID string              `json:"document_version_id"`
	NodeKind          extraction.NodeKind `json:"node_kind"`
	ParentNodeID      string              `json:"parent_node_id,omitempty"`
	Ordinal           int                 `json:"ordinal"`
	HeadingPath       string              `json:"heading_path,omitempty"`
	AnchorID          string              `json:"anchor_id,omitempty"`
	TextHash          string              `json:"text_hash,omitempty"`
}

type edgeRecord struct {
	RecordType        string              `json:"record_type"`
	EdgeID            string              `json:"edge_id"`
	CorpusID          string              `json:"corpus_id"`
	DocumentVersionID string              `json:"document_version_id"`
	EdgeKind          extraction.EdgeKind `json:"edge_kind"`
	FromNodeID        string              `json:"from_node_id"`
	ToNodeID          string              `json:"to_node_id"`
	Confidence        string              `json:"confidence"`
	AnchorID          string              `json:"anchor_id,omitempty"`
}

type sourceAnchorRecord struct {
	RecordType        string                   `json:"record_type"`
	AnchorID          string                   `json:"anchor_id"`
	CorpusID          string                   `json:"corpus_id"`
	DocumentID        string                   `json:"document_id"`
	DocumentVersionID string                   `json:"document_version_id"`
	SourceID          string                   `json:"source_id"`
	AnchorType        sourceanchors.AnchorType `json:"anchor_type"`
	SourceRef         string                   `json:"source_ref"`
	Path              string                   `json:"path"`
	HeadingPath       string                   `json:"heading_path,omitempty"`
	LineStart         int                      `json:"line_start,omitempty"`
	LineEnd           int                      `json:"line_end,omitempty"`
	ParagraphIndex    *int                     `json:"paragraph_index,omitempty"`
	JSONPath          string                   `json:"json_path,omitempty"`
	ContentHash       string                   `json:"content_hash"`
	ExtractorVersion  string                   `json:"extractor_version"`
	Caveats           []string                 `json:"caveats,omitempty"`
}

type passageRecord struct {
	RecordType        string `json:"record_type"`
	PassageID         string `json:"passage_id"`
	CorpusID          string `json:"corpus_id"`
	DocumentID        string `json:"document_id"`
	DocumentVersionID string `json:"document_version_id"`
	NodeID            string `json:"node_id"`
	AnchorID          string `json:"anchor_id"`
	SourceRef         string `json:"source_ref"`
	PassageKind       string `json:"passage_kind"`
	Text              string `json:"text"`
	TextHash          string `json:"text_hash"`
}

type caveatRecord struct {
	RecordType        string `json:"record_type"`
	CorpusID          string `json:"corpus_id"`
	DocumentID        string `json:"document_id"`
	DocumentVersionID string `json:"document_version_id"`
	Code              string `json:"code"`
	Message           string `json:"message,omitempty"`
	JSONPath          string `json:"json_path,omitempty"`
	Limit             int    `json:"limit,omitempty"`
	Observed          int    `json:"observed,omitempty"`
}

func documentRecordFrom(document extraction.Document, opts Options) documentRecord {
	return documentRecord{
		RecordType:        "document",
		DocumentID:        strings.TrimSpace(document.DocumentID),
		CorpusID:          strings.TrimSpace(document.CorpusID),
		SourceID:          strings.TrimSpace(document.SourceID),
		Path:              strings.TrimSpace(document.Path),
		Format:            document.Format,
		DocumentVersionID: strings.TrimSpace(document.DocumentVersionID),
		ContentHash:       strings.TrimSpace(document.ContentHash),
		ExtractorName:     strings.TrimSpace(opts.ExtractorName),
		ExtractorVersion:  strings.TrimSpace(opts.ExtractorVersion),
	}
}

func nodeRecordFrom(node extraction.Node) nodeRecord {
	return nodeRecord{
		RecordType:        "node",
		NodeID:            strings.TrimSpace(node.NodeID),
		CorpusID:          strings.TrimSpace(node.CorpusID),
		DocumentID:        strings.TrimSpace(node.DocumentID),
		DocumentVersionID: strings.TrimSpace(node.DocumentVersionID),
		NodeKind:          node.Kind,
		ParentNodeID:      strings.TrimSpace(node.ParentNodeID),
		Ordinal:           node.Ordinal,
		HeadingPath:       strings.TrimSpace(node.HeadingPath),
		AnchorID:          strings.TrimSpace(node.AnchorID),
		TextHash:          strings.TrimSpace(node.TextHash),
	}
}

func edgeRecordFrom(edge extraction.Edge) edgeRecord {
	return edgeRecord{
		RecordType:        "edge",
		EdgeID:            strings.TrimSpace(edge.EdgeID),
		CorpusID:          strings.TrimSpace(edge.CorpusID),
		DocumentVersionID: strings.TrimSpace(edge.DocumentVersionID),
		EdgeKind:          edge.Kind,
		FromNodeID:        strings.TrimSpace(edge.FromNodeID),
		ToNodeID:          strings.TrimSpace(edge.ToNodeID),
		Confidence:        strings.TrimSpace(edge.Confidence),
		AnchorID:          strings.TrimSpace(edge.AnchorID),
	}
}

func sourceAnchorRecordFrom(anchor sourceanchors.Anchor, sourceID string, opts Options) sourceAnchorRecord {
	var paragraphIndex *int
	if anchor.AnchorType == sourceanchors.TypeDOCX {
		value := anchor.ParagraphIndex
		paragraphIndex = &value
	}
	return sourceAnchorRecord{
		RecordType:        "source_anchor",
		AnchorID:          strings.TrimSpace(anchor.AnchorID),
		CorpusID:          strings.TrimSpace(anchor.CorpusID),
		DocumentID:        strings.TrimSpace(anchor.DocumentID),
		DocumentVersionID: strings.TrimSpace(anchor.DocumentVersionID),
		SourceID:          strings.TrimSpace(sourceID),
		AnchorType:        anchor.AnchorType,
		SourceRef:         strings.TrimSpace(anchor.SourceRef),
		Path:              strings.TrimSpace(anchor.Path),
		HeadingPath:       strings.TrimSpace(anchor.HeadingPath),
		LineStart:         anchor.LineStart,
		LineEnd:           anchor.LineEnd,
		ParagraphIndex:    paragraphIndex,
		JSONPath:          strings.TrimSpace(anchor.JSONPath),
		ContentHash:       strings.TrimSpace(anchor.ContentHash),
		ExtractorVersion:  strings.TrimSpace(opts.ExtractorVersion),
		Caveats:           append([]string(nil), anchor.Caveats...),
	}
}

func passageRecordFrom(passage extraction.Passage, anchor sourceanchors.Anchor) passageRecord {
	return passageRecord{
		RecordType:        "retrieval_passage",
		PassageID:         strings.TrimSpace(passage.PassageID),
		CorpusID:          strings.TrimSpace(passage.CorpusID),
		DocumentID:        strings.TrimSpace(passage.DocumentID),
		DocumentVersionID: strings.TrimSpace(passage.DocumentVersionID),
		NodeID:            strings.TrimSpace(passage.NodeID),
		AnchorID:          strings.TrimSpace(passage.AnchorID),
		SourceRef:         strings.TrimSpace(passage.SourceRef),
		PassageKind:       passageKind(anchor),
		Text:              passage.Text,
		TextHash:          strings.TrimSpace(passage.ContentHash),
	}
}

func caveatRecordFrom(document extraction.Document, caveat extraction.Caveat) caveatRecord {
	return caveatRecord{
		RecordType:        "caveat",
		CorpusID:          strings.TrimSpace(document.CorpusID),
		DocumentID:        strings.TrimSpace(document.DocumentID),
		DocumentVersionID: strings.TrimSpace(document.DocumentVersionID),
		Code:              strings.TrimSpace(caveat.Code),
		Message:           strings.TrimSpace(caveat.Message),
		JSONPath:          strings.TrimSpace(caveat.JSONPath),
		Limit:             caveat.Limit,
		Observed:          caveat.Observed,
	}
}

func passageKind(anchor sourceanchors.Anchor) string {
	if anchor.AnchorType == sourceanchors.TypeJSON {
		return passageKindJSONSubtree
	}
	return passageKindStructural
}

func anchorByID(anchors []sourceanchors.Anchor) map[string]sourceanchors.Anchor {
	byID := make(map[string]sourceanchors.Anchor, len(anchors))
	for _, anchor := range anchors {
		byID[anchor.AnchorID] = anchor
	}
	return byID
}

func sortedAnchors(anchors []sourceanchors.Anchor) []sourceanchors.Anchor {
	sorted := append([]sourceanchors.Anchor(nil), anchors...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].SourceRef != sorted[j].SourceRef {
			return sorted[i].SourceRef < sorted[j].SourceRef
		}
		return sorted[i].AnchorID < sorted[j].AnchorID
	})
	return sorted
}

func sortedNodes(nodes []extraction.Node) []extraction.Node {
	sorted := append([]extraction.Node(nil), nodes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Ordinal != sorted[j].Ordinal {
			return sorted[i].Ordinal < sorted[j].Ordinal
		}
		if sorted[i].Kind != sorted[j].Kind {
			return sorted[i].Kind < sorted[j].Kind
		}
		return sorted[i].NodeID < sorted[j].NodeID
	})
	return sorted
}

func sortedEdges(edges []extraction.Edge) []extraction.Edge {
	sorted := append([]extraction.Edge(nil), edges...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].EdgeID < sorted[j].EdgeID
	})
	return sorted
}

func sortedPassages(passages []extraction.Passage) []extraction.Passage {
	sorted := append([]extraction.Passage(nil), passages...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].PassageID < sorted[j].PassageID
	})
	return sorted
}

func sortedCaveats(caveats []extraction.Caveat) []extraction.Caveat {
	sorted := append([]extraction.Caveat(nil), caveats...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Code != sorted[j].Code {
			return sorted[i].Code < sorted[j].Code
		}
		if sorted[i].JSONPath != sorted[j].JSONPath {
			return sorted[i].JSONPath < sorted[j].JSONPath
		}
		return sorted[i].Message < sorted[j].Message
	})
	return sorted
}
