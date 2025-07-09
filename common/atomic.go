package common

import (
	"sync/atomic"
	"time"
)

type AtomicDuration struct {
	Value atomic.Int64 // Duration in milliseconds
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
