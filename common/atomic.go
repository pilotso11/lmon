package common

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type AtomicDuration struct {
	Value atomic.Int64 // Duration in nanoseconds (raw time.Duration is int64).
}

func (a *AtomicDuration) Load() time.Duration {
	return time.Duration(a.Value.Load())
}

func (a *AtomicDuration) Store(d time.Duration) {
	a.Value.Store(int64(d))
}

func NewAtomicDuration(d time.Duration) *AtomicDuration {
	ad := &AtomicDuration{}
	ad.Store(d)
	return ad
}

type AtomicWriter struct {
	mu sync.Mutex
	b  strings.Builder
}

func (w *AtomicWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func (w *AtomicWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.String()
}
