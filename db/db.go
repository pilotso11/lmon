// Package db provides optional metrics persistence for lmon.
// It defines the Store interface for saving and querying monitor snapshots,
// along with supporting types for history and aggregated summaries.
// The database layer is always optional -- all functionality works without it.
package db

import (
	"context"
	"time"
)

// MonitorSnapshot represents a single point-in-time recording of a monitor's status.
type MonitorSnapshot struct {
	ID          uint      `gorm:"primarykey"`
	RecordedAt  time.Time `gorm:"index;not null"`
	Node        string    `gorm:"index:idx_node_monitor;not null"`
	MonitorID   string    `gorm:"index:idx_node_monitor;not null"`
	MonitorType string    `gorm:"not null"`
	Status      string    `gorm:"not null"`
	Value       *float64
	Message     string
}

// MonitorSummary provides aggregated status counts for a monitor over a time range.
type MonitorSummary struct {
	Node        string
	MonitorID   string
	MonitorType string
	GreenCount  int64
	AmberCount  int64
	RedCount    int64
	ErrorCount  int64
	TotalCount  int64
}

// Store is the interface for metrics persistence.
// All methods are safe to call even when the store is unavailable;
// they return empty results and nil errors in that case.
type Store interface {
	SaveSnapshots(ctx context.Context, snapshots []MonitorSnapshot) error
	GetHistory(ctx context.Context, node, monitorID string, from, to time.Time, limit int) ([]MonitorSnapshot, error)
	GetSummary(ctx context.Context, from, to time.Time) ([]MonitorSummary, error)
	PurgeOlderThan(ctx context.Context, cutoff time.Time, batchSize int) (int64, error)
	CompactOlderThan(ctx context.Context, olderThan, notBefore time.Time, bucketMinutes, batchSize int) (int64, error)
	Close() error
	IsAvailable() bool
}
