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
	"reflect"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"

	"lmon/config"
	"lmon/monitors"
)

const CpuIcon = "cpu"  // Default icon for CPU monitors
const Group = "system" // Group name for system monitors

// CpuProvider is an interface for obtaining CPU usage statistics.
// It allows for production and mock implementations.

// CpuProvider is an interface for obtaining CPU usage statistics.
// It allows for production and mock implementations.
type CpuProvider interface {
	Usage() (float64, error)
}

// defaultCpuProvider is the default implementation of CpuProvider using gopsutil.
//
// It tracks previous CPU times to calculate usage deltas.
type defaultCpuProvider struct {
	prevCPUTimes cpu.TimesStat
	lastCPUCheck time.Time
}

// newDefaultCpuProvider creates a new defaultCpuProvider and initializes its state.
func newDefaultCpuProvider() *defaultCpuProvider {
	d := defaultCpuProvider{}
	_, _ = d.Usage()
	return &d
}

// Usage returns the current CPU usage percentage.
func (d *defaultCpuProvider) Usage() (float64, error) {
	times, err := cpu.Times(false)
	if err != nil {
		return 0, err
	}

	usage := calculateCPUPercentage(times[0], d.prevCPUTimes)

	d.prevCPUTimes = times[0]
	d.lastCPUCheck = time.Now()
	return usage, nil
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
	threshold int         // Usage percentage threshold for alerting
	icon      string      // Icon for UI display
	impl      CpuProvider // Implementation for usage statistics
}

// NewCpu constructs a new Cpu monitor with the given parameters.
//
// If icon is empty, the default CpuIcon is used.
// If provider is nil, the defaultCpuProvider is used.
//
// Example:
//
//	cpuMonitor := NewCpu(80, "", nil)
func NewCpu(threshold int, icon string, provider CpuProvider) Cpu {
	if icon == "" {
		icon = CpuIcon
	}
	if provider == nil || reflect.ValueOf(provider).IsNil() {
		provider = newDefaultCpuProvider()
	}
	return Cpu{
		threshold: threshold,
		icon:      icon,
		impl:      provider,
	}
}

// DisplayName returns a human-readable name for the CPU monitor.
func (c Cpu) DisplayName() string {
	return "cpu"
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
	usage, err := c.impl.Usage()
	if err != nil {
		return monitors.Result{
			Status: monitors.RAGError,
			Value:  fmt.Sprintf("error getting CPU Current: %v", err),
		}
	}
	val := fmt.Sprintf("%.1f%%", usage)
	status := monitors.RAGGreen
	switch {
	case usage >= float64(c.threshold):
		status = monitors.RAGRed
	case usage >= float64(c.threshold)*.9:
		status = monitors.RAGAmber
	}
	return monitors.Result{
		Status: status,
		Value:  val,
	}
}
