package db

import (
	"context"
	"log"
	"sync"
	"time"
)

// RetentionManager periodically purges old snapshots from the store based on
// a configurable retention period and batch size. It also compacts older
// full-resolution data into lower-resolution buckets.
type RetentionManager struct {
	store           Store
	retentionDays   int
	batchSize       int
	pruneInterval   time.Duration
	compactAfter    time.Duration // full-res window (e.g. 3h)
	compactInterval int           // bucket size in minutes (e.g. 15)
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	started         bool
}

// NewRetentionManager creates a new RetentionManager.
// retentionDays: how many days of data to keep.
// batchSize: number of rows to delete per batch.
// pruneIntervalMinutes: how often (in minutes) the purge cycle runs.
// compactAfterMinutes: minutes of full-resolution data to keep before compacting.
// compactIntervalMinutes: target bucket size in minutes for compaction.
func NewRetentionManager(store Store, retentionDays, batchSize int, pruneIntervalMinutes int, compactAfterMinutes, compactIntervalMinutes int) *RetentionManager {
	return &RetentionManager{
		store:           store,
		retentionDays:   retentionDays,
		batchSize:       batchSize,
		pruneInterval:   time.Duration(pruneIntervalMinutes) * time.Minute,
		compactAfter:    time.Duration(compactAfterMinutes) * time.Minute,
		compactInterval: compactIntervalMinutes,
	}
}

// Start begins the periodic purge loop in a background goroutine.
// Safe to call only once; subsequent calls are no-ops.
func (r *RetentionManager) Start(ctx context.Context) {
	if r.started {
		return
	}
	r.started = true
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

			// Compact: thin out data older than compactAfter but newer than retention cutoff
			if r.compactAfter > 0 && r.compactInterval > 0 {
				compactBefore := time.Now().Add(-r.compactAfter)
				if compactBefore.After(cutoff) {
					compacted, compactErr := r.store.CompactOlderThan(ctx, compactBefore, cutoff, r.compactInterval, r.batchSize)
					if compactErr != nil {
						log.Printf("RetentionManager: compact error: %v", compactErr)
					} else if compacted > 0 {
						log.Printf("RetentionManager: compacted %d snapshots", compacted)
					}
				}
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
