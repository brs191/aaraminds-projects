package dif_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoModuleSkeletonRuns(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("go.mod"); err != nil {
		t.Fatalf("expected go.mod to be discoverable from component root: %v", err)
	}
}

func TestInitialMigrationIsDiscoverableFromComponentRoot(t *testing.T) {
	t.Parallel()

	migrationPath := filepath.Join("migrations", "001_dif_meta_initial.sql")
	info, err := os.Stat(migrationPath)
	if err != nil {
		t.Fatalf("expected initial dif_meta migration at %s: %v", migrationPath, err)
	}
	if info.IsDir() {
		t.Fatalf("expected %s to be a SQL migration file, got directory", migrationPath)
	}
}
