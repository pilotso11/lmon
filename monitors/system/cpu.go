// Package system provides CPU and memory monitor implementations for lmon.
// This file contains the CPU monitor and its provider abstractions.
//
// # CPU Monitor Overview
//
// The CPU monitor checks system CPU usage and alerts if usage exceeds a configured threshold.
//
// ## How it works:
//   - Uses a CpuProvider interface to abstract CPU usage retrieval (default: gopsutil).
//   - Configured with:
//   - threshold: Usage percentage threshold for alerting.
//   - icon: UI icon (optional).
//   - On each check:
//   - Retrieves CPU usage as a percentage.
//   - Status is:
//   - Green: Below 90% of threshold.
//   - Amber: Between 90% of threshold and threshold.
//   - Red: At or above threshold.
//   - Configuration is persisted back to the config struct for saving.
//
// ## Example usage:
//
//	cpuMonitor := system.NewCpu(80, "", nil)
//	result := cpuMonitor.Check(context.Background())
//
// The monitor can be customized with a custom CpuProvider for testing or alternative implementations.
package system

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/load"

	"lmon/common"
	"lmon/config"
	"lmon/monitors"
)

const CpuIcon = "cpu"  // Default icon for CPU monitors
const Group = "system" // Group name for system monitors
const CPUDisplayName = "cpu"

// CpuProvider is an interface for obtaining CPU usage statistics.
// It allows for production and mock implementations.

type CpuStat struct {
	Usage   float64 // CPU usage percentage
	Count   int     // Number of CPU cores
	Load1m  float64 // 1 minute load average
	Load5m  float64 // 5 minute load average
	Load15m float64 // 15 minute load average
}

// CpuProvider is an interface for obtaining CPU usage statistics.
// It allows for production and mock implementations.
type CpuProvider interface {
	Usage() (CpuStat, error)
}

// defaultCpuProvider is the default implementation of CpuProvider using gopsutil.
//
// It tracks previous CPU times to calculate usage deltas.
type defaultCpuProvider struct {
	mu           sync.Mutex // Mutex to protect access to CPU times
	prevCPUTimes cpu.TimesStat
	lastCPUCheck time.Time
	cpuCount     int
}

// newDefaultCpuProvider creates a new defaultCpuProvider and initializes its state.
func newDefaultCpuProvider() *defaultCpuProvider {
	d := defaultCpuProvider{}
	_, _ = d.Usage()
	return &d
}

// Usage returns the current CPU usage percentage.
func (d *defaultCpuProvider) Usage() (CpuStat, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	times, err := cpu.Times(false)
	if err != nil {
		return CpuStat{}, err
	}

	ld, err := load.Avg()
	if err != nil {
		return CpuStat{}, err
	}

	if len(times) == 0 {
		return CpuStat{}, fmt.Errorf("no CPU times available")
	}
	usage := calculateCPUPercentage(times[0], d.prevCPUTimes)

	d.cpuCount, _ = cpu.Counts(true) // take what's available, even if it fails

	d.prevCPUTimes = times[0]
	d.lastCPUCheck = time.Now()
	return CpuStat{
		Usage:   usage,
		Count:   d.cpuCount,
		Load1m:  ld.Load1,
		Load5m:  ld.Load5,
		Load15m: ld.Load15,
	}, nil
}

// calculateCPUPercentage calculates the CPU usage percentage based on the difference
// between current and previous CPU times.
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

// Cpu represents a CPU usage monitor.
//
// Fields:
//   - threshold: Usage percentage threshold for alerting.
//   - icon: Icon for UI display.
//   - impl: Implementation for usage statistics (defaults to defaultCpuProvider).
type Cpu struct {
	threshold      int         // Usage percentage threshold for alerting
	icon           string      // Icon for UI display
	impl           CpuProvider // Implementation for usage statistics
	alertThreshold int         // Number of consecutive failures before triggering alert
}

// NewCpu constructs a new Cpu monitor with the given parameters.
//
// If icon is empty, the default CpuIcon is used.
// If provider is nil, the defaultCpuProvider is used.
// If alertThreshold is 0, it defaults to 1.
//
// Example:
//
//	cpuMonitor := NewCpu(80, "", 0, nil)
func NewCpu(threshold int, icon string, alertThreshold int, provider CpuProvider) Cpu {
	if icon == "" {
		icon = CpuIcon
	}
	if common.IsNil(provider) {
		provider = newDefaultCpuProvider()
	}
	if alertThreshold <= 0 {
		alertThreshold = 1
	}
	return Cpu{
		threshold:      threshold,
		icon:           icon,
		alertThreshold: alertThreshold,
		impl:           provider,
	}
}

func (c Cpu) String() string {
	return fmt.Sprintf("Cpu{threshold: %d, icon: %s}", c.threshold, c.icon)
}

// DisplayName returns a human-readable name for the CPU monitor.
func (c Cpu) DisplayName() string {
	return CPUDisplayName
}

// Group returns the group/category for the CPU monitor.
func (c Cpu) Group() string {
	return Group
}

// Name returns the unique name/ID for the CPU monitor.
func (c Cpu) Name() string {
	return fmt.Sprintf("%s_cpu", Group)
}

// Save persists the CPU monitor's configuration to the provided config struct.
func (c Cpu) Save(cfg *config.Config) {
	cfg.Monitoring.System.CPU.Threshold = c.threshold
	cfg.Monitoring.System.CPU.Icon = c.icon
	cfg.Monitoring.System.CPU.AlertThreshold = c.alertThreshold
}

// AlertThreshold returns the number of consecutive failures before triggering an alert
func (c Cpu) AlertThreshold() int {
	return c.alertThreshold
}

// Check performs a usage check on the CPU and returns a Result.
// It uses the configured CpuProvider implementation.
// Check performs a usage check on the CPU and returns a Result.
// It uses the configured CpuProvider implementation.
//
// Status logic:
//   - Green: usage < 90% of threshold
//   - Amber: usage >= 90% of threshold but < threshold
//   - Red: usage >= threshold
func (c Cpu) Check(_ context.Context) monitors.Result {
	stat, err := c.impl.Usage()
	if err != nil {
		return monitors.Result{
			Status: monitors.RAGError,
			Value:  fmt.Sprintf("error getting CPU Current: %v", err),
		}
	}
	val := fmt.Sprintf("%.1f%% (%d CPUs)", stat.Usage, stat.Count)
	val2 := fmt.Sprintf("Load Avg (1m/5m/15m)  %.1f %.1f %.1f", stat.Load1m, stat.Load5m, stat.Load15m)
	status := monitors.RAGGreen
	switch {
	case stat.Usage >= float64(c.threshold):
		status = monitors.RAGRed
	case stat.Usage >= float64(c.threshold)*.9:
		status = monitors.RAGAmber
	}
	return monitors.Result{
		Status: status,
		Value:  val,
		Value2: val2,
	}
}
