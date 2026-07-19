package rifcompat

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFixtureVariantsProduceExpectedStatuses(t *testing.T) {
	t.Parallel()

	fixture := loadFixture(t)
	expected := loadExpected(t)
	entities := materializedEntities(fixture)
	variants := variantsByName(fixture)

	for _, expectation := range expected.VariantExpectations {
		expectation := expectation
		t.Run(expectation.Variant, func(t *testing.T) {
			t.Parallel()
			report := Assess(surfaceForVariant(variants[expectation.Variant], entities))
			if report.Status != Status(expectation.RIFStatus) {
				t.Fatalf("expected status %q, got %+v", expectation.RIFStatus, report)
			}
			if string(report.ShadowStatus) != expectation.ShadowStatus {
				t.Fatalf("expected shadow status %q, got %+v", expectation.ShadowStatus, report)
			}
			if len(report.Matches) != expectation.ExpectedMatchCount {
				t.Fatalf("expected %d matches, got %+v", expectation.ExpectedMatchCount, report.Matches)
			}
			for _, caveat := range expectation.RequiredCaveats {
				if !contains(report.Caveats, caveat) {
					t.Fatalf("expected caveat %q in %+v", caveat, report.Caveats)
				}
			}
			for _, missing := range expectation.RequiredMissingCapabilities {
				if !contains(report.MissingCapabilities, missing) {
					t.Fatalf("expected missing capability %q in %+v", missing, report.MissingCapabilities)
				}
			}
		})
	}
}

func TestFixtureLookupsProduceExpectedResults(t *testing.T) {
	t.Parallel()

	fixture := loadFixture(t)
	expected := loadExpected(t)
	entities := materializedEntities(fixture)
	variant := variantsByName(fixture)["age-only-compatible"]
	report := Assess(surfaceForVariant(variant, entities))

	for _, expectation := range expected.LookupExpectations {
		expectation := expectation
		t.Run(expectation.CaseID, func(t *testing.T) {
			t.Parallel()
			actualReport := report
			if expectation.Variant != "age-only-compatible" {
				actualReport = Assess(surfaceForVariant(variantsByName(fixture)[expectation.Variant], entities))
			}
			result := ResolveLookup(actualReport, LookupMode(expectation.Mode), expectation.Input)
			if expectation.ExpectedStatus != "" && string(result.Status) != expectation.ExpectedStatus {
				t.Fatalf("expected status %q, got %+v", expectation.ExpectedStatus, result)
			}
			if expectation.ExpectedConfidence != "" && string(result.Confidence) != expectation.ExpectedConfidence {
				t.Fatalf("expected confidence %q, got %+v", expectation.ExpectedConfidence, result)
			}
			if aliases(result.Matches) == nil {
				t.Fatal("aliases should never be nil")
			}
			if !reflect.DeepEqual(aliases(result.Matches), expectation.ExpectedEntityAliases) {
				t.Fatalf("expected aliases %+v, got %+v", expectation.ExpectedEntityAliases, aliases(result.Matches))
			}
			for _, caveat := range expectation.RequiredCaveats {
				if !contains(result.Caveats, caveat) {
					t.Fatalf("expected caveat %q in %+v", caveat, result.Caveats)
				}
			}
		})
	}
}

func TestNodeAndEdgeIDsUseNULSeparatedAlgorithm(t *testing.T) {
	t.Parallel()

	fixture := loadFixture(t)
	entities := materializedEntities(fixture)
	for _, entity := range entities {
		legacy := testSHA256Text(strings.Join([]string{entity.RepoID, entity.QualifiedName, entity.Kind}, " "))
		if entity.NodeID == legacy {
			t.Fatalf("node ID unexpectedly matched legacy space-separated hash for %+v", entity)
		}
		if entity.NodeID != NodeID(entity.RepoID, entity.QualifiedName, entity.Kind) {
			t.Fatalf("node ID mismatch for %+v", entity)
		}
	}
	relationship := fixture.Relationships[0]
	from := entities[relationship.FromEntityAlias]
	to := entities[relationship.ToEntityAlias]
	got := EdgeID(from.NodeID, relationship.Label, to.NodeID)
	legacy := testSHA256Text(strings.Join([]string{from.NodeID, relationship.Label, to.NodeID}, " "))
	if got == legacy {
		t.Fatal("edge ID unexpectedly matched legacy space-separated hash")
	}
}

func TestCompatibleAGEFallbackIsNotPoisonedByIncompleteShadow(t *testing.T) {
	t.Parallel()

	complete := Entity{
		NodeID:        NodeID("demo-rif", "com.example.Service", "CLASS"),
		RepoID:        "demo-rif",
		Kind:          "CLASS",
		QualifiedName: "com.example.Service",
		SimpleName:    "Service",
		SourceRef:     "demo-rif@1111111111111111111111111111111111111111:src/Service.java:1",
		Origin:        "first_party",
		Confidence:    ConfidenceExact,
	}
	incompleteShadow := complete
	incompleteShadow.SourceRef = ""

	report := Assess(Surface{
		Schemas:         []string{"rif", "rif_meta"},
		ShadowAvailable: true,
		ShadowEntities:  []Entity{incompleteShadow},
		AGEAvailable:    true,
		AGEEntities:     []Entity{complete},
		MissingFields:   []string{"source_ref"},
	})
	if report.Status != StatusCompatible {
		t.Fatalf("expected AGE fallback compatibility, got %+v", report)
	}
	if report.ShadowStatus != ShadowEmpty {
		t.Fatalf("expected empty-shadow fallback status, got %+v", report)
	}
	if len(report.MissingCapabilities) != 0 {
		t.Fatalf("complete AGE fallback should clear fatal missing capabilities, got %+v", report.MissingCapabilities)
	}
}

func TestFatalMissingFieldsIgnoreOptionalShadowWhenAGEIsComplete(t *testing.T) {
	t.Parallel()

	complete := Entity{
		NodeID:        "node",
		RepoID:        "repo",
		Kind:          "CLASS",
		QualifiedName: "pkg.Type",
		SourceRef:     "repo@sha:path:1",
		Origin:        "first_party",
		Confidence:    ConfidenceExact,
	}
	missing := fatalMissingFields([]Entity{}, []Entity{complete}, []string{"source_ref"}, nil)
	if len(missing) != 0 {
		t.Fatalf("complete AGE fallback should not inherit shadow missing fields, got %+v", missing)
	}
}

func TestIncompatibleReportDoesNotReturnSuccessShapedLookup(t *testing.T) {
	t.Parallel()

	result := ResolveLookup(Report{Status: StatusNotDeployed, Caveats: []string{"No RIF compatibility surface is available."}}, LookupQualifiedName, "x")
	if LookupStatus(result.Status) != LookupStatus(StatusNotDeployed) || len(result.Matches) != 0 {
		t.Fatalf("expected explicit non-success status with no matches, got %+v", result)
	}
}

func TestSQLStatusStoreWritesOnlyDIFSchema(t *testing.T) {
	t.Parallel()

	execer := &recordingExecer{}
	store := SQLStatusStore{Execer: execer, DatabaseName: "rif_p19"}
	report := Report{
		Status:              StatusIncompatible,
		MissingCapabilities: []string{"source_ref"},
		Caveats:             []string{"Required RIF compatibility fields are unavailable."},
	}
	if err := store.WriteStatus(context.Background(), "dif-p0-golden", report); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if len(execer.calls) != 1 {
		t.Fatalf("expected one write, got %+v", execer.calls)
	}
	if !strings.Contains(execer.calls[0].query, "dif_meta.rif_compatibility_status") {
		t.Fatalf("expected DIF-owned status write, got %s", execer.calls[0].query)
	}
	if strings.Contains(execer.calls[0].query, " rif.") || strings.Contains(execer.calls[0].query, " rif_meta.") {
		t.Fatalf("status write mutated/read RIF-owned schema: %s", execer.calls[0].query)
	}
}

type recordingExecer struct {
	calls []sqlCall
}

type sqlCall struct {
	query string
	args  []any
}

func (e *recordingExecer) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	e.calls = append(e.calls, sqlCall{query: query, args: append([]any{}, args...)})
	return fakeResult(1), nil
}

type fakeResult int64

func (r fakeResult) LastInsertId() (int64, error) { return 0, driver.ErrSkip }
func (r fakeResult) RowsAffected() (int64, error) { return int64(r), nil }

func surfaceForVariant(variant fixtureVariant, entities map[string]Entity) Surface {
	return Surface{
		Schemas:         variant.Schemas,
		ShadowAvailable: variant.ShadowAvailable,
		ShadowEntities:  entitiesForRefs(variant.ShadowEntities, entities),
		AGEAvailable:    variant.AGEAvailable,
		AGEEntities:     entitiesForRefs(variant.AGEEntities, entities),
		MissingFields:   variant.MissingFields,
	}
}

func entitiesForRefs(refs []entityRef, entities map[string]Entity) []Entity {
	var result []Entity
	for _, ref := range refs {
		entity := entities[ref.EntityAlias]
		omit := map[string]bool{}
		for _, field := range ref.OmitFields {
			omit[field] = true
		}
		if omit["node_id"] {
			entity.NodeID = ""
		}
		if omit["repo_id"] {
			entity.RepoID = ""
		}
		if omit["kind"] {
			entity.Kind = ""
		}
		if omit["qualified_name"] {
			entity.QualifiedName = ""
		}
		if omit["source_ref"] {
			entity.SourceRef = ""
		}
		if omit["origin"] {
			entity.Origin = ""
		}
		if omit["confidence"] {
			entity.Confidence = ""
		}
		result = append(result, entity)
	}
	return result
}

func materializedEntities(fixture compatFixture) map[string]Entity {
	entities := map[string]Entity{}
	for _, item := range fixture.Entities {
		entity := Entity{
			EntityAlias:   item.EntityAlias,
			RepoID:        fixture.RepoID,
			Kind:          item.Kind,
			QualifiedName: item.QualifiedName,
			SimpleName:    item.SimpleName,
			SourceRef:     item.SourceRef,
			Origin:        item.Origin,
			Confidence:    Confidence(item.Confidence),
		}
		entity.NodeID = NodeID(entity.RepoID, entity.QualifiedName, entity.Kind)
		entities[entity.EntityAlias] = entity
	}
	return entities
}

func variantsByName(fixture compatFixture) map[string]fixtureVariant {
	result := map[string]fixtureVariant{}
	for _, variant := range fixture.Variants {
		result[variant.Variant] = variant
	}
	return result
}

func aliases(matches []Entity) []string {
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		result = append(result, match.EntityAlias)
	}
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func loadFixture(t *testing.T) compatFixture {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "evaluation", "fixtures", "rif", "compat_entities.json"))
	if err != nil {
		t.Fatalf("read compat fixture: %v", err)
	}
	var fixture compatFixture
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatalf("parse compat fixture: %v", err)
	}
	return fixture
}

func loadExpected(t *testing.T) expectedResolutions {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "evaluation", "fixtures", "rif", "expected_resolutions.json"))
	if err != nil {
		t.Fatalf("read expected resolutions: %v", err)
	}
	var expected expectedResolutions
	if err := json.Unmarshal(content, &expected); err != nil {
		t.Fatalf("parse expected resolutions: %v", err)
	}
	return expected
}

func testSHA256Text(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

type compatFixture struct {
	RepoID        string           `json:"repo_id"`
	Entities      []fixtureEntity  `json:"entities"`
	Relationships []relationship   `json:"relationships"`
	Variants      []fixtureVariant `json:"variants"`
}

type fixtureEntity struct {
	EntityAlias   string `json:"entity_alias"`
	Kind          string `json:"kind"`
	QualifiedName string `json:"qualified_name"`
	SimpleName    string `json:"simple_name"`
	SourceRef     string `json:"source_ref"`
	Origin        string `json:"origin"`
	Confidence    string `json:"confidence"`
}

type relationship struct {
	FromEntityAlias string `json:"from_entity_alias"`
	ToEntityAlias   string `json:"to_entity_alias"`
	Label           string `json:"label"`
}

type fixtureVariant struct {
	Variant         string      `json:"variant"`
	Schemas         []string    `json:"schemas"`
	AGEAvailable    bool        `json:"age_available"`
	AGEEntities     []entityRef `json:"age_entities"`
	ShadowAvailable bool        `json:"shadow_available"`
	ShadowEntities  []entityRef `json:"shadow_entities"`
	MissingFields   []string    `json:"missing_fields"`
}

type entityRef struct {
	EntityAlias string   `json:"entity_alias"`
	OmitFields  []string `json:"omit_fields"`
}

type expectedResolutions struct {
	VariantExpectations []variantExpectation `json:"variant_expectations"`
	LookupExpectations  []lookupExpectation  `json:"lookup_expectations"`
}

type variantExpectation struct {
	Variant                     string   `json:"variant"`
	RIFStatus                   string   `json:"rif_status"`
	ShadowStatus                string   `json:"shadow_status"`
	ExpectedMatchCount          int      `json:"expected_match_count"`
	RequiredCaveats             []string `json:"required_caveats"`
	RequiredMissingCapabilities []string `json:"required_missing_capabilities"`
}

type lookupExpectation struct {
	CaseID                string   `json:"case_id"`
	Variant               string   `json:"variant"`
	Mode                  string   `json:"mode"`
	Input                 string   `json:"input"`
	ExpectedEntityAliases []string `json:"expected_entity_aliases"`
	ExpectedConfidence    string   `json:"expected_confidence"`
	ExpectedStatus        string   `json:"expected_status"`
	RequiredCaveats       []string `json:"required_caveats"`
}
