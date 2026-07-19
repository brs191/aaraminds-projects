// Package sourceanchors implements DIF canonical source refs and P0 anchor
// resolution.
package sourceanchors

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	TypeMarkdown AnchorType = "md"
	TypeText     AnchorType = "txt"
	TypeDOCX     AnchorType = "docx"
	TypeJSON     AnchorType = "json"

	StatusResolved                 Status = "resolved"
	StatusAnchorNotFound           Status = "anchor_not_found"
	StatusDocumentVersionNotFound  Status = "document_version_not_found"
	StatusSourceContentUnavailable Status = "source_content_unavailable"
	StatusAnchorOutOfRange         Status = "anchor_out_of_range"
	StatusAnchorTypeUnsupported    Status = "anchor_type_unsupported"
	StatusContentHashMismatch      Status = "content_hash_mismatch"
)

var (
	linePayloadPattern = regexp.MustCompile(`^(.+)#L([1-9][0-9]*)-L([1-9][0-9]*)$`)
	docxPayloadPattern = regexp.MustCompile(`^(.+)#p([0-9]+)$`)
	jsonPayloadPattern = regexp.MustCompile(`^(.+)#(\$.*)$`)
)

// AnchorType is a supported source-anchor resolver type.
type AnchorType string

// Status is the explicit source-anchor resolver outcome.
type Status string

// SourceRef is the parsed canonical source reference:
// corpus_id@document_version_id:anchor_type:anchor_payload.
type SourceRef struct {
	CorpusID          string
	DocumentVersionID string
	AnchorType        AnchorType
	Payload           string
}

// Anchor is a persisted source-anchor equivalent for P0 service code.
type Anchor struct {
	AnchorID          string
	CorpusID          string
	DocumentID        string
	DocumentVersionID string
	AnchorType        AnchorType
	SourceRef         string
	Path              string
	HeadingPath       string
	LineStart         int
	LineEnd           int
	ParagraphIndex    int
	JSONPath          string
	ContentHash       string
	Caveats           []string
}

// DocumentVersion maps an immutable document version to the stored source files
// that can resolve anchors for that version.
type DocumentVersion struct {
	DocumentVersionID string
	Sources           map[string]string
}

// Catalog is an in-memory source-anchor catalog. Persistence prompts can back
// the same resolver semantics with dif_meta queries later.
type Catalog struct {
	AnchorsByID        map[string]Anchor
	AnchorsBySourceRef map[string]Anchor
	DocumentVersions   map[string]DocumentVersion
}

// Resolution is the resolver result. Status is always explicit; unresolved
// results do not include raw source content.
type Resolution struct {
	Status            Status
	AnchorID          string
	SourceRef         string
	DocumentVersionID string
	Excerpt           string
	ContentHash       string
	Caveats           []string
}

// ParseSourceRef parses the canonical DIF source-ref string.
func ParseSourceRef(value string) (SourceRef, error) {
	value = strings.TrimSpace(value)
	at := strings.Index(value, "@")
	if at <= 0 || at == len(value)-1 {
		return SourceRef{}, fmt.Errorf("invalid source_ref %q: expected corpus_id@document_version_id:anchor_type:anchor_payload", value)
	}
	corpusID := value[:at]
	rest := value[at+1:]
	parts := strings.SplitN(rest, ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return SourceRef{}, fmt.Errorf("invalid source_ref %q: expected document_version_id:anchor_type:anchor_payload", value)
	}
	return SourceRef{
		CorpusID:          corpusID,
		DocumentVersionID: parts[0],
		AnchorType:        AnchorType(parts[1]),
		Payload:           parts[2],
	}, nil
}

// String returns the canonical DIF source-ref string.
func (r SourceRef) String() string {
	return fmt.Sprintf("%s@%s:%s:%s", strings.TrimSpace(r.CorpusID), strings.TrimSpace(r.DocumentVersionID), strings.TrimSpace(string(r.AnchorType)), strings.TrimSpace(r.Payload))
}

// ComputeAnchorID returns the deterministic ADR-007 anchor ID.
func ComputeAnchorID(corpusID, documentVersionID string, anchorType AnchorType, anchorPayload string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(corpusID) + "\x00" + strings.TrimSpace(documentVersionID) + "\x00" + strings.TrimSpace(string(anchorType)) + "\x00" + normalizePayload(anchorPayload)))
	return hex.EncodeToString(sum[:])
}

// ContentHash returns a stable sha256 hash for a resolved excerpt.
func ContentHash(excerpt string) string {
	sum := sha256.Sum256([]byte(excerpt))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// RegisterAnchor validates and stores anchor in the catalog.
func (c *Catalog) RegisterAnchor(anchor Anchor) error {
	if c.AnchorsByID == nil {
		c.AnchorsByID = map[string]Anchor{}
	}
	if c.AnchorsBySourceRef == nil {
		c.AnchorsBySourceRef = map[string]Anchor{}
	}
	parsed, err := ParseSourceRef(anchor.SourceRef)
	if err != nil {
		return err
	}
	if !isSupported(parsed.AnchorType) {
		return fmt.Errorf("unsupported anchor_type %q", parsed.AnchorType)
	}
	if anchor.AnchorID == "" {
		anchor.AnchorID = ComputeAnchorID(parsed.CorpusID, parsed.DocumentVersionID, parsed.AnchorType, parsed.Payload)
	}
	anchor.CorpusID = parsed.CorpusID
	anchor.DocumentVersionID = parsed.DocumentVersionID
	anchor.AnchorType = parsed.AnchorType
	c.AnchorsByID[anchor.AnchorID] = anchor
	c.AnchorsBySourceRef[anchor.SourceRef] = anchor
	return nil
}

// RegisterDocumentVersion validates and stores source paths for a document version.
func (c *Catalog) RegisterDocumentVersion(version DocumentVersion) error {
	if strings.TrimSpace(version.DocumentVersionID) == "" {
		return errors.New("document_version_id is required")
	}
	if len(version.Sources) == 0 {
		return fmt.Errorf("document_version %q requires at least one source", version.DocumentVersionID)
	}
	if c.DocumentVersions == nil {
		c.DocumentVersions = map[string]DocumentVersion{}
	}
	sources := map[string]string{}
	for sourcePath, filesystemPath := range version.Sources {
		sourcePath = strings.TrimSpace(sourcePath)
		filesystemPath = strings.TrimSpace(filesystemPath)
		if sourcePath == "" || filesystemPath == "" {
			return fmt.Errorf("document_version %q has blank source mapping", version.DocumentVersionID)
		}
		sources[normalizePath(sourcePath)] = filesystemPath
	}
	version.Sources = sources
	c.DocumentVersions[version.DocumentVersionID] = version
	return nil
}

// ResolveAnchorID resolves an anchor by deterministic anchor ID.
func (c Catalog) ResolveAnchorID(anchorID string) Resolution {
	anchor, ok := c.AnchorsByID[strings.TrimSpace(anchorID)]
	if !ok {
		return Resolution{Status: StatusAnchorNotFound}
	}
	return c.resolve(anchor)
}

// ResolveSourceRef resolves an anchor by canonical source ref.
func (c Catalog) ResolveSourceRef(sourceRef string) Resolution {
	parsed, err := ParseSourceRef(sourceRef)
	if err != nil {
		return Resolution{Status: StatusAnchorNotFound, SourceRef: strings.TrimSpace(sourceRef)}
	}
	if !isSupported(parsed.AnchorType) {
		return Resolution{Status: StatusAnchorTypeUnsupported, SourceRef: parsed.String(), DocumentVersionID: parsed.DocumentVersionID}
	}
	if _, ok := c.DocumentVersions[parsed.DocumentVersionID]; !ok {
		return Resolution{Status: StatusDocumentVersionNotFound, SourceRef: parsed.String(), DocumentVersionID: parsed.DocumentVersionID}
	}
	if anchor, ok := c.AnchorsBySourceRef[parsed.String()]; ok {
		return c.resolve(anchor)
	}
	if status := c.unregisteredAnchorStatus(parsed); status != "" {
		return Resolution{Status: status, SourceRef: parsed.String(), DocumentVersionID: parsed.DocumentVersionID}
	}
	return Resolution{Status: StatusAnchorNotFound, SourceRef: parsed.String(), DocumentVersionID: parsed.DocumentVersionID}
}

func (c Catalog) resolve(anchor Anchor) Resolution {
	excerpt, status := c.resolveExcerpt(anchor.DocumentVersionID, anchor.AnchorType, anchor.Path, anchor)
	sourceRef := anchor.SourceRef
	if status != StatusResolved {
		return Resolution{Status: status, AnchorID: anchor.AnchorID, SourceRef: sourceRef, DocumentVersionID: anchor.DocumentVersionID, Caveats: anchor.Caveats}
	}
	contentHash := ContentHash(excerpt)
	if anchor.ContentHash != "" && anchor.ContentHash != contentHash {
		return Resolution{Status: StatusContentHashMismatch, AnchorID: anchor.AnchorID, SourceRef: sourceRef, DocumentVersionID: anchor.DocumentVersionID, Caveats: anchor.Caveats}
	}
	return Resolution{
		Status:            StatusResolved,
		AnchorID:          anchor.AnchorID,
		SourceRef:         sourceRef,
		DocumentVersionID: anchor.DocumentVersionID,
		Excerpt:           excerpt,
		ContentHash:       contentHash,
		Caveats:           append([]string(nil), anchor.Caveats...),
	}
}

func (c Catalog) resolveExcerpt(documentVersionID string, anchorType AnchorType, sourcePath string, anchor Anchor) (string, Status) {
	filesystemPath, status := c.filesystemPath(documentVersionID, sourcePath)
	if status != StatusResolved {
		return "", status
	}
	switch anchorType {
	case TypeMarkdown, TypeText:
		return resolveLineExcerpt(filesystemPath, anchor.LineStart, anchor.LineEnd)
	case TypeDOCX:
		return resolveDOCXExcerpt(filesystemPath, anchor.ParagraphIndex)
	case TypeJSON:
		return resolveJSONExcerpt(filesystemPath, anchor.JSONPath)
	default:
		return "", StatusAnchorTypeUnsupported
	}
}

func (c Catalog) filesystemPath(documentVersionID, sourcePath string) (string, Status) {
	version, ok := c.DocumentVersions[documentVersionID]
	if !ok {
		return "", StatusDocumentVersionNotFound
	}
	path, ok := version.Sources[normalizePath(sourcePath)]
	if !ok {
		return "", StatusSourceContentUnavailable
	}
	if _, err := os.Stat(path); err != nil {
		return "", StatusSourceContentUnavailable
	}
	return path, StatusResolved
}

func (c Catalog) unregisteredAnchorStatus(ref SourceRef) Status {
	shape, ok := parsePayload(ref.AnchorType, ref.Payload)
	if !ok {
		return StatusAnchorNotFound
	}
	anchor := Anchor{
		DocumentVersionID: ref.DocumentVersionID,
		AnchorType:        ref.AnchorType,
		Path:              shape.path,
		LineStart:         shape.lineStart,
		LineEnd:           shape.lineEnd,
		ParagraphIndex:    shape.paragraphIndex,
		JSONPath:          shape.jsonPath,
	}
	_, status := c.resolveExcerpt(ref.DocumentVersionID, ref.AnchorType, shape.path, anchor)
	if status == StatusResolved {
		return StatusAnchorNotFound
	}
	return status
}

func resolveLineExcerpt(path string, lineStart, lineEnd int) (string, Status) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", StatusSourceContentUnavailable
	}
	lines := splitLines(string(content))
	if lineStart < 1 || lineEnd < lineStart || lineEnd > len(lines) {
		return "", StatusAnchorOutOfRange
	}
	return strings.Join(lines[lineStart-1:lineEnd], "\n"), StatusResolved
}

func resolveDOCXExcerpt(path string, paragraphIndex int) (string, Status) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", StatusSourceContentUnavailable
	}
	var fixture struct {
		Paragraphs []struct {
			ParagraphIndex int    `json:"paragraph_index"`
			Text           string `json:"text"`
		} `json:"paragraphs"`
	}
	if err := json.Unmarshal(content, &fixture); err != nil {
		return "", StatusSourceContentUnavailable
	}
	for _, paragraph := range fixture.Paragraphs {
		if paragraph.ParagraphIndex == paragraphIndex {
			return paragraph.Text, StatusResolved
		}
	}
	return "", StatusAnchorOutOfRange
}

func resolveJSONExcerpt(path, jsonPath string) (string, Status) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", StatusSourceContentUnavailable
	}
	var value any
	if err := json.Unmarshal(content, &value); err != nil {
		return "", StatusSourceContentUnavailable
	}
	resolved, status := jsonPathGet(value, jsonPath)
	if status != StatusResolved {
		return "", status
	}
	switch typed := resolved.(type) {
	case map[string]any, []any:
		encoded, err := json.Marshal(canonicalJSON(typed))
		if err != nil {
			return "", StatusSourceContentUnavailable
		}
		return string(encoded), StatusResolved
	case string:
		return typed, StatusResolved
	default:
		return fmt.Sprint(typed), StatusResolved
	}
}

func jsonPathGet(value any, jsonPath string) (any, Status) {
	if !strings.HasPrefix(jsonPath, "$") {
		return nil, StatusAnchorNotFound
	}
	current := value
	position := 1
	for position < len(jsonPath) {
		if jsonPath[position] == '.' {
			end := position + 1
			for end < len(jsonPath) && (jsonPath[end] == '_' || jsonPath[end] >= '0' && jsonPath[end] <= '9' || jsonPath[end] >= 'A' && jsonPath[end] <= 'Z' || jsonPath[end] >= 'a' && jsonPath[end] <= 'z') {
				end++
			}
			if end == position+1 {
				return nil, StatusAnchorNotFound
			}
			key := jsonPath[position+1 : end]
			object, ok := current.(map[string]any)
			if !ok {
				return nil, StatusAnchorNotFound
			}
			next, ok := object[key]
			if !ok {
				return nil, StatusAnchorNotFound
			}
			current = next
			position = end
			continue
		}
		if jsonPath[position] == '[' {
			end := strings.IndexByte(jsonPath[position:], ']')
			if end == -1 {
				return nil, StatusAnchorNotFound
			}
			token := jsonPath[position+1 : position+end]
			if strings.HasPrefix(token, `"`) || strings.HasPrefix(token, `'`) {
				key, ok := parseBracketKey(token)
				if !ok {
					return nil, StatusAnchorNotFound
				}
				object, ok := current.(map[string]any)
				if !ok {
					return nil, StatusAnchorNotFound
				}
				next, ok := object[key]
				if !ok {
					return nil, StatusAnchorNotFound
				}
				current = next
				position = position + end + 1
				continue
			}
			index, err := strconv.Atoi(token)
			if err != nil {
				return nil, StatusAnchorNotFound
			}
			array, ok := current.([]any)
			if !ok || index >= len(array) {
				return nil, StatusAnchorNotFound
			}
			current = array[index]
			position = position + end + 1
			continue
		}
		return nil, StatusAnchorNotFound
	}
	return current, StatusResolved
}

func parseBracketKey(token string) (string, bool) {
	if len(token) < 2 {
		return "", false
	}
	quote := token[0]
	if quote != '\'' && quote != '"' || token[len(token)-1] != quote {
		return "", false
	}
	var builder strings.Builder
	escaped := false
	for _, r := range token[1 : len(token)-1] {
		if escaped {
			builder.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
		} else {
			builder.WriteRune(r)
		}
	}
	return builder.String(), !escaped
}

func canonicalJSON(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		ordered := make(map[string]any, len(typed))
		for _, key := range keys {
			ordered[key] = canonicalJSON(typed[key])
		}
		return ordered
	case []any:
		ordered := make([]any, len(typed))
		for index, item := range typed {
			ordered[index] = canonicalJSON(item)
		}
		return ordered
	default:
		return typed
	}
}

type payloadShape struct {
	path           string
	lineStart      int
	lineEnd        int
	paragraphIndex int
	jsonPath       string
}

func parsePayload(anchorType AnchorType, payload string) (payloadShape, bool) {
	switch anchorType {
	case TypeMarkdown, TypeText:
		match := linePayloadPattern.FindStringSubmatch(payload)
		if match == nil {
			return payloadShape{}, false
		}
		lineStart, _ := strconv.Atoi(match[2])
		lineEnd, _ := strconv.Atoi(match[3])
		return payloadShape{path: normalizePath(match[1]), lineStart: lineStart, lineEnd: lineEnd}, true
	case TypeDOCX:
		match := docxPayloadPattern.FindStringSubmatch(payload)
		if match == nil {
			return payloadShape{}, false
		}
		paragraphIndex, _ := strconv.Atoi(match[2])
		return payloadShape{path: normalizePath(match[1]), paragraphIndex: paragraphIndex}, true
	case TypeJSON:
		match := jsonPayloadPattern.FindStringSubmatch(payload)
		if match == nil {
			return payloadShape{}, false
		}
		return payloadShape{path: normalizePath(match[1]), jsonPath: match[2]}, true
	default:
		return payloadShape{}, false
	}
}

func isSupported(anchorType AnchorType) bool {
	switch anchorType {
	case TypeMarkdown, TypeText, TypeDOCX, TypeJSON:
		return true
	default:
		return false
	}
}

func normalizePayload(payload string) string {
	trimmed := strings.TrimSpace(payload)
	for _, anchorType := range []AnchorType{TypeMarkdown, TypeText, TypeDOCX, TypeJSON} {
		if parsed, ok := parsePayload(anchorType, trimmed); ok {
			switch anchorType {
			case TypeMarkdown, TypeText:
				return fmt.Sprintf("%s#L%d-L%d", parsed.path, parsed.lineStart, parsed.lineEnd)
			case TypeDOCX:
				return fmt.Sprintf("%s#p%d", parsed.path, parsed.paragraphIndex)
			case TypeJSON:
				return parsed.path + "#" + strings.TrimSpace(parsed.jsonPath)
			}
		}
	}
	return normalizePath(trimmed)
}

func normalizePath(path string) string {
	return filepath.ToSlash(strings.TrimSpace(path))
}

func splitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}
