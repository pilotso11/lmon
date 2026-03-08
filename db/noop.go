package db

import (
	"context"
	"time"
)

// NoopStore is a no-operation Store used when no database URL is configured.
// All methods return empty results and nil errors. IsAvailable() always returns false.
type NoopStore struct{}

// NewNoopStore creates a new NoopStore.
func NewNoopStore() *NoopStore {
	return &NoopStore{}
}

// SaveSnapshots is a no-op that always returns nil.
func (n *NoopStore) SaveSnapshots(_ context.Context, _ []MonitorSnapshot) error {
	return nil
}

// GetHistory always returns an empty slice and nil error.
func (n *NoopStore) GetHistory(_ context.Context, _, _ string, _, _ time.Time, _ int) ([]MonitorSnapshot, error) {
	return nil, nil
}

// GetSummary always returns an empty slice and nil error.
func (n *NoopStore) GetSummary(_ context.Context, _, _ time.Time) ([]MonitorSummary, error) {
	return nil, nil
}

// PurgeOlderThan is a no-op that always returns 0 and nil.
func (n *NoopStore) PurgeOlderThan(_ context.Context, _ time.Time, _ int) (int64, error) {
	return 0, nil
}

// CompactOlderThan is a no-op that always returns 0 and nil.
func (n *NoopStore) CompactOlderThan(_ context.Context, _, _ time.Time, _, _ int) (int64, error) {
	return 0, nil
}

// Close is a no-op that always returns nil.
func (n *NoopStore) Close() error {
	return nil
}

// IsAvailable always returns false for the no-op store.
func (n *NoopStore) IsAvailable() bool {
	return false
}
