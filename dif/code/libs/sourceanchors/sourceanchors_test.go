package sourceanchors

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAndFormatSourceRef(t *testing.T) {
	t.Parallel()

	ref, err := ParseSourceRef("golden-admitted@docver-service-config:json:service-config.json#$.services[0].owner")
	if err != nil {
		t.Fatalf("parse source ref: %v", err)
	}
	if ref.CorpusID != "golden-admitted" || ref.DocumentVersionID != "docver-service-config" || ref.AnchorType != TypeJSON {
		t.Fatalf("unexpected parsed source ref: %+v", ref)
	}
	if ref.String() != "golden-admitted@docver-service-config:json:service-config.json#$.services[0].owner" {
		t.Fatalf("unexpected canonical source ref: %q", ref.String())
	}
}

func TestGoldenAnchorsResolveExpectedExcerpts(t *testing.T) {
	t.Parallel()

	catalog := loadGoldenCatalog(t)
	expected := loadExpectedAnchors(t)

	for _, anchor := range expected.Anchors {
		resolved := catalog.ResolveSourceRef(anchor.SourceRef)
		if resolved.Status != StatusResolved {
			t.Fatalf("%s: expected resolved, got %+v", anchor.AnchorAlias, resolved)
		}
		if !contains(resolved.Excerpt, anchor.ExpectedExcerpt) {
			t.Fatalf("%s: expected excerpt %q in %q", anchor.AnchorAlias, anchor.ExpectedExcerpt, resolved.Excerpt)
		}
		if resolved.AnchorID == "" || resolved.ContentHash == "" {
			t.Fatalf("%s: expected anchor_id and content_hash, got %+v", anchor.AnchorAlias, resolved)
		}

		byID := catalog.ResolveAnchorID(resolved.AnchorID)
		if byID.Status != StatusResolved || byID.SourceRef != anchor.SourceRef {
			t.Fatalf("%s: resolve by anchor_id mismatch: %+v", anchor.AnchorAlias, byID)
		}
	}
}

func TestGoldenFailureCasesReturnExplicitStatuses(t *testing.T) {
	t.Parallel()

	catalog := loadGoldenCatalog(t)
	expected := loadExpectedAnchors(t)

	for _, failure := range expected.ResolverFailureCases {
		resolved := catalog.ResolveSourceRef(failure.InputSourceRef)
		if resolved.Status != Status(failure.ExpectedStatus) {
			t.Fatalf("%s: expected %q, got %+v", failure.CaseID, failure.ExpectedStatus, resolved)
		}
		if resolved.Excerpt != "" {
			t.Fatalf("%s: unresolved result exposed excerpt %q", failure.CaseID, resolved.Excerpt)
		}
	}
}

func TestAnchorIDIsDeterministicAcrossRepeatedLoads(t *testing.T) {
	t.Parallel()

	first := loadGoldenCatalog(t)
	second := loadGoldenCatalog(t)
	for sourceRef, firstAnchor := range first.AnchorsBySourceRef {
		secondAnchor, ok := second.AnchorsBySourceRef[sourceRef]
		if !ok {
			t.Fatalf("source_ref missing from second catalog: %s", sourceRef)
		}
		if firstAnchor.AnchorID != secondAnchor.AnchorID {
			t.Fatalf("anchor_id changed for %s: %s != %s", sourceRef, firstAnchor.AnchorID, secondAnchor.AnchorID)
		}
	}
}

func TestContentHashMismatchIsExplicit(t *testing.T) {
	t.Parallel()

	catalog := loadGoldenCatalog(t)
	for sourceRef, anchor := range catalog.AnchorsBySourceRef {
		anchor.ContentHash = "sha256:wrong"
		catalog.AnchorsBySourceRef[sourceRef] = anchor
		catalog.AnchorsByID[anchor.AnchorID] = anchor

		resolved := catalog.ResolveSourceRef(sourceRef)
		if resolved.Status != StatusContentHashMismatch {
			t.Fatalf("expected content_hash_mismatch, got %+v", resolved)
		}
		return
	}
	t.Fatal("expected at least one anchor")
}

type expectedAnchors struct {
	Anchors []struct {
		AnchorAlias     string `json:"anchor_alias"`
		SourceRef       string `json:"source_ref"`
		ExpectedExcerpt string `json:"expected_excerpt"`
	} `json:"anchors"`
	ResolverFailureCases []struct {
		CaseID         string `json:"case_id"`
		InputSourceRef string `json:"input_source_ref"`
		ExpectedStatus string `json:"expected_status"`
	} `json:"resolver_failure_cases"`
}

func loadGoldenCatalog(t *testing.T) Catalog {
	t.Helper()

	catalog, err := LoadGoldenCatalog(expectedAnchorsPath(), filepath.Join("..", "..", "..", "evaluation", "golden", "sources", "admitted"))
	if err != nil {
		t.Fatalf("load golden catalog: %v", err)
	}
	return catalog
}

func loadExpectedAnchors(t *testing.T) expectedAnchors {
	t.Helper()

	content, err := os.ReadFile(expectedAnchorsPath())
	if err != nil {
		t.Fatalf("read expected anchors: %v", err)
	}
	var expected expectedAnchors
	if err := json.Unmarshal(content, &expected); err != nil {
		t.Fatalf("parse expected anchors: %v", err)
	}
	return expected
}

func expectedAnchorsPath() string {
	return filepath.Join("..", "..", "..", "evaluation", "golden", "expected-anchors.json")
}

func contains(value, substring string) bool {
	return strings.Contains(value, substring)
}
