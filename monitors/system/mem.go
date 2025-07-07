// Package system provides CPU and memory monitor implementations for lmon.
// This file contains the memory monitor and its provider abstractions.
package system

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v3/mem"

	"lmon/config"
	"lmon/monitors"
)

const MemIcon = "speedometer" // Default icon for memory monitors

// MemProvider is an interface for obtaining memory usage statistics.
// It allows for production and mock implementations.
type MemProvider interface {
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
	if provider == nil {
		provider = defaultMemProvider{}
	}
	return Mem{
		threshold: threshold,
		icon:      icon,
		impl:      provider,
	}
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
