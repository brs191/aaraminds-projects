package render

import (
	"testing"
	"unicode/utf8"
)

// TestModelShort covers BUG 3: truncation must respect UTF-8 rune boundaries
// and must not split a multibyte character. It also verifies prefix stripping
// and the pass-through case for short names.
func TestModelShort(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"short claude prefix stripped", "claude-sonnet", "sonnet"},
		{"short gpt prefix stripped", "gpt-4o", "4o"},
		{"long ascii truncated to 16 runes", "claude-some-really-long-model-name", "some-really-long"},
		// 20 multibyte runes (each 'é' is 2 bytes) after no prefix: must yield
		// exactly 16 runes and remain valid UTF-8 — never a half rune.
		{"multibyte truncated on rune boundary", "éééééééééééééééééééé", "éééééééééééééééé"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := modelShort(c.input)
			if got != c.want {
				t.Errorf("modelShort(%q) = %q, want %q", c.input, got, c.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("modelShort(%q) = %q is not valid UTF-8 (rune split)", c.input, got)
			}
			if n := utf8.RuneCountInString(got); n > 16 {
				t.Errorf("modelShort(%q) returned %d runes, want <= 16", c.input, n)
			}
		})
	}
}
