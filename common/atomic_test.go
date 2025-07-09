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

func TestAtomicWriter(t *testing.T) {
	w := &AtomicWriter{}
	n, err := w.Write([]byte("Hello, "))
	assert.NoError(t, err, "AtomicWriter should not error on first write")
	assert.Equal(t, 7, n, "AtomicWriter should return correct byte count on first write")
	_, err = w.Write([]byte("World!"))
	assert.NoError(t, err, "AtomicWriter should not error on first write")
	assert.Equal(t, "Hello, World!", w.String(), "AtomicWriter should concatenate writes correctly")

	_, err = w.Write([]byte(" More text."))
	assert.NoError(t, err, "AtomicWriter should not error on subsequent writes")
	assert.Equal(t, "Hello, World! More text.", w.String(), "AtomicWriter should concatenate all writes correctly")
}
