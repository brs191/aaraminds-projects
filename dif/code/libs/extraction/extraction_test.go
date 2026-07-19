package extraction

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aaraminds/dif/libs/ingestionruns"
	"github.com/aaraminds/dif/libs/sourceanchors"
)

func TestMarkdownExtractorEmitsDeterministicAnchoredGraph(t *testing.T) {
	t.Parallel()

	content := readGolden(t, "architecture-overview.md")
	opts := Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-architecture-overview",
		DocumentVersionID: "docver-architecture-overview",
		SourceID:          "src-golden-admitted-local",
		Path:              "architecture-overview.md",
	}
	first, err := ExtractMarkdown(content, opts)
	if err != nil {
		t.Fatalf("extract markdown: %v", err)
	}
	second, err := ExtractMarkdown(content, opts)
	if err != nil {
		t.Fatalf("extract markdown second pass: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("markdown extraction must be deterministic across repeated runs")
	}

	assertNodeKinds(t, first, map[NodeKind]int{NodeDocument: 1, NodeSection: 3, NodeBlock: 3})
	assertContainsEdgesValid(t, first)
	assertEveryPassageAnchored(t, first)

	ownership := findBlock(t, first, "Architecture Overview > Ownership", "Platform Architecture")
	if ownership.HeadingPath != "Architecture Overview > Ownership" {
		t.Fatalf("unexpected heading path: %q", ownership.HeadingPath)
	}
	resolved := resolveAnchor(t, first, ownership.AnchorID, "architecture-overview.md")
	if resolved.Status != sourceanchors.StatusResolved {
		t.Fatalf("expected ownership anchor to resolve, got %+v", resolved)
	}
	if !strings.Contains(resolved.Excerpt, "The architecture service is owned by Platform Architecture.") {
		t.Fatalf("resolved excerpt missing ownership text: %q", resolved.Excerpt)
	}

	run := first.IngestionRun("run-md", ingestionruns.StatusCompleted)
	if decision := run.PromotionDecision(); !decision.CanPromote {
		t.Fatalf("healthy markdown extraction should be promotable: %+v", decision)
	}
}

func TestTextExtractorEmitsDocumentBlocksAndAnchors(t *testing.T) {
	t.Parallel()

	content := readGolden(t, "runbook.txt")
	opts := Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-runbook",
		DocumentVersionID: "docver-runbook",
		SourceID:          "src-golden-admitted-local",
		Path:              "runbook.txt",
	}
	result, err := ExtractText(content, opts)
	if err != nil {
		t.Fatalf("extract txt: %v", err)
	}

	assertNodeKinds(t, result, map[NodeKind]int{NodeDocument: 1, NodeBlock: 1})
	assertContainsEdgesValid(t, result)
	assertEveryPassageAnchored(t, result)
	block := findBlock(t, result, "", "Retry the ingestion job once")
	resolved := resolveAnchor(t, result, block.AnchorID, "runbook.txt")
	if resolved.Status != sourceanchors.StatusResolved {
		t.Fatalf("expected TXT anchor to resolve, got %+v", resolved)
	}
	if !strings.Contains(resolved.Excerpt, "Retry the ingestion job once after verifying the source path is reachable.") {
		t.Fatalf("resolved excerpt missing retry text: %q", resolved.Excerpt)
	}
}

func TestDOCXParagraphModelAdapterEmitsUserFacingAnchors(t *testing.T) {
	t.Parallel()

	content := readGolden(t, "requirements.docx.fixture.json")
	opts := Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-requirements",
		DocumentVersionID: "docver-requirements",
		SourceID:          "src-golden-admitted-local",
		Path:              "requirements.docx",
	}
	first, err := ExtractDOCXParagraphFixture(content, opts)
	if err != nil {
		t.Fatalf("extract DOCX paragraph model: %v", err)
	}
	second, err := ExtractDOCXParagraphFixture(content, opts)
	if err != nil {
		t.Fatalf("extract DOCX paragraph model second pass: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("DOCX paragraph-model extraction must be deterministic across repeated runs")
	}

	assertNodeKinds(t, first, map[NodeKind]int{NodeDocument: 1, NodeSection: 3, NodeBlock: 3})
	assertContainsEdgesValid(t, first)
	assertEveryPassageAnchored(t, first)

	approval := findBlock(t, first, "Requirements > Governance", "owner approval")
	if approval.AnchorID == "" {
		t.Fatal("expected DOCX paragraph block to have an anchor")
	}
	for _, anchor := range first.Anchors {
		if strings.Contains(anchor.SourceRef, "fixture.json") || strings.Contains(anchor.Path, "fixture.json") {
			t.Fatalf("DOCX anchor exposed fixture wrapper instead of user-facing DOCX source: %+v", anchor)
		}
		if !strings.Contains(anchor.SourceRef, ":docx:requirements.docx#p") {
			t.Fatalf("DOCX anchor missing user-facing paragraph source ref: %+v", anchor)
		}
	}

	resolved := resolveAnchorWithFilesystemPath(t, first, approval.AnchorID, "requirements.docx", filepath.Join(goldenRoot(), "requirements.docx.fixture.json"))
	if resolved.Status != sourceanchors.StatusResolved {
		t.Fatalf("expected DOCX paragraph anchor to resolve, got %+v", resolved)
	}
	if resolved.Excerpt != "Any production corpus requires owner approval before indexing." {
		t.Fatalf("resolved DOCX paragraph excerpt mismatch: %q", resolved.Excerpt)
	}

	run := first.IngestionRun("run-docx", ingestionruns.StatusCompleted)
	if decision := run.PromotionDecision(); !decision.CanPromote {
		t.Fatalf("healthy DOCX paragraph-model extraction should be promotable: %+v", decision)
	}
}

func TestDOCXParagraphModelSortsByParagraphIndex(t *testing.T) {
	t.Parallel()

	result, err := ExtractDOCXParagraphModel(DOCXParagraphModel{
		SourcePath:  "requirements.docx",
		HeadingPath: "Requirements",
		Paragraphs: []DOCXParagraph{
			{ParagraphIndex: 2, HeadingPath: "Requirements > Later", Text: "Later paragraph."},
			{ParagraphIndex: 0, HeadingPath: "Requirements > Earlier", Text: "Earlier paragraph."},
			{ParagraphIndex: 1, HeadingPath: "Requirements > Middle", Text: "Middle paragraph."},
		},
	}, Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-requirements",
		DocumentVersionID: "docver-requirements",
		SourceID:          "src-golden-admitted-local",
		Path:              "requirements.docx",
	})
	if err != nil {
		t.Fatalf("extract unsorted DOCX paragraph model: %v", err)
	}
	var indexes []int
	for _, anchor := range result.Anchors {
		indexes = append(indexes, anchor.ParagraphIndex)
	}
	if !reflect.DeepEqual(indexes, []int{0, 1, 2}) {
		t.Fatalf("expected paragraph anchors sorted by paragraph_index, got %+v", indexes)
	}
}

func TestDOCXParagraphModelRejectsInvalidFixtureShape(t *testing.T) {
	t.Parallel()

	opts := Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-requirements",
		DocumentVersionID: "docver-requirements",
		SourceID:          "src-golden-admitted-local",
		Path:              "requirements.docx",
	}
	cases := []struct {
		name  string
		model DOCXParagraphModel
	}{
		{
			name:  "fixture source path must match user-facing extraction path",
			model: DOCXParagraphModel{SourcePath: "requirements.docx.fixture.json", Paragraphs: []DOCXParagraph{{ParagraphIndex: 0, Text: "Text."}}},
		},
		{
			name:  "duplicate paragraph index",
			model: DOCXParagraphModel{SourcePath: "requirements.docx", Paragraphs: []DOCXParagraph{{ParagraphIndex: 0, Text: "One."}, {ParagraphIndex: 0, Text: "Two."}}},
		},
		{
			name:  "negative paragraph index",
			model: DOCXParagraphModel{SourcePath: "requirements.docx", Paragraphs: []DOCXParagraph{{ParagraphIndex: -1, Text: "Text."}}},
		},
		{
			name:  "empty paragraph model",
			model: DOCXParagraphModel{SourcePath: "requirements.docx", Paragraphs: []DOCXParagraph{{ParagraphIndex: 0, Text: " \n\t"}}},
		},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ExtractDOCXParagraphModel(testCase.model, opts); err == nil {
				t.Fatal("expected invalid DOCX paragraph model to fail")
			}
		})
	}
}

func TestDegenerateExtractionCountsCannotPromote(t *testing.T) {
	t.Parallel()

	result := Result{
		Document: Document{
			DocumentID:        "doc-empty",
			CorpusID:          "golden-admitted",
			SourceID:          "src-golden-admitted-local",
			DocumentVersionID: "docver-empty",
		},
		Nodes: []Node{{NodeID: "node-empty", Kind: NodeDocument}},
	}
	run := result.IngestionRun("run-empty", ingestionruns.StatusCompleted)
	decision := run.PromotionDecision()
	if decision.CanPromote {
		t.Fatalf("degenerate extraction without anchors/passages must not promote: %+v", decision)
	}
	if decision.Reason != ingestionruns.ReasonDegenerateNoAnchors {
		t.Fatalf("expected no-anchor degenerate reason, got %+v", decision)
	}
}

func TestExtractorsRejectEmptyContent(t *testing.T) {
	t.Parallel()

	opts := Options{
		CorpusID:          "golden-admitted",
		DocumentID:        "doc-empty",
		DocumentVersionID: "docver-empty",
		SourceID:          "src-golden-admitted-local",
		Path:              "empty.txt",
	}
	if _, err := ExtractText(" \n\t", opts); err == nil {
		t.Fatal("expected TXT extractor to reject empty content")
	}
	if _, err := ExtractMarkdown(" \n\t", opts); err == nil {
		t.Fatal("expected Markdown extractor to reject empty content")
	}
}

func assertNodeKinds(t *testing.T, result Result, expected map[NodeKind]int) {
	t.Helper()
	actual := map[NodeKind]int{}
	for _, node := range result.Nodes {
		actual[node.Kind]++
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("node kind counts mismatch: expected %+v, got %+v", expected, actual)
	}
}

func assertContainsEdgesValid(t *testing.T, result Result) {
	t.Helper()
	nodes := map[string]bool{}
	for _, node := range result.Nodes {
		nodes[node.NodeID] = true
	}
	for _, edge := range result.Edges {
		if edge.Kind != EdgeContains {
			t.Fatalf("unexpected edge kind: %+v", edge)
		}
		if !nodes[edge.FromNodeID] || !nodes[edge.ToNodeID] {
			t.Fatalf("edge points to unknown node: %+v", edge)
		}
	}
}

func assertEveryPassageAnchored(t *testing.T, result Result) {
	t.Helper()
	anchors := map[string]bool{}
	for _, anchor := range result.Anchors {
		anchors[anchor.AnchorID] = true
	}
	for _, passage := range result.Passages {
		if passage.AnchorID == "" || passage.SourceRef == "" {
			t.Fatalf("passage missing anchor: %+v", passage)
		}
		if !anchors[passage.AnchorID] {
			t.Fatalf("passage references unknown anchor: %+v", passage)
		}
	}
}

func findBlock(t *testing.T, result Result, headingPath, contains string) Node {
	t.Helper()
	for _, node := range result.Nodes {
		if node.Kind == NodeBlock && node.HeadingPath == headingPath && strings.Contains(node.Text, contains) {
			return node
		}
	}
	t.Fatalf("missing block heading=%q containing %q", headingPath, contains)
	return Node{}
}

func resolveAnchor(t *testing.T, result Result, anchorID, sourcePath string) sourceanchors.Resolution {
	t.Helper()
	return resolveAnchorWithFilesystemPath(t, result, anchorID, sourcePath, filepath.Join(goldenRoot(), sourcePath))
}

func resolveAnchorWithFilesystemPath(t *testing.T, result Result, anchorID, sourcePath, filesystemPath string) sourceanchors.Resolution {
	t.Helper()
	catalog := sourceanchors.Catalog{}
	if err := catalog.RegisterDocumentVersion(sourceanchors.DocumentVersion{
		DocumentVersionID: result.Document.DocumentVersionID,
		Sources: map[string]string{
			sourcePath: filesystemPath,
		},
	}); err != nil {
		t.Fatalf("register document version: %v", err)
	}
	for _, anchor := range result.Anchors {
		if err := catalog.RegisterAnchor(anchor); err != nil {
			t.Fatalf("register anchor: %v", err)
		}
	}
	return catalog.ResolveAnchorID(anchorID)
}

func readGolden(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(goldenRoot(), name))
	if err != nil {
		t.Fatalf("read golden fixture %s: %v", name, err)
	}
	return string(content)
}

func goldenRoot() string {
	return filepath.Join("..", "..", "..", "evaluation", "golden", "sources", "admitted")
}
