package service

import (
	"context"

	"github.com/aaraminds/rif/phase5/ingestion/queue"
)

// QueueDispatcher bridges Phase 5 queue worker dispatches into lane-aware
// incremental/full indexing triggers.
type QueueDispatcher struct {
	incrementalService *IncrementalService
}

func NewQueueDispatcher(incrementalService *IncrementalService) *QueueDispatcher {
	return &QueueDispatcher{incrementalService: incrementalService}
}

func (d *QueueDispatcher) Dispatch(ctx context.Context, item queue.Item) error {
	_, err := d.incrementalService.TriggerIncremental(ctx, item)
	return err
}
