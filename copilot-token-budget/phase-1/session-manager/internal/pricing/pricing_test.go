package pricing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// useTempConfig points platform.ConfigDir at a temp dir for the duration of a
// test by overriding the env vars os.UserConfigDir consults. Returns the
// resolved copilot-token-budget config dir path.
func useTempConfig(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	// Linux/macOS: os.UserConfigDir uses XDG_CONFIG_HOME, else $HOME/.config.
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
	// Windows: os.UserConfigDir uses %AppData%.
	t.Setenv("AppData", tmp)
	return filepath.Join(tmp, "copilot-token-budget")
}

func TestDefault(t *testing.T) {
	c := Default()
	if c.AllowanceCredits != 7000 {
		t.Errorf("AllowanceCredits = %d, want 7000", c.AllowanceCredits)
	}
	cases := []struct {
		model   string
		in, out float64
		window  int64
	}{
		{"sonnet", 300, 1500, 200000},
		{"opus", 500, 2500, 200000},
		{"haiku", 100, 500, 200000},
	}
	for _, tc := range cases {
		r := c.Models[tc.model]
		if r.InputPerMillion != tc.in || r.OutputPerMillion != tc.out || r.ContextWindowTokens != tc.window {
			t.Errorf("%s = %+v, want {%v %v %v}", tc.model, r, tc.in, tc.out, tc.window)
		}
	}
	if c.Default.InputPerMillion != 300 || c.Default.OutputPerMillion != 1500 {
		t.Errorf("Default = %+v, want sonnet rates", c.Default)
	}
}

func TestDefaults_IndependentCopies(t *testing.T) {
	a := defaults()
	a.Models["sonnet"] = ModelRate{InputPerMillion: 999}
	b := defaults()
	if b.Models["sonnet"].InputPerMillion == 999 {
		t.Error("defaults() shares its Models map across calls")
	}
}

func TestRateFor(t *testing.T) {
	c := Default()
	cases := []struct {
		name      string
		model     string
		wantInput float64
	}{
		{"exact sonnet", "sonnet", 300},
		{"copilot sonnet id", "claude-sonnet-4.6", 300},
		{"uppercase opus", "Claude-OPUS-4.8", 500},
		{"haiku substring", "anthropic/claude-3-haiku", 100},
		{"unknown falls to default", "gpt-4o", 300},
		{"empty falls to default", "", 300},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.RateFor(tc.model).InputPerMillion; got != tc.wantInput {
				t.Errorf("RateFor(%q).InputPerMillion = %v, want %v", tc.model, got, tc.wantInput)
			}
		})
	}
}

func TestLoad_NoFile_ReturnsDefaults(t *testing.T) {
	useTempConfig(t)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.AllowanceCredits != 7000 {
		t.Errorf("AllowanceCredits = %d, want bundled 7000", c.AllowanceCredits)
	}
}

func TestLoad_MalformedFile_FallsBack(t *testing.T) {
	dir := useTempConfig(t)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte("{not json"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatalf("Load should not hard-fail on malformed file: %v", err)
	}
	if c.AllowanceCredits != 7000 {
		t.Errorf("AllowanceCredits = %d, want bundled 7000 fallback", c.AllowanceCredits)
	}
}

func TestLoad_MergesOverDefaults(t *testing.T) {
	dir := useTempConfig(t)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	// User overrides allowance and only sonnet's input rate, plus adds a new model.
	override := Config{
		AllowanceCredits: 9000,
		Models: map[string]ModelRate{
			"sonnet": {InputPerMillion: 350},
			"gemini": {InputPerMillion: 75, OutputPerMillion: 200, ContextWindowTokens: 1000000},
		},
	}
	data, _ := json.Marshal(override)
	if err := os.WriteFile(filepath.Join(dir, configFileName), data, 0600); err != nil {
		t.Fatal(err)
	}

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.AllowanceCredits != 9000 {
		t.Errorf("AllowanceCredits = %d, want overridden 9000", c.AllowanceCredits)
	}
	son := c.Models["sonnet"]
	if son.InputPerMillion != 350 {
		t.Errorf("sonnet InputPerMillion = %v, want overridden 350", son.InputPerMillion)
	}
	// Unspecified sonnet fields keep bundled defaults.
	if son.OutputPerMillion != 1500 || son.ContextWindowTokens != 200000 {
		t.Errorf("sonnet partial override clobbered defaults: %+v", son)
	}
	// Untouched models retain defaults.
	if c.Models["opus"].InputPerMillion != 500 {
		t.Errorf("opus = %+v, want bundled defaults", c.Models["opus"])
	}
	// Newly added model is present.
	if c.Models["gemini"].InputPerMillion != 75 {
		t.Errorf("gemini not merged in: %+v", c.Models["gemini"])
	}
}

func TestWriteDefaultIfAbsent(t *testing.T) {
	dir := useTempConfig(t)
	path := filepath.Join(dir, configFileName)

	if err := WriteDefaultIfAbsent(); err != nil {
		t.Fatalf("WriteDefaultIfAbsent: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected file written: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("mode = %o, want 0600", perm)
	}

	// Content round-trips to the bundled defaults.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("written file is not valid json: %v", err)
	}
	if got.AllowanceCredits != 7000 {
		t.Errorf("written AllowanceCredits = %d, want 7000", got.AllowanceCredits)
	}

	// Second call must not overwrite (no-op when present): mutate then re-run.
	if err := os.WriteFile(path, []byte(`{"allowanceCredits":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WriteDefaultIfAbsent(); err != nil {
		t.Fatalf("second WriteDefaultIfAbsent: %v", err)
	}
	data, _ = os.ReadFile(path)
	_ = json.Unmarshal(data, &got)
	if got.AllowanceCredits != 1 {
		t.Errorf("WriteDefaultIfAbsent overwrote an existing file (allowance=%d)", got.AllowanceCredits)
	}
}
