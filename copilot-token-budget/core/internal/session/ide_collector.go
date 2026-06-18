// Package session provides IDE Chat session collection from Xodus DB (Nitrite format).
// IDE sessions are collected from ~/.config/github-copilot/ic/ and are stamped with
// Source="ide-chat" and TokenCostSource="estimated" (Phase 6 limitation: no real token cost).
//
// Two collection strategies are employed:
// 1. (Primary) Nitrite SDK parsing of binary DB for per-turn granularity.
// 2. (Fallback) JSON metadata parsing from ~/.copilot/vscode.session.metadata.cache.json.
//
// If the IDE DB is absent, the fallback JSON is consulted. If both fail, an error is
// returned (not nil, not empty slice) to distinguish "no IDE sessions" from "cannot read IDE".
package session

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// ideCollectorImpl is the real IDE Chat session collector. It must be hermetic:
// it reads only from ~/.config/github-copilot/ic/ (Nitrite DB) or the JSON metadata
// fallback, and never modifies those sources.
type ideCollectorImpl struct {
	ideDBPath      string
	metadataPath   string
	preferNitrite  bool // if true, try Nitrite first; fallback to JSON metadata
}

// newIDECollector returns a new IDE collector. It is exported only for testing.
func newIDECollector() *ideCollectorImpl {
	home, _ := os.UserHomeDir()
	return &ideCollectorImpl{
		// Primary source: Nitrite DB under ~/.config/github-copilot/ic/
		ideDBPath:      filepath.Join(home, ".config/github-copilot/ic"),
		// Fallback source: JSON metadata cache
		metadataPath:   filepath.Join(home, ".copilot/vscode.session.metadata.cache.json"),
		preferNitrite:  true,
	}
}

// Collect implements the Collector interface by reading VS Code Copilot Chat sessions
// from Xodus DB (via Nitrite SDK) or falling back to JSON metadata.
//
// Returns error (not nil) if:
// - IDE DB path does not exist AND metadata path does not exist
// - Metadata path exists but cannot be parsed
// Does NOT return error if:
// - IDE DB exists but Nitrite SDK is unavailable (falls back to metadata or zero sessions)
// - Both DB and metadata are empty
func (c *ideCollectorImpl) Collect() ([]Session, error) {
	// Check if IDE DB path exists (permissive; may be Nitrite or may require fallback)
	_, dbErr := os.Stat(c.ideDBPath)
	dbExists := dbErr == nil

	// Check if metadata fallback exists
	_, metaErr := os.Stat(c.metadataPath)
	metaExists := metaErr == nil

	// If neither source exists, return error (not nil/nil, not nil/[]Session)
	if !dbExists && !metaExists {
		return nil, fmt.Errorf("session: IDE data sources not found: %s or %s", c.ideDBPath, c.metadataPath)
	}

	// Try primary source: Nitrite SDK (if available and DB exists)
	if c.preferNitrite && dbExists {
		sessions, err := c.collectFromNitrite()
		if err == nil && len(sessions) > 0 {
			// Success: return Nitrite sessions
			return sessions, nil
		}
		// Nitrite failed; log and fall through to metadata
		if err != nil {
			log.Printf("session: Nitrite SDK collection failed: %v (falling back to metadata)", err)
		}
	}

	// Fall back to metadata-only parsing (returns error only if metadata is unparseable)
	if metaExists {
		sessions, err := c.collectFromMetadata()
		if err != nil {
			return nil, fmt.Errorf("session: IDE metadata parsing failed: %w", err)
		}
		return sessions, nil
	}

	// DB exists but no SDK and no metadata; return empty (not an error)
	return nil, nil
}

// collectFromNitrite attempts to parse IDE sessions from Nitrite DB using the SDK.
// This is a placeholder that will be filled in once the Nitrite SDK version/API is confirmed.
//
// Expected Nitrite SDK: github.com/noelyoo/go-nitrite or equivalent
// Expected collections: ChatSessions, EditSessions, ChatAgents (from Phase 6.0 discovery)
// Expected fields: id, startTime, endTime, model, turnCount, etc. (to be verified with real schema)
//
// TODO(Phase 6.2): Integrate real Nitrite SDK. For now, this is a no-op that returns empty.
func (c *ideCollectorImpl) collectFromNitrite() ([]Session, error) {
	// PLACEHOLDER: Real Nitrite SDK integration here
	// 1. Open DB: db, err := nitrite.Open(c.ideDBPath)
	// 2. Query collections: ChatSessions, EditSessions, ChatAgents
	// 3. Extract sessions with metadata (timestamps, model, etc.)
	// 4. Return sessions with Source="ide-chat", TokenCostSource="estimated", empty Tokens

	// For Phase 6.2, return empty (no sessions) to signal fallback to metadata
	return nil, fmt.Errorf("Nitrite SDK not yet integrated")
}

// collectFromMetadata parses IDE sessions from ~/.copilot/vscode.session.metadata.cache.json.
// This JSON file contains basic metadata for VS Code Copilot Chat sessions: timestamps,
// workspace paths, but NOT token counts. It is the fallback when Nitrite SDK is unavailable.
//
// JSON schema (from Phase 6.0 discovery):
// {
//   "<sessionId>": {
//     "origin": "other",
//     "created": 1780896909035,        // Unix epoch milliseconds
//     "modified": 1780898895790,       // Unix epoch milliseconds
//     "writtenToDisc": true,
//     "workspaceFolder": {
//       "folderPath": "/path/to/workspace",
//       "timestamp": 1780898895787
//     }
//   }
// }
func (c *ideCollectorImpl) collectFromMetadata() ([]Session, error) {
	f, err := os.Open(c.metadataPath)
	if err != nil {
		return nil, fmt.Errorf("open metadata: %w", err)
	}
	defer f.Close()

	var metadata map[string]map[string]interface{}
	if err := json.NewDecoder(f).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("parse metadata JSON: %w", err)
	}

	var sessions []Session
	for sessionID, entry := range metadata {
		// Extract timestamps (Unix epoch milliseconds -> time.Time)
		var startTime, endTime time.Time
		if created, ok := entry["created"].(float64); ok && created > 0 {
			startTime = time.UnixMilli(int64(created)).UTC()
		}
		if modified, ok := entry["modified"].(float64); ok && modified > 0 {
			endTime = time.UnixMilli(int64(modified)).UTC()
		}

		// Extract workspace path (best-effort; may be empty)
		var workspaceDir string
		if wsFolder, ok := entry["workspaceFolder"].(map[string]interface{}); ok {
			if folderPath, ok := wsFolder["folderPath"].(string); ok && folderPath != "" {
				workspaceDir = folderPath
			}
		}

		s := Session{
			ID:              sessionID,
			Source:          "ide-chat",
			TokenCostSource: "estimated", // Phase 6 limitation: no real token cost
			StartTime:       startTime,
			EndTime:         endTime,
			WorkspaceDir:    workspaceDir,
			ProjectName:     filepath.Base(workspaceDir),
			IsActive:        false, // metadata-only sessions are never "active"
			IsFinal:         true,  // metadata is final (not a live snapshot)
			Tokens:          TokenBreakdown{}, // Phase 6: no per-turn token data from metadata
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
}
