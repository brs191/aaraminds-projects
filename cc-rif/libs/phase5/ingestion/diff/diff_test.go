package diff

import "testing"

func TestComputeForcePushShortCircuits(t *testing.T) {
	result, err := Compute("/does/not/matter", "0000000000000000000000000000000000000000", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if !result.ForceReindex {
		t.Fatalf("expected force reindex=true")
	}
}
