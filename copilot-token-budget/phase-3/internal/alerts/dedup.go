// Package alerts — alert deduplication.
//
// state.json schema: { "thresholdAlerts": { "60": "2026-06-13", "90": "2026-06-14" } }
// Writes are durably atomic: write to state.json.tmp, fsync the temp file, os.Rename
// to state.json, then fsync the parent directory so the rename survives a crash.
// Dedup days are computed in UTC to match the forecast/month math (avoids TZ-change
// re-fires near midnight). File permissions: 0600 (ADR-006).
package alerts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aaraminds/copilot-session-manager/internal/platform"
)

const stateFileName = "state.json"

// alertState is the on-disk JSON schema for the dedup state file.
type alertState struct {
	ThresholdAlerts map[string]string `json:"thresholdAlerts"`
}

// nowFn returns the current time. Replaced in tests to fix the clock.
var nowFn = time.Now

// ShouldAlert returns true when threshold has not already been fired today.
// An absent or stale state file is treated as "not yet alerted" — first-run safe.
func ShouldAlert(threshold int) (bool, error) {
	state, err := loadState()
	if err != nil {
		return false, err
	}
	return shouldAlert(state, threshold, nowFn()), nil
}

// MarkAlerted records that threshold fired today by writing today's date into
// state.json atomically (write-to-tmp then rename). Sets file permissions 0600.
func MarkAlerted(threshold int) error {
	state, err := loadState()
	if err != nil {
		return err
	}
	// UTC day to match the forecast/month math — a TZ change between runs must not
	// re-fire or miss an alert near midnight.
	today := nowFn().UTC().Format("2006-01-02")
	state.ThresholdAlerts[strconv.Itoa(threshold)] = today
	return saveState(state)
}

// shouldAlert is the pure, testable core — accepts an explicit now so tests can
// fix the clock without touching package state.
func shouldAlert(state alertState, threshold int, now time.Time) bool {
	if state.ThresholdAlerts == nil {
		return true
	}
	today := now.UTC().Format("2006-01-02")
	if date, ok := state.ThresholdAlerts[strconv.Itoa(threshold)]; ok && date == today {
		return false
	}
	return true
}

// loadState reads state.json from platform.ConfigDir(). Returns an empty state
// (with initialised map) when the file is absent — first-run safe.
func loadState() (alertState, error) {
	configDir, err := platform.ConfigDir()
	if err != nil {
		return alertState{}, fmt.Errorf("dedup: get config dir: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(configDir, stateFileName))
	if os.IsNotExist(err) {
		return alertState{ThresholdAlerts: make(map[string]string)}, nil
	}
	if err != nil {
		return alertState{}, fmt.Errorf("dedup: read state file: %w", err)
	}

	var state alertState
	if err := json.Unmarshal(data, &state); err != nil {
		// Corrupt state.json (e.g. truncated by a killed write): treat as "not yet alerted"
		// and reset — better to fire a duplicate alert than to permanently silence them.
		// The file will be atomically overwritten on the next MarkAlerted call.
		fmt.Fprintf(os.Stderr, "dedup: corrupt state.json — resetting (error: %v)\n", err)
		return alertState{ThresholdAlerts: make(map[string]string)}, nil
	}
	if state.ThresholdAlerts == nil {
		state.ThresholdAlerts = make(map[string]string)
	}
	return state, nil
}

// saveState writes state durably and atomically to state.json. It writes a .tmp
// intermediary, fsyncs it, renames over the target, then fsyncs the parent directory
// so the rename itself is durable — a crash cannot leave a zero-length state.json.
// Permissions 0600 — never world-readable.
func saveState(state alertState) error {
	configDir, err := platform.ConfigDir()
	if err != nil {
		return fmt.Errorf("dedup: get config dir: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("dedup: marshal state: %w", err)
	}

	statePath := filepath.Join(configDir, stateFileName)
	tmpPath := statePath + ".tmp"

	// Write + fsync the temp file before renaming, so its contents hit the disk.
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("dedup: open temp state file: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("dedup: write temp state file: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("dedup: fsync temp state file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("dedup: close temp state file: %w", err)
	}

	if err := os.Rename(tmpPath, statePath); err != nil {
		_ = os.Remove(tmpPath) // best-effort cleanup on rename failure
		return fmt.Errorf("dedup: atomic rename state file: %w", err)
	}

	// Fsync the parent directory so the rename (a directory metadata change) is durable.
	// Without this, a crash could revert the directory entry to the pre-rename state.
	dir, err := os.Open(configDir)
	if err != nil {
		return fmt.Errorf("dedup: open config dir for fsync: %w", err)
	}
	if err := dir.Sync(); err != nil {
		_ = dir.Close()
		return fmt.Errorf("dedup: fsync config dir: %w", err)
	}
	if err := dir.Close(); err != nil {
		return fmt.Errorf("dedup: close config dir: %w", err)
	}
	return nil
}
