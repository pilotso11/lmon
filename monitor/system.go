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
	Times(percpu bool) ([]cpu.TimesStat, error)
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

// Times returns CPU time statistics
func (p *DefaultCPUUsageProvider) Times(percpu bool) ([]cpu.TimesStat, error) {
	return cpu.Times(percpu)
}

// DefaultMemoryUsageProvider is the default implementation of MemoryUsageProvider
type DefaultMemoryUsageProvider struct{}

// VirtualMemory returns memory usage statistics
// This will return system-wide memory usage when running in a Docker container
// as long as the container has access to the host's /proc filesystem
func (p *DefaultMemoryUsageProvider) VirtualMemory() (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemory()
}

// SystemMonitor represents a system monitor for CPU and memory
type SystemMonitor struct {
	config       *config.Config
	cpuProvider  CPUUsageProvider
	memProvider  MemoryUsageProvider
	prevCPUTimes []cpu.TimesStat
	lastCPUCheck time.Time
	useCPUTimes  bool
}

// NewSystemMonitor creates a new system monitor
func NewSystemMonitor(cfg *config.Config) *SystemMonitor {
	return &SystemMonitor{
		config:       cfg,
		cpuProvider:  &DefaultCPUUsageProvider{},
		memProvider:  &DefaultMemoryUsageProvider{},
		useCPUTimes:  true, // Use CPU times by default for more accurate system-wide metrics
		lastCPUCheck: time.Now(),
	}
}

// NewSystemMonitorWithProviders creates a new system monitor with custom providers
func NewSystemMonitorWithProviders(cfg *config.Config, cpuProvider CPUUsageProvider, memProvider MemoryUsageProvider) *SystemMonitor {
	return &SystemMonitor{
		config:       cfg,
		cpuProvider:  cpuProvider,
		memProvider:  memProvider,
		useCPUTimes:  true, // Use CPU times by default for more accurate system-wide metrics
		lastCPUCheck: time.Now(),
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
	var cpuPercent float64

	if m.useCPUTimes {
		// Get CPU times for all CPUs combined
		times, err := m.cpuProvider.Times(false)
		if err != nil {
			return nil, fmt.Errorf("failed to get CPU times: %w", err)
		}

		if len(times) == 0 {
			return nil, fmt.Errorf("no CPU times data available")
		}

		// Calculate CPU usage based on the difference between current and previous times
		if len(m.prevCPUTimes) > 0 {
			// Calculate CPU usage percentage
			cpuPercent = calculateCPUPercentage(times[0], m.prevCPUTimes[0])
		}

		// Store current times for next check
		m.prevCPUTimes = times
		m.lastCPUCheck = time.Now()
	} else {
		// Use the standard Percent method (may not be accurate in containers)
		percentages, err := m.cpuProvider.Percent(time.Second, false)
		if err != nil {
			return nil, fmt.Errorf("failed to get CPU usage: %w", err)
		}

		if len(percentages) == 0 {
			return nil, fmt.Errorf("no CPU usage data available")
		}

		cpuPercent = percentages[0]
	}

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
		Message:   fmt.Sprintf("%.2f%% used", cpuPercent),
	}

	return item, nil
}

// calculateCPUPercentage calculates the CPU usage percentage based on the difference between current and previous CPU times
func calculateCPUPercentage(current, previous cpu.TimesStat) float64 {
	// Calculate the total time spent by the CPU
	prevTotal := previous.User + previous.System + previous.Idle + previous.Nice + previous.Iowait + previous.Irq + previous.Softirq + previous.Steal
	currTotal := current.User + current.System + current.Idle + current.Nice + current.Iowait + current.Irq + current.Softirq + current.Steal

	// Calculate the total time spent by the CPU doing work
	prevBusy := previous.User + previous.System + previous.Nice + previous.Irq + previous.Softirq + previous.Steal
	currBusy := current.User + current.System + current.Nice + current.Irq + current.Softirq + current.Steal

	// Calculate the difference in total and busy time
	totalDiff := currTotal - prevTotal
	busyDiff := currBusy - prevBusy

	// Calculate the percentage of time spent by the CPU doing work
	if totalDiff > 0 {
		return (busyDiff / totalDiff) * 100.0
	}
	return 0.0
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
		Message:   fmt.Sprintf("%.2f%% used (%.1f GB / %.1f GB)", memPercent, float64(memory.Used)/1024/1024/1024, float64(memory.Total)/1024/1024/1024),
	}

	return item, nil
}
