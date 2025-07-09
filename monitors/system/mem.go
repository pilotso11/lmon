// Package system provides CPU and memory monitor implementations for lmon.
// This file contains the memory monitor and its provider abstractions.
//
// # Memory Monitor
//
// The Memory monitor checks system memory usage and alerts if usage exceeds a configured threshold.
//
// How it works:
//   - Uses a MemProvider interface to abstract memory usage retrieval (default: gopsutil).
//   - Configured with:
//   - threshold: Usage percentage threshold for alerting.
//   - icon: UI icon (optional).
//   - On each check:
//   - Retrieves memory usage stats.
//   - Formats the used percentage as a string.
//   - Status is:
//   - Green: Below 90% of threshold.
//   - Amber: Between 90% of threshold and threshold.
//   - Red: At or above threshold.
//   - Configuration is persisted back to the config struct for saving.
//
// Example usage:
//
//	memMonitor := NewMem(80, "", nil)
//	result := memMonitor.Check(context.Background())
//	fmt.Println(result.Status, result.Value)
package system

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v3/mem"

	"lmon/common"
	"lmon/config"
	"lmon/monitors"
)

const MemIcon = "speedometer" // Default icon for memory monitors

// MemProvider is an interface for obtaining memory usage statistics.
// It allows for production and mock implementations.
type MemProvider interface {
	// Usage returns the current memory usage statistics.
	Usage() (*mem.VirtualMemoryStat, error)
}

// defaultMemProvider is the default implementation of MemProvider using gopsutil.
type defaultMemProvider struct {
}

// Usage returns the current memory usage statistics.
func (d defaultMemProvider) Usage() (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemory()
}

// Mem represents a memory usage monitor.
//
// Fields:
//   - threshold: Usage percentage threshold for alerting.
//   - icon: Icon for UI display.
//   - impl: Implementation for usage statistics (defaults to defaultMemProvider).
type Mem struct {
	threshold int         // Usage percentage threshold for alerting
	icon      string      // Icon for UI display
	impl      MemProvider // Implementation for usage statistics
}

// NewMem constructs a new Mem monitor with the given parameters.
// If icon is empty, the default MemIcon is used.
// If provider is nil, the defaultMemProvider is used.
func NewMem(threshold int, icon string, provider MemProvider) Mem {
	if icon == "" {
		icon = MemIcon
	}
	if common.IsNil(provider) {
		provider = defaultMemProvider{}
	}
	return Mem{
		threshold: threshold,
		icon:      icon,
		impl:      provider,
	}
}

func (c Mem) String() string {
	return fmt.Sprintf("Mem{threshold: %d, icon: %s}", c.threshold, c.icon)
}

// DisplayName returns a human-readable name for the memory monitor.
func (c Mem) DisplayName() string {
	return "mem"
}

// Group returns the group/category for the memory monitor.
func (c Mem) Group() string {
	return Group
}

// Name returns the unique name/ID for the memory monitor.
func (c Mem) Name() string {
	return fmt.Sprintf("%s_mem", Group)
}

// Save persists the memory monitor's configuration to the provided config struct.
func (c Mem) Save(cfg *config.Config) {
	cfg.Monitoring.System.Memory.Threshold = c.threshold
	cfg.Monitoring.System.Memory.Icon = c.icon
}

// Check performs a usage check on memory and returns a Result.
// It uses the configured MemProvider implementation.
//
// The returned Result's Status is:
//   - RAGGreen: if usage is below 90% of threshold
//   - RAGAmber: if usage is between 90% of threshold and threshold
//   - RAGRed: if usage is at or above threshold
//   - RAGError: if there was an error retrieving memory usage
func (c Mem) Check(_ context.Context) monitors.Result {
	usage, err := c.impl.Usage()
	if err != nil {
		return monitors.Result{
			Status: monitors.RAGError,
			Value:  fmt.Sprintf("error getting mem usage: %v", err),
		}
	}
	val := fmt.Sprintf("%.1f%%", usage.UsedPercent)
	status := monitors.RAGGreen
	switch {
	case usage.UsedPercent >= float64(c.threshold):
		status = monitors.RAGRed
	case usage.UsedPercent >= float64(c.threshold)*.9:
		status = monitors.RAGAmber
	}
	return monitors.Result{
		Status: status,
		Value:  val,
	}
}
