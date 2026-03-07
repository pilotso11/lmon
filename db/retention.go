package db

import (
	"context"
	"log"
	"sync"
	"time"
)

// RetentionManager periodically purges old snapshots from the store based on
// a configurable retention period and batch size.
type RetentionManager struct {
	store         Store
	retentionDays int
	batchSize     int
	pruneInterval time.Duration
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewRetentionManager creates a new RetentionManager.
// retentionDays: how many days of data to keep.
// batchSize: number of rows to delete per batch.
// pruneIntervalMinutes: how often (in minutes) the purge cycle runs.
func NewRetentionManager(store Store, retentionDays, batchSize int, pruneIntervalMinutes int) *RetentionManager {
	return &RetentionManager{
		store:         store,
		retentionDays: retentionDays,
		batchSize:     batchSize,
		pruneInterval: time.Duration(pruneIntervalMinutes) * time.Minute,
	}
}

// Start begins the periodic purge loop in a background goroutine.
func (r *RetentionManager) Start(ctx context.Context) {
	ctx, r.cancel = context.WithCancel(ctx)
	r.wg.Add(1)
	go r.loop(ctx)
}

// loop runs the purge cycle at the configured interval.
func (r *RetentionManager) loop(ctx context.Context) {
	defer r.wg.Done()
	ticker := time.NewTicker(r.pruneInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-time.Duration(r.retentionDays) * 24 * time.Hour)
			deleted, err := r.store.PurgeOlderThan(ctx, cutoff, r.batchSize)
			if err != nil {
				log.Printf("RetentionManager: purge error: %v", err)
			} else if deleted > 0 {
				log.Printf("RetentionManager: purged %d old snapshots", deleted)
			}
		}
	}
}

// Stop cancels the purge loop and waits for it to finish.
func (r *RetentionManager) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()
}
