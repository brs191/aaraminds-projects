// Package migrations loads and validates DIF-owned SQL migrations.
package migrations

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	DefaultDir       = "migrations"
	DefaultPSQLPath  = "psql"
	DIFSchema        = "dif_meta"
	databaseURLRedax = "[REDACTED_DATABASE_URL]"
)

// ExpectedTables is the P0 dif_meta table inventory from the initial schema design.
var ExpectedTables = []string{
	"audit_log",
	"code_entity_candidates",
	"corpora",
	"document_versions",
	"documents",
	"edges",
	"ingestion_runs",
	"nodes",
	"retrieval_passages",
	"rif_compatibility_status",
	"source_anchors",
	"sources",
	"usage_events",
}

var rifOwnedDDLPattern = regexp.MustCompile(`(?is)\b(create|alter|drop)\s+(schema|table|index|view|materialized\s+view|function|procedure|trigger|sequence)\s+(if\s+(not\s+)?exists\s+)?(rif|rif_meta)(\.|\b)`)

// Migration is one ordered SQL migration file.
type Migration struct {
	Name string
	Path string
	SQL  string
}

// LoadOrdered loads .sql migration files from dir in deterministic filename order.
func LoadOrdered(dir string) ([]Migration, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("migration directory is required")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migration directory %q: %w", dir, err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("no SQL migrations found in %q", dir)
	}

	migrations := make([]Migration, 0, len(names))
	for _, name := range names {
		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", path, err)
		}
		migration := Migration{Name: name, Path: path, SQL: string(content)}
		if err := ValidateDIFOnly(migration); err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}
	return migrations, nil
}

// ValidateDIFOnly rejects migrations that appear to create, alter, or drop RIF-owned schemas.
func ValidateDIFOnly(migration Migration) error {
	if strings.TrimSpace(migration.Name) == "" {
		return errors.New("migration name is required")
	}
	if strings.TrimSpace(migration.SQL) == "" {
		return fmt.Errorf("migration %q is empty", migration.Name)
	}
	if rifOwnedDDLPattern.MatchString(migration.SQL) {
		return fmt.Errorf("migration %q contains DDL targeting RIF-owned schemas", migration.Name)
	}
	return nil
}

// ValidateInventory compares discovered dif_meta tables with the expected P0 inventory.
func ValidateInventory(discovered []string) error {
	expected := sortedCopy(ExpectedTables)
	actual := sortedUnique(discovered)

	var missing []string
	actualSet := map[string]struct{}{}
	for _, table := range actual {
		actualSet[table] = struct{}{}
	}
	for _, table := range expected {
		if _, ok := actualSet[table]; !ok {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing expected %s tables: %s", DIFSchema, strings.Join(missing, ", "))
	}
	return nil
}

// ParseInventoryOutput parses psql -A -t table-name output.
func ParseInventoryOutput(output string) []string {
	lines := strings.Split(output, "\n")
	tables := make([]string, 0, len(lines))
	for _, line := range lines {
		table := strings.TrimSpace(line)
		if table == "" {
			continue
		}
		tables = append(tables, table)
	}
	return sortedUnique(tables)
}

// PSQLRunner applies migrations and checks schema inventory using the local psql binary.
type PSQLRunner struct {
	DatabaseURL   string
	MigrationsDir string
	PSQLPath      string
}

// Apply runs all ordered migrations against the configured database.
func (r PSQLRunner) Apply(ctx context.Context) error {
	if strings.TrimSpace(r.DatabaseURL) == "" {
		return errors.New("database URL is required")
	}
	migrations, err := LoadOrdered(defaultString(r.MigrationsDir, DefaultDir))
	if err != nil {
		return err
	}
	for _, migration := range migrations {
		if err := r.run(ctx, "-v", "ON_ERROR_STOP=1", "-d", r.databaseURL(), "-f", migration.Path); err != nil {
			return fmt.Errorf("apply migration %s: %w", migration.Name, err)
		}
	}
	return nil
}

// CheckInventory verifies the expected dif_meta table inventory exists.
func (r PSQLRunner) CheckInventory(ctx context.Context) error {
	if strings.TrimSpace(r.DatabaseURL) == "" {
		return errors.New("database URL is required")
	}
	output, err := r.output(ctx, "-X", "-A", "-t", "-v", "ON_ERROR_STOP=1", "-d", r.databaseURL(), "-c", inventoryQuery())
	if err != nil {
		return fmt.Errorf("query %s inventory: %w", DIFSchema, err)
	}
	if err := ValidateInventory(ParseInventoryOutput(output)); err != nil {
		return err
	}
	return nil
}

func (r PSQLRunner) run(ctx context.Context, args ...string) error {
	_, err := r.output(ctx, args...)
	return err
}

func (r PSQLRunner) output(ctx context.Context, args ...string) (string, error) {
	command := exec.CommandContext(ctx, defaultString(r.PSQLPath, DefaultPSQLPath), args...)
	out, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("psql failed: %w: %s", err, redactDatabaseURL(string(out), r.DatabaseURL))
	}
	return string(out), nil
}

func (r PSQLRunner) databaseURL() string {
	return strings.TrimSpace(r.DatabaseURL)
}

func inventoryQuery() string {
	return "SELECT table_name FROM information_schema.tables WHERE table_schema = 'dif_meta' AND table_type = 'BASE TABLE' ORDER BY table_name;"
}

func sortedCopy(values []string) []string {
	copied := append([]string(nil), values...)
	sort.Strings(copied)
	return copied
}

func sortedUnique(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func redactDatabaseURL(value, databaseURL string) string {
	if databaseURL == "" {
		return value
	}
	return strings.ReplaceAll(value, databaseURL, databaseURLRedax)
}
