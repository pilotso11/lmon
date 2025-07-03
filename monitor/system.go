package monitor

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"lmon/config"
)

// CPUUsageProvider is an interface for getting CPU usage
type CPUUsageProvider interface {
	Percent(interval time.Duration, percpu bool) ([]float64, error)
}

// MemoryUsageProvider is an interface for getting memory usage
type MemoryUsageProvider interface {
	VirtualMemory() (*mem.VirtualMemoryStat, error)
}

// DefaultCPUUsageProvider is the default implementation of CPUUsageProvider
type DefaultCPUUsageProvider struct{}

// Percent returns CPU usage percentages
func (p *DefaultCPUUsageProvider) Percent(interval time.Duration, percpu bool) ([]float64, error) {
	return cpu.Percent(interval, percpu)
}

// DefaultMemoryUsageProvider is the default implementation of MemoryUsageProvider
type DefaultMemoryUsageProvider struct{}

// VirtualMemory returns memory usage statistics
func (p *DefaultMemoryUsageProvider) VirtualMemory() (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemory()
}

// SystemMonitor represents a system monitor for CPU and memory
type SystemMonitor struct {
	config      *config.Config
	cpuProvider CPUUsageProvider
	memProvider MemoryUsageProvider
}

// NewSystemMonitor creates a new system monitor
func NewSystemMonitor(cfg *config.Config) *SystemMonitor {
	return &SystemMonitor{
		config:      cfg,
		cpuProvider: &DefaultCPUUsageProvider{},
		memProvider: &DefaultMemoryUsageProvider{},
	}
}

// NewSystemMonitorWithProviders creates a new system monitor with custom providers
func NewSystemMonitorWithProviders(cfg *config.Config, cpuProvider CPUUsageProvider, memProvider MemoryUsageProvider) *SystemMonitor {
	return &SystemMonitor{
		config:      cfg,
		cpuProvider: cpuProvider,
		memProvider: memProvider,
	}
}

// Check checks CPU and memory usage
func (m *SystemMonitor) Check() ([]*Item, error) {
	var items []*Item

	// Check CPU usage
	cpuItem, err := m.checkCPU()
	if err != nil {
		return nil, fmt.Errorf("failed to check CPU: %w", err)
	}
	items = append(items, cpuItem)

	// Check memory usage
	memItem, err := m.checkMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to check memory: %w", err)
	}
	items = append(items, memItem)

	return items, nil
}

// checkCPU checks CPU usage
func (m *SystemMonitor) checkCPU() (*Item, error) {
	// Get CPU usage percentage (average across all cores)
	percentages, err := m.cpuProvider.Percent(time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}

	if len(percentages) == 0 {
		return nil, fmt.Errorf("no CPU usage data available")
	}

	cpuPercent := percentages[0]

	// Determine status based on threshold
	status := StatusOK
	if cpuPercent >= float64(m.config.Monitoring.System.CPUThreshold) {
		status = StatusCritical
	} else if cpuPercent >= float64(m.config.Monitoring.System.CPUThreshold)*0.8 {
		status = StatusWarning
	}

	// Create item
	item := &Item{
		ID:        "cpu",
		Name:      "CPU Usage",
		Type:      "cpu",
		Status:    status,
		Value:     cpuPercent,
		Threshold: float64(m.config.Monitoring.System.CPUThreshold),
		Unit:      "%",
		Icon:      m.config.Monitoring.System.CPUIcon,
		LastCheck: time.Now(),
		Message:   fmt.Sprintf("%.1f%% used", cpuPercent),
	}

	return item, nil
}

// checkMemory checks memory usage
func (m *SystemMonitor) checkMemory() (*Item, error) {
	// Get memory usage statistics
	memory, err := m.memProvider.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory usage: %w", err)
	}

	memPercent := memory.UsedPercent

	// Determine status based on threshold
	status := StatusOK
	if memPercent >= float64(m.config.Monitoring.System.MemoryThreshold) {
		status = StatusCritical
	} else if memPercent >= float64(m.config.Monitoring.System.MemoryThreshold)*0.8 {
		status = StatusWarning
	}

	// Create item
	item := &Item{
		ID:        "memory",
		Name:      "Memory Usage",
		Type:      "memory",
		Status:    status,
		Value:     memPercent,
		Threshold: float64(m.config.Monitoring.System.MemoryThreshold),
		Unit:      "%",
		Icon:      m.config.Monitoring.System.MemoryIcon,
		LastCheck: time.Now(),
		Message:   fmt.Sprintf("%.1f%% used (%.1f GB / %.1f GB)", memPercent, float64(memory.Used)/1024/1024/1024, float64(memory.Total)/1024/1024/1024),
	}

	return item, nil
}
