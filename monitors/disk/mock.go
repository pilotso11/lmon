// mock.go provides a mock implementation of UsageProvider for testing disk monitors.
package disk

import (
	"github.com/shirou/gopsutil/v3/disk"
	"go.uber.org/atomic"

	"lmon/common"
)

var _ UsageProvider = (*MockDiskProvider)(nil)

// MockDiskProvider is a mock implementation of UsageProvider for testing.
// It allows simulation of disk usage and errors.
type MockDiskProvider struct {
	Current *atomic.Float64 // Current usage percentage (0-100)
	total   float64         // Total disk size in bytes
	path    string          // Filesystem path
	err     error           // Error to return from Usage, if any
}

// Usage returns a mocked disk.UsageStat based on the Current value and total size.
func (m MockDiskProvider) Usage(_ string) (*disk.UsageStat, error) {
	return &disk.UsageStat{
		Path:        m.path,
		Fstype:      "ext2",
		Total:       uint64(m.total),
		Free:        uint64(m.total - m.total*m.Current.Load()/100.0),
		Used:        uint64(m.total * m.Current.Load() / 100.0),
		UsedPercent: m.Current.Load(),
	}, m.err
}

// NewMockDiskProvider creates a new MockDiskProvider with the given initial usage percentage.
// The total disk size is set to 100 GB.
func NewMockDiskProvider(initial float64) *MockDiskProvider {
	return &MockDiskProvider{
		Current: atomic.NewFloat64(initial),
		total:   100 * common.Gigibyte,
	}
}
