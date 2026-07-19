package extraction

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aaraminds/dif/libs/sourceanchors"
)

func TestJSONExtractorTraversesSortedKeysAndArrayOrder(t *testing.T) {
	t.Parallel()

	result, err := ExtractJSON(`{"z":1,"a":[{"b":2},{"a":1}],"m":"middle"}`, jsonOptions("sorted.json", "doc-sorted", "docver-sorted"))
	if err != nil {
		t.Fatalf("extract json: %v", err)
	}
	paths := jsonAnchorPaths(result)
	expectedPrefix := []string{"$", "$.a", "$.a[0]", "$.a[0].b", "$.a[1]", "$.a[1].a", "$.m", "$.z"}
	if len(paths) < len(expectedPrefix) || !reflect.DeepEqual(paths[:len(expectedPrefix)], expectedPrefix) {
		t.Fatalf("expected sorted JSONPath prefix %+v, got %+v", expectedPrefix, paths)
	}
}

func TestJSONExtractorEmitsJSONPathAnchorsAndResolvablePassages(t *testing.T) {
	t.Parallel()

	content := readGolden(t, "service-config.json")
	opts := jsonOptions("service-config.json", "doc-service-config", "docver-service-config")
	result, err := ExtractJSON(content, opts)
	if err != nil {
		t.Fatalf("extract service config: %v", err)
	}
	assertEveryPassageAnchored(t, result)
	owner := findJSONAnchor(t, result, "$.services[0].owner")
	catalog := sourceanchors.Catalog{}
	if err := catalog.RegisterDocumentVersion(sourceanchors.DocumentVersion{
		DocumentVersionID: opts.DocumentVersionID,
		Sources:           map[string]string{opts.Path: filepath.Join(goldenRoot(), opts.Path)},
	}); err != nil {
		t.Fatalf("register document version: %v", err)
	}
	for _, anchor := range result.Anchors {
		if err := catalog.RegisterAnchor(anchor); err != nil {
			t.Fatalf("register anchor: %v", err)
		}
	}
	resolved := catalog.ResolveAnchorID(owner.AnchorID)
	if resolved.Status != sourceanchors.StatusResolved || resolved.Excerpt != "platform-payments" {
		t.Fatalf("expected JSONPath owner anchor to resolve, got %+v", resolved)
	}
}

func TestJSONCapSpecCoversExpectedCaveatCodes(t *testing.T) {
	t.Parallel()

	spec := readGolden(t, "large-capped.json")
	caveats, err := CaveatsFromJSONCapSpec(spec)
	if err != nil {
		t.Fatalf("generate caveats from spec: %v", err)
	}
	expected := loadExpectedCaveats(t)
	actual := map[string]Caveat{}
	for _, caveat := range caveats {
		actual[caveat.Code] = caveat
	}
	for _, expectedCaveat := range expected.JSONCaveatExpectations {
		if expectedCaveat.Code == CodeJSONParseError {
			continue
		}
		caveat, ok := actual[expectedCaveat.Code]
		if !ok {
			t.Fatalf("missing caveat code %s", expectedCaveat.Code)
		}
		if caveat.JSONPath != expectedCaveat.JSONPath || caveat.Limit != expectedCaveat.Limit || caveat.Observed != expectedCaveat.Observed {
			t.Fatalf("caveat %s mismatch: expected %+v got %+v", expectedCaveat.Code, expectedCaveat, caveat)
		}
	}
}

func TestInvalidAndTooLargeJSONFailClosed(t *testing.T) {
	t.Parallel()

	invalid := readGolden(t, "invalid.json")
	result, err := ExtractJSON(invalid, jsonOptions("invalid.json", "doc-invalid-json", "docver-invalid-json"))
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
	if len(result.Nodes) != 0 || len(result.Anchors) != 0 || len(result.Passages) != 0 {
		t.Fatalf("invalid JSON must not emit partial graph: %+v", result)
	}
	var jsonErr JSONExtractionError
	if !errors.As(err, &jsonErr) || jsonErr.Code != CodeJSONParseError {
		t.Fatalf("expected json_parse_error, got %T %v", err, err)
	}

	tooLarge := strings.Repeat(" ", JSONMaxFileBytes+1)
	result, err = ExtractJSON(tooLarge, jsonOptions("too-large.json", "doc-too-large", "docver-too-large"))
	if err == nil {
		t.Fatal("expected too-large JSON error")
	}
	if len(result.Nodes) != 0 || len(result.Anchors) != 0 || len(result.Passages) != 0 {
		t.Fatalf("too-large JSON must not emit partial graph: %+v", result)
	}
	if !errors.As(err, &jsonErr) || jsonErr.Code != CodeJSONFileTooLarge {
		t.Fatalf("expected json_file_too_large, got %T %v", err, err)
	}

	result, err = ExtractJSON(`{"a":1} {"ignored":true}`, jsonOptions("trailing.json", "doc-trailing", "docver-trailing"))
	if err == nil {
		t.Fatal("expected trailing JSON content to fail closed")
	}
	if len(result.Nodes) != 0 || len(result.Anchors) != 0 || len(result.Passages) != 0 {
		t.Fatalf("trailing JSON content must not emit partial graph: %+v", result)
	}
	if !errors.As(err, &jsonErr) || jsonErr.Code != CodeJSONParseError {
		t.Fatalf("expected json_parse_error for trailing content, got %T %v", err, err)
	}
}

func TestJSONSecretLikeValuesAreRedactedInPassages(t *testing.T) {
	t.Parallel()

	result, err := ExtractJSON(`{"apiKey":"super-secret-value","name":"safe","services":[{"token":"nested-secret"}]}`, jsonOptions("secrets.json", "doc-secrets", "docver-secrets"))
	if err != nil {
		t.Fatalf("extract secret JSON: %v", err)
	}
	for _, passage := range result.Passages {
		if strings.Contains(passage.Text, "super-secret-value") || strings.Contains(passage.Text, "nested-secret") {
			t.Fatalf("secret-like JSON value leaked into passage: %+v", passage)
		}
	}
}

func TestJSONSpecialKeyAnchorsRoundTrip(t *testing.T) {
	t.Parallel()

	content := `{"service-name":{"owner.email":"platform@example.com"}}`
	opts := jsonOptions("special-keys.json", "doc-special", "docver-special")
	result, err := ExtractJSON(content, opts)
	if err != nil {
		t.Fatalf("extract special key JSON: %v", err)
	}
	anchor := findJSONAnchor(t, result, "$['service-name']['owner.email']")
	tmp := t.TempDir()
	path := filepath.Join(tmp, "special-keys.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write special key fixture: %v", err)
	}
	catalog := sourceanchors.Catalog{}
	if err := catalog.RegisterDocumentVersion(sourceanchors.DocumentVersion{
		DocumentVersionID: opts.DocumentVersionID,
		Sources:           map[string]string{opts.Path: path},
	}); err != nil {
		t.Fatalf("register document version: %v", err)
	}
	if err := catalog.RegisterAnchor(anchor); err != nil {
		t.Fatalf("register anchor: %v", err)
	}
	resolved := catalog.ResolveAnchorID(anchor.AnchorID)
	if resolved.Status != sourceanchors.StatusResolved || resolved.Excerpt != "platform@example.com" {
		t.Fatalf("expected special-key JSONPath to resolve, got %+v", resolved)
	}
}

func TestJSONExtractionDeterministicAcrossRepeatedRuns(t *testing.T) {
	t.Parallel()

	content := readGolden(t, "service-config.json")
	opts := jsonOptions("service-config.json", "doc-service-config", "docver-service-config")
	first, err := ExtractJSON(content, opts)
	if err != nil {
		t.Fatalf("first extraction: %v", err)
	}
	second, err := ExtractJSON(content, opts)
	if err != nil {
		t.Fatalf("second extraction: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("JSON extraction must be deterministic across repeated runs")
	}
}

func findJSONAnchor(t *testing.T, result Result, jsonPath string) sourceanchors.Anchor {
	t.Helper()
	for _, anchor := range result.Anchors {
		if anchor.JSONPath == jsonPath {
			return anchor
		}
	}
	t.Fatalf("missing JSON anchor for %s", jsonPath)
	return sourceanchors.Anchor{}
}

func jsonAnchorPaths(result Result) []string {
	paths := make([]string, 0, len(result.Anchors))
	for _, anchor := range result.Anchors {
		paths = append(paths, anchor.JSONPath)
	}
	return paths
}

func jsonOptions(path, documentID, documentVersionID string) Options {
	return Options{
		CorpusID:          "golden-admitted",
		DocumentID:        documentID,
		DocumentVersionID: documentVersionID,
		SourceID:          "src-golden-admitted-local",
		Path:              path,
	}
}

type expectedCaveats struct {
	JSONCaveatExpectations []struct {
		Code     string `json:"code"`
		JSONPath string `json:"json_path"`
		Limit    int    `json:"limit"`
		Observed int    `json:"observed"`
	} `json:"json_caveat_expectations"`
}

func loadExpectedCaveats(t *testing.T) expectedCaveats {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "evaluation", "golden", "expected-caveats.json"))
	if err != nil {
		t.Fatalf("read expected caveats: %v", err)
	}
	var expected expectedCaveats
	if err := json.Unmarshal(content, &expected); err != nil {
		t.Fatalf("parse expected caveats: %v", err)
	}
	return expected
}
