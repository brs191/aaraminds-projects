// util.go — shared helpers for the mcpserver package.
package mcpserver

import (
	"crypto/rand"
	"encoding/hex"
)

// newAuditID returns a random 16-byte hex string used as an audit identifier.
// It is not a UUID but is unique enough for audit log correlation within a
// single process run.
func newAuditID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: zero-filled string signals entropy failure without panicking.
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}
