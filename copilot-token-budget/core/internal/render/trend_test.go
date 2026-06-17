package render

import (
	"testing"
	"unicode/utf8"

	"github.com/aaraminds/copilot-token-budget/internal/analytics"
)

// TestDailyBar covers the proportional-bar math: empty for non-positive inputs,
// at-least-one block for any non-zero value, full width at the max, and a clamp
// so it never exceeds maxWidth.
func TestDailyBar(t *testing.T) {
	cases := []struct {
		name     string
		value    float64
		max      float64
		width    int
		wantLen  int // expected number of █ blocks
		validUTF bool
	}{
		{"zero value -> empty", 0, 100, 10, 0, true},
		{"negative value -> empty", -5, 100, 10, 0, true},
		{"zero max -> empty", 50, 0, 10, 0, true},
		{"zero width -> empty", 50, 100, 0, 0, true},
		{"value at max -> full width", 100, 100, 10, 10, true},
		{"half -> half width", 50, 100, 10, 5, true},
		{"tiny non-zero rounds up to 1", 1, 1000, 10, 1, true},
		{"value above max clamps to width", 500, 100, 10, 10, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := dailyBar(c.value, c.max, c.width)
			if n := utf8.RuneCountInString(got); n != c.wantLen {
				t.Errorf("dailyBar(%v,%v,%d) = %q (%d runes), want %d runes",
					c.value, c.max, c.width, got, n, c.wantLen)
			}
			if c.validUTF && !utf8.ValidString(got) {
				t.Errorf("dailyBar(%v,%v,%d) = %q is not valid UTF-8", c.value, c.max, c.width, got)
			}
		})
	}
}

// TestLastBuckets verifies the trailing-window reslice: all when n>=len or n<=0,
// otherwise exactly the last n in order.
func TestLastBuckets(t *testing.T) {
	mk := func(keys ...string) []analytics.Bucket {
		out := make([]analytics.Bucket, len(keys))
		for i, k := range keys {
			out[i] = analytics.Bucket{Key: k}
		}
		return out
	}
	full := mk("a", "b", "c", "d")

	if got := lastBuckets(full, 2); len(got) != 2 || got[0].Key != "c" || got[1].Key != "d" {
		t.Errorf("lastBuckets(full,2) = %v, want [c d]", keysOf(got))
	}
	if got := lastBuckets(full, 0); len(got) != 4 {
		t.Errorf("lastBuckets(full,0) should return all, got %v", keysOf(got))
	}
	if got := lastBuckets(full, 10); len(got) != 4 {
		t.Errorf("lastBuckets(full,10) should return all, got %v", keysOf(got))
	}
	if got := lastBuckets(nil, 3); len(got) != 0 {
		t.Errorf("lastBuckets(nil,3) should be empty, got %v", keysOf(got))
	}
}

func keysOf(b []analytics.Bucket) []string {
	out := make([]string, len(b))
	for i, x := range b {
		out[i] = x.Key
	}
	return out
}
