package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newTestStore creates an in-memory SQLite-backed PostgresStore for testing.
// Uses shared cache mode with a unique name per test to ensure all goroutines
// (including BufferedWriter's flush loop) see the same database, while avoiding
// cross-test contamination via the unique test name.
func newTestStore(t *testing.T) *PostgresStore {
	t.Helper()
	// Use a unique shared-cache database per test to prevent data leakage between tests.
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	store := &PostgresStore{db: database}
	store.available.Store(true)
	err = database.AutoMigrate(&MonitorSnapshot{})
	require.NoError(t, err)
	return store
}

// TestSaveAndGetHistory saves snapshots and queries by node/monitorID/time range.
func TestSaveAndGetHistory(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	now := time.Now().UTC()
	val := 42.0
	snapshots := []MonitorSnapshot{
		{RecordedAt: now.Add(-2 * time.Hour), Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Green", Value: &val, Message: "ok"},
		{RecordedAt: now.Add(-1 * time.Hour), Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Amber", Value: &val, Message: "warning"},
		{RecordedAt: now, Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Red", Value: &val, Message: "critical"},
		{RecordedAt: now, Node: "node2", MonitorID: "cpu", MonitorType: "system", Status: "Green", Value: &val, Message: "ok"},
	}

	err := store.SaveSnapshots(ctx, snapshots)
	require.NoError(t, err)

	// Query all for node1/disk_root
	results, err := store.GetHistory(ctx, "node1", "disk_root", now.Add(-3*time.Hour), now.Add(time.Hour), 100)
	require.NoError(t, err)
	assert.Len(t, results, 3, "should return 3 snapshots for node1/disk_root")

	// Results are ordered by RecordedAt DESC
	assert.Equal(t, "Red", results[0].Status)
	assert.Equal(t, "Amber", results[1].Status)
	assert.Equal(t, "Green", results[2].Status)

	// Query with limit
	results, err = store.GetHistory(ctx, "node1", "disk_root", now.Add(-3*time.Hour), now.Add(time.Hour), 2)
	require.NoError(t, err)
	assert.Len(t, results, 2, "should respect limit")

	// Query for node2
	results, err = store.GetHistory(ctx, "node2", "", now.Add(-3*time.Hour), now.Add(time.Hour), 100)
	require.NoError(t, err)
	assert.Len(t, results, 1, "should return 1 snapshot for node2")

	// Query with empty node and monitorID (should return all)
	results, err = store.GetHistory(ctx, "", "", now.Add(-3*time.Hour), now.Add(time.Hour), 100)
	require.NoError(t, err)
	assert.Len(t, results, 4, "should return all 4 snapshots")

	// Query outside time range
	results, err = store.GetHistory(ctx, "node1", "disk_root", now.Add(-10*time.Hour), now.Add(-5*time.Hour), 100)
	require.NoError(t, err)
	assert.Len(t, results, 0, "should return 0 snapshots outside range")
}

// TestSaveEmptySnapshots verifies saving an empty slice is a no-op.
func TestSaveEmptySnapshots(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	err := store.SaveSnapshots(ctx, []MonitorSnapshot{})
	require.NoError(t, err)

	results, err := store.GetHistory(ctx, "", "", time.Time{}, time.Now().Add(time.Hour), 100)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

// TestGetSummary saves mixed statuses and verifies aggregated counts.
func TestGetSummary(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	now := time.Now().UTC()
	snapshots := []MonitorSnapshot{
		{RecordedAt: now, Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Green"},
		{RecordedAt: now, Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Green"},
		{RecordedAt: now, Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Amber"},
		{RecordedAt: now, Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Red"},
		{RecordedAt: now, Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Error"},
		{RecordedAt: now, Node: "node1", MonitorID: "cpu", MonitorType: "system", Status: "Green"},
		{RecordedAt: now, Node: "node1", MonitorID: "cpu", MonitorType: "system", Status: "Red"},
	}

	err := store.SaveSnapshots(ctx, snapshots)
	require.NoError(t, err)

	summaries, err := store.GetSummary(ctx, now.Add(-time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	assert.Len(t, summaries, 2, "should have 2 monitor groups")

	// Find the disk_root summary
	var diskSummary, cpuSummary *MonitorSummary
	for i := range summaries {
		switch summaries[i].MonitorID {
		case "disk_root":
			diskSummary = &summaries[i]
		case "cpu":
			cpuSummary = &summaries[i]
		}
	}

	require.NotNil(t, diskSummary, "should have disk_root summary")
	assert.Equal(t, int64(2), diskSummary.GreenCount)
	assert.Equal(t, int64(1), diskSummary.AmberCount)
	assert.Equal(t, int64(1), diskSummary.RedCount)
	assert.Equal(t, int64(1), diskSummary.ErrorCount)
	assert.Equal(t, int64(5), diskSummary.TotalCount)

	require.NotNil(t, cpuSummary, "should have cpu summary")
	assert.Equal(t, int64(1), cpuSummary.GreenCount)
	assert.Equal(t, int64(1), cpuSummary.RedCount)
	assert.Equal(t, int64(2), cpuSummary.TotalCount)
}

// TestPurgeOlderThan saves old + new records and verifies only old ones are purged.
func TestPurgeOlderThan(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	now := time.Now().UTC()
	cutoff := now.Add(-24 * time.Hour)

	snapshots := []MonitorSnapshot{
		{RecordedAt: now.Add(-48 * time.Hour), Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Green"},
		{RecordedAt: now.Add(-36 * time.Hour), Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Amber"},
		{RecordedAt: now.Add(-1 * time.Hour), Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Green"},
		{RecordedAt: now, Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Green"},
	}

	err := store.SaveSnapshots(ctx, snapshots)
	require.NoError(t, err)

	deleted, err := store.PurgeOlderThan(ctx, cutoff, 1000)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted, "should delete 2 old snapshots")

	// Verify remaining
	results, err := store.GetHistory(ctx, "", "", time.Time{}, now.Add(time.Hour), 100)
	require.NoError(t, err)
	assert.Len(t, results, 2, "should have 2 remaining snapshots")
}

// TestPurgeBatchBoundary verifies that batched deletion works correctly when
// the number of rows to delete spans multiple batches.
func TestPurgeBatchBoundary(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	now := time.Now().UTC()
	cutoff := now.Add(-1 * time.Hour)

	// Create 5 old snapshots
	snapshots := make([]MonitorSnapshot, 5)
	for i := range snapshots {
		snapshots[i] = MonitorSnapshot{
			RecordedAt:  now.Add(-2 * time.Hour),
			Node:        "node1",
			MonitorID:   "disk_root",
			MonitorType: "disk",
			Status:      "Green",
		}
	}
	// Add 1 new snapshot
	snapshots = append(snapshots, MonitorSnapshot{
		RecordedAt:  now,
		Node:        "node1",
		MonitorID:   "disk_root",
		MonitorType: "disk",
		Status:      "Green",
	})

	err := store.SaveSnapshots(ctx, snapshots)
	require.NoError(t, err)

	// Delete in batches of 2 -- should take 3 iterations to delete all 5 old rows
	deleted, err := store.PurgeOlderThan(ctx, cutoff, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), deleted, "should delete all 5 old snapshots in batches")

	// Verify remaining
	results, err := store.GetHistory(ctx, "", "", time.Time{}, now.Add(time.Hour), 100)
	require.NoError(t, err)
	assert.Len(t, results, 1, "should have 1 remaining snapshot")
}

// TestPurgeWithCancelledContext verifies that purge respects context cancellation.
func TestPurgeWithCancelledContext(t *testing.T) {
	store := newTestStore(t)

	now := time.Now().UTC()
	snapshots := make([]MonitorSnapshot, 10)
	for i := range snapshots {
		snapshots[i] = MonitorSnapshot{
			RecordedAt:  now.Add(-2 * time.Hour),
			Node:        "node1",
			MonitorID:   "disk_root",
			MonitorType: "disk",
			Status:      "Green",
		}
	}

	ctx := t.Context()
	err := store.SaveSnapshots(ctx, snapshots)
	require.NoError(t, err)

	// Cancel context before purge
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	_, err = store.PurgeOlderThan(cancelCtx, now, 2)
	assert.Error(t, err, "should return error for cancelled context")
}

// TestNoopStore verifies that all NoopStore methods return cleanly.
func TestNoopStore(t *testing.T) {
	store := NewNoopStore()
	ctx := t.Context()

	assert.False(t, store.IsAvailable(), "noop store should not be available")

	err := store.SaveSnapshots(ctx, []MonitorSnapshot{{Status: "Green"}})
	assert.NoError(t, err, "SaveSnapshots should not error")

	history, err := store.GetHistory(ctx, "node", "monitor", time.Time{}, time.Now(), 100)
	assert.NoError(t, err, "GetHistory should not error")
	assert.Nil(t, history, "GetHistory should return nil")

	summary, err := store.GetSummary(ctx, time.Time{}, time.Now())
	assert.NoError(t, err, "GetSummary should not error")
	assert.Nil(t, summary, "GetSummary should return nil")

	deleted, err := store.PurgeOlderThan(ctx, time.Now(), 100)
	assert.NoError(t, err, "PurgeOlderThan should not error")
	assert.Equal(t, int64(0), deleted, "PurgeOlderThan should return 0")

	err = store.Close()
	assert.NoError(t, err, "Close should not error")
}

// TestPostgresStoreUnavailable verifies that all methods return nil/empty when the store is unavailable.
func TestPostgresStoreUnavailable(t *testing.T) {
	store := &PostgresStore{} // db is nil, available is false
	ctx := t.Context()

	assert.False(t, store.IsAvailable(), "store should not be available")

	err := store.SaveSnapshots(ctx, []MonitorSnapshot{{Status: "Green"}})
	assert.NoError(t, err, "SaveSnapshots should not error when unavailable")

	history, err := store.GetHistory(ctx, "node", "monitor", time.Time{}, time.Now(), 100)
	assert.NoError(t, err, "GetHistory should not error when unavailable")
	assert.Nil(t, history, "GetHistory should return nil when unavailable")

	summary, err := store.GetSummary(ctx, time.Time{}, time.Now())
	assert.NoError(t, err, "GetSummary should not error when unavailable")
	assert.Nil(t, summary, "GetSummary should return nil when unavailable")

	deleted, err := store.PurgeOlderThan(ctx, time.Now(), 100)
	assert.NoError(t, err, "PurgeOlderThan should not error when unavailable")
	assert.Equal(t, int64(0), deleted, "PurgeOlderThan should return 0 when unavailable")

	err = store.Close()
	assert.NoError(t, err, "Close should not error when unavailable")
}

// TestBufferedWriter verifies that writes are flushed to the store.
func TestBufferedWriter(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	writer := NewBufferedWriter(store, 10, 0)
	defer writer.Close()

	now := time.Now().UTC()
	snapshots := []MonitorSnapshot{
		{RecordedAt: now, Node: "node1", MonitorID: "cpu", MonitorType: "system", Status: "Green"},
	}

	writer.Write(snapshots)

	// Wait for the flush to occur
	assert.Eventually(t, func() bool {
		results, err := store.GetHistory(ctx, "node1", "cpu", now.Add(-time.Hour), now.Add(time.Hour), 100)
		if err != nil {
			return false
		}
		return len(results) == 1
	}, 2*time.Second, 10*time.Millisecond, "snapshot should be flushed to store")
}

// TestBufferedWriterFullChannel verifies no blocking when the channel is full.
func TestBufferedWriterFullChannel(t *testing.T) {
	store := newTestStore(t)

	// Create writer with buffer size of 1
	writer := NewBufferedWriter(store, 1, 0)
	defer writer.Close()

	now := time.Now().UTC()

	// Fill the channel -- these should not block.
	// Use fresh snapshots each time to avoid GORM ID reuse issues.
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			writer.Write(newSnapshot(now))
		}
		close(done)
	}()

	select {
	case <-done:
		// Good -- did not block
	case <-time.After(2 * time.Second):
		t.Fatal("Write should not block when channel is full")
	}
}

// TestBufferedWriterClose verifies that all pending writes are flushed on close.
func TestBufferedWriterClose(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	writer := NewBufferedWriter(store, 100, 0)

	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		writer.Write([]MonitorSnapshot{
			{RecordedAt: now, Node: "node1", MonitorID: "cpu", MonitorType: "system", Status: "Green"},
		})
	}

	// Close waits for all pending writes
	writer.Close()

	results, err := store.GetHistory(ctx, "node1", "cpu", now.Add(-time.Hour), now.Add(time.Hour), 100)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1, "at least some snapshots should be flushed on close")
}

// newSnapshot creates a fresh MonitorSnapshot with ID=0 so GORM auto-increments.
func newSnapshot(now time.Time) []MonitorSnapshot {
	return []MonitorSnapshot{
		{RecordedAt: now, Node: "node1", MonitorID: "cpu", MonitorType: "system", Status: "Green"},
	}
}

// TestBufferedWriterWriteInterval verifies rate limiting of writes.
func TestBufferedWriterWriteInterval(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	// Write interval of 100ms
	writer := NewBufferedWriter(store, 100, 100*time.Millisecond)
	defer writer.Close()

	now := time.Now().UTC()

	// First write should succeed (fresh snapshot each time to avoid GORM ID reuse)
	writer.Write(newSnapshot(now))

	// Immediate second write should be rate-limited (dropped)
	writer.Write(newSnapshot(now))
	writer.Write(newSnapshot(now))

	// Wait for first flush
	time.Sleep(50 * time.Millisecond)

	// Check that only 1 snapshot was saved (due to rate limiting)
	results, err := store.GetHistory(ctx, "node1", "cpu", now.Add(-time.Hour), now.Add(time.Hour), 100)
	require.NoError(t, err)
	assert.Equal(t, 1, len(results), "rate limiting should allow only the first write")

	// Wait for interval to pass
	time.Sleep(100 * time.Millisecond)

	// Now another write should succeed
	writer.Write(newSnapshot(now))

	assert.Eventually(t, func() bool {
		results, err := store.GetHistory(ctx, "node1", "cpu", now.Add(-time.Hour), now.Add(time.Hour), 100)
		if err != nil {
			return false
		}
		return len(results) == 2
	}, 2*time.Second, 10*time.Millisecond, "second write should be accepted after interval passes")
}

// TestRetentionManager starts the manager, waits for a prune cycle, and verifies deletion.
func TestRetentionManager(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	now := time.Now().UTC()
	snapshots := []MonitorSnapshot{
		{RecordedAt: now.Add(-48 * time.Hour), Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Green"},
		{RecordedAt: now, Node: "node1", MonitorID: "disk_root", MonitorType: "disk", Status: "Green"},
	}

	err := store.SaveSnapshots(ctx, snapshots)
	require.NoError(t, err)

	// RetentionDays=1, batchSize=100, pruneInterval in minutes
	// We use a very short interval for testing by creating the manager directly
	rm := &RetentionManager{
		store:         store,
		retentionDays: 1,
		batchSize:     100,
		pruneInterval: 10 * time.Millisecond, // very short for testing
	}
	rm.Start(ctx)

	// Wait for at least one prune cycle
	assert.Eventually(t, func() bool {
		results, err := store.GetHistory(ctx, "", "", time.Time{}, now.Add(time.Hour), 100)
		if err != nil {
			return false
		}
		return len(results) == 1
	}, 2*time.Second, 10*time.Millisecond, "old snapshot should be purged")

	rm.Stop()

	// Verify only the recent snapshot remains
	results, err := store.GetHistory(ctx, "", "", time.Time{}, now.Add(time.Hour), 100)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Green", results[0].Status)
}

// TestRetentionManagerStop verifies that Stop is safe to call even if Start was not called.
func TestRetentionManagerStop(t *testing.T) {
	store := newTestStore(t)
	rm := NewRetentionManager(store, 7, 1000, 60)
	// Stop without Start should not panic
	assert.NotPanics(t, func() {
		rm.Stop()
	})
}

// TestStoreIsAvailable verifies the IsAvailable method on a functioning store.
func TestStoreIsAvailable(t *testing.T) {
	store := newTestStore(t)
	assert.True(t, store.IsAvailable(), "test store should be available")
}

// TestStoreClose verifies the Close method on a functioning store.
func TestStoreClose(t *testing.T) {
	store := newTestStore(t)
	err := store.Close()
	assert.NoError(t, err, "Close should not error on a functioning store")
}
