package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAtomicDuration(t *testing.T) {
	tests := []struct {
		name     string
		initial  time.Duration
		newValue time.Duration
	}{
		{"zero", 0, time.Nanosecond},
		{"one second", time.Second, time.Minute},
		{"ten minutes", 10 * time.Minute, 5 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ad := NewAtomicDuration(tt.initial)
			assert.Equal(t, tt.initial, ad.Load(), "initial value should match")
			ad.Store(tt.newValue)
			assert.Equal(t, tt.newValue, ad.Load(), "new value should match after store")
		})
	}
}
