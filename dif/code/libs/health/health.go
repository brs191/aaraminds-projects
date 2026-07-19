// Package health implements DIF runtime health and readiness checks.
package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/aaraminds/dif/libs/migrations"
)

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusReady     Status = "ready"
	StatusNotReady  Status = "not_ready"

	ComponentPostgres  ComponentName = "postgres"
	ComponentDIFSchema ComponentName = "dif_meta_schema"
	ComponentRIF       ComponentName = "rif_compatibility"

	RIFStatusUnknown Status = "unknown"
)

// Status is an explicit health/readiness state.
type Status string

// ComponentName names one checked dependency.
type ComponentName string

// Pinger is implemented by *sql.DB.
type Pinger interface {
	PingContext(context.Context) error
}

// Inspector reads readiness evidence from Postgres.
type Inspector interface {
	ListDIFTables(context.Context) ([]string, error)
	LatestRIFStatus(context.Context, string) (RIFCompatibility, error)
}

// Checker runs health and readiness checks.
type Checker struct {
	DB        Pinger
	Inspector Inspector
	ProjectID string
}

// Handler exposes HTTP health/readiness endpoints over a Checker.
type Handler struct {
	Checker Checker
}

// Report is the health/readiness response shape.
type Report struct {
	Status     Status           `json:"status"`
	Components []Component      `json:"components"`
	RIF        RIFCompatibility `json:"rif_compatibility"`
}

// Component is one checked dependency.
type Component struct {
	Name       ComponentName `json:"name"`
	Status     Status        `json:"status"`
	ErrorClass string        `json:"error_class,omitempty"`
	Message    string        `json:"message,omitempty"`
}

// RIFCompatibility is informational for P0 doc-only mode.
type RIFCompatibility struct {
	Status        Status `json:"status"`
	Informational bool   `json:"informational"`
	ErrorClass    string `json:"error_class,omitempty"`
	Message       string `json:"message,omitempty"`
}

// SQLInspector queries Postgres for schema and RIF status evidence.
type SQLInspector struct {
	DB SQLQueryer
}

// SQLQueryer is implemented by *sql.DB and *sql.Tx.
type SQLQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// Health verifies PostgreSQL connectivity. It is not a static OK.
func (c Checker) Health(ctx context.Context) Report {
	if c.DB == nil {
		return Report{
			Status:     StatusUnhealthy,
			Components: []Component{failure(ComponentPostgres, "db_not_configured", "postgres health dependency is not configured")},
			RIF:        unknownRIF("not_checked", "readiness check was not run"),
		}
	}
	if err := c.DB.PingContext(ctx); err != nil {
		return Report{
			Status:     StatusUnhealthy,
			Components: []Component{failure(ComponentPostgres, "db_unavailable", "postgres connectivity check failed")},
			RIF:        unknownRIF("not_checked", "readiness check was not run"),
		}
	}
	return Report{
		Status:     StatusHealthy,
		Components: []Component{{Name: ComponentPostgres, Status: StatusHealthy}},
		RIF:        unknownRIF("not_checked", "readiness check was not run"),
	}
}

// Readiness verifies PostgreSQL connectivity and dif_meta schema inventory.
// RIF compatibility is included as informational and does not fail P0 doc-only
// readiness.
func (c Checker) Readiness(ctx context.Context) Report {
	components := []Component{}
	if c.DB == nil {
		return Report{
			Status:     StatusNotReady,
			Components: []Component{failure(ComponentPostgres, "db_not_configured", "postgres health dependency is not configured")},
			RIF:        unknownRIF("not_checked", "postgres was not available"),
		}
	}
	if err := c.DB.PingContext(ctx); err != nil {
		return Report{
			Status:     StatusNotReady,
			Components: []Component{failure(ComponentPostgres, "db_unavailable", "postgres connectivity check failed")},
			RIF:        unknownRIF("not_checked", "postgres was not available"),
		}
	}
	components = append(components, Component{Name: ComponentPostgres, Status: StatusHealthy})

	if c.Inspector == nil {
		return Report{
			Status:     StatusNotReady,
			Components: append(components, failure(ComponentDIFSchema, "schema_inspector_not_configured", "dif_meta schema inspector is not configured")),
			RIF:        unknownRIF("not_checked", "schema inspector was not configured"),
		}
	}

	tables, err := c.Inspector.ListDIFTables(ctx)
	if err != nil {
		return Report{
			Status:     StatusNotReady,
			Components: append(components, failure(ComponentDIFSchema, "schema_query_failed", "dif_meta schema inventory check failed")),
			RIF:        unknownRIF("not_checked", "schema inventory failed"),
		}
	}
	if err := migrations.ValidateInventory(tables); err != nil {
		return Report{
			Status:     StatusNotReady,
			Components: append(components, failure(ComponentDIFSchema, "schema_missing", err.Error())),
			RIF:        rifInfo(ctx, c.Inspector, c.ProjectID),
		}
	}
	components = append(components, Component{Name: ComponentDIFSchema, Status: StatusReady})

	return Report{
		Status:     StatusReady,
		Components: components,
		RIF:        rifInfo(ctx, c.Inspector, c.ProjectID),
	}
}

// ServeHealthHTTP returns the liveness/DB-connectivity health report.
func (h Handler) ServeHealthHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.Checker.Health(r.Context()))
}

// ServeReadinessHTTP returns the DB/schema readiness report.
func (h Handler) ServeReadinessHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.Checker.Readiness(r.Context()))
}

// ListDIFTables reads the dif_meta table inventory from Postgres.
func (i SQLInspector) ListDIFTables(ctx context.Context) ([]string, error) {
	if i.DB == nil {
		return nil, errors.New("sql inspector requires a database")
	}
	rows, err := i.DB.QueryContext(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema = 'dif_meta' AND table_type = 'BASE TABLE' ORDER BY table_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}

// LatestRIFStatus reads the most recent RIF compatibility status. Missing RIF
// rows are informational, not a P0 readiness failure.
func (i SQLInspector) LatestRIFStatus(ctx context.Context, projectID string) (RIFCompatibility, error) {
	if i.DB == nil {
		return RIFCompatibility{}, errors.New("sql inspector requires a database")
	}
	rows, err := i.DB.QueryContext(ctx, `
SELECT rif_status
FROM dif_meta.rif_compatibility_status
WHERE project_id = $1
ORDER BY checked_at DESC
LIMIT 1`, strings.TrimSpace(projectID))
	if err != nil {
		return RIFCompatibility{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return RIFCompatibility{}, err
		}
		return RIFCompatibility{
			Status:        "rif_not_deployed",
			Informational: true,
			Message:       "no RIF compatibility status recorded",
		}, nil
	}
	var status string
	if err := rows.Scan(&status); err != nil {
		return RIFCompatibility{}, err
	}
	return RIFCompatibility{Status: Status(strings.TrimSpace(status)), Informational: true}, nil
}

func rifInfo(ctx context.Context, inspector Inspector, projectID string) RIFCompatibility {
	rif, err := inspector.LatestRIFStatus(ctx, projectID)
	if err != nil {
		return unknownRIF("rif_status_unavailable", "RIF compatibility status could not be read")
	}
	if rif.Status == "" {
		rif.Status = RIFStatusUnknown
	}
	rif.Informational = true
	return rif
}

func failure(name ComponentName, class, message string) Component {
	return Component{
		Name:       name,
		Status:     StatusUnhealthy,
		ErrorClass: strings.TrimSpace(class),
		Message:    secretSafe(message),
	}
}

func unknownRIF(class, message string) RIFCompatibility {
	return RIFCompatibility{
		Status:        RIFStatusUnknown,
		Informational: true,
		ErrorClass:    strings.TrimSpace(class),
		Message:       secretSafe(message),
	}
}

func secretSafe(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	for _, marker := range []string{"postgres://", "postgresql://", "password=", "passwd=", "pwd=", "token=", "secret="} {
		if strings.Contains(strings.ToLower(message), marker) {
			return "dependency check failed"
		}
	}
	return fmt.Sprint(message)
}

func writeJSON(w http.ResponseWriter, report Report) {
	w.Header().Set("Content-Type", "application/json")
	code := http.StatusOK
	if report.Status == StatusUnhealthy || report.Status == StatusNotReady {
		code = http.StatusServiceUnavailable
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(report)
}
