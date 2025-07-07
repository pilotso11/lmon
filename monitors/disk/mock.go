package disk

import (
	"github.com/shirou/gopsutil/v3/disk"
	"go.uber.org/atomic"
)

type MockDiskProvider struct {
	Current *atomic.Float64
	total   float64
	path    string
	err     error
}

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

func NewMockDiskProvider(initial float64) *MockDiskProvider {
	return &MockDiskProvider{
		Current: atomic.NewFloat64(initial),
		total:   100 * gigabyte,
	}
}
