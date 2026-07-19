package extraction

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/aaraminds/dif/libs/sourceanchors"
)

// DOCXParagraphModel is the P0 fixture-backed paragraph model for DOCX
// extraction. A binary DOCX parser can later populate this same model before
// calling ExtractDOCXParagraphModel.
type DOCXParagraphModel struct {
	FixtureType string          `json:"fixture_type"`
	SourcePath  string          `json:"source_path"`
	HeadingPath string          `json:"heading_path"`
	Paragraphs  []DOCXParagraph `json:"paragraphs"`
}

// DOCXParagraph is a deterministic paragraph record extracted from a DOCX
// source.
type DOCXParagraph struct {
	ParagraphIndex int    `json:"paragraph_index"`
	HeadingPath    string `json:"heading_path"`
	Text           string `json:"text"`
}

// ExtractDOCXParagraphFixture decodes the committed P0 DOCX paragraph-model
// fixture and emits user-facing DOCX source refs.
func ExtractDOCXParagraphFixture(content string, opts Options) (Result, error) {
	var model DOCXParagraphModel
	if err := json.Unmarshal([]byte(content), &model); err != nil {
		return Result{}, fmt.Errorf("decode DOCX paragraph model: %w", err)
	}
	return ExtractDOCXParagraphModel(model, opts)
}

// ExtractDOCXParagraphModel emits deterministic document, section, block,
// anchor, passage, and CONTAINS edge records from the P0 DOCX paragraph model.
func ExtractDOCXParagraphModel(model DOCXParagraphModel, opts Options) (Result, error) {
	if err := opts.validate(); err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(model.SourcePath) == "" {
		return Result{}, fmt.Errorf("DOCX paragraph model requires source_path")
	}
	if normalizePath(model.SourcePath) != normalizePath(opts.Path) {
		return Result{}, fmt.Errorf("DOCX source_path %q does not match extraction path %q", model.SourcePath, opts.Path)
	}
	paragraphs, err := normalizedDOCXParagraphs(model.Paragraphs)
	if err != nil {
		return Result{}, err
	}
	if len(paragraphs) == 0 {
		return Result{}, fmt.Errorf("DOCX paragraph model contains no non-empty paragraphs")
	}

	documentText := docxDocumentText(paragraphs)
	result := newBaseResult(opts, FormatDOCX, []string{documentText})
	documentNode := Node{
		NodeID:            stableID("node", opts.CorpusID, opts.DocumentVersionID, string(NodeDocument), normalizePath(opts.Path), "document"),
		CorpusID:          strings.TrimSpace(opts.CorpusID),
		DocumentID:        strings.TrimSpace(opts.DocumentID),
		DocumentVersionID: strings.TrimSpace(opts.DocumentVersionID),
		Kind:              NodeDocument,
		Ordinal:           0,
		HeadingPath:       strings.TrimSpace(model.HeadingPath),
		TextHash:          sourceanchors.ContentHash(documentText),
		Text:              documentText,
	}
	result.Nodes = append(result.Nodes, documentNode)

	sectionByHeading := map[string]string{}
	for _, paragraph := range paragraphs {
		sectionHeading := strings.TrimSpace(paragraph.HeadingPath)
		if sectionHeading == "" {
			sectionHeading = strings.TrimSpace(model.HeadingPath)
		}
		parentID := documentNode.NodeID
		if sectionHeading != "" {
			if existingID := sectionByHeading[sectionHeading]; existingID != "" {
				parentID = existingID
			} else {
				section := Node{
					NodeID:            stableID("node", opts.CorpusID, opts.DocumentVersionID, string(NodeSection), normalizePath(opts.Path), sectionHeading),
					CorpusID:          strings.TrimSpace(opts.CorpusID),
					DocumentID:        strings.TrimSpace(opts.DocumentID),
					DocumentVersionID: strings.TrimSpace(opts.DocumentVersionID),
					Kind:              NodeSection,
					ParentNodeID:      documentNode.NodeID,
					Ordinal:           len(result.Nodes),
					HeadingPath:       sectionHeading,
					TextHash:          sourceanchors.ContentHash(sectionHeading),
					Text:              sectionHeading,
				}
				result.Nodes = append(result.Nodes, section)
				result.addEdge(opts, documentNode.NodeID, section.NodeID, "", len(result.Edges))
				sectionByHeading[sectionHeading] = section.NodeID
				parentID = section.NodeID
			}
		}
		anchor := docxAnchor(opts, paragraph)
		block := Node{
			NodeID:            stableID("node", opts.CorpusID, opts.DocumentVersionID, string(NodeBlock), normalizePath(opts.Path), fmt.Sprint(paragraph.ParagraphIndex)),
			CorpusID:          strings.TrimSpace(opts.CorpusID),
			DocumentID:        strings.TrimSpace(opts.DocumentID),
			DocumentVersionID: strings.TrimSpace(opts.DocumentVersionID),
			Kind:              NodeBlock,
			ParentNodeID:      parentID,
			Ordinal:           len(result.Nodes),
			HeadingPath:       sectionHeading,
			AnchorID:          anchor.AnchorID,
			TextHash:          sourceanchors.ContentHash(paragraph.Text),
			Text:              paragraph.Text,
		}
		result.Anchors = append(result.Anchors, anchor)
		result.Nodes = append(result.Nodes, block)
		result.addEdge(opts, parentID, block.NodeID, anchor.AnchorID, len(result.Edges))
		result.Passages = append(result.Passages, Passage{
			PassageID:         stableID("passage", opts.CorpusID, opts.DocumentVersionID, block.NodeID, anchor.AnchorID),
			CorpusID:          strings.TrimSpace(opts.CorpusID),
			DocumentID:        strings.TrimSpace(opts.DocumentID),
			DocumentVersionID: strings.TrimSpace(opts.DocumentVersionID),
			NodeID:            block.NodeID,
			AnchorID:          anchor.AnchorID,
			SourceRef:         anchor.SourceRef,
			Text:              paragraph.Text,
			ContentHash:       sourceanchors.ContentHash(paragraph.Text),
		})
	}
	return result, nil
}

func normalizedDOCXParagraphs(paragraphs []DOCXParagraph) ([]DOCXParagraph, error) {
	normalized := make([]DOCXParagraph, 0, len(paragraphs))
	seen := map[int]bool{}
	for _, paragraph := range paragraphs {
		if paragraph.ParagraphIndex < 0 {
			return nil, fmt.Errorf("DOCX paragraph_index must be non-negative: %d", paragraph.ParagraphIndex)
		}
		if seen[paragraph.ParagraphIndex] {
			return nil, fmt.Errorf("DOCX paragraph_index %d is duplicated", paragraph.ParagraphIndex)
		}
		if strings.TrimSpace(paragraph.Text) == "" {
			continue
		}
		seen[paragraph.ParagraphIndex] = true
		paragraph.HeadingPath = strings.TrimSpace(paragraph.HeadingPath)
		paragraph.Text = strings.TrimSpace(paragraph.Text)
		normalized = append(normalized, paragraph)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		return normalized[i].ParagraphIndex < normalized[j].ParagraphIndex
	})
	return normalized, nil
}

func docxDocumentText(paragraphs []DOCXParagraph) string {
	parts := make([]string, len(paragraphs))
	for index, paragraph := range paragraphs {
		parts[index] = paragraph.Text
	}
	return strings.Join(parts, "\n")
}

func docxAnchor(opts Options, paragraph DOCXParagraph) sourceanchors.Anchor {
	payload := fmt.Sprintf("%s#p%d", normalizePath(opts.Path), paragraph.ParagraphIndex)
	sourceRef := sourceanchors.SourceRef{
		CorpusID:          strings.TrimSpace(opts.CorpusID),
		DocumentVersionID: strings.TrimSpace(opts.DocumentVersionID),
		AnchorType:        sourceanchors.TypeDOCX,
		Payload:           payload,
	}.String()
	return sourceanchors.Anchor{
		AnchorID:          sourceanchors.ComputeAnchorID(opts.CorpusID, opts.DocumentVersionID, sourceanchors.TypeDOCX, payload),
		CorpusID:          strings.TrimSpace(opts.CorpusID),
		DocumentID:        strings.TrimSpace(opts.DocumentID),
		DocumentVersionID: strings.TrimSpace(opts.DocumentVersionID),
		AnchorType:        sourceanchors.TypeDOCX,
		SourceRef:         sourceRef,
		Path:              normalizePath(opts.Path),
		HeadingPath:       strings.TrimSpace(paragraph.HeadingPath),
		ParagraphIndex:    paragraph.ParagraphIndex,
		ContentHash:       sourceanchors.ContentHash(paragraph.Text),
	}
}
