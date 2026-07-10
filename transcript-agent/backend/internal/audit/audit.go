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
	"github.com/aaraminds/transcript-agent/internal/metrics"
	"github.com/aaraminds/transcript-agent/internal/store"
)

// Actor types (PRD 13.3 audit_events.actor_type).
const (
	ActorUser   = "user"
	ActorSystem = "system"
	ActorTool   = "tool"
)

// Writer appends audit events via the store. Two failure disciplines exist
// (PRD 19 "Audit logging failure"):
//
//   - Event: informational/tool events are fire-and-forget — a failed append
//     is logged loudly (and counted in audit_write_failures) but never blocks
//     the workflow.
//   - EventStrict: high-risk actions (approve, export generation, cancel,
//     replace-media, caption-decision) MUST have an audit record. A failed
//     append returns AUDIT_UNAVAILABLE so the caller fails the request (503).
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

// Event appends one audit event (fire-and-forget discipline). The returned
// error exists for callers that want to observe the failure; informational
// call sites ignore it.
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
		metrics.AuditWriteFailures.Add(1)
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

// EventStrict is the must-succeed variant for high-risk actions (PRD 19 audit
// row): when the append fails it returns AUDIT_UNAVAILABLE, which the API
// surfaces as 503 so the action is refused rather than performed unaudited.
func (w *Writer) EventStrict(ctx context.Context, jobID *uuid.UUID, actorType, actorID, eventType string, payload map[string]any) error {
	if err := w.Event(ctx, jobID, actorType, actorID, eventType, payload); err != nil {
		return domain.E(domain.CodeAuditUnavailable,
			"the audit log is unavailable; high-risk actions are paused until audit writes recover")
	}
	return nil
}
