package db

import (
	"context"
	"log"
	"sync"
	"time"
)

// BufferedWriter provides non-blocking, buffered writes of monitor snapshots to a Store.
// Snapshots are pushed to a channel and flushed asynchronously. If the channel is full,
// writes are silently dropped to avoid blocking the monitoring loop.
type BufferedWriter struct {
	store         Store
	ch            chan []MonitorSnapshot
	wg            sync.WaitGroup
	writeInterval time.Duration
	lastWrite     time.Time
	mu            sync.Mutex
}

// NewBufferedWriter creates a new BufferedWriter with the given buffer size and write interval.
// The writer starts a background goroutine that drains the channel and persists snapshots.
// writeInterval controls the minimum time between accepting writes (0 = accept every write).
func NewBufferedWriter(store Store, bufferSize int, writeInterval time.Duration) *BufferedWriter {
	w := &BufferedWriter{
		store:         store,
		ch:            make(chan []MonitorSnapshot, bufferSize),
		writeInterval: writeInterval,
	}
	w.wg.Add(1)
	go w.flushLoop()
	return w
}

// Write pushes snapshots to the channel in a non-blocking fashion.
// If the write interval has not elapsed since the last accepted write, the snapshots are dropped.
// If the channel is full, the snapshots are silently dropped.
func (w *BufferedWriter) Write(snapshots []MonitorSnapshot) {
	// Check write interval
	if w.writeInterval > 0 {
		w.mu.Lock()
		if time.Since(w.lastWrite) < w.writeInterval {
			w.mu.Unlock()
			return
		}
		w.lastWrite = time.Now()
		w.mu.Unlock()
	}

	select {
	case w.ch <- snapshots:
	default:
		// Channel full, drop silently to avoid blocking monitoring
	}
}

// flushLoop drains the channel and writes batches to the store.
func (w *BufferedWriter) flushLoop() {
	defer w.wg.Done()
	for batch := range w.ch {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := w.store.SaveSnapshots(ctx, batch); err != nil {
			log.Printf("BufferedWriter: flush error: %v", err)
		}
		cancel()
	}
}

// Close closes the channel and waits for all pending writes to complete.
func (w *BufferedWriter) Close() {
	close(w.ch)
	w.wg.Wait()
}
