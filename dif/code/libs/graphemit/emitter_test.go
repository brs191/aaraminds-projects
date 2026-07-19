package graphemit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aaraminds/dif/libs/extraction"
)

func TestEmitNDJSONIsByteStableForEveryP0Extractor(t *testing.T) {
	t.Parallel()

	results := []extraction.Result{
		mustExtractMarkdown(t),
		mustExtractText(t),
		mustExtractDOCX(t),
		mustExtractJSON(t),
	}
	for _, result := range results {
		result := result
		t.Run(string(result.Document.Format), func(t *testing.T) {
			t.Parallel()
			first, err := EmitNDJSON(result)
			if err != nil {
				t.Fatalf("emit first NDJSON: %v", err)
			}
			second, err := EmitNDJSON(result)
			if err != nil {
				t.Fatalf("emit second NDJSON: %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("NDJSON output must be byte-stable for unchanged input")
			}
			records := decodeRecords(t, first)
			assertRecordTypePresent(t, records, "document")
			assertRecordTypePresent(t, records, "source_anchor")
			assertRecordTypePresent(t, records, "node")
			assertRecordTypePresent(t, records, "edge")
			assertRecordTypePresent(t, records, "retrieval_passage")
		})
	}
}

func TestEmitNDJSONPreservesDOCXUserFacingSourceRefs(t *testing.T) {
	t.Parallel()

	output, err := EmitNDJSON(mustExtractDOCX(t))
	if err != nil {
		t.Fatalf("emit DOCX NDJSON: %v", err)
	}
	text := string(output)
	if !strings.Contains(text, `"source_ref":"golden-admitted@docver-requirements:docx:requirements.docx#p2"`) {
		t.Fatalf("expected user-facing DOCX source_ref in NDJSON:\n%s", text)
	}
	if strings.Contains(text, "requirements.docx.fixture.json") {
		t.Fatalf("NDJSON must not cite fixture wrapper as DOCX source:\n%s", text)
	}
}

func TestValidateRejectsDanglingEdgesAndUnanchoredPassages(t *testing.T) {
	t.Parallel()

	t.Run("dangling edge", func(t *testing.T) {
		t.Parallel()
		result := mustExtractMarkdown(t)
		result.Edges[0].ToNodeID = "missing-node"
		if _, err := EmitNDJSON(result); err == nil || !strings.Contains(err.Error(), "unknown to_node_id") {
			t.Fatalf("expected dangling edge error, got %v", err)
		}
	})

	t.Run("unanchored passage", func(t *testing.T) {
		t.Parallel()
		result := mustExtractText(t)
		result.Passages[0].AnchorID = ""
		if _, err := EmitNDJSON(result); err == nil || !strings.Contains(err.Error(), "missing anchor_id") {
			t.Fatalf("expected missing anchor error, got %v", err)
		}
	})

	t.Run("passage source ref mismatch", func(t *testing.T) {
		t.Parallel()
		result := mustExtractText(t)
		result.Passages[0].SourceRef = "golden-admitted@docver-runbook:txt:runbook.txt#L1-L1"
		if _, err := EmitNDJSON(result); err == nil || !strings.Contains(err.Error(), "source_ref does not match anchor") {
			t.Fatalf("expected source_ref mismatch error, got %v", err)
		}
	})
}

func TestEmitNDJSONPreservesCaveats(t *testing.T) {
	t.Parallel()

	result := mustExtractJSON(t)
	result.Caveats = append(result.Caveats, extraction.Caveat{
		Code:     extraction.CodeJSONObjectPropsCapped,
		Message:  "maximum object properties reached",
		JSONPath: "$.services",
		Limit:    extraction.JSONMaxObjectProperties,
		Observed: extraction.JSONMaxObjectProperties + 1,
	})
	output, err := EmitNDJSON(result)
	if err != nil {
		t.Fatalf("emit caveated NDJSON: %v", err)
	}
	records := decodeRecords(t, output)
	var found bool
	for _, record := range records {
		if record["record_type"] == "caveat" && record["code"] == extraction.CodeJSONObjectPropsCapped && record["json_path"] == "$.services" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected caveat record in NDJSON:\n%s", output)
	}
}

func mustExtractMarkdown(t *testing.T) extraction.Result {
	t.Helper()
	result, err := extraction.ExtractMarkdown(readGolden(t, "architecture-overview.md"), extraction.Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-architecture-overview",
		DocumentVersionID: "docver-architecture-overview",
		SourceID:          "src-golden-admitted-local",
		Path:              "architecture-overview.md",
	})
	if err != nil {
		t.Fatalf("extract markdown: %v", err)
	}
	return result
}

func mustExtractText(t *testing.T) extraction.Result {
	t.Helper()
	result, err := extraction.ExtractText(readGolden(t, "runbook.txt"), extraction.Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-runbook",
		DocumentVersionID: "docver-runbook",
		SourceID:          "src-golden-admitted-local",
		Path:              "runbook.txt",
	})
	if err != nil {
		t.Fatalf("extract text: %v", err)
	}
	return result
}

func mustExtractDOCX(t *testing.T) extraction.Result {
	t.Helper()
	result, err := extraction.ExtractDOCXParagraphFixture(readGolden(t, "requirements.docx.fixture.json"), extraction.Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-requirements",
		DocumentVersionID: "docver-requirements",
		SourceID:          "src-golden-admitted-local",
		Path:              "requirements.docx",
	})
	if err != nil {
		t.Fatalf("extract DOCX: %v", err)
	}
	return result
}

func mustExtractJSON(t *testing.T) extraction.Result {
	t.Helper()
	result, err := extraction.ExtractJSON(readGolden(t, "service-config.json"), extraction.Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-service-config",
		DocumentVersionID: "docver-service-config",
		SourceID:          "src-golden-admitted-local",
		Path:              "service-config.json",
	})
	if err != nil {
		t.Fatalf("extract JSON: %v", err)
	}
	return result
}

func readGolden(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "evaluation", "golden", "sources", "admitted", name))
	if err != nil {
		t.Fatalf("read golden fixture %s: %v", name, err)
	}
	return string(content)
}

func decodeRecords(t *testing.T, output []byte) []map[string]string {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	records := make([]map[string]string, 0, len(lines))
	for _, line := range lines {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("decode NDJSON record %q: %v", line, err)
		}
		stringsOnly := map[string]string{}
		for key, value := range record {
			if text, ok := value.(string); ok {
				stringsOnly[key] = text
			}
		}
		records = append(records, stringsOnly)
	}
	return records
}

func assertRecordTypePresent(t *testing.T, records []map[string]string, recordType string) {
	t.Helper()
	for _, record := range records {
		if record["record_type"] == recordType {
			return
		}
	}
	t.Fatalf("missing NDJSON record type %q in %+v", recordType, records)
}
