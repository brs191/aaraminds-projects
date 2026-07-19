package migrations

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadOrderedLoadsSQLMigrationsDeterministically(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "010_second.sql", "CREATE TABLE IF NOT EXISTS dif_meta.second(id text);")
	writeFile(t, dir, "001_first.sql", "CREATE SCHEMA IF NOT EXISTS dif_meta;")
	writeFile(t, dir, "README.md", "ignored")

	got, err := LoadOrdered(dir)
	if err != nil {
		t.Fatalf("expected migrations to load: %v", err)
	}

	names := []string{got[0].Name, got[1].Name}
	want := []string{"001_first.sql", "010_second.sql"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("unexpected migration order: got %v want %v", names, want)
	}
}

func TestLoadOrderedRejectsRIFOwnedDDL(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, dir, "001_bad.sql", "ALTER TABLE rif_meta.method_nodes ADD COLUMN bad text;")

	_, err := LoadOrdered(dir)
	if err == nil {
		t.Fatal("expected RIF-owned DDL to be rejected")
	}
	if !strings.Contains(err.Error(), "RIF-owned") {
		t.Fatalf("expected explicit RIF-owned error, got %q", err.Error())
	}
}

func TestInitialMigrationLoadsAndTargetsDIFOnly(t *testing.T) {
	t.Parallel()

	got, err := LoadOrdered(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatalf("expected component migrations to load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected two SQL migrations, got %d", len(got))
	}
	if got[0].Name != "001_dif_meta_initial.sql" {
		t.Fatalf("unexpected first migration name %q", got[0].Name)
	}
	if got[1].Name != "002_dif_meta_describes_edges.sql" {
		t.Fatalf("unexpected second migration name %q", got[1].Name)
	}
}

func TestParseInventoryOutputAndValidateInventory(t *testing.T) {
	t.Parallel()

	output := strings.Join(append(ExpectedTables, "corpora"), "\n")
	discovered := ParseInventoryOutput(output)
	if err := ValidateInventory(discovered); err != nil {
		t.Fatalf("expected inventory to validate: %v", err)
	}
}

func TestValidateInventoryReportsMissingTables(t *testing.T) {
	t.Parallel()

	err := ValidateInventory([]string{"corpora"})
	if err == nil {
		t.Fatal("expected missing table inventory error")
	}
	if !strings.Contains(err.Error(), "audit_log") || !strings.Contains(err.Error(), "usage_events") {
		t.Fatalf("expected missing table names in error, got %q", err.Error())
	}
}

func TestPSQLRunnerRequiresDatabaseURL(t *testing.T) {
	t.Parallel()

	err := PSQLRunner{MigrationsDir: DefaultDir}.Apply(t.Context())
	if err == nil {
		t.Fatal("expected missing database URL to fail")
	}
	if !strings.Contains(err.Error(), "database URL is required") {
		t.Fatalf("expected explicit database URL error, got %q", err.Error())
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
