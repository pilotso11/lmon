// mock.go provides mock implementations of CpuProvider and MemProvider for testing system monitors.
package system

import (
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/atomic"
)

// MockCpuProvider is a mock implementation of CpuProvider for testing.
// It allows simulation of CPU usage and errors.
type MockCpuProvider struct {
	Current *atomic.Float64 // Current CPU usage percentage (0-100)
	err     error           // Error to return from Usage, if any
}

// Usage returns the mocked CPU usage or an error if set.
func (m MockCpuProvider) Usage() (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.Current.Load(), nil
}

// NewMockCpuProvider creates a new MockCpuProvider with the given initial usage percentage.
func NewMockCpuProvider(initial float64) *MockCpuProvider {
	return &MockCpuProvider{Current: atomic.NewFloat64(initial)}
}

// MockMemProvider is a mock implementation of MemProvider for testing.
// It allows simulation of memory usage and errors.
type MockMemProvider struct {
	Current *atomic.Float64 // Current memory usage percentage (0-100)
	total   float64         // Total memory size in bytes
	err     error           // Error to return from Usage, if any
}

// Usage returns a mocked mem.VirtualMemoryStat based on the Current value and total size.
func (m MockMemProvider) Usage() (*mem.VirtualMemoryStat, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &mem.VirtualMemoryStat{
		Total:       uint64(m.total),
		Available:   uint64(m.total - m.Current.Load()*m.total),
		Used:        uint64(m.Current.Load() * m.total),
		UsedPercent: m.Current.Load(),
	}, nil
}

// NewMockMemProvider creates a new MockMemProvider with the given initial usage percentage.
func NewMockMemProvider(initial float64) *MockMemProvider {
	return &MockMemProvider{Current: atomic.NewFloat64(initial)}
}
