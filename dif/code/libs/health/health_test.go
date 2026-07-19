package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aaraminds/dif/libs/migrations"
)

func TestHealthVerifiesDatabaseConnectivity(t *testing.T) {
	t.Parallel()

	report := Checker{
		DB: fakePinger{},
	}.Health(context.Background())

	if report.Status != StatusHealthy {
		t.Fatalf("expected healthy report, got %+v", report)
	}
	if len(report.Components) != 1 || report.Components[0].Name != ComponentPostgres || report.Components[0].Status != StatusHealthy {
		t.Fatalf("expected postgres healthy component, got %+v", report.Components)
	}
}

func TestHealthReportsDBUnavailableWithoutLeakingConnectionString(t *testing.T) {
	t.Parallel()

	rawURL := "postgres://user:super-secret@localhost:5432/dif"
	report := Checker{
		DB: fakePinger{err: errors.New("dial " + rawURL + " failed password=super-secret")},
	}.Health(context.Background())

	if report.Status != StatusUnhealthy {
		t.Fatalf("expected unhealthy report, got %+v", report)
	}
	rendered := report.Components[0].Message
	if strings.Contains(rendered, rawURL) || strings.Contains(rendered, "super-secret") || strings.Contains(rendered, "password=") {
		t.Fatalf("health error leaked secret-bearing DB detail: %+v", report)
	}
	if report.Components[0].ErrorClass != "db_unavailable" {
		t.Fatalf("expected db_unavailable, got %+v", report.Components[0])
	}
}

func TestReadinessVerifiesDIFSchemaInventory(t *testing.T) {
	t.Parallel()

	report := Checker{
		DB:        fakePinger{},
		Inspector: fakeInspector{tables: migrations.ExpectedTables, rif: RIFCompatibility{Status: "rif_not_deployed"}},
		ProjectID: "dif-p0-golden",
	}.Readiness(context.Background())

	if report.Status != StatusReady {
		t.Fatalf("expected ready report, got %+v", report)
	}
	if !componentStatus(report.Components, ComponentDIFSchema, StatusReady) {
		t.Fatalf("expected ready dif_meta schema component, got %+v", report.Components)
	}
	if !report.RIF.Informational || report.RIF.Status != "rif_not_deployed" {
		t.Fatalf("expected informational RIF status, got %+v", report.RIF)
	}
}

func TestReadinessReportsSchemaMissing(t *testing.T) {
	t.Parallel()

	report := Checker{
		DB:        fakePinger{},
		Inspector: fakeInspector{tables: []string{"corpora"}},
		ProjectID: "dif-p0-golden",
	}.Readiness(context.Background())

	if report.Status != StatusNotReady {
		t.Fatalf("expected not_ready report, got %+v", report)
	}
	if !componentStatus(report.Components, ComponentDIFSchema, StatusUnhealthy) {
		t.Fatalf("expected unhealthy schema component, got %+v", report.Components)
	}
	if !strings.Contains(report.Components[1].Message, "audit_log") {
		t.Fatalf("expected missing table evidence, got %+v", report.Components[1])
	}
}

func TestRIFStatusIsInformationalNotReadinessFailure(t *testing.T) {
	t.Parallel()

	report := Checker{
		DB: fakePinger{},
		Inspector: fakeInspector{
			tables: migrations.ExpectedTables,
			rifErr: errors.New("relation dif_meta.rif_compatibility_status unavailable"),
		},
		ProjectID: "dif-p0-golden",
	}.Readiness(context.Background())

	if report.Status != StatusReady {
		t.Fatalf("RIF status query failure should not fail P0 readiness, got %+v", report)
	}
	if !report.RIF.Informational || report.RIF.Status != RIFStatusUnknown || report.RIF.ErrorClass != "rif_status_unavailable" {
		t.Fatalf("expected informational unknown RIF status, got %+v", report.RIF)
	}
}

func TestReadinessReportsDBUnavailableBeforeSchemaCheck(t *testing.T) {
	t.Parallel()

	inspector := &recordingInspector{fakeInspector: fakeInspector{tables: migrations.ExpectedTables}}
	report := Checker{
		DB:        fakePinger{err: errors.New("connection refused")},
		Inspector: inspector,
	}.Readiness(context.Background())

	if report.Status != StatusNotReady {
		t.Fatalf("expected not_ready report, got %+v", report)
	}
	if inspector.called {
		t.Fatal("schema inspector was called even though DB ping failed")
	}
}

func TestHTTPReadinessReturnsServiceUnavailableForMissingSchema(t *testing.T) {
	t.Parallel()

	handler := Handler{Checker: Checker{
		DB:        fakePinger{},
		Inspector: fakeInspector{tables: []string{"corpora"}},
	}}
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.ServeReadinessHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected readiness HTTP 503, got %d body=%s", rec.Code, rec.Body.String())
	}
	var report Report
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode readiness report: %v", err)
	}
	if report.Status != StatusNotReady || !componentStatus(report.Components, ComponentDIFSchema, StatusUnhealthy) {
		t.Fatalf("expected not_ready schema report, got %+v", report)
	}
}

type fakePinger struct {
	err error
}

func (p fakePinger) PingContext(context.Context) error {
	return p.err
}

type fakeInspector struct {
	tables []string
	err    error
	rif    RIFCompatibility
	rifErr error
}

func (i fakeInspector) ListDIFTables(context.Context) ([]string, error) {
	return i.tables, i.err
}

func (i fakeInspector) LatestRIFStatus(context.Context, string) (RIFCompatibility, error) {
	if i.rifErr != nil {
		return RIFCompatibility{}, i.rifErr
	}
	if i.rif.Status == "" {
		return RIFCompatibility{Status: "rif_not_deployed", Informational: true}, nil
	}
	i.rif.Informational = true
	return i.rif, nil
}

type recordingInspector struct {
	fakeInspector
	called bool
}

func (i *recordingInspector) ListDIFTables(ctx context.Context) ([]string, error) {
	i.called = true
	return i.fakeInspector.ListDIFTables(ctx)
}

func componentStatus(components []Component, name ComponentName, status Status) bool {
	for _, component := range components {
		if component.Name == name && component.Status == status {
			return true
		}
	}
	return false
}
