package disk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMockDiskProvider(t *testing.T) {
	m := NewMockDiskProvider(80)
	assert.Equal(t, 80.0, m.Current.Load(), "Initial value should be 80")
}
