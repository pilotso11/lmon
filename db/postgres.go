package db

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresStore implements Store using PostgreSQL via GORM.
// If the database is unreachable at startup, the store remains in an unavailable
// state and all methods return empty results without error.
type PostgresStore struct {
	db        *gorm.DB
	available atomic.Bool
}

// NewPostgresStore creates a new PostgresStore connecting to the given DSN.
// Connection failure is non-fatal: the store will be created in an unavailable
// state and all methods will return empty/nil gracefully.
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	store := &PostgresStore{}

	// Connect with timeout
	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	type result struct {
		db  *gorm.DB
		err error
	}
	ch := make(chan result, 1)
	go func() {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		ch <- result{db, err}
	}()

	select {
	case <-connectCtx.Done():
		log.Printf("PostgresStore: connection timeout, database unavailable")
		return store, nil // Non-fatal
	case r := <-ch:
		if r.err != nil {
			log.Printf("PostgresStore: connection failed: %v", r.err)
			return store, nil // Non-fatal
		}
		store.db = r.db
	}

	// AutoMigrate
	if err := store.db.AutoMigrate(&MonitorSnapshot{}); err != nil {
		log.Printf("PostgresStore: migration failed: %v", err)
		return store, nil
	}

	store.available.Store(true)
	log.Printf("PostgresStore: connected and migrated successfully")
	return store, nil
}

// SaveSnapshots persists a batch of monitor snapshots to the database.
func (s *PostgresStore) SaveSnapshots(ctx context.Context, snapshots []MonitorSnapshot) error {
	if !s.available.Load() {
		return nil
	}
	if len(snapshots) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Create(&snapshots).Error
}

// GetHistory retrieves monitor snapshots for a given node and monitor ID within a time range.
// Results are ordered by RecordedAt descending, limited to the specified count.
func (s *PostgresStore) GetHistory(ctx context.Context, node, monitorID string, from, to time.Time, limit int) ([]MonitorSnapshot, error) {
	if !s.available.Load() {
		return nil, nil
	}
	var snapshots []MonitorSnapshot
	query := s.db.WithContext(ctx).Where("recorded_at BETWEEN ? AND ?", from, to)
	if node != "" {
		query = query.Where("node = ?", node)
	}
	if monitorID != "" {
		query = query.Where("monitor_id = ?", monitorID)
	}
	err := query.Order("recorded_at DESC").Limit(limit).Find(&snapshots).Error
	return snapshots, err
}

// GetSummary returns aggregated status counts grouped by node, monitor ID, and type
// for snapshots recorded within the given time range.
func (s *PostgresStore) GetSummary(ctx context.Context, from, to time.Time) ([]MonitorSummary, error) {
	if !s.available.Load() {
		return nil, nil
	}
	var summaries []MonitorSummary
	err := s.db.WithContext(ctx).Model(&MonitorSnapshot{}).
		Select(`node, monitor_id, monitor_type,
			SUM(CASE WHEN status = 'Green' THEN 1 ELSE 0 END) as green_count,
			SUM(CASE WHEN status = 'Amber' THEN 1 ELSE 0 END) as amber_count,
			SUM(CASE WHEN status = 'Red' THEN 1 ELSE 0 END) as red_count,
			SUM(CASE WHEN status = 'Error' THEN 1 ELSE 0 END) as error_count,
			COUNT(*) as total_count`).
		Where("recorded_at BETWEEN ? AND ?", from, to).
		Group("node, monitor_id, monitor_type").
		Scan(&summaries).Error
	return summaries, err
}

// PurgeOlderThan deletes snapshots older than the cutoff time in batches.
// Returns the total number of deleted rows.
func (s *PostgresStore) PurgeOlderThan(ctx context.Context, cutoff time.Time, batchSize int) (int64, error) {
	if !s.available.Load() {
		return 0, nil
	}
	var totalDeleted int64
	for {
		if ctx.Err() != nil {
			return totalDeleted, ctx.Err()
		}
		result := s.db.WithContext(ctx).Where("recorded_at < ?", cutoff).Limit(batchSize).Delete(&MonitorSnapshot{})
		if result.Error != nil {
			return totalDeleted, result.Error
		}
		totalDeleted += result.RowsAffected
		if result.RowsAffected < int64(batchSize) {
			break
		}
	}
	return totalDeleted, nil
}

// Close closes the underlying database connection.
func (s *PostgresStore) Close() error {
	if !s.available.Load() {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// IsAvailable returns true if the database connection is established and usable.
func (s *PostgresStore) IsAvailable() bool {
	return s.available.Load()
}
