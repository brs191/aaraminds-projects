// Package audit is the append-only audit writer helper (PRD R9, 13.3
// audit_events, 18.1 logging rules). Every tool call, status transition,
// approval, export, and error flows through here. Audit write failures are
// treated as control failures (PRD 19) and logged loudly.
package audit

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/aaraminds/transcript-agent/internal/domain"
	"github.com/aaraminds/transcript-agent/internal/store"
)

// Actor types (PRD 13.3 audit_events.actor_type).
const (
	ActorUser   = "user"
	ActorSystem = "system"
	ActorTool   = "tool"
)

// Writer appends audit events via the store. It never returns errors to the
// caller: an audit failure must not silently corrupt workflow state, but it is
// surfaced as an error-level log so alerting can pause high-risk actions
// (PRD 18.4 alert 5, 19 "Audit logging failure").
type Writer struct {
	Store store.AuditStore
	Log   *slog.Logger
}

// New returns a Writer.
func New(s store.AuditStore, log *slog.Logger) *Writer {
	if log == nil {
		log = slog.Default()
	}
	return &Writer{Store: s, Log: log}
}

// Event appends one audit event.
func (w *Writer) Event(ctx context.Context, jobID *uuid.UUID, actorType, actorID, eventType string, payload map[string]any) error {
	e := &domain.AuditEvent{
		AuditEventID: uuid.New(),
		JobID:        jobID,
		ActorType:    actorType,
		ActorID:      actorID,
		EventType:    eventType,
		EventPayload: payload,
		CreatedAt:    time.Now().UTC(),
	}
	if err := w.Store.Append(ctx, e); err != nil {
		log := w.Log
		if log == nil {
			log = slog.Default()
		}
		log.Error("AUDIT LOGGING FAILURE — control failure, pause high-risk actions",
			"event_type", eventType, "error", err)
		return domain.E(domain.CodeAuditWriteFailed, "audit append failed for %s: %v", eventType, err)
	}
	return nil
}
