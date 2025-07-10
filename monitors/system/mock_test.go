// mock_test.go contains unit tests for the MockCpuProvider and MockMemProvider in the system package.
package system

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"lmon/common"
)

// TestNewMockCpuProvider verifies that NewMockCpuProvider initializes the Current value correctly.
func TestNewMockCpuProvider(t *testing.T) {
	m := NewMockCpuProvider(50)
	assert.NotNil(t, m, "should not be nil")
	assert.Equalf(t, 50.0, m.Current.Load(), "Initial value")
	u, err := m.Usage()
	assert.NoError(t, err, "should not error")
	assert.Equal(t, 50.0, u.Usage, "Usage should match initial value")
	assert.Equal(t, 8, u.Count, "Count should be 8")
}

// TestNewMockMemProvider verifies that NewMockMemProvider initializes the Current value correctly.
func TestNewMockMemProvider(t *testing.T) {
	m := NewMockMemProvider(40)
	assert.NotNil(t, m, "should not be nil")
	assert.Equalf(t, 40.0, m.Current.Load(), "Initial value")
	u, err := m.Usage()
	assert.NoError(t, err, "should not error")
	assert.Equal(t, 40.0, u.UsedPercent, "UsedPercent should match initial value")
	assert.Equal(t, uint64(common.Gigabyte*100), u.Total, "Total should match initial value")
	assert.Equal(t, uint64(common.Gigabyte*40), u.Used, "Used should match 50% of total")
	assert.Equal(t, uint64(common.Gigabyte*60), u.Available, "Available should match 50% of total")

}
