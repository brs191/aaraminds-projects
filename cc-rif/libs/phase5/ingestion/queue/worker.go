package queue

import (
	"context"
	"fmt"
	"time"
)

type Dispatcher interface {
	Dispatch(ctx context.Context, item Item) error
}

type Worker struct {
	store          *Store
	dispatcher     Dispatcher
	pollInterval   time.Duration
	coalesceWindow time.Duration
	batchSize      int
}

func NewWorker(store *Store, dispatcher Dispatcher) *Worker {
	return &Worker{
		store:          store,
		dispatcher:     dispatcher,
		pollInterval:   5 * time.Second,
		coalesceWindow: 30 * time.Second,
		batchSize:      100,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.tick(ctx); err != nil {
				return err
			}
		}
	}
}

func (w *Worker) tick(ctx context.Context) error {
	items, err := w.store.FetchQueued(ctx, w.batchSize)
	if err != nil {
		return err
	}
	dispatch, coalesced := Coalesce(items, w.coalesceWindow)
	if err := w.store.MarkCoalesced(ctx, coalesced); err != nil {
		return fmt.Errorf("mark coalesced: %w", err)
	}
	for _, item := range dispatch {
		if err := w.store.MarkStatus(ctx, item.ID, "running", ""); err != nil {
			return err
		}
		if err := w.dispatcher.Dispatch(ctx, item); err != nil {
			_ = w.store.MarkStatus(ctx, item.ID, "failed", err.Error())
			continue
		}
		if err := w.store.MarkStatus(ctx, item.ID, "dispatched", ""); err != nil {
			return err
		}
	}
	return nil
}
