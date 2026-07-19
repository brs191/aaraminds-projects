package sourceanchors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadGoldenCatalog loads the P0 source-anchor golden expectations into a
// resolver catalog. It is intended for scaffold validation and component tests.
func LoadGoldenCatalog(expectedAnchorsPath, sourcesRoot string) (Catalog, error) {
	content, err := os.ReadFile(expectedAnchorsPath)
	if err != nil {
		return Catalog{}, fmt.Errorf("read expected anchors %q: %w", expectedAnchorsPath, err)
	}
	var payload struct {
		Anchors []struct {
			AnchorAlias     string `json:"anchor_alias"`
			DocumentAlias   string `json:"document_alias"`
			AnchorType      string `json:"anchor_type"`
			SourceRef       string `json:"source_ref"`
			Path            string `json:"path"`
			HeadingPath     string `json:"heading_path"`
			LineStart       int    `json:"line_start"`
			LineEnd         int    `json:"line_end"`
			ParagraphIndex  int    `json:"paragraph_index"`
			JSONPath        string `json:"json_path"`
			ExpectedExcerpt string `json:"expected_excerpt"`
		} `json:"anchors"`
		ResolverFailureCases []struct {
			InputSourceRef string `json:"input_source_ref"`
			ExpectedStatus string `json:"expected_status"`
		} `json:"resolver_failure_cases"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		return Catalog{}, fmt.Errorf("parse expected anchors %q: %w", expectedAnchorsPath, err)
	}

	catalog := Catalog{}
	anchors := make([]Anchor, 0, len(payload.Anchors))
	for _, item := range payload.Anchors {
		parsed, err := ParseSourceRef(item.SourceRef)
		if err != nil {
			return Catalog{}, err
		}
		shape, ok := parsePayload(parsed.AnchorType, parsed.Payload)
		if !ok {
			return Catalog{}, fmt.Errorf("parse golden anchor payload %q", parsed.Payload)
		}
		sourcePath := normalizePath(firstNonEmpty(item.Path, shape.path))
		actualPath := filepath.Join(sourcesRoot, sourcePath)
		if parsed.AnchorType == TypeDOCX {
			actualPath = filepath.Join(sourcesRoot, "requirements.docx.fixture.json")
		}
		if _, ok := catalog.DocumentVersions[parsed.DocumentVersionID]; !ok {
			if err := catalog.RegisterDocumentVersion(DocumentVersion{
				DocumentVersionID: parsed.DocumentVersionID,
				Sources:           map[string]string{sourcePath: actualPath},
			}); err != nil {
				return Catalog{}, err
			}
		} else {
			version := catalog.DocumentVersions[parsed.DocumentVersionID]
			version.Sources[sourcePath] = actualPath
			catalog.DocumentVersions[parsed.DocumentVersionID] = version
		}

		anchor := Anchor{
			AnchorID:          ComputeAnchorID(parsed.CorpusID, parsed.DocumentVersionID, parsed.AnchorType, parsed.Payload),
			CorpusID:          parsed.CorpusID,
			DocumentID:        strings.TrimSpace(item.DocumentAlias),
			DocumentVersionID: parsed.DocumentVersionID,
			AnchorType:        parsed.AnchorType,
			SourceRef:         parsed.String(),
			Path:              sourcePath,
			HeadingPath:       strings.TrimSpace(item.HeadingPath),
			LineStart:         firstNonZero(item.LineStart, shape.lineStart),
			LineEnd:           firstNonZero(item.LineEnd, shape.lineEnd),
			ParagraphIndex:    firstNonZero(item.ParagraphIndex, shape.paragraphIndex),
			JSONPath:          firstNonEmpty(item.JSONPath, shape.jsonPath),
		}
		anchors = append(anchors, anchor)
	}

	for _, anchor := range anchors {
		excerpt, status := catalog.resolveExcerpt(anchor.DocumentVersionID, anchor.AnchorType, anchor.Path, anchor)
		if status != StatusResolved {
			return Catalog{}, fmt.Errorf("resolve golden anchor %q: %s", anchor.SourceRef, status)
		}
		anchor.ContentHash = ContentHash(excerpt)
		if err := catalog.RegisterAnchor(anchor); err != nil {
			return Catalog{}, err
		}
	}
	for _, failure := range payload.ResolverFailureCases {
		if failure.ExpectedStatus != string(StatusSourceContentUnavailable) {
			continue
		}
		parsed, err := ParseSourceRef(failure.InputSourceRef)
		if err != nil {
			return Catalog{}, err
		}
		shape, ok := parsePayload(parsed.AnchorType, parsed.Payload)
		if !ok {
			return Catalog{}, fmt.Errorf("parse source-content-unavailable payload %q", parsed.Payload)
		}
		if err := catalog.RegisterDocumentVersion(DocumentVersion{
			DocumentVersionID: parsed.DocumentVersionID,
			Sources:           map[string]string{shape.path: filepath.Join(sourcesRoot, shape.path)},
		}); err != nil {
			return Catalog{}, err
		}
	}
	return catalog, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
