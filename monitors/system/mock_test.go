// mock_test.go contains unit tests for the MockCpuProvider and MockMemProvider in the system package.
package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewMockCpuProvider verifies that NewMockCpuProvider initializes the Current value correctly.
func TestNewMockCpuProvider(t *testing.T) {
	m := NewMockCpuProvider(50)
	assert.NotNil(t, m, "should not be nil")
	assert.Equalf(t, 50.0, m.Current.Load(), "Initial value")
}

// TestNewMockMemProvider verifies that NewMockMemProvider initializes the Current value correctly.
func TestNewMockMemProvider(t *testing.T) {
	m := NewMockMemProvider(50)
	assert.NotNil(t, m, "should not be nil")
	assert.Equalf(t, 50.0, m.Current.Load(), "Initial value")
}
