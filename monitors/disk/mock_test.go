// mock_test.go contains unit tests for the MockDiskProvider in the disk package.
package disk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewMockDiskProvider verifies that NewMockDiskProvider initializes the Current value correctly.
func TestNewMockDiskProvider(t *testing.T) {
	m := NewMockDiskProvider(80)
	assert.Equal(t, 80.0, m.Current.Load(), "Initial value should be 80")
}
