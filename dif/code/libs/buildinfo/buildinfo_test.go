package buildinfo

import "testing"

func TestModulePath(t *testing.T) {
	t.Parallel()

	if ModulePath != "github.com/aaraminds/dif" {
		t.Fatalf("unexpected module path: %q", ModulePath)
	}
}
