package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMockCpuProvider(t *testing.T) {
	m := NewMockCpuProvider(50)
	assert.NotNil(t, m, "should not be nil")
	assert.Equalf(t, 50.0, m.Current.Load(), "Initial value")
}

func TestNewMockMemProvider(t *testing.T) {
	m := NewMockMemProvider(50)
	assert.NotNil(t, m, "should not be nil")
	assert.Equalf(t, 50.0, m.Current.Load(), "Initial value")
}
