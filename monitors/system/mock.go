package system

import (
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/atomic"
)

type MockCpuProvider struct {
	Current *atomic.Float64
	err     error
}

func (m MockCpuProvider) Usage() (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.Current.Load(), nil
}

func NewMockCpuProvider(initial float64) *MockCpuProvider {
	return &MockCpuProvider{Current: atomic.NewFloat64(initial)}
}

type MockMemProvider struct {
	Current *atomic.Float64
	total   float64
	err     error
}

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

func NewMockMemProvider(initial float64) *MockMemProvider {
	return &MockMemProvider{Current: atomic.NewFloat64(initial)}
}
