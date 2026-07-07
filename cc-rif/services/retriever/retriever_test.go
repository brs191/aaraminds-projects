package retriever

import (
	"testing"
)

func TestVectorLiteral(t *testing.T) {
	got := vectorLiteral([]float32{1.25, -2})
	if got != "[1.25,-2]" {
		t.Fatalf("unexpected vector literal: %s", got)
	}
}
