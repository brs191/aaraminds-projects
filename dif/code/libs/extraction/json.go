package extraction

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/aaraminds/dif/libs/sourceanchors"
)

const (
	JSONMaxDepth              = 12
	JSONMaxBlocks             = 2000
	JSONMaxObjectProperties   = 200
	JSONMaxArrayElements      = 100
	JSONMaxScalarLength       = 8192
	JSONMaxBlockTextLength    = 16384
	JSONMaxTotalTextBytes     = 5242880
	JSONMaxFileBytes          = 26214400
	CodeJSONDepthCapped       = "json_depth_capped"
	CodeJSONBlockCountCapped  = "json_block_count_capped"
	CodeJSONObjectPropsCapped = "json_object_properties_capped"
	CodeJSONArrayElemsCapped  = "json_array_elements_capped"
	CodeJSONScalarTruncated   = "json_scalar_truncated"
	CodeJSONBlockTruncated    = "json_block_text_truncated"
	CodeJSONTotalTextCapped   = "json_total_text_capped"
	CodeJSONFileTooLarge      = "json_file_too_large"
	CodeJSONParseError        = "json_parse_error"
)

var secretLikeKeys = []string{"password", "secret", "token", "apikey", "clientsecret", "privatekey"}

var simpleJSONPathKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// JSONExtractionError is returned when JSON extraction fails closed and must not
// emit a partial graph.
type JSONExtractionError struct {
	Code    string
	Message string
	Caveats []Caveat
}

// Error returns the stable failure code plus message.
func (e JSONExtractionError) Error() string {
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

// ExtractJSON emits deterministic bounded JSON graph records and JSONPath
// anchors. Invalid or physically too-large JSON fails closed with caveats.
func ExtractJSON(content string, opts Options) (Result, error) {
	if err := opts.validate(); err != nil {
		return Result{}, err
	}
	if len([]byte(content)) > JSONMaxFileBytes {
		return Result{}, JSONExtractionError{
			Code:    CodeJSONFileTooLarge,
			Message: "JSON file exceeds P0 parser size cap",
			Caveats: []Caveat{{
				Code:     CodeJSONFileTooLarge,
				Message:  "JSON file exceeds P0 parser size cap",
				JSONPath: "$",
				Limit:    JSONMaxFileBytes,
				Observed: len([]byte(content)),
			}},
		}
	}
	if err := ensureNonEmptyContent(content); err != nil {
		return Result{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return Result{}, JSONExtractionError{
			Code:    CodeJSONParseError,
			Message: "invalid JSON",
			Caveats: []Caveat{{
				Code:     CodeJSONParseError,
				Message:  "invalid JSON",
				JSONPath: "$",
			}},
		}
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Result{}, JSONExtractionError{
			Code:    CodeJSONParseError,
			Message: "invalid JSON: trailing content after first value",
			Caveats: []Caveat{{
				Code:     CodeJSONParseError,
				Message:  "invalid JSON: trailing content after first value",
				JSONPath: "$",
			}},
		}
	}

	normalized := canonicalJSONString(value)
	result := newBaseResult(opts, FormatJSON, []string{normalized})
	state := jsonState{opts: opts, result: &result}
	documentNode, ok := state.addJSONNode(NodeDocument, "", 0, "$", "JSON path: $\n"+normalized, normalized)
	if !ok {
		return result, nil
	}
	state.walk(value, "$", 1, documentNode.NodeID)
	sortCaveats(result.Caveats)
	return result, nil
}

// CaveatsFromJSONCapSpec returns caveats from the compact golden cap-generator
// fixture without materializing oversized JSON in the repository.
func CaveatsFromJSONCapSpec(content string) ([]Caveat, error) {
	var spec struct {
		FixtureType string `json:"fixture_type"`
		Deep        struct {
			GenerateDepth int `json:"generateDepth"`
		} `json:"deep"`
		Generated struct {
			BlockCount struct {
				GenerateBlocks int `json:"generateBlocks"`
			} `json:"blockCount"`
			ObjectWithManyProperties struct {
				GenerateProperties int `json:"generateProperties"`
			} `json:"objectWithManyProperties"`
			ArrayWithManyElements struct {
				GenerateElements int `json:"generateElements"`
			} `json:"arrayWithManyElements"`
			LongScalar struct {
				GenerateStringLength int `json:"generateStringLength"`
			} `json:"longScalar"`
			LongBlock struct {
				GenerateNormalizedBlockLength int `json:"generateNormalizedBlockLength"`
			} `json:"longBlock"`
			TotalText struct {
				GenerateTotalTextBytes int `json:"generateTotalTextBytes"`
			} `json:"totalText"`
			TooLargeFile struct {
				GenerateFileBytes int `json:"generateFileBytes"`
			} `json:"tooLargeFile"`
		} `json:"generated"`
	}
	if err := json.Unmarshal([]byte(content), &spec); err != nil {
		return nil, err
	}
	if spec.FixtureType != "json_cap_generator_spec" {
		return nil, errors.New("not a JSON cap-generator spec")
	}
	caveats := []Caveat{
		{Code: CodeJSONDepthCapped, Message: "maximum JSON depth reached", JSONPath: "$.deep", Limit: JSONMaxDepth, Observed: spec.Deep.GenerateDepth},
		{Code: CodeJSONBlockCountCapped, Message: "maximum emitted JSON blocks reached", JSONPath: "$.generated.blockCount", Limit: JSONMaxBlocks, Observed: spec.Generated.BlockCount.GenerateBlocks},
		{Code: CodeJSONObjectPropsCapped, Message: "maximum object properties reached", JSONPath: "$.generated.objectWithManyProperties", Limit: JSONMaxObjectProperties, Observed: spec.Generated.ObjectWithManyProperties.GenerateProperties},
		{Code: CodeJSONArrayElemsCapped, Message: "maximum array elements reached", JSONPath: "$.generated.arrayWithManyElements", Limit: JSONMaxArrayElements, Observed: spec.Generated.ArrayWithManyElements.GenerateElements},
		{Code: CodeJSONScalarTruncated, Message: "JSON scalar truncated", JSONPath: "$.generated.longScalar", Limit: JSONMaxScalarLength, Observed: spec.Generated.LongScalar.GenerateStringLength},
		{Code: CodeJSONBlockTruncated, Message: "JSON block text truncated", JSONPath: "$.generated.longBlock", Limit: JSONMaxBlockTextLength, Observed: spec.Generated.LongBlock.GenerateNormalizedBlockLength},
		{Code: CodeJSONTotalTextCapped, Message: "total emitted JSON text capped", JSONPath: "$", Limit: JSONMaxTotalTextBytes, Observed: spec.Generated.TotalText.GenerateTotalTextBytes},
		{Code: CodeJSONFileTooLarge, Message: "JSON file exceeds P0 parser size cap", JSONPath: "$.generated.tooLargeFile", Limit: JSONMaxFileBytes, Observed: spec.Generated.TooLargeFile.GenerateFileBytes},
	}
	sortCaveats(caveats)
	return caveats, nil
}

type jsonState struct {
	opts       Options
	result     *Result
	blockCount int
	totalText  int
}

func (s *jsonState) walk(value any, path string, depth int, parentNodeID string) {
	if depth > JSONMaxDepth {
		s.addCaveat(CodeJSONDepthCapped, "maximum JSON depth reached", path, JSONMaxDepth, depth)
		return
	}
	if s.blockCount >= JSONMaxBlocks {
		s.addCaveat(CodeJSONBlockCountCapped, "maximum emitted JSON blocks reached", path, JSONMaxBlocks, s.blockCount+1)
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		keys := sortedJSONKeys(typed)
		if len(keys) > JSONMaxObjectProperties {
			s.addCaveat(CodeJSONObjectPropsCapped, "maximum object properties reached", path, JSONMaxObjectProperties, len(keys))
			keys = keys[:JSONMaxObjectProperties]
		}
		for _, key := range keys {
			childPath := appendJSONPathKey(path, key)
			child := typed[key]
			nodeKind := NodeBlock
			if path == "$" && isComposite(child) {
				nodeKind = NodeSection
			}
			if nodeKind == NodeBlock && s.blockCount >= JSONMaxBlocks {
				s.addCaveat(CodeJSONBlockCountCapped, "maximum emitted JSON blocks reached", childPath, JSONMaxBlocks, s.blockCount+1)
				continue
			}
			node, ok := s.addJSONNode(nodeKind, parentNodeID, s.nextOrdinal(), childPath, s.blockText(childPath, key, child), jsonExcerpt(child))
			if !ok {
				continue
			}
			s.walk(child, childPath, depth+1, node.NodeID)
		}
	case []any:
		limit := len(typed)
		if limit > JSONMaxArrayElements {
			s.addCaveat(CodeJSONArrayElemsCapped, "maximum array elements reached", path, JSONMaxArrayElements, len(typed))
			limit = JSONMaxArrayElements
		}
		for index := 0; index < limit; index++ {
			childPath := fmt.Sprintf("%s[%d]", path, index)
			child := typed[index]
			if s.blockCount >= JSONMaxBlocks {
				s.addCaveat(CodeJSONBlockCountCapped, "maximum emitted JSON blocks reached", childPath, JSONMaxBlocks, s.blockCount+1)
				continue
			}
			node, ok := s.addJSONNode(NodeBlock, parentNodeID, s.nextOrdinal(), childPath, s.blockText(childPath, strconv.Itoa(index), child), jsonExcerpt(child))
			if !ok {
				continue
			}
			s.walk(child, childPath, depth+1, node.NodeID)
		}
	}
}

func (s *jsonState) addJSONNode(kind NodeKind, parentNodeID string, ordinal int, jsonPath, text, anchorExcerpt string) (Node, bool) {
	if kind == NodeBlock && s.blockCount >= JSONMaxBlocks {
		s.addCaveat(CodeJSONBlockCountCapped, "maximum emitted JSON blocks reached", jsonPath, JSONMaxBlocks, s.blockCount+1)
		return Node{}, false
	}
	if len(text) > JSONMaxBlockTextLength {
		s.addCaveat(CodeJSONBlockTruncated, "JSON block text truncated", jsonPath, JSONMaxBlockTextLength, len(text))
		text = text[:JSONMaxBlockTextLength]
	}
	if s.totalText >= JSONMaxTotalTextBytes {
		s.addCaveat(CodeJSONTotalTextCapped, "total emitted JSON text capped", "$", JSONMaxTotalTextBytes, s.totalText+len(text))
		return Node{}, false
	}
	if s.totalText+len(text) > JSONMaxTotalTextBytes {
		s.addCaveat(CodeJSONTotalTextCapped, "total emitted JSON text capped", "$", JSONMaxTotalTextBytes, s.totalText+len(text))
		text = text[:max(0, JSONMaxTotalTextBytes-s.totalText)]
	}
	s.totalText += len(text)
	payload := normalizePath(s.opts.Path) + "#" + jsonPath
	sourceRef := sourceanchors.SourceRef{
		CorpusID:          s.opts.CorpusID,
		DocumentVersionID: s.opts.DocumentVersionID,
		AnchorType:        sourceanchors.TypeJSON,
		Payload:           payload,
	}.String()
	anchor := sourceanchors.Anchor{
		AnchorID:          sourceanchors.ComputeAnchorID(s.opts.CorpusID, s.opts.DocumentVersionID, sourceanchors.TypeJSON, payload),
		CorpusID:          strings.TrimSpace(s.opts.CorpusID),
		DocumentID:        strings.TrimSpace(s.opts.DocumentID),
		DocumentVersionID: strings.TrimSpace(s.opts.DocumentVersionID),
		AnchorType:        sourceanchors.TypeJSON,
		SourceRef:         sourceRef,
		Path:              normalizePath(s.opts.Path),
		JSONPath:          jsonPath,
		ContentHash:       sourceanchors.ContentHash(anchorExcerpt),
	}
	node := Node{
		NodeID:            stableID("node", s.opts.CorpusID, s.opts.DocumentVersionID, string(kind), normalizePath(s.opts.Path), fmt.Sprint(ordinal), jsonPath),
		CorpusID:          strings.TrimSpace(s.opts.CorpusID),
		DocumentID:        strings.TrimSpace(s.opts.DocumentID),
		DocumentVersionID: strings.TrimSpace(s.opts.DocumentVersionID),
		Kind:              kind,
		ParentNodeID:      parentNodeID,
		Ordinal:           ordinal,
		AnchorID:          anchor.AnchorID,
		TextHash:          sourceanchors.ContentHash(text),
		Text:              text,
	}
	s.result.Anchors = append(s.result.Anchors, anchor)
	s.result.Nodes = append(s.result.Nodes, node)
	if parentNodeID != "" {
		s.result.addEdge(s.opts, parentNodeID, node.NodeID, anchor.AnchorID, len(s.result.Edges))
	}
	if kind == NodeBlock {
		s.result.Passages = append(s.result.Passages, Passage{
			PassageID:         stableID("passage", s.opts.CorpusID, s.opts.DocumentVersionID, node.NodeID, anchor.AnchorID),
			CorpusID:          strings.TrimSpace(s.opts.CorpusID),
			DocumentID:        strings.TrimSpace(s.opts.DocumentID),
			DocumentVersionID: strings.TrimSpace(s.opts.DocumentVersionID),
			NodeID:            node.NodeID,
			AnchorID:          anchor.AnchorID,
			SourceRef:         anchor.SourceRef,
			Text:              text,
			ContentHash:       sourceanchors.ContentHash(text),
		})
		s.blockCount++
	}
	return node, true
}

func (s *jsonState) blockText(jsonPath, key string, value any) string {
	if scalar, ok := value.(string); ok && len(scalar) > JSONMaxScalarLength {
		s.addCaveat(CodeJSONScalarTruncated, "JSON scalar truncated", jsonPath, JSONMaxScalarLength, len(scalar))
	}
	rendered := renderJSONValue(key, value)
	return "JSON path: " + jsonPath + "\n" + rendered
}

func (s *jsonState) nextOrdinal() int {
	return len(s.result.Nodes)
}

func (s *jsonState) addCaveat(code, message, jsonPath string, limit, observed int) {
	s.result.Caveats = append(s.result.Caveats, Caveat{Code: code, Message: message, JSONPath: jsonPath, Limit: limit, Observed: observed})
}

func renderJSONValue(key string, value any) string {
	if isSecretLikeKey(key) {
		return key + ": [REDACTED_SECRET]"
	}
	switch typed := value.(type) {
	case map[string]any, []any:
		return canonicalJSONString(redactJSONValue(typed))
	case string:
		if len(typed) > JSONMaxScalarLength {
			return typed[:JSONMaxScalarLength]
		}
		return key + ": " + typed
	case json.Number:
		return key + ": " + typed.String()
	case bool:
		return key + ": " + strconv.FormatBool(typed)
	case nil:
		return key + ": null"
	default:
		return key + ": " + fmt.Sprint(typed)
	}
}

func redactJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(typed))
		for _, key := range sortedJSONKeys(typed) {
			if isSecretLikeKey(key) {
				redacted[key] = "[REDACTED_SECRET]"
				continue
			}
			redacted[key] = redactJSONValue(typed[key])
		}
		return redacted
	case []any:
		redacted := make([]any, len(typed))
		for index, item := range typed {
			redacted[index] = redactJSONValue(item)
		}
		return redacted
	default:
		return typed
	}
}

func jsonExcerpt(value any) string {
	switch typed := value.(type) {
	case map[string]any, []any:
		return canonicalJSONString(typed)
	case string:
		return typed
	case json.Number:
		return typed.String()
	case bool:
		return strconv.FormatBool(typed)
	case nil:
		return "null"
	default:
		return fmt.Sprint(typed)
	}
}

func canonicalJSONString(value any) string {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(canonicalJSONValue(value))
	return strings.TrimSpace(buf.String())
}

func canonicalJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		keys := sortedJSONKeys(typed)
		ordered := make(map[string]any, len(typed))
		for _, key := range keys {
			ordered[key] = canonicalJSONValue(typed[key])
		}
		return ordered
	case []any:
		items := make([]any, len(typed))
		for index, item := range typed {
			items[index] = canonicalJSONValue(item)
		}
		return items
	default:
		return typed
	}
}

func sortedJSONKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isComposite(value any) bool {
	switch value.(type) {
	case map[string]any, []any:
		return true
	default:
		return false
	}
}

func isSecretLikeKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
	for _, candidate := range secretLikeKeys {
		if strings.Contains(normalized, candidate) {
			return true
		}
	}
	return false
}

func appendJSONPathKey(parent, key string) string {
	if simpleJSONPathKey.MatchString(key) {
		return parent + "." + key
	}
	return parent + "['" + strings.ReplaceAll(strings.ReplaceAll(key, `\`, `\\`), `'`, `\'`) + "']"
}

func sortCaveats(caveats []Caveat) {
	sort.SliceStable(caveats, func(i, j int) bool {
		if caveats[i].Code != caveats[j].Code {
			return caveats[i].Code < caveats[j].Code
		}
		return caveats[i].JSONPath < caveats[j].JSONPath
	})
}
