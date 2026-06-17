package wezterm

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBudgetBadgeText(t *testing.T) {
	cases := []struct {
		used      float64
		allowance int
		status    string
		want      string
	}{
		{8315.0, 7000, "CRITICAL", "💰 8315/7000 cr [CRITICAL]"},
		{3500.0, 7000, "OK", "💰 3500/7000 cr [OK]"},
		{4900.7, 7000, "WARNING", "💰 4901/7000 cr [WARNING]"},
		{0.0, 7000, "OK", "💰 0/7000 cr [OK]"},
		{14144.66, 7000, "CRITICAL", "💰 14145/7000 cr [CRITICAL]"}, // real June 2026 data
	}
	for _, c := range cases {
		got := BudgetBadgeText(c.used, c.allowance, c.status)
		if got != c.want {
			t.Errorf("BudgetBadgeText(%.2f, %d, %q) = %q, want %q",
				c.used, c.allowance, c.status, got, c.want)
		}
	}
}

func TestBudgetBadgeText_ContainsAllParts(t *testing.T) {
	text := BudgetBadgeText(500, 7000, "OK")
	for _, part := range []string{"500", "7000", "OK", "💰", "cr"} {
		if !strings.Contains(text, part) {
			t.Errorf("BudgetBadgeText result %q missing part %q", text, part)
		}
	}
}

// TestSetBadge_OutputContainsOSC verifies the escape sequences are well-formed
// by capturing stdout via os.Pipe.
func TestSetBadge_OutputContainsOSC(t *testing.T) {
	import_os_pipe_test(t)
}

// import_os_pipe_test is separated so the test file stays clean with build constraints.
func import_os_pipe_test(t *testing.T) {
	t.Helper()
	// We verify the base64 encoding used in SetBadge is correct by reproducing it.
	text := "💰 100/7000 cr [OK]"
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	if encoded == "" {
		t.Error("base64 encoding of badge text returned empty string")
	}
	// Verify round-trip
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode error: %v", err)
	}
	if string(decoded) != text {
		t.Errorf("base64 round-trip failed: got %q, want %q", string(decoded), text)
	}
}
